// ═══════════════════════════════════════════════════════════════════════════
// INTERACTIVE SESSION INACTIVITY (Wave 5 Mission B)
//
// Policy (decided with the Commander at wave kickoff): an interactive
// login expires after 30 minutes without a bound call; every bound call
// counts as activity and extends the session; an expired session's bound
// calls are refused with a clear "session expired" error and the frontend
// returns to the login screen (auth:session-expired event).
//
// Enforcement is server-side: requirePermission — the chokepoint every
// bound endpoint already passes through — calls touchInteractiveSession
// before any permission logic. The session is DB-backed through the same
// UserSession table AuthManager owns (auth_session.go), so login,
// expiry, and logout leave an audit trail (InvalidatedReason). This is
// the deliberate replacement for the write-only SessionManager deleted in
// Wave 4 (W4-D3): here the READ side is requirePermission itself, and the
// tests exercise it.
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// InteractiveSessionTimeout is the default idle window for interactive
// logins. Wave 6 Mission C.2: the window is configurable via settings
// ("security.session_timeout_minutes"); this constant is the fallback.
const InteractiveSessionTimeout = 30 * time.Minute

// Configurable bounds for the idle window: below 5 minutes the app is
// unusable (every pause re-authenticates); above 8 hours the timeout no
// longer protects an unattended terminal within a working day.
const (
	minInteractiveTimeoutMinutes = 5
	maxInteractiveTimeoutMinutes = 480
)

// interactiveSessionTimeout returns the effective idle window: the
// settings override when one was loaded, otherwise the default.
func (a *App) interactiveSessionTimeout() time.Duration {
	if a.interactiveTimeout > 0 {
		return a.interactiveTimeout
	}
	return InteractiveSessionTimeout
}

// applySessionTimeoutSetting validates and applies a timeout-in-minutes
// value from settings. Out-of-range values are clamped, not refused —
// settings.json is hand-editable and a typo must not lock the policy out.
func (a *App) applySessionTimeoutSetting(minutes float64) {
	if minutes <= 0 {
		return
	}
	if minutes < minInteractiveTimeoutMinutes {
		minutes = minInteractiveTimeoutMinutes
	}
	if minutes > maxInteractiveTimeoutMinutes {
		minutes = maxInteractiveTimeoutMinutes
	}
	a.interactiveTimeout = time.Duration(minutes) * time.Minute
}

// loadSessionTimeoutSetting pulls the configured idle window out of the
// user settings file (default when absent). Called at login so each
// interactive session starts with the current policy.
func (a *App) loadSessionTimeoutSetting() {
	if a.config == nil {
		return // no settings file location yet (early startup, tests)
	}
	settings, err := a.loadUserSettings()
	if err != nil {
		return
	}
	if security, ok := settings["security"].(map[string]any); ok {
		if minutes, ok := security["session_timeout_minutes"].(float64); ok {
			a.applySessionTimeoutSetting(minutes)
		}
	}
}

// interactiveActivityFlushInterval throttles last_activity_at writes so a
// chatty dashboard doesn't turn every poll into a DB write. The in-memory
// timestamp is authoritative for the timeout check; the row lags by at
// most this interval.
const interactiveActivityFlushInterval = time.Minute

// beginInteractiveSession records a DB-backed session for a successful
// interactive login (LoginDevice / SetupAdminAccount). A previous
// interactive session for this process is invalidated first.
func (a *App) beginInteractiveSession(userID string) {
	if a.db == nil {
		return
	}
	if a.interactiveSessionID != "" {
		_ = a.authSessionManager().InvalidateSession(a.interactiveSessionID, "superseded_by_new_login")
	}
	a.loadSessionTimeoutSetting()

	now := time.Now()
	// The UserSession token columns are unique-indexed; interactive
	// sessions have no OAuth tokens, so mint opaque per-session values.
	session := UserSession{
		UserID:             userID,
		Token:              hashToken("interactive:" + uuid.New().String()),
		RefreshToken:       hashToken("interactive-refresh:" + uuid.New().String()),
		AccessTokenExpiry:  now.Add(DefaultAccessTokenLifetime),
		RefreshTokenExpiry: now.Add(DefaultAccessTokenLifetime),
		LastActivityAt:     now,
		IsActive:           true,
	}
	if err := a.db.Create(&session).Error; err != nil {
		// The login itself must not fail on session bookkeeping; without a
		// row there is simply no inactivity enforcement for this login.
		log.Printf("⚠️ Failed to record interactive session: %v", err)
		return
	}

	a.interactiveSessionID = session.ID
	a.interactiveLastTouch = now
	a.interactiveLastPersist = now
	log.Printf("🔐 Interactive session started for user %s (idle timeout %v)", userID, a.interactiveSessionTimeout())
}

// touchInteractiveSession enforces the inactivity timeout and records
// activity. Called from requirePermission on every bound call. A nil
// return means the session is live (or no interactive session exists —
// license-based flows and tests that set currentUser directly are
// unaffected).
func (a *App) touchInteractiveSession() error {
	if a.currentUser == nil || a.interactiveSessionID == "" {
		return nil
	}

	now := time.Now()
	timeout := a.interactiveSessionTimeout()
	if now.Sub(a.interactiveLastTouch) > timeout {
		a.expireInteractiveSession("inactivity_timeout")
		return fmt.Errorf("session expired after %v of inactivity - please sign in again", timeout)
	}

	a.interactiveLastTouch = now
	if now.Sub(a.interactiveLastPersist) >= interactiveActivityFlushInterval {
		a.interactiveLastPersist = now
		_ = a.db.Model(&UserSession{}).
			Where("id = ?", a.interactiveSessionID).
			Update("last_activity_at", now).Error
	}
	return nil
}

// expireInteractiveSession invalidates the DB session, clears the
// in-memory user context, and tells the frontend to return to the login
// screen.
func (a *App) expireInteractiveSession(reason string) {
	if a.interactiveSessionID != "" {
		if err := a.authSessionManager().InvalidateSession(a.interactiveSessionID, reason); err != nil {
			log.Printf("⚠️ Failed to invalidate interactive session: %v", err)
		}
	}
	a.interactiveSessionID = ""
	a.currentUser = nil
	a.currentUserID = ""

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "auth:session-expired", map[string]any{"reason": reason})
	}
	log.Printf("🔒 Interactive session expired (%s)", reason)
}

// LogoutInteractiveSession ends the interactive login: the DB session is
// invalidated (audit trail: user_logout) and the in-memory user context
// cleared. Safe to call when no session exists.
func (a *App) LogoutInteractiveSession() error {
	if a.interactiveSessionID != "" {
		if err := a.authSessionManager().InvalidateSession(a.interactiveSessionID, "user_logout"); err != nil {
			log.Printf("⚠️ Failed to invalidate interactive session on logout: %v", err)
		}
	}
	a.interactiveSessionID = ""
	a.currentUser = nil
	a.currentUserID = ""
	log.Println("✅ Interactive logout complete")
	return nil
}

// authSessionManager returns the AuthManager used for session rows,
// creating a DB-only one when OAuth was never configured (offline-first:
// interactive sessions must not depend on OAuth setup).
func (a *App) authSessionManager() *AuthManager {
	if a.authManager != nil {
		return a.authManager
	}
	return &AuthManager{app: a}
}
