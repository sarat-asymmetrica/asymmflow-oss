package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	cashflowevidence "ph_holdings_app/pkg/cashflow/evidence"
)

func (a *App) GetCashflowEvidenceCommandCenter(days int) (cashflowevidence.CommandCenter, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return cashflowevidence.CommandCenter{}, err
	}
	if a.db == nil {
		return cashflowevidence.CommandCenter{}, fmt.Errorf("database connection not available")
	}
	if days <= 0 {
		days = 30
	}

	now := time.Now()
	window := cashflowevidence.TimeWindow{
		Start: now,
		End:   now.AddDate(0, 0, days),
		Label: fmt.Sprintf("Next %d days", days),
	}
	service := cashflowevidence.NewService(appCashflowEvidenceReader{app: a})
	return service.BuildCommandCenter(context.Background(), window)
}

func (a *App) ExportCashflowEvidencePack(days int) (string, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return "", err
	}

	generatedAt := time.Now()
	center, err := a.GetCashflowEvidenceCommandCenter(days)
	if err != nil {
		return "", err
	}
	payload, err := cashflowevidence.MarshalEvidencePackJSON(center, generatedAt.UTC())
	if err != nil {
		return "", fmt.Errorf("build cashflow evidence pack: %w", err)
	}

	exportDir := a.getExportDir("report", "", "Cashflow Evidence", generatedAt.Year())
	filename := fmt.Sprintf("Cashflow_Evidence_Pack_%s.json", generatedAt.Format("20060102_150405"))
	outputPath := filepath.Join(exportDir, filename)
	if err := os.WriteFile(outputPath, payload, 0640); err != nil {
		return "", fmt.Errorf("write cashflow evidence pack: %w", err)
	}
	return outputPath, nil
}

func (a *App) GetCashflowEvidenceAgentBrief(days int, maxChars int) (string, error) {
	if err := a.requirePermission("finance:view"); err != nil {
		return "", err
	}
	center, err := a.GetCashflowEvidenceCommandCenter(days)
	if err != nil {
		return "", err
	}
	return cashflowevidence.MarshalAgentBriefTOON(center, maxChars)
}

type appCashflowEvidenceReader struct {
	app *App
}

func (r appCashflowEvidenceReader) LoadCashflowEvidence(ctx context.Context, window cashflowevidence.TimeWindow) (cashflowevidence.CommandCenterInput, error) {
	if r.app == nil || r.app.db == nil {
		return cashflowevidence.CommandCenterInput{}, fmt.Errorf("database connection not available")
	}

	now := time.Now()
	if !window.Start.IsZero() {
		now = window.Start
	}
	end := window.End
	if end.IsZero() {
		end = now.AddDate(0, 0, 30)
	}

	coverage, err := r.app.GetPostingCoverageReport()
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}
	gate, err := r.app.GetTrialBalanceGate(now.Year(), 0)
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}

	openAR, err := r.sumOpenAR()
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}
	overdueAR, err := r.sumOverdueAR(now)
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}
	dueInWindow, err := r.sumDueInWindow(now, end)
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}

	totalBankLines, matchedBankLines, unmatchedBankAmount, err := r.bankMatchEvidence()
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}
	totalTraceableInvoices, traceableInvoices, err := r.invoiceTraceabilityEvidence()
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}
	openFollowUpTasks, err := r.openCashflowFollowUpTasks()
	if err != nil {
		return cashflowevidence.CommandCenterInput{}, err
	}

	evidenceSources := []cashflowevidence.EvidenceSourceInput{
		{
			SourceType: "finance_journal_links",
			Label:      "Finance journal links",
			Required:   int(coverage.Total),
			Present:    int(coverage.Linked),
			Confidence: 1,
		},
		{
			SourceType: "bank_reconciliation",
			Label:      "Bank reconciliation",
			Required:   int(totalBankLines),
			Present:    int(matchedBankLines),
			Confidence: 1,
		},
		{
			SourceType: "invoice_traceability",
			Label:      "Invoice traceability",
			Required:   int(totalTraceableInvoices),
			Present:    int(traceableInvoices),
			Confidence: 0.9,
		},
	}

	return cashflowevidence.CommandCenterInput{
		Window: window,
		Cash: cashflowevidence.CashExposureInput{
			OpenAR:      openAR,
			OverdueAR:   overdueAR,
			DueInWindow: dueInWindow,
		},
		EvidenceSources:      evidenceSources,
		PostingCoverage:      coverage,
		TrialBalanceGate:     gate,
		UnmatchedBankLines:   int(totalBankLines - matchedBankLines),
		UnmatchedBankAmount:  unmatchedBankAmount,
		OpenFollowUpTasks:    int(openFollowUpTasks),
		ExportableAuditItems: int(coverage.Linked + matchedBankLines + traceableInvoices),
	}, nil
}

func (r appCashflowEvidenceReader) sumOpenAR() (float64, error) {
	var total float64
	err := r.app.db.Model(&Invoice{}).
		Where("deleted_at IS NULL AND status IN ? AND outstanding_bhd > 0", openInvoiceStatusesForCashflow()).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&total).Error
	return total, err
}

func (r appCashflowEvidenceReader) sumOverdueAR(now time.Time) (float64, error) {
	var total float64
	err := r.app.db.Model(&Invoice{}).
		Where("deleted_at IS NULL AND status IN ? AND outstanding_bhd > 0 AND due_date < ?", openInvoiceStatusesForCashflow(), now).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&total).Error
	return total, err
}

func (r appCashflowEvidenceReader) sumDueInWindow(start, end time.Time) (float64, error) {
	var total float64
	err := r.app.db.Model(&Invoice{}).
		Where("deleted_at IS NULL AND status IN ? AND outstanding_bhd > 0 AND due_date >= ? AND due_date < ?", openInvoiceStatusesForCashflow(), start, end).
		Select("COALESCE(SUM(outstanding_bhd), 0)").
		Scan(&total).Error
	return total, err
}

func (r appCashflowEvidenceReader) bankMatchEvidence() (totalLines int64, matchedLines int64, unmatchedAmount float64, err error) {
	if err = r.app.db.Model(&BankStatementLine{}).Where("deleted_at IS NULL").Count(&totalLines).Error; err != nil {
		return 0, 0, 0, err
	}
	if err = r.app.db.Model(&BankStatementLine{}).Where("deleted_at IS NULL AND is_matched = ?", true).Count(&matchedLines).Error; err != nil {
		return 0, 0, 0, err
	}
	err = r.app.db.Model(&BankStatementLine{}).
		Where("deleted_at IS NULL AND is_matched = ?", false).
		Select("COALESCE(SUM(debit + credit), 0)").
		Scan(&unmatchedAmount).Error
	if err != nil {
		return 0, 0, 0, err
	}
	return totalLines, matchedLines, unmatchedAmount, nil
}

func (r appCashflowEvidenceReader) invoiceTraceabilityEvidence() (totalInvoices int64, traceableInvoices int64, err error) {
	if err = r.app.db.Model(&Invoice{}).
		Where("deleted_at IS NULL AND status IN ?", openInvoiceStatusesForCashflow()).
		Count(&totalInvoices).Error; err != nil {
		return 0, 0, err
	}
	err = r.app.db.Model(&Invoice{}).
		Where("deleted_at IS NULL AND status IN ?", openInvoiceStatusesForCashflow()).
		Where(
			"order_id <> '' OR rfq_id <> '' OR quote_id <> '' OR offer_id <> '' OR delivery_note_id <> '' OR delivery_note_number <> '' OR customer_po_number <> '' OR buyers_order_number <> '' OR despatch_document_no <> ''",
		).Count(&traceableInvoices).Error
	if err != nil {
		return 0, 0, err
	}
	return totalInvoices, traceableInvoices, nil
}

func (r appCashflowEvidenceReader) openCashflowFollowUpTasks() (int64, error) {
	var total int64
	err := r.app.db.Model(&FollowUpTask{}).
		Where("deleted_at IS NULL AND status NOT IN ?", []string{"completed", "cancelled"}).
		Where(
			"type IN ? OR LOWER(title) LIKE ? OR LOWER(description) LIKE ?",
			[]string{"collection", "payment", "receivables", "follow_up"},
			"%receivable%",
			"%payment%",
		).
		Count(&total).Error
	return total, err
}

func openInvoiceStatusesForCashflow() []string {
	return []string{"Sent", "PartiallyPaid", "Partially Paid", "Overdue"}
}
