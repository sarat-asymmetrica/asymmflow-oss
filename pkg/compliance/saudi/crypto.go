package saudi

import (
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	secpecdsa "github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// GenesisPIH is the Previous Invoice Hash for the FIRST invoice in an EGS
// unit's chain: base64 of the SHA-256 of "0", per ZATCA's implementation
// standard.
const GenesisPIH = "NWZlY2ViNjZmZmM4NmYzOGQ5NTI3ODZjNmQ2OTZjNzljMmRiYzIzOWRkNGU5MWI0NjcyOWQ3M2EyN2ZiNTdlOQ=="

// InvoiceHashB64 computes the ZATCA invoice hash: base64(SHA-256(xml)).
//
// Canonicalization boundary (documented, deliberate): ZATCA specifies
// C14N 1.1 of the invoice with UBLExtensions, cac:Signature and the QR
// AdditionalDocumentReference removed. This package GENERATES the invoice
// XML itself in already-canonical form (deterministic ordering, no
// self-closing tags, canonical attribute quoting) and hashes the exact
// pre-signature bytes it emitted — so canonicalizing is the identity
// transform on our own output. Hashing arbitrary third-party XML would
// require a full C14N 1.1 implementation, which is out of scope here.
func InvoiceHashB64(preSignatureXML []byte) string {
	sum := sha256.Sum256(preSignatureXML)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// KeyPair is an EGS unit's ECDSA secp256k1 signing key.
// secp256k1 (not P-256) is mandated by ZATCA's security features standard —
// the most common integration mistake is using the wrong curve.
type KeyPair struct {
	priv *secp256k1.PrivateKey
}

// GenerateKeyPair creates a fresh secp256k1 key (for onboarding/CSR flows).
func GenerateKeyPair() (*KeyPair, error) {
	priv, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, fmt.Errorf("zatca: key generation failed: %w", err)
	}
	return &KeyPair{priv: priv}, nil
}

// ParsePrivateKeyPEM loads a secp256k1 private key from PEM. Supports the
// OpenSSL "EC PRIVATE KEY" (SEC 1) block produced by
// `openssl ecparam -name secp256k1 -genkey`, and — Wave 3 fix, exposed by
// the ZATCA SDK's own reference materials — the header-less base64 DER form
// the SDK ships (Data/Certificates/ec-secp256k1-priv-key.pem has no PEM
// armor at all).
func ParsePrivateKeyPEM(pemBytes []byte) (*KeyPair, error) {
	var keyDER []byte
	if block, _ := pem.Decode(pemBytes); block != nil {
		if block.Type != "EC PRIVATE KEY" {
			return nil, fmt.Errorf("zatca: unsupported PEM block %q (want EC PRIVATE KEY)", block.Type)
		}
		keyDER = block.Bytes
	} else {
		der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(pemBytes)), ""))
		if err != nil {
			return nil, errors.New("zatca: input is neither PEM nor bare base64 DER")
		}
		keyDER = der
	}
	// SEC 1 ECPrivateKey ::= SEQUENCE { version, privateKey OCTET STRING, ... }
	var sec1 struct {
		Version    int
		PrivateKey []byte
		Rest       asn1.RawValue `asn1:"optional"`
		Rest2      asn1.RawValue `asn1:"optional"`
	}
	if _, err := asn1.Unmarshal(keyDER, &sec1); err != nil {
		return nil, fmt.Errorf("zatca: cannot parse EC private key: %w", err)
	}
	if len(sec1.PrivateKey) == 0 || len(sec1.PrivateKey) > 32 {
		return nil, errors.New("zatca: EC private key has unexpected length")
	}
	var scalar [32]byte
	copy(scalar[32-len(sec1.PrivateKey):], sec1.PrivateKey)
	priv := secp256k1.PrivKeyFromBytes(scalar[:])
	return &KeyPair{priv: priv}, nil
}

// SignBytes signs SHA-256(message) with ECDSA secp256k1 and returns the
// DER-encoded signature (the encoding ZATCA's reference implementations
// produce via OpenSSL/BouncyCastle).
func (k *KeyPair) SignBytes(message []byte) []byte {
	digest := sha256.Sum256(message)
	sig := secpecdsa.Sign(k.priv, digest[:])
	return sig.Serialize()
}

// SignBase64 signs and returns base64(DER signature) — the form embedded in
// ds:SignatureValue and QR tag 7.
func (k *KeyPair) SignBase64(message []byte) string {
	return base64.StdEncoding.EncodeToString(k.SignBytes(message))
}

// Verify checks a DER signature over SHA-256(message) with this key's public
// half. Used by tests and QR verification.
func (k *KeyPair) Verify(message, derSignature []byte) bool {
	sig, err := secpecdsa.ParseDERSignature(derSignature)
	if err != nil {
		return false
	}
	digest := sha256.Sum256(message)
	return sig.Verify(digest[:], k.priv.PubKey())
}

// asn1 object identifiers for the SPKI encoding.
var (
	oidECPublicKey = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}
	oidSecp256k1   = asn1.ObjectIdentifier{1, 3, 132, 0, 10}
)

// PublicKeyDER returns the DER SubjectPublicKeyInfo for the key — QR tag 8.
// Go's crypto/x509 cannot marshal secp256k1 keys (unsupported curve), so the
// SPKI structure is assembled explicitly:
//
//	SubjectPublicKeyInfo ::= SEQUENCE {
//	  algorithm  SEQUENCE { id-ecPublicKey, secp256k1 },
//	  subjectPublicKey BIT STRING (uncompressed EC point) }
func (k *KeyPair) PublicKeyDER() ([]byte, error) {
	type algorithmIdentifier struct {
		Algorithm asn1.ObjectIdentifier
		Curve     asn1.ObjectIdentifier
	}
	type subjectPublicKeyInfo struct {
		Algorithm algorithmIdentifier
		PublicKey asn1.BitString
	}
	point := k.priv.PubKey().SerializeUncompressed()
	der, err := asn1.Marshal(subjectPublicKeyInfo{
		Algorithm: algorithmIdentifier{Algorithm: oidECPublicKey, Curve: oidSecp256k1},
		PublicKey: asn1.BitString{Bytes: point, BitLength: len(point) * 8},
	})
	if err != nil {
		return nil, fmt.Errorf("zatca: SPKI encoding failed: %w", err)
	}
	return der, nil
}

// Certificate is a minimally-parsed X.509 CSID certificate. Go's stdlib
// x509.ParseCertificate rejects secp256k1 SPKIs outright, so the fields
// ZATCA signing needs are extracted with a manual ASN.1 walk instead.
type Certificate struct {
	Raw            []byte   // full DER, for the XAdES cert digest
	SerialNumber   *big.Int // XAdES IssuerSerial
	IssuerName     string   // RFC 2253-style rendering for XAdES IssuerName
	SignatureValue []byte   // the CA's signature bytes — QR tag 9
}

// ParseCertificatePEM parses a CSID certificate from PEM, or from the
// header-less base64-DER form ZATCA actually uses (the gateway's
// binarySecurityToken and the SDK's Data/Certificates/cert.pem both carry
// bare base64 with no PEM armor). Wave 3 fix: the previous version claimed
// to handle the token but required PEM headers — real onboarding material
// would have failed at the door; the SDK's reference cert exposed it.
func ParseCertificatePEM(pemBytes []byte) (*Certificate, error) {
	if block, _ := pem.Decode(pemBytes); block != nil {
		return ParseCertificateDER(block.Bytes)
	}
	der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(pemBytes)), ""))
	if err != nil {
		return nil, errors.New("zatca: input is neither certificate PEM nor bare base64 DER")
	}
	// The token may be base64 of a PEM (double-wrapped) or base64 of DER.
	if block, _ := pem.Decode(der); block != nil {
		return ParseCertificateDER(block.Bytes)
	}
	return ParseCertificateDER(der)
}

// ParseCertificateDER extracts the ZATCA-relevant fields from a DER
// certificate.
func ParseCertificateDER(der []byte) (*Certificate, error) {
	// Certificate ::= SEQUENCE { tbsCertificate, signatureAlgorithm, signature BIT STRING }
	var outer struct {
		TBS       asn1.RawValue
		Algorithm asn1.RawValue
		Signature asn1.BitString
	}
	if rest, err := asn1.Unmarshal(der, &outer); err != nil {
		return nil, fmt.Errorf("zatca: cannot parse certificate: %w", err)
	} else if len(rest) != 0 {
		return nil, errors.New("zatca: trailing bytes after certificate")
	}

	// TBSCertificate ::= SEQUENCE { [0] version OPTIONAL, serialNumber,
	//   signature, issuer, validity, subject, subjectPublicKeyInfo, ... }
	// Walked element-by-element: encoding/asn1's struct tags interact badly
	// with RawValue fields for the context-specific version element.
	elements := make([]asn1.RawValue, 0, 8)
	rest := outer.TBS.Bytes
	for len(rest) > 0 && len(elements) < 8 {
		var el asn1.RawValue
		var err error
		rest, err = asn1.Unmarshal(rest, &el)
		if err != nil {
			return nil, fmt.Errorf("zatca: cannot parse TBS certificate: %w", err)
		}
		elements = append(elements, el)
	}
	idx := 0
	if len(elements) > 0 && elements[0].Class == asn1.ClassContextSpecific && elements[0].Tag == 0 {
		idx = 1 // explicit [0] version present
	}
	if len(elements) < idx+4 {
		return nil, errors.New("zatca: TBS certificate too short")
	}

	serial := new(big.Int)
	if _, err := asn1.Unmarshal(elements[idx].FullBytes, &serial); err != nil {
		return nil, fmt.Errorf("zatca: cannot parse certificate serial: %w", err)
	}
	// elements[idx+1] = signature algorithm, elements[idx+2] = issuer.
	issuer, err := renderRDNSequence(elements[idx+2].FullBytes)
	if err != nil {
		return nil, fmt.Errorf("zatca: cannot render issuer: %w", err)
	}

	return &Certificate{
		Raw:            der,
		SerialNumber:   serial,
		IssuerName:     issuer,
		SignatureValue: outer.Signature.Bytes,
	}, nil
}

// NewSelfSignedCertificate builds a minimal self-signed secp256k1 X.509
// certificate for OFFLINE use: demo verticals, local development, and the
// pre-onboarding phase before ZATCA issues a real CSID. It is NOT accepted by
// the ZATCA gateway — production signing requires the CSID certificate from
// the compliance/production onboarding flow (see api.go).
//
// Assembled manually because crypto/x509.CreateCertificate rejects secp256k1.
func NewSelfSignedCertificate(key *KeyPair, commonName, countryCode string, notBefore, notAfter time.Time) (*Certificate, error) {
	if key == nil {
		return nil, errors.New("zatca: nil key")
	}
	type algorithmIdentifier struct {
		Algorithm asn1.ObjectIdentifier
	}
	type validity struct {
		NotBefore, NotAfter time.Time
	}
	oidECDSAWithSHA256 := asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	oidCommonName := asn1.ObjectIdentifier{2, 5, 4, 3}
	oidCountry := asn1.ObjectIdentifier{2, 5, 4, 6}

	type atv struct {
		// UTF8String, not PrintableString: real names carry accents/Arabic
		// (e.g. "Wasela Café LLC"), which PrintableString rejects.
		Type  asn1.ObjectIdentifier
		Value string `asn1:"utf8"`
	}
	rdnDER := func(entries ...atv) (asn1.RawValue, error) {
		var seq []asn1.RawValue
		for _, e := range entries {
			setBytes, err := asn1.Marshal([]atv{e})
			if err != nil {
				return asn1.RawValue{}, err
			}
			// asn1.Marshal of a slice yields SEQUENCE OF; re-tag as SET.
			setBytes[0] = 0x31
			seq = append(seq, asn1.RawValue{FullBytes: setBytes})
		}
		full, err := asn1.Marshal(seq)
		if err != nil {
			return asn1.RawValue{}, err
		}
		return asn1.RawValue{FullBytes: full}, nil
	}

	name, err := rdnDER(atv{oidCountry, countryCode}, atv{oidCommonName, commonName})
	if err != nil {
		return nil, fmt.Errorf("zatca: cannot encode certificate name: %w", err)
	}
	spkiDER, err := key.PublicKeyDER()
	if err != nil {
		return nil, err
	}

	versionInner, _ := asn1.Marshal(2) // v3
	// Serial derived from the public key so the cert is deterministic per key.
	sum := sha256.Sum256(spkiDER)
	serial := new(big.Int).SetBytes(sum[:8])
	serial.Abs(serial)

	tbs := struct {
		Version      asn1.RawValue
		SerialNumber *big.Int
		Signature    algorithmIdentifier
		Issuer       asn1.RawValue
		Validity     validity
		Subject      asn1.RawValue
		SPKI         asn1.RawValue
	}{
		Version:      asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: versionInner},
		SerialNumber: serial,
		Signature:    algorithmIdentifier{Algorithm: oidECDSAWithSHA256},
		Issuer:       name,
		Validity:     validity{NotBefore: notBefore.UTC(), NotAfter: notAfter.UTC()},
		Subject:      name,
		SPKI:         asn1.RawValue{FullBytes: spkiDER},
	}
	tbsDER, err := asn1.Marshal(tbs)
	if err != nil {
		return nil, fmt.Errorf("zatca: cannot encode TBS certificate: %w", err)
	}

	sigBytes := key.SignBytes(tbsDER)
	certDER, err := asn1.Marshal(struct {
		TBS       asn1.RawValue
		Algorithm algorithmIdentifier
		Signature asn1.BitString
	}{
		TBS:       asn1.RawValue{FullBytes: tbsDER},
		Algorithm: algorithmIdentifier{Algorithm: oidECDSAWithSHA256},
		Signature: asn1.BitString{Bytes: sigBytes, BitLength: len(sigBytes) * 8},
	})
	if err != nil {
		return nil, fmt.Errorf("zatca: cannot encode certificate: %w", err)
	}
	return ParseCertificateDER(certDER)
}

// DigestB64 returns base64(SHA-256(raw DER)) — the XAdES CertDigest value.
func (c *Certificate) DigestB64() string {
	sum := sha256.Sum256(c.Raw)
	return base64.StdEncoding.EncodeToString(sum[:])
}

// renderRDNSequence renders an X.501 Name as "CN=..., O=..., C=..."
// (most-specific first, the ordering ZATCA reference implementations emit).
func renderRDNSequence(fullBytes []byte) (string, error) {
	var rdns []asn1.RawValue
	var seq asn1.RawValue
	if _, err := asn1.Unmarshal(fullBytes, &seq); err != nil {
		return "", err
	}
	rest := seq.Bytes
	for len(rest) > 0 {
		var rdn asn1.RawValue
		var err error
		rest, err = asn1.Unmarshal(rest, &rdn)
		if err != nil {
			return "", err
		}
		rdns = append(rdns, rdn)
	}

	shortNames := map[string]string{
		"2.5.4.3":  "CN",
		"2.5.4.6":  "C",
		"2.5.4.7":  "L",
		"2.5.4.8":  "ST",
		"2.5.4.10": "O",
		"2.5.4.11": "OU",
	}

	parts := make([]string, 0, len(rdns))
	// RFC 2253 renders the sequence in reverse (most-specific first).
	for i := len(rdns) - 1; i >= 0; i-- {
		var atv struct {
			Type  asn1.ObjectIdentifier
			Value asn1.RawValue
		}
		inner := rdns[i].Bytes // SET OF → first AttributeTypeAndValue
		if _, err := asn1.Unmarshal(inner, &atv); err != nil {
			return "", err
		}
		name, ok := shortNames[atv.Type.String()]
		if !ok {
			name = atv.Type.String()
		}
		var value string
		if _, err := asn1.Unmarshal(atv.Value.FullBytes, &value); err != nil {
			// Fall back to raw bytes for unusual string types.
			value = string(atv.Value.Bytes)
		}
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, ", "), nil
}
