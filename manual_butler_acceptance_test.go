//go:build manual

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestManualButlerBusinessAcceptance(t *testing.T) {
	sourceDB := strings.TrimSpace(os.Getenv("BUTLER_ACCEPTANCE_DB"))
	if sourceDB == "" {
		t.Skip("set BUTLER_ACCEPTANCE_DB=/absolute/path/to/ph_holdings.db to run the live Butler acceptance pass")
	}

	tempDir := t.TempDir()
	workingDB := filepath.Join(tempDir, "butler_acceptance.db")
	copyFileForAcceptance(t, sourceDB, workingDB)

	db, err := gorm.Open(sqlite.Open(workingDB), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
	})
	require.NoError(t, err)

	app := &App{
		db:            db,
		cache:         NewCache(),
		currentUserID: "manual-butler-acceptance",
		currentUser: &User{
			Base:     Base{ID: "manual-butler-acceptance"},
			Username: "manual-butler-acceptance",
			RoleName: "admin",
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator",
				Permissions: `["*"]`,
			},
		},
	}
	t.Cleanup(app.cache.Stop)

	cases := []struct {
		name        string
		prompt      string
		mustContain []string
	}{
		{"peri_tasks", "How many tasks are assigned to Jamie right now?", []string{"Jamie", "active task(s)", "unread notification(s)"}},
		{"peri_notifications", "What notifications does Jamie have?", []string{"Jamie", "unread notification(s)", "Recent notifications"}},
		{"npc_notes", "What notes do we have for NPC?", []string{"I checked notes for", "NPC"}},
		{"npc_invoices_quarter", "Show me NPC invoices this quarter", []string{"I checked", "NPC", "Q2 2026"}},
		{"npc_line_items", "What have we sold to NPC?", []string{"NPC", "line-item"}},
		{"npc_offers_year", "Show me NPC offers this year", []string{"NPC", "2026"}},
		{"rhine_payment_history", "Tell me about Rhine Instruments payment history", []string{"supplier payment history", "Recent payments"}},
		{"rhine_purchase_history", "What did we buy from Rhine Instruments?", []string{"what we have bought", "Recent purchased line items"}},
		{"rhine_issue_history", "Are there any active supplier issues for Rhine Instruments?", []string{"supplier issues for Rhine Instruments", "active issue records"}},
		{"revenue_projection", "Give me the revenue projection", []string{"Latest revenue projection", "Projection scenarios"}},
	}

	reportLines := []string{
		"# Butler Acceptance Report",
		"",
		fmt.Sprintf("- Date: %s", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("- Source DB: `%s`", sourceDB),
		fmt.Sprintf("- Working copy: `%s`", workingDB),
		"",
		"## Results",
		"",
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := app.ChatWithButlerPersistent("", tc.prompt)
			require.NoError(t, err)
			for _, expected := range tc.mustContain {
				require.Contains(t, resp.Response, expected)
			}
			t.Logf("prompt=%q", tc.prompt)
			t.Logf("response=%s", resp.Response)

			reportLines = append(reportLines,
				fmt.Sprintf("### %s", tc.name),
				"",
				fmt.Sprintf("- Prompt: %s", tc.prompt),
				"- Status: PASS",
				fmt.Sprintf("- Response excerpt: %s", firstAcceptanceExcerpt(resp.Response)),
				"",
			)
		})
	}

	if reportPath := strings.TrimSpace(os.Getenv("BUTLER_ACCEPTANCE_REPORT")); reportPath != "" {
		require.NoError(t, os.MkdirAll(filepath.Dir(reportPath), 0o755))
		require.NoError(t, os.WriteFile(reportPath, []byte(strings.Join(reportLines, "\n")), 0o644))
		t.Logf("wrote Butler acceptance report to %s", reportPath)
	}
}

func copyFileForAcceptance(t *testing.T, src, dst string) {
	t.Helper()

	source, err := os.Open(src)
	require.NoError(t, err)
	defer source.Close()

	target, err := os.Create(dst)
	require.NoError(t, err)
	defer target.Close()

	_, err = io.Copy(target, source)
	require.NoError(t, err)
	require.NoError(t, target.Sync())
}

func firstAcceptanceExcerpt(text string) string {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
	if len(text) <= 220 {
		return text
	}
	return text[:220] + "..."
}
