package saudi

// ZATCA-profile CSR generation (Wave 3 C.2) — the offline half of Fatoora
// onboarding. The generated PKCS#10 request is what you POST to the
// compliance endpoint (with the portal OTP) to receive a CCSID.
//
// Hand-assembled ASN.1 throughout: crypto/x509.CreateCertificateRequest
// rejects secp256k1 outright, the same stdlib hostility NewSelfSignedCertificate
// works around. Profile cross-checked against two independent SDK-aligned
// implementations (Saleh7/php-zatca-xml CertificateBuilder, wes4m/zatca-xml-js
// csr_template) — both agree:
//
//   - subject: CN, OU, O, C (=SA)
//   - extensionRequest attribute carrying exactly two extensions:
//     1. certificateTemplateName (MS OID 1.3.6.1.4.1.311.20.2) as UTF8String —
//        "ZATCA-Code-Signing" (production) / "PREZATCA-Code-Signing"
//        (simulation) / "TSTZATCA-Code-Signing" (sandbox)
//     2. subjectAltName = a directoryName with SN, UID, title,
//        registeredAddress, businessCategory
//   - NO keyUsage extension (both references omit it)
//
// Encoding note: the reference implementations feed the SAN through OpenSSL
// config where the key "SN" is OpenSSL's short name for SURNAME (2.5.4.4),
// not serialNumber (2.5.4.5) — so the EGS serial travels as surname, and we
// match that byte-for-byte rather than "correcting" it.

import (
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CSRConfig describes one EGS unit for onboarding.
type CSRConfig struct {
	// CommonName uniquely names the EGS unit (e.g. "TST-886431145-399999999900003").
	CommonName string
	// Organization is the registered taxpayer name.
	Organization string
	// OrganizationalUnit is the branch/unit name (10-digit TIN for VAT groups).
	OrganizationalUnit string
	// VATNumber is the 15-digit VAT registration number (starts AND ends with 3).
	VATNumber string
	// SolutionName, ModelVersion and DeviceSerial compose the EGS serial
	// number SAN attribute: "1-<solution>|2-<model>|3-<device>".
	SolutionName string
	ModelVersion string
	DeviceSerial string
	// InvoiceTypeFlags is the 4-digit TSCZ document-type support string for
	// the SAN title attribute (e.g. "1100" = standard + simplified, "0100" =
	// simplified only).
	InvoiceTypeFlags string
	// RegisteredAddress is the branch location; BusinessCategory the industry.
	RegisteredAddress string
	BusinessCategory  string
	// Environment selects the certificate template name. Uses the same
	// constants as the API client (EnvSandbox/EnvSimulation/EnvProduction).
	Environment Environment
}

// EGSSerialNumber renders the SAN serial attribute.
func (c CSRConfig) EGSSerialNumber() string {
	return fmt.Sprintf("1-%s|2-%s|3-%s", c.SolutionName, c.ModelVersion, c.DeviceSerial)
}

// templateName maps the environment to ZATCA's certificate template value.
func (c CSRConfig) templateName() (string, error) {
	switch c.Environment {
	case EnvProduction:
		return "ZATCA-Code-Signing", nil
	case EnvSimulation:
		return "PREZATCA-Code-Signing", nil
	case EnvSandbox, "":
		return "TSTZATCA-Code-Signing", nil
	default:
		return "", fmt.Errorf("zatca: unknown environment %q", c.Environment)
	}
}

var vatNumberRe = regexp.MustCompile(`^3\d{13}3$`)
var invoiceFlagsRe = regexp.MustCompile(`^[01]{4}$`)

func (c CSRConfig) validate() error {
	switch {
	case strings.TrimSpace(c.CommonName) == "":
		return errors.New("zatca: CSR requires a common name")
	case strings.TrimSpace(c.Organization) == "":
		return errors.New("zatca: CSR requires an organization name")
	case strings.TrimSpace(c.OrganizationalUnit) == "":
		return errors.New("zatca: CSR requires an organizational unit")
	case !vatNumberRe.MatchString(c.VATNumber):
		return fmt.Errorf("zatca: VAT number %q must be 15 digits starting and ending with 3", c.VATNumber)
	case strings.TrimSpace(c.SolutionName) == "" || strings.TrimSpace(c.ModelVersion) == "" || strings.TrimSpace(c.DeviceSerial) == "":
		return errors.New("zatca: CSR requires solution name, model version and device serial (EGS serial number)")
	case !invoiceFlagsRe.MatchString(c.InvoiceTypeFlags):
		return fmt.Errorf("zatca: invoice type flags %q must be 4 binary digits (TSCZ, e.g. 1100)", c.InvoiceTypeFlags)
	case strings.TrimSpace(c.RegisteredAddress) == "":
		return errors.New("zatca: CSR requires a registered address")
	case strings.TrimSpace(c.BusinessCategory) == "":
		return errors.New("zatca: CSR requires a business category")
	}
	return nil
}

// Attribute OIDs used by the ZATCA CSR profile.
var (
	oidExtensionRequest        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 14}
	oidCertificateTemplateName = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 311, 20, 2}
	oidSubjectAltName          = asn1.ObjectIdentifier{2, 5, 29, 17}
	oidOrganization            = asn1.ObjectIdentifier{2, 5, 4, 10}
	oidOrganizationalUnit      = asn1.ObjectIdentifier{2, 5, 4, 11}
	oidSurname                 = asn1.ObjectIdentifier{2, 5, 4, 4} // OpenSSL "SN"
	oidUserID                  = asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1}
	oidTitle                   = asn1.ObjectIdentifier{2, 5, 4, 12}
	oidRegisteredAddress       = asn1.ObjectIdentifier{2, 5, 4, 26}
	oidBusinessCategory        = asn1.ObjectIdentifier{2, 5, 4, 15}
	oidCommonNameCSR           = asn1.ObjectIdentifier{2, 5, 4, 3}
	oidCountryCSR              = asn1.ObjectIdentifier{2, 5, 4, 6}
	oidECDSAWithSHA256CSR      = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
)

type csrATV struct {
	Type  asn1.ObjectIdentifier
	Value string `asn1:"utf8"`
}

// rdnSequenceDER encodes a Name as RDNSequence with one AttributeTypeAndValue
// per RDN, in the given order. Every value is a UTF8String (real names carry
// Arabic/accents; PrintableString rejects them).
func rdnSequenceDER(entries []csrATV) ([]byte, error) {
	var seq []asn1.RawValue
	for _, e := range entries {
		setBytes, err := asn1.Marshal([]csrATV{e})
		if err != nil {
			return nil, err
		}
		setBytes[0] = 0x31 // re-tag SEQUENCE OF → SET OF
		seq = append(seq, asn1.RawValue{FullBytes: setBytes})
	}
	return asn1.Marshal(seq)
}

// GenerateCSR builds and signs a ZATCA-profile PKCS#10 certificate request
// for the key. It returns the DER bytes and the PEM ("CERTIFICATE REQUEST")
// rendering; the Fatoora compliance endpoint accepts base64 of the PEM.
func GenerateCSR(key *KeyPair, cfg CSRConfig) (der []byte, pemStr string, err error) {
	if key == nil {
		return nil, "", errors.New("zatca: nil key")
	}
	if err := cfg.validate(); err != nil {
		return nil, "", err
	}
	template, err := cfg.templateName()
	if err != nil {
		return nil, "", err
	}

	// --- Subject (CN, OU, O, C — the reference template order) ---
	subjectDER, err := rdnSequenceDER([]csrATV{
		{oidCommonNameCSR, cfg.CommonName},
		{oidOrganizationalUnit, cfg.OrganizationalUnit},
		{oidOrganization, cfg.Organization},
		{oidCountryCSR, "SA"},
	})
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode CSR subject: %w", err)
	}

	// --- Extension 1: certificateTemplateName (UTF8String) ---
	templateValue, err := asn1.MarshalWithParams(template, "utf8")
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode template name: %w", err)
	}

	// --- Extension 2: subjectAltName = directoryName (GeneralName [4]) ---
	sanNameDER, err := rdnSequenceDER([]csrATV{
		{oidSurname, cfg.EGSSerialNumber()},
		{oidUserID, cfg.VATNumber},
		{oidTitle, cfg.InvoiceTypeFlags},
		{oidRegisteredAddress, cfg.RegisteredAddress},
		{oidBusinessCategory, cfg.BusinessCategory},
	})
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode SAN directory name: %w", err)
	}
	// GeneralName directoryName is [4] EXPLICIT Name.
	directoryName := asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 4, IsCompound: true, Bytes: sanNameDER}
	sanValue, err := asn1.Marshal([]asn1.RawValue{directoryName}) // GeneralNames ::= SEQUENCE OF GeneralName
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode SAN: %w", err)
	}

	type extension struct {
		ID    asn1.ObjectIdentifier
		Value []byte // OCTET STRING wrapping the inner DER
	}
	extensionsDER, err := asn1.Marshal([]extension{
		{ID: oidCertificateTemplateName, Value: templateValue},
		{ID: oidSubjectAltName, Value: sanValue},
	})
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode extensions: %w", err)
	}

	// --- extensionRequest attribute: SEQUENCE { OID, SET { Extensions } } ---
	extSet, err := asn1.Marshal([]asn1.RawValue{{FullBytes: extensionsDER}})
	if err != nil {
		return nil, "", err
	}
	extSet[0] = 0x31 // SEQUENCE OF → SET OF
	attrDER, err := asn1.Marshal(struct {
		Type   asn1.ObjectIdentifier
		Values asn1.RawValue
	}{oidExtensionRequest, asn1.RawValue{FullBytes: extSet}})
	if err != nil {
		return nil, "", err
	}

	// --- CertificationRequestInfo ---
	spkiDER, err := key.PublicKeyDER()
	if err != nil {
		return nil, "", err
	}
	// attributes is [0] IMPLICIT SET OF Attribute: re-tag the marshalled
	// SEQUENCE OF by re-parsing for its contents (never slice headers by
	// hand — DER lengths go multi-byte past 127 bytes of content).
	attrsSeq, err := asn1.Marshal([]asn1.RawValue{{FullBytes: attrDER}})
	if err != nil {
		return nil, "", err
	}
	var attrsRV asn1.RawValue
	if _, err := asn1.Unmarshal(attrsSeq, &attrsRV); err != nil {
		return nil, "", err
	}
	criDER, err := asn1.Marshal(struct {
		Version    int
		Subject    asn1.RawValue
		SPKI       asn1.RawValue
		Attributes asn1.RawValue
	}{
		Version:    0,
		Subject:    asn1.RawValue{FullBytes: subjectDER},
		SPKI:       asn1.RawValue{FullBytes: spkiDER},
		Attributes: asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: attrsRV.Bytes},
	})
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode CertificationRequestInfo: %w", err)
	}

	// --- Sign and assemble CertificationRequest ---
	sigBytes := key.SignBytes(criDER)
	der, err = asn1.Marshal(struct {
		CRI       asn1.RawValue
		Algorithm struct{ Algorithm asn1.ObjectIdentifier }
		Signature asn1.BitString
	}{
		CRI:       asn1.RawValue{FullBytes: criDER},
		Algorithm: struct{ Algorithm asn1.ObjectIdentifier }{oidECDSAWithSHA256CSR},
		Signature: asn1.BitString{Bytes: sigBytes, BitLength: len(sigBytes) * 8},
	})
	if err != nil {
		return nil, "", fmt.Errorf("zatca: cannot encode CSR: %w", err)
	}

	pemStr = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der}))
	return der, pemStr, nil
}

// CSRBase64 renders the CSR the way the Fatoora compliance endpoint wants it
// in the JSON body: base64 of the full PEM text.
func CSRBase64(pemStr string) string {
	return base64.StdEncoding.EncodeToString([]byte(pemStr))
}
