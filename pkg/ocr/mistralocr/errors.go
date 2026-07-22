// Typed errors for the Mistral OCR 4 client.
// σ: MistralOCR-Errors | ρ: pkg/ocr/mistralocr | γ: Production
package mistralocr

import "fmt"

// APIError carries the raw Mistral error envelope (see docs.mistral.ai/resources/error-glossary):
//
//	{"object":"error","message":"...","type":"invalid_request_error","param":"model","code":"unknown_model"}
//
// Classification is by HTTP status + Type, never by undocumented Code strings — the OCR
// endpoint's error codes are not enumerated in the public docs as of the 2026-07-22 A0 pass.
type APIError struct {
	StatusCode int
	Type       string
	Code       string
	Param      string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("mistralocr: %s (status %d, type=%s, code=%s)", e.Message, e.StatusCode, e.Type, e.Code)
	}
	return fmt.Sprintf("mistralocr: %s (status %d, type=%s)", e.Message, e.StatusCode, e.Type)
}

// AuthError: invalid/missing API key (HTTP 401/403).
type AuthError struct{ *APIError }

// QuotaError: rate limit or quota exhaustion (HTTP 429).
type QuotaError struct{ *APIError }

// TooLargeError: document exceeds size/page limits (HTTP 413, or 422 with a size-shaped message).
type TooLargeError struct{ *APIError }

// SchemaMismatchError: the supplied annotation JSON schema was rejected or could not be
// satisfied by the model (HTTP 400/422 while a DocumentSchema was in the request).
type SchemaMismatchError struct{ *APIError }

// classifyError builds the narrowest typed error the status/envelope supports, defaulting to
// the plain *APIError when nothing more specific applies.
func classifyError(statusCode int, env errorEnvelope, hadSchema bool) error {
	base := &APIError{
		StatusCode: statusCode,
		Type:       env.Type,
		Code:       env.Code,
		Param:      env.Param,
		Message:    env.Message,
	}

	switch statusCode {
	case 401, 403:
		return &AuthError{base}
	case 429:
		return &QuotaError{base}
	case 413:
		return &TooLargeError{base}
	}

	if statusCode == 400 || statusCode == 422 {
		if looksLikeTooLarge(env) {
			return &TooLargeError{base}
		}
		if hadSchema {
			return &SchemaMismatchError{base}
		}
	}

	return base
}

func looksLikeTooLarge(env errorEnvelope) bool {
	needle := env.Code + " " + env.Message
	for _, marker := range []string{"too_large", "too large", "file_size", "page_limit", "max_pages", "exceeds"} {
		if containsFold(needle, marker) {
			return true
		}
	}
	return false
}
