package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// STATEMENT EXPORTS — Balance Sheet / General Ledger / Journal (Wave 9.3 B5)
// ============================================================================
//
// The owner needs to hand statements to their accountant. These exporters
// write CSV straight from the same generators/queries the Accounting screen
// already uses — they do not recompute anything. Balance Sheet reuses
// GenerateBalanceSheet (the Tally-derived report the Reports view already
// calls); General Ledger and Journal read the same JournalEntry/JournalLine
// rows GetJournalEntries lists, just made visible with the double-entry
// lines the Journal view currently collapses away.
//
// Scope note: the accounting chain here is single-tenant — ChartOfAccount
// and JournalEntry carry no company/branch field — so these exports honor
// only the active fiscal year. There is no company scope to filter by, and
// none is invented here.

// ExportBalanceSheetCSV writes the Balance Sheet as of Dec 31 of the given
// year to CSV. Figures come from GenerateBalanceSheet unchanged.
func (a *App) ExportBalanceSheetCSV(year int) (string, error) {
	bs, err := a.GenerateBalanceSheet(year) // gates "reports:view"
	if err != nil {
		return "", err
	}

	exportDir := a.getExportDir("report", "", "", year)
	filename := fmt.Sprintf("Balance_Sheet_%d.csv", year)
	csvPath := filepath.Join(exportDir, filename)

	file, err := os.Create(csvPath)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{activeOverlay.CompanyDisplayName + " - Balance Sheet"})
	writer.Write([]string{"As Of", bs.AsOfDate.Format("2006-01-02")})
	writer.Write([]string{"Generated", bs.GeneratedAt.Format("2006-01-02 15:04")})
	writer.Write([]string{""})
	writer.Write([]string{"Category", "Amount (BHD)"})
	writer.Write([]string{"ASSETS", ""})
	writer.Write([]string{"Cash", fmt.Sprintf("%.3f", bs.Cash)})
	writer.Write([]string{"Accounts Receivable", fmt.Sprintf("%.3f", bs.AccountsReceivable)})
	writer.Write([]string{"Inventory", fmt.Sprintf("%.3f", bs.Inventory)})
	writer.Write([]string{"Total Current Assets", fmt.Sprintf("%.3f", bs.TotalCurrentAssets)})
	writer.Write([]string{"TOTAL ASSETS", fmt.Sprintf("%.3f", bs.TotalAssets)})
	writer.Write([]string{""})
	writer.Write([]string{"LIABILITIES", ""})
	writer.Write([]string{"Accounts Payable", fmt.Sprintf("%.3f", bs.AccountsPayable)})
	writer.Write([]string{"Total Current Liabilities", fmt.Sprintf("%.3f", bs.TotalCurrentLiabilities)})
	writer.Write([]string{"TOTAL LIABILITIES", fmt.Sprintf("%.3f", bs.TotalLiabilities)})
	writer.Write([]string{""})
	writer.Write([]string{"EQUITY", ""})
	writer.Write([]string{"Retained Earnings", fmt.Sprintf("%.3f", bs.RetainedEarnings)})
	writer.Write([]string{"TOTAL EQUITY", fmt.Sprintf("%.3f", bs.TotalEquity)})

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync CSV file: %w", err)
	}

	return csvPath, nil
}

// ExportGeneralLedgerCSV writes a per-account ledger to CSV for the given
// fiscal year: posted journal lines only, chronological, with a running
// balance that opens on each account's carried-forward opening balance and
// closes on the account's year-end balance. Only posted entries are
// included — draft entries have not yet moved a ChartOfAccount balance (see
// PostJournalEntry), so mixing them in would misstate the ledger.
//
// Opening balance semantics (Wave 9.8 B2): computed at export time, NOT
// read from ChartOfAccount.Balance (which is an all-time cumulative field,
// not year-scoped, and would double count). Instead this sums every posted
// JournalLine for fiscal years strictly before the requested year, per
// account, using the identical Asset/Expense debit-increases rule the
// posting engine applies in PostJournalEntry (app_accounting_inventory.go).
// Year 1 (no prior posted fiscal years) naturally opens at 0. This is an
// export-truth computation only — it reads journal_entries/journal_lines
// with a different fiscal_year predicate; it does not write, post, or
// touch any transactional table.
//
// CSV column semantics: one "Opening Balance" row per account (Debit/Credit
// blank, Balance = carried-forward opening), followed by each in-year
// posted line (Debit (BHD), Credit (BHD) = that line's movement, Balance
// (BHD) = running balance after the line, opening included). An account
// with a nonzero opening balance but no in-year activity still gets an
// Opening Balance row so its carried balance isn't silently dropped from
// the ledger.
func (a *App) ExportGeneralLedgerCSV(year int) (string, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return "", err
	}
	if a.db == nil {
		return "", newError("DB_NOT_INITIALIZED", "Database connection not available", "")
	}

	var accounts []ChartOfAccount
	if err := a.db.Where("deleted_at IS NULL").Order("account_code ASC").Find(&accounts).Error; err != nil {
		return "", newError("DB_QUERY_FAILED", "Failed to retrieve chart of accounts", err.Error())
	}
	accountByID := make(map[string]ChartOfAccount, len(accounts))
	for _, acc := range accounts {
		accountByID[acc.ID] = acc
	}

	var entries []JournalEntry
	if err := a.db.Preload("Lines").
		Where("deleted_at IS NULL AND fiscal_year = ? AND is_posted = ?", year, true).
		Order("entry_date ASC, entry_number ASC").
		Find(&entries).Error; err != nil {
		return "", newError("DB_QUERY_FAILED", "Failed to retrieve journal entries", err.Error())
	}

	type ledgerRow struct {
		Date   time.Time
		Entry  string
		Desc   string
		Debit  float64
		Credit float64
	}
	linesByAccount := make(map[string][]ledgerRow)
	for _, entry := range entries {
		for _, line := range entry.Lines {
			linesByAccount[line.AccountID] = append(linesByAccount[line.AccountID], ledgerRow{
				Date:   entry.EntryDate,
				Entry:  entry.EntryNumber,
				Desc:   firstNonEmptyString(line.Description, entry.Description),
				Debit:  line.Debit,
				Credit: line.Credit,
			})
		}
	}

	// Opening balances: every posted line from a fiscal year strictly
	// before the requested year, reduced to a single net movement per
	// account. Year 1 (no prior posted fiscal years) yields an empty
	// result set, so every account naturally opens at 0.
	var priorEntries []JournalEntry
	if err := a.db.Preload("Lines").
		Where("deleted_at IS NULL AND fiscal_year < ? AND is_posted = ?", year, true).
		Find(&priorEntries).Error; err != nil {
		return "", newError("DB_QUERY_FAILED", "Failed to retrieve prior-year journal entries", err.Error())
	}
	openingByAccount := make(map[string]float64)
	for _, entry := range priorEntries {
		for _, line := range entry.Lines {
			acc := accountByID[line.AccountID]
			balanceIncreasesOnDebit := acc.AccountType == "Asset" || acc.AccountType == "Expense"
			if balanceIncreasesOnDebit {
				openingByAccount[line.AccountID] += line.Debit - line.Credit
			} else {
				openingByAccount[line.AccountID] += line.Credit - line.Debit
			}
		}
	}

	exportDir := a.getExportDir("report", "", "", year)
	filename := fmt.Sprintf("General_Ledger_%d.csv", year)
	csvPath := filepath.Join(exportDir, filename)

	file, err := os.Create(csvPath)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{fmt.Sprintf("%s - General Ledger %d", activeOverlay.CompanyDisplayName, year)})
	writer.Write([]string{"Generated", time.Now().Format("2006-01-02 15:04")})
	writer.Write([]string{"Note", "Posted entries only. Balance opens on the carried-forward opening balance (prior posted fiscal years) and shows running movement within the year."})
	writer.Write([]string{""})

	// Sort account IDs by account code for a stable, human-ordered ledger.
	// Include accounts with in-year activity AND accounts with a nonzero
	// carried opening balance but no in-year activity, so a carried
	// balance is never silently dropped from the ledger.
	idSet := make(map[string]bool, len(linesByAccount))
	for id := range linesByAccount {
		idSet[id] = true
	}
	for id, opening := range openingByAccount {
		if opening != 0 {
			idSet[id] = true
		}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return accountByID[ids[i]].AccountCode < accountByID[ids[j]].AccountCode
	})

	for _, id := range ids {
		acc := accountByID[id]
		rows := linesByAccount[id]
		sort.Slice(rows, func(i, j int) bool { return rows[i].Date.Before(rows[j].Date) })

		label := strings.TrimSpace(acc.AccountCode + " - " + acc.AccountName)
		if label == "-" {
			label = "(deleted account)"
		}
		writer.Write([]string{label})
		writer.Write([]string{"Date", "Entry Number", "Description", "Debit (BHD)", "Credit (BHD)", "Balance (BHD)"})

		balance := openingByAccount[id]
		writer.Write([]string{"Opening Balance", "", "", "", "", fmt.Sprintf("%.3f", balance)})

		balanceIncreasesOnDebit := acc.AccountType == "Asset" || acc.AccountType == "Expense"
		for _, row := range rows {
			if balanceIncreasesOnDebit {
				balance += row.Debit - row.Credit
			} else {
				balance += row.Credit - row.Debit
			}
			writer.Write([]string{
				row.Date.Format("2006-01-02"),
				row.Entry,
				row.Desc,
				fmt.Sprintf("%.3f", row.Debit),
				fmt.Sprintf("%.3f", row.Credit),
				fmt.Sprintf("%.3f", balance),
			})
		}
		writer.Write([]string{""})
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync CSV file: %w", err)
	}

	return csvPath, nil
}

// ExportJournalCSV writes every journal entry (draft and posted) for the
// given fiscal year to CSV, one row per line. Reuses GetJournalEntries with
// a raised limit — the Journal screen caps at 100 rows for display, which
// would silently truncate a full-year export.
func (a *App) ExportJournalCSV(year int) (string, error) {
	entries, err := a.GetJournalEntries(year, 0, nil, 10000) // gates "finance:view"
	if err != nil {
		return "", err
	}

	exportDir := a.getExportDir("report", "", "", year)
	filename := fmt.Sprintf("Journal_%d.csv", year)
	csvPath := filepath.Join(exportDir, filename)

	file, err := os.Create(csvPath)
	if err != nil {
		return "", fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{fmt.Sprintf("%s - Journal %d", activeOverlay.CompanyDisplayName, year)})
	writer.Write([]string{"Generated", time.Now().Format("2006-01-02 15:04")})
	writer.Write([]string{""})
	writer.Write([]string{"Entry Number", "Date", "Description", "Posted", "Account", "Debit (BHD)", "Credit (BHD)", "Line Description"})

	for _, entry := range entries {
		posted := "No"
		if entry.IsPosted {
			posted = "Yes"
		}
		if len(entry.Lines) == 0 {
			writer.Write([]string{entry.EntryNumber, entry.EntryDate.Format("2006-01-02"), entry.Description, posted, "", "", "", ""})
			continue
		}
		for _, line := range entry.Lines {
			writer.Write([]string{
				entry.EntryNumber,
				entry.EntryDate.Format("2006-01-02"),
				entry.Description,
				posted,
				line.AccountName,
				fmt.Sprintf("%.3f", line.Debit),
				fmt.Sprintf("%.3f", line.Credit),
				line.Description,
			})
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV write error: %w", err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync CSV file: %w", err)
	}

	return csvPath, nil
}
