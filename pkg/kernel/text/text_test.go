package text_test

import (
	"testing"

	"ph_holdings_app/pkg/kernel/text"
)

func TestFirstNonEmptyReturnsFirstNonBlank(t *testing.T) {
	got := text.FirstNonEmpty("", "  ", "hello", "world")
	if got != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got)
	}
}

func TestFirstNonEmptyAllEmpty(t *testing.T) {
	got := text.FirstNonEmpty("", "  ", "\t")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFirstNonEmptyNoArgs(t *testing.T) {
	got := text.FirstNonEmpty()
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestFirstNonEmptySingleValue(t *testing.T) {
	got := text.FirstNonEmpty("value")
	if got != "value" {
		t.Fatalf("expected %q, got %q", "value", got)
	}
}

func TestFirstNonEmptyTrimsResult(t *testing.T) {
	got := text.FirstNonEmpty("  spaced  ")
	if got != "spaced" {
		t.Fatalf("expected %q, got %q", "spaced", got)
	}
}

func TestWrap(t *testing.T) {
	lines := text.Wrap("alpha beta gamma delta", 11)
	if len(lines) != 2 || lines[0] != "alpha beta" || lines[1] != "gamma delta" {
		t.Fatalf("unexpected wrap: %v", lines)
	}
	// Paragraph breaks survive as empty entries.
	lines = text.Wrap("one\n\ntwo", 80)
	if len(lines) != 3 || lines[1] != "" {
		t.Fatalf("paragraph break lost: %v", lines)
	}
	// A word longer than the width stays on its own line.
	lines = text.Wrap("tiny enormouslylongword", 6)
	if len(lines) != 2 || lines[1] != "enormouslylongword" {
		t.Fatalf("long word handling: %v", lines)
	}
}

func TestEscapeLike(t *testing.T) {
	cases := map[string]string{
		"plain":      "plain",
		"50%":        `50\%`,
		"a_b":        `a\_b`,
		`back\slash`: `back\\slash`,
		`\%_`:        `\\\%\_`,
	}
	for in, want := range cases {
		if got := text.EscapeLike(in); got != want {
			t.Fatalf("EscapeLike(%q): expected %q, got %q", in, want, got)
		}
	}
}
