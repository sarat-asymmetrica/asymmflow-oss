package main

import "testing"

// PH convergence D1 (PH 10f96a7): the loose folder-name fallback must never
// accept a digit-less token (a customer word) as a folder number — doing so
// collapsed every OneDrive opportunity for that customer onto one canonical
// key, hiding the rest.
func TestParseOneDriveFolderMeta_DigitGuard(t *testing.T) {
	// Digit-less customer word: NOT a folder number.
	meta := parseOneDriveFolderMeta("NORTHGRID")
	if meta.FolderNumber != "" {
		t.Fatalf("customer word must not become a folder number, got %q", meta.FolderNumber)
	}
	if meta.Title != "NORTHGRID" {
		t.Fatalf("customer word should be kept as title, got %q", meta.Title)
	}

	// Multi-word customer name: also refused.
	meta = parseOneDriveFolderMeta("RIVERSIDE UTILITIES")
	if meta.FolderNumber != "" {
		t.Fatalf("multi-word customer name must not become a folder number, got %q", meta.FolderNumber)
	}

	// Real folder numbers keep working through the loose fallback.
	meta = parseOneDriveFolderMeta("PH-104-26 Flow meters")
	if meta.FolderNumber == "" {
		t.Fatal("a real numbered folder must still parse")
	}
	meta = parseOneDriveFolderMeta("2026-17 Compressor spares")
	if meta.FolderNumber == "" || meta.Year != 2026 {
		t.Fatalf("year-prefixed folder must still parse, got %+v", meta)
	}
}
