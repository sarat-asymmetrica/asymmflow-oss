package compliance

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"ph_holdings_app/pkg/infra/events"
)

// ValidationEntry records one compliance validation outcome.
type ValidationEntry struct {
	Timestamp    time.Time    `json:"timestamp"`
	EventName    string       `json:"event_name"`
	Jurisdiction Jurisdiction `json:"jurisdiction"`
	Valid        bool         `json:"valid"`
	Errors       []string     `json:"errors"`
	Warnings     []string     `json:"warnings"`
}

// ComplianceHook listens for finance events and validates compliance.
type ComplianceHook struct {
	registry *Registry
	bus      events.Bus

	mu          sync.Mutex
	validations []ValidationEntry
}

// NewComplianceHook creates and registers compliance event listeners.
func NewComplianceHook(registry *Registry, bus events.Bus) *ComplianceHook {
	hook := &ComplianceHook{registry: registry, bus: bus}
	if bus != nil {
		for _, eventName := range []string{
			events.EventInvoiceCreated,
			events.EventInvoiceUpdated,
			events.EventPaymentRecorded,
			events.EventExpenseCreated,
			events.EventCreditNoteIssued,
		} {
			bus.Subscribe(eventName, hook.handleEvent)
		}
	}
	return hook
}

// OnInvoiceCreated validates invoice tax compliance.
func (h *ComplianceHook) OnInvoiceCreated(data map[string]any) error {
	return h.validate(events.EventInvoiceCreated, data)
}

// validate runs the jurisdiction engine over one event's tax data. Credit
// notes validate under the same rate arithmetic as invoices (amounts are
// positive magnitudes; the event type conveys direction), so every carrier
// event funnels here — with its OWN event name on the record.
func (h *ComplianceHook) validate(eventName string, data map[string]any) error {
	if h == nil || h.registry == nil {
		return nil
	}

	invoice := invoiceDataFromMap(data)
	jurisdiction := jurisdictionFromData(data, invoice.Currency)
	engine, ok := h.registry.Get(jurisdiction)
	if !ok {
		h.record(ValidationEntry{
			Timestamp:    time.Now(),
			EventName:    eventName,
			Jurisdiction: jurisdiction,
			Valid:        true,
			Warnings:     []string{"no compliance engine registered for jurisdiction"},
		})
		return nil
	}

	result, err := engine.ValidateInvoice(invoice)
	if err != nil {
		h.record(ValidationEntry{
			Timestamp:    time.Now(),
			EventName:    eventName,
			Jurisdiction: jurisdiction,
			Valid:        false,
			Errors:       []string{err.Error()},
		})
		log.Printf("compliance validation error: jurisdiction=%s error=%v", jurisdiction, err)
		return nil
	}

	entry := ValidationEntry{
		Timestamp:    time.Now(),
		EventName:    eventName,
		Jurisdiction: jurisdiction,
		Valid:        result.Valid,
		Errors:       append([]string(nil), result.Errors...),
		Warnings:     append([]string(nil), result.Warnings...),
	}
	h.record(entry)
	if !entry.Valid || len(entry.Warnings) > 0 {
		log.Printf("compliance validation result: jurisdiction=%s valid=%v errors=%v warnings=%v", jurisdiction, entry.Valid, entry.Errors, entry.Warnings)
	}
	return nil
}

// RecentValidations returns the newest validation entries up to limit.
func (h *ComplianceHook) RecentValidations(limit int) []ValidationEntry {
	if h == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if limit <= 0 || limit > len(h.validations) {
		limit = len(h.validations)
	}
	start := len(h.validations) - limit
	out := append([]ValidationEntry(nil), h.validations[start:]...)
	return out
}

func (h *ComplianceHook) handleEvent(ctx context.Context, event events.Event) error {
	if carrier, ok := event.(interface{ ComplianceData() map[string]any }); ok {
		data := carrier.ComplianceData()
		eventName := event.Name()
		go func() {
			_ = h.validate(eventName, data)
		}()
	}
	return nil
}

func (h *ComplianceHook) record(entry ValidationEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.validations = append(h.validations, entry)
	if len(h.validations) > 100 {
		h.validations = append([]ValidationEntry(nil), h.validations[len(h.validations)-100:]...)
	}
}

func invoiceDataFromMap(data map[string]any) InvoiceData {
	if data == nil {
		return InvoiceData{}
	}
	return InvoiceData{
		InvoiceNumber: stringValue(data["invoice_number"]),
		InvoiceDate:   timeValue(data["invoice_date"]),
		SellerTaxID:   stringValue(data["seller_tax_id"]),
		BuyerTaxID:    stringValue(data["buyer_tax_id"]),
		Amount:        floatValue(data["amount"]),
		TaxAmount:     floatValue(data["tax_amount"]),
		Currency:      stringValue(data["currency"]),
	}
}

func jurisdictionFromData(data map[string]any, currency string) Jurisdiction {
	if data != nil {
		if value := strings.ToUpper(strings.TrimSpace(stringValue(data["jurisdiction"]))); value != "" {
			return Jurisdiction(value)
		}
	}
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "INR":
		return JurisdictionIndia
	case "SAR":
		return JurisdictionSaudi
	default:
		return JurisdictionBahrain
	}
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func floatValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return 0
	}
}

func timeValue(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		parsed, _ := time.Parse(time.RFC3339, v)
		return parsed
	default:
		return time.Time{}
	}
}
