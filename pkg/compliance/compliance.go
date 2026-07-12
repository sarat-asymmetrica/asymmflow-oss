package compliance

import (
	"encoding/json"
	"sort"
	"time"
)

// Jurisdiction represents a tax/regulatory jurisdiction.
type Jurisdiction string

const (
	JurisdictionBahrain Jurisdiction = "BH"
	JurisdictionIndia   Jurisdiction = "IN"
	JurisdictionSaudi   Jurisdiction = "SA"
)

// TaxEngine is the interface all compliance modules implement.
type TaxEngine interface {
	Jurisdiction() Jurisdiction
	CalculateTax(tx TaxableTransaction) (*TaxResult, error)
	ValidateInvoice(inv InvoiceData) (*ValidationResult, error)
	TaxRates() []TaxRate
	Name() string
}

// TaxableTransaction represents a transaction that may be taxed.
type TaxableTransaction struct {
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	Date          time.Time `json:"date"`
	Category      string    `json:"category"`
	CustomerType  string    `json:"customer_type"`
	SupplierType  string    `json:"supplier_type"`
	HSNCode       string    `json:"hsn_code"`
	PlaceOfSupply string    `json:"place_of_supply"`
}

// TaxResult holds computed tax amounts.
type TaxResult struct {
	BaseAmount   float64        `json:"base_amount"`
	TaxAmount    float64        `json:"tax_amount"`
	TotalAmount  float64        `json:"total_amount"`
	TaxBreakdown []TaxComponent `json:"tax_breakdown"`
	Jurisdiction Jurisdiction   `json:"jurisdiction"`
}

// TaxComponent is one line of tax.
type TaxComponent struct {
	Name   string  `json:"name"`
	Rate   float64 `json:"rate"`
	Amount float64 `json:"amount"`
}

// TaxRate represents a configured tax rate.
type TaxRate struct {
	Name        string  `json:"name"`
	Rate        float64 `json:"rate"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
}

// InvoiceData holds invoice fields for validation.
type InvoiceData struct {
	InvoiceNumber string         `json:"invoice_number"`
	InvoiceDate   time.Time      `json:"invoice_date"`
	SellerTaxID   string         `json:"seller_tax_id"`
	BuyerTaxID    string         `json:"buyer_tax_id"`
	Amount        float64        `json:"amount"`
	TaxAmount     float64        `json:"tax_amount"`
	Currency      string         `json:"currency"`
	LineItems     []LineItemData `json:"line_items"`
}

// LineItemData holds one invoice line item.
type LineItemData struct {
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TaxRate     float64 `json:"tax_rate"`
	HSNCode     string  `json:"hsn_code"`
}

// ValidationResult holds invoice validation outcome.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// Registry holds all registered tax engines.
type Registry struct {
	engines map[Jurisdiction]TaxEngine
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{engines: make(map[Jurisdiction]TaxEngine)}
}

// Register adds a tax engine.
func (r *Registry) Register(engine TaxEngine) {
	if r == nil || engine == nil {
		return
	}
	if r.engines == nil {
		r.engines = make(map[Jurisdiction]TaxEngine)
	}
	r.engines[engine.Jurisdiction()] = engine
}

// Get returns the engine for a jurisdiction.
func (r *Registry) Get(j Jurisdiction) (TaxEngine, bool) {
	if r == nil || r.engines == nil {
		return nil, false
	}
	engine, ok := r.engines[j]
	return engine, ok
}

// All returns all registered engines.
func (r *Registry) All() []TaxEngine {
	if r == nil || r.engines == nil {
		return nil
	}
	engines := make([]TaxEngine, 0, len(r.engines))
	for _, engine := range r.engines {
		engines = append(engines, engine)
	}
	sort.Slice(engines, func(i, j int) bool {
		return engines[i].Jurisdiction() < engines[j].Jurisdiction()
	})
	return engines
}

// MarshalJSON keeps Registry serializable for diagnostics and tests.
func (r *Registry) MarshalJSON() ([]byte, error) {
	type engineInfo struct {
		Jurisdiction Jurisdiction `json:"jurisdiction"`
		Name         string       `json:"name"`
	}
	engines := r.All()
	out := make([]engineInfo, 0, len(engines))
	for _, engine := range engines {
		out = append(out, engineInfo{Jurisdiction: engine.Jurisdiction(), Name: engine.Name()})
	}
	return json.Marshal(out)
}
