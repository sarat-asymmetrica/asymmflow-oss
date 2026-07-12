package trident

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"ph_holdings_app/pkg/math/quaternion"
	"ph_holdings_app/pkg/math/vedic"
)

// PromptToQuaternion encodes a text prompt as a unit quaternion on S3.
func PromptToQuaternion(prompt string) quaternion.Quaternion {
	if len(prompt) == 0 {
		return quaternion.Identity()
	}

	bytes := []byte(prompt)
	sum := 0.0
	for _, b := range bytes {
		sum += float64(b)
	}
	mean := sum / float64(len(bytes))

	variance := 0.0
	for _, b := range bytes {
		diff := float64(b) - mean
		variance += diff * diff
	}
	variance /= float64(len(bytes))

	punctCount := 0
	for _, r := range prompt {
		if unicode.IsPunct(r) {
			punctCount++
		}
	}
	punctDensity := float64(punctCount) / float64(utf8.RuneCountInString(prompt))

	words := strings.Fields(strings.ToLower(prompt))
	uniqueWords := make(map[string]bool)
	for _, w := range words {
		uniqueWords[w] = true
	}
	uniqueRatio := 0.0
	if len(words) > 0 {
		uniqueRatio = float64(len(uniqueWords)) / float64(len(words))
	}

	return quaternion.New(
		mean/255.0,
		variance/65025.0,
		punctDensity,
		uniqueRatio,
	).Normalize()
}

// ClassifyRegime determines the computational regime for a query.
func ClassifyRegime(text string) Regime {
	regime, matched := ClassifyRegimeKeywords(text)
	if matched {
		return regime
	}
	return RegimeStabilization
}

// ClassifyRegimeKeywords attempts keyword-based regime classification.
func ClassifyRegimeKeywords(text string) (Regime, bool) {
	lower := strings.ToLower(text)

	explorationKW := []string{
		"imagine", "create", "brainstorm", "what if", "story",
		"design", "invent", "dream", "visualize", "explore",
		"write a poem", "write a story", "write a song",
		"poem", "compose", "fiction", "fantasy",
	}
	for _, kw := range explorationKW {
		if strings.Contains(lower, kw) {
			return RegimeExploration, true
		}
	}

	optimizationKW := []string{
		"calculate", "compute", "compare", "optimize", "best",
		"exact", "precise", "analyze", "measure", "minimize",
		"maximize", "solve", "prove", "verify",
	}
	for _, kw := range optimizationKW {
		if strings.Contains(lower, kw) {
			return RegimeOptimization, true
		}
	}

	return RegimeStabilization, false
}

// ComputeDRSignature computes the digital root signature of text.
func ComputeDRSignature(text string) int64 {
	sum := int64(0)
	for _, b := range []byte(text) {
		sum += int64(b)
	}
	return vedic.DigitalRoot(sum)
}

// DrToRegime maps digital root to regime per GenomicsEngine proof.
func DrToRegime(dr int64) Regime {
	switch dr {
	case 1, 4, 7:
		return RegimeExploration
	case 2, 5, 8:
		return RegimeOptimization
	default:
		return RegimeStabilization
	}
}

// OilRatio estimates the information density of text.
func OilRatio(text string) float64 {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return 0.0
	}

	oilCount := 0
	for _, word := range words {
		if stopwords[word] {
			continue
		}
		if isNumeric(word) {
			oilCount++
			continue
		}
		if len(word) > 5 {
			oilCount++
			continue
		}
		hasSpecial := false
		for _, r := range word {
			if !unicode.IsLetter(r) {
				hasSpecial = true
				break
			}
		}
		if hasSpecial {
			oilCount++
		}
	}

	return float64(oilCount) / float64(len(words))
}

// ShunyamOilContrast computes information contrast using the Shunyam identity.
func ShunyamOilContrast(text string) float64 {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return 0.0
	}

	oil, water := 0, 0
	for _, word := range words {
		if stopwords[word] {
			water++
		} else {
			oil++
		}
	}

	total := oil + water
	if total == 0 {
		return 0.0
	}

	contrast := float64(oil-water) / float64(total)
	return (contrast + 1.0) / 2.0
}

// EstimateTokens estimates token count from text.
func EstimateTokens(text string) int {
	return len(text) / 4
}

// PredictConvergence tests the four Pi Emergence conditions.
func PredictConvergence(result OptimizationResult) (bool, [4]bool) {
	conditions := [4]bool{}
	conditions[0] = result.DRSignature > 0
	conditions[1] = result.OilRatio > 0.03 && result.OilRatio < 0.30

	maxWeight := result.RegimeDistribution[0]
	if result.RegimeDistribution[1] > maxWeight {
		maxWeight = result.RegimeDistribution[1]
	}
	if result.RegimeDistribution[2] > maxWeight {
		maxWeight = result.RegimeDistribution[2]
	}
	conditions[2] = maxWeight > 0.5
	conditions[3] = true

	allMet := conditions[0] && conditions[1] && conditions[2] && conditions[3]
	return allMet, conditions
}

func isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' && r != '-' && r != ',' {
			return false
		}
	}
	return len(s) > 0
}

func formatPercent(ratio float64) string {
	pct := int(math.Round(ratio * 100))
	if pct < 0 {
		pct = 0
	}
	if pct > 99 {
		pct = 99
	}
	return string(rune('0'+pct/10)) + string(rune('0'+pct%10))
}

var stopwords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true,
	"but": true, "in": true, "on": true, "at": true, "to": true,
	"for": true, "of": true, "with": true, "by": true, "from": true,
	"is": true, "are": true, "was": true, "were": true, "be": true,
	"been": true, "being": true, "have": true, "has": true, "had": true,
	"do": true, "does": true, "did": true, "will": true, "would": true,
	"should": true, "could": true, "can": true, "may": true, "might": true,
	"this": true, "that": true, "these": true, "those": true,
	"i": true, "you": true, "he": true, "she": true, "it": true,
	"we": true, "they": true, "what": true, "which": true, "who": true,
	"when": true, "where": true, "why": true, "how": true,
}
