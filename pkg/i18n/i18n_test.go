package i18n

import "testing"

func TestEnglishTranslationReturnsString(t *testing.T) {
	tr := New(EN)

	if got := tr.T("nav.dashboard"); got != "Dashboard" {
		t.Fatalf("T(nav.dashboard) = %q, want Dashboard", got)
	}
}

func TestMissingLocaleKeyFallsBackToEnglish(t *testing.T) {
	tr := New(AR)

	if got := tr.T("test.english_only"); got != "English fallback" {
		t.Fatalf("fallback translation = %q, want English fallback", got)
	}
}

func TestMissingKeyReturnsKey(t *testing.T) {
	tr := New(HI)

	if got := tr.T("missing.everywhere"); got != "missing.everywhere" {
		t.Fatalf("missing key = %q, want original key", got)
	}
}

func TestTfFormatsWithArguments(t *testing.T) {
	tr := &Translator{locale: EN, messages: map[string]string{"hello": "Hello, %s"}}

	if got := tr.Tf("hello", "Asha"); got != "Hello, Asha" {
		t.Fatalf("formatted translation = %q, want Hello, Asha", got)
	}
}

func TestSetLocaleChangesActiveLocale(t *testing.T) {
	tr := New(EN)
	tr.SetLocale(HI)

	if got := tr.CurrentLocale(); got != HI {
		t.Fatalf("locale = %s, want %s", got, HI)
	}
	if got := tr.T("common.save"); got != "सहेजें" {
		t.Fatalf("Hindi save translation = %q", got)
	}
}
