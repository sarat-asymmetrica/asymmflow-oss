package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
)

// Locale represents a supported language.
type Locale string

const (
	EN Locale = "en"
	AR Locale = "ar"
	HI Locale = "hi"
	FR Locale = "fr"
	ES Locale = "es"
)

//go:embed messages/*.json
var embeddedMessages embed.FS

// Translator provides string localization.
type Translator struct {
	locale   Locale
	messages map[string]string
	fallback *Translator
}

// New creates a translator for the given locale.
func New(locale Locale) *Translator {
	locale = normalizeLocale(locale)
	messages, err := LoadEmbedded(locale)
	if err != nil {
		messages = map[string]string{}
	}

	var fallback *Translator
	if locale != EN {
		fallback = New(EN)
	}

	return &Translator{
		locale:   locale,
		messages: messages,
		fallback: fallback,
	}
}

// T returns the translated string for a key.
func (t *Translator) T(key string) string {
	if t == nil {
		return key
	}
	if value, ok := t.messages[key]; ok {
		return value
	}
	if t.fallback != nil {
		return t.fallback.T(key)
	}
	return key
}

// Tf returns a formatted translated string.
func (t *Translator) Tf(key string, args ...any) string {
	return fmt.Sprintf(t.T(key), args...)
}

// SetLocale changes the active locale.
func (t *Translator) SetLocale(locale Locale) {
	if t == nil {
		return
	}
	next := New(locale)
	t.locale = next.locale
	t.messages = next.messages
	t.fallback = next.fallback
}

// CurrentLocale returns the current locale.
func (t *Translator) CurrentLocale() Locale {
	if t == nil {
		return EN
	}
	return t.locale
}

// AvailableLocales returns all supported locales.
func AvailableLocales() []Locale {
	return []Locale{EN, AR, HI, FR, ES}
}

// LoadMessages loads translation messages from a JSON file.
func LoadMessages(locale Locale, path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeMessages(locale, content)
}

// LoadEmbedded loads translation messages from embedded resources.
func LoadEmbedded(locale Locale) (map[string]string, error) {
	locale = normalizeLocale(locale)
	content, err := embeddedMessages.ReadFile(fmt.Sprintf("messages/%s.json", locale))
	if err != nil {
		return nil, err
	}
	return decodeMessages(locale, content)
}

func decodeMessages(locale Locale, content []byte) (map[string]string, error) {
	var messages map[string]string
	if err := json.Unmarshal(content, &messages); err != nil {
		return nil, fmt.Errorf("load %s messages: %w", locale, err)
	}
	if messages == nil {
		messages = map[string]string{}
	}
	return messages, nil
}

func normalizeLocale(locale Locale) Locale {
	switch locale {
	case EN, AR, HI, FR, ES:
		return locale
	default:
		return EN
	}
}
