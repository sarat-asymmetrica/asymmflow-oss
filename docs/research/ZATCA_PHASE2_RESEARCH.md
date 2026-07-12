# ZATCA Phase 2 (Fatoora) — Implementation Research Notes

**Date:** 2026-07-03 · **For:** Wave 2 Mission B (`pkg/compliance/saudi/`)
**Method:** web research cross-checking ZATCA's own spec PDFs (XML Implementation
Standard v1.2, Security Features Implementation Standard v1.2, Detailed Technical
Guidelines) against Microsoft Dynamics onboarding docs, ClearTax API docs, and OSS
implementations. Items marked ⚠️ are places community code is commonly wrong;
items marked ❓ could not be verified against primary source and must be checked
against ZATCA's sample invoice set before production use.

---

## 1. VAT rules

- Standard rate **15%** (since 2020-07-01). Category code **S**.
- **Zero-rated (Z, 0%):** exports outside GCC, international transport, qualifying
  medicines/medical goods, investment metals ≥99% purity, specific non-resident supplies.
- **Exempt (E):** margin-based financial services, life insurance/reinsurance,
  residential real-estate lease/sale (with exceptions). No input recovery.
- **Out-of-scope (O):** supplies outside KSA VAT scope.
- **Reverse charge (imported services):** KSA buyer self-accounts 15% output VAT and
  reclaims as input if entitled (net-zero when fully deductible). Represented via tax
  category + reason; largely a VAT-return mechanism rather than a cleared e-invoice.
- **Rounding:** SAR, 2 decimals (halalas). BR-KSA rules enforce
  `TaxAmount = round(TaxableAmount × rate/100, 2)` per tax subtotal; document totals
  must reconcile. Unit prices may carry more decimals.
- For Z/E/O a `TaxExemptionReasonCode` (+ text) from ZATCA's `VATEX-SA-*` list is mandatory.
- Input-tax deduction requires a valid **standard** (cleared) invoice showing the buyer's
  VAT number — this drives B2B→clearance routing.

## 2. UBL 2.1 XML

- `ProfileID = "reporting:1.0"`. XML is the legal artifact (PDF/A-3 embed allowed for humans).
- **InvoiceTypeCode (UNTDID 1001):** 388 tax invoice · 381 credit note · 383 debit note ·
  386 prepayment. Credit/debit notes carry `BillingReference` to the original +
  mandatory `InstructionNote` (reason).
- **Subtype in `InvoiceTypeCode/@name`** — ⚠️ 7-position digit string per ZATCA core
  (some vendor docs say 9; make width version-configurable, default 7):
  - Pos 1–2: `01` standard (B2B, cleared) · `02` simplified (B2C, reported)
  - Pos 3 third-party · 4 nominal · 5 exports · 6 summary · 7 self-billed (each 0/1)
  - e.g. `0100000` plain standard, `0200000` plain simplified.
- **Key mandatory fields:** `ID` (human number), `UUID` (v4, distinct from ID),
  `IssueDate`/`IssueTime`, `DocumentCurrencyCode` + `TaxCurrencyCode=SAR` (VAT must be
  expressed in SAR even for FX invoices — second `TaxTotal` with SAR amount only),
  **ICV** (AdditionalDocumentReference ID="ICV", monotonic counter per EGS, never resets),
  **PIH** (AdditionalDocumentReference ID="PIH", base64 SHA-256 of previous invoice;
  genesis = base64(SHA256("0")) =
  `NWZlY2ViNjZmZmM4NmYzOGQ5NTI3ODZjNmQ2OTZjNzljMmRiYzIzOWRkNGU5MWI0NjcyOWQ3M2EyN2ZiNTdlOQ==`),
  **QR** (AdditionalDocumentReference ID="QR"; for standard invoices ZATCA returns it on clearance).
- **Seller:** registration name; VAT number in `PartyTaxScheme/CompanyID` — **15 digits,
  starts AND ends with 3**; full address (4-digit building number, district, city,
  5-digit postal code, country SA); CRN/other ID in `PartyIdentification` (schemes CRN/MOM/MLS/700/SAG…).
- **Buyer:** mandatory (VAT number + address) for standard; largely optional for simplified.
- **Lines:** qty+unitCode, LineExtensionAmount, Item/Name, Price, per-line `TaxTotal`
  (KSA line-level rounding), `ClassifiedTaxCategory` (S/Z/E/O + Percent + TaxScheme VAT).
- **LegalMonetaryTotal:** LineExtension, TaxExclusive, TaxInclusive, AllowanceTotal,
  Prepaid, Payable. AllowanceCharge blocks carry their own TaxCategory and must reconcile.
- **Signature skeleton:** XAdES-BES enveloped inside
  `UBLExtensions/UBLExtension/ExtensionContent/sig:UBLDocumentSignatures` — two
  `ds:Reference`s (invoice digest with enveloped+XPath transforms; `#xadesSignedProperties`
  digest), `ds:SignatureValue`, cert in `KeyInfo`, `xades:SignedProperties` with
  SigningTime + CertDigest + IssuerSerial.

## 3. Cryptography (exact)

- **Invoice hash:** remove `UBLExtensions`, `cac:Signature`, and the QR
  AdditionalDocumentReference → canonicalize with **C14N 1.1**
  (`http://www.w3.org/2006/12/xml-c14n11`) → SHA-256 → **base64 of raw digest**.
  ⚠️ C14N 1.0 (`$dom->C14N()`-style) is the #1 cause of "invalid invoice hash".
- **Signature:** **ECDSA secp256k1 + SHA-256** over canonicalized `SignedInfo`.
  ⚠️ secp256k1, NOT P-256 — #1 cause of signature rejections.
- **SignedProperties digest:** base64 SHA-256 of canonicalized `xades:SignedProperties`.
- **Cert digest:** base64 SHA-256 of the signing cert in `xades:CertDigest`.
- **CSR:** EC secp256k1 key; Subject `C=SA, OU=<unit/10-digit TIN for VAT groups>,
  O=<org>, CN=<unique unit name>`; extension `certificateTemplateName`
  (OID 1.3.6.1.4.1.311.20.2) = `ZATCA-Code-Signing` (prod) / `PREZATCA-Code-Signing`
  (simulation) / `TSTZATCA-Code-Signing` (sandbox); SAN dirName:
  `SN=1-<solution>|2-<version>|3-<deviceUUID>`, `UID=<15-digit VAT>`,
  `title=<TSCZ 4-digit doc-type flags, e.g. 1100 = standard+simplified>`,
  `registeredAddress`, `businessCategory`. keyUsage digitalSignature+nonRepudiation+keyEncipherment.
- **Onboarding:** Fatoora portal OTP (1h validity) → POST CSR to compliance endpoint
  (headers `OTP`, `Accept-Version: V2`) → returns `{binarySecurityToken (CCSID cert),
  secret, requestID}` → run compliance sample invoices (count depends on `title` flags)
  → POST `production/csids` with `{compliance_request_id}` → PCSID. Renewal = PATCH
  same path. Basic-auth = base64(token:secret) throughout.

## 4. QR code TLV

`[1-byte tag][1-byte length][value]` triples concatenated → base64 whole → QR content.

| Tag | Value | Encoding |
|---|---|---|
| 1 | Seller name | UTF-8 |
| 2 | Seller VAT number (15 digits) | UTF-8 |
| 3 | Timestamp `YYYY-MM-DDTHH:MM:SSZ` | UTF-8 |
| 4 | Invoice total WITH VAT | UTF-8 decimal string |
| 5 | VAT total | UTF-8 decimal string |
| 6 | Invoice hash (base64 SHA-256) | UTF-8 string of the base64 |
| 7 | ECDSA signature | raw bytes |
| 8 | EGS public key (DER SubjectPublicKeyInfo) | raw bytes |
| 9 | ZATCA CA's signature over the cert (simplified only) ❓ | raw bytes |

- Phase 1 QR = tags 1–5. Phase 2 simplified = 1–9 (offline-verifiable). Phase 2 standard:
  ZATCA generates the QR on clearance — embed what ZATCA returns (tags 6–8 present).
- ⚠️ Tags 8/9 are raw binary — double-base64ing them, or using the whole cert instead of
  just its signature for tag 9, breaks verification.

## 5. API

Host: `https://gw-fatoora.zatca.gov.sa` — ⚠️ legacy `gw-apic-gov.gazt.gov.sa`
decommissioned 2025-09-14.

Environments by path segment: `developer-portal` (sandbox, TSTZATCA) ·
`simulation` (PREZATCA) · `core` (production, ZATCA).

| Purpose | Method + path (core shown) |
|---|---|
| Compliance CSID | `POST /e-invoicing/core/compliance` (header OTP, body `{csr}`) |
| Compliance invoice check | `POST /e-invoicing/core/compliance/invoices` (CCSID auth) |
| Production CSID | `POST /e-invoicing/core/production/csids` (`{compliance_request_id}`); `PATCH` to renew |
| Reporting (simplified, ≤24h) | `POST /e-invoicing/core/invoices/reporting/single` |
| Clearance (standard, pre-share) | `POST /e-invoicing/core/invoices/clearance/single` |

Headers: `Authorization: Basic base64(token:secret)`, `Accept-Version: V2`,
`Accept-Language: en`, `Content-Type: application/json`, `Clearance-Status: 1` (clearance).

Body: `{"invoiceHash": "<base64 sha256>", "uuid": "<cbc:UUID>", "invoice": "<base64 signed XML>"}`.

Status codes (raw ZATCA gateway; ⚠️ middleware providers collapse to 200+body-status —
branch on BOTH HTTP code and body):
- **200** cleared/reported · **202** accepted WITH warnings (do NOT resubmit) ·
  **400** rejected (errorMessages) · **401** auth failed · **413** too large ·
  **429** rate-limited · **5xx** retry with backoff.

Response: `clearanceStatus`/`reportingStatus`, `validationResults{info,warning,error}`,
and for clearance `clearedInvoice` = base64 ZATCA-stamped XML — **persist and share THAT,
not your submitted XML**.

## 6. Go ecosystem

No production-grade Phase-2 Go library exists (Haraj-backend/zatca-sdk-go = Phase-1 QR
only, Dec 2021). Best OSS reference architecture: `Saleh7/php-zatca-xml` (PHP, aligned to
ZATCA SDK R3.4.8). Building blocks for us:
- secp256k1: `github.com/decred/dcrd/dcrec/secp256k1/v4` (pure Go, RFC 6979, no CGO ✅).
- C14N 1.1: **no good pure-Go lib** — `goxmldsig`/`signedxml` don't do c14n11 +
  secp256k1 + ZATCA transforms. Highest-risk component; implement and pin with
  byte-for-byte tests against ZATCA sample invoices.
- QR render: `github.com/skip2/go-qrcode` (TLV encoding is trivial, do it ourselves).

## 7. Rollout context (mid-2026)

Wave 24 (revenue > SAR 375k) integration deadline was 2026-06-30; waves keep coming
~quarterly with dropping thresholds. **Design as "already mandated" — don't gate on wave.**
Routing rule: buyer has valid VAT number + B2B → clearance; else → reporting ≤24h.

## Sources

- ZATCA XML Implementation Standard v1.2 / Security Features Standard v1.2 /
  Detailed Technical Guideline (zatca.gov.sa → E-Invoicing → Systems Developers)
- Microsoft Dynamics 365 ZATCA onboarding: learn.microsoft.com/en-us/dynamics365/finance/localizations/mea/gs-e-invoicing-sa-onboarding
- ClearTax KSA API reference (status codes/response fields)
- github.com/Saleh7/php-zatca-xml · github.com/SallaApp/ZATCA · Haraj-backend/zatca-sdk-go
- Fatoora Developer Community: zatca1.discourse.group/t/http-response-errors/991
- Wave 23/24: flick.network + elitemindz.co 2026 guides

## ❓ Resolution (Wave 3, 2026-07-03)

Both open flags are RESOLVED against ZATCA's own SDK (downloaded from
zatca.gov.sa → Systems Developers → Download SDK; the zip served is SDK
v2.03 / cli-3.0.8, and its reference materials are vendored in
`pkg/compliance/saudi/testdata/sdk203/`):

- **ds:Reference XPath transforms — CONFIRMED (primary source).** The SDK
  jar's own signing template (`xml/ubl.xml`) carries exactly the three
  XPath transforms + trailing c14n11 transform our emitter produces, in the
  same order, with the same CanonicalizationMethod (c14n11) and
  SignatureMethod (ecdsa-sha256). Pinned by
  `TestSignedInfo_MatchesSDKTemplate`, which extracts the strings from the
  vendored template rather than hardcoding them twice. Two independent
  SDK-aligned implementations (Saleh7/php-zatca-xml, wes4m/zatca-xml-js)
  agree byte-for-byte.
- **QR tag 9 — CONFIRMED as the certificate's signature bytes** (the ZATCA
  CA's signature over the EGS certificate), per two current SDK-aligned
  implementations; our `Certificate.SignatureValue` matches. For STANDARD
  invoices the question is moot in this codebase: the QR is only minted
  locally for simplified invoices; cleared invoices embed ZATCA's returned
  QR. **Version caveat:** the SDK v2.03 zip predates the final QR spec —
  its validator still expects the DRAFT semantics (tags 8/9 = signature R/S
  values). Do not use that validator for QR checks; current-SDK (R3.4.x)
  semantics are 7=signature, 8=SPKI, 9=cert signature.

Bonus primary-source confirmations from the same materials: the ZATCA test
CA encodes the SAN EGS serial as **surname (2.5.4.4)** — the OpenSSL "SN"
alias — validating our CSR profile (W3-D8); the SDK's `Data/PIH/pih.txt`
genesis equals our `GenesisPIH`; and the SDK's bare-base64 (headerless)
key/cert files exposed that `ParseCertificatePEM`/`ParsePrivateKeyPEM`
required PEM armor the real binarySecurityToken doesn't have — fixed, with
the SDK materials as the regression fixtures (`sdk_compat_test.go`).

**Still open for Wave 4 (needs portal OTP / live gateway):** compliance-check
round-trip of our emitted invoices against the sandbox gateway, and an
R3.4.x SDK validator run (the current SDK requires JDK 11–14, not installed).
