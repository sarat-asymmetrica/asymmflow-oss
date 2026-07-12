package main

import (
	"strings"

	"github.com/jung-kurt/gofpdf"

	"ph_holdings_app/pkg/overlay"
)

// This file ports the deployed PH offer/credit-note signature blocks. The PDF
// geometry, font styling, and line ordering are reproduced byte-for-byte in
// appearance; the only substantive difference is that staff identities are
// NOT hardcoded here — they come from the active company overlay
// (activeOverlay.SignatureBlocks), which ships synthetic-canon defaults and is
// overridden with real identities via the sovereign overlay.json at deploy
// time. See pkg/overlay.SignatureBlockProfile.

// OfferSignatureBlock is the identity printed as a document's "Best Regards"
// block. It aliases the overlay profile so there is a single source of truth
// for the field set and the render order.
type OfferSignatureBlock = overlay.SignatureBlockProfile

// resolveOfferSignatureBlock returns the configured signature block matching
// preparedBy (by DisplayName or alias) and whether a match was found.
func resolveOfferSignatureBlock(preparedBy string) (OfferSignatureBlock, bool) {
	return activeOverlay.SignatureBlockFor(preparedBy)
}

// defaultOfferSignatureNames lists the DisplayName of every configured
// signature block (used by collaboration/assignment surfaces).
func defaultOfferSignatureNames() []string {
	return activeOverlay.SignatureNames()
}

// offerIssuerDisplayName canonicalises a prepared-by/issuer string to the
// matched block's DisplayName (so an alias like "Sam" prints as "Sam Rivera"),
// falling back to the trimmed input when nothing matches.
func offerIssuerDisplayName(preparedBy string) string {
	if block, ok := resolveOfferSignatureBlock(preparedBy); ok {
		return block.DisplayName
	}
	return strings.TrimSpace(preparedBy)
}

// signaturePDFLine is one rendered line of a signature block.
type signaturePDFLine struct {
	Text string
	Bold bool
}

// offerSignaturePDFLines builds the ordered lines for a signature block: an
// optional "Best Regards," lead-in, then the bold DisplayName, Title, Company,
// address lines, prefixed Mobile/Office/Fax, and Email. Empty fields are
// skipped. Reproduces the deployed PH ordering exactly.
func offerSignaturePDFLines(block OfferSignatureBlock, includeRegards bool) []signaturePDFLine {
	lines := []signaturePDFLine{}
	if includeRegards {
		lines = append(lines,
			signaturePDFLine{Text: "Best Regards,"},
			signaturePDFLine{},
		)
	}
	lines = append(lines, signaturePDFLine{Text: block.DisplayName, Bold: true})
	for _, value := range []string{block.Title, block.Company} {
		if strings.TrimSpace(value) != "" {
			lines = append(lines, signaturePDFLine{Text: value})
		}
	}
	for _, line := range block.AddressLines {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, signaturePDFLine{Text: line})
		}
	}
	if strings.TrimSpace(block.Mobile) != "" {
		lines = append(lines, signaturePDFLine{Text: "Mob: " + block.Mobile})
	}
	if strings.TrimSpace(block.Office) != "" {
		lines = append(lines, signaturePDFLine{Text: "Office: " + block.Office})
	}
	if strings.TrimSpace(block.Fax) != "" {
		lines = append(lines, signaturePDFLine{Text: "Fax: " + block.Fax})
	}
	if strings.TrimSpace(block.Email) != "" {
		lines = append(lines, signaturePDFLine{Text: block.Email})
	}
	return lines
}

// drawSignaturePDFLines renders a signature block into a gofpdf document at
// (x, y) with the given column width, per-line height, and font size. It
// returns the Y coordinate after the last line. Geometry, colours, and bold
// handling are byte-for-byte identical to the deployed PH renderer.
func drawSignaturePDFLines(pdf *gofpdf.Fpdf, x, y, width, lineHeight, fontSize float64, block OfferSignatureBlock, includeRegards bool) float64 {
	currentY := y
	for _, line := range offerSignaturePDFLines(block, includeRegards) {
		text := strings.TrimSpace(line.Text)
		if text == "" {
			currentY += lineHeight
			continue
		}
		style := ""
		if line.Bold {
			style = "B"
		}
		pdf.SetFont("Helvetica", style, fontSize)
		pdf.SetTextColor(60, 60, 60)
		if line.Bold {
			pdf.SetTextColor(29, 29, 31)
		}
		pdf.SetXY(x, currentY)
		pdf.MultiCell(width, lineHeight, sanitizeForPDF(text), "", "L", false)
		currentY = pdf.GetY()
	}
	return currentY
}

// resolvePreparedBySignatureBlock resolves a prepared-by/issuer string to a
// fully-populated signature block. A configured block match wins outright;
// otherwise the company-level fallback is returned with the signer's own name
// stamped as DisplayName. Unlike deployed PH there is no per-employee DB
// override here — the overlay is the single source of signer identity.
func (a *App) resolvePreparedBySignatureBlock(preparedBy string) OfferSignatureBlock {
	preparedBy = strings.TrimSpace(preparedBy)
	if block, ok := resolveOfferSignatureBlock(preparedBy); ok {
		return block
	}
	block := activeOverlay.SignatureFallback()
	block.DisplayName = preparedBy
	return block
}

// resolveDocumentSignerName picks the first non-empty candidate signer name,
// resolving a user-id candidate to a display name when it maps to a User row.
// Falls back to the current session's display name when no candidate resolves.
func (a *App) resolveDocumentSignerName(candidates ...string) string {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if a != nil && a.db != nil {
			var user User
			if err := a.db.First(&user, "id = ?", candidate).Error; err == nil {
				for _, value := range []string{user.DisplayName, user.FullName, user.Username, user.Email} {
					if strings.TrimSpace(value) != "" {
						return strings.TrimSpace(value)
					}
				}
			}
		}
		return candidate
	}
	if a != nil {
		return strings.TrimSpace(a.getCurrentUserDisplayName())
	}
	return ""
}
