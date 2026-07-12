// Package device owns device identity and the device lifecycle records:
// hardware fingerprinting, first-setup/pending registration, approval-state
// queries, and block/unblock transitions.
//
// Wave 5 A.1: a PARTIAL W4-D1 peel from the root device_service.go — the
// db-only logic moved. The auth-entangled flows (SetupAdminAccount,
// ApproveDevice, LoginDevice) stay with the host: they mint users, mutate
// the session, and lean on the auth/RBAC hub, which per W4-D9 gets ports,
// never relocation.
package device

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	shareddomain "ph_holdings_app/pkg/domain"
	"ph_holdings_app/pkg/infra"
)

// RegistrationResult represents the result of device registration.
type RegistrationResult struct {
	DeviceID     string   `json:"device_id"`
	MachineID    string   `json:"machine_id"`
	Status       string   `json:"status"` // "first_setup", "pending", "approved", "blocked"
	IsFirstSetup bool     `json:"is_first_setup"`
	UserID       string   `json:"user_id,omitempty"`
	UserName     string   `json:"user_name,omitempty"`
	RoleName     string   `json:"role_name,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
}

// MachineID generates a unique hardware fingerprint for this device.
// Uses MAC address + hostname, hashed for privacy.
func MachineID() string {
	log.Println("🖥️ GetMachineID: Starting...")
	var identifiers []string

	log.Println("🖥️ GetMachineID: Getting hostname...")
	hostname, err := os.Hostname()
	if err == nil {
		identifiers = append(identifiers, strings.TrimSpace(strings.ToLower(hostname)))
		log.Printf("🖥️ GetMachineID: Hostname=%s", hostname)
	} else {
		log.Printf("🖥️ GetMachineID: Hostname error: %v", err)
	}

	// Get MAC addresses with timeout (can hang on Windows after updates)
	log.Println("🖥️ GetMachineID: Starting network interfaces check (2s timeout)...")
	type ifaceResult struct {
		addrs []string
	}
	ifaceChan := make(chan ifaceResult, 1)

	go func() {
		var addrs []string
		interfaces, err := net.Interfaces()
		if err == nil {
			for _, iface := range interfaces {
				// Skip loopback and interfaces without MAC
				if iface.Flags&net.FlagLoopback != 0 || len(iface.HardwareAddr) == 0 {
					continue
				}
				addrs = append(addrs, iface.HardwareAddr.String())
			}
		}
		ifaceChan <- ifaceResult{addrs: addrs}
	}()

	select {
	case result := <-ifaceChan:
		sort.Strings(result.addrs)
		log.Printf("🖥️ GetMachineID: Got %d MAC addresses", len(result.addrs))
		identifiers = append(identifiers, result.addrs...)
	case <-time.After(2 * time.Second):
		log.Println("⚠️ GetMachineID: Network interfaces check timed out (2s), using hostname only")
		// Still works - hostname + OS provides uniqueness
	}

	identifiers = append(identifiers, runtime.GOOS, runtime.GOARCH)
	log.Printf("🖥️ GetMachineID: Building hash from %d identifiers", len(identifiers))

	result := HashIdentifiers(identifiers)
	log.Printf("🖥️ GetMachineID: Complete, hash=%s...", result[:16])
	return result
}

// HashIdentifiers folds machine identifiers into the fingerprint hash.
func HashIdentifiers(identifiers []string) string {
	combined := strings.Join(identifiers, "|")
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// Info returns the current device's name and OS description.
func Info() (name string, osInfo string) {
	hostname, _ := os.Hostname()
	osInfo = fmt.Sprintf("%s %s", runtime.GOOS, runtime.GOARCH)
	return hostname, osInfo
}

// ParsePermissions parses the simple JSON-array permission string stored
// on a role into its entries.
func ParsePermissions(raw string) []string {
	var perms []string
	if raw == "" {
		return perms
	}
	permStr := strings.Trim(raw, "[]")
	for _, p := range strings.Split(permStr, ",") {
		p = strings.Trim(p, ` "`)
		if p != "" {
			perms = append(perms, p)
		}
	}
	return perms
}

// Service is the device lifecycle service.
type Service struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Service { return &Service{db: db} }

// Register registers the current device and returns its status. Called on
// every app startup: the first device ever becomes "first_setup", later
// unknown devices become "pending", and known devices refresh last-seen.
func (s *Service) Register() (*RegistrationResult, error) {
	machineID := MachineID()
	deviceName, osInfo := Info()

	log.Printf("🔌 Device registration check: machineID=%s...", machineID[:16])

	var dev infra.Device
	result := s.db.Where("machine_id = ?", machineID).First(&dev)

	if result.Error != nil {
		// Device doesn't exist - check if this is the first device ever
		var deviceCount int64
		s.db.Model(&infra.Device{}).Count(&deviceCount)

		if deviceCount == 0 {
			// FIRST INSTALLATION - Create device in "first_setup" state
			log.Println("🎉 First installation detected! Creating admin device...")

			dev = infra.Device{
				Base:          shareddomain.Base{ID: uuid.New().String()},
				MachineID:     machineID,
				DeviceName:    deviceName,
				OSInfo:        osInfo,
				FirstSeenAt:   time.Now(),
				Status:        "first_setup", // Special status for first setup
				IsAdminDevice: true,
			}

			log.Printf("📝 Creating device with ID=%s, MachineID=%s, Status=%s", dev.ID, dev.MachineID[:16], dev.Status)
			createResult := s.db.Create(&dev)
			if createResult.Error != nil {
				log.Printf("❌ Device creation FAILED: %v", createResult.Error)
				return nil, fmt.Errorf("failed to create admin device: %w", createResult.Error)
			}
			log.Printf("✅ Device created successfully! RowsAffected=%d", createResult.RowsAffected)

			return &RegistrationResult{
				DeviceID:     dev.ID,
				MachineID:    machineID,
				Status:       "first_setup",
				IsFirstSetup: true,
			}, nil
		}

		// Not first device - create as pending
		log.Println("📝 New device detected, creating pending registration...")

		dev = infra.Device{
			Base:        shareddomain.Base{ID: uuid.New().String()},
			MachineID:   machineID,
			DeviceName:  deviceName,
			OSInfo:      osInfo,
			FirstSeenAt: time.Now(),
			Status:      "pending",
		}

		if err := s.db.Create(&dev).Error; err != nil {
			return nil, fmt.Errorf("failed to create pending device: %w", err)
		}

		return &RegistrationResult{
			DeviceID:  dev.ID,
			MachineID: machineID,
			Status:    "pending",
		}, nil
	}

	// Device exists - update last seen
	now := time.Now()
	s.db.Model(&dev).Update("last_seen_at", now)

	// Get user info if device is approved
	if dev.Status == "approved" {
		var deviceUser infra.DeviceUser
		if err := s.db.Preload("User").Preload("User.Role").
			Where("device_id = ? AND is_primary = ?", dev.ID, true).
			First(&deviceUser).Error; err == nil {

			return &RegistrationResult{
				DeviceID:    dev.ID,
				MachineID:   machineID,
				Status:      dev.Status,
				UserID:      deviceUser.UserID,
				UserName:    deviceUser.User.FullName,
				RoleName:    deviceUser.User.Role.DisplayName,
				Permissions: ParsePermissions(deviceUser.User.Role.Permissions),
			}, nil
		}
	}

	return &RegistrationResult{
		DeviceID:  dev.ID,
		MachineID: machineID,
		Status:    dev.Status,
	}, nil
}

// Current returns the current device's record.
func (s *Service) Current() (*infra.Device, error) {
	machineID := MachineID()

	var dev infra.Device
	if err := s.db.Where("machine_id = ?", machineID).First(&dev).Error; err != nil {
		return nil, fmt.Errorf("device not registered")
	}
	return &dev, nil
}

// ListPending returns devices awaiting approval.
func (s *Service) ListPending() ([]infra.Device, error) {
	var devices []infra.Device
	if err := s.db.Where("status = ?", "pending").
		Order("first_seen_at DESC").
		Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to list pending devices: %w", err)
	}
	return devices, nil
}

// ListAll returns all devices with approver names populated.
func (s *Service) ListAll() ([]infra.Device, error) {
	var devices []infra.Device
	if err := s.db.Order("status, first_seen_at DESC").Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("failed to list devices: %w", err)
	}

	for i := range devices {
		if devices[i].ApprovedBy != "" {
			var approver infra.User
			if err := s.db.First(&approver, "id = ?", devices[i].ApprovedBy).Error; err == nil {
				devices[i].ApproverName = approver.FullName
			}
		}
	}

	return devices, nil
}

// Block blocks a device from accessing the system. The admin device can
// never be blocked.
func (s *Service) Block(deviceID string) (*infra.Device, error) {
	var dev infra.Device
	if err := s.db.First(&dev, "id = ?", deviceID).Error; err != nil {
		return nil, fmt.Errorf("device not found")
	}

	if dev.IsAdminDevice {
		return nil, fmt.Errorf("cannot block the admin device")
	}

	if err := s.db.Model(&dev).Update("status", "blocked").Error; err != nil {
		return nil, fmt.Errorf("failed to block device: %w", err)
	}

	log.Printf("🚫 Device blocked: %s (%s)", dev.DeviceName, dev.ID)
	return &dev, nil
}

// Unblock unblocks a previously blocked device.
func (s *Service) Unblock(deviceID string) error {
	var dev infra.Device
	if err := s.db.First(&dev, "id = ?", deviceID).Error; err != nil {
		return fmt.Errorf("device not found")
	}

	if dev.Status != "blocked" {
		return fmt.Errorf("device is not blocked")
	}

	if err := s.db.Model(&dev).Update("status", "approved").Error; err != nil {
		return fmt.Errorf("failed to unblock device: %w", err)
	}

	log.Printf("✅ Device unblocked: %s (%s)", dev.DeviceName, dev.ID)
	return nil
}

// Users returns users associated with a device.
func (s *Service) Users(deviceID string) ([]infra.DeviceUser, error) {
	var deviceUsers []infra.DeviceUser
	if err := s.db.Preload("User").Preload("User.Role").
		Where("device_id = ?", deviceID).
		Find(&deviceUsers).Error; err != nil {
		return nil, fmt.Errorf("failed to get device users: %w", err)
	}
	return deviceUsers, nil
}
