// ═══════════════════════════════════════════════════════════════════════════
// AUTH SESSION MANAGER - Token Expiration & Session Management
//
// MISSION: Secure session lifecycle management with token expiration
//
// FEATURES:
//   1. Session expiration validation (24h access, 30d refresh)
//   2. Token refresh before expiration
//   3. Session invalidation on logout
//   4. Periodic cleanup of expired sessions
//   5. Activity tracking and timeout
//
// SECURITY:
//   - Stolen tokens expire automatically
//   - No indefinite token validity
//   - Database-backed session tracking
//   - Audit trail via InvalidatedAt/Reason
//
// Built with SECURITY × SIMPLICITY × WAILS INTEGRATION 🔐⚡💎
// Day 197 - Wave 2 Agent 2 - Session Expiration
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"
)

// hashToken creates a SHA-256 hash of a token for safe storage.
// The actual token value is never stored in the database - only the hash.
// Lookups use the same hash function to match.
func hashToken(token string) string {
	if token == "" {
		return ""
	}
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// ═══════════════════════════════════════════════════════════════════════════
// CONSTANTS - Session Expiration Defaults
// ═══════════════════════════════════════════════════════════════════════════

const (
	// Access token expires after 24 hours (Microsoft default: 1 hour, we extend for desktop app)
	DefaultAccessTokenLifetime = 24 * time.Hour

	// Refresh token expires after 30 days (Microsoft default: 90 days)
	DefaultRefreshTokenLifetime = 30 * 24 * time.Hour

	// Session timeout after 24 hours of inactivity
	DefaultSessionTimeout = 24 * time.Hour

	// Cleanup expired sessions every 6 hours
	SessionCleanupInterval = 6 * time.Hour
)

// ═══════════════════════════════════════════════════════════════════════════
// SESSION CREATION & STORAGE
// ═══════════════════════════════════════════════════════════════════════════

// CreateSession stores OAuth tokens with expiration in database
func (am *AuthManager) CreateSession(userID string, token *TokenResponse) error {
	if am.app == nil || am.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()

	// SECURITY FIX: Store hashed tokens instead of plaintext
	// The actual tokens remain in memory only; DB stores hashes for lookup/invalidation
	session := UserSession{
		UserID:             userID,
		Token:              hashToken(token.AccessToken),
		RefreshToken:       hashToken(token.RefreshToken),
		AccessTokenExpiry:  token.ExpiresAt,
		RefreshTokenExpiry: now.Add(DefaultRefreshTokenLifetime),
		LastActivityAt:     now,
		IsActive:           true,
	}

	// Create UUID via BeforeCreate hook
	if err := am.app.db.Create(&session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	log.Printf("✅ Session created for user %s (expires: %s)", userID, token.ExpiresAt.Format(time.RFC3339))
	return nil
}

// UpdateSessionActivity updates last activity timestamp
func (am *AuthManager) UpdateSessionActivity(accessToken string) error {
	if am.app == nil || am.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// SECURITY FIX: Compare against hashed token
	result := am.app.db.Model(&UserSession{}).
		Where("token = ? AND is_active = ?", hashToken(accessToken), true).
		Update("last_activity_at", time.Now())

	if result.Error != nil {
		return fmt.Errorf("failed to update session activity: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("session not found or inactive")
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION VALIDATION
// ═══════════════════════════════════════════════════════════════════════════

// ValidateSession checks if session exists and is not expired
func (am *AuthManager) ValidateSession(accessToken string) (*UserSession, error) {
	if am.app == nil || am.app.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var session UserSession
	// SECURITY FIX: Lookup by hashed token
	if err := am.app.db.Where("token = ? AND is_active = ?", hashToken(accessToken), true).First(&session).Error; err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	now := time.Now()

	// Check if access token expired
	if now.After(session.AccessTokenExpiry) {
		return nil, fmt.Errorf("access token expired at %s", session.AccessTokenExpiry.Format(time.RFC3339))
	}

	// Check if refresh token expired
	if now.After(session.RefreshTokenExpiry) {
		// Invalidate session - refresh token expired
		am.InvalidateSession(session.ID, "refresh_token_expired")
		return nil, fmt.Errorf("refresh token expired at %s", session.RefreshTokenExpiry.Format(time.RFC3339))
	}

	// Check session timeout (inactivity)
	if now.Sub(session.LastActivityAt) > DefaultSessionTimeout {
		am.InvalidateSession(session.ID, "session_timeout")
		return nil, fmt.Errorf("session timed out after %v of inactivity", DefaultSessionTimeout)
	}

	// Session valid - update activity
	am.UpdateSessionActivity(accessToken)

	return &session, nil
}

// IsTokenExpired checks if token needs refresh
func (am *AuthManager) IsTokenExpired(token *TokenResponse, bufferMinutes int) bool {
	if token == nil {
		return true
	}

	buffer := time.Duration(bufferMinutes) * time.Minute
	return time.Now().Add(buffer).After(token.ExpiresAt)
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION INVALIDATION
// ═══════════════════════════════════════════════════════════════════════════

// InvalidateSession marks session as inactive (logout, timeout, etc.)
func (am *AuthManager) InvalidateSession(sessionID string, reason string) error {
	if am.app == nil || am.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	result := am.app.db.Model(&UserSession{}).
		Where("id = ?", sessionID).
		Updates(map[string]any{
			"is_active":          false,
			"invalidated_at":     &now,
			"invalidated_reason": reason,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to invalidate session: %w", result.Error)
	}

	log.Printf("🔒 Session %s invalidated (reason: %s)", sessionID, reason)
	return nil
}

// InvalidateSessionByToken invalidates session by access token
func (am *AuthManager) InvalidateSessionByToken(accessToken string, reason string) error {
	if am.app == nil || am.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	// SECURITY FIX: Lookup by hashed token
	var session UserSession
	if err := am.app.db.Where("token = ?", hashToken(accessToken)).First(&session).Error; err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	return am.InvalidateSession(session.ID, reason)
}

// InvalidateAllUserSessions invalidates all sessions for a user (e.g., password reset)
func (am *AuthManager) InvalidateAllUserSessions(userID string, reason string) error {
	if am.app == nil || am.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	result := am.app.db.Model(&UserSession{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Updates(map[string]any{
			"is_active":          false,
			"invalidated_at":     &now,
			"invalidated_reason": reason,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to invalidate user sessions: %w", result.Error)
	}

	log.Printf("🔒 Invalidated %d sessions for user %s (reason: %s)", result.RowsAffected, userID, reason)
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// SESSION CLEANUP
// ═══════════════════════════════════════════════════════════════════════════

// CleanupExpiredSessions removes old expired sessions (periodic background job)
func (am *AuthManager) CleanupExpiredSessions() error {
	if am.app == nil || am.app.db == nil {
		return fmt.Errorf("database not initialized")
	}

	now := time.Now()
	cutoffDate := now.Add(-7 * 24 * time.Hour) // Keep invalidated sessions for 7 days

	// Find sessions to delete (inactive AND either old invalidation OR expired refresh token)
	var sessionsToDelete []UserSession
	err := am.app.db.Unscoped().
		Where("is_active = ? AND (invalidated_at < ? OR refresh_token_expiry < ?)",
			false, cutoffDate, now).
		Find(&sessionsToDelete).Error

	if err != nil {
		return fmt.Errorf("failed to find expired sessions: %w", err)
	}

	if len(sessionsToDelete) == 0 {
		return nil // Nothing to cleanup
	}

	// Hard delete these sessions
	result := am.app.db.Unscoped().Delete(&sessionsToDelete)

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log.Printf("🧹 Cleaned up %d expired sessions", result.RowsAffected)
	}

	return nil
}

// StartSessionCleanupWorker starts background worker for periodic cleanup
func (am *AuthManager) StartSessionCleanupWorker() {
	go func() {
		ticker := time.NewTicker(SessionCleanupInterval)
		defer ticker.Stop()

		log.Printf("🔄 Session cleanup worker started (interval: %v)", SessionCleanupInterval)

		for range ticker.C {
			if err := am.CleanupExpiredSessions(); err != nil {
				log.Printf("⚠️ Session cleanup failed: %v", err)
			}
		}
	}()
}

// ═══════════════════════════════════════════════════════════════════════════
// WAILS API - Enhanced Methods with Session Validation
// ═══════════════════════════════════════════════════════════════════════════

// GetAccessTokenWithValidation returns valid access token with session check
func (a *App) GetAccessTokenWithValidation() (string, error) {
	if a.authManager == nil || a.authManager.Token == nil {
		return "", fmt.Errorf("not authenticated")
	}

	// Check session validity in database
	session, err := a.authManager.ValidateSession(a.authManager.Token.AccessToken)
	if err != nil {
		// Session invalid - clear in-memory state
		a.authManager.mu.Lock()
		a.authManager.Token = nil
		a.authManager.Profile = nil
		a.authManager.mu.Unlock()
		return "", fmt.Errorf("session invalid: %w", err)
	}

	// Check if token needs refresh (5 min buffer)
	if a.authManager.IsTokenExpired(a.authManager.Token, 5) {
		log.Println("🔄 Token expiring soon, refreshing...")
		if err := a.RefreshAuthWithSession(session); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	return a.authManager.Token.AccessToken, nil
}

// RefreshAuthWithSession refreshes token and updates session
func (a *App) RefreshAuthWithSession(session *UserSession) error {
	if a.authManager == nil {
		return fmt.Errorf("auth manager not initialized")
	}

	// Use refresh token from in-memory state (DB only stores hash)
	a.authManager.mu.RLock()
	refreshToken := ""
	if a.authManager.Token != nil {
		refreshToken = a.authManager.Token.RefreshToken
	}
	a.authManager.mu.RUnlock()
	if refreshToken == "" {
		return fmt.Errorf("no refresh token available in memory")
	}
	newToken, err := a.authManager.refreshToken(refreshToken)
	if err != nil {
		// Refresh failed - invalidate session
		a.authManager.InvalidateSession(session.ID, "refresh_failed")
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// Update in-memory token
	a.authManager.mu.Lock()
	a.authManager.Token = newToken
	a.authManager.mu.Unlock()

	// Update session in database
	now := time.Now()
	err = a.db.Model(&UserSession{}).
		Where("id = ?", session.ID).
		Updates(map[string]any{
			"token":               hashToken(newToken.AccessToken),
			"access_token_expiry": newToken.ExpiresAt,
			"last_activity_at":    now,
		}).Error

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Save to cache
	a.authManager.saveTokenCache()

	log.Println("✅ Token refreshed and session updated")
	return nil
}

// LogoutWithSession clears tokens and invalidates database session
func (a *App) LogoutWithSession() error {
	if a.authManager == nil {
		return nil
	}

	// Get current token before clearing
	var accessToken string
	a.authManager.mu.RLock()
	if a.authManager.Token != nil {
		accessToken = a.authManager.Token.AccessToken
	}
	a.authManager.mu.RUnlock()

	// Invalidate session in database
	if accessToken != "" {
		if err := a.authManager.InvalidateSessionByToken(accessToken, "user_logout"); err != nil {
			log.Printf("⚠️ Failed to invalidate session: %v", err)
		}
	}

	// Clear in-memory state directly (do NOT call a.Logout() — that calls back here, causing infinite recursion)
	a.authManager.mu.Lock()
	a.authManager.Token = nil
	a.authManager.Profile = nil
	a.authManager.mu.Unlock()

	// Remove cached token file from disk
	os.Remove(".auth_token.json")

	log.Println("✅ Logout complete: tokens cleared, session invalidated")
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// ENHANCED AUTH FLOW WITH SESSION MANAGEMENT
// ═══════════════════════════════════════════════════════════════════════════

// Enhanced waitForCallback with session creation
func (am *AuthManager) waitForCallbackWithSession() {
	defer am.stopCallbackServer()

	timeout := time.After(5 * time.Minute)

	select {
	case code := <-am.callbackChan:
		// Exchange code for token
		am.mu.RLock()
		pkce := am.pendingPKCE
		am.mu.RUnlock()

		if pkce == nil {
			log.Println("❌ PKCE params missing")
			am.emitAuthError("PKCE parameters missing")
			return
		}

		token, err := am.exchangeCodeForToken(code, pkce)
		if err != nil {
			log.Printf("❌ Token exchange failed: %v", err)
			am.emitAuthError(err.Error())
			return
		}

		am.mu.Lock()
		am.Token = token
		am.pendingPKCE = nil
		am.mu.Unlock()

		// Fetch user profile
		profile, err := am.fetchUserProfile()
		if err != nil {
			log.Printf("⚠️ Failed to fetch user profile: %v", err)
		} else {
			am.mu.Lock()
			am.Profile = profile
			am.mu.Unlock()
		}

		// Create session in database
		userID := ""
		if profile != nil {
			userID = profile.ID
		}
		if err := am.CreateSession(userID, token); err != nil {
			log.Printf("⚠️ Failed to create session: %v", err)
		}

		// Save token to cache
		am.saveTokenCache()

		// Emit success event
		am.emitAuthSuccess()

		log.Printf("✅ Authentication complete! User: %s", profile.DisplayName)

	case err := <-am.errorChan:
		log.Printf("❌ Authentication error: %v", err)
		am.emitAuthError(err.Error())

	case <-timeout:
		log.Println("❌ Authentication timeout")
		am.emitAuthError("Authentication timeout - please try again")
	}
}
