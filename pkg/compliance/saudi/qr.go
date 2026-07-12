package saudi

import (
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// ZATCA QR code TLV tags (Fatoora Phase 1 tags 1-5; Phase 2 simplified
// invoices add the cryptographic-stamp tags 6-9).
const (
	QRTagSellerName    = 1 // UTF-8 seller name
	QRTagVATNumber     = 2 // UTF-8 15-digit VAT registration number
	QRTagTimestamp     = 3 // UTF-8 ISO-8601 UTC timestamp
	QRTagTotalWithVAT  = 4 // UTF-8 decimal string
	QRTagVATTotal      = 5 // UTF-8 decimal string
	QRTagInvoiceHash   = 6 // UTF-8 string of the base64 invoice hash
	QRTagSignature     = 7 // raw ECDSA signature bytes
	QRTagPublicKey     = 8 // raw DER SubjectPublicKeyInfo of the EGS key
	QRTagCertSignature = 9 // raw signature bytes from the CSID certificate
)

// QRData carries the fields encoded into a ZATCA QR code.
type QRData struct {
	SellerName   string
	VATNumber    string
	Timestamp    time.Time
	TotalWithVAT string // formatted decimal, e.g. "264.50"
	VATTotal     string // formatted decimal, e.g. "34.50"

	// Phase 2 cryptographic stamp (simplified invoices). Leave empty for a
	// Phase-1-style QR (tags 1-5 only).
	InvoiceHashB64 string // the base64 invoice hash, encoded as its UTF-8 string
	Signature      []byte // raw ECDSA signature bytes
	PublicKey      []byte // DER SubjectPublicKeyInfo
	CertSignature  []byte // the CSID certificate's signature bytes
}

// EncodeQR renders the TLV byte stream and returns its base64 — the string a
// QR code renderer should encode. Each field is [1-byte tag][1-byte
// length][value]; tags 1-6 are UTF-8 text, tags 7-9 raw binary.
func EncodeQR(d QRData) (string, error) {
	if d.SellerName == "" || d.VATNumber == "" || d.Timestamp.IsZero() ||
		d.TotalWithVAT == "" || d.VATTotal == "" {
		return "", errors.New("zatca qr: tags 1-5 (seller, VAT number, timestamp, totals) are all required")
	}

	var buf []byte
	appendTLV := func(tag byte, value []byte) error {
		if len(value) > 255 {
			return fmt.Errorf("zatca qr: tag %d value exceeds 255 bytes (%d)", tag, len(value))
		}
		buf = append(buf, tag, byte(len(value)))
		buf = append(buf, value...)
		return nil
	}

	// ZATCA QR timestamps are ISO-8601 Zulu: YYYY-MM-DDTHH:MM:SSZ.
	fields := []struct {
		tag   byte
		value []byte
	}{
		{QRTagSellerName, []byte(d.SellerName)},
		{QRTagVATNumber, []byte(d.VATNumber)},
		{QRTagTimestamp, []byte(d.Timestamp.UTC().Format("2006-01-02T15:04:05Z"))},
		{QRTagTotalWithVAT, []byte(d.TotalWithVAT)},
		{QRTagVATTotal, []byte(d.VATTotal)},
	}
	for _, f := range fields {
		if err := appendTLV(f.tag, f.value); err != nil {
			return "", err
		}
	}

	if d.InvoiceHashB64 != "" {
		if err := appendTLV(QRTagInvoiceHash, []byte(d.InvoiceHashB64)); err != nil {
			return "", err
		}
	}
	if len(d.Signature) > 0 {
		if err := appendTLV(QRTagSignature, d.Signature); err != nil {
			return "", err
		}
	}
	if len(d.PublicKey) > 0 {
		if err := appendTLV(QRTagPublicKey, d.PublicKey); err != nil {
			return "", err
		}
	}
	if len(d.CertSignature) > 0 {
		if err := appendTLV(QRTagCertSignature, d.CertSignature); err != nil {
			return "", err
		}
	}

	return base64.StdEncoding.EncodeToString(buf), nil
}

// DecodeQR parses a base64 TLV QR payload back into its tagged fields.
// Used for verification and tests; unknown tags are preserved.
func DecodeQR(encoded string) (map[byte][]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("zatca qr: not base64: %w", err)
	}
	out := make(map[byte][]byte)
	for i := 0; i < len(raw); {
		if i+2 > len(raw) {
			return nil, errors.New("zatca qr: truncated TLV header")
		}
		tag, length := raw[i], int(raw[i+1])
		i += 2
		if i+length > len(raw) {
			return nil, fmt.Errorf("zatca qr: tag %d declares %d bytes but only %d remain", tag, length, len(raw)-i)
		}
		out[tag] = raw[i : i+length]
		i += length
	}
	return out, nil
}
