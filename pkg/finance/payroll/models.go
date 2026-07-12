// Package payroll owns the payroll domain: compensation profiles, payroll
// periods, run generation (the gross/net arithmetic), the kernel-gated
// approval, and the accrual/payout posting journals.
//
// Wave 5 A.2: a W4-D1 peel from the root payroll_service.go, executed
// AFTER the golden tests in payroll_golden_test.go were committed green
// against the untouched code (invariant 5 — the journal numbers are
// sacred). The finance models (JournalEntry, ChartOfAccount, ExpenseEntry)
// already live in pkg/finance, so posting moved inward; what stays behind
// ports: employee identity/lookup (the collaboration hub), UI event
// emission (Wails runtime), and the payroll→expense-ledger bridge (the
// expense service is host-side).
package payroll

import (
	"time"

	shareddomain "ph_holdings_app/pkg/domain"
)

type Base = shareddomain.Base

type CompensationProfile struct {
	Base
	EmployeeID         string     `gorm:"uniqueIndex;size:36" json:"employee_id"`
	Division           string     `gorm:"size:100" json:"division"`
	PayFrequency       string     `gorm:"size:20;default:'monthly'" json:"pay_frequency"`
	Currency           string     `gorm:"size:3;default:'BHD'" json:"currency"`
	BaseSalary         float64    `gorm:"type:decimal(15,3)" json:"base_salary"`
	HousingAllowance   float64    `gorm:"type:decimal(15,3)" json:"housing_allowance"`
	TransportAllowance float64    `gorm:"type:decimal(15,3)" json:"transport_allowance"`
	OtherAllowance     float64    `gorm:"type:decimal(15,3)" json:"other_allowance"`
	StandardDeduction  float64    `gorm:"type:decimal(15,3)" json:"standard_deduction"`
	TaxDeduction       float64    `gorm:"type:decimal(15,3)" json:"tax_deduction"`
	EmployerCost       float64    `gorm:"type:decimal(15,3)" json:"employer_cost"`
	EffectiveFrom      *time.Time `json:"effective_from"`
	EffectiveTo        *time.Time `json:"effective_to"`
	IsActive           bool       `gorm:"default:true;index" json:"is_active"`
	Notes              string     `gorm:"type:text" json:"notes"`

	EmployeeName string `gorm:"-" json:"employee_name,omitempty"`
	JobTitle     string `gorm:"-" json:"job_title,omitempty"`
}

func (CompensationProfile) TableName() string { return "employee_compensation_profiles" }

type Period struct {
	Base
	Name        string     `gorm:"uniqueIndex;size:120" json:"name"`
	Division    string     `gorm:"size:100" json:"division"`
	PeriodStart time.Time  `gorm:"index" json:"period_start"`
	PeriodEnd   time.Time  `gorm:"index" json:"period_end"`
	PaymentDate *time.Time `gorm:"index" json:"payment_date"`
	Status      string     `gorm:"size:20;default:'open';index" json:"status"`
	Notes       string     `gorm:"type:text" json:"notes"`
}

func (Period) TableName() string { return "payroll_periods" }

type Run struct {
	Base
	RunNumber            string     `gorm:"uniqueIndex;size:50" json:"run_number"`
	PayrollPeriodID      string     `gorm:"index;size:36" json:"payroll_period_id"`
	Division             string     `gorm:"size:100" json:"division"`
	Status               string     `gorm:"size:20;default:'draft';index" json:"status"`
	GeneratedAt          *time.Time `json:"generated_at"`
	ApprovedAt           *time.Time `json:"approved_at"`
	ApprovedBy           string     `gorm:"size:36" json:"approved_by"`
	PostedAt             *time.Time `json:"posted_at"`
	PostedBy             string     `gorm:"size:36" json:"posted_by"`
	PaidAt               *time.Time `json:"paid_at"`
	PaymentReference     string     `gorm:"size:120" json:"payment_reference"`
	BankAccountID        *string    `gorm:"size:36" json:"bank_account_id"`
	JournalEntryID       *string    `gorm:"size:36" json:"journal_entry_id"`
	PayoutJournalEntryID *string    `gorm:"size:36" json:"payout_journal_entry_id"`
	TotalEmployees       int        `json:"total_employees"`
	GrossTotal           float64    `gorm:"type:decimal(15,3)" json:"gross_total"`
	DeductionsTotal      float64    `gorm:"type:decimal(15,3)" json:"deductions_total"`
	NetTotal             float64    `gorm:"type:decimal(15,3)" json:"net_total"`
	EmployerCostTotal    float64    `gorm:"type:decimal(15,3)" json:"employer_cost_total"`
	Currency             string     `gorm:"size:3;default:'BHD'" json:"currency"`
	Notes                string     `gorm:"type:text" json:"notes"`

	PeriodName string    `gorm:"-" json:"period_name,omitempty"`
	Items      []RunItem `gorm:"-" json:"items,omitempty"`
	Payouts    []Payout  `gorm:"-" json:"payouts,omitempty"`
}

func (Run) TableName() string { return "payroll_runs" }

type RunItem struct {
	Base
	PayrollRunID          string  `gorm:"index;size:36" json:"payroll_run_id"`
	EmployeeID            string  `gorm:"index;size:36" json:"employee_id"`
	CompensationProfileID *string `gorm:"size:36" json:"compensation_profile_id"`
	EmployeeNameSnapshot  string  `gorm:"size:255" json:"employee_name_snapshot"`
	JobTitleSnapshot      string  `gorm:"size:120" json:"job_title_snapshot"`
	BaseSalary            float64 `gorm:"type:decimal(15,3)" json:"base_salary"`
	AllowancesTotal       float64 `gorm:"type:decimal(15,3)" json:"allowances_total"`
	DeductionsTotal       float64 `gorm:"type:decimal(15,3)" json:"deductions_total"`
	EmployerCostTotal     float64 `gorm:"type:decimal(15,3)" json:"employer_cost_total"`
	GrossPay              float64 `gorm:"type:decimal(15,3)" json:"gross_pay"`
	NetPay                float64 `gorm:"type:decimal(15,3)" json:"net_pay"`
	Status                string  `gorm:"size:20;default:'draft';index" json:"status"`
	Notes                 string  `gorm:"type:text" json:"notes"`

	EmployeeName string      `gorm:"-" json:"employee_name,omitempty"`
	PayoutID     string      `gorm:"-" json:"payout_id,omitempty"`
	PayoutStatus string      `gorm:"-" json:"payout_status,omitempty"`
	PayoutPaidAt *time.Time  `gorm:"-" json:"payout_paid_at,omitempty"`
	Components   []Component `gorm:"-" json:"components,omitempty"`
}

func (RunItem) TableName() string { return "payroll_run_items" }

type Component struct {
	Base
	PayrollRunItemID string  `gorm:"index;size:36" json:"payroll_run_item_id"`
	ComponentType    string  `gorm:"size:20;index" json:"component_type"`
	Code             string  `gorm:"size:40" json:"code"`
	Name             string  `gorm:"size:120" json:"name"`
	Amount           float64 `gorm:"type:decimal(15,3)" json:"amount"`
}

func (Component) TableName() string { return "payroll_components" }

type Payout struct {
	Base
	PayrollRunID        string     `gorm:"index;size:36" json:"payroll_run_id"`
	PayrollRunItemID    string     `gorm:"index;size:36" json:"payroll_run_item_id"`
	EmployeeID          string     `gorm:"index;size:36" json:"employee_id"`
	Division            string     `gorm:"size:100" json:"division"`
	ScheduledAt         *time.Time `gorm:"index" json:"scheduled_at"`
	PaidAt              *time.Time `gorm:"index" json:"paid_at"`
	Amount              float64    `gorm:"type:decimal(15,3)" json:"amount"`
	Currency            string     `gorm:"size:3;default:'BHD'" json:"currency"`
	Status              string     `gorm:"size:20;default:'scheduled';index" json:"status"`
	PaymentReference    string     `gorm:"size:120" json:"payment_reference"`
	BankAccountID       *string    `gorm:"size:36" json:"bank_account_id"`
	BankStatementLineID *string    `gorm:"size:36" json:"bank_statement_line_id"`

	EmployeeName string `gorm:"-" json:"employee_name,omitempty"`
	RunNumber    string `gorm:"-" json:"run_number,omitempty"`
}

func (Payout) TableName() string { return "payroll_payouts" }

type DashboardSummary struct {
	ActiveProfiles           int     `json:"active_profiles"`
	OpenPeriods              int     `json:"open_periods"`
	DraftRuns                int     `json:"draft_runs"`
	ApprovedUnpaidRuns       int     `json:"approved_unpaid_runs"`
	MonthToDateNetPayroll    float64 `json:"month_to_date_net_payroll"`
	UpcomingPayrollLiability float64 `json:"upcoming_payroll_liability"`
}
