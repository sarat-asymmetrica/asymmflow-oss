// ═══════════════════════════════════════════════════════════════════════════
// PDF GENERATOR - Multi-Language Invoice/Quote PDF Generation
//
// FEATURES:
//   - 9-language support via LangPack
//   - Arabic RTL text shaping
//   - Zone-based template positioning
//   - Letterhead overlay support
//   - BHD currency (3 decimal places)
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/signintech/gopdf"
	"ph_holdings_app/pkg/fonts"
	"ph_holdings_app/pkg/overlay"
)

// MM to PDF points conversion (1 mm = 2.83465 points at 72 DPI)
const mmToPoints = 2.83465

// InvoiceItem represents a single line item on the invoice
type InvoiceItem struct {
	SlNo          int     // Serial number
	Description   string  // Item description
	Quantity      float64 // Quantity
	Unit          string  // Unit (Each, Service, etc.)
	Rate          float64 // Rate per unit
	AnnualTotal   float64 // Annual total (for services)
	MonthlyAmount float64 // Monthly amount
	VATPercent    float64 // VAT percentage (usually 10%)
	TaxableValue  float64 // Taxable value in BHD
	VAT           float64 // VAT amount in BHD
	Total         float64 // Total including VAT
}

// InvoiceData represents complete invoice information
type InvoiceData struct {
	// Multi-language support
	Language string // Language code: "en", "ar", "zh-CN", etc.

	// Header information
	InvoiceType   string    // "TAX INVOICE", "QUOTATION", "PROFORMA INVOICE"
	InvoiceNumber string    // e.g., "INV-2025-0106"
	InvoiceDate   time.Time // Invoice date
	TRN           string    // Tax Registration Number: 200010357800002

	// Buyer information
	BuyerName      string // e.g., "Electricity and Water Authority"
	BuyerBuilding  string // Building number
	BuyerRoad      string // Road/Street
	BuyerTown      string // Town/City
	BuyerBlock     string // Block number
	BuyerCountry   string // Country (Kingdom of Bahrain)
	BuyerTRN       string // Buyer's TRN
	BuyerOrderNo   string // Purchase Order number
	BuyerOrderDate time.Time

	// Delivery information
	DeliveryNote     string    // Delivery note reference
	DeliveryNoteDate time.Time // Delivery note date
	PaymentTerms     string    // "30 Days", etc.
	ModeOfPayment    string    // Payment mode
	Destination      string    // Delivery destination
	TermsOfDelivery  string    // Delivery terms description

	// Line items
	Items []InvoiceItem

	// Totals
	Subtotal       float64 // Total excluding VAT
	TotalVAT       float64 // Total VAT amount
	GrandTotal     float64 // Total including VAT
	Currency       string  // "BHD" (Bahraini Dinars)
	CurrencySymbol string  // "B.D" or "BHD"

	// Additional fields
	SupplierRef     string // Supplier reference
	OtherReferences string // Other references
	DespatchDoc     string // Despatch document number
	DespatchMethod  string // How goods are despatched
	PlaceOfSupply   string // Place of supply (Kingdom of Bahrain)
}

// TemplateZoneConfig represents zone positioning from JSON config
type TemplateZoneConfig struct {
	Name        string                        `json:"name"`
	XMM         float64                       `json:"x_mm"`
	YMM         float64                       `json:"y_mm"`
	WidthMM     float64                       `json:"width_mm"`
	HeightMM    float64                       `json:"height_mm"`
	Purpose     string                        `json:"purpose"`
	Description string                        `json:"description"`
	Anchors     map[string]map[string]float64 `json:"anchors"`
	Columns     []ColumnConfig                `json:"columns,omitempty"`
}

// ColumnConfig for table columns
type ColumnConfig struct {
	Name    string  `json:"name"`
	XMM     float64 `json:"x_mm"`
	WidthMM float64 `json:"width_mm"`
}

// TemplateLayoutConfig is the root config structure
type TemplateLayoutConfig struct {
	TemplateName string `json:"template_name"`
	TemplateFile string `json:"template_file"`
	PageSize     struct {
		WidthMM  float64 `json:"width_mm"`
		HeightMM float64 `json:"height_mm"`
		Unit     string  `json:"unit"`
		DPI      int     `json:"dpi"`
	} `json:"page_size"`
	Zones         []TemplateZoneConfig `json:"zones"`
	ExcludedZones []TemplateZoneConfig `json:"excluded_zones"`
	Notes         []string             `json:"notes"`
}

// ContentBox represents the safe area for invoice content (normalized 0-1 coordinates)
type ContentBox struct {
	X      float64 // Left edge (0-1)
	Y      float64 // Bottom edge (0-1)
	Width  float64 // Width (0-1)
	Height float64 // Height (0-1)
}

// DefaultContentBox returns standard content box with margins
func DefaultContentBox() ContentBox {
	return ContentBox{
		X:      0.05, // 5% from left
		Y:      0.08, // 8% from bottom
		Width:  0.90, // 90% width
		Height: 0.84, // 84% height (leaves room for header/footer)
	}
}

// PDFGenerator handles PDF invoice generation with multi-language support
type PDFGenerator struct {
	pdf            *gopdf.GoPdf
	letterheadPath string
	fontPath       string
	langPack       *LangPack
	zoneConfig     *TemplateLayoutConfig
	contentBox     ContentBox
	arabicShaper   *ArabicShaper
	pageWidth      float64 // A4 = 595 points
	pageHeight     float64 // A4 = 842 points
}

// NewPDFGenerator creates a new PDF generator instance with multi-language support
func NewPDFGenerator(letterheadPath string) (*PDFGenerator, error) {
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4}) // A4 size: 595 x 842 points

	return &PDFGenerator{
		pdf:            pdf,
		letterheadPath: letterheadPath,
		langPack:       NewLangPack(),
		contentBox:     DefaultContentBox(),
		arabicShaper:   NewArabicShaper(),
		pageWidth:      595.0,
		pageHeight:     842.0,
	}, nil
}

// NewPDFGeneratorWithZones creates generator with zone-based positioning from JSON config
func NewPDFGeneratorWithZones(letterheadPath, zoneConfigPath string) (*PDFGenerator, error) {
	gen, err := NewPDFGenerator(letterheadPath)
	if err != nil {
		return nil, err
	}

	// Load zone configuration
	if zoneConfigPath != "" && fileExists(zoneConfigPath) {
		data, err := os.ReadFile(zoneConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read zone config: %v", err)
		}

		var config TemplateLayoutConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse zone config: %v", err)
		}
		gen.zoneConfig = &config
		fmt.Printf("✓ Loaded zone config: %s (%d zones)\n", config.TemplateName, len(config.Zones))
	}

	return gen, nil
}

// getZone retrieves a zone by name from the loaded config
func (g *PDFGenerator) getZone(name string) *TemplateZoneConfig {
	if g.zoneConfig == nil {
		return nil
	}
	for i := range g.zoneConfig.Zones {
		if g.zoneConfig.Zones[i].Name == name {
			return &g.zoneConfig.Zones[i]
		}
	}
	return nil
}

// getAnchorPoint retrieves anchor coordinates in PDF points
func (g *PDFGenerator) getAnchorPoint(zoneName, anchorName string) (x, y float64, found bool) {
	zone := g.getZone(zoneName)
	if zone == nil {
		return 0, 0, false
	}
	anchor, exists := zone.Anchors[anchorName]
	if !exists {
		return 0, 0, false
	}
	return anchor["x_mm"] * mmToPoints, anchor["y_mm"] * mmToPoints, true
}

// mmToPt converts millimeters to PDF points
func mmToPt(mm float64) float64 {
	return mm * mmToPoints
}

// normalizedToAbsolute converts normalized (0-1) coordinates to PDF points
// Uses the content box for layout-agnostic positioning
func (g *PDFGenerator) normalizedToAbsolute(u, v float64) (x, y float64) {
	// Convert from normalized content box coordinates to page coordinates
	x = g.pageWidth * (g.contentBox.X + u*g.contentBox.Width)
	y = g.pageHeight * (g.contentBox.Y + v*g.contentBox.Height)
	return x, y
}

// drawTextShaped draws text with proper shaping for Arabic/RTL
func (g *PDFGenerator) drawTextShaped(text string, x, y float64, langCode string) {
	if g.langPack.IsRTL(langCode) && IsArabicText(text) {
		// Shape Arabic text for proper glyph rendering
		shaped := g.arabicShaper.ShapeForPDF(text)
		g.pdf.SetXY(x, y)
		g.pdf.Cell(nil, shaped)
	} else {
		g.pdf.SetXY(x, y)
		g.pdf.Cell(nil, text)
	}
}

// SetContentBox sets a custom content box (e.g., from letterhead analysis)
func (g *PDFGenerator) SetContentBox(box ContentBox) {
	g.contentBox = box
}

// Doc returns the underlying *gopdf.GoPdf for direct PDF operations.
func (g *PDFGenerator) Doc() *gopdf.GoPdf { return g.pdf }

// ZoneConfig returns the loaded TemplateLayoutConfig, or nil if none loaded.
func (g *PDFGenerator) ZoneConfig() *TemplateLayoutConfig { return g.zoneConfig }

// GetAnchorPoint retrieves anchor coordinates in PDF points (exported wrapper).
func (g *PDFGenerator) GetAnchorPoint(zoneName, anchorName string) (float64, float64, bool) {
	return g.getAnchorPoint(zoneName, anchorName)
}

// loadLanguageFonts loads appropriate fonts for the specified language
func (g *PDFGenerator) loadLanguageFonts(langCode string) error {
	pack := g.langPack.Get(langCode)

	// Font map: language code -> list of font paths to try (Windows, macOS, Linux)
	fontPaths := map[string][]string{
		"en": {
			"fonts/DejaVuSans.ttf",
			"C:/Windows/Fonts/arial.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		},
		"ar": {
			"fonts/NotoSansArabic-Regular.ttf",
			"C:/Windows/Fonts/tahoma.ttf",
			"/System/Library/Fonts/GeezaPro.ttc",
			"/Library/Fonts/NotoSansArabic-Regular.ttf",
			"/usr/share/fonts/truetype/noto/NotoSansArabic-Regular.ttf",
		},
		"zh-CN": {
			"fonts/NotoSansSC-Regular.ttf",
			"C:/Windows/Fonts/msyh.ttc",
			"/System/Library/Fonts/PingFang.ttc",
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		},
		"ja": {
			"fonts/NotoSansJP-Regular.ttf",
			"C:/Windows/Fonts/msgothic.ttc",
			"/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc",
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		},
		"th": {
			"fonts/NotoSansThai-Regular.ttf",
			"C:/Windows/Fonts/tahoma.ttf",
			"/System/Library/Fonts/Supplemental/Thonburi.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansThai-Regular.ttf",
		},
		"hi": {
			"fonts/NotoSansDevanagari-Regular.ttf",
			"C:/Windows/Fonts/mangal.ttf",
			"/System/Library/Fonts/Kohinoor.ttc",
			"/Library/Fonts/NotoSansDevanagari-Regular.ttf",
			"/usr/share/fonts/truetype/noto/NotoSansDevanagari-Regular.ttf",
		},
		"ko": {
			"fonts/NotoSansKR-Regular.ttf",
			"C:/Windows/Fonts/malgun.ttf",
			"/System/Library/Fonts/AppleSDGothicNeo.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		},
		"he": {
			"fonts/NotoSansHebrew-Regular.ttf",
			"C:/Windows/Fonts/arial.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/System/Library/Fonts/Supplemental/Arial Hebrew.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansHebrew-Regular.ttf",
		},
		"ru": {
			"fonts/DejaVuSans.ttf",
			"C:/Windows/Fonts/arial.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		},
	}

	// Embedded fonts (pkg/fonts) are the PRIMARY source for the two scripts we
	// ship in the binary — Latin/Cyrillic (Noto Sans) and Arabic (Noto Naskh
	// Arabic) — so those languages render identically on every machine with
	// no dependency on what's installed on the host. Host-font probing below
	// remains the source for every other script (CJK, Thai, Devanagari,
	// Korean, Hebrew) we don't embed, and the fallback if embedding fails.
	embeddedFontFor := func(code string) []byte {
		switch code {
		case "ar":
			return fonts.NotoNaskhArabic()
		case "en", "ru":
			return fonts.NotoSans()
		default:
			return nil
		}
	}

	fontLoaded := false
	if data := embeddedFontFor(langCode); data != nil {
		if err := g.pdf.AddTTFFontData(pack.FontFamily, data); err == nil {
			g.pdf.SetFont(pack.FontFamily, "", 10)
			fontLoaded = true
			fmt.Printf("✓ Loaded embedded font for %s: pkg/fonts (%d bytes)\n", pack.Name, len(data))
		} else {
			fmt.Printf("⚠ Embedded font failed for %s, falling back to host probe: %v\n", pack.Name, err)
		}
	}

	if !fontLoaded {
		// Get font paths for this language
		paths, exists := fontPaths[langCode]
		if !exists {
			// Fallback to English fonts
			paths = fontPaths["en"]
		}

		for _, fontPath := range paths {
			if fileExists(fontPath) {
				err := g.pdf.AddTTFFont(pack.FontFamily, fontPath)
				if err == nil {
					g.pdf.SetFont(pack.FontFamily, "", 10)
					fontLoaded = true
					fmt.Printf("✓ Loaded font for %s: %s\n", pack.Name, fontPath)
					break
				}
			}
		}
	}

	if !fontLoaded {
		return fmt.Errorf("could not load font for language %s (%s)", pack.Name, langCode)
	}

	return nil
}

// Generate creates the PDF invoice with multi-language support
func (g *PDFGenerator) Generate(data *InvoiceData, outputPath string) error {
	// Default to English if no language specified
	if data.Language == "" {
		data.Language = "en"
	}

	// Add page
	g.pdf.AddPage()

	// Load fonts for the specified language
	err := g.loadLanguageFonts(data.Language)
	if err != nil {
		return fmt.Errorf("failed to load fonts: %v", err)
	}

	// Add letterhead as background image (if exists)
	if g.letterheadPath != "" && fileExists(g.letterheadPath) {
		// Draw letterhead at full page size
		// A4 = 595 x 842 points (at 72 DPI)
		imgErr := g.pdf.Image(g.letterheadPath, 0, 0, &gopdf.Rect{W: 595, H: 842})
		if imgErr != nil {
			fmt.Printf("Warning: Could not load letterhead image: %v\n", imgErr)
		}
	}

	// Draw invoice header (TAX INVOICE)
	g.drawHeader(data)

	// Draw company information (ACME INSTRUMENTATION W.L.L)
	g.drawCompanyInfo(data)

	// Draw buyer information
	g.drawBuyerInfo(data)

	// Draw line items table
	g.drawItemsTable(data)

	// Draw totals section
	g.drawTotals(data)

	// Draw amount in words
	g.drawAmountInWords(data)

	// Save PDF to file
	return g.pdf.WritePdf(outputPath)
}

// drawHeader draws the "TAX INVOICE" header with language support
func (g *PDFGenerator) drawHeader(data *InvoiceData) {
	pack := g.langPack.Get(data.Language)
	g.pdf.SetFont(pack.FontFamily, "", 16)

	headerText := g.langPack.Translate(data.Language, "invoice")

	// Use zone-based positioning if available
	var x, y float64
	if zoneX, zoneY, found := g.getAnchorPoint("invoice_type", "center"); found {
		x, y = zoneX-50, zoneY
	} else {
		// Fallback positioning using normalized coordinates
		if g.langPack.IsRTL(data.Language) {
			x, y = g.normalizedToAbsolute(0.60, 0.94) // Right-aligned for RTL
		} else {
			x, y = g.normalizedToAbsolute(0.05, 0.94)
		}
	}

	// Use shaped text for Arabic
	g.drawTextShaped(headerText, x, y, data.Language)
}

// drawCompanyInfo draws ACME INSTRUMENTATION W.L.L information with language support
func (g *PDFGenerator) drawCompanyInfo(data *InvoiceData) {
	pack := g.langPack.Get(data.Language)
	g.pdf.SetFont(pack.FontFamily, "", 10)

	isRTL := g.langPack.IsRTL(data.Language)

	// Try zone-based positioning first
	var companyX, companyY float64 = 50, 80
	var invNoLabelX, invNoLabelY float64 = 400, 80
	var invNoValueX, invNoValueY float64 = 480, 80

	if x, y, found := g.getAnchorPoint("seller_info", "company_name"); found {
		companyX, companyY = x, y
	}
	if x, y, found := g.getAnchorPoint("invoice_metadata", "invoice_no_label"); found {
		invNoLabelX, invNoLabelY = x, y
	}
	if x, y, found := g.getAnchorPoint("invoice_metadata", "invoice_no_value"); found {
		invNoValueX, invNoValueY = x, y
	}

	// For RTL, swap left/right positioning
	if isRTL && g.zoneConfig == nil {
		companyX = 350.0
		invNoLabelX = 150.0
		invNoValueX = 70.0
	}

	// Seller identity comes from the active overlay (deployment identity) —
	// Wave 3 B.4: the legal name and address lines were hardcoded here,
	// drifted copies of what the overlay's division profile already carries.
	// One source of truth; a different vertical retargets via overlay.json.
	profile := overlay.Active().Profile(overlay.Active().DefaultDivision())
	g.pdf.SetXY(companyX, companyY)
	g.pdf.Cell(nil, profile.LegalName)

	// Address lines at the historical 10pt-then-7pt vertical rhythm.
	lineY := companyY + 10
	for _, line := range profile.AddressLines {
		g.pdf.SetXY(companyX, lineY)
		g.pdf.Cell(nil, line)
		lineY += 7
	}

	// TRN, 11pt below the last address line (companyY+35 for the historical
	// three-line address block).
	g.pdf.SetXY(companyX, lineY+4)
	g.pdf.Cell(nil, fmt.Sprintf("%s: %s", g.langPack.Translate(data.Language, "trn"), data.TRN))

	// Invoice number and date (right side)
	g.pdf.SetXY(invNoLabelX, invNoLabelY)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "invoiceNo"))
	g.pdf.SetXY(invNoValueX, invNoValueY)
	g.pdf.Cell(nil, data.InvoiceNumber)

	g.pdf.SetXY(invNoLabelX, invNoLabelY+10)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "dated"))
	g.pdf.SetXY(invNoValueX, invNoValueY+10)
	g.pdf.Cell(nil, g.langPack.FormatDate(data.Language, data.InvoiceDate))

	// Delivery Note
	g.pdf.SetXY(invNoLabelX, invNoLabelY+20)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "deliveryNote"))
	g.pdf.SetXY(invNoValueX, invNoValueY+20)
	g.pdf.Cell(nil, data.DeliveryNote)

	// Payment Terms
	g.pdf.SetXY(invNoLabelX, invNoLabelY+30)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "paymentTerms"))
	g.pdf.SetXY(invNoValueX, invNoValueY+30)
	g.pdf.Cell(nil, data.PaymentTerms)
}

// drawBuyerInfo draws buyer information section
func (g *PDFGenerator) drawBuyerInfo(data *InvoiceData) {
	pack := g.langPack.Get(data.Language)
	g.pdf.SetFont(pack.FontFamily, "", 10)

	// Try zone-based positioning
	var buyerX, buyerY float64 = 50, 180
	var orderX, orderY float64 = 400, 195

	if x, y, found := g.getAnchorPoint("buyer_info", "buyer_label"); found {
		buyerX, buyerY = x, y
	}
	if x, y, found := g.getAnchorPoint("buyer_order_info", "order_no_label"); found {
		orderX, orderY = x, y
	}

	// "Buyer" label
	g.pdf.SetXY(buyerX, buyerY)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "buyer"))

	// Buyer name
	g.pdf.SetXY(buyerX, buyerY+10)
	g.pdf.Cell(nil, data.BuyerName)

	// Buyer address
	g.pdf.SetXY(buyerX, buyerY+18)
	g.pdf.Cell(nil, fmt.Sprintf("%s, Building %s", data.BuyerBuilding, data.BuyerBuilding))

	g.pdf.SetXY(buyerX, buyerY+25)
	g.pdf.Cell(nil, fmt.Sprintf("Road/Street %s, Town: %s, Block %s", data.BuyerRoad, data.BuyerTown, data.BuyerBlock))

	g.pdf.SetXY(buyerX, buyerY+32)
	g.pdf.Cell(nil, fmt.Sprintf("Country: %s", data.BuyerCountry))

	// Buyer TRN
	g.pdf.SetXY(buyerX, buyerY+40)
	g.pdf.Cell(nil, fmt.Sprintf("%s: %s", g.langPack.Translate(data.Language, "trn"), data.BuyerTRN))

	// Right side - Order details
	g.pdf.SetXY(orderX, orderY)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "buyerOrderNo"))
	g.pdf.SetXY(orderX+55, orderY)
	g.pdf.Cell(nil, data.BuyerOrderNo)

	g.pdf.SetXY(orderX, orderY+10)
	g.pdf.Cell(nil, g.langPack.Translate(data.Language, "dated"))
	g.pdf.SetXY(orderX+55, orderY+10)
	g.pdf.Cell(nil, g.langPack.FormatDate(data.Language, data.BuyerOrderDate))
}

// drawItemsTable draws the line items table
func (g *PDFGenerator) drawItemsTable(data *InvoiceData) {
	startY := 280.0
	g.pdf.SetFont("dejavu", "", 8)

	// Table headers
	headers := []string{"Sl No.", "Description", "Qty", "Rate", "Annual", "Monthly", "VAT%", "Taxable", "VAT", "Total"}
	colX := []float64{50, 80, 200, 240, 280, 330, 380, 415, 465, 505}

	// Draw header row
	for i, header := range headers {
		g.pdf.SetXY(colX[i], startY)
		g.pdf.Cell(nil, header)
	}

	// Draw line items
	currentY := startY + 15
	for _, item := range data.Items {
		g.pdf.SetXY(colX[0], currentY)
		g.pdf.Cell(nil, fmt.Sprintf("%d", item.SlNo))

		g.pdf.SetXY(colX[1], currentY)
		g.pdf.Cell(nil, truncateText(item.Description, 35))

		g.pdf.SetXY(colX[2], currentY)
		g.pdf.Cell(nil, formatNumber(item.Quantity))

		g.pdf.SetXY(colX[3], currentY)
		g.pdf.Cell(nil, formatMoney(item.Rate))

		g.pdf.SetXY(colX[4], currentY)
		g.pdf.Cell(nil, formatMoney(item.AnnualTotal))

		g.pdf.SetXY(colX[5], currentY)
		g.pdf.Cell(nil, formatMoney(item.MonthlyAmount))

		g.pdf.SetXY(colX[6], currentY)
		g.pdf.Cell(nil, fmt.Sprintf("%.0f%%", item.VATPercent))

		g.pdf.SetXY(colX[7], currentY)
		g.pdf.Cell(nil, formatMoney(item.TaxableValue))

		g.pdf.SetXY(colX[8], currentY)
		g.pdf.Cell(nil, formatMoney(item.VAT))

		g.pdf.SetXY(colX[9], currentY)
		g.pdf.Cell(nil, formatMoney(item.Total))

		currentY += 15
	}
}

// drawTotals draws the totals section at bottom with language support
func (g *PDFGenerator) drawTotals(data *InvoiceData) {
	pack := g.langPack.Get(data.Language)
	g.pdf.SetFont(pack.FontFamily, "", 10)

	// Use zone-based or normalized positioning
	var labelX, valueX, startY float64
	if x, y, found := g.getAnchorPoint("totals_section", "subtotal_label"); found {
		labelX, startY = x, y
		valueX = labelX + 65
	} else {
		labelX, startY = g.normalizedToAbsolute(0.60, 0.20)
		valueX = labelX + 100
	}

	// Subtotal
	g.drawTextShaped(g.langPack.Translate(data.Language, "subtotal"), labelX, startY, data.Language)
	g.pdf.SetXY(valueX, startY)
	g.pdf.Cell(nil, g.formatMoneyWithLang(data.Language, data.Subtotal))

	// Output VAT
	g.drawTextShaped(g.langPack.Translate(data.Language, "outputVAT"), labelX, startY+15, data.Language)
	g.pdf.SetXY(valueX, startY+15)
	g.pdf.Cell(nil, g.formatMoneyWithLang(data.Language, data.TotalVAT))

	// Grand Total
	g.pdf.SetFont(pack.FontFamily, "", 12)
	g.drawTextShaped(g.langPack.Translate(data.Language, "grandTotal"), labelX, startY+35, data.Language)
	g.pdf.SetXY(valueX, startY+35)
	g.pdf.Cell(nil, g.formatMoneyWithLang(data.Language, data.GrandTotal))
}

// drawAmountInWords draws amount in words section with language support
func (g *PDFGenerator) drawAmountInWords(data *InvoiceData) {
	pack := g.langPack.Get(data.Language)
	g.pdf.SetFont(pack.FontFamily, "", 9)

	// Use normalized positioning
	labelX, startY := g.normalizedToAbsolute(0.05, 0.12)

	g.drawTextShaped(g.langPack.Translate(data.Language, "amountInWords"), labelX, startY, data.Language)

	g.pdf.SetXY(labelX, startY+15)
	g.pdf.Cell(nil, numberToWords(data.GrandTotal)+" "+g.langPack.Translate(data.Language, "only"))

	g.drawTextShaped(g.langPack.Translate(data.Language, "vatInWords"), labelX, startY+35, data.Language)

	g.pdf.SetXY(labelX, startY+50)
	g.pdf.Cell(nil, numberToWords(data.TotalVAT)+" "+g.langPack.Translate(data.Language, "only"))

	// E. & O.E (Errors and Omissions Excepted)
	eoeX, _ := g.normalizedToAbsolute(0.85, 0.12)
	g.drawTextShaped(g.langPack.Translate(data.Language, "eoe"), eoeX, startY, data.Language)
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// formatMoney formats money according to language rules (moved to use langpack in caller)
func formatMoney(amount float64) string {
	return fmt.Sprintf("%.3f", amount) // BHD uses 3 decimal places (fils)
}

// formatMoneyWithLang formats money with language-specific rules
func (g *PDFGenerator) formatMoneyWithLang(langCode string, amount float64) string {
	return g.langPack.FormatNumber(langCode, amount, true)
}

func formatNumber(num float64) string {
	if num == float64(int64(num)) {
		return fmt.Sprintf("%d", int64(num))
	}
	return fmt.Sprintf("%.2f", num)
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// numberToWords converts a number to words (simplified - only handles common invoice amounts)
func numberToWords(amount float64) string {
	// Simplified version - in production would use a full number-to-words library
	// For now, return a placeholder
	dinars := int(amount)
	fils := int((amount - float64(dinars)) * 1000)

	return fmt.Sprintf("%d Bahraini Dinars and %d fils", dinars, fils)
}
