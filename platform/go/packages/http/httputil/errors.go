// Package httputil provides HTTP-boundary validation helpers shared across
// every Go application in the workspace. It implements NFR-001 (input
// validation at every system boundary) via:
//   - DecodeAndValidate[T]: JSON body decode + struct-tag validation + size limit
//   - Router-agnostic param parsers (ParseUUID, ParseCode, ParseEnum, ParseIntRange)
//   - BodySizeLimit middleware
//   - Canonical ErrorResponse shape ({error, code, details[]})
//
// Backwards-compatible: legacy clients that only read the `error` field still
// work; new clients use `code` and `details[]` for machine-actionable errors.
package httputil

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// ErrorResponse is the canonical JSON shape returned by every handler error
// path in the workspace.
type ErrorResponse struct {
	Error   string       `json:"error"`
	Code    string       `json:"code,omitempty"`
	Details []FieldError `json:"details,omitempty"`
}

// FieldError describes a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag,omitempty"`
	Message string `json:"message"`
}

// Machine codes returned via ErrorResponse.Code. Stable identifiers — safe to
// pin against in client code.
const (
	CodeInvalidJSON      = "invalid_json"
	CodePayloadTooLarge  = "payload_too_large"
	CodeValidationFailed = "validation_failed"
	CodeInvalidParam     = "invalid_param"
	CodeNotFound         = "not_found"
	CodeUnauthorized     = "unauthorized"
	CodeForbidden        = "forbidden"
	CodeConflict         = "conflict"
	CodeInternal         = "internal"
)

// WriteError writes a JSON error response with the given status, human
// message, and machine code.
func WriteError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, ErrorResponse{Error: message, Code: code})
}

// WriteValidationError writes 422 Unprocessable Entity with field-level
// details extracted from validator.ValidationErrors. Falls back to a generic
// 422 if err is not a *validator.ValidationErrors.
func WriteValidationError(w http.ResponseWriter, err error) {
	var verr validator.ValidationErrors
	if !errors.As(err, &verr) {
		WriteError(w, http.StatusUnprocessableEntity, "validation failed", CodeValidationFailed)
		return
	}
	details := make([]FieldError, 0, len(verr))
	for _, fe := range verr {
		details = append(details, FieldError{
			Field:   fe.Field(),
			Tag:     fe.Tag(),
			Message: fieldErrorMessage(fe),
		})
	}
	writeJSON(w, http.StatusUnprocessableEntity, ErrorResponse{
		Error:   "validation failed",
		Code:    CodeValidationFailed,
		Details: details,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("httputil: encode response", "error", err)
	}
}

func fieldErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "min":
		return fe.Field() + " must be at least " + fe.Param()
	case "max":
		return fe.Field() + " must be at most " + fe.Param()
	case "len":
		return fe.Field() + " must be exactly " + fe.Param() + " characters"
	case "email":
		return fe.Field() + " must be a valid email"
	case "uuid", "uuid4":
		return fe.Field() + " must be a valid UUID"
	case "oneof":
		return fe.Field() + " must be one of: " + fe.Param()
	case "gte":
		return fe.Field() + " must be >= " + fe.Param()
	case "lte":
		return fe.Field() + " must be <= " + fe.Param()
	case "gt":
		return fe.Field() + " must be > " + fe.Param()
	case "lt":
		return fe.Field() + " must be < " + fe.Param()
	case "url":
		return fe.Field() + " must be a valid URL"
	case "dive":
		return fe.Field() + " has an invalid element"
	default:
		return fe.Field() + " is invalid (" + fe.Tag() + ")"
	}
}
