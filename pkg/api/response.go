package api

import (
	"time"
)

// APIResponse is the standardized response format for ALL API endpoints
// MANDATORY VERSION FIELD (Bezos Mandate Violation #4 Fix!)
type APIResponse struct {
	Version string    `json:"version"`         // API version (REQUIRED!)
	Data    any       `json:"data,omitempty"`  // Successful response data
	Error   *APIError `json:"error,omitempty"` // Error details (if any)
	Meta    *APIMeta  `json:"meta,omitempty"`  // Request metadata
}

// APIError provides structured error information
type APIError struct {
	Code    string `json:"code"`              // Error code (e.g., "SCREEN_NOT_FOUND")
	Message string `json:"message"`           // Human-readable error message
	Details string `json:"details,omitempty"` // Additional error context
}

// APIMeta provides request metadata
type APIMeta struct {
	RequestID string `json:"request_id"`         // Unique request ID (from middleware)
	Timestamp int64  `json:"timestamp"`          // Unix timestamp
	Duration  int64  `json:"duration,omitempty"` // Request duration in milliseconds
}

// Success creates a successful API response
func Success(data any) APIResponse {
	return APIResponse{
		Version: "v1",
		Data:    data,
		Meta: &APIMeta{
			Timestamp: time.Now().Unix(),
		},
	}
}

// SuccessWithMeta creates a successful response with custom metadata
func SuccessWithMeta(data any, requestID string, duration int64) APIResponse {
	return APIResponse{
		Version: "v1",
		Data:    data,
		Meta: &APIMeta{
			RequestID: requestID,
			Timestamp: time.Now().Unix(),
			Duration:  duration,
		},
	}
}

// Error creates an error API response
func Error(code, message string) APIResponse {
	return APIResponse{
		Version: "v1",
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Meta: &APIMeta{
			Timestamp: time.Now().Unix(),
		},
	}
}

// ErrorWithDetails creates an error response with additional context
func ErrorWithDetails(code, message, details string) APIResponse {
	return APIResponse{
		Version: "v1",
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Meta: &APIMeta{
			Timestamp: time.Now().Unix(),
		},
	}
}
