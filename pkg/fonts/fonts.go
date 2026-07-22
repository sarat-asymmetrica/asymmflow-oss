// Package fonts embeds the Noto font family used as the primary, deterministic
// font source for every PDF generator in the app. Before this package existed,
// PDF generation probed for fonts on the host filesystem (C:/Windows/Fonts/...,
// /System/Library/Fonts/..., etc.), so output varied by machine and broke
// entirely on a host missing the expected font. Embedding the fonts in the
// binary makes PDF output deterministic across machines and offline-safe.
//
// Fonts are Noto Sans (Latin/Cyrillic/Greek) and Noto Naskh Arabic, both
// licensed under the SIL Open Font License 1.1 (see OFL.txt in this
// directory). Hinted static instances sourced from
// https://github.com/notofonts/NotoSans and
// https://github.com/notofonts/NotoNaskhArabic.
package fonts

import _ "embed"

//go:embed data/NotoSans-Regular.ttf
var notoSansRegular []byte

//go:embed data/NotoSans-Bold.ttf
var notoSansBold []byte

//go:embed data/NotoNaskhArabic-Regular.ttf
var notoNaskhArabicRegular []byte

//go:embed data/NotoNaskhArabic-Bold.ttf
var notoNaskhArabicBold []byte

// NotoSans returns the embedded Noto Sans Regular TTF bytes.
func NotoSans() []byte { return notoSansRegular }

// NotoSansBold returns the embedded Noto Sans Bold TTF bytes.
func NotoSansBold() []byte { return notoSansBold }

// NotoNaskhArabic returns the embedded Noto Naskh Arabic Regular TTF bytes.
func NotoNaskhArabic() []byte { return notoNaskhArabicRegular }

// NotoNaskhArabicBold returns the embedded Noto Naskh Arabic Bold TTF bytes.
func NotoNaskhArabicBold() []byte { return notoNaskhArabicBold }
