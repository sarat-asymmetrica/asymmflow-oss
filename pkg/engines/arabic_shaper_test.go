package engines

import (
	"testing"
)

func TestArabicShaper_ShapeForPDF(t *testing.T) {
	shaper := NewArabicShaper()

	testCases := []struct {
		name  string
		input string
	}{
		{"Simple word", "فاتورة"},
		{"Tax Invoice", "فاتورة ضريبية"},
		{"Buyer", "المشتري"},
		{"Mixed", "فاتورة INV-2025-0107"},
		{"Numbers", "المبلغ: 1234.567"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shaped := shaper.ShapeForPDF(tc.input)
			if len(shaped) == 0 && len(tc.input) > 0 {
				t.Errorf("Shaped output is empty for input: %s", tc.input)
			}
		})
	}
}

func TestFormatArabicNumber(t *testing.T) {
	testCases := []struct {
		num      float64
		decimals int
	}{
		{1234.567, 3},
		{0, 0},
		{1000000, 0},
	}

	for _, tc := range testCases {
		formatted := FormatArabicNumber(tc.num, tc.decimals)
		if formatted == "" {
			t.Errorf("Formatted output is empty for %v", tc.num)
		}
	}
}

func TestIsArabicText(t *testing.T) {
	testCases := []struct {
		text     string
		expected bool
	}{
		{"Hello", false},
		{"فاتورة", true},
		{"مرحبا Hello", true},
		{"123", false},
	}

	for _, tc := range testCases {
		if got := IsArabicText(tc.text); got != tc.expected {
			t.Errorf("IsArabicText(%q) = %v, want %v", tc.text, got, tc.expected)
		}
	}
}
