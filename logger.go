package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"time"
)

// LogLevel represents the severity of log messages
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelFatal
)

// String returns the string representation of log level
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ToSlogLevel converts LogLevel to slog.Level
func (l LogLevel) ToSlogLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	case LevelFatal:
		return slog.LevelError // slog doesn't have Fatal, use Error
	default:
		return slog.LevelInfo
	}
}

// Logger wraps slog with additional functionality for Acme Instrumentation app
type Logger struct {
	slogger       *slog.Logger
	consoleLogger *slog.Logger
	level         LogLevel
	mode          string // "dev" or "production"
	requestIDKey  string
}

// LoggerConfig holds configuration for logger initialization
type LoggerConfig struct {
	Mode         string   // "dev" or "production"
	Level        LogLevel // Minimum log level to output
	OutputFile   *os.File // File for JSON logs (optional)
	RequestIDKey string   // Context key for request ID (default: "request_id")
}

// NewLogger creates a new structured logger instance
func NewLogger(cfg LoggerConfig) *Logger {
	if cfg.RequestIDKey == "" {
		cfg.RequestIDKey = "request_id"
	}

	// Console logger with emoji and human-readable format (for dev mode)
	consoleHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.Level.ToSlogLevel(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Add emoji prefixes to console output
			if a.Key == slog.LevelKey {
				level := a.Value.Any().(slog.Level)
				switch level {
				case slog.LevelDebug:
					a.Value = slog.StringValue("🔍 DEBUG")
				case slog.LevelInfo:
					a.Value = slog.StringValue("✅ INFO")
				case slog.LevelWarn:
					a.Value = slog.StringValue("⚠️  WARN")
				case slog.LevelError:
					a.Value = slog.StringValue("🔥 ERROR")
				}
			}
			return a
		},
	})
	consoleLogger := slog.New(consoleHandler)

	// JSON logger for production/cloud platforms
	var jsonLogger *slog.Logger
	if cfg.OutputFile != nil {
		// Multi-writer: both file and stdout
		multiWriter := io.MultiWriter(cfg.OutputFile, os.Stdout)
		jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
			Level:     cfg.Level.ToSlogLevel(),
			AddSource: true, // Include file:line information
		})
		jsonLogger = slog.New(jsonHandler)
	} else if cfg.Mode == "production" {
		// Production mode without file: JSON to stdout
		jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:     cfg.Level.ToSlogLevel(),
			AddSource: true,
		})
		jsonLogger = slog.New(jsonHandler)
	} else {
		// Dev mode: use console logger
		jsonLogger = consoleLogger
	}

	return &Logger{
		slogger:       jsonLogger,
		consoleLogger: consoleLogger,
		level:         cfg.Level,
		mode:          cfg.Mode,
		requestIDKey:  cfg.RequestIDKey,
	}
}

// WithRequestID returns a context with request ID embedded
func (l *Logger) WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, l.requestIDKey, requestID)
}

// extractRequestID gets request ID from context if available
func (l *Logger) extractRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if reqID, ok := ctx.Value(l.requestIDKey).(string); ok {
		return reqID
	}
	return ""
}

// buildAttrs creates slog attributes from fields map
func (l *Logger) buildAttrs(fields map[string]any) []any {
	attrs := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		attrs = append(attrs, k, v)
	}
	return attrs
}

// Debug logs a debug-level message
func (l *Logger) Debug(msg string, fields map[string]any) {
	if l.level > LevelDebug {
		return
	}
	attrs := l.buildAttrs(fields)
	if l.mode == "dev" {
		l.consoleLogger.Debug(msg, attrs...)
	} else {
		l.slogger.Debug(msg, attrs...)
	}
}

// DebugCtx logs a debug-level message with context (request ID)
func (l *Logger) DebugCtx(ctx context.Context, msg string, fields map[string]any) {
	if reqID := l.extractRequestID(ctx); reqID != "" {
		if fields == nil {
			fields = make(map[string]any)
		}
		fields[l.requestIDKey] = reqID
	}
	l.Debug(msg, fields)
}

// Info logs an info-level message
func (l *Logger) Info(msg string, fields map[string]any) {
	attrs := l.buildAttrs(fields)
	if l.mode == "dev" {
		l.consoleLogger.Info(msg, attrs...)
	} else {
		l.slogger.Info(msg, attrs...)
	}
}

// InfoCtx logs an info-level message with context (request ID)
func (l *Logger) InfoCtx(ctx context.Context, msg string, fields map[string]any) {
	if reqID := l.extractRequestID(ctx); reqID != "" {
		if fields == nil {
			fields = make(map[string]any)
		}
		fields[l.requestIDKey] = reqID
	}
	l.Info(msg, fields)
}

// Warn logs a warning-level message
func (l *Logger) Warn(msg string, fields map[string]any) {
	attrs := l.buildAttrs(fields)
	if l.mode == "dev" {
		l.consoleLogger.Warn(msg, attrs...)
	} else {
		l.slogger.Warn(msg, attrs...)
	}
}

// WarnCtx logs a warning-level message with context (request ID)
func (l *Logger) WarnCtx(ctx context.Context, msg string, fields map[string]any) {
	if reqID := l.extractRequestID(ctx); reqID != "" {
		if fields == nil {
			fields = make(map[string]any)
		}
		fields[l.requestIDKey] = reqID
	}
	l.Warn(msg, fields)
}

// Error logs an error-level message
func (l *Logger) Error(msg string, err error, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	if err != nil {
		fields["error"] = err.Error()
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	attrs := l.buildAttrs(fields)
	if l.mode == "dev" {
		l.consoleLogger.Error(msg, attrs...)
	} else {
		l.slogger.Error(msg, attrs...)
	}
}

// ErrorCtx logs an error-level message with context (request ID)
func (l *Logger) ErrorCtx(ctx context.Context, msg string, err error, fields map[string]any) {
	if reqID := l.extractRequestID(ctx); reqID != "" {
		if fields == nil {
			fields = make(map[string]any)
		}
		fields[l.requestIDKey] = reqID
	}
	l.Error(msg, err, fields)
}

// Fatal logs a fatal-level message and exits (use sparingly!)
func (l *Logger) Fatal(msg string, err error, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	if err != nil {
		fields["error"] = err.Error()
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	fields["fatal"] = true

	attrs := l.buildAttrs(fields)
	if l.mode == "dev" {
		l.consoleLogger.Error(msg, attrs...)
	} else {
		l.slogger.Error(msg, attrs...)
	}

	os.Exit(1)
}

// Startup logs a startup message with banner (special formatting)
func (l *Logger) Startup(msg string, fields map[string]any) {
	if l.mode == "dev" {
		// Pretty banner for console
		fmt.Println("═══════════════════════════════════════════════════════════")
		fmt.Printf("🚀 %s\n", msg)
		fmt.Println("═══════════════════════════════════════════════════════════")
		for k, v := range fields {
			fmt.Printf("   %s: %v\n", k, v)
		}
	} else {
		// JSON log for production
		l.Info(msg, fields)
	}
}

// Performance logs performance metrics (timing, throughput, etc.)
func (l *Logger) Performance(operation string, duration time.Duration, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["operation"] = operation
	fields["duration_ms"] = duration.Milliseconds()
	fields["duration_human"] = duration.String()
	fields["metric_type"] = "performance"

	l.Info("Performance metric", fields)
}

// Security logs security-related events (auth, access, violations)
func (l *Logger) Security(event string, success bool, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["security_event"] = event
	fields["success"] = success
	fields["metric_type"] = "security"

	level := LevelInfo
	if !success {
		level = LevelWarn
	}

	if level == LevelWarn {
		l.Warn("Security event", fields)
	} else {
		l.Info("Security event", fields)
	}
}

// Business logs business intelligence metrics (payments, orders, customers)
func (l *Logger) Business(metric string, value float64, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["business_metric"] = metric
	fields["value"] = value
	fields["metric_type"] = "business"

	l.Info("Business metric", fields)
}

// GPU logs GPU-related operations and performance
func (l *Logger) GPU(operation string, success bool, fields map[string]any) {
	if fields == nil {
		fields = make(map[string]any)
	}
	fields["gpu_operation"] = operation
	fields["success"] = success
	fields["metric_type"] = "gpu"

	if success {
		l.Info("GPU operation", fields)
	} else {
		l.Warn("GPU operation failed", fields)
	}
}

// Structured is a generic structured log for custom use cases
func (l *Logger) Structured(level LogLevel, msg string, fields map[string]any) {
	switch level {
	case LevelDebug:
		l.Debug(msg, fields)
	case LevelInfo:
		l.Info(msg, fields)
	case LevelWarn:
		l.Warn(msg, fields)
	case LevelError:
		l.Error(msg, nil, fields)
	case LevelFatal:
		l.Fatal(msg, nil, fields)
	}
}

// ToJSON converts fields to JSON string (for debugging)
func (l *Logger) ToJSON(fields map[string]any) string {
	b, err := json.Marshal(fields)
	if err != nil {
		return fmt.Sprintf("{\"error\": \"failed to marshal: %v\"}", err)
	}
	return string(b)
}

// GetStackTrace returns current stack trace (useful for debugging)
func (l *Logger) GetStackTrace(skip int) string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// WithFields creates a child logger with preset fields (useful for components)
type ComponentLogger struct {
	parent *Logger
	fields map[string]any
}

// WithFields creates a logger with preset fields
func (l *Logger) WithFields(fields map[string]any) *ComponentLogger {
	return &ComponentLogger{
		parent: l,
		fields: fields,
	}
}

// Info logs info with component fields merged
func (c *ComponentLogger) Info(msg string, additionalFields map[string]any) {
	merged := c.mergeFields(additionalFields)
	c.parent.Info(msg, merged)
}

// Error logs error with component fields merged
func (c *ComponentLogger) Error(msg string, err error, additionalFields map[string]any) {
	merged := c.mergeFields(additionalFields)
	c.parent.Error(msg, err, merged)
}

// Warn logs warning with component fields merged
func (c *ComponentLogger) Warn(msg string, additionalFields map[string]any) {
	merged := c.mergeFields(additionalFields)
	c.parent.Warn(msg, merged)
}

// Debug logs debug with component fields merged
func (c *ComponentLogger) Debug(msg string, additionalFields map[string]any) {
	merged := c.mergeFields(additionalFields)
	c.parent.Debug(msg, merged)
}

// mergeFields combines component fields with additional fields
func (c *ComponentLogger) mergeFields(additional map[string]any) map[string]any {
	merged := make(map[string]any, len(c.fields)+len(additional))
	for k, v := range c.fields {
		merged[k] = v
	}
	for k, v := range additional {
		merged[k] = v
	}
	return merged
}

// Global logger instance (initialized in main)
var AppLogger *Logger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(cfg LoggerConfig) {
	AppLogger = NewLogger(cfg)
}
