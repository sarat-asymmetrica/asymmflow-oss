package banking

// AuthorizationPort exposes the actor and permission checks needed by banking
// logic without importing the application shell.
type AuthorizationPort interface {
	CurrentUserID() string
	HasPermission(action string) bool
}

// AuditPort records banking-domain mutations through the host application.
type AuditPort interface {
	LogAction(entityType, entityID, action, detail, userID string) error
}

// FinancialAuditPort is implemented by hosts that can record amount-aware audit
// records for destructive banking actions.
type FinancialAuditPort interface {
	LogFinancialTransaction(userID, action, entityType, entityID string, amount float64, currency string, details map[string]any) error
}

// DivisionPort resolves Acme Instrumentation/Beacon Controls ownership without coupling the
// banking package to company-branding helpers.
type DivisionPort interface {
	CurrentDivision() string
	ResolveDivision(entityID string) (string, error)
}

// DeleteApprovalPort preserves the host application's privileged delete flow
// while statement mutation logic moves into the banking service.
type DeleteApprovalPort interface {
	GuardDeleteOrRequest(permission, entityType, entityID, entityLabel string) (bool, error)
}

// Ports groups optional dependencies used as banking logic moves inward.
type Ports struct {
	Auth           AuthorizationPort
	Audit          AuditPort
	Division       DivisionPort
	DeleteApproval DeleteApprovalPort
}
