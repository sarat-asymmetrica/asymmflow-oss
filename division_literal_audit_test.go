package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoSyntheticDivisionLiteralsInLiveCode is the Wave 12 regression tripwire.
//
// After the division registry (pkg/overlay) became the single source of division
// vocabulary, the synthetic division names must not reappear as hardcoded
// literals in executable code. If a code path compares, scopes, validates,
// renders, or defaults a division against a frozen literal instead of reading
// the overlay registry, that is an Article III violation (one job, one path) and
// silently re-introduces the mis-scoping landmine this wave removed.
//
// SCOPE: the ERP application only — Go source (this module) plus frontend/src.
// The separate packages/ design-system showcase and e2e test fixtures are NOT
// governed by the ERP's overlay and are out of scope.
//
// WHAT IS AUDITED: executable code. Comments are stripped before matching —
// documentation and the synthetic canon it references (per SYNTHETIC_IDENTITY.md)
// are allowed; only code that would run with a hardcoded division name fails.
//
// EXEMPTIONS (the legitimate homes for the synthetic literals):
//   - the registry source of truth (pkg/overlay/overlay.go, business_rules.go)
//   - the frontend builtin fallback mirror (divisions.svelte.ts, wailsMock.ts,
//     brand.ts) — the never-empty-selector guarantee, analogous to BuiltinDefaults
//   - the annotated example config (data/overlay*.json)
//   - seed / bulk-import / standalone tooling (import_2026_data.go, cmd/…)
//   - GORM struct-tag column defaults (pkg/crm/domain.go): compile-time
//     constants that cannot hold a runtime registry value (see report residual)
//   - tests (_test.go), generated bindings (frontend/wailsjs), deps, build output
func TestNoSyntheticDivisionLiteralsInLiveCode(t *testing.T) {
	forbidden := []string{
		"Acme Instrumentation",
		"Beacon Controls",
		"ACME INSTRUMENTATION",
		"BEACON CONTROLS",
	}

	// Exact repo-relative paths that legitimately carry the synthetic literals.
	exemptFile := map[string]bool{
		"pkg/overlay/overlay.go":              true,
		"pkg/overlay/business_rules.go":       true,
		"frontend/src/lib/divisions.svelte.ts": true,
		"frontend/src/lib/wailsMock.ts":       true,
		"frontend/src/lib/brand.ts":           true,
		"import_2026_data.go":                 true, // bulk 2026 data-import canon
		"pkg/crm/domain.go":                   true, // GORM struct-tag default (residual)
	}
	// Directory prefixes that are out of scope or generated.
	exemptPrefix := []string{
		"cmd/",                 // standalone tooling (export scripts, etc.)
		"frontend/wailsjs/",    // generated Wails bindings
		"frontend/tests/",      // e2e test fixtures
		"packages/",            // separate design-system showcase (not the ERP)
		"data/",                // overlay.json + data files
	}

	scanExt := map[string]bool{".go": true, ".ts": true, ".svelte": true, ".js": true}

	isExempt := func(rel string) bool {
		if exemptFile[rel] {
			return true
		}
		if strings.HasSuffix(rel, "_test.go") {
			return true
		}
		for _, p := range exemptPrefix {
			if strings.HasPrefix(rel, p) {
				return true
			}
		}
		return false
	}

	var offenders []string
	err := filepath.WalkDir(".", func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build":
				return filepath.SkipDir
			}
			return nil
		}
		if !scanExt[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		rel := filepath.ToSlash(path)
		rel = strings.TrimPrefix(rel, "./")
		// Only audit the ERP frontend under frontend/src (skip other frontend/*).
		if strings.HasPrefix(rel, "frontend/") && !strings.HasPrefix(rel, "frontend/src/") {
			return nil
		}
		if isExempt(rel) {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		code := stripComments(string(data))
		for _, lit := range forbidden {
			if strings.Contains(code, lit) {
				offenders = append(offenders, rel+"  →  hardcoded "+strconvQuote(lit))
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking source tree: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("Wave 12: synthetic division literal(s) found in executable code — read the "+
			"overlay registry (pkg/overlay / the divisions store) instead of hardcoding a "+
			"division name:\n  %s", strings.Join(offenders, "\n  "))
	}
}

// stripComments removes // line comments, /* */ block comments, and <!-- -->
// HTML comments so the audit matches executable code only, not documentation.
// It is intentionally conservative: it does not parse string literals, so a
// "//" inside a string truncates the (rare) rest of that line — safe here
// because a hardcoded division literal sits before any such comment marker, and
// the goal is to flag division names in real code, never in prose.
func stripComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	runes := []rune(src)
	n := len(runes)
	for i := 0; i < n; i++ {
		// Block comment /* ... */
		if i+1 < n && runes[i] == '/' && runes[i+1] == '*' {
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i++ // skip the closing '/'
			continue
		}
		// HTML comment <!-- ... -->
		if i+3 < n && runes[i] == '<' && runes[i+1] == '!' && runes[i+2] == '-' && runes[i+3] == '-' {
			i += 4
			for i+2 < n && !(runes[i] == '-' && runes[i+1] == '-' && runes[i+2] == '>') {
				i++
			}
			i += 2 // skip "->"
			continue
		}
		// Line comment // ... (to end of line)
		if i+1 < n && runes[i] == '/' && runes[i+1] == '/' {
			for i < n && runes[i] != '\n' {
				i++
			}
			if i < n {
				b.WriteRune('\n')
			}
			continue
		}
		b.WriteRune(runes[i])
	}
	return b.String()
}

// strconvQuote wraps a literal in double quotes for the failure message without
// pulling in strconv just for this test helper.
func strconvQuote(s string) string { return "\"" + s + "\"" }
