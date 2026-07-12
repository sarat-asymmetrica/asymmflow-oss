package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// maxCostingAttachmentBytes is the inline (database-stored) ceiling. PDFs larger
// than this are copied to the customer's Offers export folder and referenced by
// path; images larger than this are rejected outright.
const maxCostingAttachmentBytes = 20 * 1024 * 1024

const (
	costingAttachmentStorageDatabase  = "database"
	costingAttachmentStorageLocalFile = "local_file"
)

var costingAttachmentScopePattern = regexp.MustCompile(`[^A-Za-z0-9_.:\-]+`)

// CostingSheetAttachment is a technical datasheet (PDF / JPG / PNG) bound to a
// costing sheet or offer via a normalised ScopeID. Small files live inline as
// base64 in the database so they sync with the rest of the row; oversized PDFs
// are stored on disk with only their path persisted. Mission I item I-25.
type CostingSheetAttachment struct {
	Base
	ScopeID       string `gorm:"index;size:160" json:"scope_id"`
	CostingNumber string `gorm:"index;size:120" json:"costing_number"`
	FileName      string `gorm:"size:255" json:"file_name"`
	FileExt       string `gorm:"size:20" json:"file_ext"`
	MimeType      string `gorm:"size:100" json:"mime_type"`
	FileSize      int64  `json:"file_size"`
	FileHash      string `gorm:"index;size:64" json:"file_hash"`
	StorageMode   string `gorm:"size:30;default:'database'" json:"storage_mode"`
	LocalPath     string `gorm:"type:varchar(1000)" json:"local_path"`
	ContentBase64 string `gorm:"type:text" json:"-"`
	Notes         string `gorm:"type:text" json:"notes"`
	UploadedBy    string `gorm:"size:120" json:"uploaded_by"`
}

func (CostingSheetAttachment) TableName() string {
	return "costing_sheet_attachments"
}

// CostingSheetAttachmentSummary is the metadata-only projection returned to the
// UI and the PDF bundler; it never carries the base64 content.
type CostingSheetAttachmentSummary struct {
	ID            string    `json:"id"`
	ScopeID       string    `json:"scope_id"`
	CostingNumber string    `json:"costing_number"`
	FileName      string    `json:"file_name"`
	FileExt       string    `json:"file_ext"`
	MimeType      string    `json:"mime_type"`
	FileSize      int64     `json:"file_size"`
	FileHash      string    `json:"file_hash"`
	StorageMode   string    `json:"storage_mode"`
	LocalPath     string    `json:"local_path"`
	Notes         string    `json:"notes"`
	UploadedBy    string    `json:"uploaded_by"`
	CreatedAt     time.Time `json:"created_at"`
}

func costingAttachmentSummary(row CostingSheetAttachment) CostingSheetAttachmentSummary {
	storageMode := strings.TrimSpace(row.StorageMode)
	if storageMode == "" {
		storageMode = costingAttachmentStorageDatabase
	}
	return CostingSheetAttachmentSummary{
		ID:            row.ID,
		ScopeID:       row.ScopeID,
		CostingNumber: row.CostingNumber,
		FileName:      row.FileName,
		FileExt:       row.FileExt,
		MimeType:      row.MimeType,
		FileSize:      row.FileSize,
		FileHash:      row.FileHash,
		StorageMode:   storageMode,
		LocalPath:     row.LocalPath,
		Notes:         row.Notes,
		UploadedBy:    row.UploadedBy,
		CreatedAt:     row.CreatedAt,
	}
}

func normaliseCostingAttachmentScope(scopeID string) string {
	scope := strings.TrimSpace(scopeID)
	scope = costingAttachmentScopePattern.ReplaceAllString(scope, "-")
	scope = strings.Trim(scope, "-_.:")
	if len(scope) > 160 {
		scope = scope[:160]
	}
	return scope
}

func allowedCostingAttachmentExtension(path string) (string, string, bool) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	switch ext {
	case "pdf":
		return ext, "application/pdf", true
	case "jpg", "jpeg":
		return ext, "image/jpeg", true
	case "png":
		return ext, "image/png", true
	default:
		return ext, "", false
	}
}

func (a *App) ensureCostingAttachmentStorageReady() error {
	if a == nil || a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := a.db.AutoMigrate(&CostingSheetAttachment{}); err != nil {
		return fmt.Errorf("prepare costing attachment storage: %w", err)
	}
	return nil
}

// AttachCostingSheetFileWithDialog opens a native file picker and attaches the
// chosen datasheet. Returns (nil, nil) when the user cancels.
func (a *App) AttachCostingSheetFileWithDialog(scopeID string, costingNumber string, customerName string, notes string) (*CostingSheetAttachmentSummary, error) {
	if err := a.requireAnyPermission("offers:create", "offers:edit", "costing:update"); err != nil {
		return nil, err
	}
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Attach technical datasheet",
		Filters: []wailsRuntime.FileFilter{
			{DisplayName: "Technical datasheets", Pattern: "*.pdf;*.jpg;*.jpeg;*.png"},
			{DisplayName: "PDF Files", Pattern: "*.pdf"},
			{DisplayName: "Images", Pattern: "*.jpg;*.jpeg;*.png"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("file selection failed: %w", err)
	}
	if strings.TrimSpace(selection) == "" {
		return nil, nil
	}
	return a.AttachCostingSheetFile(scopeID, costingNumber, customerName, selection, notes)
}

// AttachCostingSheetFile validates and stores a datasheet for the given scope.
func (a *App) AttachCostingSheetFile(scopeID string, costingNumber string, customerName string, filePath string, notes string) (*CostingSheetAttachmentSummary, error) {
	if err := a.requireAnyPermission("offers:create", "offers:edit", "costing:update"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if err := a.ensureCostingAttachmentStorageReady(); err != nil {
		return nil, err
	}

	scope := normaliseCostingAttachmentScope(scopeID)
	if scope == "" {
		return nil, fmt.Errorf("costing attachment scope is required")
	}

	absPath, err := filepath.Abs(strings.TrimSpace(filePath))
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("attachment file not found: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("attachment must be a file")
	}
	if info.Size() <= 0 {
		return nil, fmt.Errorf("attachment file is empty")
	}
	ext, mimeType, ok := allowedCostingAttachmentExtension(absPath)
	if !ok {
		return nil, fmt.Errorf("unsupported attachment type .%s; use PDF, JPG, JPEG, or PNG", ext)
	}
	if detected := mime.TypeByExtension("." + ext); detected != "" {
		mimeType = detected
	}

	storageMode := costingAttachmentStorageDatabase
	localPath := ""
	contentBase64 := ""
	hash := ""

	if info.Size() > maxCostingAttachmentBytes {
		if ext != "pdf" {
			return nil, fmt.Errorf("attachment is %.1f MB; maximum allowed is 20 MB for images; oversized PDFs are stored locally", float64(info.Size())/(1024*1024))
		}
		storageMode = costingAttachmentStorageLocalFile
		var err error
		localPath, hash, err = a.copyLargeCostingPDFToCustomerOfferFolder(absPath, strings.TrimSpace(customerName), strings.TrimSpace(costingNumber), scope)
		if err != nil {
			return nil, err
		}
	} else {
		content, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read attachment: %w", err)
		}
		sum := sha256.Sum256(content)
		hash = hex.EncodeToString(sum[:])
		contentBase64 = base64.StdEncoding.EncodeToString(content)
	}

	row := CostingSheetAttachment{
		Base: Base{
			CreatedBy: a.getCurrentUserID(),
		},
		ScopeID:       scope,
		CostingNumber: strings.TrimSpace(costingNumber),
		FileName:      filepath.Base(absPath),
		FileExt:       ext,
		MimeType:      mimeType,
		FileSize:      info.Size(),
		FileHash:      hash,
		StorageMode:   storageMode,
		LocalPath:     localPath,
		ContentBase64: contentBase64,
		Notes:         strings.TrimSpace(notes),
		UploadedBy:    a.currentDisplayName(),
	}

	if err := a.db.Create(&row).Error; err != nil {
		return nil, fmt.Errorf("failed to store attachment: %w", err)
	}

	summary := costingAttachmentSummary(row)
	return &summary, nil
}

func (a *App) copyLargeCostingPDFToCustomerOfferFolder(sourcePath string, customerName string, costingNumber string, scope string) (string, string, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to read oversized PDF: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	baseDir := a.getExportDir("customer", customerName, "Offers", time.Now().Year())
	scopeFolder := sanitizeFileName(firstNonEmptyString(costingNumber, scope, "Costing_Datasheets"))
	targetDir := filepath.Join(baseDir, "Technical_Datasheets", scopeFolder)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to prepare local datasheet folder: %w", err)
	}

	targetName := sanitizeFileName(filepath.Base(sourcePath))
	if targetName == "" || targetName == "unnamed" {
		targetName = "technical_datasheet.pdf"
	}
	if !strings.HasSuffix(strings.ToLower(targetName), ".pdf") {
		targetName += ".pdf"
	}
	targetPath := uniqueCostingAttachmentPath(filepath.Join(targetDir, targetName))

	out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to create local datasheet copy: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(io.MultiWriter(out, hash), file); err != nil {
		_ = os.Remove(targetPath)
		return "", "", fmt.Errorf("failed to copy oversized PDF locally: %w", err)
	}
	if err := out.Sync(); err != nil {
		_ = os.Remove(targetPath)
		return "", "", fmt.Errorf("failed to flush oversized PDF copy: %w", err)
	}

	return targetPath, hex.EncodeToString(hash.Sum(nil)), nil
}

func uniqueCostingAttachmentPath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; i <= 999; i++ {
		candidate := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return fmt.Sprintf("%s_%d%s", base, time.Now().UnixNano(), ext)
}

// ListCostingSheetAttachments returns the datasheet metadata for a scope.
func (a *App) ListCostingSheetAttachments(scopeID string) ([]CostingSheetAttachmentSummary, error) {
	if err := a.requireAnyPermission("offers:view", "offers:create", "offers:edit", "costing:read"); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if err := a.ensureCostingAttachmentStorageReady(); err != nil {
		return nil, err
	}
	scope := normaliseCostingAttachmentScope(scopeID)
	if scope == "" {
		return []CostingSheetAttachmentSummary{}, nil
	}

	return a.listCostingSheetAttachmentsByScope(scope)
}

// listCostingSheetAttachmentsByScope is the internal, unguarded lister used by
// the PDF export/bundle path (which already gated its own entry point).
func (a *App) listCostingSheetAttachmentsByScope(scope string) ([]CostingSheetAttachmentSummary, error) {
	if err := a.ensureCostingAttachmentStorageReady(); err != nil {
		return nil, err
	}
	var rows []CostingSheetAttachment
	if err := a.db.
		Where("scope_id = ?", scope).
		Order("created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list costing attachments: %w", err)
	}

	summaries := make([]CostingSheetAttachmentSummary, 0, len(rows))
	for _, row := range rows {
		summaries = append(summaries, costingAttachmentSummary(row))
	}
	return summaries, nil
}

// DeleteCostingSheetAttachment soft-deletes an attachment (gorm DeletedAt).
func (a *App) DeleteCostingSheetAttachment(attachmentID string) error {
	if err := a.requireAnyPermission("offers:edit", "costing:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := a.ensureCostingAttachmentStorageReady(); err != nil {
		return err
	}
	id := strings.TrimSpace(attachmentID)
	if id == "" {
		return fmt.Errorf("attachment id is required")
	}
	result := a.db.Delete(&CostingSheetAttachment{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete costing attachment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// OpenCostingSheetAttachment materialises the attachment and opens it in the OS
// default handler. Local-file attachments open in place; database attachments
// are written to a temp folder first.
func (a *App) OpenCostingSheetAttachment(attachmentID string) error {
	if err := a.requireAnyPermission("offers:view", "costing:read"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	if err := a.ensureCostingAttachmentStorageReady(); err != nil {
		return err
	}
	id := strings.TrimSpace(attachmentID)
	if id == "" {
		return fmt.Errorf("attachment id is required")
	}

	var row CostingSheetAttachment
	if err := a.db.First(&row, "id = ?", id).Error; err != nil {
		return fmt.Errorf("attachment not found: %w", err)
	}

	if strings.TrimSpace(row.StorageMode) == costingAttachmentStorageLocalFile {
		localPath := strings.TrimSpace(row.LocalPath)
		if localPath == "" {
			return fmt.Errorf("local attachment path is missing")
		}
		if _, err := os.Stat(localPath); err != nil {
			return fmt.Errorf("local attachment file is not available on this device: %w", err)
		}
		return a.OpenExportedFile(localPath)
	}

	content, err := base64.StdEncoding.DecodeString(row.ContentBase64)
	if err != nil {
		return fmt.Errorf("attachment content is corrupted: %w", err)
	}

	tmpDir := filepath.Join(os.TempDir(), "asymmflow-costing-attachments")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return fmt.Errorf("failed to prepare attachment preview folder: %w", err)
	}
	fileName := sanitizeFileName(row.FileName)
	if fileName == "" {
		fileName = row.ID + "." + row.FileExt
	}
	outputPath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(outputPath, content, 0600); err != nil {
		return fmt.Errorf("failed to prepare attachment preview: %w", err)
	}
	return a.OpenExportedFile(outputPath)
}

// requireAnyPermission passes if the current session holds at least one of the
// listed permissions. It returns the last denial when none match so the error
// surfaces a real permission string.
func (a *App) requireAnyPermission(permissions ...string) error {
	var firstErr error
	for _, permission := range permissions {
		if strings.TrimSpace(permission) == "" {
			continue
		}
		if err := a.requirePermission(permission); err == nil {
			return nil
		} else if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return fmt.Errorf("access denied: no permission supplied")
}

// currentDisplayName resolves a human-readable name for audit stamping, falling
// back through the user profile to the raw ID and finally "System".
func (a *App) currentDisplayName() string {
	if a.currentUser != nil {
		if name := firstNonEmptyString(a.currentUser.DisplayName, a.currentUser.FullName, a.currentUser.Username); strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
	}
	if id := strings.TrimSpace(a.currentUserID); id != "" {
		return id
	}
	return "System"
}
