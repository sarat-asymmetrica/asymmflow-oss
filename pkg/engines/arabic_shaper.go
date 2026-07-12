// ═══════════════════════════════════════════════════════════════════════════
// ARABIC TEXT SHAPER - Pure Go Implementation
//
// Arabic letters change shape based on position:
// - Isolated (standalone)
// - Initial (start of word)
// - Medial (middle of word)
// - Final (end of word)
//
// This shaper converts logical Arabic text to presentation forms.
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"strings"
	"unicode"
)

// ArabicShaper handles Arabic text shaping for PDF rendering
type ArabicShaper struct{}

// NewArabicShaper creates a new Arabic text shaper
func NewArabicShaper() *ArabicShaper {
	return &ArabicShaper{}
}

// arabicLetterForms maps base Arabic letters to their positional forms
// Format: [isolated, final, initial, medial]
var arabicLetterForms = map[rune][4]rune{
	// Alef family (non-connecting on left)
	'ا': {'\uFE8D', '\uFE8E', '\uFE8D', '\uFE8E'}, // Alef
	'أ': {'\uFE83', '\uFE84', '\uFE83', '\uFE84'}, // Alef with Hamza above
	'إ': {'\uFE87', '\uFE88', '\uFE87', '\uFE88'}, // Alef with Hamza below
	'آ': {'\uFE81', '\uFE82', '\uFE81', '\uFE82'}, // Alef with Madda
	'ى': {'\uFEEF', '\uFEF0', '\uFEEF', '\uFEF0'}, // Alef Maksura
	'ء': {'\uFE80', '\uFE80', '\uFE80', '\uFE80'}, // Hamza (standalone)

	// Ba family
	'ب': {'\uFE8F', '\uFE90', '\uFE91', '\uFE92'}, // Ba
	'ت': {'\uFE95', '\uFE96', '\uFE97', '\uFE98'}, // Ta
	'ث': {'\uFE99', '\uFE9A', '\uFE9B', '\uFE9C'}, // Tha
	'ن': {'\uFEE5', '\uFEE6', '\uFEE7', '\uFEE8'}, // Nun
	'ي': {'\uFEF1', '\uFEF2', '\uFEF3', '\uFEF4'}, // Ya

	// Jeem family
	'ج': {'\uFE9D', '\uFE9E', '\uFE9F', '\uFEA0'}, // Jeem
	'ح': {'\uFEA1', '\uFEA2', '\uFEA3', '\uFEA4'}, // Ha
	'خ': {'\uFEA5', '\uFEA6', '\uFEA7', '\uFEA8'}, // Kha

	// Dal family (non-connecting on left)
	'د': {'\uFEA9', '\uFEAA', '\uFEA9', '\uFEAA'}, // Dal
	'ذ': {'\uFEAB', '\uFEAC', '\uFEAB', '\uFEAC'}, // Thal

	// Ra family (non-connecting on left)
	'ر': {'\uFEAD', '\uFEAE', '\uFEAD', '\uFEAE'}, // Ra
	'ز': {'\uFEAF', '\uFEB0', '\uFEAF', '\uFEB0'}, // Zay

	// Seen family
	'س': {'\uFEB1', '\uFEB2', '\uFEB3', '\uFEB4'}, // Seen
	'ش': {'\uFEB5', '\uFEB6', '\uFEB7', '\uFEB8'}, // Sheen

	// Sad family
	'ص': {'\uFEB9', '\uFEBA', '\uFEBB', '\uFEBC'}, // Sad
	'ض': {'\uFEBD', '\uFEBE', '\uFEBF', '\uFEC0'}, // Dad

	// Ta family
	'ط': {'\uFEC1', '\uFEC2', '\uFEC3', '\uFEC4'}, // Ta
	'ظ': {'\uFEC5', '\uFEC6', '\uFEC7', '\uFEC8'}, // Za

	// Ain family
	'ع': {'\uFEC9', '\uFECA', '\uFECB', '\uFECC'}, // Ain
	'غ': {'\uFECD', '\uFECE', '\uFECF', '\uFED0'}, // Ghain

	// Fa family
	'ف': {'\uFED1', '\uFED2', '\uFED3', '\uFED4'}, // Fa
	'ق': {'\uFED5', '\uFED6', '\uFED7', '\uFED8'}, // Qaf

	// Kaf family
	'ك': {'\uFED9', '\uFEDA', '\uFEDB', '\uFEDC'}, // Kaf

	// Lam family
	'ل': {'\uFEDD', '\uFEDE', '\uFEDF', '\uFEE0'}, // Lam

	// Meem family
	'م': {'\uFEE1', '\uFEE2', '\uFEE3', '\uFEE4'}, // Meem

	// Ha family
	'ه': {'\uFEE9', '\uFEEA', '\uFEEB', '\uFEEC'}, // Ha
	'ة': {'\uFE93', '\uFE94', '\uFE93', '\uFE94'}, // Ta Marbuta (non-connecting)

	// Waw (non-connecting on left)
	'و': {'\uFEED', '\uFEEE', '\uFEED', '\uFEEE'}, // Waw
	'ؤ': {'\uFE85', '\uFE86', '\uFE85', '\uFE86'}, // Waw with Hamza
}

// Non-connecting letters (don't connect to following letter)
var nonConnectingRight = map[rune]bool{
	'ا': true, 'أ': true, 'إ': true, 'آ': true, 'ى': true,
	'د': true, 'ذ': true, 'ر': true, 'ز': true, 'و': true,
	'ؤ': true, 'ء': true, 'ة': true,
}

// Shape converts Arabic text to presentation forms
func (as *ArabicShaper) Shape(text string) string {
	runes := []rune(text)
	result := make([]rune, 0, len(runes))

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		forms, isArabic := arabicLetterForms[r]
		if !isArabic {
			result = append(result, r)
			continue
		}

		prevConnects := i > 0 && canConnectRight(runes[i-1])
		nextConnects := i < len(runes)-1 && canConnectLeft(runes[i+1])
		thisConnectsRight := !nonConnectingRight[r]

		var form rune
		if !prevConnects && !nextConnects {
			form = forms[0] // Isolated
		} else if !prevConnects && nextConnects && thisConnectsRight {
			form = forms[2] // Initial
		} else if prevConnects && !nextConnects {
			form = forms[1] // Final
		} else if prevConnects && nextConnects && thisConnectsRight {
			form = forms[3] // Medial
		} else if prevConnects && nextConnects && !thisConnectsRight {
			form = forms[1] // Final (this letter doesn't connect right)
		} else {
			form = forms[0] // Fallback to isolated
		}

		result = append(result, form)
	}

	return reverseString(string(result))
}

func canConnectRight(r rune) bool {
	_, isArabic := arabicLetterForms[r]
	if !isArabic {
		return false
	}
	return !nonConnectingRight[r]
}

func canConnectLeft(r rune) bool {
	_, isArabic := arabicLetterForms[r]
	return isArabic
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// ShapeForPDF shapes Arabic text and handles mixed content
func (as *ArabicShaper) ShapeForPDF(text string) string {
	var result strings.Builder
	var currentRun strings.Builder
	var isCurrentArabic bool = false
	var hasStarted bool = false

	for _, r := range text {
		isArabic := IsArabicChar(r)

		if !hasStarted {
			hasStarted = true
			isCurrentArabic = isArabic
		}

		if isArabic == isCurrentArabic {
			currentRun.WriteRune(r)
		} else {
			if isCurrentArabic {
				result.WriteString(as.Shape(currentRun.String()))
			} else {
				result.WriteString(currentRun.String())
			}
			currentRun.Reset()
			currentRun.WriteRune(r)
			isCurrentArabic = isArabic
		}
	}

	if currentRun.Len() > 0 {
		if isCurrentArabic {
			result.WriteString(as.Shape(currentRun.String()))
		} else {
			result.WriteString(currentRun.String())
		}
	}

	return result.String()
}

// IsArabicChar checks if a rune is Arabic
func IsArabicChar(r rune) bool {
	return (r >= 0x0600 && r <= 0x06FF) ||
		(r >= 0x0750 && r <= 0x077F) ||
		(r >= 0xFB50 && r <= 0xFDFF) ||
		(r >= 0xFE70 && r <= 0xFEFF)
}

// IsArabicText checks if text contains any Arabic characters
func IsArabicText(text string) bool {
	for _, r := range text {
		if IsArabicChar(r) {
			return true
		}
	}
	return false
}

// ReverseArabicLine reverses an entire line for RTL display
func ReverseArabicLine(text string) string {
	if !IsArabicText(text) {
		return text
	}
	return reverseString(text)
}

// FormatArabicNumber formats a number with Arabic-Indic digits
func FormatArabicNumber(num float64, decimals int) string {
	arabicDigits := map[rune]rune{
		'0': '٠', '1': '١', '2': '٢', '3': '٣', '4': '٤',
		'5': '٥', '6': '٦', '7': '٧', '8': '٨', '9': '٩',
	}

	formatted := formatSimpleNumber(num, decimals)

	var result strings.Builder
	for _, r := range formatted {
		if arabic, ok := arabicDigits[r]; ok {
			result.WriteRune(arabic)
		} else if r == ',' {
			result.WriteRune('٬')
		} else if r == '.' {
			result.WriteRune('٫')
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func formatSimpleNumber(num float64, decimals int) string {
	if decimals == 0 {
		return formatInt(int64(num))
	}

	intPart := int64(num)
	fracPart := num - float64(intPart)
	if fracPart < 0 {
		fracPart = -fracPart
	}

	multiplier := 1.0
	for i := 0; i < decimals; i++ {
		multiplier *= 10
	}
	fracInt := int64(fracPart*multiplier + 0.5)

	fracStr := formatInt(fracInt)
	for len(fracStr) < decimals {
		fracStr = "0" + fracStr
	}

	return formatInt(intPart) + "." + fracStr
}

func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}

	negative := n < 0
	if negative {
		n = -n
	}

	var digits []byte
	count := 0
	for n > 0 {
		if count > 0 && count%3 == 0 {
			digits = append(digits, ',')
		}
		digits = append(digits, byte('0'+n%10))
		n /= 10
		count++
	}

	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// IsDigit checks if rune is a digit (Western or Arabic-Indic)
func IsDigit(r rune) bool {
	return unicode.IsDigit(r) || (r >= '٠' && r <= '٩')
}
