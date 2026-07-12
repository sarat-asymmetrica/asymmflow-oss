// Package text provides general-purpose string helpers that are shared across
// the ph_holdings_app codebase. It is the canonical replacement for all
// duplicated firstNonEmpty() helpers.
package text

import "strings"

// FirstNonEmpty returns the first non-empty (after trimming whitespace) string
// from the given values. Returns "" if all values are empty or whitespace.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// EscapeLike escapes SQL LIKE wildcards (%, _) and the backslash escape
// character itself, for safe LIKE queries built from user input.
// Usage: db.Where("name LIKE ? ESCAPE '\\'", "%"+text.EscapeLike(input)+"%")
func EscapeLike(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "%", "\\%")
	s = strings.ReplaceAll(s, "_", "\\_")
	return s
}

// Wrap word-wraps text to the given character width, preserving paragraph
// breaks (blank lines survive as empty entries).
func Wrap(text string, width int) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	paragraphs := strings.Split(text, "\n")
	var lines []string

	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph) == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				lines = append(lines, currentLine)
				currentLine = word
			}
		}
		lines = append(lines, currentLine)
	}

	return lines
}
