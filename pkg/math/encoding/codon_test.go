package encoding

import "testing"

func TestCodonRoundtrip(t *testing.T) {
	for i := 0; i <= 255; i++ {
		b := byte(i)
		if got := CodonDecode(CodonEncode(b)); got != b {
			t.Fatalf("roundtrip byte %d = %d", b, got)
		}
	}
}

func TestLosslessPromptRoundtrip(t *testing.T) {
	prompt := "Hello"
	if got := LosslessPromptDecode(LosslessPromptEncode(prompt)); got != prompt {
		t.Fatalf("roundtrip prompt = %q, want %q", got, prompt)
	}
}

func TestCodonGeodesicDistanceSelf(t *testing.T) {
	if got := CodonGeodesicDistance(42, 42); got != 0.0 {
		t.Fatalf("self distance = %f, want 0", got)
	}
}

func TestCodonGeodesicDistancePositive(t *testing.T) {
	if got := CodonGeodesicDistance(0, 255); got <= 0 {
		t.Fatalf("distance 0->255 = %f, want positive", got)
	}
}

func TestPromptCodonDistanceSameString(t *testing.T) {
	if got := PromptCodonDistance("abc", "abc"); got != 0.0 {
		t.Fatalf("same string distance = %f, want 0", got)
	}
}
