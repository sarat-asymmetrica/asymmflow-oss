package prism

import (
	"strings"
	"testing"

	"ph_holdings_app/pkg/math/trident"
)

func TestGeneratePrismPromptNotEmpty(t *testing.T) {
	for _, regime := range []trident.Regime{
		trident.RegimeExploration,
		trident.RegimeOptimization,
		trident.RegimeStabilization,
	} {
		result := trident.OptimizationResult{
			DetectedRegime:        regime,
			DRSignature:           5,
			ShunyamContrast:       0.5,
			ConvergenceConditions: [4]bool{true, true, true, true},
			ConvergencePredicted:  true,
		}
		if got := GeneratePrismPrompt(result); got == "" {
			t.Fatalf("GeneratePrismPrompt(%s) returned empty", regime)
		}
	}
}

func TestGeneratePersonaContainsArchetype(t *testing.T) {
	result := trident.OptimizationResult{
		DetectedRegime:  trident.RegimeExploration,
		DRSignature:     1,
		ShunyamContrast: 0.8,
	}
	got := GeneratePersona(result)
	if !strings.Contains(got, "You are") {
		t.Fatalf("persona = %q, want to contain You are", got)
	}
}

func TestDetectSignalResonance(t *testing.T) {
	result := trident.OptimizationResult{
		DetectedRegime:       trident.RegimeExploration,
		DRSignature:          1,
		ShunyamContrast:      0.2,
		ConvergencePredicted: true,
	}
	got := DetectSignalResonance(result)
	if got != Resonant && got != Harmonic && got != Dissonant {
		t.Fatalf("invalid resonance value: %v", got)
	}
}

func TestNavaYoniSynergySymmetry(t *testing.T) {
	if !HasNavaYoniSynergy(1, 2) {
		t.Fatalf("expected Sun/Moon synergy")
	}
	if !HasNavaYoniSynergy(2, 1) {
		t.Fatalf("expected Moon/Sun reciprocal synergy")
	}
}
