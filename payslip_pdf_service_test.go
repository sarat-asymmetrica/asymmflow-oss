package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Wave 13 / P6: the payslip PDF renders from stored payroll records only.
// Reuses the payroll golden-test harness (setupPayrollApp / seed helpers /
// GeneratePayrollRun) so the payslip is generated from a real run item with
// real component rows — synthetic identities only.
func TestGeneratePayslipPDF_FromGoldenRun(t *testing.T) {
	app := setupPayrollApp(t)
	period := seedPayrollPeriod(t, app)

	seedPayrollEmployee(t, app, "emp-slip-1", "Aisha Rahman", EmployeeCompensationProfile{
		BaseSalary:         1000,
		HousingAllowance:   128,
		TransportAllowance: 64,
		StandardDeduction:  32,
		TaxDeduction:       16,
		EmployerCost:       96,
	})

	_, err := app.GeneratePayrollRun(period.ID)
	require.NoError(t, err)

	path, err := app.GeneratePayslipPDF("emp-slip-1", period.ID)
	require.NoError(t, err)
	require.NotEmpty(t, path)
	t.Cleanup(func() { _ = os.Remove(path) })

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(1000), "payslip PDF should not be a trivial/empty file")

	header := make([]byte, 5)
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.Read(header)
	require.NoError(t, err)
	require.Equal(t, "%PDF-", string(header))
}

// The payslip must refuse to render for an employee with no run item in the
// period — never an empty payslip.
func TestGeneratePayslipPDF_NoRunItem_Errors(t *testing.T) {
	app := setupPayrollApp(t)
	period := seedPayrollPeriod(t, app)

	_, err := app.GeneratePayslipPDF("emp-never-ran", period.ID)
	require.Error(t, err)
}

// Blank IDs are rejected before any DB work.
func TestGeneratePayslipPDF_BlankInputs_Error(t *testing.T) {
	app := setupPayrollApp(t)

	_, err := app.GeneratePayslipPDF("", "some-period")
	require.Error(t, err)
	_, err = app.GeneratePayslipPDF("emp-1", "  ")
	require.Error(t, err)
}
