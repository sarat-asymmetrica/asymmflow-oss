package trident

import "testing"

func TestNewOptimizer(t *testing.T) {
	o := NewOptimizer(4096)
	want := [3]float64{0.30, 0.20, 0.50}
	if o.regimeDistribution != want {
		t.Fatalf("regime distribution = %v, want %v", o.regimeDistribution, want)
	}
	if o.baseTokenBudget != 4096 {
		t.Fatalf("base token budget = %d, want 4096", o.baseTokenBudget)
	}
}

func TestOptimizePromptSkipsLocalAnswer(t *testing.T) {
	o := NewOptimizer(2048)
	result := o.OptimizePrompt("what is the digital root of 123")
	if !result.SkipAPICall {
		t.Fatalf("expected local answer skip")
	}
	if result.LocalAnswer == "" {
		t.Fatalf("expected local answer")
	}
}

func TestRegimeClassification(t *testing.T) {
	tests := []struct {
		prompt string
		want   Regime
	}{
		{"imagine a world", RegimeExploration},
		{"calculate 2+2", RegimeOptimization},
		{"what is Go", RegimeStabilization},
	}

	for _, tt := range tests {
		if got := ClassifyRegime(tt.prompt); got != tt.want {
			t.Fatalf("ClassifyRegime(%q) = %s, want %s", tt.prompt, got, tt.want)
		}
	}
}

func TestDRToRegime(t *testing.T) {
	tests := []struct {
		dr   int64
		want Regime
	}{
		{1, RegimeExploration},
		{2, RegimeOptimization},
		{3, RegimeStabilization},
		{5, RegimeOptimization},
		{9, RegimeStabilization},
	}

	for _, tt := range tests {
		if got := DrToRegime(tt.dr); got != tt.want {
			t.Fatalf("DrToRegime(%d) = %s, want %s", tt.dr, got, tt.want)
		}
	}
}

func TestShunyamContrast(t *testing.T) {
	got := ShunyamOilContrast("calculate precise turbine efficiency with 42 samples")
	if got <= 0 || got > 1 {
		t.Fatalf("ShunyamOilContrast() = %f, want in (0,1]", got)
	}
}

func TestDRFusion(t *testing.T) {
	prompt := promptWithDRRegime(t, RegimeExploration)
	o := NewOptimizer(2048)
	o.EnableDRFusion()

	result := o.OptimizePrompt(prompt)
	if result.ClassificationSource != "digital_root" {
		t.Fatalf("classification source = %q, want digital_root", result.ClassificationSource)
	}
	if result.DetectedRegime != RegimeExploration {
		t.Fatalf("detected regime = %s, want Exploration", result.DetectedRegime)
	}
}

func TestPromptToQuaternionIsUnit(t *testing.T) {
	q := PromptToQuaternion("any non-empty string")
	if !q.IsUnit(0.001) {
		t.Fatalf("prompt quaternion magnitude = %f, want unit", q.Magnitude())
	}
}

func promptWithDRRegime(t *testing.T, want Regime) string {
	t.Helper()
	for _, prompt := range []string{"plain alpha", "plain beta", "plain gamma", "plain delta"} {
		if _, matched := ClassifyRegimeKeywords(prompt); matched {
			continue
		}
		if DrToRegime(ComputeDRSignature(prompt)) == want {
			return prompt
		}
	}
	t.Fatalf("failed to find prompt for regime %s", want)
	return ""
}
