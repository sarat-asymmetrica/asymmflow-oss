package india

import "testing"

func TestStateNameKnownCodes(t *testing.T) {
	cases := map[string]string{
		"01": "Jammu and Kashmir",
		"27": "Maharashtra",
		"29": "Karnataka",
		"36": "Telangana",
		"38": "Ladakh",
	}
	for code, want := range cases {
		got, ok := StateName(code)
		if !ok {
			t.Errorf("StateName(%q): not found", code)
			continue
		}
		if got != want {
			t.Errorf("StateName(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestStateNameUnknownCode(t *testing.T) {
	for _, code := range []string{"00", "25", "99", "", "AB"} {
		if _, ok := StateName(code); ok {
			t.Errorf("StateName(%q) should be unknown", code)
		}
	}
}

func TestValidStateCode(t *testing.T) {
	if !ValidStateCode("29") {
		t.Error("29 (Karnataka) should be a valid state code")
	}
	if ValidStateCode("25") {
		t.Error("25 (old standalone Daman and Diu, merged into 26 in 2020) should not be valid")
	}
	if ValidStateCode("") {
		t.Error("empty code should not be valid")
	}
}

func TestAllStatesSortedAndComplete(t *testing.T) {
	states := AllStates()
	if len(states) < 30 {
		t.Fatalf("expected at least 30 states/UTs, got %d", len(states))
	}
	for i := 1; i < len(states); i++ {
		if states[i-1].Code >= states[i].Code {
			t.Fatalf("AllStates() not sorted by code at index %d: %q >= %q", i, states[i-1].Code, states[i].Code)
		}
	}
}
