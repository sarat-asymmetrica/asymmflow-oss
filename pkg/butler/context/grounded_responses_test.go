package context

import (
	"strings"
	"testing"
)

func TestResolveCustomerScope(t *testing.T) {
	svc := testService(t)
	seed := CustomerMaster{CustomerID: "C-9", CustomerCode: "AC1", BusinessName: "Aurora Calibration"}
	if err := svc.db.Create(&seed).Error; err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	ids, name := svc.ResolveCustomerScope("Aurora Calibration")
	if len(ids) != 1 || ids[0] != "C-9" || name != "Aurora Calibration" {
		t.Fatalf("scope resolution failed: %v %q", ids, name)
	}

	if ids, _ := svc.ResolveCustomerScope("   "); ids != nil {
		t.Fatalf("blank reference must resolve to no scope, got %v", ids)
	}
}

func TestBuildCustomerNotesResponse(t *testing.T) {
	svc := testService(t)
	if err := svc.db.AutoMigrate(&EntityNote{}); err != nil {
		t.Fatalf("migrate entity notes: %v", err)
	}
	customer := CustomerMaster{CustomerID: "C-9", CustomerCode: "AC1", BusinessName: "Aurora Calibration"}
	if err := svc.db.Create(&customer).Error; err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	note := EntityNote{EntityType: "customer", EntityID: customer.ID, NoteType: "general", Content: "Prefers quarterly calibration visits"}
	if err := svc.db.Create(&note).Error; err != nil {
		t.Fatalf("seed note: %v", err)
	}

	response := svc.BuildCustomerNotesResponse([]string{"C-9"}, "Aurora Calibration")
	if !strings.Contains(response, "1 customer note(s)") || !strings.Contains(response, "Prefers quarterly calibration visits") {
		t.Fatalf("notes response missing content: %q", response)
	}

	empty := svc.BuildCustomerNotesResponse([]string{"missing"}, "Ghost Co")
	if !strings.Contains(empty, "could not resolve a note record scope") {
		t.Fatalf("unexpected empty-scope response: %q", empty)
	}
}
