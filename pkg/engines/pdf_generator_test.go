package engines

import (
	"testing"
)

func TestPDFGenerator_FormatMoney(t *testing.T) {
	tests := []struct {
		amount float64
		want   string
	}{
		{1234.567, "1234.567"},
		{0.0, "0.000"},
		{10.5, "10.500"},
	}

	for _, tt := range tests {
		got := formatMoney(tt.amount)
		if got != tt.want {
			t.Errorf("formatMoney(%v) = %v, want %v", tt.amount, got, tt.want)
		}
	}
}

func TestPDFGenerator_FormatNumber(t *testing.T) {
	tests := []struct {
		num  float64
		want string
	}{
		{1234.567, "1234.57"},
		{10, "10"},
		{10.5, "10.50"},
	}

	for _, tt := range tests {
		got := formatNumber(tt.num)
		if got != tt.want {
			t.Errorf("formatNumber(%v) = %v, want %v", tt.num, got, tt.want)
		}
	}
}

func TestPDFGenerator_TruncateText(t *testing.T) {
	s := "This is a long string"
	got := truncateText(s, 10)
	if got != "This is..." {
		t.Errorf("truncateText() = %v, want This is...", got)
	}
}

func TestPDFGenerator_NumberToWords(t *testing.T) {
	got := numberToWords(1234.567)
	want := "1234 Bahraini Dinars and 567 fils"
	if got != want {
		t.Errorf("numberToWords() = %v, want %v", got, want)
	}
}
