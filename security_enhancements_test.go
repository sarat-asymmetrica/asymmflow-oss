package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// P3 SECURITY ENHANCEMENTS TESTS - REAL IMPLEMENTATIONS
// =============================================================================
// This file contains comprehensive tests for P1 security enhancements.
// All tests are fully implemented and verify security-critical validations.
// =============================================================================

// TestInputValidation_StringLength verifies string length validation
func TestInputValidation_StringLength(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should accept valid length strings", func(t *testing.T) {
		err := validator.ValidateString("CustomerName", "Valid Company Ltd", ValidationLimits.CustomerName, true)
		require.NoError(t, err)
	})

	t.Run("should reject strings exceeding max length", func(t *testing.T) {
		longString := strings.Repeat("a", ValidationLimits.CustomerName+1)
		err := validator.ValidateString("CustomerName", longString, ValidationLimits.CustomerName, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
		assert.Contains(t, err.Error(), "CustomerName")
	})

	t.Run("should accept empty strings if not required", func(t *testing.T) {
		err := validator.ValidateString("OptionalField", "", 100, true)
		require.NoError(t, err)
	})

	t.Run("should handle UTF-8 multi-byte characters correctly", func(t *testing.T) {
		arabicText := "شركة البحرين للتجارة" // Multi-byte UTF-8
		err := validator.ValidateString("CustomerName", arabicText, ValidationLimits.CustomerName, true)
		require.NoError(t, err)
	})

	t.Run("should reject strings with null bytes", func(t *testing.T) {
		maliciousString := "Valid\x00Data"
		err := validator.ValidateString("Field", maliciousString, 100, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "null bytes")
	})

	t.Run("should reject invalid UTF-8 sequences", func(t *testing.T) {
		invalidUTF8 := string([]byte{0xFF, 0xFE, 0xFD})
		err := validator.ValidateString("Field", invalidUTF8, 100, true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid UTF-8")
	})

	t.Run("should block dangerous patterns when special chars not allowed", func(t *testing.T) {
		dangerousInputs := []string{
			"<script>alert('xss')</script>",
			"javascript:void(0)",
			"onerror=alert('xss')",
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32",
		}

		for _, input := range dangerousInputs {
			err := validator.ValidateString("Field", input, 1000, false)
			require.Error(t, err, "Should block dangerous pattern: %s", input)
			assert.Contains(t, err.Error(), "potentially dangerous pattern")
		}
	})

	t.Run("should allow special chars when explicitly allowed", func(t *testing.T) {
		legitInput := "Product code: ABC-123 (rev. 2.0)"
		err := validator.ValidateString("Description", legitInput, 1000, true)
		require.NoError(t, err)
	})
}

// TestInputValidation_EmailFormat verifies email format validation
func TestInputValidation_EmailFormat(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should accept valid email addresses", func(t *testing.T) {
		validEmails := []string{
			"user@example.com",
			"test.user@company.co.uk",
			"admin+tag@domain.io",
			"user123@sub.domain.com",
		}

		for _, email := range validEmails {
			err := validator.ValidateEmail(email)
			require.NoError(t, err, "Should accept valid email: %s", email)
		}
	})

	t.Run("should reject invalid email formats", func(t *testing.T) {
		invalidEmails := []string{
			"not-an-email",
			"@example.com",
			"user@",
			"user @example.com", // Space in email
		}

		for _, email := range invalidEmails {
			err := validator.ValidateEmail(email)
			require.Error(t, err, "Should reject invalid email: %s", email)
			assert.Contains(t, err.Error(), "invalid email format")
		}
	})

	t.Run("should reject emails exceeding max length", func(t *testing.T) {
		longEmail := strings.Repeat("a", ValidationLimits.Email) + "@example.com"
		err := validator.ValidateEmail(longEmail)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("should accept empty email as optional field", func(t *testing.T) {
		err := validator.ValidateEmail("")
		require.NoError(t, err, "Empty email should be allowed as optional field")
	})
}

// TestRateLimiter_LoginAttempts verifies login rate limiting
func TestRateLimiter_LoginAttempts(t *testing.T) {
	t.Run("should allow requests under rate limit", func(t *testing.T) {
		limiter := NewRateLimiter()
		identifier := "test-login-192.168.1.1"

		// Should allow 5 requests (standard login limit)
		for i := 0; i < 5; i++ {
			allowed := limiter.Allow(identifier, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000) // 12 seconds refill
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}
	})

	t.Run("should block requests exceeding rate limit", func(t *testing.T) {
		limiter := NewRateLimiter()
		identifier := "test-blocked-192.168.1.1"

		// Exhaust the limit (5 allowed)
		for i := 0; i < 5; i++ {
			allowed := limiter.Allow(identifier, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000)
			assert.True(t, allowed, "Request %d should be allowed", i+1)
		}

		// 6th attempt should be blocked
		allowed := limiter.Allow(identifier, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000)
		assert.False(t, allowed, "6th request should be rate limited")
	})

	t.Run("should track different identifiers separately", func(t *testing.T) {
		limiter := NewRateLimiter()
		ip1 := "test-ip1-192.168.1.1"
		ip2 := "test-ip2-192.168.1.2"

		// Exhaust rate limit for ip1
		for i := 0; i < 6; i++ {
			limiter.Allow(ip1, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000)
		}

		// ip2 should still be allowed (separate bucket)
		allowed := limiter.Allow(ip2, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000)
		assert.True(t, allowed, "Different identifier should have separate rate limit")

		// ip1 should still be blocked
		allowed = limiter.Allow(ip1, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000)
		assert.False(t, allowed, "Original identifier should still be rate limited")
	})

	t.Run("should track different limits independently", func(t *testing.T) {
		limiter := NewRateLimiter()

		// Different keys for login vs API
		loginKey := "user123-login"
		apiKey := "user123-api"

		// Exhaust login rate limit (5 attempts)
		for i := 0; i < 6; i++ {
			limiter.Allow(loginKey, RateLimitConfig.LoginAttemptsPerMinute, 12*1000000000)
		}

		// API calls should still be allowed (60 per minute)
		allowed := limiter.Allow(apiKey, RateLimitConfig.APICallsPerMinute, 1*1000000000)
		assert.True(t, allowed, "Different action should have independent limit")
	})
}

// TestDataMasker_PaymentAmount verifies sensitive data masking
func TestDataMasker_PaymentAmount(t *testing.T) {
	masker := NewDataMasker()

	t.Run("should show small amounts in full", func(t *testing.T) {
		amount := 45.678
		masked := masker.MaskPaymentAmount(amount)
		assert.Equal(t, "45.678 BHD", masked, "Small amounts (<100) should be shown in full")
	})

	t.Run("should mask large amounts showing last 3 digits", func(t *testing.T) {
		amount := 1234.567
		masked := masker.MaskPaymentAmount(amount)
		assert.Contains(t, masked, "***", "Large amounts should be masked")
		assert.Contains(t, masked, ".567", "Should show last 3 decimal digits")
		assert.Contains(t, masked, "BHD", "Should include currency")
		assert.NotContains(t, masked, "1234", "Should not reveal full amount")
	})

	t.Run("should mask very large amounts", func(t *testing.T) {
		amount := 123456.789
		masked := masker.MaskPaymentAmount(amount)
		assert.Contains(t, masked, "***", "Very large amounts should be masked")
		assert.Contains(t, masked, "BHD", "Should include currency")
	})
}

// TestDataMasker_Email verifies email masking
func TestDataMasker_Email(t *testing.T) {
	masker := NewDataMasker()

	t.Run("should mask email showing first 2 chars and domain", func(t *testing.T) {
		email := "john.doe@company.com"
		masked := masker.MaskEmail(email)
		assert.Equal(t, "jo***@company.com", masked)
	})

	t.Run("should handle short emails", func(t *testing.T) {
		email := "a@example.com"
		masked := masker.MaskEmail(email)
		assert.Equal(t, "**@example.com", masked, "Short emails should mask local part")
	})

	t.Run("should handle empty email", func(t *testing.T) {
		email := ""
		masked := masker.MaskEmail(email)
		assert.Equal(t, "", masked, "Empty email should return empty")
	})

	t.Run("should handle invalid email format safely", func(t *testing.T) {
		email := "not-an-email"
		masked := masker.MaskEmail(email)
		assert.Equal(t, "***@***", masked, "Invalid format should be safely masked")
	})
}

// TestDataMasker_Phone verifies phone number masking
func TestDataMasker_Phone(t *testing.T) {
	masker := NewDataMasker()

	t.Run("should mask phone showing last 4 digits", func(t *testing.T) {
		phone := "+973-1234-5678"
		masked := masker.MaskPhone(phone)
		assert.Contains(t, masked, "5678", "Should show last 4 digits")
		assert.Contains(t, masked, "***", "Should mask other digits")
	})

	t.Run("should handle short phone numbers", func(t *testing.T) {
		phone := "1234"
		masked := masker.MaskPhone(phone)
		assert.Equal(t, "****", masked, "Short phones should be fully masked")
	})

	t.Run("should handle empty phone", func(t *testing.T) {
		phone := ""
		masked := masker.MaskPhone(phone)
		assert.Equal(t, "", masked, "Empty phone should return empty")
	})

	t.Run("should extract digits from formatted phone", func(t *testing.T) {
		phone := "(973) 1234-5678"
		masked := masker.MaskPhone(phone)
		assert.Contains(t, masked, "5678", "Should extract and show last 4 digits")
	})
}

// TestDataMasker_BankAccount verifies bank account masking
func TestDataMasker_BankAccount(t *testing.T) {
	masker := NewDataMasker()

	t.Run("should mask account showing last 4 digits", func(t *testing.T) {
		account := "1234567890123456"
		masked := masker.MaskBankAccount(account)
		assert.Contains(t, masked, "3456", "Should show last 4 digits")
		assert.Contains(t, masked, "************", "Should mask other digits")
	})

	t.Run("should handle short account numbers", func(t *testing.T) {
		account := "123"
		masked := masker.MaskBankAccount(account)
		assert.Equal(t, "****", masked, "Short accounts should be fully masked")
	})
}

// TestDataMasker_IBAN verifies IBAN masking
func TestDataMasker_IBAN(t *testing.T) {
	masker := NewDataMasker()

	t.Run("should mask IBAN showing country code and last 4 digits", func(t *testing.T) {
		iban := "BH67BMAG00001299123456"
		masked := masker.MaskIBAN(iban)
		assert.Contains(t, masked, "BH", "Should show country code")
		assert.Contains(t, masked, "3456", "Should show last 4 digits")
		assert.Contains(t, masked, "**************", "Should mask middle")
	})

	t.Run("should handle short IBAN", func(t *testing.T) {
		iban := "BH123"
		masked := masker.MaskIBAN(iban)
		assert.Equal(t, "**************", masked, "Short IBAN should be fully masked")
	})
}

// TestDataMasker_SanitizeForLog verifies comprehensive log sanitization
func TestDataMasker_SanitizeForLog(t *testing.T) {
	masker := NewDataMasker()

	t.Run("should mask password fields", func(t *testing.T) {
		fields := map[string]any{
			"username": "john",
			"password": "secret123",
		}
		sanitized := masker.SanitizeForLog(fields)
		assert.Equal(t, "***REDACTED***", sanitized["password"])
		assert.Equal(t, "john", sanitized["username"])
	})

	t.Run("should mask email fields", func(t *testing.T) {
		fields := map[string]any{
			"user_email": "john.doe@company.com",
			"name":       "John Doe",
		}
		sanitized := masker.SanitizeForLog(fields)
		assert.Equal(t, "jo***@company.com", sanitized["user_email"])
		assert.Equal(t, "John Doe", sanitized["name"])
	})

	t.Run("should mask phone fields", func(t *testing.T) {
		fields := map[string]any{
			"phone":   "+973-1234-5678",
			"company": "Acme Instrumentation",
		}
		sanitized := masker.SanitizeForLog(fields)
		assert.Contains(t, sanitized["phone"].(string), "5678")
		assert.Equal(t, "Acme Instrumentation", sanitized["company"])
	})

	t.Run("should handle multiple sensitive fields", func(t *testing.T) {
		fields := map[string]any{
			"password":   "secret",
			"email":      "test@example.com",
			"phone":      "1234567890",
			"safe_field": "visible",
		}
		sanitized := masker.SanitizeForLog(fields)
		assert.Equal(t, "***REDACTED***", sanitized["password"])
		assert.Contains(t, sanitized["email"].(string), "***")
		assert.Contains(t, sanitized["phone"].(string), "***")
		assert.Equal(t, "visible", sanitized["safe_field"])
	})
}

// TestInputValidation_PhoneFormat verifies phone number validation
func TestInputValidation_PhoneFormat(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should accept valid Bahrain phone numbers", func(t *testing.T) {
		validPhones := []string{
			"17123456",       // Landline
			"39123456",       // Mobile
			"+973-17-123456", // With country code
			"+973 39 123456", // With spaces
		}

		for _, phone := range validPhones {
			err := validator.ValidatePhone(phone)
			require.NoError(t, err, "Should accept valid Bahrain phone: %s", phone)
		}
	})

	t.Run("should accept international phone numbers", func(t *testing.T) {
		validPhones := []string{
			"+1-555-123-4567",  // US
			"+44-20-7123-4567", // UK
			"+971-4-123-4567",  // UAE
			"(973) 1234-5678",  // With parentheses
		}

		for _, phone := range validPhones {
			err := validator.ValidatePhone(phone)
			require.NoError(t, err, "Should accept international phone: %s", phone)
		}
	})

	t.Run("should reject invalid phone formats", func(t *testing.T) {
		invalidPhones := []string{
			"abc-def-ghij", // Letters
			"phone123",     // Contains letters
			"hello@world",  // Invalid chars
		}

		for _, phone := range invalidPhones {
			err := validator.ValidatePhone(phone)
			require.Error(t, err, "Should reject invalid phone: %s", phone)
			assert.Contains(t, err.Error(), "invalid characters")
		}
	})

	t.Run("should reject phone numbers exceeding max length", func(t *testing.T) {
		longPhone := strings.Repeat("1234567890", 6) // 60 chars (limit is 50)
		err := validator.ValidatePhone(longPhone)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("should accept empty phone as optional field", func(t *testing.T) {
		err := validator.ValidatePhone("")
		require.NoError(t, err, "Empty phone should be allowed as optional field")
	})
}

// TestInputValidation_PathTraversal verifies path traversal protection
func TestInputValidation_PathTraversal(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should block path traversal attempts", func(t *testing.T) {
		maliciousPaths := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32",
			"./../../secret.txt",
		}

		for _, path := range maliciousPaths {
			_, err := validator.SanitizeFilePath(path)
			require.Error(t, err, "Should block path traversal: %s", path)
			assert.Contains(t, err.Error(), "path traversal")
		}
	})

	t.Run("should block absolute paths", func(t *testing.T) {
		absolutePaths := []string{
			"C:\\Windows\\System32",
			"C:\\Program Files",
			"D:\\Data",
		}

		for _, path := range absolutePaths {
			_, err := validator.SanitizeFilePath(path)
			require.Error(t, err, "Should block absolute path: %s", path)
			assert.Contains(t, err.Error(), "absolute paths not allowed")
		}
	})

	t.Run("should accept safe relative paths", func(t *testing.T) {
		safePaths := []string{
			"reports/customer_report.pdf",
			"exports/data.csv",
			"invoices/2026/INV-001.pdf",
		}

		for _, path := range safePaths {
			cleaned, err := validator.SanitizeFilePath(path)
			require.NoError(t, err, "Should accept safe path: %s", path)
			assert.NotEmpty(t, cleaned)
		}
	})

	t.Run("should reject paths exceeding max length", func(t *testing.T) {
		longPath := strings.Repeat("a/", 150) + "file.txt"
		_, err := validator.SanitizeFilePath(longPath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})
}

// TestInputValidation_CustomerInput verifies customer data validation
func TestInputValidation_CustomerInput(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should accept valid customer data", func(t *testing.T) {
		customer := &CustomerMaster{
			BusinessName: "NPC - National Petroleum Co. Company",
			ShortCode:    "NPC",
			TRN:          "BH123456789",
		}

		err := validator.ValidateCustomerInput(customer)
		require.NoError(t, err)
	})

	t.Run("should reject customer name exceeding max length", func(t *testing.T) {
		customer := &CustomerMaster{
			BusinessName: strings.Repeat("Very Long Company Name ", 20), // >200 chars
			ShortCode:    "TEST",
		}

		err := validator.ValidateCustomerInput(customer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum length")
	})

	t.Run("should reject dangerous patterns in customer name", func(t *testing.T) {
		customer := &CustomerMaster{
			BusinessName: "<script>alert('xss')</script>",
			ShortCode:    "TEST",
		}

		err := validator.ValidateCustomerInput(customer)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "potentially dangerous pattern")
	})
}

// TestInputValidation_SupplierInput verifies supplier data validation
func TestInputValidation_SupplierInput(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should accept valid supplier data", func(t *testing.T) {
		supplier := &SupplierMaster{
			SupplierName: "Rhine Instruments",
			Email:        "sales@rhine-instruments.example",
			Phone:        "+41-61-715-7777",
			Address:      "Kägenstrasse 2, 4153 Reinach, Switzerland",
		}

		err := validator.ValidateSupplierInput(supplier)
		require.NoError(t, err)
	})

	t.Run("should reject invalid email in supplier data", func(t *testing.T) {
		supplier := &SupplierMaster{
			SupplierName: "Test Supplier",
			Email:        "not-an-email",
			Phone:        "+973-1234-5678",
		}

		err := validator.ValidateSupplierInput(supplier)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email format")
	})

	t.Run("should reject invalid phone in supplier data", func(t *testing.T) {
		supplier := &SupplierMaster{
			SupplierName: "Test Supplier",
			Email:        "test@example.com",
			Phone:        "abc-def-ghij",
		}

		err := validator.ValidateSupplierInput(supplier)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid characters")
	})
}

// TestInputValidation_UserInput verifies user account validation
func TestInputValidation_UserInput(t *testing.T) {
	validator := NewInputValidator()

	t.Run("should accept valid user data", func(t *testing.T) {
		err := validator.ValidateUserInput("johndoe", "SecurePass123!", "John Doe", "john@example.com")
		require.NoError(t, err)
	})

	t.Run("should reject short password", func(t *testing.T) {
		err := validator.ValidateUserInput("johndoe", "short", "John Doe", "john@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be at least 8 characters")
	})

	t.Run("should reject invalid email", func(t *testing.T) {
		err := validator.ValidateUserInput("johndoe", "SecurePass123!", "John Doe", "not-an-email")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email format")
	})

	t.Run("should reject username with dangerous patterns", func(t *testing.T) {
		err := validator.ValidateUserInput("../../admin", "SecurePass123!", "John Doe", "john@example.com")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "potentially dangerous pattern")
	})
}
