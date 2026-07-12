package saudi

// ZATCA Fatoora API client: onboarding (compliance CSID → production CSID),
// simplified-invoice reporting, and standard-invoice clearance.
//
// Boundary (deliberate, documented): the crypto, XML and QR generation in
// this package are real; THIS client is a faithful HTTP implementation of
// the gateway contract that has NOT been exercised against ZATCA's live
// gateway from this codebase. Point it at EnvSandbox (developer portal)
// with real CSIDs to onboard, or inject any *http.Client (e.g. one backed
// by httptest) for offline testing. Response-code semantics follow the raw
// gateway: 200 accepted, 202 accepted WITH warnings (never resubmit a 202),
// 400 rejected, 401 auth, 413 too large.

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultBaseURL is the current ZATCA gateway host. The legacy
// gw-apic-gov.gazt.gov.sa host was decommissioned 2025-09-14 — do not use it.
const DefaultBaseURL = "https://gw-fatoora.zatca.gov.sa"

// Environment selects the gateway path segment.
type Environment string

const (
	EnvSandbox    Environment = "developer-portal" // TSTZATCA certs
	EnvSimulation Environment = "simulation"       // PREZATCA certs
	EnvProduction Environment = "core"             // ZATCA certs
)

// Credentials is a CSID basic-auth pair: username = binarySecurityToken,
// password = secret (as returned by the CSID endpoints).
type Credentials struct {
	Token  string
	Secret string
}

// Client talks to the Fatoora gateway.
type Client struct {
	BaseURL     string // defaults to DefaultBaseURL
	Environment Environment
	Credentials Credentials
	HTTPClient  *http.Client // defaults to a 30s-timeout client
}

// NewClient builds a client for an environment.
func NewClient(env Environment, creds Credentials) *Client {
	return &Client{
		BaseURL:     DefaultBaseURL,
		Environment: env,
		Credentials: creds,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// InvoiceSubmission is the payload for reporting/clearance/compliance checks.
type InvoiceSubmission struct {
	InvoiceHash string // base64 SHA-256 (SignedInvoice.InvoiceHashB64)
	UUID        string // the invoice cbc:UUID
	InvoiceXML  []byte // the FINAL signed XML (SignedInvoice.XML)
}

// ValidationMessage is one entry in the gateway's validationResults lists.
type ValidationMessage struct {
	Type     string `json:"type"`
	Code     string `json:"code"`
	Category string `json:"category"`
	Message  string `json:"message"`
	Status   string `json:"status"`
}

// ValidationResults mirrors the gateway's validation envelope.
type ValidationResults struct {
	Status          string              `json:"status"`
	InfoMessages    []ValidationMessage `json:"infoMessages"`
	WarningMessages []ValidationMessage `json:"warningMessages"`
	ErrorMessages   []ValidationMessage `json:"errorMessages"`
}

// SubmissionResult is the interpreted gateway response.
type SubmissionResult struct {
	HTTPStatus      int
	Accepted        bool // 200 or 202
	WithWarnings    bool // 202 — investigate warnings, do NOT resubmit
	ReportingStatus string
	ClearanceStatus string
	Validation      ValidationResults
	// ClearedInvoiceXML is ZATCA's stamped XML (clearance only). PERSIST AND
	// SHARE THIS, not the locally-built XML — it carries ZATCA's QR/stamp.
	ClearedInvoiceXML []byte
}

// CSIDResponse is the onboarding response for compliance/production CSIDs.
type CSIDResponse struct {
	RequestID           json.Number `json:"requestID"`
	BinarySecurityToken string      `json:"binarySecurityToken"`
	Secret              string      `json:"secret"`
	DispositionMessage  string      `json:"dispositionMessage"`
}

// Credentials converts a CSID response into basic-auth credentials.
func (r *CSIDResponse) AsCredentials() Credentials {
	return Credentials{Token: r.BinarySecurityToken, Secret: r.Secret}
}

// RequestComplianceCSID exchanges a base64 CSR + portal OTP for a compliance
// CSID (onboarding step 1). OTPs come from the Fatoora portal, valid 1 hour.
func (c *Client) RequestComplianceCSID(ctx context.Context, csrB64, otp string) (*CSIDResponse, error) {
	body, _ := json.Marshal(map[string]string{"csr": csrB64})
	req, err := c.newRequest(ctx, http.MethodPost, "compliance", body, false)
	if err != nil {
		return nil, err
	}
	req.Header.Set("OTP", otp)
	return c.doCSID(req)
}

// RequestProductionCSID exchanges a passed compliance check for a production
// CSID (onboarding step 3). Authenticate with the COMPLIANCE credentials.
func (c *Client) RequestProductionCSID(ctx context.Context, complianceRequestID string) (*CSIDResponse, error) {
	body, _ := json.Marshal(map[string]string{"compliance_request_id": complianceRequestID})
	req, err := c.newRequest(ctx, http.MethodPost, "production/csids", body, true)
	if err != nil {
		return nil, err
	}
	return c.doCSID(req)
}

// CheckCompliance submits a sample invoice to the compliance checker
// (onboarding step 2; which document types are required depends on the CSR's
// `title` flags).
func (c *Client) CheckCompliance(ctx context.Context, sub InvoiceSubmission) (*SubmissionResult, error) {
	return c.submit(ctx, "compliance/invoices", sub, nil)
}

// ReportInvoice reports a SIMPLIFIED (B2C) invoice. ZATCA requires reporting
// within 24 hours of issuance.
func (c *Client) ReportInvoice(ctx context.Context, sub InvoiceSubmission) (*SubmissionResult, error) {
	return c.submit(ctx, "invoices/reporting/single", sub, nil)
}

// ClearInvoice clears a STANDARD (B2B) invoice. The invoice must be cleared
// BEFORE being shared with the buyer; use the returned ClearedInvoiceXML.
func (c *Client) ClearInvoice(ctx context.Context, sub InvoiceSubmission) (*SubmissionResult, error) {
	return c.submit(ctx, "invoices/clearance/single", sub, map[string]string{"Clearance-Status": "1"})
}

func (c *Client) submit(ctx context.Context, path string, sub InvoiceSubmission, extraHeaders map[string]string) (*SubmissionResult, error) {
	if sub.InvoiceHash == "" || sub.UUID == "" || len(sub.InvoiceXML) == 0 {
		return nil, errors.New("zatca api: submission requires invoiceHash, uuid and invoice XML")
	}
	body, _ := json.Marshal(map[string]string{
		"invoiceHash": sub.InvoiceHash,
		"uuid":        sub.UUID,
		"invoice":     base64.StdEncoding.EncodeToString(sub.InvoiceXML),
	})
	req, err := c.newRequest(ctx, http.MethodPost, path, body, true)
	if err != nil {
		return nil, err
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("zatca api: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("zatca api: reading response: %w", err)
	}

	var envelope struct {
		ReportingStatus   string            `json:"reportingStatus"`
		ClearanceStatus   string            `json:"clearanceStatus"`
		ClearedInvoice    string            `json:"clearedInvoice"`
		ValidationResults ValidationResults `json:"validationResults"`
	}
	_ = json.Unmarshal(raw, &envelope) // some error statuses return empty/non-JSON bodies

	result := &SubmissionResult{
		HTTPStatus:      resp.StatusCode,
		ReportingStatus: envelope.ReportingStatus,
		ClearanceStatus: envelope.ClearanceStatus,
		Validation:      envelope.ValidationResults,
	}
	if envelope.ClearedInvoice != "" {
		if decoded, err := base64.StdEncoding.DecodeString(envelope.ClearedInvoice); err == nil {
			result.ClearedInvoiceXML = decoded
		}
	}

	switch resp.StatusCode {
	case http.StatusOK:
		result.Accepted = true
		return result, nil
	case http.StatusAccepted:
		// Accepted WITH warnings. The invoice IS reported/cleared —
		// resubmitting would duplicate it. Surface warnings to the caller.
		result.Accepted = true
		result.WithWarnings = true
		return result, nil
	case http.StatusBadRequest:
		return result, fmt.Errorf("zatca api: invoice rejected (400): %s", summarizeMessages(envelope.ValidationResults.ErrorMessages))
	case http.StatusUnauthorized:
		return result, errors.New("zatca api: authentication failed (401) — CSID token/secret invalid or expired")
	case http.StatusRequestEntityTooLarge:
		return result, errors.New("zatca api: invoice payload too large (413)")
	default:
		return result, fmt.Errorf("zatca api: unexpected status %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
}

func (c *Client) doCSID(req *http.Request) (*CSIDResponse, error) {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("zatca api: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("zatca api: reading response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("zatca api: CSID request failed (%d): %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out CSIDResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("zatca api: malformed CSID response: %w", err)
	}
	if out.BinarySecurityToken == "" || out.Secret == "" {
		return nil, errors.New("zatca api: CSID response missing token or secret")
	}
	return &out, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body []byte, withAuth bool) (*http.Request, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	env := c.Environment
	if env == "" {
		env = EnvSandbox
	}
	url := fmt.Sprintf("%s/e-invoicing/%s/%s", base, env, path)
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("zatca api: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Version", "V2")
	req.Header.Set("Accept-Language", "en")
	if withAuth {
		if c.Credentials.Token == "" || c.Credentials.Secret == "" {
			return nil, errors.New("zatca api: missing CSID credentials")
		}
		req.SetBasicAuth(c.Credentials.Token, c.Credentials.Secret)
	}
	return req, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func summarizeMessages(msgs []ValidationMessage) string {
	if len(msgs) == 0 {
		return "no error details returned"
	}
	parts := make([]string, 0, len(msgs))
	for _, m := range msgs {
		parts = append(parts, m.Code+": "+m.Message)
	}
	return strings.Join(parts, "; ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
