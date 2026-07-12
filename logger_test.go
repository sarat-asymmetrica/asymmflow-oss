package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// TestLoggerDevMode tests the logger in dev mode (console output with emojis)
func TestLoggerDevMode(t *testing.T) {
	// Create logger in dev mode
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelDebug,
	})

	logger.Info("Application started", map[string]any{
		"version":     "1.0.0",
		"environment": "development",
	})

	logger.Debug("Database connection", map[string]any{
		"host": "localhost",
		"port": 5432,
	})

	logger.Warn("Missing configuration", map[string]any{
		"missing_key": "API_TOKEN",
	})

	logger.Error("Database query failed", errors.New("connection timeout"), map[string]any{
		"query":       "SELECT * FROM users",
		"duration_ms": 5000,
	})
}

// TestLoggerProductionMode tests the logger in production mode (JSON output)
func TestLoggerProductionMode(t *testing.T) {
	// Create temp file for JSON logs
	tmpFile, err := os.CreateTemp("", "ph_test_*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create logger in production mode with file output
	logger := NewLogger(LoggerConfig{
		Mode:       "production",
		Level:      LevelInfo,
		OutputFile: tmpFile,
	})

	logger.Info("Application started", map[string]any{
		"version":     "1.0.0",
		"environment": "production",
	})

	logger.Error("Database query failed", errors.New("connection timeout"), map[string]any{
		"query":       "SELECT * FROM users",
		"duration_ms": 5000,
	})

	// Flush and read back
	tmpFile.Sync()
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	t.Logf("JSON Output:\n%s", string(content))

	// Verify JSON structure (basic check)
	logContent := string(content)
	if !containsString(logContent, `"level":"INFO"`) && !containsString(logContent, `"level":"info"`) {
		t.Error("Expected JSON log to contain level field")
	}
	if !containsString(logContent, `"msg"`) {
		t.Error("Expected JSON log to contain msg field")
	}
}

// TestLoggerWithRequestID tests request ID propagation through context
func TestLoggerWithRequestID(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelInfo,
	})

	// Create context with request ID
	ctx := logger.WithRequestID(context.Background(), "req-12345")

	// Log with context - request ID should be included
	logger.InfoCtx(ctx, "Processing request", map[string]any{
		"user_id": 42,
		"action":  "create_order",
	})

	// Without context - no request ID
	logger.Info("Background job completed", map[string]any{
		"job_type": "email_sender",
	})
}

// TestLoggerPerformanceMetrics tests performance logging
func TestLoggerPerformanceMetrics(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelInfo,
	})

	start := time.Now()
	time.Sleep(10 * time.Millisecond) // Simulate work
	duration := time.Since(start)

	logger.Performance("database_query", duration, map[string]any{
		"table":      "customers",
		"rows_count": 1500,
	})
}

// TestLoggerBusinessMetrics tests business intelligence logging
func TestLoggerBusinessMetrics(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelInfo,
	})

	logger.Business("revenue", 45000.50, map[string]any{
		"currency": "BHD",
		"quarter":  "Q1",
	})

	logger.Business("orders_completed", 127, map[string]any{
		"period": "daily",
	})
}

// TestLoggerGPUMetrics tests GPU operation logging
func TestLoggerGPUMetrics(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelInfo,
	})

	logger.GPU("matrix_multiplication", true, map[string]any{
		"matrix_size": "1024x1024",
		"duration_ms": 15,
		"throughput":  "71M ops/sec",
	})

	logger.GPU("gpu_initialization", false, map[string]any{
		"error":    "device not found",
		"fallback": "CPU mode",
	})
}

// TestLoggerSecurityEvents tests security event logging
func TestLoggerSecurityEvents(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelInfo,
	})

	logger.Security("login_attempt", true, map[string]any{
		"username": "admin",
		"ip":       "192.168.1.100",
	})

	logger.Security("unauthorized_access", false, map[string]any{
		"username": "hacker",
		"resource": "/api/admin/users",
		"ip":       "203.0.113.42",
	})
}

// TestLoggerComponentLogger tests component-scoped logging
func TestLoggerComponentLogger(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelDebug,
	})

	// Create component logger with preset fields
	ocrLogger := logger.WithFields(map[string]any{
		"component": "ocr_service",
		"version":   "2.0",
	})

	ocrLogger.Info("Processing document", map[string]any{
		"document_id": "DOC-123",
		"pages":       5,
	})

	ocrLogger.Error("OCR extraction failed", errors.New("invalid PDF"), map[string]any{
		"document_id": "DOC-456",
	})
}

// TestLoggerStartupBanner tests startup banner formatting
func TestLoggerStartupBanner(t *testing.T) {
	logger := NewLogger(LoggerConfig{
		Mode:  "dev",
		Level: LevelInfo,
	})

	logger.Startup("Acme Instrumentation Sovereign UI", map[string]any{
		"version":    "1.0.0",
		"go_version": "1.25.5",
		"os":         "windows",
		"arch":       "amd64",
	})
}

// Helper function for string containment check
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Example output for documentation
func ExampleLogger_Info() {
	logger := NewLogger(LoggerConfig{
		Mode:  "production",
		Level: LevelInfo,
	})

	logger.Info("Order created", map[string]any{
		"order_id":    "ORD-2024-001",
		"customer_id": 42,
		"total":       1500.50,
		"currency":    "BHD",
	})

	// Output (JSON format):
	// {"time":"2024-01-20T10:30:00Z","level":"INFO","msg":"Order created","order_id":"ORD-2024-001","customer_id":42,"total":1500.5,"currency":"BHD"}
}

func ExampleLogger_Performance() {
	logger := NewLogger(LoggerConfig{
		Mode:  "production",
		Level: LevelInfo,
	})

	start := time.Now()
	// ... some operation ...
	duration := time.Since(start)

	logger.Performance("customer_360_query", duration, map[string]any{
		"nodes_traversed": 1500,
		"depth":           5,
	})

	// Output (JSON format):
	// {"time":"2024-01-20T10:30:00Z","level":"INFO","msg":"Performance metric","operation":"customer_360_query","duration_ms":125,"duration_human":"125ms","nodes_traversed":1500,"depth":5,"metric_type":"performance"}
}
