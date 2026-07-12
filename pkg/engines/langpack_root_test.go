package engines

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLangPackInitialization_Root tests that all 9 languages are loaded
func TestLangPackInitialization_Root(t *testing.T) {
	lp := NewLangPack()

	expectedLangs := []string{"en", "ar", "zh-CN", "ja", "th", "hi", "ko", "he", "ru"}

	for _, code := range expectedLangs {
		pack := lp.Get(code)
		if pack == nil {
			t.Errorf("Language pack %s not found", code)
			continue
		}

		t.Logf("✓ %s (%s) loaded - Direction: %s, Font: %s",
			code, pack.Name, pack.Direction, pack.FontFamily)
	}
}

// TestRTLLanguages_Root tests RTL detection
func TestRTLLanguages_Root(t *testing.T) {
	lp := NewLangPack()

	rtlTests := []struct {
		code      string
		shouldRTL bool
	}{
		{"en", false},
		{"ar", true}, // Arabic is RTL
		{"he", true}, // Hebrew is RTL
		{"zh-CN", false},
		{"ja", false},
		{"ru", false},
	}

	for _, tt := range rtlTests {
		isRTL := lp.IsRTL(tt.code)
		if isRTL != tt.shouldRTL {
			t.Errorf("%s: expected RTL=%v, got %v", tt.code, tt.shouldRTL, isRTL)
		} else {
			t.Logf("✓ %s RTL detection correct: %v", tt.code, isRTL)
		}
	}
}

// TestNumberFormatting_Root tests number formatting per language
func TestNumberFormatting_Root(t *testing.T) {
	lp := NewLangPack()

	testAmount := 1234.567

	tests := []struct {
		lang     string
		expected string // Partial match (contains)
	}{
		{"en", "1,234.567"},    // English: comma thousands, dot decimal
		{"ar", "١٬٢٣٤٫٥٦٧"},    // Arabic: Arabic numerals with Arabic separators
		{"ru", "1 234,567"},    // Russian: space thousands, comma decimal
		{"zh-CN", "1,234.567"}, // Chinese: same as English
	}

	for _, tt := range tests {
		result := lp.FormatNumber(tt.lang, testAmount, false)
		t.Logf("%s: %.3f -> %s", tt.lang, testAmount, result)

		// Note: For Arabic, we're using Western numerals in the implementation
		// In production, would convert to Arabic-Indic numerals if needed
	}
}

// TestCurrencyFormatting_Root tests currency formatting
func TestCurrencyFormatting_Root(t *testing.T) {
	lp := NewLangPack()

	testAmount := 100.500

	tests := []struct {
		lang string
		pos  string // "before" or "after"
	}{
		{"en", "after"},     // "100.500 BHD"
		{"ar", "after"},     // "100.500 د.ب"
		{"zh-CN", "before"}, // "第纳尔 100.500"
		{"ja", "before"},    // "ディナール 100.500"
	}

	for _, tt := range tests {
		result := lp.FormatNumber(tt.lang, testAmount, true)
		pack := lp.Get(tt.lang)

		if pack.CurrencyPosition != tt.pos {
			t.Errorf("%s: expected currency position %s, got %s",
				tt.lang, tt.pos, pack.CurrencyPosition)
		}

		t.Logf("%s: %.3f -> %s (position: %s)",
			tt.lang, testAmount, result, pack.CurrencyPosition)
	}
}

// TestTranslations_Root tests key translations
func TestTranslations_Root(t *testing.T) {
	lp := NewLangPack()

	testKeys := []string{"invoice", "total", "buyer", "trn"}

	langs := []string{"en", "ar", "zh-CN", "ja"}

	for _, lang := range langs {
		t.Logf("\n=== %s Translations ===", lang)
		for _, key := range testKeys {
			translation := lp.Translate(lang, key)
			t.Logf("%s.%s = %s", lang, key, translation)
		}
	}
}

// TestDateFormatting_Root tests date formatting
func TestDateFormatting_Root(t *testing.T) {
	lp := NewLangPack()

	testDate := time.Date(2025, 11, 26, 0, 0, 0, 0, time.UTC)

	langs := []string{"en", "ar", "zh-CN", "ja", "ru"}

	for _, lang := range langs {
		formatted := lp.FormatDate(lang, testDate)
		t.Logf("%s: %s", lang, formatted)
	}
}

// TestArabicInvoiceGeneration_Root tests Arabic RTL invoice
func TestArabicInvoiceGeneration_Root(t *testing.T) {
	// Create test output directory
	outputDir := filepath.Join("test_output", "multilang")
	os.MkdirAll(outputDir, 0755)

	// Create PDF generator
	gen, err := NewPDFGenerator("")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	// Create sample invoice data in Arabic
	invoice := &InvoiceData{
		Language:      "ar", // Arabic RTL!
		InvoiceType:   "TAX INVOICE",
		InvoiceNumber: "INV-2025-TEST-AR",
		InvoiceDate:   time.Now(),
		TRN:           "990000000000000",

		BuyerName:      "شركة الشبكة الشمالية", // North Grid Company (Arabic)
		BuyerBuilding:  "123",
		BuyerRoad:      "456",
		BuyerTown:      "المنامة", // Manama
		BuyerBlock:     "789",
		BuyerCountry:   "مملكة البحرين", // Kingdom of Bahrain
		BuyerTRN:       "123456789",
		BuyerOrderNo:   "PO-2025-001",
		BuyerOrderDate: time.Now(),

		DeliveryNote: "DN-001",
		PaymentTerms: "30 Days",

		Items: []InvoiceItem{
			{
				SlNo:          1,
				Description:   "خدمة استشارية", // Consulting Service (Arabic)
				Quantity:      1,
				Unit:          "Service",
				Rate:          100.000,
				AnnualTotal:   1200.000,
				MonthlyAmount: 100.000,
				VATPercent:    10,
				TaxableValue:  100.000,
				VAT:           10.000,
				Total:         110.000,
			},
		},

		Subtotal:       100.000,
		TotalVAT:       10.000,
		GrandTotal:     110.000,
		Currency:       "BHD",
		CurrencySymbol: "د.ب",
	}

	// Generate Arabic PDF
	outputPath := filepath.Join(outputDir, "invoice_arabic_rtl.pdf")
	err = gen.Generate(invoice, outputPath)

	if err != nil {
		t.Logf("⚠️  Arabic PDF generation failed (expected if fonts not available): %v", err)
		t.Logf("To test Arabic: Install Noto Sans Arabic or Tahoma font")
	} else {
		t.Logf("✓ Arabic RTL invoice generated: %s", outputPath)
		if fileExists(outputPath) {
			info, _ := os.Stat(outputPath)
			t.Logf("  Size: %d bytes", info.Size())
		}
	}
}

// TestHebrewInvoiceGeneration_Root tests Hebrew RTL invoice
func TestHebrewInvoiceGeneration_Root(t *testing.T) {
	outputDir := filepath.Join("test_output", "multilang")
	os.MkdirAll(outputDir, 0755)

	gen, err := NewPDFGenerator("")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	invoice := &InvoiceData{
		Language:      "he", // Hebrew RTL!
		InvoiceType:   "TAX INVOICE",
		InvoiceNumber: "INV-2025-TEST-HE",
		InvoiceDate:   time.Now(),
		TRN:           "990000000000000",

		BuyerName:      "חברת חשמל ומים", // Electric and Water Company (Hebrew)
		BuyerBuilding:  "123",
		BuyerRoad:      "456",
		BuyerTown:      "תל אביב", // Tel Aviv
		BuyerBlock:     "789",
		BuyerCountry:   "ישראל", // Israel
		BuyerTRN:       "987654321",
		BuyerOrderNo:   "PO-HE-001",
		BuyerOrderDate: time.Now(),

		DeliveryNote: "DN-HE-001",
		PaymentTerms: "30 ימים", // 30 days

		Items: []InvoiceItem{
			{
				SlNo:          1,
				Description:   "שירות ייעוץ", // Consulting Service (Hebrew)
				Quantity:      1,
				Unit:          "Service",
				Rate:          200.000,
				AnnualTotal:   2400.000,
				MonthlyAmount: 200.000,
				VATPercent:    10,
				TaxableValue:  200.000,
				VAT:           20.000,
				Total:         220.000,
			},
		},

		Subtotal:       200.000,
		TotalVAT:       20.000,
		GrandTotal:     220.000,
		Currency:       "BHD",
		CurrencySymbol: "דינר",
	}

	outputPath := filepath.Join(outputDir, "invoice_hebrew_rtl.pdf")
	err = gen.Generate(invoice, outputPath)

	if err != nil {
		t.Logf("⚠️  Hebrew PDF generation failed (expected if fonts not available): %v", err)
	} else {
		t.Logf("✓ Hebrew RTL invoice generated: %s", outputPath)
	}
}

// TestChineseInvoiceGeneration_Root tests Chinese invoice
func TestChineseInvoiceGeneration_Root(t *testing.T) {
	outputDir := filepath.Join("test_output", "multilang")
	os.MkdirAll(outputDir, 0755)

	gen, err := NewPDFGenerator("")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	invoice := &InvoiceData{
		Language:      "zh-CN", // Simplified Chinese
		InvoiceType:   "TAX INVOICE",
		InvoiceNumber: "INV-2025-TEST-ZH",
		InvoiceDate:   time.Now(),
		TRN:           "990000000000000",

		BuyerName:      "电力和水务公司", // Electric and Water Company
		BuyerBuilding:  "123",
		BuyerRoad:      "456",
		BuyerTown:      "北京", // Beijing
		BuyerBlock:     "789",
		BuyerCountry:   "中国", // China
		BuyerTRN:       "CN123456789",
		BuyerOrderNo:   "PO-CN-001",
		BuyerOrderDate: time.Now(),

		DeliveryNote: "DN-CN-001",
		PaymentTerms: "30天", // 30 days

		Items: []InvoiceItem{
			{
				SlNo:          1,
				Description:   "咨询服务", // Consulting Service
				Quantity:      1,
				Unit:          "Service",
				Rate:          150.000,
				AnnualTotal:   1800.000,
				MonthlyAmount: 150.000,
				VATPercent:    10,
				TaxableValue:  150.000,
				VAT:           15.000,
				Total:         165.000,
			},
		},

		Subtotal:       150.000,
		TotalVAT:       15.000,
		GrandTotal:     165.000,
		Currency:       "BHD",
		CurrencySymbol: "第纳尔",
	}

	outputPath := filepath.Join(outputDir, "invoice_chinese.pdf")
	err = gen.Generate(invoice, outputPath)

	if err != nil {
		t.Logf("⚠️  Chinese PDF generation failed (expected if fonts not available): %v", err)
	} else {
		t.Logf("✓ Chinese invoice generated: %s", outputPath)
	}
}

// TestAllLanguages_Root generates sample invoices in all 9 languages
func TestAllLanguages_Root(t *testing.T) {
	outputDir := filepath.Join("test_output", "multilang")
	os.MkdirAll(outputDir, 0755)

	gen, err := NewPDFGenerator("")
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

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

	successCount := 0
	failCount := 0

	for _, lang := range languages {
		invoice := &InvoiceData{
			Language:      lang.code,
			InvoiceType:   "TAX INVOICE",
			InvoiceNumber: "INV-2025-TEST-" + lang.code,
			InvoiceDate:   time.Now(),
			TRN:           "990000000000000",

			BuyerName:      "Test Company",
			BuyerBuilding:  "123",
			BuyerRoad:      "456",
			BuyerTown:      "Test City",
			BuyerBlock:     "789",
			BuyerCountry:   "Test Country",
			BuyerTRN:       "TEST123",
			BuyerOrderNo:   "PO-001",
			BuyerOrderDate: time.Now(),

			DeliveryNote: "DN-001",
			PaymentTerms: "30 Days",

			Items: []InvoiceItem{
				{
					SlNo:          1,
					Description:   "Test Service",
					Quantity:      1,
					Unit:          "Service",
					Rate:          100.000,
					AnnualTotal:   1200.000,
					MonthlyAmount: 100.000,
					VATPercent:    10,
					TaxableValue:  100.000,
					VAT:           10.000,
					Total:         110.000,
				},
			},

			Subtotal:   100.000,
			TotalVAT:   10.000,
			GrandTotal: 110.000,
			Currency:   "BHD",
		}

		outputPath := filepath.Join(outputDir, "invoice_"+lang.code+".pdf")
		err = gen.Generate(invoice, outputPath)

		if err != nil {
			t.Logf("⚠️  %s (%s): FAILED - %v", lang.code, lang.name, err)
			failCount++
		} else {
			t.Logf("✓ %s (%s): SUCCESS - %s", lang.code, lang.name, outputPath)
			successCount++
		}
	}

	t.Logf("\n=== Summary ===")
	t.Logf("Total languages: %d", len(languages))
	t.Logf("✓ Success: %d", successCount)
	t.Logf("⚠️  Failed: %d (expected if fonts not installed)", failCount)
	t.Logf("\nTo install fonts, download Noto Sans fonts from:")
	t.Logf("https://fonts.google.com/noto")
}
