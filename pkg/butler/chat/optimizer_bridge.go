package chat

import (
	"ph_holdings_app/pkg/math/conversation"
	"ph_holdings_app/pkg/math/prism"
	"ph_holdings_app/pkg/math/trident"
)

// MathOptimizer wraps the Trident optimizer for Butler use.
type MathOptimizer struct {
	trident *trident.Optimizer
	chain   *conversation.ConversationChain
}

// NewMathOptimizer creates a Butler math optimizer.
func NewMathOptimizer(tokenBudget int) *MathOptimizer {
	opt := trident.NewOptimizer(tokenBudget)
	opt.EnableDRFusion()
	opt.SetModelRouter([3]string{"sarvam-105b", "sarvam-105b", "sarvam-30b"})
	return &MathOptimizer{
		trident: opt,
		chain:   conversation.NewConversationChain(),
	}
}

// OptimizeForButler runs Trident and returns a conversation-aware system prompt.
func (m *MathOptimizer) OptimizeForButler(userPrompt string) (trident.OptimizationResult, string) {
	result := m.trident.OptimizePrompt(userPrompt)
	m.chain.AddMessage(userPrompt)
	systemPrompt := prism.GenerateConversationPrism(result, m.chain)
	return result, systemPrompt
}

// ShouldSkipAPI returns true if the math layer can answer locally.
func (m *MathOptimizer) ShouldSkipAPI(result trident.OptimizationResult) bool {
	return result.SkipAPICall
}

// ConversationCoherence returns the current conversation focus score [0,1].
func (m *MathOptimizer) ConversationCoherence() float64 {
	return m.chain.CoherenceScore()
}
