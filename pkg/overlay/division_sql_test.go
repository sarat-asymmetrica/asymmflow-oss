package overlay

import "testing"

// TestDivisionNormalizationCase_BuiltinDefaults pins the generated SQL CASE for
// the built-in (synthetic) overlay. The generated IN-list must reproduce — in
// meaning, byte-for-byte — the division-backfill literals that used to be
// hardcoded across app.go (lowercased Key + declared aliases), so the migration
// backfills keep routing Beacon/Acme exactly as before. If the overlay's
// divisions or aliases change, this is the single place the generated SQL is
// asserted.
func TestDivisionNormalizationCase_BuiltinDefaults(t *testing.T) {
	got := BuiltinDefaults().DivisionNormalizationCase("orders.division")
	want := "CASE\n" +
		"\t\t\tWHEN LOWER(TRIM(COALESCE(orders.division, ''))) IN ('beacon controls', 'beacon controls wll', 'beacon controls w.l.l', 'beacon controls w.l.l.') THEN 'Beacon Controls'\n" +
		"\t\t\tELSE 'Acme Instrumentation'\n" +
		"\t\tEND"
	if got != want {
		t.Errorf("DivisionNormalizationCase mismatch:\n got: %q\nwant: %q", got, want)
	}
}

// TestDivisionNormalizationCase_EscapesQuotesAndAddsDivisions proves the
// generator is genuinely config-driven: a custom overlay with an apostrophe in a
// division name and an extra division produces a multi-WHEN CASE with SQL-safe
// escaping (no hardcoded Beacon/Acme assumptions leak in).
func TestDivisionNormalizationCase_EscapesQuotesAndAddsDivisions(t *testing.T) {
	o := &CompanyOverlay{
		DefaultDivisionKey: "Home Division",
		Divisions: []DivisionProfile{
			{Key: "Home Division"},
			{Key: "O'Brien Trading", Aliases: []string{"obrien"}},
			{Key: "North Unit"},
		},
	}
	got := o.DivisionNormalizationCase("t.division")
	want := "CASE\n" +
		"\t\t\tWHEN LOWER(TRIM(COALESCE(t.division, ''))) IN ('o''brien trading', 'obrien') THEN 'O''Brien Trading'\n" +
		"\t\t\tWHEN LOWER(TRIM(COALESCE(t.division, ''))) IN ('north unit') THEN 'North Unit'\n" +
		"\t\t\tELSE 'Home Division'\n" +
		"\t\tEND"
	if got != want {
		t.Errorf("DivisionNormalizationCase mismatch:\n got: %q\nwant: %q", got, want)
	}
}
