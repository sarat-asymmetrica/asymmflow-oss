package settlement

import (
	"strings"
	"testing"
	"time"

	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/money"
)

var businessDate = time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
var closeTime = time.Date(2026, 7, 3, 23, 30, 0, 0, time.UTC)

func manager(t *testing.T) actor.Actor {
	t.Helper()
	a, err := actor.New(actor.Input{
		ID: "staff-mgr-1", DisplayName: "Manager", Type: actor.TypeOperator, Authority: actor.AuthorityApprove,
	})
	if err != nil {
		t.Fatalf("actor.New: %v", err)
	}
	return a
}

func sar(v float64) money.Amount { return money.FromMinor(int64(v*100+0.5), "SAR", 2) }

func TestComputeCashVariance(t *testing.T) {
	// Realistic day: cash expected 1,250.00, counted 1,244.50 (short 5.50);
	// card 3,410.25 undeclared (processor total is authoritative).
	summary, err := Compute(
		[]TenderLine{
			{Method: "Cash", Expected: sar(1250.00)},
			{Method: "card", Expected: sar(3410.25)},
		},
		[]Declaration{{Method: "CASH", Counted: sar(1244.50)}},
	)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(summary.Tenders) != 2 {
		t.Fatalf("tenders = %d, want 2", len(summary.Tenders))
	}
	// Sorted alphabetically: card, cash.
	card, cash := summary.Tenders[0], summary.Tenders[1]
	if card.Method != "card" || cash.Method != "cash" {
		t.Fatalf("unexpected order: %s, %s", card.Method, cash.Method)
	}
	if card.Declared || !card.Variance.IsZero() {
		t.Errorf("undeclared card should settle at expected: %+v", card)
	}
	if !cash.Declared || cash.Variance.Minor() != -550 {
		t.Errorf("cash variance = %d minor, want -550", cash.Variance.Minor())
	}
	if summary.TotalVariance.Minor() != -550 {
		t.Errorf("total variance = %d, want -550", summary.TotalVariance.Minor())
	}
	if summary.TotalExpected.Minor() != 466025 {
		t.Errorf("total expected = %d, want 466025", summary.TotalExpected.Minor())
	}
	if !summary.HasVariance() {
		t.Error("HasVariance should be true")
	}
}

func TestComputeRejectsBadInput(t *testing.T) {
	if _, err := Compute(nil, nil); err == nil {
		t.Error("empty expected should error")
	}
	if _, err := Compute([]TenderLine{{Method: " ", Expected: sar(1)}}, nil); err == nil {
		t.Error("blank method should error")
	}
	if _, err := Compute(
		[]TenderLine{{Method: "cash", Expected: sar(1)}, {Method: "CASH", Expected: sar(2)}}, nil,
	); err == nil {
		t.Error("duplicate methods should error")
	}
	if _, err := Compute(
		[]TenderLine{{Method: "cash", Expected: sar(1)}},
		[]Declaration{{Method: "upi", Counted: sar(1)}},
	); err == nil {
		t.Error("declaration for unknown method should error")
	}
	if _, err := Compute(
		[]TenderLine{{Method: "cash", Expected: sar(1)}},
		[]Declaration{{Method: "cash", Counted: money.BHD(1)}},
	); err == nil {
		t.Error("mixed currencies should error")
	}
	if _, err := Compute(
		[]TenderLine{{Method: "cash", Expected: sar(1)}, {Method: "card", Expected: money.BHD(1)}}, nil,
	); err == nil {
		t.Error("mixed-currency expected lines should error")
	}
	if _, err := Compute(
		[]TenderLine{{Method: "cash", Expected: sar(1)}},
		[]Declaration{{Method: "cash", Counted: sar(1)}, {Method: "cash", Counted: sar(2)}},
	); err == nil {
		t.Error("duplicate declarations should error")
	}
}

func TestCloseHappyPath(t *testing.T) {
	summary, err := Compute([]TenderLine{{Method: "cash", Expected: sar(100)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := Close(businessDate, summary, OpenItems{"open bills": 0}, manager(t), "", closeTime)
	if err != nil {
		t.Fatalf("Close: %v", err)
	}
	if rec.ClosedBy.ID != "staff-mgr-1" || !rec.ClosedAt.Equal(closeTime) {
		t.Errorf("record not stamped: %+v", rec)
	}
}

func TestCloseBlockedByOpenItems(t *testing.T) {
	summary, _ := Compute([]TenderLine{{Method: "cash", Expected: sar(100)}}, nil)
	_, err := Close(businessDate, summary, OpenItems{"open bills": 2, "open sessions": 1}, manager(t), "", closeTime)
	if err == nil {
		t.Fatal("expected open-items error")
	}
	if !strings.Contains(err.Error(), "2 open bills") || !strings.Contains(err.Error(), "1 open sessions") {
		t.Errorf("error should enumerate open items: %v", err)
	}
}

func TestCloseVarianceRequiresNote(t *testing.T) {
	summary, err := Compute(
		[]TenderLine{{Method: "cash", Expected: sar(100)}},
		[]Declaration{{Method: "cash", Counted: sar(95)}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Close(businessDate, summary, nil, manager(t), "   ", closeTime); err == nil {
		t.Fatal("variance without note should error")
	}
	rec, err := Close(businessDate, summary, nil, manager(t), " till float miscount ", closeTime)
	if err != nil {
		t.Fatalf("Close with note: %v", err)
	}
	if rec.Note != "till float miscount" {
		t.Errorf("note = %q", rec.Note)
	}
}

func TestCloseEnforcesAIAuthorityBoundary(t *testing.T) {
	summary, _ := Compute([]TenderLine{{Method: "cash", Expected: sar(100)}}, nil)

	// actor.New itself refuses to grant an agent approve authority; construct
	// the struct directly to simulate a corrupted/hand-rolled actor and prove
	// Close still refuses it via CanApprove.
	agent := actor.Actor{ID: "butler-1", DisplayName: "Butler", Type: actor.TypeAgent, Authority: actor.AuthorityApprove}
	if _, err := Close(businessDate, summary, nil, agent, "", closeTime); err == nil {
		t.Fatal("agent actor must never close a settlement period")
	}

	// An operator without approve authority is also refused.
	waiter, err := actor.New(actor.Input{
		ID: "staff-w1", DisplayName: "Waiter", Type: actor.TypeOperator, Authority: actor.AuthorityPropose,
	})
	if err != nil {
		t.Fatalf("actor.New: %v", err)
	}
	if _, err := Close(businessDate, summary, nil, waiter, "", closeTime); err == nil {
		t.Fatal("propose-level operator must not close a settlement period")
	}
}
