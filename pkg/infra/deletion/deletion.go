// Package deletion owns the delete-approval workflow: non-admin sessions
// request deletion, admins review, and only an approved review executes the
// underlying delete. The aggregate (Request) and the workflow logic live
// here; the host application stays behind three narrow ports — identity
// (who is asking), notification delivery (how people are told), and delete
// execution (the entity-specific delete dispatch).
//
// Wave 4 A.1: this is the reference peel shape for shrinking the App
// god-object. Unlike the handler-trampoline services (whose closures call
// back into root functions), the logic here moved inward; only genuinely
// host-owned concerns remain outside.
package deletion

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"ph_holdings_app/pkg/approvals"
	shareddomain "ph_holdings_app/pkg/domain"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/approval"
)

// Request is the delete-approval aggregate. The host application aliases
// this type so the table shape, JSON contract, and model registry are
// unchanged by the move.
type Request struct {
	shareddomain.Base
	EntityType      string     `gorm:"index;size:80" json:"entity_type"`
	EntityID        string     `gorm:"index;size:80" json:"entity_id"`
	EntityLabel     string     `gorm:"size:255" json:"entity_label"`
	RequestedBy     string     `gorm:"index;size:100" json:"requested_by"`
	RequestedByName string     `gorm:"size:255" json:"requested_by_name"`
	RequestedRole   string     `gorm:"size:50" json:"requested_role"`
	Reason          string     `gorm:"type:text" json:"reason"`
	Status          string     `gorm:"index;size:30;default:'pending'" json:"status"`
	ReviewedBy      string     `gorm:"size:100" json:"reviewed_by"`
	ReviewedByName  string     `gorm:"size:255" json:"reviewed_by_name"`
	ReviewedAt      *time.Time `json:"reviewed_at"`
	ReviewNotes     string     `gorm:"type:text" json:"review_notes"`
}

func (Request) TableName() string { return "delete_approval_requests" }

// Requester identifies the employee behind the current session.
type Requester struct {
	EmployeeID   string
	EmployeeName string
	LicenseRole  string
}

// IdentityPort answers who is driving the current session. Implemented by
// the host application (session + license machinery stays there).
type IdentityPort interface {
	// CurrentEmployee resolves the authenticated employee, erroring when the
	// session is not tied to one.
	CurrentEmployee() (Requester, error)
	// IsAdmin reports whether the session may delete directly (and review).
	IsAdmin() bool
	// UserID and UserDisplayName describe the reviewer for the audit fields.
	UserID() string
	UserDisplayName() string
	// FallbackRole is used when the employee context carries no license role.
	FallbackRole() string
}

// Delivery is one notification the workflow wants delivered. The host
// supplies the mechanics (notification row, receipts, sync queue); the
// workflow supplies the content.
type Delivery struct {
	EmployeeID string
	CreatedBy  string
	Title      string
	Message    string
	SourceID   string
	Payload    map[string]any
	// Broadcast asks the host to also enqueue the notification for
	// collaborative sync (the admin fan-out historically did; the requester
	// notification did not).
	Broadcast bool
}

// NotifierPort delivers workflow notifications through the host.
type NotifierPort interface {
	// AdminRecipients lists employee IDs that hold admin/developer access.
	AdminRecipients() ([]string, error)
	Deliver(d Delivery) error
	// MarkRequestNotificationsRead marks every notification for the given
	// request as read (called when a review lands).
	MarkRequestNotificationsRead(requestID string)
	// EmitEvent publishes a UI collaboration event (e.g. "notifications:new").
	EmitEvent(name string, payload map[string]any)
}

// ExecutorPort performs the approved delete. The entity-type dispatch stays
// with the host — it fans out into every domain's delete method.
type ExecutorPort interface {
	Execute(entityType, entityID string) error
}

// Service is the delete-approval workflow.
type Service struct {
	db       *gorm.DB
	identity IdentityPort
	notifier NotifierPort
	executor ExecutorPort
	now      func() time.Time
}

func New(db *gorm.DB, identity IdentityPort, notifier NotifierPort, executor ExecutorPort) *Service {
	return &Service{db: db, identity: identity, notifier: notifier, executor: executor, now: time.Now}
}

// Request files a delete-approval request for the current (non-admin)
// employee. An identical pending request is returned instead of duplicated.
func (s *Service) Request(entityType, entityID, entityLabel, reason string) (*Request, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	entityType = strings.TrimSpace(entityType)
	entityID = strings.TrimSpace(entityID)
	entityLabel = strings.TrimSpace(entityLabel)
	if entityType == "" || entityID == "" {
		return nil, fmt.Errorf("delete approval requires an entity type and id")
	}

	current, err := s.identity.CurrentEmployee()
	if err != nil {
		return nil, fmt.Errorf("delete approval requires an authenticated employee: %w", err)
	}
	if s.identity.IsAdmin() {
		return nil, fmt.Errorf("admin users can delete directly; approval request not needed")
	}
	if entityLabel == "" {
		entityLabel = fmt.Sprintf("%s %s", strings.ReplaceAll(entityType, "_", " "), entityID)
	}

	var existing Request
	err = s.db.Where(
		"entity_type = ? AND entity_id = ? AND requested_by = ? AND status = ?",
		entityType,
		entityID,
		current.EmployeeID,
		"pending",
	).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check pending delete request: %w", err)
	}

	request := Request{
		Base:            shareddomain.Base{CreatedBy: current.EmployeeID},
		EntityType:      entityType,
		EntityID:        entityID,
		EntityLabel:     entityLabel,
		RequestedBy:     current.EmployeeID,
		RequestedByName: current.EmployeeName,
		RequestedRole:   current.LicenseRole,
		Reason:          strings.TrimSpace(reason),
		Status:          "pending",
	}
	if request.RequestedRole == "" {
		request.RequestedRole = s.identity.FallbackRole()
	}

	if err := s.db.Create(&request).Error; err != nil {
		return nil, fmt.Errorf("failed to create delete approval request: %w", err)
	}

	s.notifyAdmins(request)
	s.notifier.EmitEvent("notifications:new", map[string]any{
		"source_type": "delete_approval",
		"source_id":   request.ID,
	})
	return &request, nil
}

// Review decides a pending request. The transition must be legal per the
// kernel approval table AND made by an actor with approve authority — agent
// actors can never pass (they are refused approve authority at construction
// and again at the transition). Approval executes the delete before the
// review is persisted, exactly as the historical flow did.
func (s *Service) Review(requestID, decision, notes string, by actor.Actor) (*Request, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if !s.identity.IsAdmin() {
		return nil, fmt.Errorf("only admin can review delete approval requests")
	}

	decision = strings.ToLower(strings.TrimSpace(decision))
	if decision != "approve" && decision != "reject" {
		return nil, fmt.Errorf("decision must be approve or reject")
	}

	var request Request
	if err := s.db.First(&request, "id = ?", strings.TrimSpace(requestID)).Error; err != nil {
		return nil, fmt.Errorf("delete approval request not found: %w", err)
	}
	if request.Status != "pending" {
		return nil, fmt.Errorf("delete request is already %s", request.Status)
	}

	if err := Gate(request, decision, notes, by); err != nil {
		return nil, err
	}

	if decision == "approve" {
		if err := s.executor.Execute(request.EntityType, request.EntityID); err != nil {
			return nil, err
		}
		request.Status = "approved"
	} else {
		request.Status = "rejected"
	}

	now := s.now()
	request.ReviewedAt = &now
	request.ReviewedBy = s.identity.UserID()
	request.ReviewedByName = s.identity.UserDisplayName()
	request.ReviewNotes = strings.TrimSpace(notes)

	if err := s.db.Save(&request).Error; err != nil {
		return nil, fmt.Errorf("failed to save delete approval review: %w", err)
	}

	s.notifier.MarkRequestNotificationsRead(request.ID)
	s.notifyRequester(request, decision)
	s.notifier.EmitEvent("notifications:updated", map[string]any{
		"source_type": "delete_approval",
		"source_id":   request.ID,
		"status":      request.Status,
	})
	return &request, nil
}

// List returns delete-approval requests filtered by status (case-insensitive;
// an empty status returns every request), newest first. This is a pure read
// — no identity/authorization checks live here. The host wrapper decides who
// may call it (Article V.2/V.4: a persistent, permission-gated queue that
// does not depend on notification read-state).
func (s *Service) List(status string) ([]Request, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := s.db.Model(&Request{}).Order("created_at DESC")
	status = strings.TrimSpace(status)
	if status != "" {
		query = query.Where("LOWER(status) = ?", strings.ToLower(status))
	}

	var requests []Request
	if err := query.Find(&requests).Error; err != nil {
		return nil, fmt.Errorf("failed to list delete approval requests: %w", err)
	}
	return requests, nil
}

// Gate runs the kernel approval gate for a delete-request review. It is pure
// (no persistence): callers keep writing the same status strings they always
// did; the gate only decides legality.
func Gate(request Request, decision, notes string, by actor.Actor) error {
	from, err := approvals.DecisionFromStatus(request.Status)
	if err != nil {
		return fmt.Errorf("delete approval request %s has %w", request.ID, err)
	}
	to := approval.DecisionRejected
	if decision == "approve" {
		to = approval.DecisionApproved
	}
	if _, err := approvals.Transition(request.ID, "delete_request", from, to, by, strings.TrimSpace(notes), time.Now().UTC()); err != nil {
		return err
	}
	return nil
}

func (s *Service) notifyAdmins(request Request) {
	adminIDs, err := s.notifier.AdminRecipients()
	if err != nil {
		log.Printf("⚠️ Failed to resolve admin employees for delete request: %v", err)
		return
	}
	if len(adminIDs) == 0 {
		log.Printf("⚠️ Delete approval request %s created but no admin employee profile was found", request.ID)
		return
	}

	payload := map[string]any{
		"action":            "delete_requested",
		"request_id":        request.ID,
		"entity_type":       request.EntityType,
		"entity_id":         request.EntityID,
		"entity_label":      request.EntityLabel,
		"actor_name":        displayOr(request.RequestedByName, "Employee"),
		"requested_by":      request.RequestedBy,
		"requested_by_role": request.RequestedRole,
	}

	for _, adminID := range adminIDs {
		err := s.notifier.Deliver(Delivery{
			EmployeeID: adminID,
			CreatedBy:  request.RequestedBy,
			Title:      "Delete approval required",
			Message: fmt.Sprintf("%s requested deletion of %s.",
				displayOr(request.RequestedByName, "An employee"),
				request.EntityLabel,
			),
			SourceID:  request.ID,
			Payload:   payload,
			Broadcast: true,
		})
		if err != nil {
			log.Printf("⚠️ Failed to create delete approval notification: %v", err)
		}
	}
}

func (s *Service) notifyRequester(request Request, decision string) {
	if strings.TrimSpace(request.RequestedBy) == "" {
		return
	}
	title := "Delete request rejected"
	message := fmt.Sprintf("Your delete request for %s was rejected.", request.EntityLabel)
	if decision == "approve" {
		title = "Delete request approved"
		message = fmt.Sprintf("Your delete request for %s was approved and completed.", request.EntityLabel)
	}
	err := s.notifier.Deliver(Delivery{
		EmployeeID: request.RequestedBy,
		CreatedBy:  request.ReviewedBy,
		Title:      title,
		Message:    message,
		SourceID:   request.ID,
		Payload: map[string]any{
			"action":       "delete_reviewed",
			"request_id":   request.ID,
			"entity_type":  request.EntityType,
			"entity_id":    request.EntityID,
			"entity_label": request.EntityLabel,
			"decision":     decision,
			"actor_name":   displayOr(request.ReviewedByName, "Admin"),
		},
	})
	if err != nil {
		log.Printf("⚠️ Failed to notify delete requester: %v", err)
	}
}

// PayloadJSON renders a delivery payload the way the historical notification
// rows stored it.
func PayloadJSON(payload map[string]any) string {
	b, _ := json.Marshal(payload)
	return string(b)
}

func displayOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
