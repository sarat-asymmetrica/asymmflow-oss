// Package settlement is the generic day-close / settlement engine: it
// reconciles expected takings against operator-counted amounts per tender
// method, computes variances exactly (kernel money, integer minor units),
// and gates the close on the kernel's authority model.
//
// The pattern (expected-vs-counted by tender type, variance, block-while-open,
// approval-gated) is a re-implementation of a settlement design proven in a
// reference hospitality system, done the AsymmFlow way: the engine is PURE —
// no database, no domain vocabulary. A vertical's domain service queries its
// own payment/refund/void rows, hands the engine TenderLines and Declarations,
// and persists the returned Record however it stores things.
//
// Layer model: pure kernel (money, actor) → THIS ENGINE → domain service →
// storage adapter.
package settlement

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"ph_holdings_app/pkg/kernel/actor"
	"ph_holdings_app/pkg/kernel/money"
)

// TenderLine is the system-expected total for one tender method over the
// settlement period (e.g. sum of captured cash payments).
type TenderLine struct {
	Method   string // e.g. "cash", "card", "transfer" — normalized case-insensitively
	Expected money.Amount
}

// Declaration is an operator-counted amount for one tender method.
// Methods without a declaration settle at their expected value (variance 0,
// Declared=false) — the usual treatment for card/transfer tenders where the
// processor's total IS the count.
type Declaration struct {
	Method  string
	Counted money.Amount
}

// TenderResult is the reconciliation outcome for one tender method.
type TenderResult struct {
	Method   string       `json:"method"`
	Expected money.Amount `json:"-"`
	Counted  money.Amount `json:"-"`
	Variance money.Amount `json:"-"` // counted − expected
	Declared bool         `json:"declared"`
}

// Summary is the full reconciliation for a settlement period.
type Summary struct {
	Tenders       []TenderResult
	TotalExpected money.Amount
	TotalCounted  money.Amount
	TotalVariance money.Amount // counted − expected across all tenders
	Currency      string
}

// HasVariance reports whether any tender's counted differs from expected.
func (s Summary) HasVariance() bool { return !s.TotalVariance.IsZero() }

// Compute reconciles expected tender totals against declarations.
// All amounts must share one currency; duplicate methods (case-insensitive)
// in either input are an error, as is a declaration for an unknown method —
// a counted amount with no expectation line means the caller's query missed
// a tender, which must fail loudly rather than settle silently.
func Compute(expected []TenderLine, declared []Declaration) (Summary, error) {
	if len(expected) == 0 {
		return Summary{}, errors.New("settlement: no expected tender lines")
	}

	currency := expected[0].Expected.Currency()
	scale := expected[0].Expected.Scale()
	zero := money.FromMinor(0, currency, scale)

	seen := make(map[string]int, len(expected)) // method → index in results
	results := make([]TenderResult, 0, len(expected))
	for _, line := range expected {
		method := normalizeMethod(line.Method)
		if method == "" {
			return Summary{}, errors.New("settlement: tender line with empty method")
		}
		if line.Expected.Currency() != currency {
			return Summary{}, fmt.Errorf("settlement: mixed currencies %s and %s", currency, line.Expected.Currency())
		}
		if _, dup := seen[method]; dup {
			return Summary{}, fmt.Errorf("settlement: duplicate tender line for method %q", method)
		}
		seen[method] = len(results)
		results = append(results, TenderResult{
			Method:   method,
			Expected: line.Expected,
			Counted:  line.Expected, // undeclared tenders settle at expected
			Variance: zero,
		})
	}

	declaredSeen := make(map[string]bool, len(declared))
	for _, d := range declared {
		method := normalizeMethod(d.Method)
		idx, ok := seen[method]
		if !ok {
			return Summary{}, fmt.Errorf("settlement: declaration for unknown tender method %q", method)
		}
		if declaredSeen[method] {
			return Summary{}, fmt.Errorf("settlement: duplicate declaration for method %q", method)
		}
		declaredSeen[method] = true
		if d.Counted.Currency() != currency {
			return Summary{}, fmt.Errorf("settlement: declaration currency %s does not match %s", d.Counted.Currency(), currency)
		}
		variance, err := d.Counted.Sub(results[idx].Expected)
		if err != nil {
			return Summary{}, fmt.Errorf("settlement: %w", err)
		}
		results[idx].Counted = d.Counted
		results[idx].Variance = variance
		results[idx].Declared = true
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Method < results[j].Method })

	summary := Summary{Tenders: results, TotalExpected: zero, TotalCounted: zero, TotalVariance: zero, Currency: currency}
	for _, r := range results {
		summary.TotalExpected, _ = summary.TotalExpected.Add(r.Expected)
		summary.TotalCounted, _ = summary.TotalCounted.Add(r.Counted)
		summary.TotalVariance, _ = summary.TotalVariance.Add(r.Variance)
	}
	return summary, nil
}

// OpenItems describes work that must be finished before a period can close.
// The vertical supplies its own counts (open bills, open sessions, undispatched
// tickets, unposted vouchers — whatever "open" means in that domain).
type OpenItems map[string]int

// Total returns the sum of all open-item counts.
func (o OpenItems) Total() int {
	n := 0
	for _, v := range o {
		n += v
	}
	return n
}

// Record is a completed, authority-stamped settlement.
type Record struct {
	BusinessDate time.Time
	Summary      Summary
	ClosedBy     actor.Actor
	ClosedAt     time.Time
	Note         string // operator explanation, required when variance ≠ 0
}

// Close gates and stamps a settlement:
//   - every open item must be zero (a day with open bills cannot close);
//   - the closing actor must hold approve authority — and because this check
//     is kernel actor.CanApprove, an AI agent can NEVER close a settlement
//     period, whatever authority it was granted (AI-authority boundary);
//   - a non-zero variance requires a non-empty operator note.
func Close(businessDate time.Time, summary Summary, open OpenItems, by actor.Actor, note string, now time.Time) (Record, error) {
	if open.Total() > 0 {
		parts := make([]string, 0, len(open))
		for kind, n := range open {
			if n > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", n, kind))
			}
		}
		sort.Strings(parts)
		return Record{}, fmt.Errorf("settlement: cannot close with open items: %s", strings.Join(parts, ", "))
	}
	if !by.CanApprove() {
		return Record{}, fmt.Errorf("settlement: actor %q (type %s) lacks authority to close a settlement period", by.ID, by.Type)
	}
	if summary.HasVariance() && strings.TrimSpace(note) == "" {
		return Record{}, fmt.Errorf("settlement: variance of %s requires an explanatory note", summary.TotalVariance.Format())
	}
	return Record{
		BusinessDate: businessDate,
		Summary:      summary,
		ClosedBy:     by,
		ClosedAt:     now,
		Note:         strings.TrimSpace(note),
	}, nil
}

func normalizeMethod(m string) string {
	return strings.ToLower(strings.TrimSpace(m))
}
