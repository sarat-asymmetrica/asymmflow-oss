package main

import (
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
	"path/filepath"
	"sync"

	"golang.org/x/crypto/hkdf"
	"golang.org/x/crypto/pbkdf2"

	"ph_holdings_app/pkg/infra/deploy"
)

// FieldCrypto provides field-level encryption for sensitive database values.
// Uses HKDF-SHA256 for key derivation (adapted from Security System KeyManager)
// and AES-256-GCM for authenticated encryption (from existing SettingsService).
//
// Wire format: [version(1)][nonce(12)][ciphertext+tag(N)] -> base64 encoded
// The version byte enables key rotation without breaking old data.
type FieldCrypto struct {
	mu         sync.RWMutex
	keys       map[uint8][]byte // version -> 32-byte AES key
	currentVer uint8
	masterKey  []byte
	salt       []byte
}

const (
	fieldCryptoKeySize      = 32
	fieldCryptoSaltSize     = 32
	fieldCryptoPBKDF2Iters  = 600_000
	fieldCryptoMasterKeyEnv = "ENCRYPTION_MASTER_KEY"
)

var (
	ErrFieldCryptoNoKey      = errors.New("field_crypto: no encryption key available")
	ErrFieldCryptoShortData  = errors.New("field_crypto: ciphertext too short")
	ErrFieldCryptoUnknownVer = errors.New("field_crypto: unknown key version")

	// globalFieldCrypto is set during App startup for use in GORM hooks
	// (GORM hooks don't have access to the App instance).
	globalFieldCrypto *FieldCrypto
)

// NewFieldCrypto creates a FieldCrypto with key material from:
// 1. ENCRYPTION_MASTER_KEY env var (hex string, preferred)
// 2. Hardware ID fallback (existing getHardwareID())
//
// If the master material is valid hex >= 32 bytes, it's used directly.
// Otherwise it's strengthened via PBKDF2 with 600k iterations.
func NewFieldCrypto() (*FieldCrypto, error) {
	// Determine master secret
	secret := os.Getenv(fieldCryptoMasterKeyEnv)
	if secret == "" {
		// Fall back to hardware ID (same source as SettingsService)
		hwID, err := getHardwareID()
		if err != nil {
			hwID = "fallback-key-ace-engine"
			log.Printf("field_crypto: hardware ID unavailable, using fallback")
		}
		secret = hwID
	}

	// Load or generate random salt (stored in .field_crypto_salt file)
	// No deterministic fallback — salt MUST be random for cryptographic security
	salt, err := loadOrCreateSalt()
	if err != nil {
		return nil, fmt.Errorf("field_crypto: salt initialization failed (CRITICAL — cannot use deterministic fallback): %w", err)
	}

	// Derive master key: if secret is long enough hex, use directly; else PBKDF2
	var masterKey []byte
	if len(secret) >= 64 { // 32 bytes hex-encoded = 64 chars
		decoded := make([]byte, len(secret)/2)
		validHex := true
		for i := 0; i < len(secret)-1; i += 2 {
			b, err := hexByte(secret[i], secret[i+1])
			if err != nil {
				validHex = false
				break
			}
			decoded[i/2] = b
		}
		if validHex && len(decoded) >= fieldCryptoKeySize {
			masterKey = decoded[:fieldCryptoKeySize]
		}
	}
	if masterKey == nil {
		// Strengthen weak secret via PBKDF2
		masterKey = pbkdf2.Key([]byte(secret), salt, fieldCryptoPBKDF2Iters, fieldCryptoKeySize, sha256.New)
	}

	fc := &FieldCrypto{
		keys:      make(map[uint8][]byte),
		masterKey: masterKey,
		salt:      salt,
	}

	// Derive version 1 key
	if _, err := fc.deriveVersion(1); err != nil {
		return nil, fmt.Errorf("field_crypto: initial key derivation failed: %w", err)
	}
	fc.currentVer = 1

	return fc, nil
}

// deriveVersion derives a 32-byte AES key for the given version using HKDF-SHA256.
func (fc *FieldCrypto) deriveVersion(version uint8) ([]byte, error) {
	if key, exists := fc.keys[version]; exists {
		return key, nil
	}

	info := []byte(fmt.Sprintf("ph-field-aes256gcm-v%d", version))
	hkdfReader := hkdf.New(sha256.New, fc.masterKey, fc.salt, info)

	key := make([]byte, fieldCryptoKeySize)
	if _, err := io.ReadFull(hkdfReader, key); err != nil {
		return nil, fmt.Errorf("HKDF derivation failed for v%d: %w", version, err)
	}

	fc.keys[version] = key
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with the current key version.
// Returns base64-encoded string: [version(1)][nonce(12)][ciphertext+tag].
func (fc *FieldCrypto) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	fc.mu.RLock()
	ver := fc.currentVer
	key := fc.keys[ver]
	fc.mu.RUnlock()

	if key == nil {
		return "", ErrFieldCryptoNoKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("field_crypto: cipher init failed: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("field_crypto: GCM init failed: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("field_crypto: nonce generation failed: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)

	// Wire format: [version(1)][nonce(12)][ciphertext+tag]
	wire := make([]byte, 1+len(nonce)+len(ciphertext))
	wire[0] = ver
	copy(wire[1:], nonce)
	copy(wire[1+len(nonce):], ciphertext)

	return base64.StdEncoding.EncodeToString(wire), nil
}

// Decrypt decrypts a base64-encoded ciphertext, selecting the key version
// from the first byte of the wire format.
func (fc *FieldCrypto) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	wire, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("field_crypto: base64 decode failed: %w", err)
	}

	// Minimum: 1 (version) + 12 (nonce) + 16 (GCM tag) = 29 bytes
	if len(wire) < 29 {
		return "", ErrFieldCryptoShortData
	}

	version := wire[0]

	fc.mu.Lock()
	key, err := fc.deriveVersion(version)
	fc.mu.Unlock()

	if err != nil {
		return "", fmt.Errorf("field_crypto: %w (version %d)", ErrFieldCryptoUnknownVer, version)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("field_crypto: cipher init failed: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("field_crypto: GCM init failed: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(wire) < 1+nonceSize+aesGCM.Overhead() {
		return "", ErrFieldCryptoShortData
	}

	nonce := wire[1 : 1+nonceSize]
	ciphertext := wire[1+nonceSize:]

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("field_crypto: decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// Rotate increments the key version and derives a new key.
// Old versions remain available for decryption.
func (fc *FieldCrypto) Rotate() (uint8, error) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	nextVer := fc.currentVer + 1
	if _, err := fc.deriveVersion(nextVer); err != nil {
		return 0, fmt.Errorf("field_crypto: rotation failed: %w", err)
	}

	fc.currentVer = nextVer
	log.Printf("field_crypto: rotated to key version %d", nextVer)
	return nextVer, nil
}

// CurrentVersion returns the active key version.
func (fc *FieldCrypto) CurrentVersion() uint8 {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fc.currentVer
}

// IsEncrypted checks if a string looks like a FieldCrypto-encrypted value
// (valid base64 that decodes to at least 29 bytes with a plausible version byte).
func (fc *FieldCrypto) IsEncrypted(value string) bool {
	if value == "" {
		return false
	}
	wire, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(wire) < 29 {
		return false
	}
	// Version byte should be 1-255 (0 is never used)
	return wire[0] >= 1
}

// ExportKeyMaterial returns the master key material as a hex string for backup.
// This is the ONLY way to recover encrypted data if hardware changes.
// Store this value securely (e.g., password manager, hardware token).
func (fc *FieldCrypto) ExportKeyMaterial() string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fmt.Sprintf("%x", fc.masterKey)
}

// ExportSalt returns the salt as a hex string (needed alongside master key for recovery).
func (fc *FieldCrypto) ExportSalt() string {
	fc.mu.RLock()
	defer fc.mu.RUnlock()
	return fmt.Sprintf("%x", fc.salt)
}

// ImportKeyMaterial creates a FieldCrypto from previously exported master key hex + salt hex.
// Use this to recover encryption after hardware changes.
func ImportKeyMaterial(masterHex, saltHex string) (*FieldCrypto, error) {
	masterKey, err := hexDecode(masterHex)
	if err != nil || len(masterKey) < 16 {
		return nil, fmt.Errorf("field_crypto: invalid master key hex (need 32+ hex chars)")
	}

	saltBytes, err := hexDecode(saltHex)
	if err != nil || len(saltBytes) != fieldCryptoSaltSize {
		return nil, fmt.Errorf("field_crypto: invalid salt hex (need %d bytes)", fieldCryptoSaltSize)
	}

	fc := &FieldCrypto{
		keys:       make(map[uint8][]byte),
		currentVer: 1,
		masterKey:  masterKey,
		salt:       saltBytes,
	}

	// Derive version 1 key
	if _, err := fc.deriveVersion(1); err != nil {
		return nil, fmt.Errorf("field_crypto: key derivation failed: %w", err)
	}

	return fc, nil
}

// hexDecode decodes a hex string to bytes.
func hexDecode(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, fmt.Errorf("odd hex length")
	}
	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		b, err := hexByte(s[i], s[i+1])
		if err != nil {
			return nil, err
		}
		result[i/2] = b
	}
	return result, nil
}

// hexByte converts two hex character bytes to a single byte value.
func hexByte(hi, lo byte) (byte, error) {
	h, ok1 := hexVal(hi)
	l, ok2 := hexVal(lo)
	if !ok1 || !ok2 {
		return 0, fmt.Errorf("invalid hex")
	}
	return (h << 4) | l, nil
}

// loadOrCreateSalt reads salt from .field_crypto_salt file, or creates one with random bytes.
// Checks exe directory first (portable deployment), then AppData (installers
// put the exe under Program Files, read-only for standard users).
// Uses atomic write (write-then-rename) to prevent partial-write corruption.
func loadOrCreateSalt() ([]byte, error) {
	const saltFileName = ".field_crypto_salt"

	// 3-PLAT: build candidate paths — exe-adjacent first (portable deployment),
	// then AppData (installers put the exe under Program Files, which is
	// read-only for standard users; exe-dir-only meant silent field-encryption
	// failure there).
	candidates := make([]string, 0, 2)
	if exePath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(resolved), saltFileName))
		}
	}
	if dataDir := deploy.DataDir(); dataDir != "" {
		candidates = append(candidates, filepath.Join(dataDir, saltFileName))
	}

	// Try to read existing salt from any known location
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil && len(data) == fieldCryptoSaltSize {
			return data, nil
		}
	}

	// Find a writable location for the new salt
	saltFile := ""
	for _, path := range candidates {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}
		testPath := path + ".writetest"
		if err := os.WriteFile(testPath, []byte{0}, 0600); err == nil {
			os.Remove(testPath)
			saltFile = path
			break
		}
	}
	if saltFile == "" {
		return nil, fmt.Errorf("no writable location for salt file (tried %v)", candidates)
	}

	// Generate new random salt
	salt := make([]byte, fieldCryptoSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate random salt: %w", err)
	}

	// Atomic write: write to temp file, then rename (prevents partial-write corruption)
	tmpFile := saltFile + ".tmp"
	if err := os.WriteFile(tmpFile, salt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write temp salt file: %w", err)
	}
	if err := os.Rename(tmpFile, saltFile); err != nil {
		os.Remove(tmpFile) // cleanup on failure
		return nil, fmt.Errorf("failed to rename salt file: %w", err)
	}

	log.Printf("field_crypto: generated new random salt file at %s", saltFile)
	return salt, nil
}

func hexVal(b byte) (byte, bool) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', true
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, true
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, true
	default:
		return 0, false
	}
}
