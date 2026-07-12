package main

// Wave 5 Mission B tests. The W4-D3 mirror rule: a security component is
// only real if something READS it — here the read side is
// requirePermission itself, so these tests drive bound-call behavior, not
// just row states.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupSessionApp(t *testing.T) *App {
	t.Helper()
	app := setupTestApp(t)
	require.NoError(t, app.db.AutoMigrate(&UserSession{}))
	return app
}

func activeSessionRow(t *testing.T, app *App) UserSession {
	t.Helper()
	var session UserSession
	require.NoError(t, app.db.First(&session, "id = ?", app.interactiveSessionID).Error)
	return session
}

func TestInteractiveSession_ExpiryBlocksBoundCall(t *testing.T) {
	app := setupSessionApp(t)
	app.beginInteractiveSession("test-user")
	require.NotEmpty(t, app.interactiveSessionID)
	sessionID := app.interactiveSessionID

	// Live session: a bound call passes.
	require.NoError(t, app.requirePermission("settings:view"))

	// Simulate 31 minutes of inactivity.
	app.interactiveLastTouch = time.Now().Add(-31 * time.Minute)

	err := app.requirePermission("settings:view")
	require.Error(t, err, "expired session must refuse bound calls")
	require.Contains(t, err.Error(), "session expired")

	// The in-memory user context is cleared…
	require.Nil(t, app.currentUser)
	require.Empty(t, app.interactiveSessionID)

	// …and the DB row carries the audit trail.
	var session UserSession
	require.NoError(t, app.db.First(&session, "id = ?", sessionID).Error)
	require.False(t, session.IsActive)
	require.Equal(t, "inactivity_timeout", session.InvalidatedReason)
	require.NotNil(t, session.InvalidatedAt)
}

func TestInteractiveSession_ActivityExtends(t *testing.T) {
	app := setupSessionApp(t)
	app.beginInteractiveSession("test-user")

	// 29 minutes idle — under the limit; the call passes AND resets the clock.
	app.interactiveLastTouch = time.Now().Add(-29 * time.Minute)
	require.NoError(t, app.requirePermission("settings:view"))

	// Another 29 minutes after that call — still under the limit only
	// because the previous call extended the session.
	app.interactiveLastTouch = app.interactiveLastTouch.Add(-29 * time.Minute)
	require.NoError(t, app.requirePermission("settings:view"))

	require.True(t, activeSessionRow(t, app).IsActive)
}

func TestInteractiveSession_ActivityPersistsThrottled(t *testing.T) {
	app := setupSessionApp(t)
	app.beginInteractiveSession("test-user")

	// Backdate the persisted row so a flush strictly advances it even when
	// the flush lands in the same clock tick as session creation (the row's
	// stored precision is coarser than the monotonic clock).
	backdated := time.Now().Add(-1 * time.Minute)
	require.NoError(t, app.db.Model(&UserSession{}).
		Where("id = ?", app.interactiveSessionID).
		Update("last_activity_at", backdated).Error)
	before := activeSessionRow(t, app).LastActivityAt

	// Make the throttle window elapse so the next call flushes.
	app.interactiveLastPersist = time.Now().Add(-2 * time.Minute)
	require.NoError(t, app.requirePermission("settings:view"))

	after := activeSessionRow(t, app).LastActivityAt
	require.True(t, after.After(before), "activity flush must advance last_activity_at")
}

func TestInteractiveSession_LogoutInvalidates(t *testing.T) {
	app := setupSessionApp(t)
	app.beginInteractiveSession("test-user")
	sessionID := app.interactiveSessionID

	require.NoError(t, app.LogoutInteractiveSession())
	require.Nil(t, app.currentUser)
	require.Empty(t, app.interactiveSessionID)

	var session UserSession
	require.NoError(t, app.db.First(&session, "id = ?", sessionID).Error)
	require.False(t, session.IsActive)
	require.Equal(t, "user_logout", session.InvalidatedReason)

	// A bound call after logout fails on authentication, not on a stale
	// session check.
	err := app.requirePermission("settings:view")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "session expired")
}

func TestInteractiveSession_NewLoginSupersedesOld(t *testing.T) {
	app := setupSessionApp(t)
	app.beginInteractiveSession("test-user")
	first := app.interactiveSessionID

	app.beginInteractiveSession("test-user")
	second := app.interactiveSessionID
	require.NotEqual(t, first, second)

	var old UserSession
	require.NoError(t, app.db.First(&old, "id = ?", first).Error)
	require.False(t, old.IsActive)
	require.Equal(t, "superseded_by_new_login", old.InvalidatedReason)
}

func TestInteractiveSession_TestAppsWithoutSessionUnaffected(t *testing.T) {
	app := setupSessionApp(t)
	// setupTestApp sets currentUser directly with no interactive session —
	// the pre-existing model (and every existing test) must keep working.
	require.Empty(t, app.interactiveSessionID)
	require.NoError(t, app.requirePermission("settings:view"))
}

// Wave 6 Mission C.2: the idle window is configurable from settings.
func TestInteractiveSession_ConfigurableTimeout(t *testing.T) {
	app := setupSessionApp(t)
	app.beginInteractiveSession("test-user")

	// Tighten the window to 10 minutes: 11 idle minutes now expire the
	// session that the 30-minute default would have kept alive.
	app.applySessionTimeoutSetting(10)
	app.interactiveLastTouch = time.Now().Add(-11 * time.Minute)
	err := app.requirePermission("settings:view")
	require.Error(t, err, "session must expire under the configured window")
	require.Contains(t, err.Error(), "10m")
}

func TestApplySessionTimeoutSetting_Clamps(t *testing.T) {
	app := setupSessionApp(t)

	app.applySessionTimeoutSetting(1) // below floor
	require.Equal(t, 5*time.Minute, app.interactiveSessionTimeout())

	app.applySessionTimeoutSetting(10000) // above ceiling
	require.Equal(t, 480*time.Minute, app.interactiveSessionTimeout())

	app.applySessionTimeoutSetting(45)
	require.Equal(t, 45*time.Minute, app.interactiveSessionTimeout())

	// Zero/negative input leaves the current value untouched.
	app.applySessionTimeoutSetting(0)
	require.Equal(t, 45*time.Minute, app.interactiveSessionTimeout())
}
