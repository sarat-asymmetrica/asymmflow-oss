package saudi

// Compatibility tests against ZATCA's OWN SDK reference materials
// (testdata/sdk203/, extracted from the SDK zip linked on zatca.gov.sa →
// Systems Developers → Download SDK; SDK v2.03, cli-3.0.8). These are
// ZATCA-published synthetic test artifacts:
//
//   - ec-secp256k1-priv-key.pem / cert.pem — the SDK's EGS reference key and
//     the matching CSID certificate ISSUED BY ZATCA'S TEST CA
//     (TSZEINVOICE-SubCA-1). Both ship as bare base64 with no PEM armor —
//     the form the live gateway's binarySecurityToken uses too.
//   - ubl.xml — the SDK's own XAdES signature template, carrying the exact
//     ds:Reference transform strings (the Wave-2 ❓ flag, closed here
//     against the primary source).
//
// Version caveat (recorded in the research notes): SDK 2.03 predates the
// final QR spec — its validator still expects the draft R/S semantics in
// tags 8/9. The signature template and certificate profile are unchanged in
// current SDKs; QR tag semantics follow the current Security Features
// Standard (7=signature, 8=SPKI, 9=cert signature), cross-checked against
// two current SDK-aligned implementations.

import (
	"encoding/asn1"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "sdk203", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The SDK's reference key and certificate must parse through our
// header-less-base64 paths, belong to each other, and produce verifiable
// signatures — i.e. our crypto handles real ZATCA-issued material.
func TestSDKReferenceMaterials_KeyAndCertParse(t *testing.T) {
	key, err := ParsePrivateKeyPEM(readTestdata(t, "ec-secp256k1-priv-key.pem"))
	if err != nil {
		t.Fatalf("SDK reference key does not parse: %v", err)
	}
	cert, err := ParseCertificatePEM(readTestdata(t, "cert.pem"))
	if err != nil {
		t.Fatalf("SDK reference certificate does not parse: %v", err)
	}

	if !strings.Contains(cert.IssuerName, "TSZEINVOICE-SubCA-1") {
		t.Errorf("issuer = %q, want the ZATCA test CA", cert.IssuerName)
	}
	if cert.SerialNumber == nil || cert.SerialNumber.Sign() <= 0 {
		t.Error("certificate serial missing")
	}
	if len(cert.SignatureValue) == 0 {
		t.Error("certificate signature (QR tag 9) missing")
	}

	// The key and certificate are a pair: the cert's SPKI point must equal
	// the key's public half.
	spki, err := key.PublicKeyDER()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cert.Raw), string(spki[len(spki)-65:])) {
		t.Error("SDK certificate public key does not match the SDK private key")
	}

	// Sign-and-verify round-trip with the reference key.
	msg := []byte("zatca sdk compatibility probe")
	if !key.Verify(msg, key.SignBytes(msg)) {
		t.Error("reference key cannot verify its own signature")
	}
}

// The ZATCA test CA encodes the EGS serial in the SAN as SURNAME (2.5.4.4) —
// the OpenSSL "SN" quirk W3-D8 documents. Pin it against the primary source
// so nobody "fixes" our CSR to serialNumber (2.5.4.5).
func TestSDKReferenceCert_EGSSerialIsSurname(t *testing.T) {
	cert, err := ParseCertificatePEM(readTestdata(t, "cert.pem"))
	if err != nil {
		t.Fatal(err)
	}
	surnameOID, _ := asn1.Marshal(asn1.ObjectIdentifier{2, 5, 4, 4})
	idx := strings.Index(string(cert.Raw), string(surnameOID))
	if idx < 0 {
		t.Fatal("no surname attribute in the ZATCA-issued certificate")
	}
	// The EGS serial value follows the OID within the same ATV.
	window := string(cert.Raw[idx : idx+80])
	if !strings.Contains(window, "1-TST|2-TST|3-") {
		t.Errorf("surname attribute does not carry the EGS serial; window = %q", window)
	}
}

// Our emitted SignedInfo must carry the SDK template's exact algorithms and
// XPath transform strings — the Wave-2 ❓ flag, now pinned against the
// template ZATCA's own signer uses (testdata/sdk203/ubl.xml).
func TestSignedInfo_MatchesSDKTemplate(t *testing.T) {
	template := string(readTestdata(t, "ubl.xml"))
	ours := renderSignedInfo("HASH", "PROPS")

	// Extract every Algorithm= and ds:XPath value from the SDK template.
	algRe := regexp.MustCompile(`Algorithm="([^"]+)"`)
	xpathRe := regexp.MustCompile(`<ds:XPath>([^<]+)</ds:XPath>`)

	algs := map[string]bool{}
	for _, m := range algRe.FindAllStringSubmatch(template, -1) {
		algs[m[1]] = true
	}
	if len(algs) == 0 {
		t.Fatal("no algorithms found in SDK template — testdata broken?")
	}
	for alg := range algs {
		if !strings.Contains(ours, alg) {
			t.Errorf("our SignedInfo missing SDK algorithm %q", alg)
		}
	}

	xpaths := xpathRe.FindAllStringSubmatch(template, -1)
	if len(xpaths) != 3 {
		t.Fatalf("SDK template has %d XPath transforms, want 3", len(xpaths))
	}
	for _, m := range xpaths {
		if !strings.Contains(ours, ">"+m[1]+"<") {
			t.Errorf("our SignedInfo missing SDK XPath transform %q", m[1])
		}
	}
	// And their ORDER must match: UBLExtensions, Signature, QR-reference.
	last := -1
	for _, m := range xpaths {
		pos := strings.Index(ours, ">"+m[1]+"<")
		if pos <= last {
			t.Errorf("XPath transform %q out of SDK order", m[1])
		}
		last = pos
	}

	if !strings.Contains(ours, `Id="invoiceSignedData" URI=""`) {
		t.Error("missing invoiceSignedData reference")
	}
	if !strings.Contains(ours, `URI="#xadesSignedProperties"`) {
		t.Error("missing xadesSignedProperties reference")
	}
}

// The SDK's PIH genesis file carries the same value as our GenesisPIH
// constant (base64(sha256("0"))) — pinned here from the SDK zip's
// Data/PIH/pih.txt.
func TestGenesisPIH_MatchesSDK(t *testing.T) {
	const sdkPIH = "NWZlY2ViNjZmZmM4NmYzOGQ5NTI3ODZjNmQ2OTZjNzljMmRiYzIzOWRkNGU5MWI0NjcyOWQ3M2EyN2ZiNTdlOQ=="
	if GenesisPIH != sdkPIH {
		t.Fatalf("GenesisPIH = %q, SDK ships %q", GenesisPIH, sdkPIH)
	}
}
