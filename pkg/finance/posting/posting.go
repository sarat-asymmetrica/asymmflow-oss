// Package posting builds auditable accounting previews from finance documents.
package posting

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ph_holdings_app/pkg/kernel/money"
)

const (
	SourceCustomerInvoice = "customer_invoice"
	SourceCustomerPayment = "customer_payment"
	SourceSupplierInvoice = "supplier_invoice"
	SourceSupplierPayment = "supplier_payment"
	DefaultCurrency       = "BHD"
)

type AccountRole string

const (
	RoleAccountsReceivable AccountRole = "accounts_receivable"
	RoleAccountsPayable    AccountRole = "accounts_payable"
	RoleBank               AccountRole = "bank"
	RoleRevenue            AccountRole = "revenue"
	RolePurchases          AccountRole = "purchases"
	RoleVATOutput          AccountRole = "vat_output"
	RoleVATInput           AccountRole = "vat_input"
)

type AccountRef struct {
	Role AccountRole `json:"role"`
	Code string      `json:"code"`
	Name string      `json:"name"`
}

type AccountSet struct {
	AccountsReceivable AccountRef `json:"accounts_receivable"`
	AccountsPayable    AccountRef `json:"accounts_payable"`
	Bank               AccountRef `json:"bank"`
	Revenue            AccountRef `json:"revenue"`
	Purchases          AccountRef `json:"purchases"`
	VATOutput          AccountRef `json:"vat_output"`
	VATInput           AccountRef `json:"vat_input"`
}

func DefaultAccountSet() AccountSet {
	return AccountSet{
		AccountsReceivable: AccountRef{Role: RoleAccountsReceivable, Code: "1200", Name: "Accounts Receivable"},
		AccountsPayable:    AccountRef{Role: RoleAccountsPayable, Code: "2000", Name: "Accounts Payable"},
		Bank:               AccountRef{Role: RoleBank, Code: "1000", Name: "Bank / Cash"},
		Revenue:            AccountRef{Role: RoleRevenue, Code: "4000", Name: "Sales Revenue"},
		Purchases:          AccountRef{Role: RolePurchases, Code: "5000", Name: "Purchases / Cost of Goods Sold"},
		VATOutput:          AccountRef{Role: RoleVATOutput, Code: "2100", Name: "VAT Payable (Output)"},
		VATInput:           AccountRef{Role: RoleVATInput, Code: "2110", Name: "VAT Receivable (Input)"},
	}
}

type SourceDocument struct {
	ID          string    `json:"id"`
	Number      string    `json:"number"`
	PartyID     string    `json:"party_id"`
	PartyName   string    `json:"party_name"`
	Date        time.Time `json:"date"`
	Currency    string    `json:"currency"`
	SubtotalBHD float64   `json:"subtotal_bhd"`
	VATBHD      float64   `json:"vat_bhd"`
	TotalBHD    float64   `json:"total_bhd"`
	Division    string    `json:"division"`
	Reference   string    `json:"reference"`
}

type Entry struct {
	SourceType   string    `json:"source_type"`
	SourceID     string    `json:"source_id"`
	SourceNumber string    `json:"source_number"`
	EntryDate    time.Time `json:"entry_date"`
	Description  string    `json:"description"`
	Currency     string    `json:"currency"`
	DebitTotal   float64   `json:"debit_total"`
	CreditTotal  float64   `json:"credit_total"`
	IsBalanced   bool      `json:"is_balanced"`
	Lines        []Line    `json:"lines"`
}

type Line struct {
	Account AccountRef `json:"account"`
	Debit   float64    `json:"debit"`
	Credit  float64    `json:"credit"`
	Memo    string     `json:"memo"`
}

func CustomerInvoicePreview(doc SourceDocument) (Entry, error) {
	if err := validateDocument(doc); err != nil {
		return Entry{}, err
	}
	accounts := DefaultAccountSet()
	entry := newEntry(SourceCustomerInvoice, doc, fmt.Sprintf("Customer invoice %s - %s", doc.Number, doc.PartyName))
	entry.Lines = append(entry.Lines,
		debit(accounts.AccountsReceivable, doc.TotalBHD, "Invoice receivable"),
		credit(accounts.Revenue, doc.SubtotalBHD, "Sales revenue"),
	)
	if round(doc.VATBHD) > 0 {
		entry.Lines = append(entry.Lines, credit(accounts.VATOutput, doc.VATBHD, "Output VAT"))
	}
	return finish(entry)
}

func CustomerPaymentPreview(doc SourceDocument) (Entry, error) {
	if err := validatePaymentDocument(doc); err != nil {
		return Entry{}, err
	}
	accounts := DefaultAccountSet()
	entry := newEntry(SourceCustomerPayment, doc, fmt.Sprintf("Customer payment %s - %s", doc.ReferenceOrNumber(), doc.PartyName))
	entry.Lines = append(entry.Lines,
		debit(accounts.Bank, doc.TotalBHD, "Cash/bank received"),
		credit(accounts.AccountsReceivable, doc.TotalBHD, "Reduce customer receivable"),
	)
	return finish(entry)
}

func SupplierInvoicePreview(doc SourceDocument) (Entry, error) {
	if err := validateDocument(doc); err != nil {
		return Entry{}, err
	}
	accounts := DefaultAccountSet()
	entry := newEntry(SourceSupplierInvoice, doc, fmt.Sprintf("Supplier invoice %s - %s", doc.Number, doc.PartyName))
	entry.Lines = append(entry.Lines,
		debit(accounts.Purchases, doc.SubtotalBHD, "Purchases / cost recognized"),
	)
	if round(doc.VATBHD) > 0 {
		entry.Lines = append(entry.Lines, debit(accounts.VATInput, doc.VATBHD, "Input VAT"))
	}
	entry.Lines = append(entry.Lines, credit(accounts.AccountsPayable, doc.TotalBHD, "Supplier payable"))
	return finish(entry)
}

func SupplierPaymentPreview(doc SourceDocument) (Entry, error) {
	if err := validatePaymentDocument(doc); err != nil {
		return Entry{}, err
	}
	accounts := DefaultAccountSet()
	entry := newEntry(SourceSupplierPayment, doc, fmt.Sprintf("Supplier payment %s - %s", doc.ReferenceOrNumber(), doc.PartyName))
	entry.Lines = append(entry.Lines,
		debit(accounts.AccountsPayable, doc.TotalBHD, "Reduce supplier payable"),
		credit(accounts.Bank, doc.TotalBHD, "Cash/bank paid"),
	)
	return finish(entry)
}

func (d SourceDocument) ReferenceOrNumber() string {
	if strings.TrimSpace(d.Reference) != "" {
		return strings.TrimSpace(d.Reference)
	}
	return strings.TrimSpace(d.Number)
}

func validateDocument(doc SourceDocument) error {
	if strings.TrimSpace(doc.ID) == "" {
		return errors.New("source document id is required")
	}
	if round(doc.SubtotalBHD) < 0 || round(doc.VATBHD) < 0 || round(doc.TotalBHD) <= 0 {
		return errors.New("source amounts must be non-negative and total must be positive")
	}
	expected := round(doc.SubtotalBHD + doc.VATBHD)
	if expected != round(doc.TotalBHD) {
		return fmt.Errorf("source total %.3f does not equal subtotal %.3f + vat %.3f", doc.TotalBHD, doc.SubtotalBHD, doc.VATBHD)
	}
	return nil
}

func validatePaymentDocument(doc SourceDocument) error {
	if strings.TrimSpace(doc.ID) == "" {
		return errors.New("source document id is required")
	}
	if round(doc.TotalBHD) <= 0 {
		return errors.New("payment amount must be positive")
	}
	return nil
}

func newEntry(sourceType string, doc SourceDocument, description string) Entry {
	currency := strings.TrimSpace(doc.Currency)
	if currency == "" {
		currency = DefaultCurrency
	}
	return Entry{
		SourceType:   sourceType,
		SourceID:     strings.TrimSpace(doc.ID),
		SourceNumber: strings.TrimSpace(doc.Number),
		EntryDate:    doc.Date,
		Description:  strings.TrimSpace(description),
		Currency:     currency,
	}
}

func debit(account AccountRef, amount float64, memo string) Line {
	return Line{Account: account, Debit: round(amount), Memo: memo}
}

func credit(account AccountRef, amount float64, memo string) Line {
	return Line{Account: account, Credit: round(amount), Memo: memo}
}

func finish(entry Entry) (Entry, error) {
	var debitTotal, creditTotal float64
	lines := make([]Line, 0, len(entry.Lines))
	for _, line := range entry.Lines {
		line.Debit = round(line.Debit)
		line.Credit = round(line.Credit)
		if line.Debit == 0 && line.Credit == 0 {
			continue
		}
		if line.Debit > 0 && line.Credit > 0 {
			return Entry{}, errors.New("journal line cannot contain both debit and credit")
		}
		debitTotal += line.Debit
		creditTotal += line.Credit
		lines = append(lines, line)
	}
	entry.Lines = lines
	entry.DebitTotal = round(debitTotal)
	entry.CreditTotal = round(creditTotal)
	entry.IsBalanced = entry.DebitTotal == entry.CreditTotal && entry.DebitTotal > 0
	if !entry.IsBalanced {
		return Entry{}, fmt.Errorf("posting preview is unbalanced: debit %.3f credit %.3f", entry.DebitTotal, entry.CreditTotal)
	}
	return entry, nil
}

func round(v float64) float64 {
	return money.RoundFloat64(v, 3)
}
