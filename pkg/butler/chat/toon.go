package chat

import (
	"fmt"
	"strings"

	"ph_holdings_app/pkg/toon"
)

// MarshalContextForPromptCompact encodes Butler context as TOON for the LLM
// boundary, falling back to JSON when the compact encoder cannot normalize input.
func MarshalContextForPromptCompact(context map[string]any, maxChars int) string {
	contextTOON, err := toon.Marshal(context)
	if err != nil {
		return MarshalContextForPrompt(context, maxChars)
	}
	contextStr := "format: TOON\n" + SanitizeForPrompt(contextTOON)
	contextStr = contextCleanupPatterns.ReplaceAllString(contextStr, "")
	if maxChars > 0 && len(contextStr) > maxChars {
		return contextStr[:maxChars] + fmt.Sprintf("\n... [context truncated at %d chars]", maxChars)
	}
	return strings.TrimSpace(contextStr)
}

// ContextEncodingSavings reports compact JSON and TOON byte/token estimates for
// sample prompts and diagnostics.
func ContextEncodingSavings(context map[string]any) (jsonBytes, toonBytes, jsonTokens, toonTokens int, err error) {
	jsonBytes, toonBytes, err = toon.Savings(context)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	jsonTokens = toon.EstimatedTokens(strings.Repeat("x", jsonBytes))
	toonTokens = toon.EstimatedTokens(strings.Repeat("x", toonBytes))
	return jsonBytes, toonBytes, jsonTokens, toonTokens, nil
}
