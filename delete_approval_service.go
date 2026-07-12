package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"ph_holdings_app/pkg/infra/deletion"
	"ph_holdings_app/pkg/kernel/actor"
)

// DeleteApprovalRequest is owned by pkg/infra/deletion (Wave 4 A.1); the
// alias keeps the model registry, JSON contract, and existing callers
// unchanged.
type DeleteApprovalRequest = deletion.Request

func isDeletePermission(permission string) bool {
	parts := strings.Split(strings.TrimSpace(permission), ":")
	return len(parts) == 2 && parts[1] == "delete"
}

func (a *App) currentSessionHasAdminRoleOnly() bool {
	if a != nil {
		switch strings.ToLower(strings.TrimSpace(a.GetLicenseRole())) {
		case "admin", "administrator", "developer":
			return true
		}
		if a.HasLicensePermission("*") {
			return true
		}
	}

	roleCandidates := make([]string, 0, 4)
	if a != nil && a.currentUser != nil {
		if err := a.hydrateUserRole(a.currentUser); err != nil {
			log.Printf("⚠️ Failed to hydrate current user for admin check: %v", err)
		}
		roleCandidates = append(roleCandidates,
			a.currentUser.Role.Name,
			a.currentUser.RoleName,
			a.currentUser.Role.DisplayName,
		)
		rolePerms := strings.TrimSpace(a.currentUser.Role.Permissions)
		if rolePerms == `["*"]` || strings.Contains(rolePerms, `"*"`) {
			return true
		}
	}
	if a != nil {
		roleCandidates = append(roleCandidates, a.GetCurrentUserRole())
	}
	for _, candidate := range roleCandidates {
		switch strings.ToLower(strings.TrimSpace(candidate)) {
		case "admin", "administrator", "developer":
			return true
		}
	}
	return false
}

func (a *App) guardDeleteOrRequest(permission, entityType, entityID, entityLabel string) (bool, error) {
	if a.currentSessionHasAdminRoleOnly() {
		if err := a.requirePermission(permission); err != nil {
			return false, err
		}
		return true, nil
	}

	request, err := a.RequestDeleteApproval(entityType, entityID, entityLabel, "")
	if err != nil {
		return false, err
	}
	return false, fmt.Errorf("delete approval requested: request %s sent to admin for approval", request.ID)
}

func (a *App) RequestDeleteApproval(entityType, entityID, entityLabel, reason string) (*DeleteApprovalRequest, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	return a.deletionService().Request(entityType, entityID, entityLabel, reason)
}

func (a *App) ReviewDeleteApprovalRequest(requestID, decision, notes string) (*DeleteApprovalRequest, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	by, err := a.currentApprovalActor()
	if err != nil {
		return nil, fmt.Errorf("delete approval reviewer identity: %w", err)
	}
	return a.deletionService().Review(requestID, decision, notes, by)
}

// ListDeleteApprovalRequests returns delete-approval requests for the
// reviewer queue (Article V.2/V.4: a persistent list that does not depend on
// notification read-state — reading a notification must never strand a
// pending approval). Read-only; gated the same way Service.Review authorizes
// (admin/reviewer sessions only). Non-admin callers get an empty list rather
// than an error so the frontend can poll this safely from any session.
func (a *App) ListDeleteApprovalRequests(status string) ([]DeleteApprovalRequest, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if !a.currentSessionHasAdminRoleOnly() {
		return []DeleteApprovalRequest{}, nil
	}
	return a.deletionService().List(status)
}

// currentApprovalActor maps the authenticated human session onto a kernel
// actor for approval transitions. Sessions are operator actors; only an
// admin session carries approve authority. AI/agent code paths must mint
// their own TypeAgent actors — the kernel then refuses them approve power at
// construction AND pkg/approvals refuses the transition (belt and braces).
func (a *App) currentApprovalActor() (actor.Actor, error) {
	authority := actor.AuthorityPropose
	if a.currentSessionHasAdminRoleOnly() {
		authority = actor.AuthorityApprove
	}
	id := strings.TrimSpace(a.getCurrentUserID())
	if id == "" {
		id = "unauthenticated-session"
		authority = actor.AuthorityObserve
	}
	return actor.New(actor.Input{
		ID:          id,
		DisplayName: a.getCurrentUserDisplayName(),
		Type:        actor.TypeOperator,
		Authority:   authority,
	})
}

// gateDeleteApprovalTransition is kept as a thin delegate so callers (and
// the approval-routing tests) exercise the same gate the service runs.
func (a *App) gateDeleteApprovalTransition(request DeleteApprovalRequest, decision, notes string, by actor.Actor) error {
	return deletion.Gate(request, decision, notes, by)
}

// --- deletion service ports (host side) ---

type appDeletionIdentityPort struct{ app *App }

func (p appDeletionIdentityPort) CurrentEmployee() (deletion.Requester, error) {
	current, err := p.app.GetCurrentEmployeeContext()
	if err != nil {
		return deletion.Requester{}, err
	}
	return deletion.Requester{
		EmployeeID:   current.EmployeeID,
		EmployeeName: current.EmployeeName,
		LicenseRole:  current.LicenseRole,
	}, nil
}

func (p appDeletionIdentityPort) IsAdmin() bool           { return p.app.currentSessionHasAdminRoleOnly() }
func (p appDeletionIdentityPort) UserID() string          { return p.app.getCurrentUserID() }
func (p appDeletionIdentityPort) UserDisplayName() string { return p.app.getCurrentUserDisplayName() }
func (p appDeletionIdentityPort) FallbackRole() string    { return p.app.GetCurrentUserRole() }

type appDeletionNotifierPort struct{ app *App }

func (p appDeletionNotifierPort) AdminRecipients() ([]string, error) {
	var adminIDs []string
	err := p.app.db.Table("employees").
		Select("DISTINCT employees.id").
		Joins("JOIN employee_access_links ON employee_access_links.employee_id = employees.id").
		Joins("JOIN license_keys ON license_keys.key = employee_access_links.license_key").
		Where("employees.is_active = ? AND employee_access_links.access_status = ? AND LOWER(license_keys.role) IN ?", true, "active", []string{"admin", "developer"}).
		Pluck("employees.id", &adminIDs).Error
	if err != nil {
		return nil, err
	}
	return adminIDs, nil
}

func (p appDeletionNotifierPort) Deliver(d deletion.Delivery) error {
	notification := Notification{
		Base:             Base{CreatedBy: d.CreatedBy},
		EmployeeID:       d.EmployeeID,
		NotificationType: "delete_approval",
		Title:            d.Title,
		Message:          d.Message,
		Status:           "unread",
		SourceType:       "delete_approval",
		SourceID:         d.SourceID,
		ActionRoute:      "#notifications",
		ActionPayload:    deletion.PayloadJSON(d.Payload),
	}
	if err := p.app.db.Create(&notification).Error; err != nil {
		return err
	}
	p.app.recordNotificationReceipt(notification.ID, d.EmployeeID, "", "created")
	if d.Broadcast {
		if queuePayload, err := json.Marshal(notification); err == nil {
			p.app.enqueueCollaborativeOperation("notification", notification.ID, "create", string(queuePayload))
		}
	}
	return nil
}

func (p appDeletionNotifierPort) MarkRequestNotificationsRead(requestID string) {
	p.app.markDeleteApprovalNotificationsRead(requestID)
}

func (p appDeletionNotifierPort) EmitEvent(name string, payload map[string]any) {
	p.app.emitCollaborationEvent(name, payload)
}

func (a *App) markDeleteApprovalNotificationsRead(requestID string) {
	now := time.Now()
	if err := a.db.Model(&Notification{}).
		Where("source_type = ? AND source_id = ?", "delete_approval", requestID).
		Updates(map[string]any{"status": "read", "read_at": &now}).Error; err != nil {
		log.Printf("⚠️ Failed to mark delete approval notifications read: %v", err)
	}
}

type appDeletionExecutorPort struct{ app *App }

func (p appDeletionExecutorPort) Execute(entityType, entityID string) error {
	return p.app.performApprovedDelete(entityType, entityID)
}

func (a *App) performApprovedDelete(entityType, entityID string) error {
	switch entityType {
	case "rfq":
		id, err := strconv.ParseUint(entityID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid RFQ id: %w", err)
		}
		return a.DeleteRFQ(uint(id))
	case "rfq_cascade":
		id, err := strconv.ParseUint(entityID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid RFQ id: %w", err)
		}
		_, err = a.DeleteRFQWithCascade(uint(id), true)
		return err
	case "costing_sheet":
		id, err := strconv.ParseUint(entityID, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid costing sheet id: %w", err)
		}
		return a.DeleteCostingSheet(uint(id))
	case "offer_note":
		return a.DeleteOfferNote(entityID)
	case "order":
		return a.DeleteOrder(entityID)
	case "customer_contact":
		return a.DeleteCustomerContact(entityID)
	case "supplier_contact":
		return a.DeleteSupplierContact(entityID)
	case "customer":
		return a.DeleteCustomer(entityID)
	case "supplier":
		return a.DeleteSupplier(entityID)
	case "customer_invoice":
		return a.DeleteCustomerInvoice(entityID)
	case "payment":
		return a.DeletePayment(entityID)
	case "supplier_payment":
		return a.DeleteSupplierPayment(entityID)
	case "supplier_invoice":
		return a.DeleteSupplierInvoice(entityID)
	case "purchase_order":
		return a.DeletePurchaseOrder(entityID)
	case "delivery_note":
		return a.DeleteDeliveryNote(entityID)
	case "grn":
		return a.DeleteGRN(entityID)
	case "bank_statement":
		return a.DeleteBankStatement(entityID)
	case "bank_statement_line":
		return a.DeleteBankStatementLine(entityID)
	case "expense_category":
		return a.DeleteExpenseCategory(entityID)
	case "expense_vendor":
		return a.DeleteExpenseVendor(entityID)
	case "expense_entry":
		return a.DeleteExpenseEntry(entityID)
	case "recurring_expense":
		return a.DeleteRecurringExpense(entityID)
	case "quick_capture":
		return a.DeleteQuickCapture(entityID)
	case "collaborative_task":
		return a.DeleteCollaborativeTask(entityID)
	default:
		return fmt.Errorf("delete approval cannot execute unsupported entity type %q", entityType)
	}
}
