package saudi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testSubmission() InvoiceSubmission {
	return InvoiceSubmission{
		InvoiceHash: "x9BpmXhs7cxIkI85dWyBXOOZWLKgYQdVaBTMbeMSKGs=",
		UUID:        "8e6000cf-1a98-4174-b3e7-b5d5954bc10d",
		InvoiceXML:  []byte("<Invoice>signed</Invoice>"),
	}
}

func newTestClient(srvURL string) *Client {
	c := NewClient(EnvSandbox, Credentials{Token: "token-abc", Secret: "secret-xyz"})
	c.BaseURL = srvURL
	return c
}

func TestReportInvoiceAccepted(t *testing.T) {
	var got struct {
		path, auth, version, contentType string
		body                             map[string]string
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path = r.URL.Path
		got.auth = r.Header.Get("Authorization")
		got.version = r.Header.Get("Accept-Version")
		got.contentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&got.body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"reportingStatus":"REPORTED","validationResults":{"status":"PASS"}}`))
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ReportInvoice(context.Background(), testSubmission())
	if err != nil {
		t.Fatalf("ReportInvoice: %v", err)
	}
	if !res.Accepted || res.WithWarnings || res.ReportingStatus != "REPORTED" {
		t.Errorf("result = %+v", res)
	}
	if got.path != "/e-invoicing/developer-portal/invoices/reporting/single" {
		t.Errorf("path = %s", got.path)
	}
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("token-abc:secret-xyz"))
	if got.auth != wantAuth {
		t.Errorf("auth = %s", got.auth)
	}
	if got.version != "V2" {
		t.Errorf("Accept-Version = %s", got.version)
	}
	if got.body["invoiceHash"] != testSubmission().InvoiceHash || got.body["uuid"] != testSubmission().UUID {
		t.Errorf("body = %+v", got.body)
	}
	if decoded, _ := base64.StdEncoding.DecodeString(got.body["invoice"]); string(decoded) != "<Invoice>signed</Invoice>" {
		t.Errorf("invoice payload not base64 XML: %q", got.body["invoice"])
	}
}

func TestReportInvoice202AcceptedWithWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"reportingStatus":"REPORTED","validationResults":{"status":"WARNING","warningMessages":[{"code":"BR-KSA-W1","message":"minor issue"}]}}`))
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ReportInvoice(context.Background(), testSubmission())
	if err != nil {
		t.Fatalf("202 must NOT be an error (the invoice IS reported): %v", err)
	}
	if !res.Accepted || !res.WithWarnings {
		t.Errorf("202 semantics wrong: %+v", res)
	}
	if len(res.Validation.WarningMessages) != 1 || res.Validation.WarningMessages[0].Code != "BR-KSA-W1" {
		t.Errorf("warnings not surfaced: %+v", res.Validation)
	}
}

func TestReportInvoice400Rejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"validationResults":{"status":"ERROR","errorMessages":[{"code":"BR-KSA-08","message":"invalid VAT number"}]}}`))
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ReportInvoice(context.Background(), testSubmission())
	if err == nil {
		t.Fatal("400 must be an error")
	}
	if !strings.Contains(err.Error(), "BR-KSA-08") || !strings.Contains(err.Error(), "invalid VAT number") {
		t.Errorf("error should carry gateway messages: %v", err)
	}
	if res == nil || res.Accepted {
		t.Errorf("rejected result: %+v", res)
	}
}

func TestClearInvoiceReturnsStampedXML(t *testing.T) {
	stamped := "<Invoice>ZATCA-stamped</Invoice>"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Clearance-Status") != "1" {
			t.Errorf("Clearance-Status header missing")
		}
		if !strings.Contains(r.URL.Path, "/invoices/clearance/single") {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"clearanceStatus":"CLEARED","clearedInvoice":"` +
			base64.StdEncoding.EncodeToString([]byte(stamped)) + `"}`))
	}))
	defer srv.Close()

	res, err := newTestClient(srv.URL).ClearInvoice(context.Background(), testSubmission())
	if err != nil {
		t.Fatalf("ClearInvoice: %v", err)
	}
	if res.ClearanceStatus != "CLEARED" || string(res.ClearedInvoiceXML) != stamped {
		t.Errorf("result = %+v, cleared = %q", res, res.ClearedInvoiceXML)
	}
}

func TestAuthFailure401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	_, err := newTestClient(srv.URL).ReportInvoice(context.Background(), testSubmission())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("401 handling: %v", err)
	}
}

func TestOnboardingCSIDFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/compliance"):
			if r.Header.Get("OTP") != "123456" {
				t.Errorf("OTP header missing")
			}
			if r.Header.Get("Authorization") != "" {
				t.Errorf("compliance CSID request must not carry basic auth")
			}
			_, _ = w.Write([]byte(`{"requestID":1234567890123,"binarySecurityToken":"cc-token","secret":"cc-secret"}`))
		case strings.HasSuffix(r.URL.Path, "/production/csids"):
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["compliance_request_id"] != "1234567890123" {
				t.Errorf("compliance_request_id = %q", body["compliance_request_id"])
			}
			_, _ = w.Write([]byte(`{"requestID":99,"binarySecurityToken":"prod-token","secret":"prod-secret"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := NewClient(EnvSandbox, Credentials{})
	c.BaseURL = srv.URL
	cc, err := c.RequestComplianceCSID(context.Background(), "base64-csr", "123456")
	if err != nil {
		t.Fatalf("RequestComplianceCSID: %v", err)
	}
	if cc.BinarySecurityToken != "cc-token" {
		t.Errorf("compliance CSID = %+v", cc)
	}

	c.Credentials = cc.AsCredentials()
	prod, err := c.RequestProductionCSID(context.Background(), cc.RequestID.String())
	if err != nil {
		t.Fatalf("RequestProductionCSID: %v", err)
	}
	if prod.BinarySecurityToken != "prod-token" || prod.Secret != "prod-secret" {
		t.Errorf("production CSID = %+v", prod)
	}
}

func TestSubmissionValidation(t *testing.T) {
	c := NewClient(EnvSandbox, Credentials{Token: "t", Secret: "s"})
	if _, err := c.ReportInvoice(context.Background(), InvoiceSubmission{}); err == nil {
		t.Error("empty submission should error before any HTTP call")
	}
	c2 := NewClient(EnvSandbox, Credentials{})
	if _, err := c2.ReportInvoice(context.Background(), testSubmission()); err == nil {
		t.Error("missing credentials should error")
	}
}
