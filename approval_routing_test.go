package main

import (
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/approvals"
	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/approval"
)

// =============================================================================
// Wave 3 B.1 — approval routing on the kernel, behavior-identical
// =============================================================================

func assessFixture(t *testing.T, margin, total float64, grade CustomerGrade, hasABB int) *CostingResult {
	t.Helper()
	app := &App{}
	result := &CostingResult{ActualMarginPct: margin, TotalFinalBHD: total, CustomerID: 7}
	customer := &Customer{HasABB: hasABB}
	app.assessCostingRisk(result, customer, grade)
	return result
}

// Pins the exact pre-engine output for the canonical scenarios: status
// strings, warning texts AND their order, and the recommended action.
func TestAssessCostingRisk_BehaviorIdentical(t *testing.T) {
	// Healthy: grade A, 25% margin, small order → auto-approved, no warnings.
	r := assessFixture(t, 0.25, 500, GradeA, 0)
	if r.ApprovalStatus != "AUTO_APPROVED" || r.NeedsApproval || len(r.RiskWarnings) != 0 ||
		r.RecommendedAction != "Proceed with quotation" {
		t.Fatalf("healthy costing: %+v", r)
	}

	// Grade D + critically low margin + competitor + large order: every rule fires,
	// warnings in the historical evaluation order, final recommendation is the
	// blanket manager-approval line (which always overrode the rule-specific ones).
	r = assessFixture(t, 0.05, 20000, GradeD, 1)
	want := []string{
		"Margin (5.0%) below 20% threshold",
		"HIGH RISK Grade D customer - 100% advance or DECLINE",
		"CRITICAL: Margin (5.0%) below minimum 8%",
		"ABB COMPETING - only proceed if strategically important",
		"⚠ Low margin + ABB competition - consider declining",
		"Large order (20000.00 BHD) - verify customer credit",
	}
	if r.ApprovalStatus != "NEEDS_APPROVAL" || !r.NeedsApproval {
		t.Fatalf("grade D costing not routed to approval: %+v", r)
	}
	if len(r.RiskWarnings) != len(want) {
		t.Fatalf("warnings = %#v, want %#v", r.RiskWarnings, want)
	}
	for i := range want {
		if r.RiskWarnings[i] != want[i] {
			t.Fatalf("warning[%d] = %q, want %q", i, r.RiskWarnings[i], want[i])
		}
	}
	if r.RecommendedAction != "Requires manager approval before proceeding" {
		t.Fatalf("recommended action = %q", r.RecommendedAction)
	}

	// Warning-only rules (grade C, good margin, large order) stay auto-approved.
	r = assessFixture(t, 0.30, 15000, GradeC, 0)
	if r.NeedsApproval || r.ApprovalStatus != "AUTO_APPROVED" || len(r.RiskWarnings) != 2 {
		t.Fatalf("grade C healthy costing: %+v", r)
	}
}

// A costing routed to NEEDS_APPROVAL is a kernel pending decision; an agent
// must be refused when it tries to approve it. (Costing flow, boundary test.)
func TestCostingApproval_AgentRefused(t *testing.T) {
	r := assessFixture(t, 0.05, 500, GradeD, 0)
	from, err := approvals.DecisionFromStatus(r.ApprovalStatus) // NEEDS_APPROVAL → pending
	if err != nil {
		t.Fatal(err)
	}
	if from != approval.DecisionPending {
		t.Fatalf("NEEDS_APPROVAL mapped to %s, want pending", from)
	}
	agent, err := actor.New(actor.Input{ID: "butler-01", DisplayName: "Butler", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approvals.Transition("costing-7", "costing", from, approval.DecisionApproved, agent, "looks fine", time.Now().UTC()); err == nil {
		t.Fatal("agent approved a pending costing")
	}
	mgr, err := actor.New(actor.Input{ID: "mgr-1", DisplayName: "Manager", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := approvals.Transition("costing-7", "costing", from, approval.DecisionApproved, mgr, "reviewed", time.Now().UTC()); err != nil {
		t.Fatalf("manager refused: %v", err)
	}
}

// Delete-request reviews run through the same kernel gate; agents are refused
// and decided requests cannot be re-decided. (Delete flow, boundary test.)
func TestDeleteApprovalGate_AgentRefusedAndTerminalStates(t *testing.T) {
	app := &App{}
	request := DeleteApprovalRequest{Status: "pending"}
	request.ID = "req-123"

	agent, err := actor.New(actor.Input{ID: "butler-01", DisplayName: "Butler", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.gateDeleteApprovalTransition(request, "approve", "", agent); err == nil {
		t.Fatal("agent passed the delete approval gate")
	} else if !strings.Contains(err.Error(), "AI-authority boundary") {
		t.Fatalf("refusal must name the boundary: %v", err)
	}
	if err := app.gateDeleteApprovalTransition(request, "reject", "", agent); err == nil {
		t.Fatal("agent rejected through the gate")
	}

	admin, err := actor.New(actor.Input{ID: "admin-1", DisplayName: "Admin", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.gateDeleteApprovalTransition(request, "approve", "ok", admin); err != nil {
		t.Fatalf("admin refused on a pending request: %v", err)
	}

	// Already-decided requests are illegal transitions even for admins.
	request.Status = "approved"
	if err := app.gateDeleteApprovalTransition(request, "approve", "", admin); err == nil {
		t.Fatal("re-approval of an approved request passed the gate")
	}
	request.Status = "garbage"
	if err := app.gateDeleteApprovalTransition(request, "approve", "", admin); err == nil {
		t.Fatal("unknown stored status must fail loudly, not route")
	}
}

// Payroll-run approvals run through the same kernel gate (W4 A.3): a draft
// run maps onto pending, agents are refused at the transition whatever
// authority they claim, and decided runs are terminal.
func TestPayrollApprovalGate_AgentRefusedAndTerminalStates(t *testing.T) {
	app := &App{}
	run := PayrollRun{Base: Base{ID: "run-1"}, Status: "draft"}

	agent, err := actor.New(actor.Input{ID: "butler-01", DisplayName: "Butler", Type: actor.TypeAgent, Authority: actor.AuthorityPropose})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.gatePayrollRunApproval(run, "looks fine", agent); err == nil {
		t.Fatal("agent approved a draft payroll run")
	}

	admin, err := actor.New(actor.Input{ID: "hr-1", DisplayName: "HR Manager", Type: actor.TypeOperator, Authority: actor.AuthorityApprove})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.gatePayrollRunApproval(run, "reviewed", admin); err != nil {
		t.Fatalf("authorized operator refused: %v", err)
	}

	// Decided runs are terminal for the gate.
	run.Status = "approved"
	if err := app.gatePayrollRunApproval(run, "", admin); err == nil {
		t.Fatal("re-approved an approved payroll run")
	}
	run.Status = "posted"
	if err := app.gatePayrollRunApproval(run, "", admin); err == nil {
		t.Fatal("gate accepted an unmapped status")
	}
}
