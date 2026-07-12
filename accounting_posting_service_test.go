package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
	"ph_holdings_app/pkg/finance/posting"
)

func TestPreviewCustomerInvoicePosting(t *testing.T) {
	app := setupPostingPreviewTestApp(t)
	invoice := Invoice{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceNumber: "INV-POST-001",
		InvoiceDate:   time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
		CustomerID:    "cust-1",
		CustomerName:  "Posting Customer",
		SubtotalBHD:   100,
		VATBHD:        10,
		GrandTotalBHD: 110,
		Division:      "Acme Instrumentation",
	}
	if err := app.db.Create(&invoice).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	entry, err := app.PreviewCustomerInvoicePosting(invoice.ID)
	if err != nil {
		t.Fatalf("preview customer invoice posting: %v", err)
	}
	if !entry.IsBalanced || entry.DebitTotal != 110 || entry.CreditTotal != 110 {
		t.Fatalf("unexpected preview totals: %+v", entry)
	}
	if entry.SourceType != "customer_invoice" || entry.SourceID != invoice.ID {
		t.Fatalf("unexpected source metadata: %+v", entry)
	}
}

func TestPreviewSupplierPaymentPosting(t *testing.T) {
	app := setupPostingPreviewTestApp(t)
	payment := SupplierPayment{
		Base:              Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SupplierInvoiceID: "sinv-1",
		SupplierID:        "sup-1",
		SupplierName:      "Posting Supplier",
		InvoiceNumber:     "SUP-POST-001",
		AmountBHD:         42.125,
		AmountForeign:     42.125,
		Currency:          "BHD",
		ExchangeRate:      1,
		PaymentDate:       time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
		PaymentMethod:     "Bank Transfer",
		Reference:         "BANK-REF",
		Division:          "Acme Instrumentation",
	}
	if err := app.db.Create(&payment).Error; err != nil {
		t.Fatalf("create supplier payment: %v", err)
	}

	entry, err := app.PreviewSupplierPaymentPosting(payment.ID)
	if err != nil {
		t.Fatalf("preview supplier payment posting: %v", err)
	}
	if !entry.IsBalanced || entry.DebitTotal != 42.125 || entry.CreditTotal != 42.125 {
		t.Fatalf("unexpected preview totals: %+v", entry)
	}
	if len(entry.Lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(entry.Lines))
	}
}

func TestGetTrialBalanceGate(t *testing.T) {
	app := setupPostingPreviewTestApp(t)
	ar := ChartOfAccount{
		Base:        Base{ID: "acct-ar", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		AccountCode: "1100",
		AccountName: "Accounts Receivable",
		AccountType: "Asset",
		IsActive:    true,
	}
	revenue := ChartOfAccount{
		Base:        Base{ID: "acct-revenue", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		AccountCode: "4000",
		AccountName: "Sales Revenue",
		AccountType: "Revenue",
		IsActive:    true,
	}
	if err := app.db.Create(&ar).Error; err != nil {
		t.Fatalf("create ar account: %v", err)
	}
	if err := app.db.Create(&revenue).Error; err != nil {
		t.Fatalf("create revenue account: %v", err)
	}
	entry := JournalEntry{
		Base:         Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		EntryNumber:  "JE-POST-001",
		EntryDate:    time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
		Description:  "Posted sales invoice",
		DebitTotal:   100,
		CreditTotal:  100,
		IsPosted:     true,
		FiscalYear:   2026,
		FiscalPeriod: 5,
		Lines: []JournalLine{
			{Base: Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()}, AccountID: ar.ID, AccountName: ar.AccountName, Debit: 100},
			{Base: Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()}, AccountID: revenue.ID, AccountName: revenue.AccountName, Credit: 100},
		},
	}
	if err := app.db.Create(&entry).Error; err != nil {
		t.Fatalf("create journal entry: %v", err)
	}

	gate, err := app.GetTrialBalanceGate(2026, 5)
	if err != nil {
		t.Fatalf("trial balance gate: %v", err)
	}
	if !gate.IsBalanced || gate.DebitTotal != 100 || gate.CreditTotal != 100 {
		t.Fatalf("unexpected gate: %+v", gate)
	}
	if len(gate.Rows) != 2 {
		t.Fatalf("expected 2 trial balance rows, got %d", len(gate.Rows))
	}
}

func TestCreateDraftJournalFromPostingUsesMappingsAndLinksSource(t *testing.T) {
	app := setupPostingPreviewTestApp(t)
	ar := ChartOfAccount{
		Base:        Base{ID: "acct-ar-custom", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		AccountCode: "1210",
		AccountName: "Mapped Accounts Receivable",
		AccountType: "Asset",
		IsActive:    true,
	}
	if err := app.db.Create(&ar).Error; err != nil {
		t.Fatalf("create mapped ar account: %v", err)
	}
	if err := app.db.Create(&AccountMapping{
		Base:            Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		TransactionType: "AR",
		AccountID:       ar.ID,
		AccountCode:     ar.AccountCode,
		AccountName:     ar.AccountName,
		IsActive:        true,
	}).Error; err != nil {
		t.Fatalf("create account mapping: %v", err)
	}

	invoice := Invoice{
		Base:          Base{ID: uuid.New().String(), CreatedAt: time.Now(), UpdatedAt: time.Now()},
		InvoiceNumber: "INV-DRAFT-001",
		InvoiceDate:   time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
		CustomerID:    "cust-1",
		CustomerName:  "Posting Customer",
		SubtotalBHD:   100,
		VATBHD:        10,
		GrandTotalBHD: 110,
	}
	if err := app.db.Create(&invoice).Error; err != nil {
		t.Fatalf("create invoice: %v", err)
	}

	journal, err := app.CreateDraftJournalFromPosting(posting.SourceCustomerInvoice, invoice.ID)
	if err != nil {
		t.Fatalf("create draft journal: %v", err)
	}
	if journal.IsPosted {
		t.Fatal("draft journal should not be posted")
	}
	if journal.SourceType != posting.SourceCustomerInvoice || journal.SourceID != invoice.ID {
		t.Fatalf("unexpected source metadata: %+v", journal)
	}
	if !strings.HasPrefix(journal.EntryNumber, "AUTO-JE-2026-") {
		t.Fatalf("unexpected entry number: %s", journal.EntryNumber)
	}
	if len(journal.Lines) != 3 {
		t.Fatalf("expected 3 journal lines, got %d", len(journal.Lines))
	}
	if journal.Lines[0].AccountID != ar.ID {
		t.Fatalf("expected mapped AR account %s, got %s", ar.ID, journal.Lines[0].AccountID)
	}

	var linked Invoice
	if err := app.db.First(&linked, "id = ?", invoice.ID).Error; err != nil {
		t.Fatalf("load linked invoice: %v", err)
	}
	if linked.JournalEntryID != journal.ID {
		t.Fatalf("invoice not linked to draft journal: got %q want %q", linked.JournalEntryID, journal.ID)
	}

	again, err := app.CreateDraftJournalFromPosting(posting.SourceCustomerInvoice, invoice.ID)
	if err != nil {
		t.Fatalf("idempotent draft journal call failed: %v", err)
	}
	if again.ID != journal.ID {
		t.Fatalf("expected existing journal %s, got %s", journal.ID, again.ID)
	}
	var count int64
	app.db.Model(&JournalEntry{}).Where("source_type = ? AND source_id = ?", posting.SourceCustomerInvoice, invoice.ID).Count(&count)
	if count != 1 {
		t.Fatalf("expected one journal for source, got %d", count)
	}
}

func TestGetPostingCoverageReport(t *testing.T) {
	app := setupPostingPreviewTestApp(t)
	linkedJournalID := uuid.New().String()
	if err := app.db.Create(&JournalEntry{
		Base:       Base{ID: linkedJournalID, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		SourceType: posting.SourceCustomerInvoice,
		SourceID:   "linked-invoice",
	}).Error; err != nil {
		t.Fatalf("create journal: %v", err)
	}
	invoices := []Invoice{
		{Base: Base{ID: "linked-invoice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, InvoiceNumber: "INV-LINKED", Status: "Sent", GrandTotalBHD: 100, JournalEntryID: linkedJournalID},
		{Base: Base{ID: "missing-invoice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, InvoiceNumber: "INV-MISSING", Status: "Sent", GrandTotalBHD: 100},
		{Base: Base{ID: "draft-invoice", CreatedAt: time.Now(), UpdatedAt: time.Now()}, InvoiceNumber: "INV-DRAFT", Status: "Draft", GrandTotalBHD: 100},
	}
	if err := app.db.Create(&invoices).Error; err != nil {
		t.Fatalf("create invoices: %v", err)
	}

	report, err := app.GetPostingCoverageReport()
	if err != nil {
		t.Fatalf("coverage report: %v", err)
	}
	if report.Total != 2 || report.Linked != 1 || report.Missing != 1 {
		t.Fatalf("unexpected coverage totals: %+v", report)
	}
	if report.Rows[0].SourceType != posting.SourceCustomerInvoice || report.Rows[0].Total != 2 || report.Rows[0].Linked != 1 {
		t.Fatalf("unexpected invoice row: %+v", report.Rows[0])
	}
}

func setupPostingPreviewTestApp(t *testing.T) *App {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&Invoice{}, &Payment{}, &SupplierInvoice{}, &SupplierPayment{}, &ChartOfAccount{}, &JournalEntry{}, &JournalLine{}, &AccountMapping{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	app := &App{
		db:            db,
		cache:         NewCache(),
		currentUserID: "test-user",
		currentUser: &User{
			Base:     Base{ID: "test-user"},
			Username: "test-admin",
			RoleName: "admin",
			Role: Role{
				Name:        "admin",
				DisplayName: "Administrator",
				Permissions: `["*"]`,
			},
		},
	}
	t.Cleanup(app.cache.Stop)
	return app
}
