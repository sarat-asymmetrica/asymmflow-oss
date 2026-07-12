// Identity-write policy for customers and suppliers (PH convergence Band-2,
// PH customer_write_policy.go / supplier_write_policy.go, 35bb48c..3f87e3a,
// plus the later G1 field-mask merge). The host keeps RBAC and validation;
// identifier assignment, blank-refill, and the non-destructive update merge
// live here so every write path shares one seam.
package customer

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"ph_holdings_app/pkg/crm"
	"ph_holdings_app/pkg/kernel/apperr"
)

// businessPrefix collects the first limit A-Z runes of the uppercased name,
// skipping everything else until enough letters are found (PH's rule — the
// pre-convergence root truncated to limit chars first and then stripped,
// which shortened prefixes for names with early spaces/punctuation).
func businessPrefix(name string, limit int, fallback string) string {
	name = strings.ToUpper(name)
	builder := strings.Builder{}
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			builder.WriteRune(r)
			if builder.Len() == limit {
				break
			}
		}
	}
	if builder.Len() == 0 {
		return fallback
	}
	return builder.String()
}

// AssignCustomerIdentifiers fills CustomerCode and CustomerID bidirectionally:
// either one seeds the other, and when both are blank a default code of the
// form CUST-<PREFIX4><ms%100000> is generated from the business name.
func AssignCustomerIdentifiers(c *crm.CustomerMaster, now time.Time) {
	c.CustomerCode = strings.TrimSpace(c.CustomerCode)
	c.CustomerID = strings.TrimSpace(c.CustomerID)

	if c.CustomerCode == "" {
		if c.CustomerID != "" {
			c.CustomerCode = c.CustomerID
		} else {
			c.CustomerCode = fmt.Sprintf("CUST-%s%d", businessPrefix(c.BusinessName, 4, "CUST"), now.UnixMilli()%100000)
		}
	}
	if c.CustomerID == "" {
		c.CustomerID = c.CustomerCode
	}
}

// AssignSupplierIdentifiers fills a blank SupplierCode with a generated
// SUP-<PREFIX4><ms%100000> code (suppliers carry no second identifier).
func AssignSupplierIdentifiers(s *crm.SupplierMaster, now time.Time) {
	s.SupplierCode = strings.TrimSpace(s.SupplierCode)
	if s.SupplierCode == "" {
		s.SupplierCode = fmt.Sprintf("SUP-%s%d", businessPrefix(s.SupplierName, 4, "SUP"), now.UnixMilli()%100000)
	}
}

// PrepareCustomerCreate normalizes a new customer's identity fields.
func PrepareCustomerCreate(c *crm.CustomerMaster, now time.Time) error {
	c.BusinessName = strings.TrimSpace(c.BusinessName)
	if c.BusinessName == "" {
		return apperr.New("BUSINESS_NAME_REQUIRED", "Business name is required", "")
	}
	AssignCustomerIdentifiers(c, now)
	return nil
}

// PrepareSupplierCreate normalizes a new supplier's identity fields and
// applies the default rating (0 means "not provided"; valid ratings are 1-5).
func PrepareSupplierCreate(s *crm.SupplierMaster, now time.Time) error {
	s.SupplierName = strings.TrimSpace(s.SupplierName)
	if s.SupplierName == "" {
		return apperr.New("SUPPLIER_NAME_REQUIRED", "Supplier name is required", "")
	}
	AssignSupplierIdentifiers(s, now)
	if s.Rating == 0 {
		s.Rating = 3
	}
	return nil
}

// MergeCustomerUpdate overlays only the user-editable fields of incoming onto
// existing (PH's G1 fix). A full gorm Save of the incoming struct writes every
// column, so any field the caller omitted (zero-valued) silently clobbered the
// stored value — wiping server-owned metrics (order totals, AR risk,
// outstanding, computed grade) on every partial update. Blanking an editable
// field (website, phone, ...) remains a legitimate edit; the unique business
// keys fall back to existing when blank. Server-owned columns are never taken
// from the caller: CustomerID, TotalOrdersValue, TotalOrdersCount,
// AvgOrderValue, LastOrderDate, AvgPaymentDays, DisputeCount, ARRiskTier,
// OutstandingBHD, OverdueDays, CustomerGrade, and the Base audit fields.
func MergeCustomerUpdate(existing *crm.CustomerMaster, incoming crm.CustomerMaster, now time.Time) {
	if code := strings.TrimSpace(incoming.CustomerCode); code != "" {
		existing.CustomerCode = code
	}
	if name := strings.TrimSpace(incoming.BusinessName); name != "" {
		existing.BusinessName = name
	}
	existing.CustomerType = incoming.CustomerType
	existing.ShortCode = incoming.ShortCode
	existing.TradingName = incoming.TradingName
	existing.CRNumber = incoming.CRNumber
	existing.Status = incoming.Status
	// Contact & regional
	existing.PrimaryPhone = incoming.PrimaryPhone
	existing.PrimaryEmail = incoming.PrimaryEmail
	existing.Website = incoming.Website
	existing.AddressLine1 = incoming.AddressLine1
	existing.City = incoming.City
	existing.Country = incoming.Country
	existing.TRN = incoming.TRN
	// Wave 9.6 (C2 residue): CustomerFullProfile doesn't expose mobile_number, so
	// the detail edit form blank-seeds it and every save was wiping a real mobile.
	// Fall back on blank rather than clobber — same contract as CreditLimitBHD.
	if strings.TrimSpace(incoming.MobileNumber) != "" {
		existing.MobileNumber = incoming.MobileNumber
	}
	// Business details
	existing.Industry = incoming.Industry
	existing.RelationYears = incoming.RelationYears
	// Financial (user-set)
	existing.PaymentGrade = incoming.PaymentGrade
	existing.PaymentTermsDays = incoming.PaymentTermsDays
	// Wave 9.6 Sh2: CreditLimitBHD 0 means "not provided" (the detail-edit form
	// doesn't carry this field) — fall back rather than wiping a negotiated limit.
	// Mirrors the supplier Rating fallback. Set a real 0 limit via IsCreditBlocked.
	if incoming.CreditLimitBHD > 0 {
		existing.CreditLimitBHD = incoming.CreditLimitBHD
	}
	existing.IsCreditBlocked = incoming.IsCreditBlocked
	existing.RequiresPrepayment = incoming.RequiresPrepayment
	existing.HasABBCompetition = incoming.HasABBCompetition
	existing.IsEmergencyOnly = incoming.IsEmergencyOnly

	// Repair pathologically blank identifiers on legacy rows in passing.
	AssignCustomerIdentifiers(existing, now)
	existing.Version++
}

// MergeSupplierUpdate is the supplier analog of MergeCustomerUpdate. Every
// supplier column is user-editable except the Base audit fields; the unique
// code/name fall back to existing when blank, and Rating 0 means "not
// provided" so it falls back rather than wiping a real rating.
func MergeSupplierUpdate(existing *crm.SupplierMaster, incoming crm.SupplierMaster, now time.Time) {
	if code := strings.TrimSpace(incoming.SupplierCode); code != "" {
		existing.SupplierCode = code
	}
	if name := strings.TrimSpace(incoming.SupplierName); name != "" {
		existing.SupplierName = name
	}
	existing.Country = incoming.Country
	existing.LeadTimeDays = incoming.LeadTimeDays
	existing.TaxID = incoming.TaxID
	existing.SupplierType = incoming.SupplierType
	existing.BrandsHandled = incoming.BrandsHandled
	existing.ProductTypes = incoming.ProductTypes
	existing.PrimaryContact = incoming.PrimaryContact
	existing.Email = incoming.Email
	existing.Phone = incoming.Phone
	existing.Address = incoming.Address
	existing.BankName = incoming.BankName
	existing.AccountNumber = incoming.AccountNumber
	existing.IBAN = incoming.IBAN
	existing.SwiftCode = incoming.SwiftCode
	existing.PaymentTerms = incoming.PaymentTerms
	if incoming.Rating != 0 {
		existing.Rating = incoming.Rating
	}
	// Wave 9.6 (C2 residue): the supplier detail edit payload can't carry the
	// free-text Notes column (the read profile exposes Notes as []EntityNote, so
	// the builder sends ""), which was wiping a real note on every save. Fall
	// back on blank rather than clobber — same contract as Rating above.
	if strings.TrimSpace(incoming.Notes) != "" {
		existing.Notes = incoming.Notes
	}

	AssignSupplierIdentifiers(existing, now)
	existing.Version++
}

var uuidShapedRef = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// NormalizeBusinessIdentifier discards identifiers that are not real business
// identifiers: blanks, the record's own UUID, or anything UUID-shaped (legacy
// rows where the primary key leaked into the business-identifier column).
func NormalizeBusinessIdentifier(raw, recordID string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == strings.TrimSpace(recordID) || uuidShapedRef.MatchString(raw) {
		return ""
	}
	return raw
}

// RepairCustomerBusinessID derives a stable business identifier for a legacy
// customer row: the (normalized) CustomerCode when present, otherwise
// CUST-<PREFIX6> from the business name. Unlike the create default it carries
// no timestamp suffix, so repeat repairs are deterministic.
func RepairCustomerBusinessID(c crm.CustomerMaster) string {
	if code := NormalizeBusinessIdentifier(c.CustomerCode, c.ID); code != "" {
		return code
	}
	return fmt.Sprintf("CUST-%s", businessPrefix(c.BusinessName, 6, "CUST"))
}
