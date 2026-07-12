// Package apperr carries coded operational errors: a stable error code,
// a human message, and optional detail. The string format matches the
// host application's historical error contract ("[CODE] Message: details"),
// so logic moved into pkg/ keeps returning byte-identical errors to the
// UI. Kernel-pure: no dependencies, no domain vocabulary.
package apperr

import "fmt"

// Error is a coded operational error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// New constructs a coded error.
func New(code, message, details string) *Error {
	return &Error{Code: code, Message: message, Details: details}
}
