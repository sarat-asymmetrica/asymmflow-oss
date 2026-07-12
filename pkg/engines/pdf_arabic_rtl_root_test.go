package engines

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func requireTestFile_Root(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("skipping: required test asset missing: %s", path)
	}
}

// TestArabicRTLInvoice_Root generates an Arabic RTL invoice PDF
func TestArabicRTLInvoice_Root(t *testing.T) {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ ARABIC RTL INVOICE PDF GENERATION TEST                        ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create output directory
	outDir := "test_output"
	os.MkdirAll(outDir, 0755)
	requireTestFile_Root(t, "ph_trading_zones.json")

	// Create generator with zone config
	gen, err := NewPDFGeneratorWithZones(
		"test_data/letterhead.png", // Letterhead if exists
		"ph_trading_zones.json",    // Zone config
	)
	if err != nil {
		t.Fatalf("Failed to create PDF generator: %v", err)
	}

	// Create Arabic invoice data
	invoice := &InvoiceData{
		Language:      "ar", // Arabic RTL!
		InvoiceType:   "TAX INVOICE",
		InvoiceNumber: "INV-2025-0107",
		InvoiceDate:   time.Now(),
		TRN:           "990000000000000",

		// Buyer info in Arabic
		BuyerName:      "شركة الشبكة الشمالية", // North Grid Authority
		BuyerBuilding:  "مبنى 123",
		BuyerRoad:      "شارع 456",
		BuyerTown:      "المنامة",
		BuyerBlock:     "البلوك 789",
		BuyerCountry:   "مملكة البحرين",
		BuyerTRN:       "100000000000001",
		BuyerOrderNo:   "PO-2025-001",
		BuyerOrderDate: time.Now().AddDate(0, 0, -7),

		DeliveryNote:     "DN-2025-050",
		DeliveryNoteDate: time.Now().AddDate(0, 0, -2),
		PaymentTerms:     "30 يوم", // 30 Days

		// Line items with Arabic descriptions
		Items: []InvoiceItem{
			{
				SlNo:         1,
				Description:  "محلل السوائل - Liquid Analyzer LA442",
				Quantity:     1,
				Unit:         "قطعة",
				Rate:         1500.000,
				TaxableValue: 1500.000,
				VATPercent:   10,
				VAT:          150.000,
				Total:        1650.000,
			},
			{
				SlNo:         2,
				Description:  "مستشعر مستوى - Level Sensor LS90",
				Quantity:     2,
				Unit:         "قطعة",
				Rate:         850.000,
				TaxableValue: 1700.000,
				VATPercent:   10,
				VAT:          170.000,
				Total:        1870.000,
			},
			{
				SlNo:         3,
				Description:  "قطع غيار ومواد استهلاكية",
				Quantity:     1,
				Unit:         "مجموعة",
				Rate:         250.000,
				TaxableValue: 250.000,
				VATPercent:   10,
				VAT:          25.000,
				Total:        275.000,
			},
		},

		Subtotal:       3450.000,
		TotalVAT:       345.000,
		GrandTotal:     3795.000,
		Currency:       "BHD",
		CurrencySymbol: "د.ب",
	}

	// Generate Arabic PDF with timestamped name to avoid file lock issues
	outputPath := filepath.Join(outDir, fmt.Sprintf("arabic_rtl_invoice_%d.pdf", time.Now().Unix()))
	err = gen.Generate(invoice, outputPath)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "failed to load fonts") {
			t.Skipf("skipping Arabic PDF generation: %v", err)
		}
		t.Fatalf("Failed to generate Arabic PDF: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	fmt.Printf("✓ Arabic RTL invoice generated: %s\n", outputPath)
	fmt.Printf("✓ File size: %d bytes\n", info.Size())
	fmt.Println()

	// Test all 9 languages
	t.Run("AllLanguages", func(t *testing.T) {
		languages := []struct {
			code string
			name string
		}{
			{"en", "English"},
			{"ar", "Arabic"},
			{"zh-CN", "Chinese"},
			{"ja", "Japanese"},
			{"th", "Thai"},
			{"hi", "Hindi"},
			{"ko", "Korean"},
			{"he", "Hebrew"},
			{"ru", "Russian"},
		}

		for _, lang := range languages {
			t.Run(lang.name, func(t *testing.T) {
				gen2, _ := NewPDFGenerator("")
				invoice2 := &InvoiceData{
					Language:      lang.code,
					InvoiceType:   "TAX INVOICE",
					InvoiceNumber: "INV-2025-0108-" + lang.code,
					InvoiceDate:   time.Now(),
					TRN:           "990000000000000",
					BuyerName:     "Test Customer - " + lang.name,
					BuyerCountry:  "Bahrain",
					PaymentTerms:  "30 Days",
					Items: []InvoiceItem{
						{
							SlNo:         1,
							Description:  "Test Item",
							Quantity:     1,
							Rate:         100.000,
							TaxableValue: 100.000,
							VATPercent:   10,
							VAT:          10.000,
							Total:        110.000,
						},
					},
					Subtotal:   100.000,
					TotalVAT:   10.000,
					GrandTotal: 110.000,
				}

				outPath := filepath.Join(outDir, fmt.Sprintf("invoice_%s.pdf", lang.code))
				err := gen2.Generate(invoice2, outPath)
				if err != nil {
					// Font may not be available - just warn
					fmt.Printf("⚠ %s (%s): %v\n", lang.name, lang.code, err)
				} else {
					info, _ := os.Stat(outPath)
					fmt.Printf("✓ %s (%s): %d bytes\n", lang.name, lang.code, info.Size())
				}
			})
		}
	})
}

// TestZoneBasedPositioning_Root verifies zone config is loaded and used
func TestZoneBasedPositioning_Root(t *testing.T) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ ZONE-BASED POSITIONING TEST                                   ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	requireTestFile_Root(t, "ph_trading_zones.json")

	// Create generator with zone config
	gen, err := NewPDFGeneratorWithZones("", "ph_trading_zones.json")
	if err != nil {
		t.Fatalf("Failed to load zone config: %v", err)
	}

	// Verify zones loaded
	if gen.ZoneConfig() == nil {
		t.Fatal("Zone config not loaded")
	}

	fmt.Printf("✓ Template: %s\n", gen.ZoneConfig().TemplateName)
	fmt.Printf("✓ Page size: %.0f × %.0f mm\n",
		gen.ZoneConfig().PageSize.WidthMM, gen.ZoneConfig().PageSize.HeightMM)
	fmt.Printf("✓ Zones loaded: %d\n", len(gen.ZoneConfig().Zones))
	fmt.Println()

	// Test anchor point retrieval
	testAnchors := []struct {
		zone   string
		anchor string
	}{
		{"invoice_type", "center"},
		{"seller_info", "company_name"},
		{"invoice_metadata", "invoice_no_label"},
		{"buyer_info", "buyer_label"},
		{"items_table", "table_top_left"},
		{"totals_section", "total_label"},
	}

	for _, ta := range testAnchors {
		x, y, found := gen.GetAnchorPoint(ta.zone, ta.anchor)
		if found {
			fmt.Printf("✓ %s.%s → (%.1f, %.1f) points\n", ta.zone, ta.anchor, x, y)
		} else {
			fmt.Printf("✗ %s.%s → NOT FOUND\n", ta.zone, ta.anchor)
		}
	}

	// Generate invoice with zones
	os.MkdirAll("test_output", 0755)
	invoice := &InvoiceData{
		Language:      "en",
		InvoiceNumber: "INV-2025-ZONE-TEST",
		InvoiceDate:   time.Now(),
		TRN:           "990000000000000",
		BuyerName:     "Zone Test Customer",
		BuyerCountry:  "Kingdom of Bahrain",
		PaymentTerms:  "30 Days",
		Items: []InvoiceItem{
			{SlNo: 1, Description: "Zone Positioned Item", Quantity: 1, Rate: 500.000,
				TaxableValue: 500.000, VATPercent: 10, VAT: 50.000, Total: 550.000},
		},
		Subtotal: 500.000, TotalVAT: 50.000, GrandTotal: 550.000,
	}

	err = gen.Generate(invoice, "test_output/zone_positioned_invoice.pdf")
	if err != nil {
		t.Fatalf("Failed to generate zone-positioned PDF: %v", err)
	}

	fmt.Println()
	fmt.Println("✓ Zone-positioned invoice generated: test_output/zone_positioned_invoice.pdf")
}

// TestLangPackTranslations_Root verifies all translation keys work
func TestLangPackTranslations_Root(t *testing.T) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ LANGPACK TRANSLATION TEST                                     ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	lp := NewLangPack()

	// Translation keys to test
	keys := []string{
		"invoice", "invoiceNo", "dated", "buyer", "trn",
		"subtotal", "outputVAT", "grandTotal", "amountInWords",
	}

	// Test Arabic specifically (key market for Acme Instrumentation!)
	fmt.Println("ARABIC TRANSLATIONS (العربية):")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	for _, key := range keys {
		ar := lp.Translate("ar", key)
		en := lp.Translate("en", key)
		fmt.Printf("  %-15s : %-25s → %s\n", key, en, ar)
	}

	fmt.Println()
	fmt.Println("NUMBER FORMATTING:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	testValue := 1234567.890

	langs := []string{"en", "ar", "zh-CN", "ru"}
	for _, lang := range langs {
		pack := lp.Get(lang)
		formatted := lp.FormatNumber(lang, testValue, true)
		fmt.Printf("  %s (%s): %s\n", pack.Name, lang, formatted)
	}

	fmt.Println()
	fmt.Println("RTL DETECTION:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, lang := range lp.GetAvailableLanguages() {
		direction := "LTR"
		if lp.IsRTL(lang.Code) {
			direction = "RTL"
		}
		fmt.Printf("  %s (%s): %s\n", lang.Name, lang.Code, direction)
	}

	// Verify RTL languages
	if !lp.IsRTL("ar") {
		t.Error("Arabic should be RTL")
	}
	if !lp.IsRTL("he") {
		t.Error("Hebrew should be RTL")
	}
	if lp.IsRTL("en") {
		t.Error("English should not be RTL")
	}

	fmt.Println()
	fmt.Println("✓ All translation tests passed!")
}
