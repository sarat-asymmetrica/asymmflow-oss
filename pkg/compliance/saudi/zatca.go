package saudi

// ZATCA Phase 2 (Fatoora) e-invoice generation: UBL 2.1 XML with the KSA
// profile, the tamper-evident ICV/PIH chain, XAdES-BES enveloped signature
// (ECDSA secp256k1 + SHA-256), and the Phase-2 QR stamp for simplified
// invoices.
//
// Design: ONE builder emits the document; the pre-signature form (no
// UBLExtensions / no QR reference / no cac:Signature — exactly the element
// set ZATCA excludes from the invoice hash) and the final signed form come
// from the same code path, so the hashed bytes and the submitted bytes can
// never structurally diverge. The XML is emitted in canonical form
// (deterministic order, explicit end tags, canonical escaping); see
// InvoiceHashB64 for the canonicalization boundary.

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// UNTDID 1001 document type codes used by ZATCA.
const (
	TypeTaxInvoice = 388
	TypeCreditNote = 381
	TypeDebitNote  = 383
	TypePrepayment = 386
)

// Subtype is the InvoiceTypeCode/@name transaction code: "01"/"02" plus five
// one-digit flags (ZATCA's 7-position layout; some vendor docs describe 9 —
// the width is a spec-version parameter, default 7).
type Subtype struct {
	Simplified bool // true → "02" (B2C, reported); false → "01" (B2B, cleared)
	ThirdParty bool
	Nominal    bool
	Exports    bool
	Summary    bool
	SelfBilled bool
}

// Code renders the 7-digit transaction code, e.g. "0100000" / "0200000".
func (s Subtype) Code() string {
	code := "01"
	if s.Simplified {
		code = "02"
	}
	for _, flag := range []bool{s.ThirdParty, s.Nominal, s.Exports, s.Summary, s.SelfBilled} {
		if flag {
			code += "1"
		} else {
			code += "0"
		}
	}
	return code
}

// Party is a seller or buyer.
type Party struct {
	RegistrationName string
	VATNumber        string // 15 digits, starts/ends with 3 (seller mandatory)
	CRN              string // commercial registration number (scheme CRN)
	Street           string
	BuildingNumber   string // 4 digits for KSA sellers
	District         string
	City             string
	PostalCode       string // 5 digits for KSA sellers
	CountryCode      string // ISO-3166 alpha-2, e.g. "SA"
}

// Line is one invoice line. Category maps to ZATCA's S/Z/E/O tax category
// via CategoryCode; Z/E/O lines must carry an exemption reason.
type Line struct {
	Name                string
	Quantity            float64
	UnitPrice           float64 // net unit price (before VAT)
	TaxRate             float64 // 0.15 or 0
	Category            string  // e.g. "services", "exports", "financial_services"
	ExemptionReasonCode string  // e.g. "VATEX-SA-32" — required when rate is 0
	ExemptionReason     string
}

// EInvoice is the document handed to GenerateXML/Sign.
type EInvoice struct {
	ID       string // seller's human invoice number
	UUID     string // v4, distinct from ID
	IssuedAt time.Time
	TypeCode int // TypeTaxInvoice / TypeCreditNote / TypeDebitNote
	Subtype  Subtype
	Currency string // document currency; VAT totals are always also in SAR

	ICV int64  // invoice counter value (monotonic per EGS, never resets)
	PIH string // previous invoice hash (GenesisPIH for the first)

	Seller Party
	Buyer  *Party // required for standard (01) invoices

	Lines []Line

	// For credit/debit notes: the original invoice and the reason.
	BillingReferenceID string
	InstructionNote    string
}

// Totals computed from the lines.
type Totals struct {
	LineExtension float64 // sum of net line amounts
	TaxAmount     float64 // total VAT
	TaxInclusive  float64
	Subtotals     []TaxSubtotal
}

// TaxSubtotal is the per-category tax breakdown.
type TaxSubtotal struct {
	CategoryID    string // S/Z/E/O
	Rate          float64
	TaxableAmount float64
	TaxAmount     float64
	ReasonCode    string
	Reason        string
}

// ComputeTotals derives line/category/document totals with 2-decimal (halala)
// rounding at each aggregation point, per BR-KSA arithmetic rules.
func (inv *EInvoice) ComputeTotals() Totals {
	type bucket struct {
		taxable, tax float64
		rate         float64
		reasonCode   string
		reason       string
	}
	buckets := map[string]*bucket{}
	var totals Totals
	for _, line := range inv.Lines {
		net := round2(line.Quantity * line.UnitPrice)
		vat := round2(net * line.TaxRate)
		totals.LineExtension = round2(totals.LineExtension + net)
		totals.TaxAmount = round2(totals.TaxAmount + vat)

		cat := CategoryCode(line.Category)
		b, ok := buckets[cat]
		if !ok {
			b = &bucket{rate: line.TaxRate, reasonCode: line.ExemptionReasonCode, reason: line.ExemptionReason}
			buckets[cat] = b
		}
		b.taxable = round2(b.taxable + net)
		b.tax = round2(b.tax + vat)
	}
	totals.TaxInclusive = round2(totals.LineExtension + totals.TaxAmount)

	cats := make([]string, 0, len(buckets))
	for c := range buckets {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, c := range cats {
		b := buckets[c]
		totals.Subtotals = append(totals.Subtotals, TaxSubtotal{
			CategoryID: c, Rate: b.rate, TaxableAmount: b.taxable, TaxAmount: b.tax,
			ReasonCode: b.reasonCode, Reason: b.reason,
		})
	}
	return totals
}

// Validate checks structural requirements before generation.
func (inv *EInvoice) Validate() error {
	var errs []string
	if strings.TrimSpace(inv.ID) == "" {
		errs = append(errs, "ID is required")
	}
	if strings.TrimSpace(inv.UUID) == "" {
		errs = append(errs, "UUID is required")
	}
	if inv.IssuedAt.IsZero() {
		errs = append(errs, "IssuedAt is required")
	}
	switch inv.TypeCode {
	case TypeTaxInvoice, TypeCreditNote, TypeDebitNote, TypePrepayment:
	default:
		errs = append(errs, fmt.Sprintf("TypeCode %d is not a ZATCA document type", inv.TypeCode))
	}
	if inv.ICV < 1 {
		errs = append(errs, "ICV must be >= 1")
	}
	if strings.TrimSpace(inv.PIH) == "" {
		errs = append(errs, "PIH is required (GenesisPIH for the first invoice)")
	}
	if !ValidVATNumber(inv.Seller.VATNumber) {
		errs = append(errs, "seller VAT number must be 15 digits starting and ending with 3")
	}
	if strings.TrimSpace(inv.Seller.RegistrationName) == "" {
		errs = append(errs, "seller registration name is required")
	}
	if !inv.Subtype.Simplified {
		if inv.Buyer == nil {
			errs = append(errs, "standard (01) invoices require a buyer")
		} else if !ValidVATNumber(inv.Buyer.VATNumber) {
			errs = append(errs, "standard (01) invoices require a valid buyer VAT number")
		}
	}
	if len(inv.Lines) == 0 {
		errs = append(errs, "at least one line is required")
	}
	for i, line := range inv.Lines {
		if line.Quantity <= 0 {
			errs = append(errs, fmt.Sprintf("line %d: quantity must be positive", i+1))
		}
		if line.UnitPrice < 0 {
			errs = append(errs, fmt.Sprintf("line %d: unit price cannot be negative", i+1))
		}
		if line.TaxRate != 0 && math.Abs(line.TaxRate-StandardVATRate) > 1e-9 {
			errs = append(errs, fmt.Sprintf("line %d: tax rate must be 0 or 0.15", i+1))
		}
		if CategoryCode(line.Category) != CategoryStandard && strings.TrimSpace(line.ExemptionReasonCode) == "" {
			errs = append(errs, fmt.Sprintf("line %d: %s-category lines require an exemption reason code", i+1, CategoryCode(line.Category)))
		}
	}
	if (inv.TypeCode == TypeCreditNote || inv.TypeCode == TypeDebitNote) &&
		(strings.TrimSpace(inv.BillingReferenceID) == "" || strings.TrimSpace(inv.InstructionNote) == "") {
		errs = append(errs, "credit/debit notes require BillingReferenceID and InstructionNote")
	}
	if len(errs) > 0 {
		return errors.New("zatca invoice invalid: " + strings.Join(errs, "; "))
	}
	return nil
}

// GenerateXML emits the pre-signature invoice XML — the exact byte sequence
// the invoice hash is computed over.
func (inv *EInvoice) GenerateXML() ([]byte, error) {
	if err := inv.Validate(); err != nil {
		return nil, err
	}
	return inv.buildXML(nil), nil
}

// signedArtifacts carries everything spliced into the final signed document.
type signedArtifacts struct {
	InvoiceHashB64   string
	SignatureB64     string
	SignedProperties string // rendered <xades:SignedProperties> block
	SignedPropsHash  string // base64 SHA-256 of that block
	CertificateB64   string // base64 DER of the CSID certificate
	QRBase64         string // TLV QR content ("" for standard invoices)
}

// buildXML renders the invoice. artifacts == nil → pre-signature form
// (no UBLExtensions, no QR reference, no cac:Signature).
func (inv *EInvoice) buildXML(artifacts *signedArtifacts) []byte {
	var b strings.Builder
	b.Grow(8192)

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2" xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2" xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2" xmlns:ext="urn:oasis:names:specification:ubl:schema:xsd:CommonExtensionComponents-2">`)

	if artifacts != nil {
		b.WriteString(renderUBLExtensions(artifacts))
	}

	tag(&b, "cbc:ProfileID", "reporting:1.0")
	tag(&b, "cbc:ID", esc(inv.ID))
	tag(&b, "cbc:UUID", esc(inv.UUID))
	tag(&b, "cbc:IssueDate", inv.IssuedAt.Format("2006-01-02"))
	tag(&b, "cbc:IssueTime", inv.IssuedAt.Format("15:04:05"))
	fmt.Fprintf(&b, `<cbc:InvoiceTypeCode name="%s">%d</cbc:InvoiceTypeCode>`, inv.Subtype.Code(), inv.TypeCode)
	tag(&b, "cbc:DocumentCurrencyCode", esc(inv.Currency))
	tag(&b, "cbc:TaxCurrencyCode", "SAR")

	if inv.BillingReferenceID != "" {
		b.WriteString(`<cac:BillingReference><cac:InvoiceDocumentReference>`)
		tag(&b, "cbc:ID", esc(inv.BillingReferenceID))
		b.WriteString(`</cac:InvoiceDocumentReference></cac:BillingReference>`)
	}

	// ICV and PIH document references (KSA rules KSA-16).
	b.WriteString(`<cac:AdditionalDocumentReference>`)
	tag(&b, "cbc:ID", "ICV")
	tag(&b, "cbc:UUID", fmt.Sprintf("%d", inv.ICV))
	b.WriteString(`</cac:AdditionalDocumentReference>`)

	b.WriteString(`<cac:AdditionalDocumentReference>`)
	tag(&b, "cbc:ID", "PIH")
	b.WriteString(`<cac:Attachment>`)
	fmt.Fprintf(&b, `<cbc:EmbeddedDocumentBinaryObject mimeCode="text/plain">%s</cbc:EmbeddedDocumentBinaryObject>`, esc(inv.PIH))
	b.WriteString(`</cac:Attachment>`)
	b.WriteString(`</cac:AdditionalDocumentReference>`)

	if artifacts != nil && artifacts.QRBase64 != "" {
		b.WriteString(`<cac:AdditionalDocumentReference>`)
		tag(&b, "cbc:ID", "QR")
		b.WriteString(`<cac:Attachment>`)
		fmt.Fprintf(&b, `<cbc:EmbeddedDocumentBinaryObject mimeCode="text/plain">%s</cbc:EmbeddedDocumentBinaryObject>`, artifacts.QRBase64)
		b.WriteString(`</cac:Attachment>`)
		b.WriteString(`</cac:AdditionalDocumentReference>`)
	}
	if artifacts != nil {
		// UBL signature placeholder referencing the XAdES signature.
		b.WriteString(`<cac:Signature><cbc:ID>urn:oasis:names:specification:ubl:signature:Invoice</cbc:ID><cbc:SignatureMethod>urn:oasis:names:specification:ubl:dsig:enveloped:xades</cbc:SignatureMethod></cac:Signature>`)
	}

	renderParty(&b, "cac:AccountingSupplierParty", inv.Seller)
	if inv.Buyer != nil {
		renderParty(&b, "cac:AccountingCustomerParty", *inv.Buyer)
	} else {
		// Simplified invoices still carry an (empty-buyer) customer block.
		b.WriteString(`<cac:AccountingCustomerParty><cac:Party></cac:Party></cac:AccountingCustomerParty>`)
	}

	// Credit/debit notes carry their reason as PaymentMeans/InstructionNote —
	// the location BR-KSA-17 validates (KSA-10), not a free-text cbc:Note.
	// PaymentMeansCode 10 = "in cash" (UNTDID 4461), the reference default.
	if inv.InstructionNote != "" {
		b.WriteString(`<cac:PaymentMeans><cbc:PaymentMeansCode>10</cbc:PaymentMeansCode>`)
		tag(&b, "cbc:InstructionNote", esc(inv.InstructionNote))
		b.WriteString(`</cac:PaymentMeans>`)
	}

	totals := inv.ComputeTotals()

	// Document-level TaxTotal in document currency with subtotals…
	fmt.Fprintf(&b, `<cac:TaxTotal><cbc:TaxAmount currencyID="%s">%s</cbc:TaxAmount>`, esc(inv.Currency), money2(totals.TaxAmount))
	for _, st := range totals.Subtotals {
		fmt.Fprintf(&b, `<cac:TaxSubtotal><cbc:TaxableAmount currencyID="%s">%s</cbc:TaxableAmount><cbc:TaxAmount currencyID="%s">%s</cbc:TaxAmount><cac:TaxCategory><cbc:ID>%s</cbc:ID><cbc:Percent>%s</cbc:Percent>`,
			esc(inv.Currency), money2(st.TaxableAmount), esc(inv.Currency), money2(st.TaxAmount), st.CategoryID, percent(st.Rate))
		if st.CategoryID != CategoryStandard {
			tag(&b, "cbc:TaxExemptionReasonCode", esc(st.ReasonCode))
			if st.Reason != "" {
				tag(&b, "cbc:TaxExemptionReason", esc(st.Reason))
			}
		}
		b.WriteString(`<cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:TaxCategory></cac:TaxSubtotal>`)
	}
	b.WriteString(`</cac:TaxTotal>`)
	// …plus the mandatory second TaxTotal carrying only the SAR tax amount.
	fmt.Fprintf(&b, `<cac:TaxTotal><cbc:TaxAmount currencyID="SAR">%s</cbc:TaxAmount></cac:TaxTotal>`, money2(totals.TaxAmount))

	fmt.Fprintf(&b, `<cac:LegalMonetaryTotal><cbc:LineExtensionAmount currencyID="%[1]s">%[2]s</cbc:LineExtensionAmount><cbc:TaxExclusiveAmount currencyID="%[1]s">%[2]s</cbc:TaxExclusiveAmount><cbc:TaxInclusiveAmount currencyID="%[1]s">%[3]s</cbc:TaxInclusiveAmount><cbc:AllowanceTotalAmount currencyID="%[1]s">0.00</cbc:AllowanceTotalAmount><cbc:PrepaidAmount currencyID="%[1]s">0.00</cbc:PrepaidAmount><cbc:PayableAmount currencyID="%[1]s">%[3]s</cbc:PayableAmount></cac:LegalMonetaryTotal>`,
		esc(inv.Currency), money2(totals.LineExtension), money2(totals.TaxInclusive))

	for i, line := range inv.Lines {
		net := round2(line.Quantity * line.UnitPrice)
		vat := round2(net * line.TaxRate)
		cat := CategoryCode(line.Category)
		fmt.Fprintf(&b, `<cac:InvoiceLine><cbc:ID>%d</cbc:ID><cbc:InvoicedQuantity unitCode="PCE">%s</cbc:InvoicedQuantity><cbc:LineExtensionAmount currencyID="%s">%s</cbc:LineExtensionAmount>`,
			i+1, trimFloat(line.Quantity), esc(inv.Currency), money2(net))
		fmt.Fprintf(&b, `<cac:TaxTotal><cbc:TaxAmount currencyID="%[1]s">%[2]s</cbc:TaxAmount><cbc:RoundingAmount currencyID="%[1]s">%[3]s</cbc:RoundingAmount></cac:TaxTotal>`,
			esc(inv.Currency), money2(vat), money2(round2(net+vat)))
		b.WriteString(`<cac:Item>`)
		tag(&b, "cbc:Name", esc(line.Name))
		fmt.Fprintf(&b, `<cac:ClassifiedTaxCategory><cbc:ID>%s</cbc:ID><cbc:Percent>%s</cbc:Percent><cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:ClassifiedTaxCategory>`, cat, percent(line.TaxRate))
		b.WriteString(`</cac:Item>`)
		fmt.Fprintf(&b, `<cac:Price><cbc:PriceAmount currencyID="%s">%s</cbc:PriceAmount></cac:Price>`, esc(inv.Currency), money2(line.UnitPrice))
		b.WriteString(`</cac:InvoiceLine>`)
	}

	b.WriteString(`</Invoice>`)
	return []byte(b.String())
}

// Sign produces the final, submittable invoice: computes the invoice hash
// over the pre-signature XML, builds the XAdES SignedProperties, signs with
// the EGS key, assembles the QR stamp (simplified invoices only — for
// standard invoices ZATCA generates the QR at clearance), and splices the
// UBLExtensions + QR + Signature into the document.
type SignedInvoice struct {
	XML            []byte
	InvoiceHashB64 string
	SignatureB64   string
	QRBase64       string // empty for standard invoices
}

func (inv *EInvoice) Sign(key *KeyPair, cert *Certificate, signingTime time.Time) (*SignedInvoice, error) {
	preXML, err := inv.GenerateXML()
	if err != nil {
		return nil, err
	}
	hashB64 := InvoiceHashB64(preXML)

	signedProps := renderSignedProperties(cert, signingTime)
	signedPropsHash := InvoiceHashB64([]byte(signedProps))

	// XMLDSig: the signature is computed over SignedInfo, which binds both
	// the invoice digest and the SignedProperties digest.
	signedInfo := renderSignedInfo(hashB64, signedPropsHash)
	signatureB64 := key.SignBase64([]byte(signedInfo))

	artifacts := &signedArtifacts{
		InvoiceHashB64:   hashB64,
		SignatureB64:     signatureB64,
		SignedProperties: signedProps,
		SignedPropsHash:  signedPropsHash,
		CertificateB64:   base64Std(cert.Raw),
	}

	if inv.Subtype.Simplified {
		totals := inv.ComputeTotals()
		pub, err := key.PublicKeyDER()
		if err != nil {
			return nil, err
		}
		sigBytes := key.SignBytes([]byte(signedInfo))
		qr, err := EncodeQR(QRData{
			SellerName:     inv.Seller.RegistrationName,
			VATNumber:      inv.Seller.VATNumber,
			Timestamp:      inv.IssuedAt,
			TotalWithVAT:   money2(totals.TaxInclusive),
			VATTotal:       money2(totals.TaxAmount),
			InvoiceHashB64: hashB64,
			Signature:      sigBytes,
			PublicKey:      pub,
			CertSignature:  cert.SignatureValue,
		})
		if err != nil {
			return nil, err
		}
		artifacts.QRBase64 = qr
	}

	return &SignedInvoice{
		XML:            inv.buildXML(artifacts),
		InvoiceHashB64: hashB64,
		SignatureB64:   signatureB64,
		QRBase64:       artifacts.QRBase64,
	}, nil
}

// ---------- rendering helpers ----------

func renderParty(b *strings.Builder, element string, p Party) {
	b.WriteString("<" + element + "><cac:Party>")
	if p.CRN != "" {
		fmt.Fprintf(b, `<cac:PartyIdentification><cbc:ID schemeID="CRN">%s</cbc:ID></cac:PartyIdentification>`, esc(p.CRN))
	}
	b.WriteString(`<cac:PostalAddress>`)
	tag(b, "cbc:StreetName", esc(p.Street))
	tag(b, "cbc:BuildingNumber", esc(p.BuildingNumber))
	tag(b, "cbc:CitySubdivisionName", esc(p.District))
	tag(b, "cbc:CityName", esc(p.City))
	tag(b, "cbc:PostalZone", esc(p.PostalCode))
	fmt.Fprintf(b, `<cac:Country><cbc:IdentificationCode>%s</cbc:IdentificationCode></cac:Country>`, esc(p.CountryCode))
	b.WriteString(`</cac:PostalAddress>`)
	if p.VATNumber != "" {
		fmt.Fprintf(b, `<cac:PartyTaxScheme><cbc:CompanyID>%s</cbc:CompanyID><cac:TaxScheme><cbc:ID>VAT</cbc:ID></cac:TaxScheme></cac:PartyTaxScheme>`, esc(p.VATNumber))
	}
	fmt.Fprintf(b, `<cac:PartyLegalEntity><cbc:RegistrationName>%s</cbc:RegistrationName></cac:PartyLegalEntity>`, esc(p.RegistrationName))
	b.WriteString(`</cac:Party></` + element + ">")
}

func renderSignedProperties(cert *Certificate, signingTime time.Time) string {
	return `<xades:SignedProperties xmlns:xades="http://uri.etsi.org/01903/v1.3.2#" Id="xadesSignedProperties"><xades:SignedSignatureProperties><xades:SigningTime>` +
		signingTime.UTC().Format("2006-01-02T15:04:05Z") +
		`</xades:SigningTime><xades:SigningCertificate><xades:Cert><xades:CertDigest><ds:DigestMethod xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"></ds:DigestMethod><ds:DigestValue xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		cert.DigestB64() +
		`</ds:DigestValue></xades:CertDigest><xades:IssuerSerial><ds:X509IssuerName xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		esc(cert.IssuerName) +
		`</ds:X509IssuerName><ds:X509SerialNumber xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
		cert.SerialNumber.String() +
		`</ds:X509SerialNumber></xades:IssuerSerial></xades:SigningCertificate></xades:SignedSignatureProperties></xades:SignedProperties>`
}

func renderSignedInfo(invoiceHashB64, signedPropsHashB64 string) string {
	return `<ds:SignedInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:CanonicalizationMethod Algorithm="http://www.w3.org/2006/12/xml-c14n11"></ds:CanonicalizationMethod><ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"></ds:SignatureMethod><ds:Reference Id="invoiceSignedData" URI=""><ds:Transforms><ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116"><ds:XPath>not(//ancestor-or-self::ext:UBLExtensions)</ds:XPath></ds:Transform><ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116"><ds:XPath>not(//ancestor-or-self::cac:Signature)</ds:XPath></ds:Transform><ds:Transform Algorithm="http://www.w3.org/TR/1999/REC-xpath-19991116"><ds:XPath>not(//ancestor-or-self::cac:AdditionalDocumentReference[cbc:ID='QR'])</ds:XPath></ds:Transform><ds:Transform Algorithm="http://www.w3.org/2006/12/xml-c14n11"></ds:Transform></ds:Transforms><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"></ds:DigestMethod><ds:DigestValue>` +
		invoiceHashB64 +
		`</ds:DigestValue></ds:Reference><ds:Reference Type="http://www.w3.org/2000/09/xmldsig#SignatureProperties" URI="#xadesSignedProperties"><ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"></ds:DigestMethod><ds:DigestValue>` +
		signedPropsHashB64 +
		`</ds:DigestValue></ds:Reference></ds:SignedInfo>`
}

func renderUBLExtensions(a *signedArtifacts) string {
	var b strings.Builder
	b.WriteString(`<ext:UBLExtensions><ext:UBLExtension><ext:ExtensionURI>urn:oasis:names:specification:ubl:dsig:enveloped:xades</ext:ExtensionURI><ext:ExtensionContent><sig:UBLDocumentSignatures xmlns:sig="urn:oasis:names:specification:ubl:schema:xsd:CommonSignatureComponents-2" xmlns:sac="urn:oasis:names:specification:ubl:schema:xsd:SignatureAggregateComponents-2" xmlns:sbc="urn:oasis:names:specification:ubl:schema:xsd:SignatureBasicComponents-2"><sac:SignatureInformation><cbc:ID>urn:oasis:names:specification:ubl:signature:1</cbc:ID><sbc:ReferencedSignatureID>urn:oasis:names:specification:ubl:signature:Invoice</sbc:ReferencedSignatureID><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#" Id="signature">`)
	b.WriteString(renderSignedInfo(a.InvoiceHashB64, a.SignedPropsHash))
	b.WriteString(`<ds:SignatureValue>` + a.SignatureB64 + `</ds:SignatureValue>`)
	b.WriteString(`<ds:KeyInfo><ds:X509Data><ds:X509Certificate>` + a.CertificateB64 + `</ds:X509Certificate></ds:X509Data></ds:KeyInfo>`)
	b.WriteString(`<ds:Object><xades:QualifyingProperties xmlns:xades="http://uri.etsi.org/01903/v1.3.2#" Target="signature">`)
	b.WriteString(a.SignedProperties)
	b.WriteString(`</xades:QualifyingProperties></ds:Object></ds:Signature></sac:SignatureInformation></sig:UBLDocumentSignatures></ext:ExtensionContent></ext:UBLExtension></ext:UBLExtensions>`)
	return b.String()
}

func tag(b *strings.Builder, name, escapedValue string) {
	b.WriteString("<" + name + ">" + escapedValue + "</" + name + ">")
}

// esc escapes text content / attribute values canonically.
func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func money2(v float64) string { return fmt.Sprintf("%.2f", v) }

// percent renders a tax rate fraction as a percentage (0.15 → "15.00").
func percent(rate float64) string { return fmt.Sprintf("%.2f", rate*100) }

// trimFloat renders quantities without trailing zeros (2 → "2", 1.5 → "1.5").
func trimFloat(v float64) string {
	s := fmt.Sprintf("%.6f", v)
	s = strings.TrimRight(s, "0")
	return strings.TrimRight(s, ".")
}

func base64Std(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}
