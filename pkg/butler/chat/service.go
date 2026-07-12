// Package chat contains Butler chat core helpers.
package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var promptInjectionPatterns = regexp.MustCompile(`(?i)(\[/?INST\]|<</?SYS>>|<\||(\|>)|</?s>)`)
var contextCleanupPatterns = regexp.MustCompile(`(?i)(\[/?ACTIONS\]|SYSTEM:|INSTRUCTIONS:)`)

func InferActionLabel(actionType, target string) string {
	if actionType == "" {
		return "Action"
	}

	prettyTarget := strings.ReplaceAll(target, "_", " ")
	if prettyTarget == "" {
		return fmt.Sprintf("%s action", strings.ToUpper(string(actionType[0]))+actionType[1:])
	}

	return fmt.Sprintf("%s %s", strings.ToUpper(string(actionType[0]))+actionType[1:], strings.TrimSpace(prettyTarget))
}

// SanitizeForPrompt removes potential prompt injection markers from
// user-supplied data before it is interpolated into model prompts.
func SanitizeForPrompt(input string) string {
	sanitized := strings.ReplaceAll(input, "```", "")
	sanitized = promptInjectionPatterns.ReplaceAllString(sanitized, "")
	return sanitized
}

func MarshalContextForPrompt(context map[string]any, maxChars int) string {
	contextJSON, _ := json.MarshalIndent(context, "", "  ")
	contextStr := SanitizeForPrompt(string(contextJSON))
	contextStr = contextCleanupPatterns.ReplaceAllString(contextStr, "")
	if maxChars > 0 && len(contextStr) > maxChars {
		return contextStr[:maxChars] + fmt.Sprintf("\n... [context truncated at %d chars]", maxChars)
	}
	return contextStr
}

func CleanPromptContext(input string) string {
	return contextCleanupPatterns.ReplaceAllString(input, "")
}
