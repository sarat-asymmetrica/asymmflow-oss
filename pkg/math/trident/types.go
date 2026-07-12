package trident

// Regime represents which computational regime a query falls into.
// Based on three-regime dynamics: [30%, 20%, 50%].
type Regime int

const (
	RegimeExploration Regime = iota
	RegimeOptimization
	RegimeStabilization
)

// String returns the regime name.
func (r Regime) String() string {
	switch r {
	case RegimeExploration:
		return "Exploration"
	case RegimeOptimization:
		return "Optimization"
	case RegimeStabilization:
		return "Stabilization"
	default:
		return "Unknown"
	}
}

// TargetPercentage returns the ideal percentage for this regime.
func (r Regime) TargetPercentage() float64 {
	switch r {
	case RegimeExploration:
		return 0.30
	case RegimeOptimization:
		return 0.20
	case RegimeStabilization:
		return 0.50
	default:
		return 0.0
	}
}

// OptimizationResult holds metrics from applying Trident optimizations.
// This is decoupled from API request types and works with string prompts.
type OptimizationResult struct {
	OriginalPrompt        string
	OptimizedPrompt       string
	DetectedRegime        Regime
	RegimeDistribution    [3]float64
	Temperature           float64
	MaxTokensBudget       int
	DRSignature           int64
	WilliamsBatchSize     int
	TokenEstimate         int
	SkipAPICall           bool
	LocalAnswer           string
	Explanation           string
	RecommendedModel      string
	DRRegime              Regime
	ClassificationSource  string
	ShunyamContrast       float64
	ConvergencePredicted  bool
	ConvergenceConditions [4]bool
	PromptQuaternion      any
	NearestPrompts        []string
	OilRatio              float64
}

// TokenSavings calculates the percentage of tokens saved.
func (o *OptimizationResult) TokenSavings() float64 {
	if o.TokenEstimate == 0 {
		return 0.0
	}
	originalEstimate := len(o.OriginalPrompt) / 4
	if originalEstimate == 0 {
		return 0.0
	}
	saved := float64(originalEstimate-o.TokenEstimate) / float64(originalEstimate)
	if saved < 0 {
		return 0.0
	}
	return saved * 100.0
}

// BoundaryViolation represents a regime boundary breach.
type BoundaryViolation struct {
	Regime  Regime
	Current float64
	Minimum float64
	Deficit float64
}

// ConvergenceConditionName returns a human-readable name for each Pi Emergence condition.
func ConvergenceConditionName(idx int) string {
	names := [4]string{
		"Restoring Force (DR!=0)",
		"Proportional (3%<oil<30%)",
		"Single DOF (regime>50%)",
		"Energy Conservation (||q||=1)",
	}
	if idx >= 0 && idx < 4 {
		return names[idx]
	}
	return "Unknown"
}
