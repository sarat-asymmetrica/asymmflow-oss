// Grounded fast-path response builders: the deterministic customer,
// supplier, and revenue-projection answers the Butler returns without a
// model round-trip. Moved from the trading root's
// butler_grounded_fastpath.go in Wave 6 (Mission A.1, second cut) — the
// try* orchestrators (intent gating, hint inference, hub task creation)
// stay at the host and call these through the Service.
package context

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func (svc *Service) ResolveCustomerScope(reference string) ([]string, string) {
	ref := strings.ToLower(strings.TrimSpace(reference))
	if ref == "" {
		return nil, ""
	}

	resolution := svc.ResolveCustomerReference(reference)
	if resolution == nil || strings.TrimSpace(resolution.EntityID) == "" {
		return nil, ""
	}
	return []string{resolution.EntityID}, FirstNonEmpty(strings.TrimSpace(resolution.DisplayName), reference)
}

func (svc *Service) resolveCustomerRecordIDs(customerIDs []string) []string {
	if len(customerIDs) == 0 {
		return nil
	}

	type row struct {
		ID string
	}
	var rows []row
	svc.db.Table("customers").
		Select("id").
		Where("customer_id IN ?", customerIDs).
		Find(&rows)

	recordIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ID) != "" {
			recordIDs = append(recordIDs, strings.TrimSpace(row.ID))
		}
	}
	return uniqueStrings(recordIDs)
}

func (svc *Service) BuildCustomerNotesResponse(customerIDs []string, customerName string) string {
	recordIDs := svc.resolveCustomerRecordIDs(customerIDs)
	if len(recordIDs) == 0 {
		return fmt.Sprintf("I checked customer notes for %s and could not resolve a note record scope.", customerName)
	}

	type noteRow struct {
		NoteType  string
		Content   string
		CreatedAt time.Time
	}
	var notes []noteRow
	svc.db.Table("entity_notes").
		Select("note_type, content, created_at").
		Where("entity_type = ? AND entity_id IN ?", "customer", recordIDs).
		Order("created_at DESC").
		Limit(10).
		Scan(&notes)

	if len(notes) == 0 {
		return fmt.Sprintf("I checked notes for %s and found no recorded customer notes.", customerName)
	}

	lines := make([]string, 0, minInt(6, len(notes)))
	for i, note := range notes {
		if i >= 6 {
			break
		}
		lines = append(lines, fmt.Sprintf("- %s | %s | %s",
			note.CreatedAt.Format("2006-01-02"),
			FirstNonEmpty(strings.TrimSpace(note.NoteType), "general"),
			strings.TrimSpace(note.Content),
		))
	}

	return fmt.Sprintf("I found %d customer note(s) for %s.\n\nRecent notes:\n%s",
		len(notes), customerName, strings.Join(lines, "\n"))
}

func (svc *Service) BuildCustomerQuarterInvoiceAndHandlerResponse(customerIDs []string, customerName, q string) string {
	start, end, label, ok := ParseQuarterWindowFromQuery("this quarter")
	if !ok {
		now := time.Now()
		month := int(now.Month())
		quarter := (month-1)/3 + 1
		startMonth := time.Month((quarter-1)*3 + 1)
		start = time.Date(now.Year(), startMonth, 1, 0, 0, 0, 0, time.Local)
		end = start.AddDate(0, 3, 0).Add(-time.Second)
		label = fmt.Sprintf("Q%d %d", quarter, now.Year())
	}

	type invRow struct {
		InvoiceNumber string
		InvoiceDate   time.Time
		GrandTotalBHD float64
		Status        string
		IssuedBy      string
		Attention     string
		CreatedBy     string
	}

	var invoices []invRow
	svc.db.Table("invoices").
		Select("invoice_number, invoice_date, grand_total_bhd, status, issued_by, attention_person, created_by").
		Where("customer_id IN ? AND invoice_date >= ? AND invoice_date <= ? AND status NOT IN ?", customerIDs, start, end, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Order("invoice_date DESC").
		Limit(20).
		Scan(&invoices)

	handlers := svc.collectAccountHandlers(customerIDs)
	for _, inv := range invoices {
		handlers = appendIfPresent(handlers, inv.IssuedBy)
		handlers = appendIfPresent(handlers, inv.Attention)
		handlers = appendIfPresent(handlers, inv.CreatedBy)
	}

	if len(invoices) == 0 {
		return fmt.Sprintf("I checked %s invoices for %s and found no recorded invoices in %s.\n\nAccount handlers on record: %s.",
			label, customerName, label, joinOrNone(handlers))
	}

	total := 0.0
	lines := make([]string, 0, minInt(5, len(invoices)))
	for i, inv := range invoices {
		total += inv.GrandTotalBHD
		if i < 5 {
			lines = append(lines, fmt.Sprintf("- %s | %s | %.3f BHD | %s",
				strings.TrimSpace(inv.InvoiceNumber),
				inv.InvoiceDate.Format("2006-01-02"),
				inv.GrandTotalBHD,
				strings.TrimSpace(inv.Status),
			))
		}
	}

	return fmt.Sprintf("I found %d invoice(s) for %s in %s, totaling %.3f BHD.\n\nRecent invoices:\n%s\n\nPeople handling this account (from contacts/docs): %s.",
		len(invoices), customerName, label, total, strings.Join(lines, "\n"), joinOrNone(handlers))
}

func (svc *Service) BuildCustomerInvoiceOverviewResponse(customerIDs []string, customerName string) string {
	type invRow struct {
		ID            string
		InvoiceNumber string
		InvoiceDate   time.Time
		GrandTotalBHD float64
		Status        string
		IssuedBy      string
		Attention     string
		CreatedBy     string
	}

	var invoices []invRow
	svc.db.Table("invoices").
		Select("id, invoice_number, invoice_date, grand_total_bhd, status, issued_by, attention_person, created_by").
		Where("customer_id IN ? AND status NOT IN ?", customerIDs, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Order("invoice_date DESC").
		Limit(25).
		Scan(&invoices)

	handlers := svc.collectAccountHandlers(customerIDs)
	total := 0.0
	invoiceIDs := make([]string, 0, len(invoices))
	for _, inv := range invoices {
		total += inv.GrandTotalBHD
		invoiceIDs = append(invoiceIDs, inv.ID)
		handlers = appendIfPresent(handlers, inv.IssuedBy)
		handlers = appendIfPresent(handlers, inv.Attention)
		handlers = appendIfPresent(handlers, inv.CreatedBy)
	}
	handlers = uniqueStrings(handlers)

	if len(invoices) == 0 {
		return fmt.Sprintf("I checked invoices for %s and found no recorded invoices.\n\nPeople handling this account: %s.",
			customerName, joinOrNone(handlers))
	}

	type instRow struct {
		Equipment   string
		Model       string
		Description string
	}
	var instRows []instRow
	svc.db.Table("invoice_items").
		Select("equipment, model, description").
		Where("invoice_id IN ?", invoiceIDs).
		Limit(80).
		Scan(&instRows)

	instruments := make([]string, 0, len(instRows))
	for _, r := range instRows {
		label := strings.TrimSpace(FirstNonEmpty(r.Equipment, r.Model, r.Description))
		if label != "" {
			instruments = append(instruments, label)
		}
	}
	instruments = uniqueStrings(instruments)

	type payRow struct {
		InvoiceNumber string
		PaymentDate   time.Time
		AmountBHD     float64
	}
	var payments []payRow
	svc.db.Table("payments").
		Select("invoice_number, payment_date, amount_bhd").
		Where("invoice_id IN ?", invoiceIDs).
		Order("payment_date DESC").
		Limit(20).
		Scan(&payments)

	receivedLines := make([]string, 0, minInt(6, len(payments)))
	for i, p := range payments {
		if i >= 6 {
			break
		}
		receivedLines = append(receivedLines, fmt.Sprintf("- %s | %s | %.3f BHD",
			strings.TrimSpace(p.InvoiceNumber),
			p.PaymentDate.Format("2006-01-02"),
			p.AmountBHD,
		))
	}

	invoiceLines := make([]string, 0, minInt(6, len(invoices)))
	for i, inv := range invoices {
		if i >= 6 {
			break
		}
		invoiceLines = append(invoiceLines, fmt.Sprintf("- %s | %s | %.3f BHD | %s",
			strings.TrimSpace(inv.InvoiceNumber),
			inv.InvoiceDate.Format("2006-01-02"),
			inv.GrandTotalBHD,
			strings.TrimSpace(inv.Status),
		))
	}

	instText := "not captured on invoice lines"
	if len(instruments) > 0 {
		max := minInt(8, len(instruments))
		instText = strings.Join(instruments[:max], "; ")
	}

	receivedText := "no payment receipt entries found"
	if len(receivedLines) > 0 {
		receivedText = strings.Join(receivedLines, "\n")
	}

	return fmt.Sprintf("Here is the latest invoice snapshot for %s.\n\nTotal invoices: %d\nTotal billed: %.3f BHD\n\nRecent invoices:\n%s\n\nInstruments seen on invoice lines:\n%s\n\nPayments received (latest):\n%s\n\nPeople handling this account:\n%s",
		customerName,
		len(invoices),
		total,
		strings.Join(invoiceLines, "\n"),
		instText,
		receivedText,
		joinOrNone(handlers),
	)
}

func (svc *Service) BuildCustomerYearOffersResponse(customerIDs []string, customerName string, year int) string {
	start := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	end := time.Date(year, time.December, 31, 23, 59, 59, 0, time.Local)

	type offerRow struct {
		OfferNumber   string
		QuotationDate time.Time
		Stage         string
		TotalValueBHD float64
		QuoteType     string
		IssuedBy      string
		Attention     string
		CreatedBy     string
	}

	var offers []offerRow
	svc.db.Table("offers").
		Select("offer_number, quotation_date, stage, total_value_bhd, quote_type, issued_by, attention_person, created_by").
		Where("customer_id IN ? AND quotation_date >= ? AND quotation_date <= ?", customerIDs, start, end).
		Order("quotation_date DESC").
		Limit(30).
		Scan(&offers)

	handlers := svc.collectAccountHandlers(customerIDs)
	for _, off := range offers {
		handlers = appendIfPresent(handlers, off.IssuedBy)
		handlers = appendIfPresent(handlers, off.Attention)
		handlers = appendIfPresent(handlers, off.CreatedBy)
	}

	if len(offers) == 0 {
		return fmt.Sprintf("I checked offers for %s in %d and found no recorded offers.\n\nAccount handlers on record: %s.",
			customerName, year, joinOrNone(handlers))
	}

	total := 0.0
	lines := make([]string, 0, minInt(6, len(offers)))
	for i, off := range offers {
		total += off.TotalValueBHD
		if i < 6 {
			lines = append(lines, fmt.Sprintf("- %s | %s | %s | %.3f BHD | %s",
				strings.TrimSpace(off.OfferNumber),
				off.QuotationDate.Format("2006-01-02"),
				strings.TrimSpace(off.Stage),
				off.TotalValueBHD,
				FirstNonEmpty(strings.TrimSpace(off.QuoteType), "Quotation"),
			))
		}
	}

	return fmt.Sprintf("I found %d offer(s) for %s in %d, totaling %.3f BHD.\n\nRecent offers:\n%s\n\nPeople handling this account (from contacts/docs): %s.",
		len(offers), customerName, year, total, strings.Join(lines, "\n"), joinOrNone(handlers))
}

func (svc *Service) BuildCustomerOfferOverviewResponse(customerIDs []string, customerName string) string {
	type offerRow struct {
		OfferNumber   string
		QuotationDate time.Time
		Stage         string
		TotalValueBHD float64
		QuoteType     string
	}

	var offers []offerRow
	svc.db.Table("offers").
		Select("offer_number, quotation_date, stage, total_value_bhd, quote_type").
		Where("customer_id IN ?", customerIDs).
		Order("quotation_date DESC").
		Limit(20).
		Scan(&offers)

	if len(offers) == 0 {
		return fmt.Sprintf("I checked offers for %s and found no recorded offers.", customerName)
	}

	total := 0.0
	lines := make([]string, 0, minInt(8, len(offers)))
	for i, off := range offers {
		total += off.TotalValueBHD
		if i >= 8 {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s | %s | %s | %.3f BHD | %s",
			strings.TrimSpace(off.OfferNumber),
			off.QuotationDate.Format("2006-01-02"),
			strings.TrimSpace(off.Stage),
			off.TotalValueBHD,
			FirstNonEmpty(strings.TrimSpace(off.QuoteType), "Quotation"),
		))
	}

	return fmt.Sprintf("I found %d offer(s) for %s totaling %.3f BHD.\n\nRecent offers:\n%s",
		len(offers), customerName, total, strings.Join(lines, "\n"))
}

func (svc *Service) BuildCustomerLineItemsResponse(customerIDs []string, customerName string) string {
	type lineRow struct {
		InvoiceDate time.Time
		InvoiceNo   string
		Equipment   string
		Model       string
		Description string
		Qty         float64
		TotalBHD    float64
	}

	var lines []lineRow
	svc.db.Table("invoice_items").
		Select("invoices.invoice_date as invoice_date, invoices.invoice_number as invoice_no, invoice_items.equipment, invoice_items.model, invoice_items.description, invoice_items.quantity as qty, invoice_items.total_bhd as total_bhd").
		Joins("JOIN invoices ON invoices.id = invoice_items.invoice_id").
		Where("invoices.customer_id IN ? AND invoices.status NOT IN ?", customerIDs, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Order("invoices.invoice_date DESC").
		Limit(80).
		Scan(&lines)

	if len(lines) == 0 {
		return fmt.Sprintf("I checked sold line items for %s but found no invoice line-item records in current data.", customerName)
	}

	aggQty := make(map[string]float64)
	aggValue := make(map[string]float64)
	for _, ln := range lines {
		key := strings.TrimSpace(FirstNonEmpty(ln.Equipment, ln.Model, ln.Description))
		if key == "" {
			continue
		}
		aggQty[key] += ln.Qty
		aggValue[key] += ln.TotalBHD
	}

	type itemAgg struct {
		Name  string
		Qty   float64
		Value float64
	}
	items := make([]itemAgg, 0, len(aggQty))
	for k, q := range aggQty {
		items = append(items, itemAgg{Name: k, Qty: q, Value: aggValue[k]})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Value > items[j].Value })

	summary := make([]string, 0, minInt(8, len(items)))
	for i, it := range items {
		if i >= 8 {
			break
		}
		summary = append(summary, fmt.Sprintf("- %s | Qty %.2f | %.3f BHD", it.Name, it.Qty, it.Value))
	}

	recent := make([]string, 0, minInt(6, len(lines)))
	for i, ln := range lines {
		if i >= 6 {
			break
		}
		label := strings.TrimSpace(FirstNonEmpty(ln.Equipment, ln.Model, ln.Description))
		recent = append(recent, fmt.Sprintf("- %s | %s | %s | Qty %.2f | %.3f BHD",
			strings.TrimSpace(ln.InvoiceNo),
			ln.InvoiceDate.Format("2006-01-02"),
			label,
			ln.Qty,
			ln.TotalBHD,
		))
	}

	return fmt.Sprintf("Yes, I have line-item visibility for %s.\n\nTop sold items (by billed value):\n%s\n\nRecent sold lines:\n%s",
		customerName, strings.Join(summary, "\n"), strings.Join(recent, "\n"))
}

func (svc *Service) BuildRevenueProjectionResponse() string {
	type yearRevenue struct {
		Year  string
		Total float64
	}
	var yearly []yearRevenue
	svc.db.Table("invoices").
		Select("strftime('%Y', invoice_date) as year, COALESCE(SUM(grand_total_bhd),0) as total").
		Where("status NOT IN ?", []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Group("strftime('%Y', invoice_date)").
		Order("year DESC").
		Limit(4).
		Scan(&yearly)

	currentYear := time.Now().Year()
	lastYear := currentYear - 1
	var lastYearRevenue float64
	var currentYTD float64
	for _, yr := range yearly {
		if yr.Year == fmt.Sprintf("%d", lastYear) {
			lastYearRevenue = yr.Total
		}
		if yr.Year == fmt.Sprintf("%d", currentYear) {
			currentYTD = yr.Total
		}
	}

	var activePipeline float64
	svc.db.Table("offers").
		Where("stage NOT IN ?", []string{"Lost", "Cancelled", "Expired"}).
		Select("COALESCE(SUM(total_value_bhd),0)").
		Scan(&activePipeline)

	month := int(time.Now().Month())
	runRateProjection := lastYearRevenue
	if month > 0 && currentYTD > 0 {
		runRateProjection = (currentYTD / float64(month)) * 12.0
	}
	blendedProjection := (0.7 * runRateProjection) + (0.3 * (lastYearRevenue + 0.20*activePipeline))

	return fmt.Sprintf("Latest revenue projection (grounded from current records):\n\n- Last full year (%d) revenue: %.3f BHD\n- Current year (%d) YTD billed: %.3f BHD\n- Active offer pipeline: %.3f BHD\n\nProjection scenarios:\n- Run-rate projection: %.3f BHD\n- Blended projection (run-rate + pipeline conversion): %.3f BHD\n\nAssumption used: ~20%% pipeline conversion in-year for blended view.",
		lastYear, lastYearRevenue,
		currentYear, currentYTD,
		activePipeline,
		runRateProjection,
		blendedProjection,
	)
}

func (svc *Service) BuildSupplierPaymentHistoryResponse(supplierID, supplierName string) string {
	type paymentRow struct {
		PaymentDate   time.Time
		AmountBHD     float64
		PaymentMethod string
		Reference     string
		InvoiceNumber string
	}
	var payments []paymentRow
	svc.db.Table("supplier_payments").
		Select("payment_date, amount_bhd, payment_method, reference, invoice_number").
		Where("supplier_id = ?", supplierID).
		Order("payment_date DESC").
		Limit(12).
		Scan(&payments)

	type invoiceSummary struct {
		Count         int64
		TotalBHD      float64
		PendingCount  int64
		PendingAmount float64
	}
	var summary invoiceSummary
	svc.db.Table("supplier_invoices").
		Select("COUNT(*) as count, COALESCE(SUM(total_bhd), 0) as total_bhd, SUM(CASE WHEN payment_status <> 'Paid' THEN 1 ELSE 0 END) as pending_count, COALESCE(SUM(CASE WHEN payment_status <> 'Paid' THEN total_bhd ELSE 0 END), 0) as pending_amount").
		Where("supplier_id = ?", supplierID).
		Scan(&summary)

	if len(payments) == 0 && summary.Count == 0 {
		return fmt.Sprintf("I checked supplier payment history for %s and found no recorded supplier invoices or payments.", supplierName)
	}

	lines := make([]string, 0, minInt(6, len(payments)))
	totalPaid := 0.0
	for i, payment := range payments {
		totalPaid += payment.AmountBHD
		if i >= 6 {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s | %s | %.3f BHD | %s | %s",
			payment.PaymentDate.Format("2006-01-02"),
			FirstNonEmpty(strings.TrimSpace(payment.InvoiceNumber), "no invoice ref"),
			payment.AmountBHD,
			FirstNonEmpty(strings.TrimSpace(payment.PaymentMethod), "payment method n/a"),
			FirstNonEmpty(strings.TrimSpace(payment.Reference), "no reference"),
		))
	}

	paymentText := "No supplier payment rows found."
	if len(lines) > 0 {
		paymentText = strings.Join(lines, "\n")
	}

	return fmt.Sprintf("Here is the supplier payment history for %s.\n\nRecorded supplier invoices: %d totaling %.3f BHD\nPending supplier invoices: %d totaling %.3f BHD\nRecent supplier payments totaled %.3f BHD in the sampled history.\n\nRecent payments:\n%s",
		supplierName, summary.Count, summary.TotalBHD, summary.PendingCount, summary.PendingAmount, totalPaid, paymentText)
}

func (svc *Service) BuildSupplierPurchaseOverviewResponse(supplierID, supplierName string) string {
	type invoiceRow struct {
		InvoiceNumber string
		InvoiceDate   time.Time
		TotalBHD      float64
		Status        string
	}
	var invoices []invoiceRow
	svc.db.Table("supplier_invoices").
		Select("invoice_number, invoice_date, total_bhd, status").
		Where("supplier_id = ?", supplierID).
		Order("invoice_date DESC").
		Limit(12).
		Scan(&invoices)

	type lineRow struct {
		Description string
		Qty         float64
		TotalBHD    float64
	}
	var lines []lineRow
	svc.db.Table("supplier_invoice_items").
		Select("supplier_invoice_items.description, supplier_invoice_items.quantity as qty, COALESCE(supplier_invoice_items.total_bhd, supplier_invoice_items.total_price, 0) as total_bhd").
		Joins("JOIN supplier_invoices ON supplier_invoices.id = supplier_invoice_items.supplier_invoice_id").
		Where("supplier_invoices.supplier_id = ?", supplierID).
		Order("supplier_invoices.invoice_date DESC").
		Limit(20).
		Scan(&lines)

	if len(invoices) == 0 && len(lines) == 0 {
		return fmt.Sprintf("I checked purchasing history for %s and found no recorded supplier invoices or line items.", supplierName)
	}

	total := 0.0
	invoiceLines := make([]string, 0, minInt(5, len(invoices)))
	for i, invoice := range invoices {
		total += invoice.TotalBHD
		if i >= 5 {
			continue
		}
		invoiceLines = append(invoiceLines, fmt.Sprintf("- %s | %s | %.3f BHD | %s",
			strings.TrimSpace(invoice.InvoiceNumber),
			invoice.InvoiceDate.Format("2006-01-02"),
			invoice.TotalBHD,
			strings.TrimSpace(invoice.Status),
		))
	}

	itemLines := make([]string, 0, minInt(6, len(lines)))
	for i, line := range lines {
		if i >= 6 {
			break
		}
		itemLines = append(itemLines, fmt.Sprintf("- %s | qty %.3f | %.3f BHD",
			FirstNonEmpty(strings.TrimSpace(line.Description), "line item"),
			line.Qty,
			line.TotalBHD,
		))
	}

	itemsText := "No supplier line items captured."
	if len(itemLines) > 0 {
		itemsText = strings.Join(itemLines, "\n")
	}

	return fmt.Sprintf("Here is what we have bought from %s.\n\nRecorded supplier invoices: %d totaling %.3f BHD\n\nRecent supplier invoices:\n%s\n\nRecent purchased line items:\n%s",
		supplierName, len(invoices), total, strings.Join(invoiceLines, "\n"), itemsText)
}

func (svc *Service) BuildSupplierIssueOverviewResponse(supplierID, supplierName string) string {
	type issueRow struct {
		OrderRef    string
		Description string
		Status      string
		CostBHD     float64
		CreatedAt   time.Time
	}
	var issues []issueRow
	svc.db.Table("supplier_issues").
		Select("order_ref, description, status, cost_bhd, created_at").
		Where("supplier_id = ?", supplierID).
		Order("created_at DESC").
		Limit(10).
		Scan(&issues)

	if len(issues) == 0 {
		return fmt.Sprintf("I checked supplier issues for %s and found no active issue records.", supplierName)
	}

	openCount := 0
	lines := make([]string, 0, minInt(6, len(issues)))
	for i, issue := range issues {
		if !strings.EqualFold(strings.TrimSpace(issue.Status), "resolved") && !strings.EqualFold(strings.TrimSpace(issue.Status), "closed") {
			openCount++
		}
		if i >= 6 {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s | %s | %.3f BHD | %s | %s",
			FirstNonEmpty(strings.TrimSpace(issue.OrderRef), "no order ref"),
			FirstNonEmpty(strings.TrimSpace(issue.Status), "unknown"),
			issue.CostBHD,
			issue.CreatedAt.Format("2006-01-02"),
			strings.TrimSpace(issue.Description),
		))
	}

	return fmt.Sprintf("I found %d supplier issue record(s) for %s, with %d still open.\n\nRecent issues:\n%s",
		len(issues), supplierName, openCount, strings.Join(lines, "\n"))
}

func (svc *Service) collectAccountHandlers(customerIDs []string) []string {
	type contactRow struct {
		ContactName string
		JobTitle    string
		Email       string
		Phone       string
		IsPrimary   bool
	}

	var contacts []contactRow
	svc.db.Table("customer_contacts").
		Select("contact_name, job_title, email, phone, is_primary_contact").
		Where("customer_id IN ?", customerIDs).
		Order("is_primary_contact DESC, contact_name ASC").
		Limit(20).
		Scan(&contacts)

	handlers := make([]string, 0, len(contacts))
	for _, c := range contacts {
		name := strings.TrimSpace(c.ContactName)
		if name == "" {
			continue
		}
		role := strings.TrimSpace(c.JobTitle)
		email := strings.TrimSpace(c.Email)
		if role != "" && email != "" {
			handlers = append(handlers, fmt.Sprintf("%s (%s, %s)", name, role, email))
			continue
		}
		if role != "" {
			handlers = append(handlers, fmt.Sprintf("%s (%s)", name, role))
			continue
		}
		if email != "" {
			handlers = append(handlers, fmt.Sprintf("%s (%s)", name, email))
			continue
		}
		handlers = append(handlers, name)
	}

	sort.Strings(handlers)
	return uniqueStrings(handlers)
}

func appendIfPresent(items []string, raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return items
	}
	return append(items, value)
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none found in current records"
	}
	if len(items) > 8 {
		items = items[:8]
	}
	return strings.Join(items, "; ")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
