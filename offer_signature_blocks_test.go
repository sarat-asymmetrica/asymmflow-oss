package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ph_holdings_app/pkg/overlay"
)

// withSyntheticOverlay pins activeOverlay to the built-in synthetic canon for
// the duration of a test (restoring whatever a sibling test may have set),
// so signature resolution is deterministic regardless of test order.
func withSyntheticOverlay(t *testing.T) {
	t.Helper()
	orig := activeOverlay
	activeOverlay = overlay.BuiltinDefaults()
	t.Cleanup(func() { activeOverlay = orig })
}

// pdfText renders raw PDF bytes to text via pdftotext, or skips the assertion
// when the tool is unavailable (mirrors the existing costing PDF regression).
func pdfText(t *testing.T, pdfBytes []byte) (string, bool) {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", false
	}
	f, err := os.CreateTemp(t.TempDir(), "sig-*.pdf")
	require.NoError(t, err)
	_, err = f.Write(pdfBytes)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	out, err := exec.Command("pdftotext", "-layout", f.Name(), "-").Output()
	require.NoError(t, err)
	return string(out), true
}

func TestResolvePreparedBySignatureBlock_KnownName(t *testing.T) {
	withSyntheticOverlay(t)
	app := &App{}

	block := app.resolvePreparedBySignatureBlock("Sam Rivera")
	assert.Equal(t, "Sam Rivera", block.DisplayName)
	assert.Equal(t, "Business Development Manager", block.Title)
	assert.Equal(t, "ACME INSTRUMENTATION W.L.L", block.Company)
	assert.True(t, strings.HasSuffix(block.Email, ".example"))
}

func TestResolvePreparedBySignatureBlock_AliasCanonicalises(t *testing.T) {
	withSyntheticOverlay(t)
	app := &App{}

	block := app.resolvePreparedBySignatureBlock("Sam")
	assert.Equal(t, "Sam Rivera", block.DisplayName, "an alias must resolve to the canonical identity")
	assert.Equal(t, "Business Development Manager", block.Title)
}

func TestResolvePreparedBySignatureBlock_UnknownFallsBackToCompanyBlock(t *testing.T) {
	withSyntheticOverlay(t)
	app := &App{}

	block := app.resolvePreparedBySignatureBlock("Unlisted Signer")
	assert.Equal(t, "Unlisted Signer", block.DisplayName, "the unmatched signer's own name is stamped on the fallback")
	assert.Equal(t, "ACME INSTRUMENTATION W.L.L", block.Company)
	assert.Equal(t, "sales@acme-instrumentation.example", block.Email)
	assert.Empty(t, block.Title, "the company fallback carries no personal title")
}

func TestOfferIssuerDisplayNameCanonicalises(t *testing.T) {
	withSyntheticOverlay(t)
	assert.Equal(t, "Sam Rivera", offerIssuerDisplayName("Sam"))
	assert.Equal(t, "Sam Rivera", offerIssuerDisplayName("SAM RIVERA"))
	assert.Equal(t, "Someone Else", offerIssuerDisplayName("Someone Else"), "unmatched names pass through trimmed")
	assert.Equal(t, "Trimmed", offerIssuerDisplayName("  Trimmed  "))
}

func TestDefaultOfferSignatureNames(t *testing.T) {
	withSyntheticOverlay(t)
	assert.Equal(t,
		[]string{"Jordan Avery", "Alex Morgan", "Sam Rivera", "Casey Quinn", "Taylor Brooks", "Jamie Ellis"},
		defaultOfferSignatureNames(),
	)
}

// drawSignaturePDFLines must advance the cursor and emit the identity text. A
// wiring regression (e.g. the block silently not drawn) fails the Y-advance
// check without any external tool; the text check is gated on pdftotext.
func TestDrawSignaturePDFLinesRendersIdentity(t *testing.T) {
	withSyntheticOverlay(t)
	app := &App{}
	block := app.resolvePreparedBySignatureBlock("Sam")

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	startY := 40.0
	endY := drawSignaturePDFLines(pdf, 20, startY, 90, 4.4, 8.2, block, true)
	assert.Greater(t, endY, startY, "rendering the block must advance the Y cursor")

	var buf bytes.Buffer
	require.NoError(t, pdf.Output(&buf))
	require.NotEmpty(t, buf.Bytes())

	if text, ok := pdfText(t, buf.Bytes()); ok {
		assert.Contains(t, text, "Best Regards,")
		assert.Contains(t, text, "Sam Rivera")
		assert.Contains(t, text, "sam@acme-instrumentation.example")
	}
}

// The offer PDF export data must carry the CANONICAL issuer name so the info
// table and the signature block agree even when the offer stored an alias.
func TestBuildCostingExportDataFromOfferCanonicalisesIssuer(t *testing.T) {
	withSyntheticOverlay(t)
	offer := Offer{
		Base:     Base{ID: uuid.New().String()},
		IssuedBy: "Sam",
		Division: "Acme Instrumentation",
		VatRate:  10,
		Items:    []OfferItem{{LineNumber: 1, Description: "Line", Quantity: 1, UnitPrice: 100, TotalPrice: 100}},
	}
	data := buildCostingExportDataFromOffer(offer, CustomerMaster{}, CustomerContact{})
	assert.Equal(t, "Sam Rivera", data.PreparedBy)
}

// End-to-end wiring: the offer PDF renders the resolved signature block, and
// the credit note PDF renders the block INSTEAD of the old static
// "Authorized Signatory" footer. Text assertions are gated on pdftotext.
func TestOfferAndCreditNotePDFsRenderSignatureBlock(t *testing.T) {
	withSyntheticOverlay(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&Offer{}, &OfferItem{}, &CreditNote{}, &CreditNoteItem{}))

	now := time.Now()
	customer := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: "AquaPure Technologies",
		CustomerCode: "APT-001",
		CustomerID:   "APT-001",
		TRN:          "990000000000000",
		Status:       "Active",
	}
	require.NoError(t, app.db.Create(&customer).Error)

	offer := Offer{
		Base:          Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OfferNumber:   "OFF-SIG-001",
		CustomerID:    customer.ID,
		CustomerName:  customer.BusinessName,
		QuotationDate: now,
		Stage:         "Quoted",
		IssuedBy:      "Sam", // alias — must canonicalise to "Sam Rivera"
		QuoteType:     "Quotation",
		VatRate:       10,
		Division:      "Acme Instrumentation",
	}
	require.NoError(t, app.db.Create(&offer).Error)
	require.NoError(t, app.db.Create(&OfferItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		OfferID:     offer.ID,
		LineNumber:  1,
		Description: "Flow transmitter",
		Quantity:    1,
		UnitPrice:   250,
		TotalPrice:  250,
	}).Error)

	offerPath, err := app.GenerateOfferPDF(offer.ID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(offerPath) })
	offerBytes, err := os.ReadFile(offerPath)
	require.NoError(t, err)
	require.NotEmpty(t, offerBytes)
	if text, ok := pdfText(t, offerBytes); ok {
		assert.Contains(t, text, "Best Regards,")
		assert.Contains(t, text, "Sam Rivera")
	}

	creditNote := CreditNote{
		Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CreditNoteNumber: "CN-SIG-001",
		CreditNoteDate:   now,
		CustomerID:       customer.ID,
		CustomerName:     customer.BusinessName,
		Reason:           "Adjustment",
		SubtotalBHD:      10,
		VATBHD:           1,
		VATPercent:       10,
		GrandTotalBHD:    11,
		Status:           "Issued",
	}
	require.NoError(t, app.db.Create(&creditNote).Error)
	require.NoError(t, app.db.Create(&CreditNoteItem{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CreditNoteID: creditNote.ID,
		LineNumber:   1,
		Description:  "Credit line",
		Quantity:     1,
		Rate:         10,
		TotalBHD:     10,
	}).Error)

	cnPath, err := app.GenerateCreditNotePDF(creditNote.ID)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(cnPath) })
	cnBytes, err := os.ReadFile(cnPath)
	require.NoError(t, err)
	require.NotEmpty(t, cnBytes)
	if text, ok := pdfText(t, cnBytes); ok {
		assert.NotContains(t, text, "Authorized Signatory", "the static signatory footer must be gone")
		// The signer resolves from the test session identity ("test-admin"),
		// which is stamped onto the company fallback block.
		assert.Contains(t, text, "test-admin")
		assert.Contains(t, text, "ACME INSTRUMENTATION W.L.L")
	}
}
