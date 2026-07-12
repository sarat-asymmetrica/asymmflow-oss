package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

type customerLinkIndex struct {
	customers     []CustomerMaster
	byCanonicalID map[string]CustomerMaster
	byID          map[string]CustomerMaster
	nameToID      map[string]string
}

type customerCommercialEvent struct {
	CustomerID string
	OfferID    string
	Ref        string
	Project    string
	Status     string
	Source     string
	Date       time.Time
	Value      float64
}

type customerCommercialEventCollector struct {
	seen   map[string]bool
	events []customerCommercialEvent
}

func newCustomerCommercialEventCollector() *customerCommercialEventCollector {
	return &customerCommercialEventCollector{
		seen:   make(map[string]bool),
		events: []customerCommercialEvent{},
	}
}

func (c *customerCommercialEventCollector) add(event customerCommercialEvent, aliases ...string) bool {
	if c == nil || strings.TrimSpace(event.CustomerID) == "" {
		return false
	}
	allAliases := make([]string, 0, len(aliases)+1)
	for _, alias := range aliases {
		if normalized := commercialEventAlias(alias); normalized != "" {
			allAliases = append(allAliases, normalized)
		}
	}
	if fingerprint := commercialEventFingerprint(event); fingerprint != "" {
		allAliases = append(allAliases, fingerprint)
	}
	for _, alias := range allAliases {
		if c.seen[alias] {
			return false
		}
	}
	for _, alias := range allAliases {
		c.seen[alias] = true
	}
	c.events = append(c.events, event)
	return true
}

func commercialEventAlias(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	return value
}

func commercialEventFingerprint(event customerCommercialEvent) string {
	if strings.TrimSpace(event.CustomerID) == "" || event.Date.IsZero() || event.Value <= 0 {
		return ""
	}
	return strings.Join([]string{
		"fingerprint",
		event.CustomerID,
		event.Date.Format("2006-01-02"),
		fmt.Sprintf("%.3f", roundTo3(event.Value)),
		strings.ToLower(strings.TrimSpace(event.Status)),
	}, "|")
}

func zeroPipelineEventAlias(customerID, project, status string) string {
	projectKey := normalizeCustomerName(project)
	if strings.TrimSpace(customerID) == "" || projectKey == "" {
		return ""
	}
	return "zero_pipeline:" + customerID + "|" + projectKey + "|" + strings.ToLower(strings.TrimSpace(status))
}

func buildCustomerPipelineEvents(customerID string, rfqs []RFQData, offers []Offer, opportunities []Opportunity, includeZero bool) []customerCommercialEvent {
	collector := newCustomerCommercialEventCollector()
	for _, opp := range opportunities {
		if !includeZero && opp.RevenueBHD <= 0 {
			continue
		}
		eventDate := opp.OfferDate
		if eventDate.IsZero() {
			eventDate = opp.CreatedAt
		}
		if eventDate.IsZero() && opp.Year > 0 {
			eventDate = time.Date(opp.Year, time.January, 1, 0, 0, 0, 0, time.UTC)
		}
		event := customerCommercialEvent{
			CustomerID: customerID,
			OfferID:    opp.OfferID,
			Ref:        opp.FolderNumber,
			Project:    firstNonEmpty(opp.Title, opp.FolderName, opp.FolderNumber),
			Status:     opp.Stage,
			Source:     "opportunity",
			Date:       eventDate,
			Value:      opp.RevenueBHD,
		}
		collector.add(event,
			"opportunity:"+opp.ID,
			"offer:"+opp.OfferID,
			"folder:"+opp.FolderNumber,
			zeroPipelineEventAlias(customerID, event.Project, event.Status),
		)
	}
	for _, offer := range offers {
		if !includeZero && offer.TotalValueBHD <= 0 {
			continue
		}
		event := customerCommercialEvent{
			CustomerID: customerID,
			OfferID:    offer.ID,
			Ref:        offer.OfferNumber,
			Project:    firstNonEmpty(offer.ProjectName, offer.CustomerReference, offer.OfferNumber),
			Status:     offer.Stage,
			Source:     "offer",
			Date:       offer.QuotationDate,
			Value:      offer.TotalValueBHD,
		}
		collector.add(event,
			"offer:"+offer.ID,
			"offer_number:"+offer.OfferNumber,
			"rfq:"+offer.RFQID,
			zeroPipelineEventAlias(customerID, event.Project, event.Status),
		)
	}
	for _, rfq := range rfqs {
		if !includeZero && rfq.Value <= 0 {
			continue
		}
		event := customerCommercialEvent{
			CustomerID: customerID,
			Ref:        rfq.RFQNumber,
			Project:    firstNonEmpty(rfq.Project, rfq.RFQNumber),
			Status:     rfq.Status,
			Source:     "rfq",
			Date:       rfq.CreatedAt,
			Value:      rfq.Value,
		}
		collector.add(event,
			fmt.Sprintf("rfq:%d", rfq.ID),
			"rfq_number:"+rfq.RFQNumber,
			zeroPipelineEventAlias(customerID, event.Project, event.Status),
		)
	}
	sort.Slice(collector.events, func(i, j int) bool {
		if collector.events[i].Date.Equal(collector.events[j].Date) {
			return collector.events[i].Value > collector.events[j].Value
		}
		return collector.events[i].Date.After(collector.events[j].Date)
	})
	return collector.events
}

func (a *App) buildCustomerLinkIndex() *customerLinkIndex {
	idx := &customerLinkIndex{
		byCanonicalID: make(map[string]CustomerMaster),
		byID:          make(map[string]CustomerMaster),
		nameToID:      make(map[string]string),
	}
	if a == nil || a.db == nil {
		return idx
	}

	if err := a.db.Find(&idx.customers).Error; err != nil {
		log.Printf("⚠️ Customer linkage: failed to load customers: %v", err)
		return idx
	}
	for _, customer := range idx.customers {
		idx.addCustomer(customer)
	}

	var mappings []CustomerNameMapping
	if err := a.db.Find(&mappings).Error; err != nil {
		// Some focused test databases do not migrate this optional table.
		return idx
	}
	for _, mapping := range mappings {
		customer, ok := idx.resolve(mapping.CustomerID, mapping.CanonicalName)
		if !ok {
			continue
		}
		idx.addID(mapping.CustomerID, customer)
		idx.addName(mapping.ExtractedName, customer)
		idx.addName(mapping.CanonicalName, customer)
	}

	return idx
}

func (idx *customerLinkIndex) addCustomer(customer CustomerMaster) {
	if idx == nil || strings.TrimSpace(customer.ID) == "" {
		return
	}
	idx.byCanonicalID[customer.ID] = customer
	idx.addID(customer.ID, customer)
	idx.addID(customer.CustomerID, customer)
	idx.addID(customer.CustomerCode, customer)
	idx.addName(customer.BusinessName, customer)
	idx.addName(customer.TradingName, customer)
	if len(strings.TrimSpace(customer.ShortCode)) >= 3 {
		idx.addName(customer.ShortCode, customer)
	}
}

func (idx *customerLinkIndex) addID(raw string, customer CustomerMaster) {
	key := customerIDLookupKey(raw)
	if idx == nil || key == "" || strings.TrimSpace(customer.ID) == "" {
		return
	}
	if existing, ok := idx.byID[key]; ok && existing.ID != customer.ID {
		return
	}
	idx.byID[key] = customer
}

func (idx *customerLinkIndex) addName(raw string, customer CustomerMaster) {
	key := customerNameLookupKey(raw)
	if idx == nil || key == "" || strings.TrimSpace(customer.ID) == "" {
		return
	}
	if existingID, ok := idx.nameToID[key]; ok && existingID != customer.ID {
		return
	}
	idx.nameToID[key] = customer.ID
}

func (idx *customerLinkIndex) resolve(idHint, nameHint string) (CustomerMaster, bool) {
	if idx == nil {
		return CustomerMaster{}, false
	}
	for _, raw := range []string{idHint, nameHint} {
		if customer, ok := idx.byID[customerIDLookupKey(raw)]; ok {
			return customer, true
		}
	}
	for _, raw := range []string{nameHint, idHint} {
		if canonicalID, ok := idx.nameToID[customerNameLookupKey(raw)]; ok {
			if customer, found := idx.byCanonicalID[canonicalID]; found {
				return customer, true
			}
		}
	}
	return CustomerMaster{}, false
}

func (idx *customerLinkIndex) matches(customer CustomerMaster, idHint, nameHint string) bool {
	resolved, ok := idx.resolve(idHint, nameHint)
	return ok && resolved.ID == customer.ID
}

func customerIDLookupKey(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	return "id:" + value
}

func customerNameLookupKey(raw string) string {
	value := normalizeCustomerName(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	return "name:" + value
}

func customerCommercialOrderValue(order Order) float64 {
	if order.GrandTotalBHD > 0 {
		return order.GrandTotalBHD
	}
	return order.TotalValueBHD
}

func invoicePostedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "sent", "paid", "partiallypaid", "partially paid", "overdue":
		return true
	default:
		return false
	}
}

func invoiceOpenStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "sent", "partiallypaid", "partially paid", "overdue":
		return true
	default:
		return false
	}
}

func invoiceOutstandingValue(inv Invoice) float64 {
	if inv.OutstandingBHD > 0 {
		return inv.OutstandingBHD
	}
	if invoiceOpenStatus(inv.Status) {
		return inv.GrandTotalBHD
	}
	return 0
}

func commercialOrderStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "cancelled", "canceled", "void", "draft":
		return false
	default:
		return true
	}
}

func commercialOfferStage(stage string) bool {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "rfq", "quoted", "proposal", "qualified", "won":
		return true
	default:
		return false
	}
}

func closedWonStage(stage string) bool {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "won", "execution", "order received", "converted":
		return true
	default:
		return false
	}
}

func closedLostStage(stage string) bool {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "lost", "expired", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func sortOpportunitySummariesByCreatedAt(rows []OpportunitySummary) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
}

func sortOrderSummariesByDate(rows []OrderSummary) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].OrderDate.After(rows[j].OrderDate)
	})
}

func sortInvoiceSummariesByDate(rows []InvoiceSummary) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].InvoiceDate.After(rows[j].InvoiceDate)
	})
}

func daysOverdueFrom(now, dueDate time.Time) int {
	if dueDate.IsZero() {
		return 0
	}
	return int(now.Sub(dueDate).Hours() / 24)
}

func (a *App) linkedInvoicesForCustomer(idx *customerLinkIndex, customer CustomerMaster) []Invoice {
	if a == nil || a.db == nil {
		return nil
	}
	var invoices []Invoice
	if err := a.db.Find(&invoices).Error; err != nil {
		return nil
	}
	linked := make([]Invoice, 0, len(invoices))
	for _, inv := range invoices {
		if idx.matches(customer, inv.CustomerID, inv.CustomerName) {
			linked = append(linked, inv)
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		return linked[i].InvoiceDate.After(linked[j].InvoiceDate)
	})
	return linked
}

func (a *App) linkedOrdersForCustomer(idx *customerLinkIndex, customer CustomerMaster) []Order {
	if a == nil || a.db == nil {
		return nil
	}
	var orders []Order
	if err := a.db.Find(&orders).Error; err != nil {
		return nil
	}
	linked := make([]Order, 0, len(orders))
	for _, order := range orders {
		if idx.matches(customer, order.CustomerID, order.CustomerName) {
			linked = append(linked, order)
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		return linked[i].OrderDate.After(linked[j].OrderDate)
	})
	return linked
}

func (a *App) linkedOffersForCustomer(idx *customerLinkIndex, customer CustomerMaster) []Offer {
	if a == nil || a.db == nil {
		return nil
	}
	var offers []Offer
	if err := a.db.Find(&offers).Error; err != nil {
		return nil
	}
	linked := make([]Offer, 0, len(offers))
	for _, offer := range offers {
		if idx.matches(customer, offer.CustomerID, offer.CustomerName) {
			linked = append(linked, offer)
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		return linked[i].QuotationDate.After(linked[j].QuotationDate)
	})
	return linked
}

func (a *App) linkedRFQsForCustomer(idx *customerLinkIndex, customer CustomerMaster) []RFQData {
	if a == nil || a.db == nil {
		return nil
	}
	var rfqs []RFQData
	if err := a.db.Find(&rfqs).Error; err != nil {
		return nil
	}
	linked := make([]RFQData, 0, len(rfqs))
	for _, rfq := range rfqs {
		if idx.matches(customer, "", rfq.Client) {
			linked = append(linked, rfq)
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		return linked[i].CreatedAt.After(linked[j].CreatedAt)
	})
	return linked
}

func (a *App) linkedOpportunitiesForCustomer(idx *customerLinkIndex, customer CustomerMaster) []Opportunity {
	if a == nil || a.db == nil {
		return nil
	}
	var opportunities []Opportunity
	if err := a.db.Find(&opportunities).Error; err != nil {
		return nil
	}
	linked := make([]Opportunity, 0, len(opportunities))
	for _, opp := range opportunities {
		if idx.matches(customer, opp.CustomerID, opp.CustomerName) {
			linked = append(linked, opp)
		}
	}
	sort.Slice(linked, func(i, j int) bool {
		left := linked[i].OfferDate
		right := linked[j].OfferDate
		if left.IsZero() {
			left = linked[i].CreatedAt
		}
		if right.IsZero() {
			right = linked[j].CreatedAt
		}
		return left.After(right)
	})
	return linked
}

func linkedInvoiceAmountByOrder(invoices []Invoice) map[string]float64 {
	amounts := make(map[string]float64)
	for _, inv := range invoices {
		if !invoicePostedStatus(inv.Status) || strings.TrimSpace(inv.OrderID) == "" {
			continue
		}
		amounts[inv.OrderID] += inv.GrandTotalBHD
	}
	return amounts
}

func linkedOrderExposure(orders []Order, invoices []Invoice) float64 {
	invoicedByOrder := linkedInvoiceAmountByOrder(invoices)
	var exposure float64
	for _, order := range orders {
		if !commercialOrderStatus(order.Status) {
			continue
		}
		remainder := customerCommercialOrderValue(order) - invoicedByOrder[order.ID]
		if remainder > 0.001 {
			exposure += remainder
		}
	}
	return roundTo3(exposure)
}

func linkedReceivablesAging(invoices []Invoice, orderExposure float64) ReceivablesAgingSummary {
	now := time.Now()
	aging := ReceivablesAgingSummary{}
	for _, inv := range invoices {
		if !invoiceOpenStatus(inv.Status) {
			continue
		}
		outstanding := invoiceOutstandingValue(inv)
		if outstanding <= 0 {
			continue
		}
		daysOverdue := daysOverdueFrom(now, inv.DueDate)
		switch {
		case daysOverdue <= 30:
			aging.Current += outstanding
		case daysOverdue <= 60:
			aging.Days30_60 += outstanding
		case daysOverdue <= 90:
			aging.Days60_90 += outstanding
		case daysOverdue <= 120:
			aging.Days90_120 += outstanding
		default:
			aging.Days120Plus += outstanding
		}
		aging.TotalOutstanding += outstanding
	}
	if orderExposure > 0 {
		aging.Current += orderExposure
		aging.TotalOutstanding += orderExposure
	}
	aging.Current = roundTo3(aging.Current)
	aging.Days30_60 = roundTo3(aging.Days30_60)
	aging.Days60_90 = roundTo3(aging.Days60_90)
	aging.Days90_120 = roundTo3(aging.Days90_120)
	aging.Days120Plus = roundTo3(aging.Days120Plus)
	aging.TotalOutstanding = roundTo3(aging.TotalOutstanding)
	return aging
}

func (a *App) linkedPaymentHistoryForCustomer(idx *customerLinkIndex, customer CustomerMaster, limit int) []PaymentHistoryEntry {
	if a == nil || a.db == nil || limit == 0 {
		return []PaymentHistoryEntry{}
	}
	if limit < 0 {
		limit = 0
	}
	invoices := a.linkedInvoicesForCustomer(idx, customer)
	invoiceByID := make(map[string]Invoice, len(invoices))
	invoiceByNumber := make(map[string]Invoice, len(invoices))
	for _, inv := range invoices {
		if strings.TrimSpace(inv.ID) != "" {
			invoiceByID[inv.ID] = inv
		}
		if strings.TrimSpace(inv.InvoiceNumber) != "" {
			invoiceByNumber[strings.ToLower(strings.TrimSpace(inv.InvoiceNumber))] = inv
		}
	}

	var payments []Payment
	if err := a.db.Order("payment_date DESC").Find(&payments).Error; err != nil {
		return []PaymentHistoryEntry{}
	}
	history := make([]PaymentHistoryEntry, 0, limit)
	for _, payment := range payments {
		invoice, ok := invoiceByID[payment.InvoiceID]
		if !ok {
			invoice, ok = invoiceByNumber[strings.ToLower(strings.TrimSpace(payment.InvoiceNumber))]
		}
		if !ok {
			continue
		}
		daysToPayment := payment.DaysToPayment
		if daysToPayment == 0 && !invoice.InvoiceDate.IsZero() && !payment.PaymentDate.IsZero() {
			daysToPayment = int(payment.PaymentDate.Sub(invoice.InvoiceDate).Hours() / 24)
			if daysToPayment < 0 {
				daysToPayment = 0
			}
		}
		history = append(history, PaymentHistoryEntry{
			PaymentDate:   payment.PaymentDate,
			AmountBHD:     payment.AmountBHD,
			InvoiceNumber: payment.InvoiceNumber,
			DaysToPayment: daysToPayment,
			PaymentMethod: payment.PaymentMethod,
		})
		if limit > 0 && len(history) >= limit {
			break
		}
	}
	return history
}
