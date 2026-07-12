package customer

import (
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/crm"
)

var policyNow = time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)

func TestAssignCustomerIdentifiers_Bidirectional(t *testing.T) {
	// Code seeds ID.
	c := crm.CustomerMaster{BusinessName: "Nimbus Controls", CustomerCode: "CUST-NIMB1"}
	AssignCustomerIdentifiers(&c, policyNow)
	if c.CustomerID != "CUST-NIMB1" {
		t.Fatalf("code should seed blank ID, got %q", c.CustomerID)
	}

	// ID seeds code (the direction the pre-convergence root missed).
	c = crm.CustomerMaster{BusinessName: "Nimbus Controls", CustomerID: "CUST-NIMB2"}
	AssignCustomerIdentifiers(&c, policyNow)
	if c.CustomerCode != "CUST-NIMB2" {
		t.Fatalf("ID should seed blank code, got %q", c.CustomerCode)
	}

	// Both blank: generated default from the business name.
	c = crm.CustomerMaster{BusinessName: "Atlas Traders"}
	AssignCustomerIdentifiers(&c, policyNow)
	if !strings.HasPrefix(c.CustomerCode, "CUST-ATLA") {
		t.Fatalf("default code should carry ATLA prefix, got %q", c.CustomerCode)
	}
	if c.CustomerID != c.CustomerCode {
		t.Fatalf("ID should mirror the generated code, got %q vs %q", c.CustomerID, c.CustomerCode)
	}
}

func TestBusinessPrefix_CollectsLettersAcrossSeparators(t *testing.T) {
	// PH rule: skip non-letters until enough letters are found. The old root
	// truncated to 4 chars first, which made "A B Corp" yield "AB".
	if got := businessPrefix("A B Corp", 4, "CUST"); got != "ABCO" {
		t.Fatalf("expected ABCO, got %q", got)
	}
	if got := businessPrefix("42 GmbH", 4, "CUST"); got != "GMBH" {
		t.Fatalf("expected GMBH, got %q", got)
	}
	if got := businessPrefix("123", 4, "CUST"); got != "CUST" {
		t.Fatalf("expected fallback CUST, got %q", got)
	}
}

func TestPrepareSupplierCreate_DefaultsAndValidation(t *testing.T) {
	s := crm.SupplierMaster{SupplierName: "  Meridian Instruments GmbH  "}
	if err := PrepareSupplierCreate(&s, policyNow); err != nil {
		t.Fatal(err)
	}
	if s.SupplierName != "Meridian Instruments GmbH" {
		t.Fatalf("name should be trimmed, got %q", s.SupplierName)
	}
	if !strings.HasPrefix(s.SupplierCode, "SUP-MERI") {
		t.Fatalf("expected SUP-MERI prefix, got %q", s.SupplierCode)
	}
	if s.Rating != 3 {
		t.Fatalf("unrated supplier should default to 3, got %d", s.Rating)
	}

	if err := PrepareSupplierCreate(&crm.SupplierMaster{SupplierName: "   "}, policyNow); err == nil {
		t.Fatal("blank supplier name must be rejected")
	}
}

func TestMergeCustomerUpdate_PreservesServerOwnedFields(t *testing.T) {
	last := policyNow.AddDate(0, -1, 0)
	existing := crm.CustomerMaster{
		CustomerID:   "CUST-WASE1",
		CustomerCode: "CUST-WASE1",
		BusinessName: "Wasela Trading",
		Website:      "https://wasela.example",
		// Server-owned metrics a partial payload must never wipe:
		TotalOrdersValue: 125000.5,
		TotalOrdersCount: 42,
		AvgOrderValue:    2976.2,
		LastOrderDate:    &last,
		AvgPaymentDays:   38.5,
		DisputeCount:     2,
		ARRiskTier:       "Medium",
		OutstandingBHD:   6400.125,
		OverdueDays:      12,
		CustomerGrade:    "B",
	}
	existing.Version = 3

	// A partial payload: only city edited, everything else zero-valued.
	incoming := crm.CustomerMaster{City: "Manama", Website: ""}
	MergeCustomerUpdate(&existing, incoming, policyNow)

	if existing.TotalOrdersValue != 125000.5 || existing.TotalOrdersCount != 42 ||
		existing.OutstandingBHD != 6400.125 || existing.ARRiskTier != "Medium" ||
		existing.CustomerGrade != "B" || existing.LastOrderDate == nil {
		t.Fatalf("server-owned metrics were clobbered: %+v", existing)
	}
	if existing.BusinessName != "Wasela Trading" || existing.CustomerCode != "CUST-WASE1" || existing.CustomerID != "CUST-WASE1" {
		t.Fatalf("blank unique keys must fall back to existing: %+v", existing)
	}
	if existing.City != "Manama" {
		t.Fatalf("edited field must apply, got %q", existing.City)
	}
	if existing.Website != "" {
		t.Fatal("blanking an editable field is a legitimate edit and must apply")
	}
	if existing.Version != 4 {
		t.Fatalf("version should increment to 4, got %d", existing.Version)
	}
}

func TestMergeSupplierUpdate_RatingFallback(t *testing.T) {
	existing := crm.SupplierMaster{SupplierCode: "SUP-NORT1", SupplierName: "NORTHGRID", Rating: 5}

	MergeSupplierUpdate(&existing, crm.SupplierMaster{Rating: 0, Country: "Bahrain"}, policyNow)
	if existing.Rating != 5 {
		t.Fatalf("rating 0 means not-provided and must fall back, got %d", existing.Rating)
	}
	if existing.Country != "Bahrain" {
		t.Fatalf("edited field must apply, got %q", existing.Country)
	}

	MergeSupplierUpdate(&existing, crm.SupplierMaster{Rating: 2}, policyNow)
	if existing.Rating != 2 {
		t.Fatalf("explicit rating must apply, got %d", existing.Rating)
	}
}

func TestNormalizeBusinessIdentifier_DiscardsUUIDShapes(t *testing.T) {
	recordID := "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	if got := NormalizeBusinessIdentifier(recordID, recordID); got != "" {
		t.Fatalf("record's own UUID must be discarded, got %q", got)
	}
	if got := NormalizeBusinessIdentifier("A1B2C3D4-E5F6-7890-ABCD-EF0123456789", recordID); got != "" {
		t.Fatalf("any UUID-shaped identifier must be discarded, got %q", got)
	}
	if got := NormalizeBusinessIdentifier(" CUST-RIVE1 ", recordID); got != "CUST-RIVE1" {
		t.Fatalf("real business identifiers pass through trimmed, got %q", got)
	}
}

func TestRepairCustomerBusinessID(t *testing.T) {
	c := crm.CustomerMaster{BusinessName: "Riverside Utilities", CustomerCode: "CUST-RIVE1"}
	if got := RepairCustomerBusinessID(c); got != "CUST-RIVE1" {
		t.Fatalf("real code wins, got %q", got)
	}

	c = crm.CustomerMaster{BusinessName: "Riverside Utilities", CustomerCode: "3f2504e0-4f89-11d3-9a0c-0305e82c3301"}
	if got := RepairCustomerBusinessID(c); got != "CUST-RIVERS" {
		t.Fatalf("UUID-shaped code must be repaired deterministically, got %q", got)
	}
}
