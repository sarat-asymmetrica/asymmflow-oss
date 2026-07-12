package trident

import (
	"math"
	"strconv"
	"strings"

	"ph_holdings_app/pkg/math/quaternion"
	"ph_holdings_app/pkg/math/vedic"
)

// Optimizer applies Trident mathematics to prompts.
type Optimizer struct {
	promptLibrary map[string]quaternion.Quaternion

	defaultMaxTokens int
	baseTokenBudget  int

	regimeDistribution [3]float64
	modelRouter        [3]string

	useDRFusion bool
	drCache     map[string]int64
}

// NewOptimizer creates an optimizer with the given base token budget.
func NewOptimizer(baseTokenBudget int) *Optimizer {
	if baseTokenBudget <= 0 {
		baseTokenBudget = 2048
	}

	return &Optimizer{
		promptLibrary:      make(map[string]quaternion.Quaternion),
		defaultMaxTokens:   512,
		baseTokenBudget:    baseTokenBudget,
		regimeDistribution: [3]float64{0.30, 0.20, 0.50},
		drCache:            make(map[string]int64),
		modelRouter: [3]string{
			"sarvam-m",
			"gemma-12b",
			"gemma-4b",
		},
	}
}

// RegisterPrompt adds a known-good prompt to the library for SLERP navigation.
func (o *Optimizer) RegisterPrompt(name, prompt string) {
	o.promptLibrary[name] = PromptToQuaternion(prompt)
}

// OptimizePrompt applies the full Trident optimization pipeline.
func (o *Optimizer) OptimizePrompt(prompt string) OptimizationResult {
	if prompt == "" {
		return OptimizationResult{
			OriginalPrompt:  "",
			OptimizedPrompt: "",
			SkipAPICall:     true,
			LocalAnswer:     "Error: No prompt found",
		}
	}

	result := OptimizationResult{
		OriginalPrompt:  prompt,
		OptimizedPrompt: prompt,
	}

	result.DRSignature = o.CachedDRSignature(prompt)
	promptQuat := PromptToQuaternion(prompt)
	result.PromptQuaternion = promptQuat

	if o.canAnswerLocally(prompt, result.DRSignature) {
		result.SkipAPICall = true
		result.LocalAnswer = o.answerLocally(prompt, result.DRSignature)
		return result
	}

	result.DRRegime = DrToRegime(result.DRSignature)
	result.DetectedRegime = ClassifyRegime(prompt)
	result.ClassificationSource = "keyword"

	if o.useDRFusion {
		if _, kwMatched := ClassifyRegimeKeywords(prompt); !kwMatched {
			result.DetectedRegime = result.DRRegime
			result.ClassificationSource = "digital_root"
		}
	}

	result.RegimeDistribution = o.computeRegimeWeights(prompt)
	result.RecommendedModel = o.ModelForRegime(result.DetectedRegime)

	switch result.DetectedRegime {
	case RegimeExploration:
		result.Temperature = 0.8
		result.MaxTokensBudget = int(float64(o.baseTokenBudget) * 0.30)
		if result.MaxTokensBudget < 256 {
			result.MaxTokensBudget = 256
		}
	case RegimeOptimization:
		result.Temperature = 0.1
		result.MaxTokensBudget = int(float64(o.baseTokenBudget) * 0.20)
		if result.MaxTokensBudget < 128 {
			result.MaxTokensBudget = 128
		}
	case RegimeStabilization:
		result.Temperature = 0.3
		result.MaxTokensBudget = int(float64(o.baseTokenBudget) * 0.50)
		if result.MaxTokensBudget < 256 {
			result.MaxTokensBudget = 256
		}
	}

	result.TokenEstimate = EstimateTokens(prompt)
	if result.TokenEstimate > 1000 {
		batchSize := vedic.WilliamsBatchSizeInt(result.TokenEstimate)
		result.WilliamsBatchSize = batchSize
		if batchSize < result.TokenEstimate {
			result.OptimizedPrompt = o.addChunkingHint(prompt, batchSize)
		}
	}

	if len(o.promptLibrary) >= 2 {
		result.NearestPrompts = o.findNearestPrompts(promptQuat, 2)
	}

	result.OilRatio = OilRatio(prompt)
	if result.OilRatio < 0.085 {
		result.OptimizedPrompt = o.suggestRefinement(prompt, result.OilRatio)
	}

	result.ShunyamContrast = ShunyamOilContrast(prompt)
	result.ConvergencePredicted, result.ConvergenceConditions = PredictConvergence(result)

	return result
}

// ModelForRegime returns the recommended model for a given regime.
func (o *Optimizer) ModelForRegime(r Regime) string {
	if int(r) >= 0 && int(r) < len(o.modelRouter) {
		return o.modelRouter[r]
	}
	return o.modelRouter[2]
}

// SetModelRouter overrides the default model routing.
func (o *Optimizer) SetModelRouter(models [3]string) {
	o.modelRouter = models
}

// EnableDRFusion enables Digital Root Regime Fusion.
func (o *Optimizer) EnableDRFusion() {
	o.useDRFusion = true
}

// CachedDRSignature returns the DR signature, using cache if available.
func (o *Optimizer) CachedDRSignature(text string) int64 {
	if dr, ok := o.drCache[text]; ok {
		return dr
	}
	dr := ComputeDRSignature(text)
	o.drCache[text] = dr
	return dr
}

// ComposeBatchDR computes the aggregate DR signature of a batch using cached DRs.
func (o *Optimizer) ComposeBatchDR(prompts []string) int64 {
	drs := make([]int64, len(prompts))
	for i, p := range prompts {
		drs[i] = o.CachedDRSignature(p)
	}
	return vedic.DigitalRootChain(drs)
}

// DRCacheStats returns the number of cached DR entries.
func (o *Optimizer) DRCacheStats() int {
	return len(o.drCache)
}

// ComputeRegimeDistribution computes the global regime distribution across prompts.
func (o *Optimizer) ComputeRegimeDistribution(prompts []string) [3]float64 {
	counts := [3]int{0, 0, 0}
	for _, p := range prompts {
		regime := ClassifyRegime(p)
		counts[regime]++
	}

	total := float64(len(prompts))
	if total < 0.01 {
		return [3]float64{0, 0, 0}
	}

	return [3]float64{
		float64(counts[0]) / total,
		float64(counts[1]) / total,
		float64(counts[2]) / total,
	}
}

// ValidateThreeRegimeTheorem checks if regime distribution matches [30%, 20%, 50%].
func (o *Optimizer) ValidateThreeRegimeTheorem(prompts []string, tolerance float64) bool {
	dist := o.ComputeRegimeDistribution(prompts)
	expected := [3]float64{0.30, 0.20, 0.50}
	for i := 0; i < 3; i++ {
		if math.Abs(dist[i]-expected[i]) > tolerance {
			return false
		}
	}
	return true
}

func (o *Optimizer) computeRegimeWeights(text string) [3]float64 {
	lower := strings.ToLower(text)
	weights := [3]float64{0.0, 0.0, 0.0}

	for _, w := range []string{"imagine", "create", "brainstorm", "what if", "story", "design"} {
		if strings.Contains(lower, w) {
			weights[0] += 0.15
		}
	}
	for _, w := range []string{"calculate", "compute", "optimize", "precise", "exact", "solve"} {
		if strings.Contains(lower, w) {
			weights[1] += 0.15
		}
	}
	for _, w := range []string{"explain", "what", "describe", "how", "why", "is"} {
		if strings.Contains(lower, w) {
			weights[2] += 0.10
		}
	}

	total := weights[0] + weights[1] + weights[2]
	if total < 0.01 {
		return o.regimeDistribution
	}

	for i := range weights {
		weights[i] /= total
	}
	return weights
}

func (o *Optimizer) findNearestPrompts(q quaternion.Quaternion, k int) []string {
	type distanceEntry struct {
		name     string
		distance float64
	}

	distances := make([]distanceEntry, 0, len(o.promptLibrary))
	for name, libQ := range o.promptLibrary {
		distances = append(distances, distanceEntry{name: name, distance: q.GeodesicDistance(libQ)})
	}

	for i := 0; i < k && i < len(distances); i++ {
		minIdx := i
		for j := i + 1; j < len(distances); j++ {
			if distances[j].distance < distances[minIdx].distance {
				minIdx = j
			}
		}
		distances[i], distances[minIdx] = distances[minIdx], distances[i]
	}

	result := make([]string, 0, k)
	for i := 0; i < k && i < len(distances); i++ {
		result = append(result, distances[i].name)
	}
	return result
}

func (o *Optimizer) addChunkingHint(prompt string, batchSize int) string {
	hint := "\n\n[Note: For optimal processing, consider breaking this into chunks of ~" +
		strconv.Itoa(batchSize) + " tokens based on Williams batching.]"
	return prompt + hint
}

func (o *Optimizer) suggestRefinement(prompt string, oilRatio float64) string {
	suggestion := "\n\n[Optimization suggestion: Information density is " +
		formatPercent(oilRatio) +
		"%. Consider adding more specific technical terms to improve answer quality.]"
	return prompt + suggestion
}

func (o *Optimizer) canAnswerLocally(prompt string, drSignature int64) bool {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "digital root") && strings.Contains(lower, "of") {
		return true
	}
	if drSignature == 9 && strings.Contains(lower, "divisible by 9") {
		return true
	}
	return false
}

func (o *Optimizer) answerLocally(prompt string, drSignature int64) string {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "digital root") {
		return "The digital root signature of your query is: " + strconv.FormatInt(drSignature, 10)
	}
	if strings.Contains(lower, "divisible by 9") {
		if drSignature == 9 {
			return "Based on digital root analysis (DR=9), the number MAY be divisible by 9. Further verification needed for certainty."
		}
		return "Based on digital root analysis (DR!=9), the number is definitely NOT divisible by 9."
	}
	return "Query can be answered locally but handler not implemented."
}
