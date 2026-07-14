package main

import (
	"os"
	"strings"
)

// sqlStringLiteral returns s as a single-quoted, escaped SQL string literal
// (e.g. Acme's -> 'Acme''s'). Used to build DDL fragments (column DEFAULTs)
// from overlay-configured values safely.
func sqlStringLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
