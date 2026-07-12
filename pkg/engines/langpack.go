// ═══════════════════════════════════════════════════════════════════════════
// LANGPACK - 9-Language Multi-Language Support for Invoice Generation
//
// SUPPORTED LANGUAGES:
//   en     - English (LTR)
//   ar     - Arabic (RTL) 🔥 Bahrain market
//   zh-CN  - Simplified Chinese (LTR)
//   ja     - Japanese (LTR)
//   th     - Thai (LTR)
//   hi     - Hindi/Devanagari (LTR)
//   ko     - Korean (LTR)
//   he     - Hebrew (RTL)
//   ru     - Russian/Cyrillic (LTR)
//
// Built with LOVE × SIMPLICITY × TRUTH × JOY 🕉️💎⚡
// ═══════════════════════════════════════════════════════════════════════════

package engines

import (
	"fmt"
	"strings"
	"time"
)

// NumberFormat defines number formatting rules per language
type NumberFormat struct {
	Decimal   string // "." or "," or "٫"
	Thousands string // "," or "." or " " or "٬"
	Currency  string // "$", "BHD", "¥", "₹", etc.
}

// LangPackConfig defines complete language configuration
type LangPackConfig struct {
	Code             string            // ISO code: "en", "ar", "zh-CN"
	Name             string            // Display name: "English", "العربية"
	Direction        string            // "ltr" or "rtl"
	FontFamily       string            // "Noto Sans Arabic", "Noto Sans SC"
	FallbackFont     string            // Fallback if primary unavailable
	NumberFormat     NumberFormat      // Number formatting rules
	DateFormat       string            // Date format pattern
	CurrencyPosition string            // "before" or "after"
	Translations     map[string]string // Key-value translation map
}

// LangPack manages multi-language support for invoice generation
type LangPack struct {
	packs       map[string]*LangPackConfig
	defaultLang string
}

// NewLangPack creates a new LangPack with all supported languages
func NewLangPack() *LangPack {
	lp := &LangPack{
		packs:       make(map[string]*LangPackConfig),
		defaultLang: "en",
	}

	// Initialize all 9 language packs
	lp.initializeEnglish()
	lp.initializeArabic()
	lp.initializeChinese()
	lp.initializeJapanese()
	lp.initializeThai()
	lp.initializeHindi()
	lp.initializeKorean()
	lp.initializeHebrew()
	lp.initializeRussian()

	return lp
}

// initializeEnglish - Default LTR language
func (lp *LangPack) initializeEnglish() {
	lp.packs["en"] = &LangPackConfig{
		Code:         "en",
		Name:         "English",
		Direction:    "ltr",
		FontFamily:   "dejavu",
		FallbackFont: "sans-serif",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "BHD",
		},
		DateFormat:       "02-Jan-2006",
		CurrencyPosition: "after",
		Translations: map[string]string{
			"invoice":       "TAX INVOICE",
			"invoiceNo":     "Invoice No.",
			"dated":         "Dated",
			"deliveryNote":  "Delivery Note",
			"paymentTerms":  "Mode/Terms of Payment",
			"buyer":         "Buyer",
			"trn":           "TRN",
			"buyerOrderNo":  "Buyer's Order No.",
			"slNo":          "Sl No.",
			"description":   "Description",
			"quantity":      "Qty",
			"rate":          "Rate",
			"annual":        "Annual",
			"monthly":       "Monthly",
			"vat":           "VAT%",
			"taxable":       "Taxable",
			"total":         "Total",
			"subtotal":      "Subtotal:",
			"outputVAT":     "Output VAT:",
			"grandTotal":    "Total:",
			"amountInWords": "Amount Chargeable (in words)",
			"vatInWords":    "VAT Amount (in words)",
			"only":          "Only.",
			"eoe":           "E. & O.E",
		},
	}
}

// initializeArabic - RTL language for Bahrain market! 🔥
func (lp *LangPack) initializeArabic() {
	lp.packs["ar"] = &LangPackConfig{
		Code:         "ar",
		Name:         "العربية",
		Direction:    "rtl",
		FontFamily:   "arabic", // Will be registered as "arabic" font
		FallbackFont: "Arial",
		NumberFormat: NumberFormat{
			Decimal:   "٫",
			Thousands: "٬",
			Currency:  "د.ب", // Bahraini Dinar in Arabic
		},
		DateFormat:       "02-يناير-2006", // Will need Arabic month translation
		CurrencyPosition: "after",
		Translations: map[string]string{
			"invoice":       "فاتورة ضريبية",
			"invoiceNo":     "رقم الفاتورة",
			"dated":         "التاريخ",
			"deliveryNote":  "إشعار التسليم",
			"paymentTerms":  "شروط الدفع",
			"buyer":         "المشتري",
			"trn":           "الرقم الضريبي",
			"buyerOrderNo":  "رقم طلب المشتري",
			"slNo":          "م.",
			"description":   "الوصف",
			"quantity":      "الكمية",
			"rate":          "السعر",
			"annual":        "سنوي",
			"monthly":       "شهري",
			"vat":           "ضريبة القيمة المضافة%",
			"taxable":       "الخاضع للضريبة",
			"total":         "المجموع",
			"subtotal":      "المجموع الفرعي:",
			"outputVAT":     "ضريبة القيمة المضافة:",
			"grandTotal":    "المجموع الإجمالي:",
			"amountInWords": "المبلغ بالكلمات",
			"vatInWords":    "قيمة الضريبة بالكلمات",
			"only":          "فقط.",
			"eoe":           "الأخطاء والسهو مستثناة",
		},
	}
}

// initializeChinese - Simplified Chinese
func (lp *LangPack) initializeChinese() {
	lp.packs["zh-CN"] = &LangPackConfig{
		Code:         "zh-CN",
		Name:         "简体中文",
		Direction:    "ltr",
		FontFamily:   "chinese",
		FallbackFont: "SimSun",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "第纳尔", // Dinar in Chinese
		},
		DateFormat:       "2006年01月02日",
		CurrencyPosition: "before",
		Translations: map[string]string{
			"invoice":       "税务发票",
			"invoiceNo":     "发票编号",
			"dated":         "日期",
			"deliveryNote":  "交货单",
			"paymentTerms":  "付款条件",
			"buyer":         "买方",
			"trn":           "税务登记号",
			"buyerOrderNo":  "买方订单号",
			"slNo":          "序号",
			"description":   "描述",
			"quantity":      "数量",
			"rate":          "单价",
			"annual":        "年度",
			"monthly":       "月度",
			"vat":           "增值税%",
			"taxable":       "应税",
			"total":         "总计",
			"subtotal":      "小计:",
			"outputVAT":     "增值税:",
			"grandTotal":    "总计:",
			"amountInWords": "金额大写",
			"vatInWords":    "税额大写",
			"only":          "整。",
			"eoe":           "错误与遗漏除外",
		},
	}
}

// initializeJapanese - Japanese
func (lp *LangPack) initializeJapanese() {
	lp.packs["ja"] = &LangPackConfig{
		Code:         "ja",
		Name:         "日本語",
		Direction:    "ltr",
		FontFamily:   "japanese",
		FallbackFont: "Yu Gothic",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "ディナール", // Dinar in Japanese
		},
		DateFormat:       "2006年01月02日",
		CurrencyPosition: "before",
		Translations: map[string]string{
			"invoice":       "税務請求書",
			"invoiceNo":     "請求書番号",
			"dated":         "日付",
			"deliveryNote":  "納品書",
			"paymentTerms":  "支払条件",
			"buyer":         "購入者",
			"trn":           "税登録番号",
			"buyerOrderNo":  "注文番号",
			"slNo":          "番号",
			"description":   "品目",
			"quantity":      "数量",
			"rate":          "単価",
			"annual":        "年間",
			"monthly":       "月間",
			"vat":           "付加価値税%",
			"taxable":       "課税対象",
			"total":         "合計",
			"subtotal":      "小計:",
			"outputVAT":     "消費税:",
			"grandTotal":    "総計:",
			"amountInWords": "金額（文字）",
			"vatInWords":    "税額（文字）",
			"only":          "のみ。",
			"eoe":           "誤記・脱落除く",
		},
	}
}

// initializeThai - Thai (complex script)
func (lp *LangPack) initializeThai() {
	lp.packs["th"] = &LangPackConfig{
		Code:         "th",
		Name:         "ไทย",
		Direction:    "ltr",
		FontFamily:   "thai",
		FallbackFont: "Tahoma",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "ดีนาร์", // Dinar in Thai
		},
		DateFormat:       "02-มกราคม-2006",
		CurrencyPosition: "before",
		Translations: map[string]string{
			"invoice":       "ใบกำกับภาษี",
			"invoiceNo":     "เลขที่ใบแจ้งหนี้",
			"dated":         "วันที่",
			"deliveryNote":  "ใบส่งของ",
			"paymentTerms":  "เงื่อนไขการชำระเงิน",
			"buyer":         "ผู้ซื้อ",
			"trn":           "เลขประจำตัวผู้เสียภาษี",
			"buyerOrderNo":  "เลขที่ใบสั่งซื้อ",
			"slNo":          "ลำดับ",
			"description":   "รายการ",
			"quantity":      "จำนวน",
			"rate":          "ราคา",
			"annual":        "รายปี",
			"monthly":       "รายเดือน",
			"vat":           "ภาษีมูลค่าเพิ่ม%",
			"taxable":       "มูลค่าที่ต้องเสียภาษี",
			"total":         "รวม",
			"subtotal":      "ยอดรวม:",
			"outputVAT":     "ภาษีมูลค่าเพิ่ม:",
			"grandTotal":    "รวมทั้งสิ้น:",
			"amountInWords": "จำนวนเงินตัวอักษร",
			"vatInWords":    "ภาษีตัวอักษร",
			"only":          "เท่านั้น。",
			"eoe":           "ยกเว้นข้อผิดพลาด",
		},
	}
}

// initializeHindi - Devanagari script
func (lp *LangPack) initializeHindi() {
	lp.packs["hi"] = &LangPackConfig{
		Code:         "hi",
		Name:         "हिन्दी",
		Direction:    "ltr",
		FontFamily:   "hindi",
		FallbackFont: "Arial",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "दीनार", // Dinar in Hindi
		},
		DateFormat:       "02-जनवरी-2006",
		CurrencyPosition: "before",
		Translations: map[string]string{
			"invoice":       "कर चालान",
			"invoiceNo":     "चालान संख्या",
			"dated":         "तिथि",
			"deliveryNote":  "डिलीवरी नोट",
			"paymentTerms":  "भुगतान की शर्तें",
			"buyer":         "क्रेता",
			"trn":           "कर पंजीकरण संख्या",
			"buyerOrderNo":  "क्रेता का आदेश संख्या",
			"slNo":          "क्र.सं.",
			"description":   "विवरण",
			"quantity":      "मात्रा",
			"rate":          "दर",
			"annual":        "वार्षिक",
			"monthly":       "मासिक",
			"vat":           "वैट%",
			"taxable":       "कर योग्य",
			"total":         "कुल",
			"subtotal":      "उप-योग:",
			"outputVAT":     "वैट:",
			"grandTotal":    "कुल योग:",
			"amountInWords": "राशि शब्दों में",
			"vatInWords":    "कर राशि शब्दों में",
			"only":          "केवल।",
			"eoe":           "त्रुटियां और चूक को छोड़कर",
		},
	}
}

// initializeKorean - Hangul script
func (lp *LangPack) initializeKorean() {
	lp.packs["ko"] = &LangPackConfig{
		Code:         "ko",
		Name:         "한국어",
		Direction:    "ltr",
		FontFamily:   "korean",
		FallbackFont: "Malgun Gothic",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "디나르", // Dinar in Korean
		},
		DateFormat:       "2006년 01월 02일",
		CurrencyPosition: "before",
		Translations: map[string]string{
			"invoice":       "세금 송장",
			"invoiceNo":     "송장 번호",
			"dated":         "날짜",
			"deliveryNote":  "배송 노트",
			"paymentTerms":  "결제 조건",
			"buyer":         "구매자",
			"trn":           "세금 등록 번호",
			"buyerOrderNo":  "구매 주문 번호",
			"slNo":          "번호",
			"description":   "설명",
			"quantity":      "수량",
			"rate":          "요율",
			"annual":        "연간",
			"monthly":       "월간",
			"vat":           "부가가치세%",
			"taxable":       "과세 대상",
			"total":         "합계",
			"subtotal":      "소계:",
			"outputVAT":     "부가가치세:",
			"grandTotal":    "총계:",
			"amountInWords": "금액 (문자)",
			"vatInWords":    "세액 (문자)",
			"only":          "만.",
			"eoe":           "오류 및 누락 제외",
		},
	}
}

// initializeHebrew - RTL language #2
func (lp *LangPack) initializeHebrew() {
	lp.packs["he"] = &LangPackConfig{
		Code:         "he",
		Name:         "עברית",
		Direction:    "rtl",
		FontFamily:   "hebrew",
		FallbackFont: "Arial",
		NumberFormat: NumberFormat{
			Decimal:   ".",
			Thousands: ",",
			Currency:  "דינר", // Dinar in Hebrew
		},
		DateFormat:       "02-ינואר-2006",
		CurrencyPosition: "before",
		Translations: map[string]string{
			"invoice":       "חשבונית מס",
			"invoiceNo":     "מספר חשבונית",
			"dated":         "תאריך",
			"deliveryNote":  "תעודת משלוח",
			"paymentTerms":  "תנאי תשלום",
			"buyer":         "קונה",
			"trn":           "מספר רישום מס",
			"buyerOrderNo":  "מספר הזמנת קונה",
			"slNo":          "מס'",
			"description":   "תיאור",
			"quantity":      "כמות",
			"rate":          "תעריף",
			"annual":        "שנתי",
			"monthly":       "חודשי",
			"vat":           "מע״ם%",
			"taxable":       "חייב במס",
			"total":         "סה״כ",
			"subtotal":      "סכום ביניים:",
			"outputVAT":     "מע״ם:",
			"grandTotal":    "סה״כ כולל:",
			"amountInWords": "סכום במילים",
			"vatInWords":    "מע״ם במילים",
			"only":          "בלבד.",
			"eoe":           "טעויות ושגיאות חריגות",
		},
	}
}

// initializeRussian - Cyrillic script
func (lp *LangPack) initializeRussian() {
	lp.packs["ru"] = &LangPackConfig{
		Code:         "ru",
		Name:         "Русский",
		Direction:    "ltr",
		FontFamily:   "dejavu", // DejaVu supports Cyrillic
		FallbackFont: "Arial",
		NumberFormat: NumberFormat{
			Decimal:   ",",
			Thousands: " ",
			Currency:  "динар", // Dinar in Russian
		},
		DateFormat:       "02.01.2006",
		CurrencyPosition: "after",
		Translations: map[string]string{
			"invoice":       "Налоговый счет",
			"invoiceNo":     "Номер счета",
			"dated":         "Дата",
			"deliveryNote":  "Накладная",
			"paymentTerms":  "Условия оплаты",
			"buyer":         "Покупатель",
			"trn":           "Налоговый номер",
			"buyerOrderNo":  "Номер заказа",
			"slNo":          "№",
			"description":   "Описание",
			"quantity":      "Кол-во",
			"rate":          "Цена",
			"annual":        "Годовой",
			"monthly":       "Месячный",
			"vat":           "НДС%",
			"taxable":       "Налогооблагаемая",
			"total":         "Итого",
			"subtotal":      "Промежуточный итог:",
			"outputVAT":     "НДС:",
			"grandTotal":    "Всего:",
			"amountInWords": "Сумма прописью",
			"vatInWords":    "НДС прописью",
			"only":          "Только.",
			"eoe":           "Ошибки и пропуски исключены",
		},
	}
}

// Get retrieves a language pack by code
func (lp *LangPack) Get(code string) *LangPackConfig {
	if pack, exists := lp.packs[code]; exists {
		return pack
	}
	// Fallback to default
	return lp.packs[lp.defaultLang]
}

// Translate returns translated string for key in given language
func (lp *LangPack) Translate(langCode, key string) string {
	pack := lp.Get(langCode)
	if translation, exists := pack.Translations[key]; exists {
		return translation
	}
	// Fallback to key itself
	return key
}

// FormatNumber formats a number according to language rules
func (lp *LangPack) FormatNumber(langCode string, value float64, isCurrency bool) string {
	pack := lp.Get(langCode)

	// Format with 3 decimal places for BHD (fils precision)
	formatted := fmt.Sprintf("%.3f", value)

	// Split into integer and decimal parts
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	decPart := ""
	if len(parts) > 1 {
		decPart = parts[1]
	}

	// Add thousands separators
	if len(intPart) > 3 {
		// Insert thousands separator from right to left
		result := ""
		for i, c := range reverseString(intPart) {
			if i > 0 && i%3 == 0 {
				result = pack.NumberFormat.Thousands + result
			}
			result = string(c) + result
		}
		intPart = result
	}

	// Combine with decimal separator
	finalNumber := intPart
	if decPart != "" {
		finalNumber = intPart + pack.NumberFormat.Decimal + decPart
	}

	// Add currency if requested
	if isCurrency {
		if pack.CurrencyPosition == "before" {
			return pack.NumberFormat.Currency + " " + finalNumber
		}
		return finalNumber + " " + pack.NumberFormat.Currency
	}

	return finalNumber
}

// FormatDate formats date according to language rules
// IMPORTANT: Guards against DateTime.MinValue (0001-01-01) bug!
func (lp *LangPack) FormatDate(langCode string, date time.Time) string {
	// Guard against zero/default dates (the 01.01.0001 bug!)
	if date.IsZero() || date.Year() < 1900 {
		return "—" // Em-dash for missing dates
	}

	pack := lp.Get(langCode)

	// For Arabic, use Arabic month names
	if langCode == "ar" {
		return lp.formatArabicDate(date)
	}

	// For now, use the date format pattern as-is
	return date.Format(pack.DateFormat)
}

// formatArabicDate formats date with Arabic month names
func (lp *LangPack) formatArabicDate(date time.Time) string {
	arabicMonths := []string{
		"يناير", "فبراير", "مارس", "أبريل", "مايو", "يونيو",
		"يوليو", "أغسطس", "سبتمبر", "أكتوبر", "نوفمبر", "ديسمبر",
	}
	day := date.Day()
	month := arabicMonths[date.Month()-1]
	year := date.Year()
	return fmt.Sprintf("%d %s %d", day, month, year)
}

// IsRTL returns true if language is right-to-left
func (lp *LangPack) IsRTL(langCode string) bool {
	pack := lp.Get(langCode)
	return pack.Direction == "rtl"
}

// GetAvailableLanguages returns list of all supported languages
func (lp *LangPack) GetAvailableLanguages() []struct{ Code, Name string } {
	result := make([]struct{ Code, Name string }, 0, len(lp.packs))
	for code, pack := range lp.packs {
		result = append(result, struct{ Code, Name string }{
			Code: code,
			Name: pack.Name,
		})
	}
	return result
}

// Note: reverseString is defined in arabic_shaper.go and shared across the package
