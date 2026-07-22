package main

// gstr1_export_service_test.go
//
// India Spec-01 B5 tests: the GSTR-1 portal-JSON export. Fixtures reuse the
// canon India demo overlay (overlays/india-demo -- Meridian Instruments &
// Controls Pvt Ltd, two divisions/states) and the withIndiaOverlay/
// migrateIndiaDocTables/synthGSTIN helpers B4 already built in
// india_documents_test.go (same package, same fixtures -- SYNTHETIC_IDENTITY.md
// "India demo canon"). All names/GSTINs fictional.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// createIndiaCustomer seeds a minimal India-side customer for GSTR-1 fixtures.
func createIndiaCustomer(t *testing.T, app *App, name string) CustomerMaster {
	t.Helper()
	now := time.Now()
	c := CustomerMaster{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		BusinessName: name,
		CustomerCode: "GSTR1-" + uuid.New().String()[:8],
		CustomerID:   "GSTR1-" + uuid.New().String()[:8],
		Country:      "India",
		Status:       "Active",
	}
	require.NoError(t, app.db.Create(&c).Error)
	return c
}

// gstr1InvoiceOpts is the fixture shape for createGSTR1Invoice -- every field
// a GSTR-1 test scenario needs to control, with sensible zero-value defaults
// (Status "Sent", GrandTotal = qty*rate when left 0).
type gstr1InvoiceOpts struct {
	Number       string
	Date         time.Time
	Division     string
	BuyerGSTIN   string
	PosStateCode string
	HSN          string
	UQC          string
	Description  string
	Qty          float64
	Rate         float64
	GrandTotal   float64
	Status       string
}

func createGSTR1Invoice(t *testing.T, app *App, customer CustomerMaster, opts gstr1InvoiceOpts) Invoice {
	t.Helper()
	status := opts.Status
	if status == "" {
		status = "Sent"
	}
	desc := opts.Description
	if desc == "" {
		desc = "Test line"
	}
	taxable := opts.Qty * opts.Rate
	grandTotal := opts.GrandTotal
	if grandTotal == 0 {
		grandTotal = taxable
	}

	inv := Invoice{
		Base:                   Base{ID: uuid.New().String(), CreatedAt: opts.Date, UpdatedAt: opts.Date},
		InvoiceNumber:          opts.Number,
		InvoiceDate:            opts.Date,
		DueDate:                opts.Date.AddDate(0, 0, 30),
		CustomerID:             customer.ID,
		CustomerName:           customer.BusinessName,
		Status:                 status,
		Division:               opts.Division,
		BuyerGSTIN:             opts.BuyerGSTIN,
		PlaceOfSupplyStateCode: opts.PosStateCode,
		SubtotalBHD:            taxable,
		GrandTotalBHD:          grandTotal,
		OutstandingBHD:         grandTotal,
	}
	require.NoError(t, app.db.Create(&inv).Error)
	require.NoError(t, app.db.Create(&DBInvoiceItem{
		Base:        Base{ID: uuid.New().String(), CreatedAt: opts.Date, UpdatedAt: opts.Date},
		InvoiceID:   inv.ID,
		LineNumber:  1,
		Description: desc,
		Quantity:    opts.Qty,
		Rate:        opts.Rate,
		TotalBHD:    taxable,
		HSNCode:     opts.HSN,
		UQC:         opts.UQC,
	}).Error)
	return inv
}

// readGSTR1Payload reads and decodes an exported GSTR-1 JSON file into the
// package's own (unexported) payload shape -- fine from an in-package test.
func readGSTR1Payload(t *testing.T, path string) gstr1Payload {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var payload gstr1Payload
	require.NoError(t, json.Unmarshal(data, &payload))
	return payload
}

// findGSTR1File picks the one file among files whose name embeds gstin. The
// india-demo overlay mounts TWO divisions (Meridian Mumbai + Bengaluru)
// sharing one PAN, so ExportGSTR1JSON always writes one file per division --
// a registered GSTIN files a return even in a nil period, mirroring
// ExportVATReturnData's per-division-TRN law (Wave 12.5). Most scenarios
// below only care about Mumbai's file.
func findGSTR1File(t *testing.T, files []string, gstin string) string {
	t.Helper()
	for _, f := range files {
		if strings.Contains(f, gstin) {
			return f
		}
	}
	t.Fatalf("no GSTR-1 file found for GSTIN %s among %v", gstin, files)
	return ""
}

const meridianMumbaiGSTIN = "27AABCM0472E1ZT"

// TestGSTR1Export_B2BIntraStateSplitsCGSTSGST covers the B2B intra-state
// happy path: Meridian Mumbai (state 27) selling to a Maharashtra B2B buyer
// splits tax CGST+SGST, never IGST, and the invoice lands under its buyer's
// ctin in the b2b section.
func TestGSTR1Export_B2BIntraStateSplitsCGSTSGST(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	customer := createIndiaCustomer(t, app, "Sahyadri Process Equipment Pvt Ltd")
	buyerGSTIN := synthGSTIN(t, "27")
	createGSTR1Invoice(t, app, customer, gstr1InvoiceOpts{
		Number: "INV/26-27/101", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: buyerGSTIN, PosStateCode: "27",
		HSN: "9026", UQC: "NOS", Description: "Flow transmitter",
		Qty: 2, Rate: 5000, GrandTotal: 11800,
	})

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)
	require.Len(t, files, 2, "one file per India-mounted division (Mumbai + Bengaluru)")

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Equal(t, "27AABCM0472E1ZT", payload.GSTIN)
	require.Equal(t, "062026", payload.FP)
	require.Equal(t, "GST3.2.4", payload.Version)

	require.Len(t, payload.B2B, 1)
	require.Equal(t, buyerGSTIN, payload.B2B[0].Ctin)
	require.Len(t, payload.B2B[0].Inv, 1)
	inv := payload.B2B[0].Inv[0]
	require.Equal(t, "INV/26-27/101", inv.Inum)
	require.Equal(t, "15-06-2026", inv.Idt)
	require.Equal(t, "27", inv.Pos)
	require.Equal(t, "N", inv.Rchrg)
	require.Equal(t, "R", inv.InvTyp)
	require.Equal(t, 11800.0, inv.Val)
	require.Len(t, inv.Itms, 1)
	require.Equal(t, gstr1ItemDet{Rt: 18, Txval: 10000, Iamt: 0, Camt: 900, Samt: 900, Csamt: 0}, inv.Itms[0].ItmDet)

	require.Empty(t, payload.B2CL)
	require.Empty(t, payload.B2CS)
	require.Empty(t, payload.CDNR)
	require.Empty(t, payload.CDNUR)
}

// TestGSTR1Export_B2BInterStateIGST is the inter-state twin: Meridian Mumbai
// (state 27) selling to Charminar Engineering Co, Telangana (state 36),
// splits tax as IGST only.
func TestGSTR1Export_B2BInterStateIGST(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	customer := createIndiaCustomer(t, app, "Charminar Engineering Co")
	buyerGSTIN := synthGSTIN(t, "36")
	createGSTR1Invoice(t, app, customer, gstr1InvoiceOpts{
		Number: "INV/26-27/102", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: buyerGSTIN, PosStateCode: "36",
		HSN: "8481", UQC: "NOS", Description: "Gate valve",
		Qty: 1, Rate: 10000, GrandTotal: 11800,
	})

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Len(t, payload.B2B, 1)
	require.Equal(t, buyerGSTIN, payload.B2B[0].Ctin)
	itms := payload.B2B[0].Inv[0].Itms
	require.Len(t, itms, 1)
	require.Equal(t, gstr1ItemDet{Rt: 18, Txval: 10000, Iamt: 1800, Camt: 0, Samt: 0, Csamt: 0}, itms[0].ItmDet)
}

// TestGSTR1Export_B2CSmallIntraAggregatesToB2CS covers an intra-state B2C
// (no BuyerGSTIN) supply: it must aggregate into b2cs, never b2b/b2cl,
// regardless of value (b2cl requires inter-state).
func TestGSTR1Export_B2CSmallIntraAggregatesToB2CS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC)
	customer := createIndiaCustomer(t, app, "Walk-in Consumer")
	createGSTR1Invoice(t, app, customer, gstr1InvoiceOpts{
		Number: "INV/26-27/103", Date: now, Division: "Meridian Mumbai",
		PosStateCode: "27", // intra, no BuyerGSTIN
		HSN:          "8413", UQC: "NOS", Description: "Pump (walk-in sale)",
		Qty: 1, Rate: 2000, GrandTotal: 2360,
	})

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Empty(t, payload.B2B)
	require.Empty(t, payload.B2CL)
	require.Len(t, payload.B2CS, 1)
	row := payload.B2CS[0]
	require.Equal(t, "INTRA", row.SplyTy)
	require.Equal(t, "27", row.Pos)
	require.Equal(t, "OE", row.Typ)
	require.Equal(t, 18.0, row.Rt)
	require.Equal(t, 2000.0, row.Txval)
	require.Equal(t, 180.0, row.Camt)
	require.Equal(t, 180.0, row.Samt)
	require.Equal(t, 0.0, row.Iamt)
}

// TestGSTR1Export_B2CInterAboveThreshold_IsB2CL_BelowThreshold_IsB2CS covers
// the B2CL/B2CS split on the configured threshold (default Rs.1,00,000,
// R-A4-1): an inter-state B2C invoice above the threshold is invoice-wise
// b2cl; below it, it folds into the b2cs aggregate.
func TestGSTR1Export_B2CInterAboveThreshold_IsB2CL_BelowThreshold_IsB2CS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	customerHigh := createIndiaCustomer(t, app, "Walk-in Consumer High")
	createGSTR1Invoice(t, app, customerHigh, gstr1InvoiceOpts{
		Number: "INV/26-27/104", Date: now, Division: "Meridian Mumbai",
		PosStateCode: "36", // inter-state, no BuyerGSTIN
		HSN:          "8481", UQC: "NOS", Description: "Gate valve (large B2C)",
		Qty: 1, Rate: 150000, GrandTotal: 177000, // 150000 taxable > 100000 threshold
	})
	customerLow := createIndiaCustomer(t, app, "Walk-in Consumer Low")
	createGSTR1Invoice(t, app, customerLow, gstr1InvoiceOpts{
		Number: "INV/26-27/105", Date: now, Division: "Meridian Mumbai",
		PosStateCode: "36",
		HSN:          "8481", UQC: "NOS", Description: "Gate valve (small B2C)",
		Qty: 1, Rate: 20000, GrandTotal: 23600, // 20000 taxable < 100000 threshold
	})

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Empty(t, payload.B2B)
	// Gate ruling: b2cl is pos-grouped (like b2b's ctin grouping) -- one
	// group per place-of-supply state, invoices inside.
	require.Len(t, payload.B2CL, 1, "the above-threshold invoice must be invoice-wise b2cl (one pos group)")
	require.Equal(t, "36", payload.B2CL[0].Pos)
	require.Len(t, payload.B2CL[0].Inv, 1)
	require.Equal(t, "INV/26-27/104", payload.B2CL[0].Inv[0].Inum)
	require.Equal(t, 177000.0, payload.B2CL[0].Inv[0].Val)
	require.Equal(t, gstr1ItemDet{Rt: 18, Txval: 150000, Iamt: 27000, Camt: 0, Samt: 0, Csamt: 0}, payload.B2CL[0].Inv[0].Itms[0].ItmDet)

	require.Len(t, payload.B2CS, 1, "the below-threshold invoice must aggregate into b2cs, not b2cl")
	require.Equal(t, "INTER", payload.B2CS[0].SplyTy)
	require.Equal(t, 20000.0, payload.B2CS[0].Txval)
	require.Equal(t, 3600.0, payload.B2CS[0].Iamt)
}

// TestGSTR1Export_AppliedCreditNoteBecomesCDNR covers CDNR: an Applied
// credit note against a B2B India invoice lands under its buyer's ctin, with
// the original invoice's number/date carried as the note's reference.
func TestGSTR1Export_AppliedCreditNoteBecomesCDNR(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	customer := createIndiaCustomer(t, app, "Sahyadri Process Equipment Pvt Ltd")
	buyerGSTIN := synthGSTIN(t, "27")
	invoice := createGSTR1Invoice(t, app, customer, gstr1InvoiceOpts{
		Number: "INV/26-27/106", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: buyerGSTIN, PosStateCode: "27",
		HSN: "9026", UQC: "NOS", Description: "Flow transmitter",
		Qty: 2, Rate: 5000, GrandTotal: 11800,
	})

	appliedAt := now
	cn := CreditNote{
		Base:             Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CreditNoteNumber: "CN/26-27/101",
		CreditNoteDate:   now,
		InvoiceID:        invoice.ID,
		InvoiceNumber:    invoice.InvoiceNumber,
		CustomerID:       customer.ID,
		CustomerName:     customer.BusinessName,
		Reason:           "Partial return",
		SubtotalBHD:      5000,
		GrandTotalBHD:    5900,
		Status:           "Applied",
		AppliedAt:        &appliedAt,
		Division:         invoice.Division,
	}
	require.NoError(t, app.db.Create(&cn).Error)
	require.NoError(t, app.db.Create(&CreditNoteItem{
		Base:         Base{ID: uuid.New().String(), CreatedAt: now, UpdatedAt: now},
		CreditNoteID: cn.ID, LineNumber: 1,
		Description: "Flow transmitter (returned unit)", Quantity: 1, Rate: 5000, TotalBHD: 5000,
		HSNCode: "9026", UQC: "NOS",
	}).Error)

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Len(t, payload.CDNR, 1)
	require.Equal(t, buyerGSTIN, payload.CDNR[0].Ctin)
	require.Len(t, payload.CDNR[0].Nt, 1)
	note := payload.CDNR[0].Nt[0]
	require.Equal(t, "C", note.Ntty)
	require.Equal(t, "CN/26-27/101", note.NtNum)
	require.Equal(t, "19-06-2026", note.NtDt)
	require.Equal(t, "27", note.Pos)
	require.Equal(t, 5900.0, note.Val)
	require.Equal(t, invoice.InvoiceNumber, note.Inum, "note must reference the original invoice number")
	require.Equal(t, "19-06-2026", note.Idt, "note must reference the original invoice date")
	require.Empty(t, payload.CDNUR)
}

// TestGSTR1Export_HSNSummarySplitsAndAggregates covers Table 12 (R-A4-3):
// two B2B invoices sharing an (HSN, UQC, rate) must collapse into ONE
// hsn_b2b row with summed qty/txval; a B2C invoice's HSN feeds hsn_b2c
// instead, never hsn_b2b.
func TestGSTR1Export_HSNSummarySplitsAndAggregates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 20, 10, 0, 0, 0, time.UTC)
	buyerGSTIN := synthGSTIN(t, "27")

	c1 := createIndiaCustomer(t, app, "Sahyadri Process Equipment Pvt Ltd")
	createGSTR1Invoice(t, app, c1, gstr1InvoiceOpts{
		Number: "INV/26-27/107", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: buyerGSTIN, PosStateCode: "27",
		HSN: "9026", UQC: "NOS", Description: "Flow transmitter",
		Qty: 2, Rate: 5000, GrandTotal: 11800,
	})
	c2 := createIndiaCustomer(t, app, "Sahyadri Process Equipment Pvt Ltd (2nd order)")
	createGSTR1Invoice(t, app, c2, gstr1InvoiceOpts{
		Number: "INV/26-27/108", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: buyerGSTIN, PosStateCode: "27",
		HSN: "9026", UQC: "NOS", Description: "Flow transmitter",
		Qty: 3, Rate: 5000, GrandTotal: 17700,
	})
	c3 := createIndiaCustomer(t, app, "Walk-in Consumer")
	createGSTR1Invoice(t, app, c3, gstr1InvoiceOpts{
		Number: "INV/26-27/109", Date: now, Division: "Meridian Mumbai",
		PosStateCode: "27", // B2C, no BuyerGSTIN
		HSN:          "8413", UQC: "NOS", Description: "Pump (walk-in sale)",
		Qty: 1, Rate: 2000, GrandTotal: 2360,
	})

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Len(t, payload.HSNB2B, 1, "the two B2B 9026@18%% invoices must aggregate into one row")
	require.Equal(t, "9026", payload.HSNB2B[0].HSNSC)
	require.Equal(t, "NOS", payload.HSNB2B[0].UQC)
	require.Equal(t, 18.0, payload.HSNB2B[0].Rt)
	require.Equal(t, 5.0, payload.HSNB2B[0].Qty, "2 + 3 units")
	require.Equal(t, 25000.0, payload.HSNB2B[0].Txval, "10000 + 15000 taxable")

	require.Len(t, payload.HSNB2C, 1)
	require.Equal(t, "8413", payload.HSNB2C[0].HSNSC)
	require.Equal(t, 1.0, payload.HSNB2C[0].Qty)
}

// TestGSTR1Export_DocIssueFromToCounts covers Table 13: a division's
// invoice-number series in the period summarizes to one from/to/count row.
func TestGSTR1Export_DocIssueFromToCounts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)
	for _, n := range []string{"INV/26-27/201", "INV/26-27/202", "INV/26-27/203"} {
		customer := createIndiaCustomer(t, app, "Doc Issue Customer "+n)
		createGSTR1Invoice(t, app, customer, gstr1InvoiceOpts{
			Number: n, Date: now, Division: "Meridian Mumbai",
			PosStateCode: "27", HSN: "9026", UQC: "NOS", Description: "Flow transmitter",
			Qty: 1, Rate: 1000, GrandTotal: 1180,
		})
	}

	files, err := app.ExportGSTR1JSON(2026, 6)
	require.NoError(t, err)

	payload := readGSTR1Payload(t, findGSTR1File(t, files, meridianMumbaiGSTIN))
	require.Len(t, payload.DocIssue, 1)
	series := payload.DocIssue[0]
	require.Equal(t, 1, series.DocNum, "invoices are Table-13 nature category 1")
	require.Len(t, series.Docs, 1)
	require.Equal(t, "INV/26-27", series.Docs[0].Num)
	require.Equal(t, "INV/26-27/201", series.Docs[0].From)
	require.Equal(t, "INV/26-27/203", series.Docs[0].To)
	require.Equal(t, 3, series.Docs[0].Totnum)
	require.Equal(t, 0, series.Docs[0].Cancel)
	require.Equal(t, 3, series.Docs[0].NetIssue)
}

// TestGSTR1Export_DeterministicOutputAndGolden pins byte-identical output
// across repeated builds (no map-iteration nondeterminism) and an exact
// golden JSON shape for one minimal scenario.
func TestGSTR1Export_DeterministicOutputAndGolden(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	customer := createIndiaCustomer(t, app, "Sahyadri Process Equipment Pvt Ltd")
	buyerGSTIN := synthGSTIN(t, "27")
	createGSTR1Invoice(t, app, customer, gstr1InvoiceOpts{
		Number: "INV/26-27/900", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: buyerGSTIN, PosStateCode: "27",
		HSN: "9026", UQC: "NOS", Description: "Flow transmitter",
		Qty: 2, Rate: 5000, GrandTotal: 11800,
	})

	_, payloads1, err := app.buildGSTR1Export(2026, 6)
	require.NoError(t, err)
	raw1, err := json.Marshal(payloads1["Meridian Mumbai"])
	require.NoError(t, err)

	_, payloads2, err := app.buildGSTR1Export(2026, 6)
	require.NoError(t, err)
	raw2, err := json.Marshal(payloads2["Meridian Mumbai"])
	require.NoError(t, err)

	require.Equal(t, string(raw1), string(raw2), "GSTR-1 JSON must be byte-identical across repeated builds")

	expected := `{
		"gstin": "27AABCM0472E1ZT",
		"fp": "062026",
		"version": "GST3.2.4",
		"b2b": [
			{
				"ctin": "` + buyerGSTIN + `",
				"inv": [
					{
						"inum": "INV/26-27/900",
						"idt": "15-06-2026",
						"val": 11800,
						"pos": "27",
						"rchrg": "N",
						"inv_typ": "R",
						"itms": [
							{"num": 1, "itm_det": {"rt": 18, "txval": 10000, "iamt": 0, "camt": 900, "samt": 900, "csamt": 0}}
						]
					}
				]
			}
		],
		"b2cl": [],
		"b2cs": [],
		"cdnr": [],
		"cdnur": [],
		"hsn_b2b": [
			{"num": 1, "hsn_sc": "9026", "desc": "Flow transmitter", "uqc": "NOS", "qty": 2, "txval": 10000, "iamt": 0, "camt": 900, "samt": 900, "csamt": 0, "rt": 18}
		],
		"hsn_b2c": [],
		"doc_issue": [
			{"doc_num": 1, "docs": [{"num": "INV/26-27", "from": "INV/26-27/900", "to": "INV/26-27/900", "totnum": 1, "cancel": 0, "net_issue": 1}]}
		],
		"nil": [
			{"sply_ty": "INTRA", "reg_ty": "REGISTERED", "nil_amt": 0, "exmp_amt": 0, "ngsup_amt": 0},
			{"sply_ty": "INTRA", "reg_ty": "UNREGISTERED", "nil_amt": 0, "exmp_amt": 0, "ngsup_amt": 0},
			{"sply_ty": "INTER", "reg_ty": "REGISTERED", "nil_amt": 0, "exmp_amt": 0, "ngsup_amt": 0},
			{"sply_ty": "INTER", "reg_ty": "UNREGISTERED", "nil_amt": 0, "exmp_amt": 0, "ngsup_amt": 0}
		]
	}`
	require.JSONEq(t, expected, string(raw1))
}

// TestValidateGSTR1Period_CatchesValidationIssues covers the dry-run
// validation pass: a malformed buyer GSTIN, a line missing HSN, and an
// invoice missing place-of-supply must all surface as issues BEFORE any
// file is written (missing HSN surfaces as an "error"-severity refusal, per
// the B3 engine's refuse-to-generate doctrine; the other two as warnings).
func TestValidateGSTR1Period_CatchesValidationIssues(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app)
	withIndiaOverlay(t, "overlays/india-demo")

	now := time.Date(2026, 6, 22, 10, 0, 0, 0, time.UTC)

	// Bad checksum GSTIN.
	badGSTINCustomer := createIndiaCustomer(t, app, "Bad GSTIN Customer")
	createGSTR1Invoice(t, app, badGSTINCustomer, gstr1InvoiceOpts{
		Number: "INV/26-27/301", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: "27AABCM0472E1ZZ", PosStateCode: "27", // valid format, wrong check digit
		HSN: "9026", UQC: "NOS", Qty: 1, Rate: 1000, GrandTotal: 1180,
	})

	// Missing HSN.
	noHSNCustomer := createIndiaCustomer(t, app, "No HSN Customer")
	createGSTR1Invoice(t, app, noHSNCustomer, gstr1InvoiceOpts{
		Number: "INV/26-27/302", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: synthGSTIN(t, "27"), PosStateCode: "27",
		HSN: "", UQC: "NOS", Qty: 1, Rate: 1000, GrandTotal: 1180,
	})

	// Missing place of supply.
	noPosCustomer := createIndiaCustomer(t, app, "No POS Customer")
	createGSTR1Invoice(t, app, noPosCustomer, gstr1InvoiceOpts{
		Number: "INV/26-27/303", Date: now, Division: "Meridian Mumbai",
		BuyerGSTIN: synthGSTIN(t, "27"), PosStateCode: "",
		HSN: "9026", UQC: "NOS", Qty: 1, Rate: 1000, GrandTotal: 1180,
	})

	result, err := app.ValidateGSTR1Period(2026, 6)
	require.NoError(t, err)
	require.Empty(t, result.Files, "a dry-run validation must never populate Files")

	var sawBadGSTIN, sawMissingHSN, sawMissingPOS bool
	for _, issue := range result.Issues {
		require.NotEmpty(t, issue.Severity)
		switch issue.Section {
		case "b2b":
			if strings.Contains(issue.Message, "27AABCM0472E1ZZ") {
				sawBadGSTIN = true
			}
		case "hsn":
			// The B3 engine refuses the whole invoice before this export ever
			// sees the blank HSN line (refuse-to-generate doctrine) -- the
			// issue surfaces as an "error"-severity "hsn" section, not a
			// softer per-line warning.
			if strings.Contains(issue.Message, "INV/26-27/302") {
				sawMissingHSN = true
				require.Equal(t, "error", issue.Severity)
			}
		case "pos":
			if strings.Contains(issue.Message, "INV/26-27/303") {
				sawMissingPOS = true
			}
		}
	}
	require.True(t, sawBadGSTIN, "must flag the malformed buyer GSTIN: %+v", result.Issues)
	require.True(t, sawMissingHSN, "must flag the missing/invalid HSN as a refuse-to-generate error: %+v", result.Issues)
	require.True(t, sawMissingPOS, "must flag the missing place-of-supply: %+v", result.Issues)

	// Dry run must not have written any file to disk.
	exportDir := app.getExportDir("report", "", "", 2026)
	entries, _ := os.ReadDir(exportDir)
	for _, e := range entries {
		require.NotContains(t, e.Name(), "GSTR1_", "validation must not write any GSTR-1 file")
	}
}

// TestGSTR1Export_GCCDivisionsProduceNoFileNoLeak covers the hard law: a GCC
// deployment (no India plane mounted) must produce ZERO GSTR-1 files, and no
// GCC data can ever leak into an India export.
func TestGSTR1Export_GCCDivisionsProduceNoFileNoLeak(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	app := setupTestApp(t)
	migrateIndiaDocTables(t, app) // CreditNote/CreditNoteItem tables buildGSTR1Export always queries
	// activeOverlay defaults to BuiltinDefaults() (Acme/Beacon, no India plane).

	buildEInvoiceTestInvoice(t, app, "Acme Instrumentation", "GSTR1-GCC-001")
	buildEInvoiceTestInvoice(t, app, "Beacon Controls", "GSTR1-GCC-002")

	now := time.Now()
	files, err := app.ExportGSTR1JSON(now.Year(), int(now.Month()))
	require.NoError(t, err)
	require.Empty(t, files, "a GCC-only deployment must produce zero GSTR-1 files")
}
