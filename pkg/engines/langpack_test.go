package engines

import (
	"testing"
)

func TestLangPack_Translate(t *testing.T) {
	lp := NewLangPack()

	// Test English translation
	en := lp.Translate("en", "invoice")
	if en != "TAX INVOICE" {
		t.Errorf("Expected TAX INVOICE, got %s", en)
	}

	// Test Arabic translation
	ar := lp.Translate("ar", "invoice")
	if ar != "فاتورة ضريبية" {
		t.Errorf("Expected فاتورة ضريبية, got %s", ar)
	}

	// Test fallback
	fb := lp.Translate("xx", "invoice")
	if fb != "TAX INVOICE" {
		t.Errorf("Expected fallback to English TAX INVOICE, got %s", fb)
	}
}

func TestLangPack_GetConfig(t *testing.T) {
	lp := NewLangPack()

	conf := lp.Get("ar")
	if conf == nil {
		t.Fatal("Expected Arabic config, got nil")
	}

	if conf.Direction != "rtl" {
		t.Errorf("Expected Arabic to be rtl, got %s", conf.Direction)
	}
}
