package main

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		shouldError bool
	}{
		{"Valid password", "SecurePass123", false},
		{"Empty password", "", true},
		{"Long password", "ThisIsAVeryLongPasswordWith123Numbers", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hashPassword(tt.password)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if hash == "" {
				t.Error("Hash should not be empty")
			}

			// Verify hash is different from password
			if hash == tt.password {
				t.Error("Hash should not equal plain password")
			}

			// Verify hash starts with bcrypt prefix
			if len(hash) < 20 || hash[:4] != "$2a$" && hash[:4] != "$2b$" {
				t.Errorf("Hash doesn't look like bcrypt: %s", hash)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	password := "TestPass123"
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name        string
		password    string
		shouldMatch bool
	}{
		{"Correct password", password, true},
		{"Wrong password", "WrongPass456", false},
		{"Empty password", "", false},
		{"Similar password", "TestPass124", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyPassword(tt.password, hash)

			if tt.shouldMatch {
				if err != nil {
					t.Errorf("Expected password to match, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("Expected password mismatch, got match")
				}
			}
		})
	}
}

func TestValidatePasswordComplexity(t *testing.T) {
	tests := []struct {
		name        string
		password    string
		shouldError bool
	}{
		{"Valid password", "Password123", false},
		{"Too short", "Pass1", true},
		{"No numbers", "PasswordOnly", true},
		{"No letters", "12345678", true},
		{"Minimum valid", "Pass1234", false},
		{"Very complex", "C0mpl3x!P@ssw0rd#2024", false},
		{"Exactly 8 chars", "Pass1234", false},
		{"7 chars", "Pass123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePasswordComplexity(tt.password)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for password: %s", tt.password)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for password: %s, error: %v", tt.password, err)
				}
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Default length", 16},
		{"Minimum length", 8},
		{"Long password", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := generateSecurePassword(tt.length)
			if err != nil {
				t.Errorf("Failed to generate password: %v", err)
				return
			}

			if len(password) != tt.length {
				t.Errorf("Expected length %d, got %d", tt.length, len(password))
			}

			// Verify it meets complexity requirements
			if err := validatePasswordComplexity(password); err != nil {
				t.Errorf("Generated password doesn't meet complexity: %v", err)
			}

			// Generate another and verify they're different (randomness check)
			password2, err := generateSecurePassword(tt.length)
			if err != nil {
				t.Errorf("Failed to generate second password: %v", err)
				return
			}

			if password == password2 {
				t.Error("Two generated passwords should not be identical")
			}
		})
	}
}

func TestPasswordEndToEnd(t *testing.T) {
	// Generate a secure password
	password, err := generateSecurePassword(16)
	if err != nil {
		t.Fatalf("Failed to generate password: %v", err)
	}

	// Hash it
	hash, err := hashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Verify correct password
	if err := verifyPassword(password, hash); err != nil {
		t.Errorf("Failed to verify correct password: %v", err)
	}

	// Verify wrong password fails
	if err := verifyPassword("WrongPassword123", hash); err == nil {
		t.Error("Wrong password should not verify")
	}
}

func BenchmarkHashPassword(b *testing.B) {
	password := "BenchmarkPass123"

	for i := 0; i < b.N; i++ {
		_, err := hashPassword(password)
		if err != nil {
			b.Fatalf("Hash failed: %v", err)
		}
	}
}

func BenchmarkVerifyPassword(b *testing.B) {
	password := "BenchmarkPass123"
	hash, _ := hashPassword(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = verifyPassword(password, hash)
	}
}

func BenchmarkGenerateSecurePassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := generateSecurePassword(16)
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}
