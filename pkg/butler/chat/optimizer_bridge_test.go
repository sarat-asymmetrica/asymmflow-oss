package chat

import "testing"

func TestMathOptimizerCreation(t *testing.T) {
	if got := NewMathOptimizer(2048); got == nil {
		t.Fatalf("NewMathOptimizer returned nil")
	}
}

func TestOptimizeForButlerReturnsResult(t *testing.T) {
	optimizer := NewMathOptimizer(2048)
	result, systemPrompt := optimizer.OptimizeForButler("calculate 2+2")

	if result.OriginalPrompt == "" {
		t.Fatalf("expected original prompt")
	}
	if systemPrompt == "" {
		t.Fatalf("expected system prompt")
	}
}

func TestConversationCoherenceStartsAtOne(t *testing.T) {
	optimizer := NewMathOptimizer(2048)
	if got := optimizer.ConversationCoherence(); got != 1.0 {
		t.Fatalf("coherence = %f, want 1", got)
	}
}
