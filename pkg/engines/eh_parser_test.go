package engines

import (
	"testing"

	"ph_holdings_app/pkg/overlay"
)

func TestEHParser_ParseBatch_Errors(t *testing.T) {
	parser := NewEHParser()

	// Test with non-existent directory
	_, err := parser.ParseBatch("non_existent_dir_12345")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

func TestEHParser_ClassifyProductType(t *testing.T) {
	parser := NewEHParser()

	tests := []struct {
		orderCode   string
		description string
		want        string
	}{
		{"CM442-3RT0/0", "Liquiline CM442 transmitter", "Rhine Flow"},
		{"FMU90-R11CA111AA3A", "Prosonic FMU90 level measurement", "Rhine Level"},
		{"53P50-EA0A1AA0ACAA", "Cerabar PMC51 pressure sensor", "Rhine Instruments Pressure"},
		{"TMT162-A1BA2ADAA1", "iTHERM TrustSens temperature", "Rhine Instruments Temperature"},
		{"10W40-UA0A1AA0ACAA", "Promag W 400 flowmeter", "Rhine Flow"},
		{"UNKNOWN", "Unknown device", "Rhine Instruments General"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			if got := parser.ClassifyProductType(tt.orderCode, tt.description); got != tt.want {
				t.Errorf("EHParser.ClassifyProductType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEHParser_Conversion(t *testing.T) {
	// The EUR→BHD rate is config-driven from the active overlay (the single
	// source of truth for FX), not a hardcoded constant. With built-in defaults
	// that rate is 0.45, so 100 EUR = 45 BHD — and the parser must use exactly
	// that configured rate (this is what unified the old 0.41-vs-0.45 split).
	parser := NewEHParser()
	want := overlay.Active().ExchangeRateToBase("EUR")
	if parser.ConversionRate != want {
		t.Errorf("NewEHParser ConversionRate = %v, want overlay EUR rate %v", parser.ConversionRate, want)
	}
	if got := 100.0 * parser.ConversionRate; got != 45.0 {
		t.Errorf("100 EUR = %v BHD, want 45 (overlay default EUR rate 0.45)", got)
	}
}
