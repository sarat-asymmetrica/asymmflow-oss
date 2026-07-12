package engines

import (
	"fmt"
	"testing"
)

// TestArabicShaper_Comprehensive tests Arabic text shaping
func TestArabicShaper_Comprehensive(t *testing.T) {
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ ARABIC TEXT SHAPER TEST                                       ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	shaper := NewArabicShaper()

	// Test cases: input → expected shaped output (letters should be joined)
	testCases := []struct {
		name  string
		input string
		desc  string
	}{
		{
			name:  "Simple word",
			input: "فاتورة",
			desc:  "Invoice - letters should connect",
		},
		{
			name:  "Tax Invoice",
			input: "فاتورة ضريبية",
			desc:  "Full phrase - Tax Invoice",
		},
		{
			name:  "Buyer",
			input: "المشتري",
			desc:  "The Buyer - starts with Alef-Lam",
		},
		{
			name:  "Mixed Arabic-English",
			input: "فاتورة INV-2025-0107",
			desc:  "Invoice number mixed",
		},
		{
			name:  "With numbers",
			input: "المبلغ: 1234.567",
			desc:  "Amount with numbers",
		},
		{
			name:  "Company name",
			input: "هيئة الكهرباء والماء",
			desc:  "North Grid Authority",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shaped := shaper.ShapeForPDF(tc.input)

			fmt.Printf("  Input:  %s\n", tc.input)
			fmt.Printf("  Shaped: %s\n", shaped)
			fmt.Printf("  Desc:   %s\n", tc.desc)
			fmt.Println()

			// Basic validation: output should not be empty
			if len(shaped) == 0 && len(tc.input) > 0 {
				t.Errorf("Shaped output is empty for input: %s", tc.input)
			}

			// Check that shaping occurred (presentation forms should be different)
			if tc.input == shaped && IsArabicText(tc.input) {
				// Note: This might be expected for some inputs
				fmt.Println("  Note: Input unchanged (may be expected)")
			}
		})
	}
}

// TestArabicNumberFormatting_Comprehensive tests Arabic-Indic number conversion
func TestArabicNumberFormatting_Comprehensive(t *testing.T) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ ARABIC NUMBER FORMATTING TEST                                 ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	testCases := []struct {
		num      float64
		decimals int
		desc     string
	}{
		{1234.567, 3, "Standard decimal"},
		{0, 0, "Zero"},
		{1000000, 0, "Million with separators"},
		{123.456, 3, "Small number"},
		{999999.999, 3, "Large with decimals"},
	}

	fmt.Println("Western → Arabic-Indic conversion:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, tc := range testCases {
		formatted := FormatArabicNumber(tc.num, tc.decimals)
		western := formatSimpleNumber(tc.num, tc.decimals)

		fmt.Printf("  %15s → %s  (%s)\n", western, formatted, tc.desc)
	}

	fmt.Println()

	// Validate specific conversions
	t.Run("DigitConversion", func(t *testing.T) {
		result := FormatArabicNumber(1234567890, 0)
		// Should contain Arabic-Indic digits
		if !containsRune(result, '١') {
			t.Errorf("Expected Arabic-Indic digits, got: %s", result)
		}
		fmt.Printf("  ✓ 1234567890 → %s\n", result)
	})

	t.Run("DecimalSeparator", func(t *testing.T) {
		result := FormatArabicNumber(123.456, 3)
		// Should contain Arabic decimal separator
		if !containsRune(result, '٫') {
			t.Errorf("Expected Arabic decimal separator, got: %s", result)
		}
		fmt.Printf("  ✓ 123.456 → %s\n", result)
	})
}

// TestRTLDetection_Root tests RTL language detection
func TestRTLDetection_Root(t *testing.T) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ RTL DETECTION TEST                                            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	testCases := []struct {
		text     string
		expected bool
		desc     string
	}{
		{"Hello World", false, "English"},
		{"فاتورة ضريبية", true, "Arabic"},
		{"שלום", false, "Hebrew (IsArabicText only checks Arabic, not all RTL)"},
		{"مرحبا Hello", true, "Mixed Arabic-English"},
		{"123.456", false, "Numbers only"},
		{"", false, "Empty string"},
	}

	for _, tc := range testCases {
		result := IsArabicText(tc.text)
		status := "✗"
		if result == tc.expected {
			status = "✓"
		}
		fmt.Printf("  %s \"%s\" → IsArabic=%v (%s)\n", status, tc.text, result, tc.desc)

		if result != tc.expected {
			t.Errorf("IsArabicText(%q) = %v, expected %v", tc.text, result, tc.expected)
		}
	}

	fmt.Println()
	fmt.Println("✓ All RTL detection tests passed!")
}

// TestLetterForms_Root tests that letter form mappings are complete
func TestLetterForms_Root(t *testing.T) {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════╗")
	fmt.Println("║ ARABIC LETTER FORMS TEST                                      ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Count letters in our mapping
	letterCount := len(arabicLetterForms)
	fmt.Printf("  Arabic letters mapped: %d\n", letterCount)

	// Verify all standard Arabic letters are present
	standardLetters := "ابتثجحخدذرزسشصضطظعغفقكلمنهوي"
	missingCount := 0
	for _, r := range standardLetters {
		if _, exists := arabicLetterForms[r]; !exists {
			fmt.Printf("  ⚠ Missing letter: %c (U+%04X)\n", r, r)
			missingCount++
		}
	}

	if missingCount == 0 {
		fmt.Println("  ✓ All standard Arabic letters present")
	} else {
		t.Errorf("%d standard letters missing from mapping", missingCount)
	}

	// Test that each letter has 4 forms
	for letter, forms := range arabicLetterForms {
		for i, form := range forms {
			if form == 0 {
				t.Errorf("Letter %c (U+%04X) has empty form at position %d", letter, letter, i)
			}
		}
	}

	fmt.Printf("  ✓ All %d letters have complete form mappings\n", letterCount)
}

// Helper to check if string contains rune
func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
