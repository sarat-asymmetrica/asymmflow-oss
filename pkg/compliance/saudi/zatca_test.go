package saudi

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

// makeTestCert wraps NewSelfSignedCertificate (crypto.go) with test plumbing.
func makeTestCert(t *testing.T, key *KeyPair) *Certificate {
	t.Helper()
	cert, err := NewSelfSignedCertificate(key, "TSTZATCA-Code-Signing-CA", "SA",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewSelfSignedCertificate: %v", err)
	}
	return cert
}

func sampleSimplifiedInvoice() *EInvoice {
	return &EInvoice{
		ID:       "SME00062",
		UUID:     "8e6000cf-1a98-4174-b3e7-b5d5954bc10d",
		IssuedAt: time.Date(2026, 7, 3, 14, 40, 40, 0, time.UTC),
		TypeCode: TypeTaxInvoice,
		Subtype:  Subtype{Simplified: true},
		Currency: "SAR",
		ICV:      1,
		PIH:      GenesisPIH,
		Seller: Party{
			RegistrationName: "Wasela Café LLC",
			VATNumber:        "310122393500003",
			CRN:              "1010010000",
			Street:           "King Fahd Road",
			BuildingNumber:   "1234",
			District:         "Al Olaya",
			City:             "Riyadh",
			PostalCode:       "12244",
			CountryCode:      "SA",
		},
		Lines: []Line{
			{Name: "Karak chai", Quantity: 2, UnitPrice: 8.48, TaxRate: 0.15, Category: "services"},
			{Name: "Shakshuka plate", Quantity: 1, UnitPrice: 70.00, TaxRate: 0.15, Category: "services"},
		},
	}
}

func sampleStandardInvoice() *EInvoice {
	inv := sampleSimplifiedInvoice()
	inv.ID = "INV-2026-0044"
	inv.Subtype = Subtype{Simplified: false}
	inv.Buyer = &Party{
		RegistrationName: "Najd Trading Co",
		VATNumber:        "399999999900003",
		Street:           "Prince Sultan Road",
		BuildingNumber:   "5678",
		District:         "Al Malaz",
		City:             "Riyadh",
		PostalCode:       "12629",
		CountryCode:      "SA",
	}
	return inv
}

func TestSubtypeCodes(t *testing.T) {
	cases := []struct {
		s    Subtype
		want string
	}{
		{Subtype{}, "0100000"},
		{Subtype{Simplified: true}, "0200000"},
		{Subtype{Simplified: true, Summary: true}, "0200010"},
		{Subtype{SelfBilled: true}, "0100001"},
		{Subtype{Exports: true}, "0100100"},
	}
	for _, tc := range cases {
		if got := tc.s.Code(); got != tc.want {
			t.Errorf("Subtype%+v.Code() = %s, want %s", tc.s, got, tc.want)
		}
	}
}

func TestComputeTotalsCafeScenario(t *testing.T) {
	totals := sampleSimplifiedInvoice().ComputeTotals()
	// 2×8.48=16.96 → VAT 2.54; 70.00 → VAT 10.50; net 86.96, VAT 13.04, gross 100.00.
	if totals.LineExtension != 86.96 {
		t.Errorf("net = %v, want 86.96", totals.LineExtension)
	}
	if totals.TaxAmount != 13.04 {
		t.Errorf("VAT = %v, want 13.04", totals.TaxAmount)
	}
	if totals.TaxInclusive != 100.00 {
		t.Errorf("gross = %v, want 100.00", totals.TaxInclusive)
	}
	if len(totals.Subtotals) != 1 || totals.Subtotals[0].CategoryID != "S" {
		t.Errorf("subtotals = %+v", totals.Subtotals)
	}
}

func TestValidateCatchesStructuralDefects(t *testing.T) {
	std := sampleStandardInvoice()
	std.Buyer = nil
	if err := std.Validate(); err == nil || !strings.Contains(err.Error(), "require a buyer") {
		t.Errorf("standard without buyer: %v", err)
	}

	zr := sampleSimplifiedInvoice()
	zr.Lines[0].TaxRate = 0
	zr.Lines[0].Category = "exports"
	if err := zr.Validate(); err == nil || !strings.Contains(err.Error(), "exemption reason") {
		t.Errorf("zero-rated without reason: %v", err)
	}

	cn := sampleSimplifiedInvoice()
	cn.TypeCode = TypeCreditNote
	if err := cn.Validate(); err == nil || !strings.Contains(err.Error(), "credit/debit notes require") {
		t.Errorf("credit note without reference: %v", err)
	}

	bad := sampleSimplifiedInvoice()
	bad.ICV = 0
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "ICV") {
		t.Errorf("ICV 0: %v", err)
	}
}

func TestGenerateXMLStructure(t *testing.T) {
	xml, err := sampleSimplifiedInvoice().GenerateXML()
	if err != nil {
		t.Fatalf("GenerateXML: %v", err)
	}
	s := string(xml)
	for _, want := range []string{
		`<cbc:ProfileID>reporting:1.0</cbc:ProfileID>`,
		`<cbc:InvoiceTypeCode name="0200000">388</cbc:InvoiceTypeCode>`,
		`<cbc:ID>ICV</cbc:ID><cbc:UUID>1</cbc:UUID>`,
		`<cbc:ID>PIH</cbc:ID>`,
		GenesisPIH,
		`<cbc:CompanyID>310122393500003</cbc:CompanyID>`,
		`<cbc:TaxCurrencyCode>SAR</cbc:TaxCurrencyCode>`,
		`<cac:TaxTotal><cbc:TaxAmount currencyID="SAR">13.04</cbc:TaxAmount></cac:TaxTotal>`,
		`<cbc:PayableAmount currencyID="SAR">100.00</cbc:PayableAmount>`,
		`<cbc:Percent>15.00</cbc:Percent>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("XML missing %q", want)
		}
	}
	// Pre-signature form must NOT contain signing artifacts.
	for _, forbidden := range []string{"UBLExtensions", ">QR<", "ds:Signature"} {
		if strings.Contains(s, forbidden) {
			t.Errorf("pre-signature XML must not contain %q", forbidden)
		}
	}
}

func TestInvoiceHashDeterministicAndSensitive(t *testing.T) {
	a, _ := sampleSimplifiedInvoice().GenerateXML()
	b, _ := sampleSimplifiedInvoice().GenerateXML()
	if InvoiceHashB64(a) != InvoiceHashB64(b) {
		t.Error("hash not deterministic")
	}
	mutated := sampleSimplifiedInvoice()
	mutated.Lines[0].UnitPrice = 8.49
	m, _ := mutated.GenerateXML()
	if InvoiceHashB64(a) == InvoiceHashB64(m) {
		t.Error("hash must change when a line changes")
	}
	// Sanity: hash is base64 of a 32-byte digest.
	raw, err := base64.StdEncoding.DecodeString(InvoiceHashB64(a))
	if err != nil || len(raw) != sha256.Size {
		t.Errorf("hash is not base64(SHA-256): err=%v len=%d", err, len(raw))
	}
}

func TestGenesisPIHConstant(t *testing.T) {
	// GenesisPIH = base64 of the ASCII hex SHA-256 of "0" per ZATCA.
	decoded, err := base64.StdEncoding.DecodeString(GenesisPIH)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("0"))
	hexDigest := ""
	for _, b := range sum {
		hexDigest += string("0123456789abcdef"[b>>4]) + string("0123456789abcdef"[b&0xF])
	}
	if string(decoded) != hexDigest {
		t.Errorf("GenesisPIH decodes to %q, want hex digest %q", decoded, hexDigest)
	}
}

func TestSignSimplifiedInvoiceEndToEnd(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	cert := makeTestCert(t, key)

	inv := sampleSimplifiedInvoice()
	signed, err := inv.Sign(key, cert, time.Date(2026, 7, 3, 14, 41, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	s := string(signed.XML)
	for _, want := range []string{
		"<ext:UBLExtensions>",
		"<ds:SignatureValue>" + signed.SignatureB64 + "</ds:SignatureValue>",
		"<cbc:ID>QR</cbc:ID>",
		signed.QRBase64,
		"xadesSignedProperties",
		"<ds:X509SerialNumber",
		"ecdsa-sha256",
		"xml-c14n11",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("signed XML missing %q", want)
		}
	}

	// The recorded hash must equal the hash of the pre-signature XML.
	pre, _ := inv.GenerateXML()
	if signed.InvoiceHashB64 != InvoiceHashB64(pre) {
		t.Error("recorded invoice hash does not match pre-signature XML hash")
	}

	// The signature must verify over the SignedInfo built from the two digests.
	signedProps := renderSignedProperties(cert, time.Date(2026, 7, 3, 14, 41, 0, 0, time.UTC))
	signedInfo := renderSignedInfo(signed.InvoiceHashB64, InvoiceHashB64([]byte(signedProps)))
	sigDER, err := base64.StdEncoding.DecodeString(signed.SignatureB64)
	if err != nil {
		t.Fatal(err)
	}
	if !key.Verify([]byte(signedInfo), sigDER) {
		t.Error("signature does not verify over SignedInfo")
	}

	// QR: decode and check the stamp tags.
	tags, err := DecodeQR(signed.QRBase64)
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}
	if string(tags[QRTagSellerName]) != "Wasela Café LLC" {
		t.Errorf("tag1 = %q", tags[QRTagSellerName])
	}
	if string(tags[QRTagVATNumber]) != "310122393500003" {
		t.Errorf("tag2 = %q", tags[QRTagVATNumber])
	}
	if string(tags[QRTagTimestamp]) != "2026-07-03T14:40:40Z" {
		t.Errorf("tag3 = %q", tags[QRTagTimestamp])
	}
	if string(tags[QRTagTotalWithVAT]) != "100.00" || string(tags[QRTagVATTotal]) != "13.04" {
		t.Errorf("totals: tag4=%q tag5=%q", tags[QRTagTotalWithVAT], tags[QRTagVATTotal])
	}
	if string(tags[QRTagInvoiceHash]) != signed.InvoiceHashB64 {
		t.Error("tag6 does not match invoice hash")
	}
	if !key.Verify([]byte(signedInfo), tags[QRTagSignature]) {
		t.Error("tag7 signature does not verify")
	}
	pub, _ := key.PublicKeyDER()
	if string(tags[QRTagPublicKey]) != string(pub) {
		t.Error("tag8 is not the EGS public key SPKI")
	}
	if string(tags[QRTagCertSignature]) != string(cert.SignatureValue) {
		t.Error("tag9 is not the certificate signature")
	}
}

func TestSignStandardInvoiceHasNoQR(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	cert := makeTestCert(t, key)
	signed, err := sampleStandardInvoice().Sign(key, cert, time.Now().UTC())
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if signed.QRBase64 != "" {
		t.Error("standard invoices must not self-generate a QR (ZATCA stamps it at clearance)")
	}
	if strings.Contains(string(signed.XML), "<cbc:ID>QR</cbc:ID>") {
		t.Error("standard signed XML must not contain a QR reference")
	}
	if !strings.Contains(string(signed.XML), "<ext:UBLExtensions>") {
		t.Error("standard signed XML must contain the signature extension")
	}
}

func TestCreditNoteCarriesReferenceAndReason(t *testing.T) {
	cn := sampleSimplifiedInvoice()
	cn.TypeCode = TypeCreditNote
	cn.BillingReferenceID = "SME00061"
	cn.InstructionNote = "order cancelled by customer"
	xml, err := cn.GenerateXML()
	if err != nil {
		t.Fatalf("GenerateXML: %v", err)
	}
	s := string(xml)
	if !strings.Contains(s, `name="0200000">381<`) {
		t.Error("credit note type code missing")
	}
	if !strings.Contains(s, "<cac:BillingReference><cac:InvoiceDocumentReference><cbc:ID>SME00061</cbc:ID>") {
		t.Error("billing reference missing")
	}
	if !strings.Contains(s, "order cancelled by customer") {
		t.Error("instruction note missing")
	}
}

func TestCertificateParsing(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	cert := makeTestCert(t, key)
	if cert.SerialNumber == nil || cert.SerialNumber.Sign() <= 0 {
		t.Errorf("serial should be a positive key-derived integer, got %v", cert.SerialNumber)
	}
	if !strings.Contains(cert.IssuerName, "CN=TSTZATCA-Code-Signing-CA") {
		t.Errorf("issuer = %q", cert.IssuerName)
	}
	if len(cert.SignatureValue) == 0 {
		t.Error("certificate signature empty")
	}
	sum := sha256.Sum256(cert.Raw)
	if cert.DigestB64() != base64.StdEncoding.EncodeToString(sum[:]) {
		t.Error("cert digest mismatch")
	}
}

func TestQRRoundTripStandalone(t *testing.T) {
	encoded, err := EncodeQR(QRData{
		SellerName:   "متجر التمور", // Arabic seller names are the common case
		VATNumber:    "310122393500003",
		Timestamp:    time.Date(2026, 7, 3, 9, 0, 0, 0, time.UTC),
		TotalWithVAT: "115.00",
		VATTotal:     "15.00",
	})
	if err != nil {
		t.Fatalf("EncodeQR: %v", err)
	}
	tags, err := DecodeQR(encoded)
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}
	if string(tags[QRTagSellerName]) != "متجر التمور" {
		t.Errorf("UTF-8 seller name mangled: %q", tags[QRTagSellerName])
	}
	if len(tags) != 5 {
		t.Errorf("phase-1 QR should have 5 tags, got %d", len(tags))
	}
	if _, err := EncodeQR(QRData{}); err == nil {
		t.Error("missing mandatory tags should error")
	}
}
