// ═══════════════════════════════════════════════════════════════════════════
// AUTH SESSION TESTS - Verify Token Expiration & Session Management
//
// Run with: go test -v -run TestAuthSession
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"os"
	"testing"
	"time"

	sqlite "github.com/ncruces/go-sqlite3/gormlite"
	"gorm.io/gorm"
)

// TestSessionCreation verifies session is created with correct expiration
func TestSessionCreation(t *testing.T) {
	// Create in-memory test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Migrate schema
	if err := db.AutoMigrate(&UserSession{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test app
	app := &App{db: db}
	authManager := NewAuthManager(app)

	// Create test token
	now := time.Now()
	token := &TokenResponse{
		AccessToken:  "test_access_token",
		RefreshToken: "test_refresh_token",
		ExpiresAt:    now.Add(24 * time.Hour),
	}

	// Create session
	err = authManager.CreateSession("test_user_123", token)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session in database
	var session UserSession
	err = db.Where("token = ?", hashToken(token.AccessToken)).First(&session).Error
	if err != nil {
		t.Fatalf("Session not found in database: %v", err)
	}

	// Verify fields
	if session.UserID != "test_user_123" {
		t.Errorf("Expected UserID = test_user_123, got %s", session.UserID)
	}

	if session.Token != hashToken(token.AccessToken) {
		t.Errorf("Expected Token hash = %s, got %s", hashToken(token.AccessToken), session.Token)
	}

	if session.RefreshToken != hashToken(token.RefreshToken) {
		t.Errorf("Expected RefreshToken hash = %s, got %s", hashToken(token.RefreshToken), session.RefreshToken)
	}

	if !session.IsActive {
		t.Error("Expected IsActive = true")
	}

	if session.InvalidatedAt != nil {
		t.Error("Expected InvalidatedAt = nil for new session")
	}

	// Check expiration timestamps are set correctly
	if session.AccessTokenExpiry.Before(now) {
		t.Error("AccessTokenExpiry should be in the future")
	}

	if session.RefreshTokenExpiry.Before(now) {
		t.Error("RefreshTokenExpiry should be in the future")
	}

	t.Logf("✅ Session created successfully with ID: %s", session.ID)
}

// TestSessionValidation verifies session validation logic
func TestSessionValidation(t *testing.T) {
	// Create in-memory test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Migrate schema
	if err := db.AutoMigrate(&UserSession{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test app
	app := &App{db: db}
	authManager := NewAuthManager(app)

	now := time.Now()

	// Test Case 1: Valid session
	validToken := &TokenResponse{
		AccessToken:  "valid_token",
		RefreshToken: "valid_refresh",
		ExpiresAt:    now.Add(1 * time.Hour),
	}
	authManager.CreateSession("user1", validToken)

	session, err := authManager.ValidateSession("valid_token")
	if err != nil {
		t.Errorf("Valid session should pass validation: %v", err)
	}
	if session == nil {
		t.Error("ValidateSession should return session object")
	}

	// Test Case 2: Expired access token
	expiredToken := &TokenResponse{
		AccessToken:  "expired_token",
		RefreshToken: "expired_refresh",
		ExpiresAt:    now.Add(-1 * time.Hour), // Expired 1 hour ago
	}
	authManager.CreateSession("user2", expiredToken)

	_, err = authManager.ValidateSession("expired_token")
	if err == nil {
		t.Error("Expired token should fail validation")
	}

	// Test Case 3: Non-existent token
	_, err = authManager.ValidateSession("nonexistent_token")
	if err == nil {
		t.Error("Non-existent token should fail validation")
	}

	t.Log("✅ Session validation tests passed")
}

// TestSessionInvalidation verifies logout invalidates sessions
func TestSessionInvalidation(t *testing.T) {
	// Create in-memory test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Migrate schema
	if err := db.AutoMigrate(&UserSession{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test app
	app := &App{db: db}
	authManager := NewAuthManager(app)

	// Create session
	now := time.Now()
	token := &TokenResponse{
		AccessToken:  "test_token",
		RefreshToken: "test_refresh",
		ExpiresAt:    now.Add(1 * time.Hour),
	}
	authManager.CreateSession("test_user", token)

	// Get session ID
	var session UserSession
	db.Where("token = ?", hashToken(token.AccessToken)).First(&session)

	// Invalidate session
	err = authManager.InvalidateSession(session.ID, "user_logout")
	if err != nil {
		t.Fatalf("Failed to invalidate session: %v", err)
	}

	// Verify session is inactive
	db.Where("id = ?", session.ID).First(&session)

	if session.IsActive {
		t.Error("Session should be inactive after invalidation")
	}

	if session.InvalidatedAt == nil {
		t.Error("InvalidatedAt should be set")
	}

	if session.InvalidatedReason != "user_logout" {
		t.Errorf("Expected InvalidatedReason = user_logout, got %s", session.InvalidatedReason)
	}

	// Try to validate invalidated session
	_, err = authManager.ValidateSession(token.AccessToken)
	if err == nil {
		t.Error("Invalidated session should fail validation")
	}

	t.Log("✅ Session invalidation tests passed")
}

// TestCleanupExpiredSessions verifies cleanup removes old sessions
func TestCleanupExpiredSessions(t *testing.T) {
	// Create in-memory test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Migrate schema
	if err := db.AutoMigrate(&UserSession{}); err != nil {
		t.Fatalf("Failed to migrate schema: %v", err)
	}

	// Create test app
	app := &App{db: db}
	authManager := NewAuthManager(app)

	now := time.Now()

	// Create old expired session (8 days ago)
	oldTime := now.Add(-8 * 24 * time.Hour)
	oldSession := UserSession{}
	oldSession.ID = "old_session_id" // Set ID explicitly
	oldSession.UserID = "old_user"
	oldSession.Token = "old_token"
	oldSession.RefreshToken = "old_refresh"
	oldSession.AccessTokenExpiry = oldTime
	oldSession.RefreshTokenExpiry = oldTime
	oldSession.LastActivityAt = oldTime
	oldSession.IsActive = false
	oldSession.InvalidatedAt = &oldTime
	oldSession.InvalidatedReason = "test_old"
	oldSession.CreatedAt = oldTime
	oldSession.UpdatedAt = oldTime

	// Use raw SQL to bypass BeforeCreate hook
	db.Exec(`INSERT INTO user_sessions (id, user_id, token, refresh_token, access_token_expiry, refresh_token_expiry,
		last_activity_at, is_active, invalidated_at, invalidated_reason, created_at, updated_at, version, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		oldSession.ID, oldSession.UserID, oldSession.Token, oldSession.RefreshToken,
		oldSession.AccessTokenExpiry, oldSession.RefreshTokenExpiry, oldSession.LastActivityAt,
		oldSession.IsActive, oldSession.InvalidatedAt, oldSession.InvalidatedReason,
		oldSession.CreatedAt, oldSession.UpdatedAt, 1, "")

	// Create recent expired session (5 days ago - within 7-day retention)
	// BUT refresh token is still valid (expires in future)
	recentTime := now.Add(-5 * 24 * time.Hour)
	recentSession := UserSession{}
	recentSession.ID = "recent_session_id"
	recentSession.UserID = "recent_user"
	recentSession.Token = "recent_token"
	recentSession.RefreshToken = "recent_refresh"
	recentSession.AccessTokenExpiry = recentTime
	recentSession.RefreshTokenExpiry = now.Add(25 * 24 * time.Hour) // Still valid!
	recentSession.LastActivityAt = recentTime
	recentSession.IsActive = false
	recentSession.InvalidatedAt = &recentTime
	recentSession.InvalidatedReason = "test_recent"
	recentSession.CreatedAt = recentTime
	recentSession.UpdatedAt = recentTime

	db.Exec(`INSERT INTO user_sessions (id, user_id, token, refresh_token, access_token_expiry, refresh_token_expiry,
		last_activity_at, is_active, invalidated_at, invalidated_reason, created_at, updated_at, version, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		recentSession.ID, recentSession.UserID, recentSession.Token, recentSession.RefreshToken,
		recentSession.AccessTokenExpiry, recentSession.RefreshTokenExpiry, recentSession.LastActivityAt,
		recentSession.IsActive, recentSession.InvalidatedAt, recentSession.InvalidatedReason,
		recentSession.CreatedAt, recentSession.UpdatedAt, 1, "")

	// Create active session
	activeToken := &TokenResponse{
		AccessToken:  "active_token",
		RefreshToken: "active_refresh",
		ExpiresAt:    now.Add(1 * time.Hour),
	}
	authManager.CreateSession("active_user", activeToken)

	// Count before cleanup
	var countBefore int64
	db.Model(&UserSession{}).Count(&countBefore)
	t.Logf("Sessions before cleanup: %d", countBefore)

	// Run cleanup
	err = authManager.CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Count after cleanup
	var countAfter int64
	db.Model(&UserSession{}).Count(&countAfter)
	t.Logf("Sessions after cleanup: %d", countAfter)

	// Old session should be deleted (8 days old)
	var oldCount int64
	db.Model(&UserSession{}).Where("user_id = ?", "old_user").Count(&oldCount)
	if oldCount != 0 {
		t.Error("Old expired session should be deleted")
	}

	// Recent session should still exist (2 days old, within 7-day retention)
	var recentCount int64
	db.Model(&UserSession{}).Where("user_id = ?", "recent_user").Count(&recentCount)
	if recentCount != 1 {
		t.Error("Recent expired session should be retained")
	}

	// Active session should still exist
	var activeCount int64
	db.Model(&UserSession{}).Where("user_id = ?", "active_user").Count(&activeCount)
	if activeCount != 1 {
		t.Error("Active session should be retained")
	}

	t.Log("✅ Cleanup tests passed")
}

// TestTokenExpiration verifies IsTokenExpired logic
func TestTokenExpiration(t *testing.T) {
	// Create test app
	app := &App{}
	authManager := NewAuthManager(app)

	now := time.Now()

	// Test Case 1: Token expires in 10 minutes (should NOT be expired with 5 min buffer)
	token1 := &TokenResponse{
		ExpiresAt: now.Add(10 * time.Minute),
	}
	if authManager.IsTokenExpired(token1, 5) {
		t.Error("Token expiring in 10 minutes should not be expired (5 min buffer)")
	}

	// Test Case 2: Token expires in 3 minutes (SHOULD be expired with 5 min buffer)
	token2 := &TokenResponse{
		ExpiresAt: now.Add(3 * time.Minute),
	}
	if !authManager.IsTokenExpired(token2, 5) {
		t.Error("Token expiring in 3 minutes should be expired (5 min buffer)")
	}

	// Test Case 3: Token already expired
	token3 := &TokenResponse{
		ExpiresAt: now.Add(-1 * time.Minute),
	}
	if !authManager.IsTokenExpired(token3, 5) {
		t.Error("Expired token should be detected")
	}

	// Test Case 4: Nil token
	if !authManager.IsTokenExpired(nil, 5) {
		t.Error("Nil token should be considered expired")
	}

	t.Log("✅ Token expiration tests passed")
}

// Run all tests
func TestAuthSession(t *testing.T) {
	// Disable logging during tests
	if testing.Verbose() {
		// Keep logging enabled in verbose mode
	} else {
		// Silence logs in normal mode
		oldLogOutput := os.Stdout
		os.Stdout = nil
		defer func() { os.Stdout = oldLogOutput }()
	}

	t.Run("SessionCreation", TestSessionCreation)
	t.Run("SessionValidation", TestSessionValidation)
	t.Run("SessionInvalidation", TestSessionInvalidation)
	t.Run("CleanupExpiredSessions", TestCleanupExpiredSessions)
	t.Run("TokenExpiration", TestTokenExpiration)
}
