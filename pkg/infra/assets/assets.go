// Package assets owns database-backed binary assets (letterhead, logos,
// stamps): base64 storage, cache-file extraction, and the placeholder
// letterhead used when no branded artwork is bundled (the open-source
// build intentionally ships none).
//
// Wave 5 A.1: a W4-D1 peel from the root assets_service.go. The service
// needs the database and a cache directory; host-owned concerns stay in
// root (RBAC guards, application-path discovery for default seeding, the
// filesystem letterhead fallback).
package assets

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"
)

// Asset represents a binary asset stored in the database.
type Asset struct {
	ID          string    `gorm:"primaryKey;size:50" json:"id"`
	Name        string    `gorm:"size:255;uniqueIndex" json:"name"`
	Description string    `gorm:"size:500" json:"description"`
	MimeType    string    `gorm:"size:100" json:"mime_type"`
	Data        string    `gorm:"type:text" json:"data"` // Base64 encoded
	Size        int64     `json:"size"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName ensures the table is named 'assets'
func (Asset) TableName() string {
	return "assets"
}

// Info is returned when listing assets (without the data).
type Info struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MimeType    string    `json:"mime_type"`
	Size        int64     `json:"size"`
	CreatedAt   time.Time `json:"created_at"`
}

// Known asset names.
const (
	Letterhead    = "letterhead"
	LetterheadAHS = "letterhead_ahs"
	Logo          = "company_logo"
	Stamp         = "company_stamp"
)

// Service is the binary-asset store.
type Service struct {
	db       *gorm.DB
	cacheDir string
}

func New(db *gorm.DB, cacheDir string) *Service { return &Service{db: db, cacheDir: cacheDir} }

// EnsureTable creates the assets table if it doesn't exist.
func (s *Service) EnsureTable() error {
	return s.db.AutoMigrate(&Asset{})
}

// UpsertFromFile stores a binary file as a base64-encoded asset.
func (s *Service) UpsertFromFile(name, description, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	return s.UpsertBytes(name, description, MimeTypeForExtension(filepath.Ext(filePath)), data)
}

// UpsertBytes stores raw asset bytes in the database (create or update).
func (s *Service) UpsertBytes(name, description, mimeType string, data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)

	asset := Asset{
		ID:          name, // Use name as ID for easy lookup
		Name:        name,
		Description: description,
		MimeType:    mimeType,
		Data:        encoded,
		Size:        int64(len(data)),
	}

	result := s.db.Where("name = ?", name).First(&Asset{})
	if result.Error != nil {
		if err := s.db.Create(&asset).Error; err != nil {
			return fmt.Errorf("failed to create asset: %w", err)
		}
		log.Printf("Created asset: %s (%d bytes)", name, len(data))
	} else {
		if err := s.db.Model(&Asset{}).Where("name = ?", name).Updates(asset).Error; err != nil {
			return fmt.Errorf("failed to update asset: %w", err)
		}
		log.Printf("Updated asset: %s (%d bytes)", name, len(data))
	}

	return nil
}

// Get retrieves an asset's raw bytes.
func (s *Service) Get(name string) ([]byte, error) {
	var asset Asset
	if err := s.db.Where("name = ?", name).First(&asset).Error; err != nil {
		return nil, fmt.Errorf("asset not found: %s", name)
	}

	data, err := base64.StdEncoding.DecodeString(asset.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode asset: %w", err)
	}

	return data, nil
}

// GetToFile retrieves an asset and saves it under the cache directory,
// returning the path (useful for PDF generation which needs a file path).
func (s *Service) GetToFile(name string) (string, error) {
	data, err := s.Get(name)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(s.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache dir: %w", err)
	}

	var asset Asset
	s.db.Where("name = ?", name).First(&asset)
	ext := ExtensionForMimeType(asset.MimeType)

	cachePath := filepath.Join(s.cacheDir, name+ext)
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write cache file: %w", err)
	}

	return cachePath, nil
}

// List returns info about all stored assets (without the data).
func (s *Service) List() ([]Info, error) {
	var assets []Asset
	if err := s.db.Find(&assets).Error; err != nil {
		return nil, err
	}

	infos := make([]Info, len(assets))
	for i, asset := range assets {
		infos[i] = Info{
			ID:          asset.ID,
			Name:        asset.Name,
			Description: asset.Description,
			MimeType:    asset.MimeType,
			Size:        asset.Size,
			CreatedAt:   asset.CreatedAt,
		}
	}
	return infos, nil
}

// Delete removes an asset and its cached file.
func (s *Service) Delete(name string) error {
	result := s.db.Where("name = ?", name).Delete(&Asset{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("asset not found: %s", name)
	}

	files, _ := filepath.Glob(filepath.Join(s.cacheDir, name+".*"))
	for _, f := range files {
		os.Remove(f)
	}

	return nil
}

// Has checks if an asset exists in the database.
func (s *Service) Has(name string) bool {
	var count int64
	s.db.Model(&Asset{}).Where("name = ?", name).Count(&count)
	return count > 0
}

// EnsureDefaultLetterhead uploads a default letterhead if absent: the
// first existing candidate path wins; with no artwork found, a generated
// placeholder is seeded so document generation works out of the box.
func (s *Service) EnsureDefaultLetterhead(assetName, letterheadFile, description string, candidatePaths []string) {
	if s.Has(assetName) {
		log.Printf("%s asset already in database", assetName)
		return
	}

	for _, path := range candidatePaths {
		if _, err := os.Stat(path); err == nil {
			if err := s.UpsertFromFile(assetName, description, path); err != nil {
				log.Printf("Warning: Failed to upload letterhead: %v", err)
			} else {
				log.Printf("%s uploaded to database from: %s", assetName, path)
			}
			return
		}
	}

	// No bundled letterhead artwork found (expected in the open-source build).
	if placeholder := RenderPlaceholderLetterheadPNG(); len(placeholder) > 0 {
		if err := s.UpsertBytes(assetName, description, "image/png", placeholder); err != nil {
			log.Printf("Warning: Failed to seed placeholder letterhead %s: %v", assetName, err)
			return
		}
		log.Printf("%s seeded with generated placeholder (no bundled artwork found)", assetName)
		return
	}

	log.Printf("Warning: %s file not found in any location", letterheadFile)
}

// RenderPlaceholderLetterheadPNG produces a simple, dependency-free
// A4-proportioned PNG used as a default letterhead when no branded image
// is bundled.
func RenderPlaceholderLetterheadPNG() []byte {
	const width, height = 1240, 1754 // ~A4 at 150 DPI
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	band := color.RGBA{R: 0x1d, G: 0x4e, B: 0x89, A: 0xff}
	draw.Draw(img, image.Rect(0, 0, width, 120), &image.Uniform{C: band}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, height-60, width, height), &image.Uniform{C: band}, image.Point{}, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// MimeTypeForExtension maps a file extension to its MIME type.
func MimeTypeForExtension(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".svg":
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

// ExtensionForMimeType maps a MIME type to a file extension.
func ExtensionForMimeType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	case "image/svg+xml":
		return ".svg"
	default:
		return ".bin"
	}
}
