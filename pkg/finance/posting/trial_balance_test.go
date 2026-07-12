package posting

import "testing"

func TestBuildTrialBalanceGateBalancesPostedEntries(t *testing.T) {
	entries := []PostedJournalEntry{
		{
			ID:          "je-1",
			EntryNumber: "JE-1",
			Lines: []PostedJournalLine{
				{Account: DefaultAccountSet().AccountsReceivable, Debit: 110},
				{Account: DefaultAccountSet().Revenue, Credit: 100},
				{Account: DefaultAccountSet().VATOutput, Credit: 10},
			},
		},
		{
			ID:          "je-2",
			EntryNumber: "JE-2",
			Lines: []PostedJournalLine{
				{Account: DefaultAccountSet().Bank, Debit: 110},
				{Account: DefaultAccountSet().AccountsReceivable, Credit: 110},
			},
		},
	}

	gate := BuildTrialBalanceGate(2026, 5, entries)
	if !gate.IsBalanced {
		t.Fatalf("trial balance should balance: %+v", gate)
	}
	if gate.EntryCount != 2 || gate.LineCount != 5 {
		t.Fatalf("unexpected counts: %+v", gate)
	}
	if gate.DebitTotal != 220 || gate.CreditTotal != 220 {
		t.Fatalf("unexpected totals: %+v", gate)
	}
}

func TestBuildTrialBalanceGateClassifiesAccountBalance(t *testing.T) {
	accounts := DefaultAccountSet()
	// Account A (AccountsReceivable): debit=100, credit=100 → balanced
	// Account B (Bank): debit=200, credit=150 → imbalanced
	entries := []PostedJournalEntry{
		{
			ID:          "je-1",
			EntryNumber: "JE-1",
			Lines: []PostedJournalLine{
				{Account: accounts.AccountsReceivable, Debit: 100},
				{Account: accounts.AccountsReceivable, Credit: 100},
				{Account: accounts.Bank, Debit: 200},
				{Account: accounts.Bank, Credit: 150},
			},
		},
	}

	gate := BuildTrialBalanceGate(2026, 5, entries)

	if gate.IsBalanced {
		t.Fatalf("global trial balance should not be balanced: %+v", gate)
	}

	arCode := accounts.AccountsReceivable.Code
	if arCode == "" {
		arCode = accounts.AccountsReceivable.Name
	}
	bankCode := accounts.Bank.Code
	if bankCode == "" {
		bankCode = accounts.Bank.Name
	}

	foundBalanced := false
	for _, code := range gate.BalancedAccounts {
		if code == arCode {
			foundBalanced = true
			break
		}
	}
	if !foundBalanced {
		t.Fatalf("expected %q in BalancedAccounts %v", arCode, gate.BalancedAccounts)
	}

	foundImbalanced := false
	for _, code := range gate.ImbalancedAccounts {
		if code == bankCode {
			foundImbalanced = true
			break
		}
	}
	if !foundImbalanced {
		t.Fatalf("expected %q in ImbalancedAccounts %v", bankCode, gate.ImbalancedAccounts)
	}
}

func TestBuildTrialBalanceGateFlagsDifference(t *testing.T) {
	gate := BuildTrialBalanceGate(2026, 5, []PostedJournalEntry{
		{
			ID:          "je-1",
			EntryNumber: "JE-1",
			Lines: []PostedJournalLine{
				{Account: DefaultAccountSet().Bank, Debit: 100},
				{Account: DefaultAccountSet().Revenue, Credit: 90},
			},
		},
	})
	if gate.IsBalanced {
		t.Fatalf("trial balance should not balance: %+v", gate)
	}
	if gate.Difference != 10 {
		t.Fatalf("unexpected difference: %.3f", gate.Difference)
	}
}
