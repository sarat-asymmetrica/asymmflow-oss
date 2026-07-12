package finance

import (
	"fmt"
	"math"
	"strings"
	"time"

	vm "ph_holdings_app/internal/viewmodel"
	"ph_holdings_app/internal/viewmodel/shared"
	domain "ph_holdings_app/pkg/finance"
	"ph_holdings_app/pkg/kernel/text"
)

// BuildInvoiceListVM constructs the invoice list ViewModel from domain data.
func BuildInvoiceListVM(invoices []domain.Invoice, page, pageSize int) InvoiceListVM {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = len(invoices)
	}

	rows := make([]shared.TableRow, 0, len(invoices))
	var totalOutstanding float64
	var overdueAmount float64
	var overdueCount int
	var paidThisMonth float64
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	for _, invoice := range invoices {
		status := InvoiceStatusBadge(invoice.Status)
		totalOutstanding += invoice.OutstandingBHD
		if strings.EqualFold(invoice.Status, "Overdue") || (!invoice.DueDate.IsZero() && invoice.DueDate.Before(now) && invoice.OutstandingBHD > 0) {
			overdueCount++
			overdueAmount += invoice.OutstandingBHD
		}
		if strings.EqualFold(invoice.Status, "Paid") && !invoice.InvoiceDate.Before(currentMonthStart) {
			paidThisMonth += invoice.GrandTotalBHD
		}
		rows = append(rows, shared.TableRow{
			ID: invoice.ID,
			Fields: map[string]any{
				"invoiceNumber": invoice.InvoiceNumber,
				"customerName":  invoice.CustomerName,
				"invoiceDate":   FormatDate(invoice.InvoiceDate),
				"dueDate":       FormatDate(invoice.DueDate),
				"status":        status,
				"total":         FormatMoney(invoice.GrandTotalBHD, "BHD"),
				"outstanding":   FormatMoney(invoice.OutstandingBHD, "BHD"),
			},
			Status: status.Label,
			Actions: []vm.ActionButton{
				{Label: "View", Action: "invoice.view", Icon: "eye", Variant: "secondary", Enabled: true},
				{Label: "Record Payment", Action: "invoice.recordPayment", Icon: "credit-card", Variant: "primary", Enabled: invoice.OutstandingBHD > 0},
			},
		})
	}

	return InvoiceListVM{
		Table: shared.TableVM{
			Columns: []shared.TableColumn{
				{Key: "invoiceNumber", Label: "Invoice", Type: "text", Sortable: true, Width: "130px"},
				{Key: "customerName", Label: "Customer", Type: "text", Sortable: true},
				{Key: "invoiceDate", Label: "Invoice Date", Type: "date", Sortable: true, Width: "130px"},
				{Key: "dueDate", Label: "Due Date", Type: "date", Sortable: true, Width: "130px"},
				{Key: "status", Label: "Status", Type: "status", Sortable: true, Width: "120px"},
				{Key: "total", Label: "Total", Type: "currency", Sortable: true, Align: "right", Currency: "BHD"},
				{Key: "outstanding", Label: "Outstanding", Type: "currency", Sortable: true, Align: "right", Currency: "BHD"},
			},
			Rows:       rows,
			TotalRows:  len(invoices),
			Page:       page,
			PageSize:   pageSize,
			SortColumn: "invoiceDate",
			SortDesc:   true,
			Filters: []shared.TableFilter{
				{Column: "status", Type: "select", Options: InvoiceStatusOptions()},
				{Column: "customerName", Type: "text"},
				{Column: "invoiceDate", Type: "dateRange"},
			},
		},
		Summary: InvoiceSummaryVM{
			TotalOutstanding:   FormatMoney(totalOutstanding, "BHD"),
			OverdueCount:       overdueCount,
			OverdueAmount:      FormatMoney(overdueAmount, "BHD"),
			PaidThisMonth:      FormatMoney(paidThisMonth, "BHD"),
			AveragePaymentDays: averagePaymentDays(invoices),
		},
		Filters: InvoiceFiltersVM{
			StatusOptions: InvoiceStatusOptions(),
		},
		Actions: []vm.ActionButton{
			{Label: "New Invoice", Action: "invoice.create", Icon: "plus", Variant: "primary", Enabled: true},
			{Label: "Export", Action: "invoice.export", Icon: "download", Variant: "secondary", Enabled: len(invoices) > 0},
		},
	}
}

// BuildInvoiceDetailVM constructs a single invoice detail ViewModel.
func BuildInvoiceDetailVM(invoice domain.Invoice, payments []domain.Payment) InvoiceDetailVM {
	items := make([]InvoiceItemVM, 0, len(invoice.Items))
	for _, item := range invoice.Items {
		items = append(items, InvoiceItemVM{
			ID:           item.ID,
			LineNumber:   item.LineNumber,
			Description:  item.Description,
			Quantity:     FormatQuantity(item.Quantity),
			RateDisplay:  FormatMoney(item.Rate, "BHD"),
			TotalDisplay: FormatMoney(item.TotalBHD, "BHD"),
			ProductCode:  text.FirstNonEmpty(item.ProductCode, item.Model),
		})
	}

	history := make([]PaymentRowVM, 0, len(payments))
	for _, payment := range payments {
		history = append(history, PaymentRowVM{
			ID:            payment.ID,
			PaymentDate:   FormatDate(payment.PaymentDate),
			AmountDisplay: FormatMoney(payment.AmountBHD, "BHD"),
			Method:        text.FirstNonEmpty(payment.PaymentMethod, "Other"),
			Reference:     payment.Reference,
			DaysToPayment: payment.DaysToPayment,
		})
	}

	return InvoiceDetailVM{
		ID:              invoice.ID,
		InvoiceNumber:   invoice.InvoiceNumber,
		CustomerName:    invoice.CustomerName,
		InvoiceDate:     FormatDate(invoice.InvoiceDate),
		DueDate:         FormatDate(invoice.DueDate),
		Status:          InvoiceStatusBadge(invoice.Status),
		Items:           items,
		SubtotalDisplay: FormatMoney(invoice.SubtotalBHD, "BHD"),
		VATDisplay:      FormatMoney(invoice.VATBHD, "BHD"),
		TotalDisplay:    FormatMoney(invoice.GrandTotalBHD, "BHD"),
		PaymentHistory:  history,
		Actions: []vm.ActionButton{
			{Label: "Generate PDF", Action: "invoice.generatePdf", Icon: "file-text", Variant: "secondary", Enabled: true},
			{Label: "Record Payment", Action: "invoice.recordPayment", Icon: "credit-card", Variant: "primary", Enabled: invoice.OutstandingBHD > 0},
			{Label: "Void", Action: "invoice.void", Icon: "ban", Variant: "danger", Enabled: invoice.Status != "Paid" && invoice.Status != "Void"},
		},
		Breadcrumbs: []vm.BreadcrumbItem{
			{Label: "Finance", Path: "/finance"},
			{Label: "Invoices", Path: "/finance/invoices"},
			{Label: invoice.InvoiceNumber},
		},
	}
}

// BuildCashPositionVM constructs a cash position widget from account balances.
func BuildCashPositionVM(accounts []domain.CompanyBankAccount, balances map[string]float64) CashPositionVM {
	rows := make([]AccountBalanceVM, 0, len(accounts))
	totalBHD := 0.0
	for _, account := range accounts {
		balance := balances[account.ID]
		if account.Currency == "" || strings.EqualFold(account.Currency, "BHD") {
			totalBHD += balance
		}
		status := "Inactive"
		if account.IsActive {
			status = "Active"
		}
		rows = append(rows, AccountBalanceVM{
			ID:             account.ID,
			BankName:       account.BankName,
			AccountName:    account.AccountName,
			Currency:       text.FirstNonEmpty(account.Currency, "BHD"),
			BalanceDisplay: FormatMoney(balance, text.FirstNonEmpty(account.Currency, "BHD")),
			Status:         status,
		})
	}
	return CashPositionVM{
		TotalCashDisplay: FormatMoney(totalBHD, "BHD"),
		Accounts:         rows,
		Trend:            "stable",
	}
}

// InvoiceStatusBadge maps persisted invoice statuses to display badges.
func InvoiceStatusBadge(status string) shared.StatusBadgeVM {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "paid":
		return shared.StatusBadgeVM{Label: "Paid", Color: "green", Icon: "check-circle"}
	case "partiallypaid", "partially paid":
		return shared.StatusBadgeVM{Label: "Partially Paid", Color: "blue", Icon: "circle-dollar-sign"}
	case "overdue":
		return shared.StatusBadgeVM{Label: "Overdue", Color: "red", Icon: "alert-triangle"}
	case "sent":
		return shared.StatusBadgeVM{Label: "Sent", Color: "amber", Icon: "send"}
	case "cancelled", "void":
		return shared.StatusBadgeVM{Label: strings.Title(strings.ToLower(status)), Color: "gray", Icon: "ban"}
	case "proforma":
		return shared.StatusBadgeVM{Label: "Proforma", Color: "blue", Icon: "file-text"}
	default:
		return shared.StatusBadgeVM{Label: text.FirstNonEmpty(status, "Draft"), Color: "gray", Icon: "edit"}
	}
}

// InvoiceStatusOptions returns the standard invoice status filter options.
func InvoiceStatusOptions() []vm.Option {
	return []vm.Option{
		{Value: "Sent", Label: "Sent"},
		{Value: "Overdue", Label: "Overdue"},
		{Value: "PartiallyPaid", Label: "Partially Paid"},
		{Value: "Paid", Label: "Paid"},
		{Value: "Draft", Label: "Draft"},
	}
}

// FormatMoney formats monetary values for display.
func FormatMoney(amount float64, currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		currency = "BHD"
	}
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = math.Abs(amount)
	}
	return fmt.Sprintf("%s%s %s", sign, currency, formatThousands(amount))
}

// FormatDate formats dates for human display.
func FormatDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2 Jan 2006")
}

// FormatQuantity formats quantities without noisy decimals.
func FormatQuantity(v float64) string {
	if math.Abs(v-math.Round(v)) < 0.000001 {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.2f", v)
}

func averagePaymentDays(invoices []domain.Invoice) int {
	total := 0
	count := 0
	for _, invoice := range invoices {
		if strings.EqualFold(invoice.Status, "Paid") && !invoice.InvoiceDate.IsZero() && !invoice.DueDate.IsZero() {
			days := int(invoice.DueDate.Sub(invoice.InvoiceDate).Hours() / 24)
			if days >= 0 {
				total += days
				count++
			}
		}
	}
	if count == 0 {
		return 0
	}
	return int(math.Round(float64(total) / float64(count)))
}

func formatThousands(amount float64) string {
	raw := fmt.Sprintf("%.2f", amount)
	parts := strings.Split(raw, ".")
	intPart := parts[0]
	var out []byte
	for i, r := range reverse(intPart) {
		if i > 0 && i%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, byte(r))
	}
	formatted := string(reverse(string(out)))
	if len(parts) == 2 {
		return formatted + "." + parts[1]
	}
	return formatted
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
