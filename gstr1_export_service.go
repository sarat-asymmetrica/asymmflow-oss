// ═══════════════════════════════════════════════════════════════════════════
// GSTR-1 PORTAL-JSON EXPORT (India Spec-01 B5)
//
// Generates the period's GSTR-1 outward-supplies return as the free
// offline-utility JSON the GST portal accepts for upload -- zero external
// APIs, zero recurring cost (§0 G7). One file per India-mounted, NON-
// composition division: GSTR-1 is filed per GSTIN, and composition dealers
// file CMP-08/GSTR-4 instead (a genuine statutory scope decision, not an
// oversight -- see composition-skip below). GCC divisions (India == nil)
// never produce a file.
//
// This is pure read-only derivation over Invoice/CreditNote, mirroring
// ExportVATReturnData's shape (einvoice_service.go): it posts nothing and
// mutates nothing except the export-directory JSON file(s) it writes.
//
// SCHEMA PROVENANCE (R-A4-5, honesty law): every field-level shape below is
// built against the A4 schema-verification record (scratchpad/
// A4_GSTR1_SCHEMA_VERIFICATION.md), not training-data memory. The literal
// offline-tool JSON payload was not read this session -- only its CSV/Excel
// vocabulary (via the downloaded Returns Offline Tool ZIP) and a GSP
// conformance mirror. Every section carries a confidence note below; the
// weakest is `nil` (Table 8), whose JSON key names are UNVERIFIED.
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ph_holdings_app/pkg/compliance/india"
)

// ---- validation result -----------------------------------------------------

// GSTR1ValidationIssue is one thing the portal would likely reject or flag,
// surfaced in-app BEFORE the accountant uploads (never blocking the export
// itself, except the "error" severity structural cases that abort the whole
// division -- see buildGSTR1Export).
type GSTR1ValidationIssue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Section  string `json:"section"`
	Message  string `json:"message"`
}

// GSTR1ExportResult carries both what ExportGSTR1JSON wrote (Files) and what
// ValidateGSTR1Period found without writing anything (Issues only, Files
// left empty for a dry run).
type GSTR1ExportResult struct {
	Files  []string               `json:"files"`
	Issues []GSTR1ValidationIssue `json:"issues"`
}

// ---- JSON payload shapes ----------------------------------------------------
// R-A4-4: dates DD-MM-YYYY, fp MMYYYY, pos a 2-digit state-code string, rt a
// numeric percent (18, not 0.18), amounts rounded to 2 decimals (round2).
// Every slice is initialized non-nil in newGSTR1Payload so an empty section
// marshals as `[]`, never `null` -- the portal's schema expects arrays.

type gstr1Payload struct {
	GSTIN    string           `json:"gstin"`
	FP       string           `json:"fp"`
	Version  string           `json:"version"`
	B2B      []gstr1B2BGroup  `json:"b2b"`
	B2CL     []gstr1B2CLGroup `json:"b2cl"`
	B2CS     []gstr1B2CSRow   `json:"b2cs"`
	CDNR     []gstr1CDNGroup  `json:"cdnr"`
	CDNUR    []gstr1CDNNote   `json:"cdnur"`
	HSNB2B   []gstr1HSNRow    `json:"hsn_b2b"`
	HSNB2C   []gstr1HSNRow    `json:"hsn_b2c"`
	DocIssue []gstr1DocSeries `json:"doc_issue"`
	Nil      []gstr1NilRow    `json:"nil"`
}

func newGSTR1Payload(gstin, fp, version string) *gstr1Payload {
	return &gstr1Payload{
		GSTIN:    gstin,
		FP:       fp,
		Version:  version,
		B2B:      []gstr1B2BGroup{},
		B2CL:     []gstr1B2CLGroup{},
		B2CS:     []gstr1B2CSRow{},
		CDNR:     []gstr1CDNGroup{},
		CDNUR:    []gstr1CDNNote{},
		HSNB2B:   []gstr1HSNRow{},
		HSNB2C:   []gstr1HSNRow{},
		DocIssue: []gstr1DocSeries{},
		Nil:      []gstr1NilRow{},
	}
}

// gstr1B2BGroup is one receiver's (ctin's) invoices -- Table 4A.
type gstr1B2BGroup struct {
	Ctin string         `json:"ctin"`
	Inv  []gstr1Invoice `json:"inv"`
}

// gstr1B2CLGroup is one place-of-supply state's large B2C invoices -- Table
// 5. Gate ruling: the live GSTN schema groups b2cl by pos the same way b2b
// groups by ctin (SECONDARY confidence -- neither the A4 GSP mirror nor the
// offline-tool CSVs settle the grouping wrapper; re-verify against the
// offline tool's own sample JSON before any real portal upload, R-A4-5).
// The inner invoices keep their pos field populated (redundant with the
// group key, tolerated for a shared struct -- the portal ignores unknown/
// extra fields more gracefully than missing ones).
type gstr1B2CLGroup struct {
	Pos string         `json:"pos"`
	Inv []gstr1Invoice `json:"inv"`
}

// gstr1Invoice is one invoice's header (shared shape for b2b's inv[] and
// b2cl's pos-grouped inv[]).
type gstr1Invoice struct {
	Inum   string      `json:"inum"`
	Idt    string      `json:"idt"`
	Val    float64     `json:"val"`
	Pos    string      `json:"pos"`
	Rchrg  string      `json:"rchrg"`   // "Y"/"N"
	InvTyp string      `json:"inv_typ"` // R-A4-6: R/SEWP/SEWOP/DE/CBW, default "R" this wave
	Itms   []gstr1Item `json:"itms"`
}

// gstr1Item is one rate-bucket line within an invoice/note (invoice lines
// sharing a rate are aggregated into one itm_det row, matching GSTN's own
// behavior).
type gstr1Item struct {
	Num    int          `json:"num"`
	ItmDet gstr1ItemDet `json:"itm_det"`
}

type gstr1ItemDet struct {
	Rt    float64 `json:"rt"`
	Txval float64 `json:"txval"`
	Iamt  float64 `json:"iamt"`
	Camt  float64 `json:"camt"`
	Samt  float64 `json:"samt"`
	Csamt float64 `json:"csamt"`
}

// gstr1B2CSRow is one (sply_ty, pos, rt) aggregate bucket -- Table 7.
type gstr1B2CSRow struct {
	SplyTy string  `json:"sply_ty"` // "INTER" / "INTRA"
	Pos    string  `json:"pos"`
	Typ    string  `json:"typ"` // fixed "OE" (Other than E-commerce) this wave
	Rt     float64 `json:"rt"`
	Txval  float64 `json:"txval"`
	Iamt   float64 `json:"iamt"`
	Camt   float64 `json:"camt"`
	Samt   float64 `json:"samt"`
	Csamt  float64 `json:"csamt"`
}

// gstr1CDNGroup is CDNR's ctin grouping -- mirrors b2b's shape (a judgment
// call: the B5 spec text didn't say "grouped by ctin" explicitly for CDNR
// the way it did for B2B, but the live GSTN schema genuinely groups
// registered-recipient notes by ctin the same way; see report).
type gstr1CDNGroup struct {
	Ctin string         `json:"ctin"`
	Nt   []gstr1CDNNote `json:"nt"`
}

// gstr1CDNNote is one credit/debit note. Inum/Idt reference the ORIGINAL
// invoice this note adjusts -- CDN-invoice delinking (effective since 2020)
// makes this reference optional in the real schema (honesty flag, A4 §(f)6);
// our model always carries the link so we always populate it.
type gstr1CDNNote struct {
	Ntty   string      `json:"ntty"` // "C" = credit note, this wave never emits debit notes
	NtNum  string      `json:"nt_num"`
	NtDt   string      `json:"nt_dt"`
	Pos    string      `json:"pos"`
	Rchrg  string      `json:"rchrg"`
	InvTyp string      `json:"inv_typ"`
	Val    float64     `json:"val"`
	Itms   []gstr1Item `json:"itms"`
	Inum   string      `json:"inum"`
	Idt    string      `json:"idt"`
}

// gstr1HSNRow is one HSN/UQC/rate aggregate row -- Table 12, split hsn_b2b /
// hsn_b2c (R-A4-3, the Phase-3 2025 shape; a flat hsn[] is stale and never
// emitted here).
type gstr1HSNRow struct {
	Num   int     `json:"num"`
	HSNSC string  `json:"hsn_sc"`
	Desc  string  `json:"desc"`
	UQC   string  `json:"uqc"`
	Qty   float64 `json:"qty"`
	Txval float64 `json:"txval"`
	Iamt  float64 `json:"iamt"`
	Camt  float64 `json:"camt"`
	Samt  float64 `json:"samt"`
	Csamt float64 `json:"csamt"`
	Rt    float64 `json:"rt"`
}

// gstr1DocSeries is one document-nature category's series summary -- Table
// 13. DocNum follows the widely-corroborated (but not this-session-verified)
// Table-13 nature-of-document numbering: 1 = "Invoices for outward supply",
// 5 = "Credit Note".
type gstr1DocSeries struct {
	DocNum int                   `json:"doc_num"`
	Docs   []gstr1DocSeriesEntry `json:"docs"`
}

type gstr1DocSeriesEntry struct {
	Num      string `json:"num"` // series prefix, e.g. "INV/26-27"
	From     string `json:"from"`
	To       string `json:"to"`
	Totnum   int    `json:"totnum"`
	Cancel   int    `json:"cancel"`
	NetIssue int    `json:"net_issue"`
}

// gstr1NilRow is Table 8's fixed 4-row shape (Inter/Intra x Registered/
// Unregistered). Field names are UNVERIFIED this session (A4 §(c): no
// primary artifact read for Table 8) -- built from the offline tool's
// exemp.csv column structure (Nil Rated / Exempted / Non-GST), all zero this
// wave since exempt-supply classification isn't modeled yet.
type gstr1NilRow struct {
	SplyTy   string  `json:"sply_ty"`
	RegTy    string  `json:"reg_ty"`
	NilAmt   float64 `json:"nil_amt"`
	ExmpAmt  float64 `json:"exmp_amt"`
	NgsupAmt float64 `json:"ngsup_amt"`
}

// ---- round2 -----------------------------------------------------------------

// round2 rounds to 2 decimal places (R-A4-2: GSTR-1 JSON serializes 2-decimal
// amounts; Section 170 nearest-rupee rounding is a separate, non-default
// report-layer concern, not applied here).
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ---- ExportGSTR1JSON / ValidateGSTR1Period ---------------------------------

// ExportGSTR1JSON generates the GSTR-1 portal-upload JSON for the given
// calendar month, one file per India-mounted non-composition division
// (GSTR-1 is filed per GSTIN). Returns the paths written. Read-only
// derivation: it posts nothing, mutates nothing except the files themselves.
func (a *App) ExportGSTR1JSON(year, month int) ([]string, error) {
	if err := a.requirePermission("finance:read"); err != nil {
		return nil, err
	}

	result, payloads, err := a.buildGSTR1Export(year, month)
	if err != nil {
		return nil, err
	}

	exportDir := a.getExportDir("report", "", "", year)
	var files []string
	for _, division := range activeOverlay.Divisions {
		payload, ok := payloads[division.Key]
		if !ok {
			continue
		}
		profile := companyDocumentProfile(division.Key)

		raw, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return files, fmt.Errorf("failed to marshal GSTR-1 JSON for %s: %w", division.Key, marshalErr)
		}
		checkGSTR1SizeCaps(result, payload, raw)

		filename := fmt.Sprintf("GSTR1_%s_%s.json", profile.India.GSTIN, payload.FP)
		path := filepath.Join(exportDir, filename)
		if writeErr := os.WriteFile(path, raw, 0640); writeErr != nil {
			return files, fmt.Errorf("failed to write GSTR-1 JSON for %s: %w", division.Key, writeErr)
		}
		files = append(files, path)
	}

	for _, issue := range result.Issues {
		log.Printf("⚠️ GSTR-1 %s [%s]: %s", issue.Severity, issue.Section, issue.Message)
	}
	log.Printf("✅ GSTR-1 JSON exported: %d file(s) for %02d/%d", len(files), month, year)
	return files, nil
}

// ValidateGSTR1Period is the dry-run twin of ExportGSTR1JSON: it runs the
// exact same derivation and validation pass but writes nothing, so the
// accountant can see portal-rejection risks (missing GSTINs, HSN gaps, size
// caps) before ever generating a file.
func (a *App) ValidateGSTR1Period(year, month int) (*GSTR1ExportResult, error) {
	if err := a.requirePermission("finance:read"); err != nil {
		return nil, err
	}

	result, payloads, err := a.buildGSTR1Export(year, month)
	if err != nil {
		return nil, err
	}
	for _, payload := range payloads {
		raw, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			continue
		}
		checkGSTR1SizeCaps(result, payload, raw)
	}
	return result, nil
}

// checkGSTR1SizeCaps flags the offline-tool's documented upload caps
// (Readme.txt, A4 orchestrator addendum): 5MB per file, 19,000 items.
func checkGSTR1SizeCaps(result *GSTR1ExportResult, payload *gstr1Payload, raw []byte) {
	if len(raw) > 5*1024*1024 {
		result.Issues = append(result.Issues, GSTR1ValidationIssue{
			Severity: "warning", Section: "size",
			Message: fmt.Sprintf("%s: GSTR-1 JSON is %d bytes, exceeds the portal's 5MB upload cap", payload.GSTIN, len(raw)),
		})
	}
	if n := countGSTR1Items(payload); n > 19000 {
		result.Issues = append(result.Issues, GSTR1ValidationIssue{
			Severity: "warning", Section: "size",
			Message: fmt.Sprintf("%s: %d items exceeds the portal's 19,000-item upload cap", payload.GSTIN, n),
		})
	}
}

// countGSTR1Items approximates the portal's "item" count as the total
// itm_det rows across every invoice/note-bearing section, plus one row per
// b2cs aggregate.
func countGSTR1Items(p *gstr1Payload) int {
	n := 0
	for _, g := range p.B2B {
		for _, inv := range g.Inv {
			n += len(inv.Itms)
		}
	}
	for _, g := range p.B2CL {
		for _, inv := range g.Inv {
			n += len(inv.Itms)
		}
	}
	n += len(p.B2CS)
	for _, g := range p.CDNR {
		for _, nt := range g.Nt {
			n += len(nt.Itms)
		}
	}
	for _, nt := range p.CDNUR {
		n += len(nt.Itms)
	}
	return n
}

// ---- derivation --------------------------------------------------------

// buildGSTR1Export queries the period's invoices/credit notes once, buckets
// them per division (mirroring ExportVATReturnData's bucketFor pattern), and
// builds one payload per eligible division. Eligible = India-mounted AND
// non-composition AND carries a valid GSTIN; GCC divisions and composition
// divisions never get an entry in the returned map (no file, ever).
func (a *App) buildGSTR1Export(year, month int) (*GSTR1ExportResult, map[string]*gstr1Payload, error) {
	if a.db == nil {
		return nil, nil, fmt.Errorf("database not initialized")
	}
	if month < 1 || month > 12 {
		return nil, nil, fmt.Errorf("month must be between 1 and 12")
	}
	if year < 2017 || year > 2100 {
		return nil, nil, fmt.Errorf("year must be between 2017 and 2100 (GST commenced July 2017)")
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	// Same status filters as ExportVATReturnData (mirrored per the B5 spec):
	// invoices exclude Cancelled/Void/Proforma/Draft; credit notes are only
	// counted once Applied. NOTE: because cancelled invoices are excluded
	// upstream here, doc_issue's "cancel" count can never reflect a
	// cancelled document from this query alone -- a known residue, see report.
	var invoices []Invoice
	if err := a.db.Preload("Items").
		Where("invoice_date >= ? AND invoice_date < ? AND status NOT IN ?",
			startDate, endDate, []string{"Cancelled", "Void", "Proforma", "Draft"}).
		Find(&invoices).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to query invoices: %w", err)
	}

	var creditNotes []CreditNote
	if err := a.db.Preload("Items").
		Where("applied_at >= ? AND applied_at < ? AND status = ?", startDate, endDate, "Applied").
		Find(&creditNotes).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to query credit notes: %w", err)
	}

	invByDivision := map[string][]Invoice{}
	originalByID := map[string]Invoice{}
	for _, inv := range invoices {
		key := normalizeDivisionName(inv.Division)
		invByDivision[key] = append(invByDivision[key], inv)
		originalByID[inv.ID] = inv
	}

	// A credit note's original invoice may have been issued in an earlier
	// period (the common case) -- fetch any not already in this period's set.
	for _, cn := range creditNotes {
		if _, ok := originalByID[cn.InvoiceID]; ok || strings.TrimSpace(cn.InvoiceID) == "" {
			continue
		}
		var orig Invoice
		if err := a.db.Preload("Items").Where("id = ?", cn.InvoiceID).First(&orig).Error; err == nil {
			originalByID[orig.ID] = orig
		}
	}

	cnByDivision := map[string][]CreditNote{}
	for _, cn := range creditNotes {
		key := a.resolveCreditNoteDivision(cn)
		cnByDivision[key] = append(cnByDivision[key], cn)
	}

	result := &GSTR1ExportResult{}
	payloads := map[string]*gstr1Payload{}

	for _, division := range activeOverlay.Divisions {
		profile := companyDocumentProfile(division.Key)
		if profile.India == nil {
			continue // GCC division: no India plane, no GSTR-1, ever.
		}
		if profile.India.Composition {
			// Composition dealers file CMP-08/quarterly + GSTR-4 annually,
			// never GSTR-1 -- a statutory scope decision, not an oversight.
			continue
		}
		if !india.ValidGSTIN(profile.India.GSTIN) {
			result.Issues = append(result.Issues, GSTR1ValidationIssue{
				Severity: "error", Section: "gstin",
				Message: fmt.Sprintf("division %q has no valid GSTIN configured -- cannot file GSTR-1", division.Key),
			})
			continue
		}

		payload, issues := buildGSTR1PayloadForDivision(profile, year, month,
			invByDivision[division.Key], cnByDivision[division.Key], originalByID)
		result.Issues = append(result.Issues, issues...)
		payloads[division.Key] = payload
	}

	return result, payloads, nil
}

// buildGSTR1PayloadForDivision derives one division's full GSTR-1 payload
// from its already-filtered invoices/credit notes for the period.
func buildGSTR1PayloadForDivision(profile CompanyDocumentProfile, year, month int, invoices []Invoice, creditNotes []CreditNote, originalByID map[string]Invoice) (*gstr1Payload, []GSTR1ValidationIssue) {
	// NOTE: HSN-tier/digit validation is NOT duplicated here. The B3 engine
	// (resolveIndiaGSTForInvoice -> ComputeInvoiceGST) already enforces the
	// G4 HSN-digit mandate before computing anything and refuses the whole
	// invoice on violation (refuse-to-generate doctrine) -- any invoice that
	// reaches the loop below has already passed that check, so a second
	// AATOTier/RequiredHSNDigits check here would be unreachable dead code.
	cfg := activeOverlay.IndiaConfig()

	var issues []GSTR1ValidationIssue
	flag := func(severity, section, message string) {
		issues = append(issues, GSTR1ValidationIssue{Severity: severity, Section: section, Message: message})
	}

	fp := fmt.Sprintf("%02d%d", month, year)
	payload := newGSTR1Payload(profile.India.GSTIN, fp, cfg.GSTR1SchemaVersion)

	sortedInvoices := append([]Invoice(nil), invoices...)
	sort.Slice(sortedInvoices, func(i, j int) bool { return sortedInvoices[i].InvoiceNumber < sortedInvoices[j].InvoiceNumber })

	type b2csKey struct {
		splyTy string
		pos    string
		rt     float64
	}
	b2bByCtin := map[string][]gstr1Invoice{}
	b2clByPos := map[string][]gstr1Invoice{}
	b2csAgg := map[b2csKey]*gstr1B2CSRow{}
	hsnB2BAgg := map[string]*gstr1HSNRow{}
	hsnB2CAgg := map[string]*gstr1HSNRow{}

	for _, inv := range sortedInvoices {
		gst, err := resolveIndiaGSTForInvoice(inv, profile)
		if err != nil {
			// The B3 engine's refuse-to-generate doctrine (CLAUDE.md
			// invariant 5) already validates every line's HSN digit count and
			// rate configuration BEFORE computing anything -- an
			// HSNValidationError here means the underlying invoice could
			// never have been issued at all, so it gets its own "hsn"
			// section rather than the generic "computation" bucket. Note:
			// this makes a separate per-item HSN blank/short-digit check in
			// this function unreachable dead code (the engine always catches
			// it first) -- deliberately not duplicated here.
			var hsnErr *india.HSNValidationError
			if errors.As(err, &hsnErr) {
				flag("error", "hsn", fmt.Sprintf("invoice %s: %v", inv.InvoiceNumber, err))
			} else {
				flag("error", "computation", fmt.Sprintf("invoice %s: GST computation refused: %v", inv.InvoiceNumber, err))
			}
			continue
		}

		// Canonicalize the buyer GSTIN once (Spec-07 law: canonicalize BOTH
		// sides of every comparison) -- it is the ctin grouping key below, and
		// a raw " 27aabcm..." must never split into its own group (gate fix).
		ctin := strings.ToUpper(strings.TrimSpace(inv.BuyerGSTIN))
		b2b := ctin != ""
		if b2b && !india.ValidGSTIN(ctin) {
			flag("warning", "b2b", fmt.Sprintf("invoice %s: buyer GSTIN %q fails format/checksum validation", inv.InvoiceNumber, inv.BuyerGSTIN))
		}
		if strings.TrimSpace(inv.PlaceOfSupplyStateCode) == "" {
			flag("warning", "pos", fmt.Sprintf("invoice %s: missing place-of-supply state code", inv.InvoiceNumber))
		} else if !india.ValidStateCode(inv.PlaceOfSupplyStateCode) {
			flag("warning", "pos", fmt.Sprintf("invoice %s: unknown place-of-supply state code %q", inv.InvoiceNumber, inv.PlaceOfSupplyStateCode))
		}

		rchrg := "N"
		if inv.ReverseCharge {
			rchrg = "Y"
		}
		// Gate fix: val derives from the SAME engine computation the itm_det
		// rows come from (taxable + all tax heads), never from the stored
		// GrandTotalBHD -- that field is still populated by the GCC creation
		// math (hardcoded Bahrain 10%, flagged as the IN-W2 wiring gap), and a
		// payload whose val disagrees with its own items is exactly what the
		// portal's tolerance-band validation exists to reject.
		derivedVal := round2(gst.Totals.TaxableValueINR + gst.Totals.CGST + gst.Totals.SGST + gst.Totals.IGST + gst.Totals.Cess)
		invEntry := gstr1Invoice{
			Inum:   inv.InvoiceNumber,
			Idt:    inv.InvoiceDate.Format("02-01-2006"),
			Val:    derivedVal,
			Pos:    inv.PlaceOfSupplyStateCode,
			Rchrg:  rchrg,
			InvTyp: "R",
			Itms:   itemsFromLineResults(gst.Lines),
		}

		interState := gst.Classification == india.InterState
		switch {
		case b2b:
			b2bByCtin[ctin] = append(b2bByCtin[ctin], invEntry)
		case interState && invEntry.Val > cfg.B2CLThresholdINR:
			b2clByPos[inv.PlaceOfSupplyStateCode] = append(b2clByPos[inv.PlaceOfSupplyStateCode], invEntry)
		default:
			splyTy := "INTRA"
			if interState {
				splyTy = "INTER"
			}
			for _, lr := range gst.Lines {
				key := b2csKey{splyTy, inv.PlaceOfSupplyStateCode, lr.RatePct}
				row := b2csAgg[key]
				if row == nil {
					row = &gstr1B2CSRow{SplyTy: splyTy, Pos: inv.PlaceOfSupplyStateCode, Typ: "OE", Rt: lr.RatePct}
					b2csAgg[key] = row
				}
				row.Txval = round2(row.Txval + lr.TaxableValueINR)
				row.Iamt = round2(row.Iamt + lr.IGST)
				row.Camt = round2(row.Camt + lr.CGST)
				row.Samt = round2(row.Samt + lr.SGST)
				row.Csamt = round2(row.Csamt + lr.Cess)
			}
		}

		// HSN summary (Table 12): b2b lines feed hsn_b2b, everything else
		// feeds hsn_b2c (R-A4-3). Only invoices feed this aggregate this wave
		// -- credit-note lines are deliberately NOT netted in here (a
		// simplification flagged in the report, not an attempt at exact
		// portal net-of-returns semantics).
		hsnAgg := hsnB2CAgg
		if b2b {
			hsnAgg = hsnB2BAgg
		}
		for i, lr := range gst.Lines {
			accumulateHSN(hsnAgg, inv.Items[i].HSNCode, inv.Items[i].UQC, inv.Items[i].Description, inv.Items[i].Quantity, lr)
		}
	}

	for ctin, invs := range b2bByCtin {
		payload.B2B = append(payload.B2B, gstr1B2BGroup{Ctin: ctin, Inv: invs})
	}
	sort.Slice(payload.B2B, func(i, j int) bool { return payload.B2B[i].Ctin < payload.B2B[j].Ctin })
	for pos, invs := range b2clByPos {
		payload.B2CL = append(payload.B2CL, gstr1B2CLGroup{Pos: pos, Inv: invs})
	}
	sort.Slice(payload.B2CL, func(i, j int) bool { return payload.B2CL[i].Pos < payload.B2CL[j].Pos })

	for _, row := range b2csAgg {
		payload.B2CS = append(payload.B2CS, *row)
	}
	sort.Slice(payload.B2CS, func(i, j int) bool {
		a, b := payload.B2CS[i], payload.B2CS[j]
		if a.SplyTy != b.SplyTy {
			return a.SplyTy < b.SplyTy
		}
		if a.Pos != b.Pos {
			return a.Pos < b.Pos
		}
		return a.Rt < b.Rt
	})

	// Credit/debit notes (CDNR/CDNUR): registered vs unregistered is decided
	// by the ORIGINAL invoice's BuyerGSTIN (a note has no GSTIN of its own).
	cdnrByCtin := map[string][]gstr1CDNNote{}
	sortedCN := append([]CreditNote(nil), creditNotes...)
	sort.Slice(sortedCN, func(i, j int) bool { return sortedCN[i].CreditNoteNumber < sortedCN[j].CreditNoteNumber })

	for _, cn := range sortedCN {
		orig, ok := originalByID[cn.InvoiceID]
		if !ok {
			flag("error", "cdnr", fmt.Sprintf("credit note %s: original invoice not found -- cannot classify", cn.CreditNoteNumber))
			continue
		}
		ctin := strings.ToUpper(strings.TrimSpace(orig.BuyerGSTIN)) // canonical grouping key, same gate fix as b2b
		b2b := ctin != ""
		gst, err := resolveIndiaGSTForCreditNote(cn.Items, profile, orig.PlaceOfSupplyStateCode, b2b, orig.ReverseCharge)
		if err != nil {
			flag("error", "cdnr", fmt.Sprintf("credit note %s: GST computation refused: %v", cn.CreditNoteNumber, err))
			continue
		}

		rchrg := "N"
		if orig.ReverseCharge {
			rchrg = "Y"
		}
		note := gstr1CDNNote{
			Ntty:   "C",
			NtNum:  cn.CreditNoteNumber,
			NtDt:   cn.CreditNoteDate.Format("02-01-2006"),
			Pos:    orig.PlaceOfSupplyStateCode,
			Rchrg:  rchrg,
			InvTyp: "R",
			// Gate fix: derived from the engine computation, same reasoning as
			// the invoice-side val (stored totals are GCC creation math).
			Val:    round2(gst.Totals.TaxableValueINR + gst.Totals.CGST + gst.Totals.SGST + gst.Totals.IGST + gst.Totals.Cess),
			Itms:   itemsFromLineResults(gst.Lines),
			Inum:   orig.InvoiceNumber,
			Idt:    orig.InvoiceDate.Format("02-01-2006"),
		}

		if b2b {
			cdnrByCtin[ctin] = append(cdnrByCtin[ctin], note)
		} else {
			payload.CDNUR = append(payload.CDNUR, note)
		}
	}
	for ctin, notes := range cdnrByCtin {
		payload.CDNR = append(payload.CDNR, gstr1CDNGroup{Ctin: ctin, Nt: notes})
	}
	sort.Slice(payload.CDNR, func(i, j int) bool { return payload.CDNR[i].Ctin < payload.CDNR[j].Ctin })
	sort.Slice(payload.CDNUR, func(i, j int) bool { return payload.CDNUR[i].NtNum < payload.CDNUR[j].NtNum })

	payload.HSNB2B = finalizeHSN(hsnB2BAgg)
	payload.HSNB2C = finalizeHSN(hsnB2CAgg)
	if len(payload.HSNB2B) == 0 && len(payload.HSNB2C) > 0 {
		// GSTN's Jul-2025 clarification for B2C-only filers: Table 12A wants
		// one dummy zeroed row when hsn_b2b would otherwise be empty. Not
		// auto-injected here -- surfaced for the accountant to decide,
		// per the A4 honest-residue law.
		flag("warning", "hsn_b2b", "hsn_b2b is empty while hsn_b2c is not -- GSTN's Jul-2025 note requires a dummy zeroed Table-12A row for B2C-only filers; not auto-injected, add it manually before upload if this filer has no B2B supplies")
	}

	payload.DocIssue = buildDocIssue(sortedInvoices, sortedCN)
	payload.Nil = []gstr1NilRow{
		{SplyTy: "INTRA", RegTy: "REGISTERED"},
		{SplyTy: "INTRA", RegTy: "UNREGISTERED"},
		{SplyTy: "INTER", RegTy: "REGISTERED"},
		{SplyTy: "INTER", RegTy: "UNREGISTERED"},
	}

	return payload, issues
}

// itemsFromLineResults aggregates an invoice's/note's computed lines by
// rate into itm_det rows (lines sharing a rate collapse into one row, the
// same behavior GSTN's own portal applies).
func itemsFromLineResults(lines []india.LineResult) []gstr1Item {
	byRate := map[float64]*gstr1ItemDet{}
	for _, lr := range lines {
		det := byRate[lr.RatePct]
		if det == nil {
			det = &gstr1ItemDet{Rt: lr.RatePct}
			byRate[lr.RatePct] = det
		}
		det.Txval = round2(det.Txval + lr.TaxableValueINR)
		det.Iamt = round2(det.Iamt + lr.IGST)
		det.Camt = round2(det.Camt + lr.CGST)
		det.Samt = round2(det.Samt + lr.SGST)
		det.Csamt = round2(det.Csamt + lr.Cess)
	}
	rates := make([]float64, 0, len(byRate))
	for rt := range byRate {
		rates = append(rates, rt)
	}
	sort.Float64s(rates)
	items := make([]gstr1Item, 0, len(rates))
	for i, rt := range rates {
		items = append(items, gstr1Item{Num: i + 1, ItmDet: *byRate[rt]})
	}
	return items
}

// accumulateHSN folds one computed line into its (hsn, uqc, rate) bucket.
func accumulateHSN(agg map[string]*gstr1HSNRow, hsn, uqc, desc string, qty float64, lr india.LineResult) {
	hsn = strings.TrimSpace(hsn)
	key := hsn + "|" + uqc + "|" + fmt.Sprintf("%g", lr.RatePct)
	row := agg[key]
	if row == nil {
		row = &gstr1HSNRow{HSNSC: hsn, Desc: desc, UQC: uqc, Rt: lr.RatePct}
		agg[key] = row
	}
	row.Qty += qty
	row.Txval = round2(row.Txval + lr.TaxableValueINR)
	row.Iamt = round2(row.Iamt + lr.IGST)
	row.Camt = round2(row.Camt + lr.CGST)
	row.Samt = round2(row.Samt + lr.SGST)
	row.Csamt = round2(row.Csamt + lr.Cess)
}

// finalizeHSN converts the accumulation map into a deterministically sorted,
// numbered slice.
func finalizeHSN(agg map[string]*gstr1HSNRow) []gstr1HSNRow {
	rows := make([]gstr1HSNRow, 0, len(agg))
	for _, r := range agg {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].HSNSC != rows[j].HSNSC {
			return rows[i].HSNSC < rows[j].HSNSC
		}
		if rows[i].UQC != rows[j].UQC {
			return rows[i].UQC < rows[j].UQC
		}
		return rows[i].Rt < rows[j].Rt
	})
	for i := range rows {
		rows[i].Num = i + 1
		rows[i].Qty = round2(rows[i].Qty)
	}
	return rows
}

// buildDocIssue derives Table 13's document-series summary from the actual
// numbers issued in the period, grouped by series prefix (everything before
// the number's last "/" -- e.g. "INV/26-27/001" groups under "INV/26-27").
// doc_num follows the widely-corroborated (SECONDARY) Table-13 nature
// numbering: 1 = outward-supply invoices, 5 = credit notes.
func buildDocIssue(invoices []Invoice, creditNotes []CreditNote) []gstr1DocSeries {
	out := []gstr1DocSeries{}

	invoiceRefs := make([]docIssueRef, 0, len(invoices))
	for _, inv := range invoices {
		invoiceRefs = append(invoiceRefs, docIssueRef{
			number:    inv.InvoiceNumber,
			cancelled: inv.Status == "Cancelled" || inv.Status == "Void",
		})
	}
	if docs := groupDocIssueSeries(invoiceRefs); len(docs) > 0 {
		out = append(out, gstr1DocSeries{DocNum: 1, Docs: docs})
	}

	cnRefs := make([]docIssueRef, 0, len(creditNotes))
	for _, cn := range creditNotes {
		cnRefs = append(cnRefs, docIssueRef{number: cn.CreditNoteNumber})
	}
	if docs := groupDocIssueSeries(cnRefs); len(docs) > 0 {
		out = append(out, gstr1DocSeries{DocNum: 5, Docs: docs})
	}

	return out
}

type docIssueRef struct {
	number    string
	cancelled bool
}

func groupDocIssueSeries(refs []docIssueRef) []gstr1DocSeriesEntry {
	type bucket struct {
		numbers   []string
		cancelled int
	}
	bySeries := map[string]*bucket{}
	for _, ref := range refs {
		series := docIssueSeriesKey(ref.number)
		b := bySeries[series]
		if b == nil {
			b = &bucket{}
			bySeries[series] = b
		}
		b.numbers = append(b.numbers, ref.number)
		if ref.cancelled {
			b.cancelled++
		}
	}

	seriesNames := make([]string, 0, len(bySeries))
	for s := range bySeries {
		seriesNames = append(seriesNames, s)
	}
	sort.Strings(seriesNames)

	entries := make([]gstr1DocSeriesEntry, 0, len(seriesNames))
	for _, s := range seriesNames {
		b := bySeries[s]
		nums := append([]string(nil), b.numbers...)
		sort.Strings(nums) // safe: fixed-width zero-padded sequence numbers
		entries = append(entries, gstr1DocSeriesEntry{
			Num:      s,
			From:     nums[0],
			To:       nums[len(nums)-1],
			Totnum:   len(nums),
			Cancel:   b.cancelled,
			NetIssue: len(nums) - b.cancelled,
		})
	}
	return entries
}

// docIssueSeriesKey strips the trailing "/{seq}" segment off a document
// number to get its series prefix.
func docIssueSeriesKey(number string) string {
	if idx := strings.LastIndex(number, "/"); idx >= 0 {
		return number[:idx]
	}
	return number
}
