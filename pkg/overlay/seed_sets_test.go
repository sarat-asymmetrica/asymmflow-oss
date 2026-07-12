package overlay

import (
	"encoding/json"
	"testing"
)

// Pins the A.3 seed-selection contract: existing deployments (no seed_sets
// key in overlay.json) MUST keep running every seed bundle exactly as before
// the overlay gate existed; opting out requires an explicit empty array.
func TestSeedEnabled_Semantics(t *testing.T) {
	cases := []struct {
		name string
		sets []string // nil = field absent
		ask  string
		want bool
	}{
		{"absent field enables everything (back-compat)", nil, "demo-products", true},
		{"absent field enables rbac too", nil, "rbac-roles", true},
		{"explicit empty array disables all", []string{}, "demo-products", false},
		{"listed bundle runs", []string{"rbac-roles"}, "rbac-roles", true},
		{"unlisted bundle does not", []string{"rbac-roles"}, "demo-bank", false},
		{"all sentinel enables everything", []string{"all"}, "demo-customers", true},
		{"matching is case/space insensitive", []string{"  Demo-Products "}, "demo-products", true},
	}
	for _, tc := range cases {
		o := &CompanyOverlay{SeedSets: tc.sets}
		if got := o.SeedEnabled(tc.ask); got != tc.want {
			t.Errorf("%s: SeedEnabled(%q) with sets %v = %v, want %v", tc.name, tc.ask, tc.sets, got, tc.want)
		}
	}
}

// The built-in defaults and any overlay.json WITHOUT the key must decode to a
// nil slice (not an empty one) — that nil is what carries "run everything".
func TestSeedEnabled_JSONAbsenceDecodesToNil(t *testing.T) {
	var o CompanyOverlay
	if err := json.Unmarshal([]byte(`{"divisions":[{"key":"X"}]}`), &o); err != nil {
		t.Fatal(err)
	}
	if o.SeedSets != nil {
		t.Fatalf("absent seed_sets decoded to %#v, want nil", o.SeedSets)
	}
	if !o.SeedEnabled("anything") {
		t.Fatal("absent seed_sets must enable every bundle")
	}

	var o2 CompanyOverlay
	if err := json.Unmarshal([]byte(`{"divisions":[{"key":"X"}],"seed_sets":[]}`), &o2); err != nil {
		t.Fatal(err)
	}
	if o2.SeedSets == nil || o2.SeedEnabled("anything") {
		t.Fatal("explicit empty seed_sets must disable optional seeding")
	}
	if BuiltinDefaults().SeedSets != nil {
		t.Fatal("BuiltinDefaults must not restrict seed sets")
	}
}
