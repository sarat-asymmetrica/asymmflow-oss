package main

import (
	"fmt"
	"net/mail"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"ph_holdings_app/pkg/infra/audit"
	"ph_holdings_app/pkg/infra/ratelimit"
)

// =============================================================================
// P1 SECURITY ENHANCEMENTS
// =============================================================================
// This file implements 5 critical security improvements:
// 1. Input Validation (length limits, format validation, sanitization)
// 2. Sensitive Data Masking (logs, payments, contacts)
// 3. Session Security (timeout, activity tracking)
// 4. Audit Logging (financial transactions, permission changes)
// 5. Rate Limiting (login attempts, API calls, reports)
// =============================================================================

// =============================================================================
// 1. INPUT VALIDATION
// =============================================================================

// ValidationLimits defines maximum lengths for various input fields
var ValidationLimits = struct {
	CustomerName  int
	SupplierName  int
	Email         int
	Phone         int
	Address       int
	Notes         int
	Reference     int
	InvoiceNumber int
	Description   int
	Filename      int
	ShortCode     int
	Username      int
	FullName      int
	Password      int
	ReportName    int
}{
	CustomerName:  200,
	SupplierName:  255,
	Email:         255,
	Phone:         50,
	Address:       500,
	Notes:         5000,
	Reference:     100,
	InvoiceNumber: 50,
	Description:   1000,
	Filename:      255,
	ShortCode:     10,
	Username:      50,
	FullName:      100,
	Password:      100,
	ReportName:    100,
}

// InputValidator provides validation methods for user inputs
type InputValidator struct{}

// NewInputValidator creates a new input validator
func NewInputValidator() *InputValidator {
	return &InputValidator{}
}

// ValidateString validates string length and optionally checks for dangerous characters
func (v *InputValidator) ValidateString(field, value string, maxLength int, allowSpecialChars bool) error {
	// Check UTF-8 validity
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s contains invalid UTF-8 characters", field)
	}

	// Check length
	if len(value) > maxLength {
		return fmt.Errorf("%s exceeds maximum length of %d characters (got %d)", field, maxLength, len(value))
	}

	// Check for null bytes (security risk)
	if strings.Contains(value, "\x00") {
		return fmt.Errorf("%s contains null bytes", field)
	}

	// If special characters not allowed, check for common dangerous patterns
	if !allowSpecialChars {
		// Block common injection patterns
		dangerous := []string{"<script", "javascript:", "onerror=", "onload=", "../", "..\\"}
		lowerValue := strings.ToLower(value)
		for _, pattern := range dangerous {
			if strings.Contains(lowerValue, pattern) {
				return fmt.Errorf("%s contains potentially dangerous pattern: %s", field, pattern)
			}
		}
	}

	return nil
}

// ValidateEmail validates email format
func (v *InputValidator) ValidateEmail(email string) error {
	if email == "" {
		return nil // Empty email is allowed (optional field)
	}

	if len(email) > ValidationLimits.Email {
		return fmt.Errorf("email exceeds maximum length of %d characters", ValidationLimits.Email)
	}

	// Use standard library email parser
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}

	return nil
}

// ValidatePhone validates phone number format (international format)
func (v *InputValidator) ValidatePhone(phone string) error {
	if phone == "" {
		return nil // Empty phone is allowed (optional field)
	}

	if len(phone) > ValidationLimits.Phone {
		return fmt.Errorf("phone number exceeds maximum length of %d characters", ValidationLimits.Phone)
	}

	// Allow only digits, spaces, +, -, (, )
	phoneRegex := regexp.MustCompile(`^[\d\s\+\-\(\)]+$`)
	if !phoneRegex.MatchString(phone) {
		return fmt.Errorf("phone number contains invalid characters (only digits, spaces, +, -, (, ) allowed)")
	}

	return nil
}

// SanitizeFilePath validates and sanitizes file paths to prevent path traversal
func (v *InputValidator) SanitizeFilePath(path string) (string, error) {
	// Check length
	if len(path) > ValidationLimits.Filename {
		return "", fmt.Errorf("file path exceeds maximum length of %d characters", ValidationLimits.Filename)
	}

	// Clean the path
	cleaned := filepath.Clean(path)

	// Prevent path traversal
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("path traversal detected in file path")
	}

	// Block absolute paths (Windows and Unix)
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths not allowed")
	}
	if regexp.MustCompile(`^[A-Za-z]:[\\/]+`).MatchString(cleaned) {
		return "", fmt.Errorf("absolute paths not allowed")
	}

	return cleaned, nil
}

// ValidateCustomerInput validates customer master data inputs
func (v *InputValidator) ValidateCustomerInput(customer *CustomerMaster) error {
	// BusinessName is required
	if strings.TrimSpace(customer.BusinessName) == "" {
		return fmt.Errorf("Business Name is required")
	}
	if err := v.ValidateString("Business Name", customer.BusinessName, ValidationLimits.CustomerName, false); err != nil {
		return err
	}

	if customer.ShortCode != "" {
		if err := v.ValidateString("Short Code", customer.ShortCode, ValidationLimits.ShortCode, false); err != nil {
			return err
		}
	}

	// Validate TRN length (not email — it's a tax registration number)
	if len(customer.TRN) > 100 {
		return fmt.Errorf("TRN exceeds maximum length of 100 characters")
	}

	// Validate MobileNumber if provided
	if customer.MobileNumber != "" {
		if len(customer.MobileNumber) > 50 {
			return fmt.Errorf("mobile number exceeds maximum length of 50 characters")
		}
	}

	// Validate PrimaryEmail if provided
	if customer.PrimaryEmail != "" {
		if err := v.ValidateEmail(customer.PrimaryEmail); err != nil {
			return err
		}
	}

	// Validate PrimaryPhone length
	if len(customer.PrimaryPhone) > 50 {
		return fmt.Errorf("primary phone exceeds maximum length of 50 characters")
	}

	// Validate CRNumber length
	if len(customer.CRNumber) > 100 {
		return fmt.Errorf("CR number exceeds maximum length of 100 characters")
	}

	// Validate TradingName length
	if len(customer.TradingName) > 255 {
		return fmt.Errorf("trading name exceeds maximum length of 255 characters")
	}

	// Validate Website length
	if len(customer.Website) > 255 {
		return fmt.Errorf("website exceeds maximum length of 255 characters")
	}

	// Validate Status whitelist
	if customer.Status != "" {
		validStatuses := map[string]bool{"Active": true, "Inactive": true, "Blacklisted": true}
		if !validStatuses[customer.Status] {
			return fmt.Errorf("invalid status: must be Active, Inactive, or Blacklisted")
		}
	}

	// Validate CustomerType whitelist
	if customer.CustomerType != "" {
		validTypes := map[string]bool{"Corporate": true, "Government": true, "Individual": true, "SME": true, "EC": true}
		if !validTypes[customer.CustomerType] {
			return fmt.Errorf("invalid customer type: must be Corporate, Government, Individual, SME, or EC")
		}
	}

	// Validate PaymentGrade whitelist
	if customer.PaymentGrade != "" {
		validGrades := map[string]bool{"A": true, "B": true, "C": true, "D": true}
		if !validGrades[customer.PaymentGrade] {
			return fmt.Errorf("invalid payment grade: must be A, B, C, or D")
		}
	}

	// Validate CreditLimitBHD non-negative
	if customer.CreditLimitBHD < 0 {
		return fmt.Errorf("credit limit must be >= 0")
	}

	// Validate PaymentTermsDays range
	if customer.PaymentTermsDays < 0 || customer.PaymentTermsDays > 365 {
		return fmt.Errorf("payment terms days must be between 0 and 365")
	}

	return nil
}

// ValidateSupplierInput validates supplier master data inputs
func (v *InputValidator) ValidateSupplierInput(supplier *SupplierMaster) error {
	// SupplierName must be non-empty
	if strings.TrimSpace(supplier.SupplierName) == "" {
		return fmt.Errorf("supplier name is required")
	}

	if err := v.ValidateString("Supplier Name", supplier.SupplierName, ValidationLimits.SupplierName, false); err != nil {
		return err
	}

	if err := v.ValidateEmail(supplier.Email); err != nil {
		return err
	}

	if err := v.ValidatePhone(supplier.Phone); err != nil {
		return err
	}

	if err := v.ValidateString("Address", supplier.Address, 1000, true); err != nil {
		return err
	}

	// SupplierType whitelist validation
	if supplier.SupplierType != "" {
		validTypes := map[string]bool{
			"Manufacturer":     true,
			"Distributor":      true,
			"Agent":            true,
			"Service Provider": true,
		}
		if !validTypes[supplier.SupplierType] {
			return fmt.Errorf("invalid supplier type: %s (allowed: Manufacturer, Distributor, Agent, Service Provider)", supplier.SupplierType)
		}
	}

	// LeadTimeDays bounds check
	if supplier.LeadTimeDays < 0 || supplier.LeadTimeDays > 365 {
		return fmt.Errorf("lead time days must be between 0 and 365 (got %d)", supplier.LeadTimeDays)
	}

	// Country length validation
	if len(supplier.Country) > 100 {
		return fmt.Errorf("country exceeds maximum length of 100 characters (got %d)", len(supplier.Country))
	}

	// BrandsHandled length validation
	if len(supplier.BrandsHandled) > 2000 {
		return fmt.Errorf("brands handled exceeds maximum length of 2000 characters (got %d)", len(supplier.BrandsHandled))
	}

	return nil
}

// ValidateNoteInput validates entity note inputs
func (v *InputValidator) ValidateNoteInput(content string) error {
	return v.ValidateString("Note Content", content, ValidationLimits.Notes, true)
}

// ValidateUserInput validates user account inputs
func (v *InputValidator) ValidateUserInput(username, password, fullName, email string) error {
	if err := v.ValidateString("Username", username, ValidationLimits.Username, false); err != nil {
		return err
	}

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	if len(password) > ValidationLimits.Password {
		return fmt.Errorf("password exceeds maximum length of %d characters", ValidationLimits.Password)
	}

	if err := v.ValidateString("Full Name", fullName, ValidationLimits.FullName, false); err != nil {
		return err
	}

	if err := v.ValidateEmail(email); err != nil {
		return err
	}

	return nil
}

// =============================================================================
// 2. SENSITIVE DATA MASKING
// =============================================================================

// DataMasker provides methods to mask sensitive data in logs
type DataMasker struct{}

// NewDataMasker creates a new data masker
func NewDataMasker() *DataMasker {
	return &DataMasker{}
}

// MaskPaymentAmount masks payment amounts for logging (show last 3 digits only for amounts > 100)
func (m *DataMasker) MaskPaymentAmount(amount float64) string {
	if amount < 100 {
		// Small amounts: show full value
		return fmt.Sprintf("%.3f BHD", amount)
	}
	// Large amounts: mask most digits, show last 3 only
	amountStr := fmt.Sprintf("%.3f", amount)
	if len(amountStr) > 7 {
		return "***" + amountStr[len(amountStr)-7:] + " BHD"
	}
	return "***." + amountStr[len(amountStr)-3:] + " BHD"
}

// MaskEmail masks email addresses (show first 2 chars + domain)
func (m *DataMasker) MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***"
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return "**@" + domain
	}

	return local[:2] + "***@" + domain
}

// MaskPhone masks phone numbers (show last 4 digits only)
func (m *DataMasker) MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}

	// Remove non-digit characters
	digits := regexp.MustCompile(`\d+`).FindAllString(phone, -1)
	phoneDigits := strings.Join(digits, "")

	if len(phoneDigits) <= 4 {
		return "****"
	}

	return "***-***-" + phoneDigits[len(phoneDigits)-4:]
}

// MaskBankAccount masks bank account numbers (show last 4 digits)
func (m *DataMasker) MaskBankAccount(account string) string {
	if account == "" {
		return ""
	}

	if len(account) <= 4 {
		return "****"
	}

	return "************" + account[len(account)-4:]
}

// MaskIBAN masks IBAN (show country code + last 4 digits)
func (m *DataMasker) MaskIBAN(iban string) string {
	if iban == "" {
		return ""
	}

	if len(iban) <= 6 {
		return "**************"
	}

	return iban[:2] + "**************" + iban[len(iban)-4:]
}

// SanitizeForLog removes sensitive data from log fields
func (m *DataMasker) SanitizeForLog(fields map[string]any) map[string]any {
	sanitized := make(map[string]any)

	for k, v := range fields {
		lowerKey := strings.ToLower(k)

		// Mask known sensitive fields
		switch {
		case strings.Contains(lowerKey, "password"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "amount") && strings.Contains(lowerKey, "bhd"):
			if amount, ok := v.(float64); ok {
				sanitized[k] = m.MaskPaymentAmount(amount)
			} else {
				sanitized[k] = v
			}
		case strings.Contains(lowerKey, "email"):
			if email, ok := v.(string); ok {
				sanitized[k] = m.MaskEmail(email)
			} else {
				sanitized[k] = v
			}
		case strings.Contains(lowerKey, "phone"):
			if phone, ok := v.(string); ok {
				sanitized[k] = m.MaskPhone(phone)
			} else {
				sanitized[k] = v
			}
		case strings.Contains(lowerKey, "account") || strings.Contains(lowerKey, "iban"):
			if account, ok := v.(string); ok {
				sanitized[k] = m.MaskBankAccount(account)
			} else {
				sanitized[k] = v
			}
		case strings.Contains(lowerKey, "api_key") || strings.Contains(lowerKey, "apikey") || strings.Contains(lowerKey, "api-key"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "token"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "secret"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "authorization") || strings.Contains(lowerKey, "bearer"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "private_key") || strings.Contains(lowerKey, "privatekey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "client_secret") || strings.Contains(lowerKey, "clientsecret"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "db_password") || strings.Contains(lowerKey, "database_password") || strings.Contains(lowerKey, "dbpassword"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "conn_string") || strings.Contains(lowerKey, "connection_string") || strings.Contains(lowerKey, "connstring"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "signing_key") || strings.Contains(lowerKey, "signingkey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "encryption_key") || strings.Contains(lowerKey, "encryptionkey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "access_key") || strings.Contains(lowerKey, "accesskey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "supabase_key") || strings.Contains(lowerKey, "supabasekey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "master_key") || strings.Contains(lowerKey, "masterkey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "jwt_secret") || strings.Contains(lowerKey, "jwtsecret"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "webhook_secret") || strings.Contains(lowerKey, "webhooksecret"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "aiml_key") || strings.Contains(lowerKey, "aimlkey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "credential"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "certificate") || strings.Contains(lowerKey, "cert_"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "salt") || strings.Contains(lowerKey, "hmac"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "mistral_key") || strings.Contains(lowerKey, "mistralkey"):
			sanitized[k] = "***REDACTED***"
		case strings.Contains(lowerKey, "license_key") || strings.Contains(lowerKey, "licensekey"):
			sanitized[k] = "***REDACTED***"
		default:
			sanitized[k] = v
		}
	}

	return sanitized
}

// =============================================================================
// 3. SESSION SECURITY
// =============================================================================

// NOTE (Wave 4 B.2, deliberate deletion): the in-memory SessionManager that
// used to live here was write-only security theater — LoginDevice stored a
// session into a sync.Map (discarding the return value and ignoring the
// deviceID argument), and NOTHING ever called IsSessionValid, UpdateActivity
// or EndSession, so the 8h-inactivity/24h-max-age policy it advertised was
// never enforced anywhere. The REAL session system is the DB-backed
// AuthManager in auth_session.go (create/validate/invalidate/cleanup with
// hashed tokens). One session system now exists. If interactive-session
// inactivity timeout is wanted, wire it through AuthManager as a deliberate
// security change — do not resurrect the map.

// =============================================================================
// 4. AUDIT LOGGING
// =============================================================================

// NOTE (Wave 3 B.2, deliberate deletion): the AuditEvent struct that used to
// live here was constructed and immediately DISCARDED (`_ = AuditEvent{…}`)
// by LogFinancialTransaction — a second audit vocabulary that never reached
// the database. It is deleted, not preserved: the persisted shape is
// pkg/infra/audit.Entry, and AuditLogger now writes through that engine when
// a recorder is wired (SetRecorder, from startup once the DB is up) in
// addition to the structured security log it always emitted.

// AuditLogger logs security-critical events
type AuditLogger struct {
	logger   *Logger
	masker   *DataMasker
	recorder *audit.Recorder // nil until the DB is up; log-only before that
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger *Logger) *AuditLogger {
	return &AuditLogger{
		logger: logger,
		masker: NewDataMasker(),
	}
}

// SetRecorder wires the engine-backed persistence path. Called from startup
// once the database is open; before that, events reach the security log only
// (the historical behavior).
func (a *AuditLogger) SetRecorder(r *audit.Recorder) { a.recorder = r }

// persist writes the event to the audit_logs table when a recorder is wired.
func (a *AuditLogger) persist(userID, action, resource, resourceID, description string) {
	if a.recorder == nil {
		return
	}
	a.recorder.RecordAsync(audit.Entry{
		UserID:      userID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Description: description,
	}, func(err error) {
		a.logger.Security("audit_persist_failed", false, map[string]any{"error": err.Error(), "action": action})
	})
}

// LogFinancialTransaction logs financial transactions (payments, invoices)
func (a *AuditLogger) LogFinancialTransaction(userID, action, entityType, entityID string, amount float64, currency string, success bool, metadata map[string]any) {
	// Mask sensitive data in metadata
	sanitized := a.masker.SanitizeForLog(metadata)

	a.persist(userID, action, entityType, entityID,
		fmt.Sprintf("financial_transaction %s %s success=%v", a.masker.MaskPaymentAmount(amount), currency, success))

	a.logger.Security(action, success, map[string]any{
		"audit_event":   "financial_transaction",
		"user_id":       userID,
		"entity_type":   entityType,
		"entity_id":     entityID,
		"amount_masked": a.masker.MaskPaymentAmount(amount),
		"currency":      currency,
		"metadata":      sanitized,
	})
}

// LogPermissionChange logs changes to user permissions or roles
func (a *AuditLogger) LogPermissionChange(adminUserID, targetUserID, action, oldValue, newValue string, success bool, reason string) {
	a.persist(adminUserID, action, "user_permissions", targetUserID,
		fmt.Sprintf("permission_change %q → %q success=%v reason=%s", oldValue, newValue, success, reason))
	a.logger.Security(action, success, map[string]any{
		"audit_event":    "permission_change",
		"admin_user_id":  adminUserID,
		"target_user_id": targetUserID,
		"old_value":      oldValue,
		"new_value":      newValue,
		"reason":         reason,
	})
}

// LogDeviceAction logs device approval/blocking actions
func (a *AuditLogger) LogDeviceAction(adminUserID, deviceID, action string, success bool, metadata map[string]any) {
	a.persist(adminUserID, action, "device", deviceID, fmt.Sprintf("device_action success=%v", success))
	a.logger.Security(action, success, map[string]any{
		"audit_event":   "device_action",
		"admin_user_id": adminUserID,
		"device_id":     deviceID,
		"metadata":      metadata,
	})
}

// LogDataExport logs data export operations
func (a *AuditLogger) LogDataExport(userID, exportType, format string, recordCount int, success bool) {
	a.persist(userID, "data_export", exportType, "",
		fmt.Sprintf("export format=%s records=%d success=%v", format, recordCount, success))
	a.logger.Security("data_export", success, map[string]any{
		"audit_event":  "data_export",
		"user_id":      userID,
		"export_type":  exportType,
		"format":       format,
		"record_count": recordCount,
	})
}

// =============================================================================
// 5. RATE LIMITING
// =============================================================================

// RateLimiter is the keyed token-bucket limiter, promoted to
// pkg/infra/ratelimit (Wave 4 B.2). The alias keeps call sites unchanged.
type RateLimiter = ratelimit.Limiter

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter() *RateLimiter {
	return ratelimit.New()
}

// RateLimitConfig defines rate limits for various operations
var RateLimitConfig = struct {
	LoginAttemptsPerMinute    int
	APICallsPerMinute         int
	ReportGenerationPerMinute int
}{
	LoginAttemptsPerMinute:    5,  // Max 5 login attempts per minute
	APICallsPerMinute:         60, // Max 60 API calls per minute per user
	ReportGenerationPerMinute: 3,  // Max 3 reports per minute per user
}

// Global instances (initialized in app startup)
var (
	GlobalValidator   *InputValidator
	GlobalMasker      *DataMasker
	GlobalAuditLogger *AuditLogger
	GlobalRateLimiter *RateLimiter
)

// InitSecurityEnhancements initializes all security components
func InitSecurityEnhancements(logger *Logger) {
	GlobalValidator = NewInputValidator()
	GlobalMasker = NewDataMasker()
	GlobalAuditLogger = NewAuditLogger(logger)
	GlobalRateLimiter = NewRateLimiter()
}
