package httputil

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// codePattern matches the dictionary code grammar used across the workspace:
// lowercase letters, digits, hyphen, underscore. Length is enforced separately
// via maxLen so callers can pick a per-field cap.
var codePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// ParseUUID validates raw as a UUID. fieldName is used in the error response
// so the caller can pass "id", "user_id", etc.
//
// Caller pattern (router-agnostic):
//
//	id, ok := httputil.ParseUUID(w, chi.URLParam(r, "id"), "id")
//	if !ok { return }
func ParseUUID(w http.ResponseWriter, raw, fieldName string) (uuid.UUID, bool) {
	if raw == "" {
		WriteError(w, http.StatusBadRequest, fieldName+" is required", CodeInvalidParam)
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fieldName+" must be a valid UUID", CodeInvalidParam)
		return uuid.Nil, false
	}
	return id, true
}

// ParseCode validates raw as a dictionary code (lowercase alnum + hyphen +
// underscore, bounded length). maxLen=0 disables the length check.
func ParseCode(w http.ResponseWriter, raw, fieldName string, maxLen int) (string, bool) {
	if raw == "" {
		WriteError(w, http.StatusBadRequest, fieldName+" is required", CodeInvalidParam)
		return "", false
	}
	if maxLen > 0 && len(raw) > maxLen {
		WriteError(w, http.StatusBadRequest, fieldName+" exceeds max length", CodeInvalidParam)
		return "", false
	}
	if !codePattern.MatchString(raw) {
		WriteError(w, http.StatusBadRequest, fieldName+" contains invalid characters (allowed: a-z 0-9 - _)", CodeInvalidParam)
		return "", false
	}
	return raw, true
}

// ParseCodeList splits a comma-separated coded param, trims + skips blank
// segments, and ParseCode-validates each surviving code (writing a 400 and
// returning ok=false on a grammar failure). An empty / all-blank value yields
// no codes with ok=true — "optional multi-select" needs no separate code path.
// Extracted from the BR catalog filter parser so every multi-code param (the
// public property_type pill rail, the admin extended filter) shares one grammar.
//
// maxItems caps how many codes one param may carry (surviving codes beyond it
// are a 400, never silently dropped) — an unbounded comma list on a PUBLIC
// endpoint would otherwise become an unbounded SQL IN parameter set (2607-004
// review finding S1). Callers pass a small multiple of the real dictionary size.
func ParseCodeList(w http.ResponseWriter, raw, fieldName string, maxLen, maxItems int) ([]string, bool) {
	if raw == "" {
		return nil, true
	}
	var codes []string
	for _, c := range strings.Split(raw, ",") {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if _, ok := ParseCode(w, c, fieldName, maxLen); !ok {
			return nil, false
		}
		codes = append(codes, c)
		if len(codes) > maxItems {
			WriteError(w, http.StatusBadRequest,
				fieldName+" carries too many values (max "+strconv.Itoa(maxItems)+")", CodeInvalidParam)
			return nil, false
		}
	}
	return codes, true
}

// ParseEnum returns one of the allowed values, or writes 400 and returns
// ok=false. T is constrained to ~string so it works with named string enums
// like ImagePurpose, PropertyType, etc.
//
// Caller pattern:
//
//	purpose, ok := httputil.ParseEnum(w, r.FormValue("purpose"), "purpose",
//	    []ImagePurpose{ImagePurposeProperty, ImagePurposeAgent})
//	if !ok { return }
func ParseEnum[T ~string](w http.ResponseWriter, raw, fieldName string, allowed []T) (T, bool) {
	var zero T
	if raw == "" {
		WriteError(w, http.StatusBadRequest, fieldName+" is required", CodeInvalidParam)
		return zero, false
	}
	for _, v := range allowed {
		if string(v) == raw {
			return v, true
		}
	}
	WriteError(w, http.StatusBadRequest, fieldName+" is not an allowed value", CodeInvalidParam)
	return zero, false
}

// ParseIntRange parses raw as an integer in [min, max] inclusive. Empty input
// returns defaultVal with ok=true so callers can express "optional with
// default" without a separate code path.
func ParseIntRange(w http.ResponseWriter, raw, fieldName string, min, max, defaultVal int) (int, bool) {
	if raw == "" {
		return defaultVal, true
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fieldName+" must be a number", CodeInvalidParam)
		return 0, false
	}
	if n < min || n > max {
		WriteError(w, http.StatusBadRequest, fieldName+" is out of range", CodeInvalidParam)
		return 0, false
	}
	return n, true
}

// ParseBool parses raw as a boolean. Accepts "true", "false", "1", "0", "yes",
// "no" (case-insensitive). Empty returns defaultVal with ok=true.
func ParseBool(w http.ResponseWriter, raw, fieldName string, defaultVal bool) (bool, bool) {
	if raw == "" {
		return defaultVal, true
	}
	switch raw {
	case "true", "True", "TRUE", "1", "yes", "Yes", "YES":
		return true, true
	case "false", "False", "FALSE", "0", "no", "No", "NO":
		return false, true
	default:
		WriteError(w, http.StatusBadRequest, fieldName+" must be true or false", CodeInvalidParam)
		return false, false
	}
}
