package main

// Wave 9.8 B4: employee document-expiry tracking (visa / CPR / passport /
// permit). This is the ONE new feature of the wave.
//
// Design decisions (ratified in the Wave 9.8 spec):
//   - Documents live in a NEW CHILD TABLE (employee_documents), not columns
//     bolted onto Employee — an employee can hold many documents of the same
//     type over time (renewals), and each has its own expiry lifecycle.
//   - The document number is PII: it is NEVER stored or JSON-exposed in
//     plaintext. DocNumberEncrypted carries the FieldCrypto ciphertext
//     (json:"-"); callers only ever see the decrypted value via the DTO.
//   - Expiry scanning is idempotent via NotifiedAt: once a document has been
//     notified, a second scan is a no-op for it until NotifiedAt is cleared
//     (e.g. by a renewal, which should reset it — see UpdateEmployeeDocument).
//   - RBAC mirrors the existing Employee profile surface exactly: reads use
//     hr:view (falling back to tasks:view, same as ListEmployeeProfiles),
//     writes require hr:create/hr:update PLUS the admin-only overlay used by
//     CreateEmployeeProfile / RequestEmployeeArchive. No new permission
//     strings are introduced and no role is widened.

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// documentExpiryLookaheadDays is the window used by ScanExpiringEmployeeDocuments:
// documents expiring within this many days (and not yet notified) generate a
// notification.
const documentExpiryLookaheadDays = 60

// EmployeeDocument is the child record tracking a single identity/permit
// document for an employee (visa, CPR, passport, work permit, ...).
type EmployeeDocument struct {
	Base
	EmployeeID         string     `gorm:"index;size:36" json:"employee_id"`
	DocType            string     `gorm:"index;size:20" json:"doc_type"`           // "cpr" | "passport" | "visa" | "permit"
	PermitSubtype      string     `gorm:"size:60" json:"permit_subtype,omitempty"` // e.g. "work permit", "driving permit" — only meaningful when doc_type == "permit"
	DocNumberEncrypted string     `gorm:"type:text" json:"-"`                      // FieldCrypto ciphertext — NEVER exposed raw
	ExpiresOn          *time.Time `gorm:"index" json:"expires_on"`
	Notes              string     `gorm:"type:text" json:"notes,omitempty"`
	NotifiedAt         *time.Time `json:"notified_at,omitempty"` // dedupes the expiry notification
}

func (EmployeeDocument) TableName() string { return "employee_documents" }

// EmployeeDocumentDTO is what bound methods actually return to the frontend:
// the decrypted document number under "doc_number", plus a masked
// convenience field for list views where showing the full number isn't
// necessary. The editor surface (HR-gated) may use doc_number directly.
type EmployeeDocumentDTO struct {
	ID              string     `json:"id"`
	EmployeeID      string     `json:"employee_id"`
	DocType         string     `json:"doc_type"`
	PermitSubtype   string     `json:"permit_subtype,omitempty"`
	DocNumber       string     `json:"doc_number"`
	DocNumberMasked string     `json:"doc_number_masked"`
	ExpiresOn       *time.Time `json:"expires_on"`
	Notes           string     `json:"notes,omitempty"`
	NotifiedAt      *time.Time `json:"notified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// validEmployeeDocTypes enumerates the doc_type values accepted on write.
var validEmployeeDocTypes = map[string]bool{
	"cpr":      true,
	"passport": true,
	"visa":     true,
	"permit":   true,
}

// maskDocumentNumber keeps only the last 4 characters visible, e.g.
// "1234567890" -> "••••7890". Short values are fully masked.
func maskDocumentNumber(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return strings.Repeat("•", len(value))
	}
	return strings.Repeat("•", len(value)-4) + value[len(value)-4:]
}

// encryptDocumentNumber encrypts a plaintext document number via FieldCrypto.
// Refuses to proceed if FieldCrypto isn't wired up — PII must never fall
// back to plaintext storage.
func (a *App) encryptDocumentNumber(plaintext string) (string, error) {
	if a.fieldCrypto == nil {
		return "", fmt.Errorf("field encryption unavailable: cannot store document number")
	}
	return a.fieldCrypto.Encrypt(plaintext)
}

// decryptDocumentNumber decrypts a stored ciphertext, tolerating legacy/blank
// values defensively (never panics on malformed data).
func (a *App) decryptDocumentNumber(ciphertext string) string {
	if ciphertext == "" {
		return ""
	}
	if a.fieldCrypto == nil || !a.fieldCrypto.IsEncrypted(ciphertext) {
		// Should never happen in normal operation (writes always encrypt),
		// but don't leak a raw value that merely looks unencrypted — surface
		// nothing rather than guessing.
		return ""
	}
	plaintext, err := a.fieldCrypto.Decrypt(ciphertext)
	if err != nil {
		log.Printf("⚠️ employee_compliance: failed to decrypt document number: %v", err)
		return ""
	}
	return plaintext
}

func toEmployeeDocumentDTO(a *App, doc EmployeeDocument) EmployeeDocumentDTO {
	plaintext := a.decryptDocumentNumber(doc.DocNumberEncrypted)
	return EmployeeDocumentDTO{
		ID:              doc.ID,
		EmployeeID:      doc.EmployeeID,
		DocType:         doc.DocType,
		PermitSubtype:   doc.PermitSubtype,
		DocNumber:       plaintext,
		DocNumberMasked: maskDocumentNumber(plaintext),
		ExpiresOn:       doc.ExpiresOn,
		Notes:           doc.Notes,
		NotifiedAt:      doc.NotifiedAt,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}
}

// requireHRWriteAccess mirrors the CreateEmployeeProfile / RequestEmployeeArchive
// overlay exactly: the base permission PLUS an admin-only session. Reused as
// written — no new permission strings, no role widening.
func (a *App) requireHRWriteAccess(permission string) error {
	if err := a.requirePermission(permission); err != nil {
		return err
	}
	if !a.currentSessionHasAdminRoleOnly() {
		return fmt.Errorf("only admin can manage employee compliance documents")
	}
	return nil
}

// requireHRReadAccess mirrors ListEmployeeProfiles: hr:view, falling back to
// tasks:view.
func (a *App) requireHRReadAccess() error {
	if err := a.requirePermission("hr:view"); err != nil {
		if taskErr := a.requirePermission("tasks:view"); taskErr != nil {
			return err
		}
	}
	return nil
}

// CreateEmployeeDocument records a new compliance document for an employee.
// The document number is encrypted before it ever touches the database.
func (a *App) CreateEmployeeDocument(doc EmployeeDocument, docNumber string) (EmployeeDocumentDTO, error) {
	if err := a.requireHRWriteAccess("hr:create"); err != nil {
		return EmployeeDocumentDTO{}, err
	}
	if a.db == nil {
		return EmployeeDocumentDTO{}, fmt.Errorf("database not initialized")
	}

	doc.EmployeeID = strings.TrimSpace(doc.EmployeeID)
	if doc.EmployeeID == "" {
		return EmployeeDocumentDTO{}, fmt.Errorf("employee id is required")
	}
	doc.DocType = strings.ToLower(strings.TrimSpace(doc.DocType))
	if !validEmployeeDocTypes[doc.DocType] {
		return EmployeeDocumentDTO{}, fmt.Errorf("doc_type must be one of cpr, passport, visa, permit")
	}
	docNumber = strings.TrimSpace(docNumber)
	if docNumber == "" {
		return EmployeeDocumentDTO{}, fmt.Errorf("document number is required")
	}

	encrypted, err := a.encryptDocumentNumber(docNumber)
	if err != nil {
		return EmployeeDocumentDTO{}, fmt.Errorf("failed to secure document number: %w", err)
	}
	doc.DocNumberEncrypted = encrypted
	// Reset any stale notification state on (re)creation.
	doc.NotifiedAt = nil
	doc.CreatedBy = a.getCurrentUserID()

	if err := a.db.Create(&doc).Error; err != nil {
		return EmployeeDocumentDTO{}, fmt.Errorf("failed to create employee document: %w", err)
	}

	if payload, err := json.Marshal(doc); err == nil {
		a.enqueueCollaborativeOperation("employee_document", doc.ID, "create", string(payload))
	}
	a.emitCollaborationEvent("employee_documents:updated", map[string]any{
		"employee_id": doc.EmployeeID,
		"document_id": doc.ID,
		"action":      "create",
	})
	a.queueCollaborativeSync("employee_document_create")

	return toEmployeeDocumentDTO(a, doc), nil
}

// UpdateEmployeeDocument updates an existing document. If docNumber is
// non-empty it is re-encrypted; an empty docNumber leaves the stored value
// untouched (so callers can update just the expiry date, for instance).
// Any change to ExpiresOn clears NotifiedAt so a renewal re-arms the alert.
func (a *App) UpdateEmployeeDocument(documentID string, updates EmployeeDocument, docNumber string) (EmployeeDocumentDTO, error) {
	if err := a.requireHRWriteAccess("hr:update"); err != nil {
		return EmployeeDocumentDTO{}, err
	}
	if a.db == nil {
		return EmployeeDocumentDTO{}, fmt.Errorf("database not initialized")
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return EmployeeDocumentDTO{}, fmt.Errorf("document id is required")
	}

	var existing EmployeeDocument
	if err := a.db.First(&existing, "id = ?", documentID).Error; err != nil {
		return EmployeeDocumentDTO{}, fmt.Errorf("employee document not found: %w", err)
	}

	if docType := strings.ToLower(strings.TrimSpace(updates.DocType)); docType != "" {
		if !validEmployeeDocTypes[docType] {
			return EmployeeDocumentDTO{}, fmt.Errorf("doc_type must be one of cpr, passport, visa, permit")
		}
		existing.DocType = docType
	}
	if strings.TrimSpace(updates.PermitSubtype) != "" {
		existing.PermitSubtype = strings.TrimSpace(updates.PermitSubtype)
	}
	if strings.TrimSpace(updates.Notes) != "" {
		existing.Notes = strings.TrimSpace(updates.Notes)
	}

	expiryChanged := false
	if updates.ExpiresOn != nil {
		if existing.ExpiresOn == nil || !existing.ExpiresOn.Equal(*updates.ExpiresOn) {
			expiryChanged = true
		}
		existing.ExpiresOn = updates.ExpiresOn
	}

	docNumber = strings.TrimSpace(docNumber)
	if docNumber != "" {
		encrypted, err := a.encryptDocumentNumber(docNumber)
		if err != nil {
			return EmployeeDocumentDTO{}, fmt.Errorf("failed to secure document number: %w", err)
		}
		existing.DocNumberEncrypted = encrypted
	}

	if expiryChanged {
		// Renewal / correction: re-arm the 60-day notification.
		existing.NotifiedAt = nil
	}

	if err := a.db.Save(&existing).Error; err != nil {
		return EmployeeDocumentDTO{}, fmt.Errorf("failed to update employee document: %w", err)
	}

	if payload, err := json.Marshal(existing); err == nil {
		a.enqueueCollaborativeOperation("employee_document", existing.ID, "update", string(payload))
	}
	a.emitCollaborationEvent("employee_documents:updated", map[string]any{
		"employee_id": existing.EmployeeID,
		"document_id": existing.ID,
		"action":      "update",
	})
	a.queueCollaborativeSync("employee_document_update")

	return toEmployeeDocumentDTO(a, existing), nil
}

// DeleteEmployeeDocument soft-deletes a document (GORM DeletedAt on Base).
func (a *App) DeleteEmployeeDocument(documentID string) error {
	if err := a.requireHRWriteAccess("hr:update"); err != nil {
		return err
	}
	if a.db == nil {
		return fmt.Errorf("database not initialized")
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return fmt.Errorf("document id is required")
	}

	var existing EmployeeDocument
	if err := a.db.First(&existing, "id = ?", documentID).Error; err != nil {
		return fmt.Errorf("employee document not found: %w", err)
	}
	if err := a.db.Delete(&existing).Error; err != nil {
		return fmt.Errorf("failed to delete employee document: %w", err)
	}

	a.enqueueCollaborativeOperation("employee_document", existing.ID, "delete", "")
	a.emitCollaborationEvent("employee_documents:updated", map[string]any{
		"employee_id": existing.EmployeeID,
		"document_id": existing.ID,
		"action":      "delete",
	})
	a.queueCollaborativeSync("employee_document_delete")
	return nil
}

// ListEmployeeDocuments returns every (non-deleted) document for an employee,
// decrypted for display. Opportunistically runs the expiry scan first (cheap
// and deduplicated via NotifiedAt) so opening this surface refreshes expiry
// notifications without needing a dedicated startup hook.
func (a *App) ListEmployeeDocuments(employeeID string) ([]EmployeeDocumentDTO, error) {
	if err := a.requireHRReadAccess(); err != nil {
		return nil, err
	}
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	employeeID = strings.TrimSpace(employeeID)
	if employeeID == "" {
		return nil, fmt.Errorf("employee id is required")
	}

	// Opportunistic refresh — errors here must never block the read.
	if _, err := a.ScanExpiringEmployeeDocuments(); err != nil {
		log.Printf("⚠️ employee_compliance: opportunistic expiry scan failed: %v", err)
	}

	var docs []EmployeeDocument
	if err := a.db.Where("employee_id = ?", employeeID).Order("expires_on ASC").Find(&docs).Error; err != nil {
		return nil, fmt.Errorf("failed to list employee documents: %w", err)
	}

	dtos := make([]EmployeeDocumentDTO, 0, len(docs))
	for _, doc := range docs {
		dtos = append(dtos, toEmployeeDocumentDTO(a, doc))
	}
	return dtos, nil
}

// EmployeeDocumentScanResult summarizes a single expiry-scan run.
type EmployeeDocumentScanResult struct {
	ScannedAt     time.Time `json:"scanned_at"`
	NotifiedCount int       `json:"notified_count"`
	LookaheadDays int       `json:"lookahead_days"`
}

// ScanExpiringEmployeeDocuments finds documents expiring within the next
// documentExpiryLookaheadDays days that haven't been notified yet, fires a
// notification for each, and stamps NotifiedAt so the scan is idempotent —
// running it again immediately is a no-op. Bound so the orchestrator can
// additionally wire it into app startup later; today it's also called
// opportunistically from ListEmployeeDocuments.
func (a *App) ScanExpiringEmployeeDocuments() (EmployeeDocumentScanResult, error) {
	result := EmployeeDocumentScanResult{ScannedAt: time.Now(), LookaheadDays: documentExpiryLookaheadDays}
	if a.db == nil {
		return result, fmt.Errorf("database not initialized")
	}

	horizon := time.Now().Add(documentExpiryLookaheadDays * 24 * time.Hour)

	var due []EmployeeDocument
	if err := a.db.
		Where("expires_on IS NOT NULL AND expires_on <= ? AND notified_at IS NULL", horizon).
		Find(&due).Error; err != nil {
		return result, fmt.Errorf("failed to scan expiring employee documents: %w", err)
	}

	now := time.Now()
	for _, doc := range due {
		a.createDocumentExpiryNotification(doc.EmployeeID, doc)
		if err := a.db.Model(&EmployeeDocument{}).Where("id = ?", doc.ID).Update("notified_at", now).Error; err != nil {
			log.Printf("⚠️ employee_compliance: failed to stamp notified_at for document %s: %v", doc.ID, err)
			continue
		}
		result.NotifiedCount++
	}

	return result, nil
}

// createDocumentExpiryNotification mirrors createTaskNotification's shape
// exactly (see collaboration_service.go): Notification{...} -> db.Create ->
// recordNotificationReceipt -> emitCollaborationEvent("notifications:new").
// This is how document expiry surfaces in the existing Article V
// notifications home without any change to NotificationsScreen.
func (a *App) createDocumentExpiryNotification(employeeID string, doc EmployeeDocument) {
	if strings.TrimSpace(employeeID) == "" || a.db == nil {
		return
	}

	docLabel := strings.ToUpper(doc.DocType)
	if doc.DocType == "permit" && strings.TrimSpace(doc.PermitSubtype) != "" {
		docLabel = fmt.Sprintf("%s (%s)", docLabel, doc.PermitSubtype)
	}

	expiryText := "soon"
	if doc.ExpiresOn != nil {
		expiryText = doc.ExpiresOn.Format("2 Jan 2006")
	}

	title := "Document expiring soon"
	message := fmt.Sprintf("%s document expires %s", docLabel, expiryText)

	payload, _ := json.Marshal(map[string]any{
		"document_id": doc.ID,
		"employee_id": employeeID,
		"doc_type":    doc.DocType,
		"expires_on":  doc.ExpiresOn,
	})

	notification := Notification{
		Base:             Base{CreatedBy: a.getCurrentUserID()},
		EmployeeID:       employeeID,
		NotificationType: "document_expiry",
		Title:            title,
		Message:          message,
		Status:           "unread",
		SourceType:       "employee_document",
		SourceID:         doc.ID,
		ActionRoute:      "#people",
		ActionPayload:    string(payload),
	}
	if err := a.db.Create(&notification).Error; err != nil {
		log.Printf("⚠️ Failed to create document expiry notification: %v", err)
		return
	}

	a.recordNotificationReceipt(notification.ID, employeeID, "", "created")
	if queuePayload, err := json.Marshal(notification); err == nil {
		a.enqueueCollaborativeOperation("notification", notification.ID, "create", string(queuePayload))
	}
	a.emitCollaborationEvent("notifications:new", map[string]any{
		"notification_id": notification.ID,
		"employee_id":     employeeID,
		"source_id":       doc.ID,
		"source_type":     "employee_document",
	})
}
