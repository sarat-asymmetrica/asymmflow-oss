package viewmodel

import "ph_holdings_app/pkg/compliance"

// ComplianceDashboardVM shows compliance status for the active jurisdiction.
type ComplianceDashboardVM struct {
	Jurisdiction      string               `json:"jurisdiction"`
	TaxRates          []compliance.TaxRate `json:"tax_rates"`
	RecentValidations []ValidationEntry    `json:"recent_validations"`
	ComplianceScore   float64              `json:"compliance_score"`
	Issues            []ComplianceIssue    `json:"issues"`
}

type ValidationEntry struct {
	Timestamp    string   `json:"timestamp"`
	EventName    string   `json:"event_name"`
	Jurisdiction string   `json:"jurisdiction"`
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
}

type ComplianceIssue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// TaxCalculatorVM is the seed viewmodel for a future tax calculation screen.
type TaxCalculatorVM struct {
	Jurisdictions      []string `json:"jurisdictions"`
	ActiveJurisdiction string   `json:"active_jurisdiction"`
}
