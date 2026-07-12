// ═══════════════════════════════════════════════════════════════════════════
// RHINE INSTRUMENTS PARSER - Rhine Instruments XML Pricing Parser
//
// MISSION: Parse Rhine basket XML files and extract pricing for costing
//
// FEATURES:
//   - Full XML structure parsing (basket, items, pricing)
//   - EUR to BHD currency conversion
//   - Product type classification (Flow, Level, Pressure, etc.)
//   - Batch processing for multiple files
//   - Statistics by product type
//
// Built with MATHEMATICAL RIGOR × PRODUCTION ROBUSTNESS
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"ph_holdings_app/pkg/overlay"
)

// EHBasket represents the root Rhine XML structure
type EHBasket struct {
	XMLName        xml.Name `xml:"basket"`
	PositionsCount int      `xml:"positionsCount,attr"`
	Header         EHHeader `xml:"header"`
	Items          []EHItem `xml:"item"`
}

// EHHeader contains customer and pricing totals
type EHHeader struct {
	Customer EHCustomer `xml:"customer"`
	Pricing  EHPricing  `xml:"pricing"`
}

// EHCustomer represents the customer details
type EHCustomer struct {
	Number string `xml:"number"`
	Name   string `xml:"name"`
}

// EHPricing contains total pricing information
type EHPricing struct {
	TotalsSales EHTotalsSales `xml:"totalsSales"`
}

// EHTotalsSales represents sales totals
type EHTotalsSales struct {
	NetValue   EHCurrency `xml:"netValue"`
	Freight    EHCurrency `xml:"freight"`
	GrossValue EHCurrency `xml:"grossValue"`
}

// EHCurrency represents a currency value
type EHCurrency struct {
	Value    float64 `xml:",chardata"`
	Currency string  `xml:"currency,attr"`
}

// EHItem represents a single basket item
type EHItem struct {
	Product     EHProduct     `xml:"product"`
	ItemPricing EHItemPricing `xml:"itemPricing"`
	Delivery    EHDelivery    `xml:"delivery"`
}

// EHProduct contains product details
type EHProduct struct {
	OrderCode     string     `xml:"orderCode"`
	OrderCodeLong string     `xml:"orderCodeLong"`
	Quantity      EHQuantity `xml:"quantity"`
	Texts         EHTexts    `xml:"texts"`
}

// EHQuantity represents a quantity with unit
type EHQuantity struct {
	Value int    `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
}

// EHTexts contains product descriptions
type EHTexts struct {
	ShortDescription string `xml:"shortDescription"`
}

// EHItemPricing contains item-level pricing
type EHItemPricing struct {
	ItemListPrice  EHCurrency `xml:"itemListPrice"`
	ItemDiscount   EHCurrency `xml:"itemDiscount"`
	UnitSalesPrice EHCurrency `xml:"unitSalesPrice"`
	ItemSalesPrice EHCurrency `xml:"itemSalesPrice"`
}

// EHDelivery contains delivery information
type EHDelivery struct {
	ProductionTime int `xml:"productionTime"`
}

// ParsedEHItem represents a flattened, business-ready item
type ParsedEHItem struct {
	OrderCode         string  `json:"order_code"`
	OrderCodeLong     string  `json:"order_code_long"`
	Description       string  `json:"description"`
	Quantity          int     `json:"quantity"`
	QuantityUnit      string  `json:"quantity_unit"`
	ListPriceEUR      float64 `json:"list_price_eur"`
	DiscountEUR       float64 `json:"discount_eur"`
	UnitSalesPriceEUR float64 `json:"unit_sales_price_eur"`
	ItemSalesPriceEUR float64 `json:"item_sales_price_eur"`
	ListPriceBHD      float64 `json:"list_price_bhd"`
	DiscountBHD       float64 `json:"discount_bhd"`
	UnitSalesPriceBHD float64 `json:"unit_sales_price_bhd"`
	ItemSalesPriceBHD float64 `json:"item_sales_price_bhd"`
	ProductionDays    int     `json:"production_days"`
	ProductType       string  `json:"product_type"` // Flow, Level, etc.
}

// ParsedEHBasket represents the complete parsed basket
type ParsedEHBasket struct {
	CustomerNumber  string         `json:"customer_number"`
	CustomerName    string         `json:"customer_name"`
	TotalNetEUR     float64        `json:"total_net_eur"`
	TotalFreightEUR float64        `json:"total_freight_eur"`
	TotalGrossEUR   float64        `json:"total_gross_eur"`
	TotalNetBHD     float64        `json:"total_net_bhd"`
	TotalFreightBHD float64        `json:"total_freight_bhd"`
	TotalGrossBHD   float64        `json:"total_gross_bhd"`
	ItemCount       int            `json:"item_count"`
	Items           []ParsedEHItem `json:"items"`
	SourceFile      string         `json:"source_file"`
}

// ProductTypeStats represents aggregated stats per product type
type ProductTypeStats struct {
	ProductType      string  `json:"product_type"`
	ItemCount        int     `json:"item_count"`
	TotalValueEUR    float64 `json:"total_value_eur"`
	TotalValueBHD    float64 `json:"total_value_bhd"`
	TotalDiscountEUR float64 `json:"total_discount_eur"`
	TotalDiscountBHD float64 `json:"total_discount_bhd"`
}

// EHParser handles Rhine XML parsing
type EHParser struct {
	ConversionRate float64 // EUR to BHD
}

// NewEHParser creates a new Rhine Instruments parser. The EUR→BHD conversion
// rate is read from the active company overlay (the single source of truth for
// FX), so it always matches the live costing/posting paths. Callers may still
// override ConversionRate on the returned parser if needed.
func NewEHParser() *EHParser {
	return &EHParser{
		ConversionRate: overlay.Active().ExchangeRateToBase("EUR"),
	}
}

// ParseFile parses a single Rhine XML file
func (p *EHParser) ParseFile(filePath string) (*ParsedEHBasket, error) {
	xmlData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var basket EHBasket
	err = xml.Unmarshal(xmlData, &basket)
	if err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	parsed := &ParsedEHBasket{
		CustomerNumber:  basket.Header.Customer.Number,
		CustomerName:    basket.Header.Customer.Name,
		TotalNetEUR:     basket.Header.Pricing.TotalsSales.NetValue.Value,
		TotalFreightEUR: basket.Header.Pricing.TotalsSales.Freight.Value,
		TotalGrossEUR:   basket.Header.Pricing.TotalsSales.GrossValue.Value,
		TotalNetBHD:     basket.Header.Pricing.TotalsSales.NetValue.Value * p.ConversionRate,
		TotalFreightBHD: basket.Header.Pricing.TotalsSales.Freight.Value * p.ConversionRate,
		TotalGrossBHD:   basket.Header.Pricing.TotalsSales.GrossValue.Value * p.ConversionRate,
		ItemCount:       basket.PositionsCount,
		Items:           make([]ParsedEHItem, 0, len(basket.Items)),
		SourceFile:      filePath,
	}

	for _, item := range basket.Items {
		parsedItem := ParsedEHItem{
			OrderCode:         item.Product.OrderCode,
			OrderCodeLong:     item.Product.OrderCodeLong,
			Description:       item.Product.Texts.ShortDescription,
			Quantity:          item.Product.Quantity.Value,
			QuantityUnit:      item.Product.Quantity.Unit,
			ListPriceEUR:      item.ItemPricing.ItemListPrice.Value,
			DiscountEUR:       math.Abs(item.ItemPricing.ItemDiscount.Value),
			UnitSalesPriceEUR: item.ItemPricing.UnitSalesPrice.Value,
			ItemSalesPriceEUR: item.ItemPricing.ItemSalesPrice.Value,
			ProductionDays:    item.Delivery.ProductionTime,
		}

		parsedItem.ListPriceBHD = parsedItem.ListPriceEUR * p.ConversionRate
		parsedItem.DiscountBHD = parsedItem.DiscountEUR * p.ConversionRate
		parsedItem.UnitSalesPriceBHD = parsedItem.UnitSalesPriceEUR * p.ConversionRate
		parsedItem.ItemSalesPriceBHD = parsedItem.ItemSalesPriceEUR * p.ConversionRate
		parsedItem.ProductType = p.ClassifyProductType(item.Product.OrderCode, item.Product.Texts.ShortDescription)

		parsed.Items = append(parsed.Items, parsedItem)
	}

	return parsed, nil
}

// ClassifyProductType determines product category from order code and description
func (p *EHParser) ClassifyProductType(orderCode, description string) string {
	codeUpper := strings.ToUpper(orderCode)
	descUpper := strings.ToUpper(description)

	if strings.Contains(codeUpper, "CM442") || strings.Contains(descUpper, "LIQUILINE") {
		return "Rhine Flow"
	}
	if strings.Contains(codeUpper, "10W") || strings.Contains(codeUpper, "50W") ||
		strings.Contains(descUpper, "PROMAG") || strings.Contains(descUpper, "FLOWMETER") {
		return "Rhine Flow"
	}
	if strings.Contains(codeUpper, "FMU") && !strings.Contains(codeUpper, "FMU90") {
		return "Rhine Flow"
	}
	if strings.Contains(codeUpper, "FMU90") || strings.Contains(descUpper, "PROSONIC") {
		return "Rhine Level"
	}
	if strings.Contains(codeUpper, "FMR") || strings.Contains(descUpper, "MICROPILOT") {
		return "Rhine Level"
	}
	if strings.Contains(codeUpper, "PMC") || strings.Contains(codeUpper, "53P") ||
		strings.Contains(descUpper, "CERABAR") {
		return "Rhine Instruments Pressure"
	}
	if strings.Contains(codeUpper, "TMT") || strings.Contains(descUpper, "ITHERM") {
		return "Rhine Instruments Temperature"
	}
	if strings.Contains(descUpper, "MEMOSENS") || strings.Contains(descUpper, "ANALYTICS") {
		return "Rhine Analytics"
	}
	return "Rhine Instruments General"
}

// ParseBatch parses multiple Rhine XML files from a directory
func (p *EHParser) ParseBatch(dirPath string) ([]*ParsedEHBasket, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	results := make([]*ParsedEHBasket, 0)
	errors := make([]string, 0)

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			continue
		}

		fullPath := filepath.Join(dirPath, file.Name())
		parsed, err := p.ParseFile(fullPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", file.Name(), err))
			continue
		}
		results = append(results, parsed)
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "WARNING: Failed to parse %d files:\n", len(errors))
		for _, errMsg := range errors {
			fmt.Fprintf(os.Stderr, "  - %s\n", errMsg)
		}
	}

	return results, nil
}

// GetProductTypeStats returns statistics by product type
func (p *EHParser) GetProductTypeStats(basket *ParsedEHBasket) map[string]ProductTypeStats {
	stats := make(map[string]ProductTypeStats)

	for _, item := range basket.Items {
		stat, exists := stats[item.ProductType]
		if !exists {
			stat = ProductTypeStats{ProductType: item.ProductType}
		}
		stat.ItemCount++
		stat.TotalValueEUR += item.ItemSalesPriceEUR
		stat.TotalValueBHD += item.ItemSalesPriceBHD
		stat.TotalDiscountEUR += item.DiscountEUR
		stat.TotalDiscountBHD += item.DiscountBHD
		stats[item.ProductType] = stat
	}

	return stats
}
