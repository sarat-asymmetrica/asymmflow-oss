// Package prism generates mathematical system prompts from Trident analysis.
//
// Om Lokah Samastah Sukhino Bhavantu
package prism

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/math/trident"
)

// NavaYoniQuality maps each digital root's planetary energy to a response quality.
var NavaYoniQuality = map[int64]string{
	1: "with clear authority and directness",
	2: "with intuitive warmth and empathy",
	3: "with expansive wisdom and teaching spirit",
	4: "with unconventional insight and fresh angles",
	5: "with precise communication and sharp analysis",
	6: "with aesthetic care and harmonious structure",
	7: "with reflective depth and subtle insight",
	8: "with disciplined structure and thoroughness",
	9: "with energetic clarity and decisive action",
}

// GeneratePrismPrompt creates a mathematical system prompt from harness analysis.
func GeneratePrismPrompt(result trident.OptimizationResult) string {
	sections := []string{RegimeTuning(result)}
	if quality, ok := NavaYoniQuality[result.DRSignature]; ok {
		sections = append(sections, fmt.Sprintf("Respond %s.", quality))
	}
	sections = append(sections, ConvergenceAdvisory(result))
	return strings.Join(sections, " ")
}

// RegimeTuning generates the regime-specific frequency instructions.
func RegimeTuning(result trident.OptimizationResult) string {
	switch result.DetectedRegime {
	case trident.RegimeExploration:
		return ExplorationTuning(result.ShunyamContrast)
	case trident.RegimeOptimization:
		return OptimizationTuning(result.ShunyamContrast)
	default:
		return StabilizationTuning(result.ShunyamContrast)
	}
}

// ExplorationTuning returns creative, divergent instructions.
func ExplorationTuning(contrast float64) string {
	if contrast > 0.7 {
		return "This is a creative query with rich specifics. Respond with imaginative depth, weaving the specific details into unexpected connections and vivid scenarios. Let your ideas develop fully before concluding."
	}
	if contrast < 0.3 {
		return "This is an open-ended creative prompt. Cast a wide net, offer multiple perspectives, paint vivid scenarios, and use analogies from different domains. The questioner wants to explore possibilities, not narrow to one answer."
	}
	return "This is a creative query with balanced context. Blend imagination with substance, grounding your creative ideas in real-world connections. Be both inventive and insightful."
}

// OptimizationTuning returns precise, analytical instructions.
func OptimizationTuning(contrast float64) string {
	if contrast > 0.7 {
		return "This is a precise analytical query packed with domain terms. Respond with maximum rigor, show your reasoning step by step, use specific values, and derive conclusions rather than asserting them. Every sentence should advance the argument."
	}
	if contrast < 0.3 {
		return "This query needs structured analysis but is broadly framed. Break it into clear sub-problems, define key terms precisely, and build toward a specific conclusion. Add the structure the question is missing."
	}
	return "This is an analytical query with moderate specificity. Be precise but explain your reasoning, showing both the conclusion and the path to it. Balance rigor with accessibility."
}

// StabilizationTuning returns clear, structured, reliable instructions.
func StabilizationTuning(contrast float64) string {
	if contrast > 0.7 {
		return "This is a specific factual query. Give a direct answer first, then provide concise supporting detail. Be authoritative and efficient, because the questioner knows what they're asking and wants a clear answer."
	}
	if contrast < 0.3 {
		return "This is a broad knowledge query. Structure your answer clearly, lead with the key point, then expand with examples and context. Make abstract concepts concrete. Aim for the clarity of a great teacher."
	}
	return "This is a knowledge query with moderate specificity. Give a well-structured answer that's both informative and approachable. Use an example if it helps illuminate the concept."
}

// ConvergenceAdvisory generates confidence guidance based on Pi emergence conditions.
func ConvergenceAdvisory(result trident.OptimizationResult) string {
	if result.ConvergencePredicted {
		return "The mathematical structure of this query is well-defined; your analysis will converge cleanly. Be confident in your conclusions."
	}

	conditionHints := [4]string{
		"the query may lack strong mathematical structure",
		"the information density is outside the typical range",
		"the query spans multiple domains without a clear focus",
		"",
	}

	hints := []string{}
	for i, c := range result.ConvergenceConditions {
		if !c && conditionHints[i] != "" {
			hints = append(hints, conditionHints[i])
		}
	}
	if len(hints) == 0 {
		return "Your analysis should converge well."
	}
	return fmt.Sprintf("Note: %s. Consider acknowledging this complexity in your response rather than forcing false precision.",
		strings.Join(hints, " and "))
}

// PrismStats holds metrics about prism-generated prompts for analysis.
type PrismStats struct {
	TotalGenerated  int
	AvgPromptLength float64
	RegimeCounts    [3]int
	ConvergenceRate float64
}

// DrRegimeNatural maps each DR to its natural regime.
var DrRegimeNatural = map[int64]trident.Regime{
	1: trident.RegimeExploration,
	4: trident.RegimeExploration,
	7: trident.RegimeExploration,
	2: trident.RegimeOptimization,
	5: trident.RegimeOptimization,
	8: trident.RegimeOptimization,
	3: trident.RegimeStabilization,
	6: trident.RegimeStabilization,
	9: trident.RegimeStabilization,
}

// NavaYoniSynergy maps each DR to its Vedic planetary friends.
var NavaYoniSynergy = map[int64][]int64{
	1: []int64{2, 3, 9},
	2: []int64{1, 5},
	3: []int64{1, 2, 9},
	4: []int64{6, 8},
	5: []int64{1, 6},
	6: []int64{5, 8},
	7: []int64{3, 9},
	8: []int64{5, 6},
	9: []int64{1, 3},
}

// SignalResonance describes the harmony state of mathematical signals.
type SignalResonance int

const (
	Resonant SignalResonance = iota
	Harmonic
	Dissonant
)

// String returns the resonance state name.
func (sr SignalResonance) String() string {
	switch sr {
	case Resonant:
		return "resonant"
	case Harmonic:
		return "harmonic"
	case Dissonant:
		return "dissonant"
	default:
		return "unknown"
	}
}

// DetectSignalResonance checks whether mathematical signals are in harmony.
func DetectSignalResonance(result trident.OptimizationResult) SignalResonance {
	agreements := 0
	total := 0

	if result.DRSignature >= 1 && result.DRSignature <= 9 {
		total++
		if natural, ok := DrRegimeNatural[result.DRSignature]; ok && natural == result.DetectedRegime {
			agreements++
		}
	}

	total++
	switch {
	case result.ShunyamContrast > 0.7 && result.DetectedRegime == trident.RegimeOptimization:
		agreements++
	case result.ShunyamContrast < 0.3 && result.DetectedRegime == trident.RegimeExploration:
		agreements++
	case result.ShunyamContrast >= 0.3 && result.ShunyamContrast <= 0.7 && result.DetectedRegime == trident.RegimeStabilization:
		agreements++
	}

	total++
	if result.ConvergencePredicted {
		agreements++
	}

	if agreements == total {
		return Resonant
	}
	if agreements == 0 {
		return Dissonant
	}
	return Harmonic
}

// ResonanceAdvisory generates advice based on signal harmony.
func ResonanceAdvisory(result trident.OptimizationResult) string {
	switch DetectSignalResonance(result) {
	case Resonant:
		return "All mathematical signals align; your response can flow with full confidence and natural authority."
	case Dissonant:
		return "The query sits at an intersection of categories. Honor this creative tension by blending approaches rather than forcing a single frame."
	default:
		return ""
	}
}

// HasNavaYoniSynergy returns true if two digital roots are Vedic planetary friends.
func HasNavaYoniSynergy(dr1, dr2 int64) bool {
	friends, ok := NavaYoniSynergy[dr1]
	if !ok {
		return false
	}
	for _, friend := range friends {
		if friend == dr2 {
			return true
		}
	}
	return false
}
