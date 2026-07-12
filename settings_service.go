package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// ========================================================================
// SETTINGS SERVICE - Encrypted persistence with HKDF key derivation
// ========================================================================

// SettingsService manages application settings with encryption.
// Uses FieldCrypto (HKDF + AES-256-GCM) when available, with backward
// compatibility for values encrypted with the old SHA-256 derived key.
type SettingsService struct {
	db            *gorm.DB
	encryptionKey []byte // Legacy key (SHA-256 of hardware ID) - kept for migration
	fieldCrypto   *FieldCrypto
	hardwareID    string
}

// NewSettingsService creates a settings service
func NewSettingsService(db *gorm.DB) (*SettingsService, error) {
	svc := &SettingsService{
		db: db,
	}

	// Derive encryption key from hardware ID
	hardwareID, err := getHardwareID()
	if err != nil {
		log.Printf("⚠ Failed to get hardware ID: %v (using fallback)", err)
		hardwareID = "fallback-key-ace-engine"
	}
	svc.hardwareID = hardwareID

	// Keep legacy key for backward-compatible decryption of old values
	hash := sha256.Sum256([]byte(hardwareID + "asymmetrica-salt-2025"))
	svc.encryptionKey = hash[:]

	return svc, nil
}

// SetFieldCrypto upgrades the SettingsService to use FieldCrypto for new encryptions.
// Called after FieldCrypto is initialized in App startup.
func (s *SettingsService) SetFieldCrypto(fc *FieldCrypto) {
	s.fieldCrypto = fc
}

// GetSetting retrieves a setting by key, auto-decrypting if needed.
func (s *SettingsService) GetSetting(key string) (string, error) {
	var setting Setting
	result := s.db.Where("key = ?", key).First(&setting)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("setting not found: %s", key)
		}
		return "", result.Error
	}

	if setting.IsEncrypted {
		decrypted, err := s.decryptWithFallback(setting.Value)
		if err != nil {
			return "", fmt.Errorf("decrypt setting %s: %w", key, err)
		}
		return decrypted, nil
	}

	return setting.Value, nil
}

// SetSetting saves a setting. If encrypt=true, uses FieldCrypto (preferred) or legacy.
func (s *SettingsService) SetSetting(key, value, category string, encrypt bool) error {
	valueToStore := value

	if encrypt {
		encrypted, err := s.encrypt(value)
		if err != nil {
			return err
		}
		valueToStore = encrypted
	}

	setting := Setting{
		Key:         key,
		Value:       valueToStore,
		Category:    category,
		IsEncrypted: encrypt,
	}

	result := s.db.Where("key = ?", key).Assign(setting).FirstOrCreate(&setting)
	return result.Error
}

// encrypt uses FieldCrypto if available, falls back to legacy AES-256-GCM.
func (s *SettingsService) encrypt(plaintext string) (string, error) {
	if s.fieldCrypto != nil {
		return s.fieldCrypto.Encrypt(plaintext)
	}
	return s.legacyEncrypt(plaintext)
}

// decryptWithFallback tries FieldCrypto first, then legacy key.
// If legacy key works, re-encrypts with FieldCrypto for migration.
func (s *SettingsService) decryptWithFallback(ciphertext string) (string, error) {
	// 1. Try FieldCrypto (versioned, HKDF-derived)
	if s.fieldCrypto != nil && s.fieldCrypto.IsEncrypted(ciphertext) {
		plaintext, err := s.fieldCrypto.Decrypt(ciphertext)
		if err == nil {
			return plaintext, nil
		}
		// Fall through to legacy
	}

	// 2. Try legacy decrypt (SHA-256 derived key, no version byte)
	plaintext, err := s.legacyDecrypt(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decryption failed (both new and legacy keys): %w", err)
	}

	// 3. Re-encrypt with FieldCrypto for transparent migration
	if s.fieldCrypto != nil {
		newEncrypted, reErr := s.fieldCrypto.Encrypt(plaintext)
		if reErr == nil {
			// Update in DB with new encryption (fire-and-forget)
			s.db.Model(&Setting{}).Where("value = ?", ciphertext).Update("value", newEncrypted)
			log.Printf("settings: migrated encrypted value to HKDF key")
		}
	}

	return plaintext, nil
}

// legacyEncrypt encrypts using the old SHA-256-derived key (AES-256-GCM, no version byte).
func (s *SettingsService) legacyEncrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// legacyDecrypt decrypts using the old SHA-256-derived key.
func (s *SettingsService) legacyDecrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// hardwareIDOnce/hardwareIDCached/hardwareIDErr memoize getHardwareID() for the
// process lifetime. getHardwareID is called from several places (legacy settings
// key, field crypto PBKDF2 password, auth AES key) and the Windows resolution
// path shells out to subprocesses — resolve it exactly once.
var (
	hardwareIDOnce   sync.Once
	hardwareIDCached string
	hardwareIDErr    error
)

// getHardwareID returns a machine-specific identifier, memoized for the process
// lifetime so repeated callers don't re-pay the (bounded) subprocess cost.
func getHardwareID() (string, error) {
	hardwareIDOnce.Do(func() {
		hardwareIDCached, hardwareIDErr = resolveHardwareID()
	})
	return hardwareIDCached, hardwareIDErr
}

// hardwareIDSidecarPathOverride lets tests redirect hardware-ID persistence to
// a temp file instead of the real DB directory. Always empty in production.
var hardwareIDSidecarPathOverride string

// hardwareIDSidecarPath returns the path to the (historically plaintext)
// sidecar file that persists the resolved hardware ID next to the SQLite
// database. It MUST NEVER be routed through SettingsService/FieldCrypto
// because the hardware ID is the input that DERIVES the encryption key —
// storing it in an encrypted column would require the key to read the key
// (bootstrap deadlock). Where a native OS keystore is available (Windows/
// DPAPI, see hardware_id_keystore_windows.go), the value is instead wrapped
// at rest in a sibling file at keystoreSidecarPath(this) and this plaintext
// path is only read as a migration source / fallback; see resolveHardwareID.
// Returns "" if the database directory cannot be determined yet.
func hardwareIDSidecarPath() string {
	if hardwareIDSidecarPathOverride != "" {
		return hardwareIDSidecarPathOverride
	}
	dbPath := getDatabasePath()
	if strings.TrimSpace(dbPath) == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(dbPath), ".hardware_id")
}

// errKeystoreUnavailable is returned by the non-Windows keystoreProtect/
// keystoreUnprotect passthrough (hardware_id_keystore_other.go) so callers on
// platforms without a native OS keystore fall back to the plaintext sidecar
// without treating it as a hard failure.
var errKeystoreUnavailable = errors.New("OS keystore unavailable on this platform")

// keystoreSidecarPath returns the path of the DPAPI-protected (or, on
// platforms without a keystore, never-created) sibling of the plaintext
// hardware-ID sidecar. Kept as a plain suffix of plainSidecar so it inherits
// the test override seam (hardwareIDSidecarPathOverride) automatically.
func keystoreSidecarPath(plainSidecar string) string {
	if plainSidecar == "" {
		return ""
	}
	return plainSidecar + ".dpapi"
}

// protectAndPersistKeystoreSidecar protects id via the platform keystore
// (DPAPI on Windows), writes the protected blob to keystoreSidecarPath(plainSidecar),
// then reads it back and unprotects it to verify the round-trip decrypts to
// the exact same value BEFORE returning success. Callers must not touch the
// plaintext sidecar unless this returns nil — a non-nil error means no
// key material was stranded and the plaintext (if any) is still the only
// authoritative copy.
func protectAndPersistKeystoreSidecar(plainSidecar, id string) error {
	kpath := keystoreSidecarPath(plainSidecar)
	if kpath == "" {
		return fmt.Errorf("protectAndPersistKeystoreSidecar: empty sidecar path")
	}

	protected, err := keystoreProtect([]byte(id))
	if err != nil {
		return fmt.Errorf("keystore protect: %w", err)
	}
	if dir := filepath.Dir(kpath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("keystore sidecar mkdir: %w", err)
		}
	}
	if err := os.WriteFile(kpath, protected, 0o600); err != nil {
		return fmt.Errorf("keystore sidecar write: %w", err)
	}

	// Round-trip verification: read back what was just written and unprotect
	// it. Only a verified match proves the protected blob is actually usable
	// on this machine before any plaintext copy is touched.
	readBack, err := os.ReadFile(kpath)
	if err != nil {
		_ = os.Remove(kpath)
		return fmt.Errorf("keystore sidecar post-write read: %w", err)
	}
	decrypted, err := keystoreUnprotect(readBack)
	if err != nil {
		_ = os.Remove(kpath)
		return fmt.Errorf("keystore sidecar round-trip unprotect: %w", err)
	}
	if strings.TrimSpace(string(decrypted)) != id {
		_ = os.Remove(kpath)
		return fmt.Errorf("keystore sidecar round-trip mismatch")
	}
	return nil
}

// readKeystoreSidecar attempts to resolve the hardware ID from the
// OS-keystore-protected sidecar. Returns ok=false (never an error the caller
// must handle) whenever the keystore isn't available, the protected sidecar
// doesn't exist yet, or it fails to unprotect — all of which simply mean "the
// caller should try the next source" (plaintext sidecar, then live
// resolution).
func readKeystoreSidecar(plainSidecar string) (string, bool) {
	if !keystoreAvailable() {
		return "", false
	}
	kpath := keystoreSidecarPath(plainSidecar)
	if kpath == "" {
		return "", false
	}
	protected, err := os.ReadFile(kpath)
	if err != nil {
		return "", false
	}
	plaintext, err := keystoreUnprotect(protected)
	if err != nil {
		log.Printf("⚠ Failed to unprotect hardware ID OS-keystore sidecar (%s), falling back: %v", kpath, err)
		return "", false
	}
	id := strings.TrimSpace(string(plaintext))
	if id == "" {
		return "", false
	}
	return id, true
}

// migrateHardwareIDToKeystore is called when resolveHardwareID() finds an
// old-install plaintext sidecar but no keystore sidecar yet. It protects the
// existing value, verifies the round-trip (see protectAndPersistKeystoreSidecar),
// and ONLY THEN retires the plaintext file — by renaming it to a ".migrated"
// backup rather than deleting it, so a rollback never requires re-deriving
// the hardware ID. If protection or verification fails at any step, the
// plaintext sidecar is left completely untouched and a warning is logged;
// resolveHardwareID() keeps working off the plaintext value either way.
func migrateHardwareIDToKeystore(plainSidecar, id string) {
	if !keystoreAvailable() {
		return
	}
	if err := protectAndPersistKeystoreSidecar(plainSidecar, id); err != nil {
		log.Printf("⚠ Hardware ID keystore migration failed, keeping plaintext sidecar intact (boot continues): %v", err)
		return
	}

	backup := plainSidecar + ".migrated"
	if err := os.Rename(plainSidecar, backup); err != nil {
		log.Printf("⚠ Hardware ID keystore migration verified but could not retire plaintext sidecar %s (OS keystore is authoritative going forward regardless): %v", plainSidecar, err)
		return
	}
	log.Printf("✓ Hardware ID sidecar migrated to OS keystore; plaintext retired to %s", backup)
}

// persistHardwareID best-effort persists the resolved hardware ID so
// subsequent boots prefer this exact value instead of re-resolving (and
// potentially getting a different answer if CIM/WMIC/hostname race
// differently under load). Never blocks boot: failures are logged, not
// returned as errors.
//
// When a native OS keystore is available (Windows/DPAPI), this writes ONLY
// the keystore-protected sidecar for fresh installs — no plaintext copy is
// ever created. If the keystore write/verify fails, or no keystore exists on
// this platform, it falls back to the historical plaintext sidecar.
func persistHardwareID(id string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return
	}
	sidecar := hardwareIDSidecarPath()
	if sidecar == "" {
		return
	}

	if keystoreAvailable() {
		if err := protectAndPersistKeystoreSidecar(sidecar, id); err == nil {
			return // Verified DPAPI-only persistence; no plaintext written.
		} else {
			log.Printf("⚠ Failed to persist hardware ID to OS keystore, falling back to plaintext sidecar: %v", err)
		}
	}

	if dir := filepath.Dir(sidecar); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("⚠ Failed to create directory for hardware ID sidecar: %v", err)
			return
		}
	}
	if err := os.WriteFile(sidecar, []byte(id), 0o600); err != nil {
		log.Printf("⚠ Failed to persist hardware ID sidecar (boot continues): %v", err)
	}
}

// resolveHardwareID resolves the machine-identifier, PREFERRING a previously
// persisted value so the derived encryption key stays stable across boots
// even when CIM/WMIC/hostname resolution timing varies. Resolution order:
//  1. OS-keystore-protected sidecar (DPAPI on Windows) — the steady state
//     for fresh installs and machines already migrated.
//  2. Plaintext sidecar (pre-migration installs) — if found, this value is
//     returned AND a best-effort migration to the keystore is attempted (see
//     migrateHardwareIDToKeystore) so the next boot uses step 1.
//  3. Live re-resolution (CIM/WMIC/hostname), persisted via persistHardwareID
//     for all future boots.
//
// The returned VALUE is identical regardless of which of these three sources
// answered — only the at-rest storage of that value differs.
func resolveHardwareID() (string, error) {
	if sidecar := hardwareIDSidecarPath(); sidecar != "" {
		if id, ok := readKeystoreSidecar(sidecar); ok {
			return id, nil
		}

		if data, err := os.ReadFile(sidecar); err == nil {
			if id := strings.TrimSpace(string(data)); id != "" {
				migrateHardwareIDToKeystore(sidecar, id)
				return id, nil
			}
		}
	}

	id, err := resolveHardwareIDUncached()
	if err == nil {
		persistHardwareID(id)
	}
	return id, err
}

// resolveHardwareIDUncached performs the actual machine-identifier lookup. On
// Windows it prefers the modern, WMI-independent Get-CimInstance cmdlet,
// bounded by a short timeout, and falls back to the historical `wmic`
// invocation (also bounded) to preserve byte-identical output on machines
// where wmic still answers — this matters because the hardware ID feeds
// field-crypto key derivation, where a single changed byte would make every
// encrypted field permanently undecryptable.
// exec.CommandContext is used (not go-ole/COM) specifically because a hung
// subprocess can be killed on context timeout; a COM call cannot be
// context-cancelled and would still hang the calling goroutine forever.
func resolveHardwareIDUncached() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if id, ok := getWindowsHardwareIDViaCIM(); ok {
			return id, nil
		}
		if id, ok := getWindowsHardwareIDViaWMIC(); ok {
			return id, nil
		}
	case "darwin":
		// Use ioreg to get IOPlatformSerialNumber (Mac serial number)
		cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
		suppressCommandWindow(cmd)
		output, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if strings.Contains(line, "IOPlatformSerialNumber") {
					// Format: "IOPlatformSerialNumber" = "XXXXXXXXXXXX"
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						serial := strings.TrimSpace(parts[1])
						serial = strings.Trim(serial, "\"")
						if serial != "" {
							return serial, nil
						}
					}
				}
			}
		}
	default:
		// Linux: try /etc/machine-id
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			id := strings.TrimSpace(string(data))
			if id != "" {
				return id, nil
			}
		}
	}

	// Fallback: use hostname
	return os.Hostname()
}

// getWindowsHardwareIDViaCIM resolves the motherboard serial number using the
// modern Get-CimInstance cmdlet (WMIC-independent, future-proof — wmic is
// deprecated and being removed from newer Windows builds). Bounded by a 6s
// context timeout so a wedged WMI provider (winmgmt) cannot hang the process.
func getWindowsHardwareIDViaCIM() (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command",
		"Get-CimInstance Win32_BaseBoard | Select-Object -ExpandProperty SerialNumber")
	suppressCommandWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	serial := strings.TrimSpace(string(output))
	if serial == "" {
		return "", false
	}
	return serial, true
}

// getWindowsHardwareIDViaWMIC is the last-resort, byte-identity-preserving path:
// identical query and parsing to the historical implementation (split on \n,
// trim each line, first non-empty non-"SerialNumber" line wins), so machines
// where wmic still answers keep deriving the exact same key material. Bounded
// by a 3s context timeout so a wedged wmic.exe cannot hang the process either.
func getWindowsHardwareIDViaWMIC() (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "wmic", "baseboard", "get", "serialnumber")
	suppressCommandWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.EqualFold(line, "SerialNumber") {
			return line, true
		}
	}
	return "", false
}
