package main

// G2 payroll hot-zone: UpsertEmployeeCompensationProfile is a financial + PII
// mutation. This pins the App-binding contract the kernel Payroll compensation
// form drives (bridge realUpsertProfile): the profile persists with the sent
// amounts, currency defaults to BHD, an upsert updates in place (never
// duplicates), and an unknown employee is refused. Synthetic canon only — no
// real names or salaries.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUpsertEmployeeCompensationProfile_PersistsAndGuards(t *testing.T) {
	app := setupPayrollApp(t)

	// The employee must exist in the directory (synthetic canon).
	require.NoError(t, app.db.Create(&Employee{
		Base:             Base{ID: "emp-comp-1"},
		EmployeeCode:     "EMP-emp-comp-1",
		FullName:         "Layla Hassan",
		JobTitle:         "Instrumentation Engineer",
		EmploymentStatus: "active",
		IsActive:         true,
	}).Error)

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	saved, err := app.UpsertEmployeeCompensationProfile(EmployeeCompensationProfile{
		EmployeeID:         "emp-comp-1",
		Division:           "Acme Instrumentation",
		PayFrequency:       "monthly",
		BaseSalary:         900,
		HousingAllowance:   120,
		TransportAllowance: 60,
		OtherAllowance:     30,
		StandardDeduction:  25,
		TaxDeduction:       10,
		EmployerCost:       80,
		EffectiveFrom:      &from,
		IsActive:           true,
		Notes:              "Synthetic canon comp profile",
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)

	// Persisted row reflects the sent amounts; currency defaulted to BHD.
	var row EmployeeCompensationProfile
	require.NoError(t, app.db.Where("employee_id = ?", "emp-comp-1").First(&row).Error)
	require.Equal(t, 900.0, row.BaseSalary)
	require.Equal(t, 120.0, row.HousingAllowance)
	require.Equal(t, 60.0, row.TransportAllowance)
	require.Equal(t, 80.0, row.EmployerCost)
	require.Equal(t, "BHD", row.Currency)
	require.Equal(t, "monthly", row.PayFrequency)
	require.True(t, row.IsActive)
	require.NotNil(t, row.EffectiveFrom)

	// Upsert on the same employee updates IN PLACE — never a second row.
	saved2, err := app.UpsertEmployeeCompensationProfile(EmployeeCompensationProfile{
		Base:         Base{ID: saved.ID},
		EmployeeID:   "emp-comp-1",
		Division:     "Acme Instrumentation",
		PayFrequency: "monthly",
		BaseSalary:   1100,
		IsActive:     true,
	})
	require.NoError(t, err)
	require.Equal(t, saved.ID, saved2.ID)

	var count int64
	require.NoError(t, app.db.Model(&EmployeeCompensationProfile{}).Where("employee_id = ?", "emp-comp-1").Count(&count).Error)
	require.Equal(t, int64(1), count, "upsert updates in place, never duplicates")

	var updated EmployeeCompensationProfile
	require.NoError(t, app.db.Where("employee_id = ?", "emp-comp-1").First(&updated).Error)
	require.Equal(t, 1100.0, updated.BaseSalary)

	// Unknown employee is refused — no orphan compensation rows.
	_, err = app.UpsertEmployeeCompensationProfile(EmployeeCompensationProfile{
		EmployeeID: "ghost-employee",
		BaseSalary: 5000,
	})
	require.Error(t, err)
}
