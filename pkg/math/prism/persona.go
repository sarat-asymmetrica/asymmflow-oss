package prism

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/math/conversation"
	"ph_holdings_app/pkg/math/trident"
)

// PersonaArchetypes maps regime and contrast band to a core identity.
var PersonaArchetypes = [3][3]string{
	{
		"a creative mind diving deep into rich specifics, weaving details into unexpected connections",
		"an imaginative guide blending invention with substance, grounding creative leaps in reality",
		"a boundless explorer casting wide nets across domains, opening doors to new possibilities",
	},
	{
		"a precision engineer building rigorous arguments step by step, every sentence advancing the proof",
		"an analytical navigator balancing rigor with clarity, showing both the path and the destination",
		"a systems thinker bringing structure to broad challenges, defining terms before solving problems",
	},
	{
		"a direct authority delivering clear answers with confident efficiency",
		"a knowledgeable guide making complex ideas approachable and well-structured",
		"a patient teacher illuminating concepts through examples and analogies",
	},
}

// ContrastBand returns 0=high, 1=mid, 2=low for indexing PersonaArchetypes.
func ContrastBand(contrast float64) int {
	if contrast > 0.7 {
		return 0
	}
	if contrast < 0.3 {
		return 2
	}
	return 1
}

// GeneratePersona creates a cohesive persona identity from mathematical analysis.
func GeneratePersona(result trident.OptimizationResult) string {
	regimeIdx := int(result.DetectedRegime)
	if regimeIdx < 0 || regimeIdx > 2 {
		regimeIdx = 2
	}
	archetype := PersonaArchetypes[regimeIdx][ContrastBand(result.ShunyamContrast)]

	quality, ok := NavaYoniQuality[result.DRSignature]
	if !ok {
		return fmt.Sprintf("You are %s.", archetype)
	}
	return fmt.Sprintf("You are %s, responding %s.", archetype, quality)
}

// GenerateConversationPrism creates a conversation-aware prism prompt.
func GenerateConversationPrism(result trident.OptimizationResult, chain *conversation.ConversationChain) string {
	persona := GeneratePersona(result)
	base := GeneratePrismPrompt(result)

	if chain == nil || chain.Length() < 2 {
		if advisory := ResonanceAdvisory(result); advisory != "" {
			return persona + " " + base + " " + advisory
		}
		return persona + " " + base
	}

	var extras []string

	coherence := chain.CoherenceScore()
	if coherence > 0.8 {
		extras = append(extras, "This is part of a focused conversation; build directly on the established thread.")
	} else if coherence < 0.4 {
		extras = append(extras, "The conversation has been wide-ranging; provide a self-contained answer that doesn't assume prior context.")
	}

	_, drifted, previous := chain.RegimeDrift()
	if drifted {
		extras = append(extras, fmt.Sprintf("The conversation just shifted from %s to %s mode; acknowledge the new direction while bridging naturally.", previous, result.DetectedRegime))
	}

	if chain.Momentum() > 0.7 {
		extras = append(extras, "The conversation is moving quickly; be concise and adaptive.")
	}

	if advisory := ResonanceAdvisory(result); advisory != "" {
		extras = append(extras, advisory)
	}

	compositeDR := chain.CompositeDR()
	if compositeDR >= 1 && compositeDR <= 9 && result.DRSignature >= 1 && result.DRSignature <= 9 {
		if HasNavaYoniSynergy(result.DRSignature, compositeDR) {
			extras = append(extras, "The energy of this query harmonizes with the conversation's overall signature; let the response resonate deeply.")
		}
	}

	if len(extras) == 0 {
		return persona + " " + base
	}
	return persona + " " + base + " " + strings.Join(extras, " ")
}
