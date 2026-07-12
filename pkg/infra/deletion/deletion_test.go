package deletion

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"ph_holdings_app/pkg/kernel/actor"
)

type fakeIdentity struct {
	employee Requester
	err      error
	admin    bool
	userID   string
	display  string
	fallback string
}

func (f fakeIdentity) CurrentEmployee() (Requester, error) { return f.employee, f.err }
func (f fakeIdentity) IsAdmin() bool                       { return f.admin }
func (f fakeIdentity) UserID() string                      { return f.userID }
func (f fakeIdentity) UserDisplayName() string             { return f.display }
func (f fakeIdentity) FallbackRole() string                { return f.fallback }

type fakeNotifier struct {
	admins     []string
	deliveries []Delivery
	events     []string
	markedRead []string
}

func (f *fakeNotifier) AdminRecipients() ([]string, error) { return f.admins, nil }
func (f *fakeNotifier) Deliver(d Delivery) error {
	f.deliveries = append(f.deliveries, d)
	return nil
}
func (f *fakeNotifier) MarkRequestNotificationsRead(requestID string) {
	f.markedRead = append(f.markedRead, requestID)
}
func (f *fakeNotifier) EmitEvent(name string, payload map[string]any) {
	f.events = append(f.events, name)
}

type fakeExecutor struct {
	calls []string
	err   error
}

func (f *fakeExecutor) Execute(entityType, entityID string) error {
	f.calls = append(f.calls, entityType+"/"+entityID)
	return f.err
}

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + filepath.ToSlash(filepath.Join(t.TempDir(), "deletion.db"))
	db, err := gorm.Open(gormlite.Open(dsn), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Request{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

func operator(t *testing.T, admin bool) actor.Actor {
	t.Helper()
	authority := actor.AuthorityPropose
	if admin {
		authority = actor.AuthorityApprove
	}
	op, err := actor.New(actor.Input{ID: "op-1", DisplayName: "Op", Type: actor.TypeOperator, Authority: authority})
	if err != nil {
		t.Fatalf("actor: %v", err)
	}
	return op
}

func TestRequest_DedupsPendingAndNotifiesAdmins(t *testing.T) {
	db := testDB(t)
	notifier := &fakeNotifier{admins: []string{"admin-1", "admin-2"}}
	svc := New(db, fakeIdentity{
		employee: Requester{EmployeeID: "emp-1", EmployeeName: "Aisha", LicenseRole: "sales"},
	}, notifier, &fakeExecutor{})

	first, err := svc.Request("customer", "cust-9", "", "duplicate record")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if first.EntityLabel != "customer cust-9" {
		t.Fatalf("derived label = %q", first.EntityLabel)
	}
	if first.RequestedRole != "sales" {
		t.Fatalf("requested role = %q", first.RequestedRole)
	}

	// Admin fan-out: one broadcast delivery per admin + the UI event.
	if len(notifier.deliveries) != 2 {
		t.Fatalf("deliveries = %d, want 2", len(notifier.deliveries))
	}
	for _, d := range notifier.deliveries {
		if !d.Broadcast {
			t.Fatalf("admin fan-out delivery must broadcast")
		}
		if d.CreatedBy != "emp-1" || d.SourceID != first.ID {
			t.Fatalf("delivery attribution wrong: %+v", d)
		}
		if !strings.Contains(d.Message, "Aisha") {
			t.Fatalf("message should name the requester: %q", d.Message)
		}
	}
	if len(notifier.events) != 1 || notifier.events[0] != "notifications:new" {
		t.Fatalf("events = %v", notifier.events)
	}

	// Second identical request returns the same pending row, no new fan-out.
	second, err := svc.Request("customer", "cust-9", "", "again")
	if err != nil {
		t.Fatalf("dedup request: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("expected dedup to return the pending request")
	}
	if len(notifier.deliveries) != 2 {
		t.Fatalf("dedup must not re-notify; deliveries = %d", len(notifier.deliveries))
	}
}

func TestRequest_AdminAndUnauthenticatedRefused(t *testing.T) {
	db := testDB(t)
	adminSvc := New(db, fakeIdentity{
		employee: Requester{EmployeeID: "emp-1"},
		admin:    true,
	}, &fakeNotifier{}, &fakeExecutor{})
	if _, err := adminSvc.Request("customer", "c-1", "", ""); err == nil {
		t.Fatalf("admin sessions must be told to delete directly")
	}

	anonSvc := New(db, fakeIdentity{err: fmt.Errorf("no session")}, &fakeNotifier{}, &fakeExecutor{})
	if _, err := anonSvc.Request("customer", "c-1", "", ""); err == nil {
		t.Fatalf("unauthenticated sessions must be refused")
	}
}

func TestReview_ApproveExecutesAndNotifiesRequester(t *testing.T) {
	db := testDB(t)
	notifier := &fakeNotifier{admins: []string{"admin-1"}}
	requestSvc := New(db, fakeIdentity{
		employee: Requester{EmployeeID: "emp-1", EmployeeName: "Aisha", LicenseRole: "sales"},
	}, notifier, &fakeExecutor{})
	req, err := requestSvc.Request("customer", "cust-9", "Delta Petrochemicals", "duplicate")
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	executor := &fakeExecutor{}
	reviewSvc := New(db, fakeIdentity{
		admin: true, userID: "admin-user", display: "The Admin",
	}, notifier, executor)

	reviewed, err := reviewSvc.Review(req.ID, "approve", "verified duplicate", operator(t, true))
	if err != nil {
		t.Fatalf("review: %v", err)
	}
	if reviewed.Status != "approved" || reviewed.ReviewedBy != "admin-user" || reviewed.ReviewedAt == nil {
		t.Fatalf("review fields wrong: %+v", reviewed)
	}
	if len(executor.calls) != 1 || executor.calls[0] != "customer/cust-9" {
		t.Fatalf("executor calls = %v", executor.calls)
	}
	if len(notifier.markedRead) != 1 || notifier.markedRead[0] != req.ID {
		t.Fatalf("markedRead = %v", notifier.markedRead)
	}
	// Last delivery is the requester notification (not broadcast).
	last := notifier.deliveries[len(notifier.deliveries)-1]
	if last.EmployeeID != "emp-1" || last.Broadcast {
		t.Fatalf("requester notification wrong: %+v", last)
	}
	if !strings.Contains(last.Message, "approved") {
		t.Fatalf("requester message = %q", last.Message)
	}

	// Terminal: a second review must refuse.
	if _, err := reviewSvc.Review(req.ID, "reject", "", operator(t, true)); err == nil {
		t.Fatalf("reviewing a decided request must fail")
	}
}

func TestReview_GatesAgentsAndNonAdmins(t *testing.T) {
	db := testDB(t)
	notifier := &fakeNotifier{admins: []string{"admin-1"}}
	requestSvc := New(db, fakeIdentity{
		employee: Requester{EmployeeID: "emp-1", LicenseRole: "sales"},
	}, notifier, &fakeExecutor{})
	req, err := requestSvc.Request("order", "ord-1", "", "")
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	// Non-admin session refused before the kernel gate.
	nonAdmin := New(db, fakeIdentity{admin: false}, notifier, &fakeExecutor{})
	if _, err := nonAdmin.Review(req.ID, "approve", "", operator(t, true)); err == nil {
		t.Fatalf("non-admin session must not review")
	}

	// Admin session but agent actor: the kernel gate refuses, nothing executes.
	agent, err := actor.New(actor.Input{ID: "bot-1", DisplayName: "Butler", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})
	if err != nil {
		t.Fatalf("agent actor: %v", err)
	}
	executor := &fakeExecutor{}
	adminSvc := New(db, fakeIdentity{admin: true, userID: "admin-user"}, notifier, executor)
	if _, err := adminSvc.Review(req.ID, "approve", "", agent); err == nil {
		t.Fatalf("agent actor must never pass the approval gate")
	}
	if len(executor.calls) != 0 {
		t.Fatalf("agent-refused review must not execute the delete")
	}

	var persisted Request
	if err := db.First(&persisted, "id = ?", req.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if persisted.Status != "pending" {
		t.Fatalf("refused review must leave the request pending, got %q", persisted.Status)
	}
}

func TestReview_ExecutorFailureLeavesRequestPending(t *testing.T) {
	db := testDB(t)
	notifier := &fakeNotifier{admins: []string{"admin-1"}}
	requestSvc := New(db, fakeIdentity{
		employee: Requester{EmployeeID: "emp-1", LicenseRole: "sales"},
	}, notifier, &fakeExecutor{})
	req, err := requestSvc.Request("payment", "pay-1", "", "")
	if err != nil {
		t.Fatalf("request: %v", err)
	}

	adminSvc := New(db, fakeIdentity{admin: true}, notifier, &fakeExecutor{err: fmt.Errorf("boom")})
	if _, err := adminSvc.Review(req.ID, "approve", "", operator(t, true)); err == nil {
		t.Fatalf("executor failure must surface")
	}
	var persisted Request
	if err := db.First(&persisted, "id = ?", req.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if persisted.Status != "pending" {
		t.Fatalf("failed delete must leave the request pending, got %q", persisted.Status)
	}
}
