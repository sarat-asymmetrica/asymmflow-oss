package saudi

import (
	"encoding/asn1"
	"encoding/pem"
	"strings"
	"testing"
)

func testCSRConfig() CSRConfig {
	return CSRConfig{
		CommonName:         "TST-886431145-399999999900003",
		Organization:       "Wasela Café LLC",
		OrganizationalUnit: "Riyadh Branch",
		VATNumber:          "310122393500003",
		SolutionName:       "AsymmFlow",
		ModelVersion:       "2.3",
		DeviceSerial:       "ed22f1d8-e6a2-1118-9b58-d9a8f11e445f",
		InvoiceTypeFlags:   "1100",
		RegisteredAddress:  "King Fahd Rd, Riyadh",
		BusinessCategory:   "Hospitality",
		Environment:        EnvSandbox,
	}
}

// walkCSR parses the CertificationRequest enough to check the profile —
// stdlib x509.ParseCertificateRequest rejects secp256k1 SPKIs, so the walk
// is manual, mirroring ParseCertificateDER's approach.
func walkCSR(t *testing.T, der []byte) (criDER []byte, signature []byte, body string) {
	t.Helper()
	var outer struct {
		CRI       asn1.RawValue
		Algorithm asn1.RawValue
		Signature asn1.BitString
	}
	rest, err := asn1.Unmarshal(der, &outer)
	if err != nil {
		t.Fatalf("CSR does not parse: %v", err)
	}
	if len(rest) != 0 {
		t.Fatal("trailing bytes after CSR")
	}
	return outer.CRI.FullBytes, outer.Signature.Bytes, string(outer.CRI.FullBytes)
}

func TestGenerateCSR_ProfileAndSignature(t *testing.T) {
	key, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	der, pemStr, err := GenerateCSR(key, testCSRConfig())
	if err != nil {
		t.Fatal(err)
	}

	criDER, sig, body := walkCSR(t, der)

	// The signature must verify over the CertificationRequestInfo with the
	// same key — a wrong-bytes CSR fails ZATCA at the door.
	if !key.Verify(criDER, sig) {
		t.Fatal("CSR signature does not verify over CertificationRequestInfo")
	}

	// Profile strings present in the DER (UTF8-encoded, so directly findable).
	for _, want := range []string{
		"TSTZATCA-Code-Signing", // sandbox template name
		"1-AsymmFlow|2-2.3|3-ed22f1d8-e6a2-1118-9b58-d9a8f11e445f", // EGS serial (SAN surname)
		"310122393500003",      // VAT number (SAN UID)
		"1100",                 // invoice type flags (SAN title)
		"King Fahd Rd, Riyadh", // registeredAddress
		"Hospitality",          // businessCategory
		"Wasela Café LLC",      // UTF8String survives non-ASCII
		"SA",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("CSR body missing %q", want)
		}
	}

	// PEM block round-trips to the same DER.
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		t.Fatalf("bad PEM: %v", pemStr[:40])
	}
	if string(block.Bytes) != string(der) {
		t.Fatal("PEM does not round-trip to DER")
	}

	// The MS template OID and SAN OID must be present.
	for _, oid := range []asn1.ObjectIdentifier{oidCertificateTemplateName, oidSubjectAltName, oidExtensionRequest} {
		oidDER, _ := asn1.Marshal(oid)
		if !strings.Contains(body, string(oidDER)) {
			t.Errorf("CSR missing OID %v", oid)
		}
	}
}

func TestGenerateCSR_EnvironmentTemplates(t *testing.T) {
	key, _ := GenerateKeyPair()
	for env, want := range map[Environment]string{
		EnvSandbox:    "TSTZATCA-Code-Signing",
		EnvSimulation: "PREZATCA-Code-Signing",
		EnvProduction: "ZATCA-Code-Signing",
	} {
		cfg := testCSRConfig()
		cfg.Environment = env
		der, _, err := GenerateCSR(key, cfg)
		if err != nil {
			t.Fatalf("%s: %v", env, err)
		}
		if !strings.Contains(string(der), want) {
			t.Errorf("%s CSR missing template %q", env, want)
		}
		// Production must not accidentally match by substring only: check the
		// exact UTF8String TLV (tag 0x0C, length, value).
		tlv := append([]byte{0x0c, byte(len(want))}, []byte(want)...)
		if !strings.Contains(string(der), string(tlv)) {
			t.Errorf("%s CSR template not encoded as UTF8String TLV", env)
		}
	}
}

func TestGenerateCSR_Validation(t *testing.T) {
	key, _ := GenerateKeyPair()
	bad := func(mutate func(*CSRConfig), why string) {
		cfg := testCSRConfig()
		mutate(&cfg)
		if _, _, err := GenerateCSR(key, cfg); err == nil {
			t.Errorf("accepted invalid config: %s", why)
		}
	}
	bad(func(c *CSRConfig) { c.VATNumber = "123456789012345" }, "VAT not starting/ending with 3")
	bad(func(c *CSRConfig) { c.VATNumber = "31012239350003" }, "14-digit VAT")
	bad(func(c *CSRConfig) { c.InvoiceTypeFlags = "2100" }, "non-binary type flags")
	bad(func(c *CSRConfig) { c.InvoiceTypeFlags = "110" }, "3-digit type flags")
	bad(func(c *CSRConfig) { c.CommonName = " " }, "blank common name")
	bad(func(c *CSRConfig) { c.DeviceSerial = "" }, "missing device serial")
	bad(func(c *CSRConfig) { c.Environment = "weird" }, "unknown environment")
	if _, _, err := GenerateCSR(nil, testCSRConfig()); err == nil {
		t.Error("accepted nil key")
	}
}
