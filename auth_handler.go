// ═══════════════════════════════════════════════════════════════════════════
// AUTH HANDLER - Microsoft OAuth2 PKCE Authentication for Wails Desktop App
//
// MISSION: Secure login via Microsoft Identity Platform
//
// FLOW:
//   1. Frontend calls StartLogin() → returns auth URL
//   2. User opens URL in browser → Microsoft login
//   3. Browser redirects to localhost:8080/callback with code
//   4. Backend exchanges code for tokens
//   5. Frontend receives success callback
//
// ARCHITECTURE:
//   - Uses OAuth2 PKCE (no client secret for desktop apps!)
//   - Local HTTP server captures callback
//   - Tokens cached in memory + optional disk persistence
//   - Auto-refresh before expiration
//
// Built with SECURITY × SIMPLICITY × WAILS INTEGRATION 🔐⚡💎
// Day 194 - Acme Instrumentation M365 Authentication
// ═══════════════════════════════════════════════════════════════════════════

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ═══════════════════════════════════════════════════════════════════════════
// CONSTANTS
// ═══════════════════════════════════════════════════════════════════════════

const (
	// Microsoft Identity Platform endpoints
	AuthorizeEndpointTemplate = "https://login.microsoftonline.com/%s/oauth2/v2.0/authorize"
	TokenEndpointTemplate     = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"

	// PKCE
	CodeVerifierLength = 64

	// Local callback server
	CallbackPort = 8080
	CallbackPath = "/callback"
)

// Required Microsoft Graph scopes for Acme Instrumentation
var RequiredScopes = []string{
	"User.Read",       // User profile
	"Files.ReadWrite", // OneDrive files (user's own, not .All)
	"Mail.Send",       // Send emails only (removed Mail.ReadWrite)
	"Calendars.Read",  // Read calendar (reduced from ReadWrite)
	"Sites.Read.All",  // Read SharePoint sites (reduced from ReadWrite)
	"offline_access",  // Refresh tokens
}

// ═══════════════════════════════════════════════════════════════════════════
// TYPES
// ═══════════════════════════════════════════════════════════════════════════

// AuthState represents current authentication state
type AuthState struct {
	IsAuthenticated bool      `json:"is_authenticated"`
	UserEmail       string    `json:"user_email"`
	UserName        string    `json:"user_name"`
	ExpiresAt       time.Time `json:"expires_at"`
	Scopes          []string  `json:"scopes"`
}

// TokenResponse from Microsoft Identity Platform
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	IDToken      string `json:"id_token,omitempty"`

	// Calculated
	ExpiresAt time.Time `json:"-"`
}

// UserProfile from Microsoft Graph /me endpoint
type UserProfile struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	GivenName         string `json:"givenName"`
	Surname           string `json:"surname"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
	JobTitle          string `json:"jobTitle,omitempty"`
}

// PKCEParams for OAuth2 PKCE flow
type PKCEParams struct {
	CodeVerifier  string
	CodeChallenge string
	State         string
}

// AuthManager handles the OAuth2 flow
type AuthManager struct {
	mu sync.RWMutex

	// Configuration
	ClientID    string
	TenantID    string
	RedirectURI string
	Scopes      []string

	// Current state
	Token   *TokenResponse
	Profile *UserProfile

	// PKCE flow state (temporary during auth)
	pendingPKCE  *PKCEParams
	callbackChan chan string
	errorChan    chan error
	httpServer   *http.Server

	// Wails app reference for events
	app *App
}

// ═══════════════════════════════════════════════════════════════════════════
// CONSTRUCTOR
// ═══════════════════════════════════════════════════════════════════════════

// NewAuthManager creates auth manager from config
func NewAuthManager(app *App) *AuthManager {
	clientID := ""
	tenantID := "common"

	if app.config != nil && app.config.Azure.ClientID != "" {
		clientID = app.config.Azure.ClientID
		tenantID = app.config.Azure.TenantID
	}

	return &AuthManager{
		ClientID:     clientID,
		TenantID:     tenantID,
		RedirectURI:  fmt.Sprintf("http://localhost:%d%s", CallbackPort, CallbackPath),
		Scopes:       RequiredScopes,
		callbackChan: make(chan string, 1),
		errorChan:    make(chan error, 1),
		app:          app,
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PKCE HELPERS
// ═══════════════════════════════════════════════════════════════════════════

// generateCodeVerifier creates cryptographically random PKCE code verifier
func generateCodeVerifier() (string, error) {
	bytes := make([]byte, CodeVerifierLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:CodeVerifierLength], nil
}

// generateCodeChallenge creates SHA256 hash of verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateState creates CSRF protection token
func generateState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// NewPKCEParams generates all PKCE flow parameters
func NewPKCEParams() (*PKCEParams, error) {
	verifier, err := generateCodeVerifier()
	if err != nil {
		return nil, err
	}

	state, err := generateState()
	if err != nil {
		return nil, err
	}

	return &PKCEParams{
		CodeVerifier:  verifier,
		CodeChallenge: generateCodeChallenge(verifier),
		State:         state,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// WAILS API - Exposed to Frontend
// ═══════════════════════════════════════════════════════════════════════════

// GetAuthState returns current authentication state
func (a *App) GetAuthState() *AuthState {
	if a.authManager == nil {
		return &AuthState{IsAuthenticated: false}
	}

	a.authManager.mu.RLock()
	defer a.authManager.mu.RUnlock()

	if a.authManager.Token == nil {
		return &AuthState{IsAuthenticated: false}
	}

	// Check if token is expired
	if time.Now().After(a.authManager.Token.ExpiresAt) {
		return &AuthState{IsAuthenticated: false}
	}

	state := &AuthState{
		IsAuthenticated: true,
		ExpiresAt:       a.authManager.Token.ExpiresAt,
		Scopes:          a.authManager.Scopes,
	}

	if a.authManager.Profile != nil {
		state.UserEmail = a.authManager.Profile.Mail
		if state.UserEmail == "" {
			state.UserEmail = a.authManager.Profile.UserPrincipalName
		}
		state.UserName = a.authManager.Profile.DisplayName
	}

	return state
}

// StartLogin initiates OAuth2 PKCE flow
// Returns auth URL that user should open in browser
func (a *App) StartLogin() (string, error) {
	if a.authManager == nil {
		a.authManager = NewAuthManager(a)
	}

	// Validate configuration
	if a.authManager.ClientID == "" {
		return "", fmt.Errorf("Azure Client ID not configured. Please set AZURE_CLIENT_ID in .env file")
	}

	if a.authManager.TenantID == "" {
		a.authManager.TenantID = "common" // Multi-tenant fallback
	}

	// Generate PKCE parameters
	pkce, err := NewPKCEParams()
	if err != nil {
		return "", fmt.Errorf("failed to generate PKCE parameters: %w", err)
	}

	a.authManager.mu.Lock()
	a.authManager.pendingPKCE = pkce
	a.authManager.mu.Unlock()

	// Start local callback server
	if err := a.authManager.startCallbackServer(); err != nil {
		return "", fmt.Errorf("failed to start callback server: %w", err)
	}

	// Build authorization URL
	authURL := a.authManager.buildAuthorizationURL(pkce)

	// Open in default browser
	go func() {
		time.Sleep(100 * time.Millisecond) // Give server time to start
		openBrowser(authURL)
	}()

	// Wait for callback in background
	go a.authManager.waitForCallback()

	log.Printf("🔐 OAuth2 login initiated. Auth URL: %s", authURL)
	return authURL, nil
}

// Logout clears authentication state
func (a *App) Logout() error {
	// Use enhanced logout with session invalidation
	return a.LogoutWithSession()
}

// RefreshAuth refreshes the access token if needed
func (a *App) RefreshAuth() error {
	if a.authManager == nil || a.authManager.Token == nil {
		return fmt.Errorf("not authenticated")
	}

	// Check if refresh is needed (token expires in < 5 min)
	if time.Now().Add(5 * time.Minute).Before(a.authManager.Token.ExpiresAt) {
		return nil // Token still valid
	}

	if a.authManager.Token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	// Refresh the token
	newToken, err := a.authManager.refreshToken(a.authManager.Token.RefreshToken)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	a.authManager.mu.Lock()
	a.authManager.Token = newToken
	a.authManager.mu.Unlock()

	// Save to cache
	a.authManager.saveTokenCache()

	log.Println("🔄 Token refreshed successfully")
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════
// INTERNAL METHODS
// ═══════════════════════════════════════════════════════════════════════════

// buildAuthorizationURL constructs Microsoft OAuth2 authorization URL
func (am *AuthManager) buildAuthorizationURL(pkce *PKCEParams) string {
	endpoint := fmt.Sprintf(AuthorizeEndpointTemplate, am.TenantID)

	params := url.Values{}
	params.Set("client_id", am.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", am.RedirectURI)
	params.Set("scope", strings.Join(am.Scopes, " "))
	params.Set("state", pkce.State)
	params.Set("code_challenge", pkce.CodeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("response_mode", "query")

	return endpoint + "?" + params.Encode()
}

// startCallbackServer starts HTTP server to receive OAuth callback
func (am *AuthManager) startCallbackServer() error {
	mux := http.NewServeMux()

	mux.HandleFunc(CallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errorParam := r.URL.Query().Get("error")
		errorDesc := r.URL.Query().Get("error_description")

		// Validate state to prevent CSRF
		am.mu.RLock()
		expectedState := ""
		if am.pendingPKCE != nil {
			expectedState = am.pendingPKCE.State
		}
		am.mu.RUnlock()

		if state != expectedState {
			am.errorChan <- fmt.Errorf("state mismatch - possible CSRF attack")
			fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>Security validation failed.</p></body></html>")
			return
		}

		// Check for errors
		if errorParam != "" {
			am.errorChan <- fmt.Errorf("OAuth error: %s - %s", errorParam, errorDesc)
			fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>%s</p></body></html>", html.EscapeString(errorDesc))
			return
		}

		// Validate code
		if code == "" {
			am.errorChan <- fmt.Errorf("no authorization code received")
			fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>No code received.</p></body></html>")
			return
		}

		// Send code to channel
		am.callbackChan <- code

		// Success page
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Acme Instrumentation - Login Successful</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #1a1a2e 0%%, #16213e 100%%);
            color: #fff;
        }
        .container {
            text-align: center;
            padding: 40px;
            background: rgba(255,255,255,0.1);
            border-radius: 16px;
            backdrop-filter: blur(10px);
        }
        h1 { color: #4ade80; margin-bottom: 16px; }
        p { color: #94a3b8; }
        .icon { font-size: 64px; margin-bottom: 16px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">✅</div>
        <h1>Login Successful!</h1>
        <p>You can close this window and return to Acme Instrumentation.</p>
    </div>
</body>
</html>
		`)
	})

	// Health check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Acme Instrumentation OAuth2 callback server running")
	})

	am.httpServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", CallbackPort),
		Handler: mux,
	}

	// Start in goroutine
	go func() {
		if err := am.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("⚠️ Callback server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond) // Give server time to start
	log.Printf("🌐 Callback server started on port %d", CallbackPort)
	return nil
}

// stopCallbackServer shuts down the callback server
func (am *AuthManager) stopCallbackServer() {
	if am.httpServer != nil {
		am.httpServer.Close()
		am.httpServer = nil
		log.Println("🛑 Callback server stopped")
	}
}

// waitForCallback waits for OAuth callback and completes flow
func (am *AuthManager) waitForCallback() {
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

		log.Printf("✅ Authentication complete! User: %s", am.Profile.DisplayName)

	case err := <-am.errorChan:
		log.Printf("❌ Authentication error: %v", err)
		am.emitAuthError(err.Error())

	case <-timeout:
		log.Println("❌ Authentication timeout")
		am.emitAuthError("Authentication timeout - please try again")
	}
}

// exchangeCodeForToken exchanges authorization code for access token
func (am *AuthManager) exchangeCodeForToken(code string, pkce *PKCEParams) (*TokenResponse, error) {
	endpoint := fmt.Sprintf(TokenEndpointTemplate, am.TenantID)

	data := url.Values{}
	data.Set("client_id", am.ClientID)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", am.RedirectURI)
	data.Set("code_verifier", pkce.CodeVerifier)

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// refreshToken refreshes an expired access token
func (am *AuthManager) refreshToken(refreshToken string) (*TokenResponse, error) {
	endpoint := fmt.Sprintf(TokenEndpointTemplate, am.TenantID)

	data := url.Values{}
	data.Set("client_id", am.ClientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("scope", strings.Join(am.Scopes, " "))

	resp, err := http.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &token, nil
}

// fetchUserProfile fetches user profile from Microsoft Graph
func (am *AuthManager) fetchUserProfile() (*UserProfile, error) {
	if am.Token == nil {
		return nil, fmt.Errorf("no access token")
	}

	req, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+am.Token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("profile request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile request failed (status %d): %s", resp.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse profile: %w", err)
	}

	return &profile, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// TOKEN CACHING (AES-256-GCM ENCRYPTED)
// ═══════════════════════════════════════════════════════════════════════════

// saveTokenCache saves token to disk with AES-256-GCM encryption
func (am *AuthManager) saveTokenCache() {
	if am.Token == nil {
		return
	}

	tokenPath := filepath.Join(".", ".auth_token.json")

	// Marshal token data
	data, err := json.MarshalIndent(map[string]any{
		"access_token":  am.Token.AccessToken,
		"refresh_token": am.Token.RefreshToken,
		"expires_at":    am.Token.ExpiresAt.Format(time.RFC3339),
		"profile":       am.Profile,
	}, "", "  ")
	if err != nil {
		log.Printf("⚠️ Failed to marshal token: %v", err)
		return
	}

	// Encrypt the token data
	encryptedData, err := am.encryptToken(string(data))
	if err != nil {
		log.Printf("⚠️ Failed to encrypt token: %v", err)
		return
	}

	// Write encrypted data with secure file permissions (0600 = owner read/write only)
	if err := os.WriteFile(tokenPath, []byte(encryptedData), 0600); err != nil {
		log.Printf("⚠️ Failed to save token cache: %v", err)
		return
	}

	log.Println("💾 Token saved (AES-256-GCM encrypted)")
}

// loadTokenCache loads cached token from disk with automatic migration
func (am *AuthManager) loadTokenCache() bool {
	tokenPath := filepath.Join(".", ".auth_token.json")

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return false
	}

	// Try to load as encrypted token first
	if am.loadEncryptedToken(data) {
		return true
	}

	// Fallback: Try to load as plaintext (backward compatibility)
	// If successful, re-save as encrypted
	if am.loadPlaintextToken(data) {
		log.Println("🔄 Migrating plaintext token to encrypted storage...")
		am.saveTokenCache() // Re-save encrypted
		return true
	}

	return false
}

// loadEncryptedToken attempts to decrypt and load token
func (am *AuthManager) loadEncryptedToken(encryptedData []byte) bool {
	// Decrypt the data
	plaintext, err := am.decryptToken(string(encryptedData))
	if err != nil {
		return false // Not encrypted or invalid encryption
	}

	var cache struct {
		AccessToken  string       `json:"access_token"`
		RefreshToken string       `json:"refresh_token"`
		ExpiresAt    string       `json:"expires_at"`
		Profile      *UserProfile `json:"profile"`
	}

	if err := json.Unmarshal([]byte(plaintext), &cache); err != nil {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, cache.ExpiresAt)
	if err != nil {
		return false
	}

	am.Token = &TokenResponse{
		AccessToken:  cache.AccessToken,
		RefreshToken: cache.RefreshToken,
		ExpiresAt:    expiresAt,
	}
	am.Profile = cache.Profile

	log.Println("📂 Loaded encrypted token cache")
	return true
}

// loadPlaintextToken attempts to load plaintext token (backward compatibility)
func (am *AuthManager) loadPlaintextToken(data []byte) bool {
	var cache struct {
		AccessToken  string       `json:"access_token"`
		RefreshToken string       `json:"refresh_token"`
		ExpiresAt    string       `json:"expires_at"`
		Profile      *UserProfile `json:"profile"`
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, cache.ExpiresAt)
	if err != nil {
		return false
	}

	am.Token = &TokenResponse{
		AccessToken:  cache.AccessToken,
		RefreshToken: cache.RefreshToken,
		ExpiresAt:    expiresAt,
	}
	am.Profile = cache.Profile

	log.Println("📂 Loaded plaintext token (will be migrated)")
	return true
}

// encryptToken encrypts token data using AES-256-GCM
func (am *AuthManager) encryptToken(plaintext string) (string, error) {
	// Get encryption key from hardware ID
	encryptionKey, err := am.getEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and seal
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptToken decrypts token data
func (am *AuthManager) decryptToken(ciphertext string) (string, error) {
	// Get encryption key from hardware ID
	encryptionKey, err := am.getEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce and ciphertext
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt and open
	plaintext, err := aesGCM.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// getEncryptionKey derives AES-256 key from hardware ID
func (am *AuthManager) getEncryptionKey() ([]byte, error) {
	// Get hardware ID (reuses function from settings_service.go)
	hardwareID, err := getHardwareID()
	if err != nil {
		log.Printf("⚠️ Failed to get hardware ID: %v (using fallback)", err)
		hardwareID = "fallback-key-asymmetrica-auth"
	}

	// Generate 32-byte AES-256 key using SHA-256
	hash := sha256.Sum256([]byte(hardwareID + "ph-holdings-auth-salt-2026"))
	return hash[:], nil
}

// ═══════════════════════════════════════════════════════════════════════════
// EVENT EMITTERS (Wails → Frontend)
// ═══════════════════════════════════════════════════════════════════════════

// emitAuthSuccess emits success event to frontend
func (am *AuthManager) emitAuthSuccess() {
	if am.app != nil && am.app.ctx != nil {
		state := &AuthState{
			IsAuthenticated: true,
			ExpiresAt:       am.Token.ExpiresAt,
			Scopes:          am.Scopes,
		}

		if am.Profile != nil {
			state.UserEmail = am.Profile.Mail
			if state.UserEmail == "" {
				state.UserEmail = am.Profile.UserPrincipalName
			}
			state.UserName = am.Profile.DisplayName
		}

		wailsRuntime.EventsEmit(am.app.ctx, "auth:success", state)
	}
}

// emitAuthError emits error event to frontend
func (am *AuthManager) emitAuthError(message string) {
	if am.app != nil && am.app.ctx != nil {
		wailsRuntime.EventsEmit(am.app.ctx, "auth:error", map[string]string{
			"error": message,
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// UTILITY FUNCTIONS
// ═══════════════════════════════════════════════════════════════════════════

// openBrowser opens URL in default browser
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // Linux and others
		cmd = exec.Command("xdg-open", url)
	}
	suppressCommandWindow(cmd)

	if err := cmd.Start(); err != nil {
		log.Printf("⚠️ Failed to open browser: %v", err)
		log.Printf("Please open this URL manually: %s", url)
	}
}

// GetAccessToken returns valid access token (for API calls)
func (a *App) GetAccessToken() (string, error) {
	// Use enhanced method with session validation
	return a.GetAccessTokenWithValidation()
}
