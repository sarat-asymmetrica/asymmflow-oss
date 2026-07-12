package main

import (
	"strings"
	"testing"
)

// TestIsKnownExportReportType verifies the export whitelist accepts exactly the
// supported report types (case/space-insensitive) and rejects everything else,
// including path-injection attempts.
func TestIsKnownExportReportType(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"sales", true},
		{"customers", true},
		{"operations", true},
		{"inventory", true},
		{"financial", true},
		{"  Financial  ", true}, // trimmed + lowercased
		{"SALES", true},
		{"", false},
		{"unknown", false},
		{"../../etc/passwd", false},
		{"sales/../secret", false},
	}
	for _, c := range cases {
		if got := isKnownExportReportType(c.in); got != c.want {
			t.Errorf("isKnownExportReportType(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestSanitizeFileNameStripsPathInjection guards the ExportReport filename path:
// a hostile reportType must never survive into the output filename with path
// separators or traversal sequences intact.
func TestSanitizeFileNameStripsPathInjection(t *testing.T) {
	cases := []struct {
		in      string
		mustNot []string // substrings that must be absent from the result
	}{
		{"../../etc/passwd", []string{"..", "/"}},
		{"sales/../secret", []string{"..", "/"}},
		{"a\\b:c*d?", []string{"\\", ":", "*", "?"}},
	}
	for _, c := range cases {
		got := sanitizeFileName(strings.ToLower(strings.TrimSpace(c.in)))
		for _, bad := range c.mustNot {
			if strings.Contains(got, bad) {
				t.Errorf("sanitizeFileName(%q) = %q still contains %q", c.in, got, bad)
			}
		}
	}
	// Empty input collapses to the sentinel that ExportReport rewrites to "report".
	if got := sanitizeFileName(""); got != "unnamed" {
		t.Errorf("sanitizeFileName(\"\") = %q, want \"unnamed\"", got)
	}
}
