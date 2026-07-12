package posting

import "sort"

type PostedJournalEntry struct {
	ID          string              `json:"id"`
	EntryNumber string              `json:"entry_number"`
	SourceType  string              `json:"source_type"`
	SourceID    string              `json:"source_id"`
	Lines       []PostedJournalLine `json:"lines"`
}

type PostedJournalLine struct {
	Account AccountRef `json:"account"`
	Debit   float64    `json:"debit"`
	Credit  float64    `json:"credit"`
}

type TrialBalanceGate struct {
	FiscalYear         int               `json:"fiscal_year"`
	FiscalPeriod       int               `json:"fiscal_period"`
	EntryCount         int               `json:"entry_count"`
	LineCount          int               `json:"line_count"`
	DebitTotal         float64           `json:"debit_total"`
	CreditTotal        float64           `json:"credit_total"`
	Difference         float64           `json:"difference"`
	IsBalanced         bool              `json:"is_balanced"`
	Rows               []TrialBalanceRow `json:"rows"`
	BalancedAccounts   []string          `json:"balanced_accounts,omitempty"`
	ImbalancedAccounts []string          `json:"imbalanced_accounts,omitempty"`
}

type TrialBalanceRow struct {
	Account AccountRef `json:"account"`
	Debit   float64    `json:"debit"`
	Credit  float64    `json:"credit"`
	Net     float64    `json:"net"`
}

func BuildTrialBalanceGate(fiscalYear, fiscalPeriod int, entries []PostedJournalEntry) TrialBalanceGate {
	rowsByKey := map[string]TrialBalanceRow{}
	gate := TrialBalanceGate{
		FiscalYear:   fiscalYear,
		FiscalPeriod: fiscalPeriod,
		EntryCount:   len(entries),
	}

	for _, entry := range entries {
		for _, line := range entry.Lines {
			debit := round(line.Debit)
			credit := round(line.Credit)
			if debit == 0 && credit == 0 {
				continue
			}
			gate.LineCount++
			gate.DebitTotal = round(gate.DebitTotal + debit)
			gate.CreditTotal = round(gate.CreditTotal + credit)

			key := line.Account.Code
			if key == "" {
				key = line.Account.Name
			}
			row := rowsByKey[key]
			if row.Account.Code == "" && row.Account.Name == "" {
				row.Account = line.Account
			}
			row.Debit = round(row.Debit + debit)
			row.Credit = round(row.Credit + credit)
			row.Net = round(row.Debit - row.Credit)
			rowsByKey[key] = row
		}
	}

	gate.Difference = round(gate.DebitTotal - gate.CreditTotal)
	gate.IsBalanced = gate.Difference == 0
	gate.Rows = make([]TrialBalanceRow, 0, len(rowsByKey))
	for _, row := range rowsByKey {
		gate.Rows = append(gate.Rows, row)
	}
	sort.Slice(gate.Rows, func(i, j int) bool {
		if gate.Rows[i].Account.Code == gate.Rows[j].Account.Code {
			return gate.Rows[i].Account.Name < gate.Rows[j].Account.Name
		}
		return gate.Rows[i].Account.Code < gate.Rows[j].Account.Code
	})

	for _, row := range gate.Rows {
		code := row.Account.Code
		if code == "" {
			code = row.Account.Name
		}
		if row.Net == 0 {
			gate.BalancedAccounts = append(gate.BalancedAccounts, code)
		} else {
			gate.ImbalancedAccounts = append(gate.ImbalancedAccounts, code)
		}
	}

	return gate
}
