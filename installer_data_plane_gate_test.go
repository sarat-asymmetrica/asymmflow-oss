package main

// DP2 Gate G1 — "the installer never touches the data plane."
//
// The three-plane doctrine (FABLE_CAMPAIGN_DEPLOYMENT.md §0/§4.1) makes it
// STRUCTURALLY impossible for the installer to write the data plane: the NSIS
// script must contain no reference to a data-plane path
// (%APPDATA%\Asymmetrica\<slug>\data). This test is that guarantee, run in CI
// with the build rather than trusted to a human's grep (§9.6 G1).
//
// The discriminator that makes this precise:
//   - FORBIDDEN: a `data` path component under an `Asymmetrica\` root — i.e.
//     the data plane `Asymmetrica\<slug>\data`.
//   - ALLOWED and expected: the code-plane seed `$INSTDIR\data\ph_holdings.db`
//     (a `data` dir that is NOT under Asymmetrica — it is the packaged seed the
//     app copies out on first boot), and `Asymmetrica\<slug>\identity` (the
//     identity plane, seeded if-absent), and the word "data" in the uninstall
//     prose ("your business data ... remain at %APPDATA%\Asymmetrica\<slug>").
//
// Comment lines are stripped before scanning: the doctrine header block
// deliberately spells out the forbidden `Asymmetrica\...\data` pattern to
// document it, and that documentation must not trip its own gate.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// dataPlaneUnderVendor matches a `data` path component reached through an
// `Asymmetrica\` root — the data plane, and the only thing G1 forbids. It does
// NOT match `Asymmetrica\<slug>\identity`, `Asymmetrica\<slug>` with no `\data`
// suffix (the uninstall text), or `$INSTDIR\data` (no Asymmetrica root).
var dataPlaneUnderVendor = regexp.MustCompile(`Asymmetrica\\[^"\r\n]*\\data([\\"\s]|$)`)

// stripNSISComments removes full-line NSIS comments (a line whose first
// non-whitespace character is '#' or ';'). That is enough to exclude the
// doctrine header block (which names the forbidden pattern) while keeping every
// executable line — the plane paths that matter are all in code, not comments.
func stripNSISComments(src string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

// TestG1_Discriminator proves the gate has teeth: it fires on a real data-plane
// path and stays silent on every allowed neighbour, so a green G1 above is a
// real guarantee and not a mis-tuned regex that can never match.
func TestG1_Discriminator(t *testing.T) {
	forbidden := []string{
		`File "$APPDATA\Asymmetrica\AsymmFlow-PH\data\ph_holdings.db"`,
		`RMDir /r "$APPDATA\Asymmetrica\AsymmFlow-Dev\data"`,
		`SetOutPath "$APPDATA\Asymmetrica\${INSTALL_SLUG}\data\backups"`,
	}
	for _, s := range forbidden {
		if !dataPlaneUnderVendor.MatchString(s) {
			t.Errorf("gate has no teeth: failed to flag a data-plane path: %q", s)
		}
	}
	allowed := []string{
		`SetOutPath "$INSTDIR\data"`,                                              // code-plane seed
		`File "$INSTDIR\data\ph_holdings.db"`,                                     // code-plane seed
		`CreateDirectory "$APPDATA\Asymmetrica\${INSTALL_SLUG}\identity\ssot"`,    // identity plane
		`...backups remain at %APPDATA%\Asymmetrica\${INSTALL_SLUG} and will...`,  // uninstall text
		`!define UNINST_KEY_NAME "AsymmetricaAsymmFlowDev"`,                       // concatenated key name
	}
	for _, s := range allowed {
		if dataPlaneUnderVendor.MatchString(s) {
			t.Errorf("gate over-fires: flagged an allowed path: %q", s)
		}
	}

	// Comment stripping must drop the doctrine block that names the pattern.
	stripped := stripNSISComments("## DATA PLANE %APPDATA%\\Asymmetrica\\<slug>\\data\\\n  ; also \\data\ncode line\n")
	if strings.Contains(stripped, "Asymmetrica") {
		t.Errorf("comment stripping left a comment line in: %q", stripped)
	}
	if !strings.Contains(stripped, "code line") {
		t.Errorf("comment stripping dropped a code line: %q", stripped)
	}
}

func TestG1_InstallerNeverReferencesDataPlane(t *testing.T) {
	installerDir := filepath.Join("build", "windows", "installer")

	// project.nsi is the file we author and the only place plane paths appear;
	// wails_tools.nsh + branding.nsh are scanned too when present (the former is
	// generated at build time). Absence of the generated file is not a failure —
	// the committed project.nsi carries the contract.
	scanned := 0
	for _, name := range []string{"project.nsi", "wails_tools.nsh", "branding.nsh"} {
		path := filepath.Join(installerDir, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("reading %s: %v", path, err)
		}
		scanned++
		code := stripNSISComments(string(raw))

		if loc := dataPlaneUnderVendor.FindString(code); loc != "" {
			t.Errorf("G1 VIOLATION in %s: installer references the data plane %q — the "+
				"installer must never touch %%APPDATA%%\\Asymmetrica\\<slug>\\data (§4.1); "+
				"the app's update contract owns the data plane", name, loc)
		}

		// Enforce §9.6 G1's second clause: every executable use of the vendor
		// PATH root (`Asymmetrica\`, i.e. a path — not the concatenated
		// UNINST_KEY_NAME "Asymmetrica..." or the CompanyName "Asymmetrica")
		// is either the identity plane or the uninstall dialog text.
		for i, line := range strings.Split(code, "\n") {
			if !strings.Contains(line, `Asymmetrica\`) {
				continue
			}
			isIdentity := strings.Contains(line, `\identity`)
			isUninstallText := strings.Contains(line, "business data") ||
				strings.Contains(line, "remain at") ||
				strings.Contains(line, "MUI_UNCONFIRMPAGE")
			// A bare vendor root with no deeper component (identity-dir creation,
			// the uninstall text's `...\Asymmetrica\<slug>`) is acceptable too.
			if !isIdentity && !isUninstallText {
				t.Errorf("G1 VIOLATION in %s line %d: 'Asymmetrica' appears outside the "+
					"identity-seed section and uninstall text: %q", name, i+1, strings.TrimSpace(line))
			}
		}
	}

	if scanned == 0 {
		t.Fatalf("G1 found no NSIS script under %s to scan", installerDir)
	}

	// Positive sanity: the code-plane seed IS installed (so a green G1 can never
	// be an artifact of the installer having silently dropped the seed).
	project, err := os.ReadFile(filepath.Join(installerDir, "project.nsi"))
	if err != nil {
		t.Fatalf("reading project.nsi: %v", err)
	}
	if !strings.Contains(string(project), `$INSTDIR\data`) {
		t.Errorf("expected the code-plane seed ($INSTDIR\\data\\ph_holdings.db) to be installed; " +
			"not found in project.nsi")
	}
}
