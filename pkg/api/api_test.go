package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// MockDatabase is a test database that can be configured to pass or fail
type MockDatabase struct {
	shouldFail bool
}

func (m *MockDatabase) Ping() error {
	if m.shouldFail {
		return errors.New("database connection failed")
	}
	return nil
}

func TestHealthCheck(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())
	server.SetGPUService(NewGPUService())

	// Wait a tiny bit to ensure uptime is measurable
	time.Sleep(10 * time.Millisecond)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", resp.Version)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	status, ok := data["status"].(string)
	if !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", status)
	}

	// Verify uptime is present and positive
	uptime, ok := data["uptime_seconds"].(float64)
	if !ok {
		t.Error("Expected uptime_seconds field")
	}
	if uptime <= 0 {
		t.Errorf("Expected positive uptime, got %f", uptime)
	}

	// Verify timestamp is present
	if _, ok := data["timestamp"].(string); !ok {
		t.Error("Expected timestamp field")
	}

	// Verify checks are present
	if _, ok := data["checks"].(map[string]any); !ok {
		t.Error("Expected checks field")
	}
}

func TestReadinessCheck_AllServicesReady(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())
	server.SetGPUService(NewGPUService())
	server.SetVQCService(NewVQCService())
	server.SetSurvivalGardenService(NewSurvivalGardenService())
	server.SetDatabase(&MockDatabase{shouldFail: false})

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	status, ok := data["status"].(string)
	if !ok || status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", status)
	}

	// Verify checks are present
	checks, ok := data["checks"].(map[string]any)
	if !ok {
		t.Fatal("Expected checks field")
	}

	// Verify database check
	dbCheck, ok := checks["database"].(map[string]any)
	if !ok {
		t.Fatal("Expected database check")
	}
	if dbCheck["status"] != "ok" {
		t.Errorf("Expected database status 'ok', got '%v'", dbCheck["status"])
	}

	// Verify services check
	services, ok := checks["services"].(map[string]any)
	if !ok {
		t.Fatal("Expected services check")
	}
	if services["ui_alchemy"] != "ok" {
		t.Error("Expected ui_alchemy to be ok")
	}
	if services["gpu"] != "ok" {
		t.Error("Expected gpu to be ok")
	}
	if services["vqc"] != "ok" {
		t.Error("Expected vqc to be ok")
	}
	if services["survival_garden"] != "ok" {
		t.Error("Expected survival_garden to be ok")
	}
}

func TestReadinessCheck_DatabaseFailed(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())
	server.SetGPUService(NewGPUService())
	server.SetVQCService(NewVQCService())
	server.SetSurvivalGardenService(NewSurvivalGardenService())
	server.SetDatabase(&MockDatabase{shouldFail: true}) // Simulated DB failure

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	status, ok := data["status"].(string)
	if !ok || status != "not_ready" {
		t.Errorf("Expected status 'not_ready', got '%s'", status)
	}
}

func TestReadinessCheck_ServicesNotInitialized(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetDatabase(&MockDatabase{shouldFail: false})
	// Don't set services - they'll be nil

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	status, ok := data["status"].(string)
	if !ok || status != "not_ready" {
		t.Errorf("Expected status 'not_ready', got '%s'", status)
	}

	// Verify services check shows not_initialized
	checks, ok := data["checks"].(map[string]any)
	if !ok {
		t.Fatal("Expected checks field")
	}

	services, ok := checks["services"].(map[string]any)
	if !ok {
		t.Fatal("Expected services check")
	}
	if services["ui_alchemy"] != "not_initialized" {
		t.Error("Expected ui_alchemy to be not_initialized")
	}
}

func TestReadinessCheck_NoDatabaseConfigured(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())
	server.SetGPUService(NewGPUService())
	server.SetVQCService(NewVQCService())
	server.SetSurvivalGardenService(NewSurvivalGardenService())
	// Don't set database - it'll be nil

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	checks, ok := data["checks"].(map[string]any)
	if !ok {
		t.Fatal("Expected checks field")
	}

	dbCheck, ok := checks["database"].(map[string]any)
	if !ok {
		t.Fatal("Expected database check")
	}
	if dbCheck["status"] != "not_configured" {
		t.Errorf("Expected database status 'not_configured', got '%v'", dbCheck["status"])
	}
}

func TestLivenessCheck(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)

	req := httptest.NewRequest("GET", "/live", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", resp.Version)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	status, ok := data["status"].(string)
	if !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", status)
	}

	// Verify checks are present
	checks, ok := data["checks"].(map[string]any)
	if !ok {
		t.Fatal("Expected checks field")
	}

	// Verify goroutine count
	if _, ok := checks["goroutines"].(float64); !ok {
		t.Error("Expected goroutines field")
	}

	// Verify responsive flag
	responsive, ok := checks["responsive"].(bool)
	if !ok || !responsive {
		t.Error("Expected responsive to be true")
	}
}

func TestMetricsIncludesHealthCheckCounters(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())
	server.SetGPUService(NewGPUService())
	server.SetVQCService(NewVQCService())
	server.SetSurvivalGardenService(NewSurvivalGardenService())
	server.SetDatabase(&MockDatabase{shouldFail: false})

	// Make some health check requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/ready", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}

	for i := 0; i < 1; i++ {
		req := httptest.NewRequest("GET", "/live", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)
	}

	// Now check metrics
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	body := w.Body.String()

	// Verify metrics are present
	if !contains(body, "asymmetrica_health_checks_total") {
		t.Error("Expected asymmetrica_health_checks_total metric")
	}
	if !contains(body, "asymmetrica_ready_checks_total") {
		t.Error("Expected asymmetrica_ready_checks_total metric")
	}
	if !contains(body, "asymmetrica_live_checks_total") {
		t.Error("Expected asymmetrica_live_checks_total metric")
	}
	if !contains(body, "asymmetrica_uptime_seconds") {
		t.Error("Expected asymmetrica_uptime_seconds metric")
	}

	// Verify counts (3 health, 2 ready, 1 live)
	if !contains(body, "asymmetrica_health_checks_total 3") {
		t.Errorf("Expected health check count of 3, got: %s", body)
	}
	if !contains(body, "asymmetrica_ready_checks_total 2") {
		t.Errorf("Expected ready check count of 2, got: %s", body)
	}
	if !contains(body, "asymmetrica_live_checks_total 1") {
		t.Errorf("Expected live check count of 1, got: %s", body)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetScreen(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())

	req := httptest.NewRequest("GET", "/api/v1/screens/dashboard?time_of_day=morning&flow_rate=45.0&urgency=0.2", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", resp.Version)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got: %s", resp.Error.Message)
	}
}

func TestComputeSLERP(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetGPUService(NewGPUService())

	reqBody := SLERPRequest{
		Q1: Quaternion{W: 1, X: 0, Y: 0, Z: 0},
		Q2: Quaternion{W: 0, X: 1, Y: 0, Z: 0},
		T:  0.5,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v1/gpu/slerp", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", resp.Version)
	}

	if resp.Error != nil {
		t.Errorf("Expected no error, got: %s", resp.Error.Message)
	}
}

func TestGPUStatus(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetGPUService(NewGPUService())

	req := httptest.NewRequest("GET", "/api/v1/gpu/status", nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", resp.Version)
	}
}

func TestGetEquilibrium(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetSurvivalGardenService(NewSurvivalGardenService())

	reqBody := SimulationParams{
		CashRunway:       6.0,
		MonthlyBurn:      10000.0,
		Expenses:         []Expense{},
		MonthsToSimulate: 12,
		GPUAccelerate:    false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("GET", "/api/v1/simulation/equilibrium", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Version != "v1" {
		t.Errorf("Expected version 'v1', got '%s'", resp.Version)
	}

	// Check the 87.532% attractor
	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("Response data is not a map")
	}

	attractor, ok := data["attractor"].(float64)
	if !ok {
		t.Fatal("Attractor not found in response")
	}

	if attractor != 0.87532 {
		t.Errorf("Expected attractor 0.87532, got %f", attractor)
	}
}

func TestCORS(t *testing.T) {
	server := NewServer("8080", "http://localhost:5173,http://localhost:8080", 60)

	// Test allowed origin
	req := httptest.NewRequest("OPTIONS", "/api/v1/screens/dashboard", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	corsHeader := w.Header().Get("Access-Control-Allow-Origin")
	if corsHeader != "http://localhost:5173" {
		t.Errorf("Expected CORS header 'http://localhost:5173', got '%s'", corsHeader)
	}

	// Test disallowed origin
	req2 := httptest.NewRequest("OPTIONS", "/api/v1/screens/dashboard", nil)
	req2.Header.Set("Origin", "http://evil.com")
	w2 := httptest.NewRecorder()

	server.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("Expected status 403 for disallowed origin, got %d", w2.Code)
	}

	corsHeader2 := w2.Header().Get("Access-Control-Allow-Origin")
	if corsHeader2 != "" {
		t.Errorf("Expected no CORS header for disallowed origin, got '%s'", corsHeader2)
	}
}

func TestRateLimiting(t *testing.T) {
	// Create server with very low rate limit for testing
	server := NewServer("8080", "http://localhost:5173", 2) // 2 requests per minute

	// Make requests until we hit the rate limit
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.1:1234" // Same IP for all requests
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should have allowed some requests and blocked others
	if successCount == 0 {
		t.Error("Expected some requests to succeed")
	}

	if rateLimitedCount == 0 {
		t.Error("Expected some requests to be rate limited")
	}

	t.Logf("Rate limiting test: %d succeeded, %d rate limited", successCount, rateLimitedCount)
}

func TestVersionedResponse(t *testing.T) {
	// Test that ALL responses have version field
	server := NewServer("8080", "http://localhost:5173", 60)
	server.SetUIAlchemyService(NewUIAlchemyService())
	server.SetGPUService(NewGPUService()) // FIX: Wire up GPU service!

	endpoints := []string{
		"/health",
		"/api/v1/screens/dashboard",
		"/api/v1/gpu/status",
	}

	for _, endpoint := range endpoints {
		req := httptest.NewRequest("GET", endpoint, nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		var resp APIResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response for %s: %v", endpoint, err)
		}

		if resp.Version != "v1" {
			t.Errorf("Endpoint %s missing version field or wrong version", endpoint)
		}
	}
}
