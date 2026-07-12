package main

import (
	"strings"

	"ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/pkg/compliance"
	"ph_holdings_app/pkg/compliance/bahrain"
	"ph_holdings_app/pkg/compliance/india"
	"ph_holdings_app/pkg/compliance/saudi"
	"ph_holdings_app/pkg/runtime/composition"
)

// initComplianceEventBus wires the in-process event bus + compliance
// subscriber through the shared composition seam (the same seam
// cmd/hospitality boots through), then installs the bus as the process
// default so domain publishers (e.g. Invoice.AfterCreate in pkg/finance)
// reach it. Called once during startup, after bulk bootstrap imports, so
// runtime writes — not boot-time backfills — drive validation.
func (a *App) initComplianceEventBus() {
	if a.composition == nil {
		a.composition = composition.NewRoot()
	}
	a.composition.Bus = a.eventBus // reuse if already created; seam creates one otherwise
	a.complianceHook = a.composition.WireCompliance(tradingTaxEngines()...)
	a.eventBus = a.composition.Bus
	a.composition.InstallDefaultBus()
}

// tradingTaxEngines is the trading vertical's compliance engine set. This is
// the ONE place the trading process decides which jurisdictions it can
// validate; the composition seam registers them.
func tradingTaxEngines() []compliance.TaxEngine {
	return []compliance.TaxEngine{bahrain.New(), india.NewGST(), saudi.New()}
}

// RecentComplianceValidations exposes the compliance subscriber's most recent
// validation outcomes (newest last), for the compliance dashboard / diagnostics.
func (a *App) RecentComplianceValidations(limit int) []compliance.ValidationEntry {
	if a == nil || a.complianceHook == nil {
		return nil
	}
	return a.complianceHook.RecentValidations(limit)
}

func (s *InfraService) GetComplianceDashboard(jurisdiction string) viewmodel.ComplianceDashboardVM {
	// Read the process registry wired by the composition seam (A.4: engines
	// are registered in exactly one place). Before the seam is wired — the
	// dashboard can be bound earlier in startup than initComplianceEventBus —
	// consult a throwaway registry with the same engine set.
	var registry *compliance.Registry
	if s.app != nil && s.app.composition != nil && s.app.composition.Registry != nil {
		registry = s.app.composition.Registry
	} else {
		registry = compliance.NewRegistry()
		for _, e := range tradingTaxEngines() {
			registry.Register(e)
		}
	}

	active := compliance.Jurisdiction(strings.ToUpper(strings.TrimSpace(jurisdiction)))
	if active == "" {
		active = compliance.JurisdictionBahrain
	}

	engine, ok := registry.Get(active)
	if !ok {
		return viewmodel.ComplianceDashboardVM{
			Jurisdiction:      string(active),
			ComplianceScore:   0,
			TaxRates:          nil,
			RecentValidations: nil,
			Issues: []viewmodel.ComplianceIssue{
				{Severity: "warning", Message: "No compliance engine registered for jurisdiction"},
			},
		}
	}

	return viewmodel.ComplianceDashboardVM{
		Jurisdiction:      string(engine.Jurisdiction()),
		TaxRates:          engine.TaxRates(),
		RecentValidations: []viewmodel.ValidationEntry{},
		ComplianceScore:   100,
		Issues:            []viewmodel.ComplianceIssue{},
	}
}
