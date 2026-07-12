package payroll

import "testing"

// B6: CompensationProfile is uniqueIndex'd on employee_id alone, and Division
// is a plain field. UpsertProfile looks up the existing row by employee_id
// only, so a naive update silently moves an employee's single profile row
// from one division to another (overwriting Division and every money field)
// whenever the caller submits a different division for the same employee.
// These tests pin the refusal guard that stops that clobber while still
// allowing same-division edits and first-time division assignment on
// legacy rows.

func TestUpsertProfile_RefusesCrossDivisionClobber(t *testing.T) {
	svc, _ := testService(t, map[string]EmployeeRef{
		"emp-1": {ID: "emp-1", FullName: "Cross Division", IsActive: true},
	})

	if _, err := svc.UpsertProfile(CompensationProfile{
		EmployeeID: "emp-1",
		Division:   "Acme Instrumentation",
		BaseSalary: 1000,
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	_, err := svc.UpsertProfile(CompensationProfile{
		EmployeeID: "emp-1",
		Division:   "Beacon Controls",
		BaseSalary: 9999,
	})
	if err == nil {
		t.Fatal("cross-division save must be refused")
	}

	var stored CompensationProfile
	if e := svc.db.Where("employee_id = ?", "emp-1").First(&stored).Error; e != nil {
		t.Fatalf("re-query: %v", e)
	}
	if stored.Division != "Acme Instrumentation" {
		t.Fatalf("division must be untouched by the refused save, got %q", stored.Division)
	}
	if stored.BaseSalary != 1000 {
		t.Fatalf("base salary must be untouched by the refused save, got %v", stored.BaseSalary)
	}
}

func TestUpsertProfile_SameDivisionEditSucceeds(t *testing.T) {
	svc, _ := testService(t, map[string]EmployeeRef{
		"emp-1": {ID: "emp-1", FullName: "Same Division", IsActive: true},
	})

	if _, err := svc.UpsertProfile(CompensationProfile{
		EmployeeID: "emp-1",
		Division:   "Acme Instrumentation",
		BaseSalary: 1000,
	}); err != nil {
		t.Fatalf("seed profile: %v", err)
	}

	if _, err := svc.UpsertProfile(CompensationProfile{
		EmployeeID: "emp-1",
		Division:   "Acme Instrumentation",
		BaseSalary: 1200,
	}); err != nil {
		t.Fatalf("same-division edit must succeed: %v", err)
	}

	var stored CompensationProfile
	if e := svc.db.Where("employee_id = ?", "emp-1").First(&stored).Error; e != nil {
		t.Fatalf("re-query: %v", e)
	}
	if stored.BaseSalary != 1200 {
		t.Fatalf("expected updated base salary 1200, got %v", stored.BaseSalary)
	}
	if stored.Division != "Acme Instrumentation" {
		t.Fatalf("division must remain Acme Instrumentation, got %q", stored.Division)
	}
}

func TestUpsertProfile_LegacyEmptyDivisionAllowsFirstSet(t *testing.T) {
	svc, _ := testService(t, map[string]EmployeeRef{
		"emp-1": {ID: "emp-1", FullName: "Legacy Row", IsActive: true},
	})

	// Simulate a pre-normalization legacy row inserted with an empty
	// Division, bypassing UpsertProfile (and its normalizeDivision call)
	// entirely.
	legacy := CompensationProfile{
		EmployeeID: "emp-1",
		Division:   "",
		BaseSalary: 800,
	}
	if e := svc.db.Create(&legacy).Error; e != nil {
		t.Fatalf("seed legacy row: %v", e)
	}

	if _, err := svc.UpsertProfile(CompensationProfile{
		EmployeeID: "emp-1",
		Division:   "Acme Instrumentation",
		BaseSalary: 800,
	}); err != nil {
		t.Fatalf("first division set on legacy row must succeed: %v", err)
	}

	var stored CompensationProfile
	if e := svc.db.Where("employee_id = ?", "emp-1").First(&stored).Error; e != nil {
		t.Fatalf("re-query: %v", e)
	}
	if stored.Division != "Acme Instrumentation" {
		t.Fatalf("expected division to be set to Acme Instrumentation, got %q", stored.Division)
	}
}
