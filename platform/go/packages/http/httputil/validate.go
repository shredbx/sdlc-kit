package httputil

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/shredbx/sbx-core/pkg/money"
)

// globalValidator is the singleton *validator.Validate used across the
// workspace. Consumers register domain-specific tags at boot:
//
//	httputil.Validator().RegisterValidation("strong_password", func(fl validator.FieldLevel) bool {
//	    return auth.ValidatePassword(fl.Field().String()) == nil
//	})
//
// The validator is configured to report field names using the `json` struct
// tag, so FieldError.Field() returns e.g. "image_url" not "ImageURL".
var globalValidator *validator.Validate

func init() {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})

	// currencycode is the single, registry-backed currency-validity rule shared by
	// every consumer of the global validator (e.g. property.Property,
	// PropertyUnit request bodies). Valid when EMPTY (pair with omitempty — an
	// absent code inherits a default) OR a code the governed currency registry
	// knows after normalization. Replaces the former hand-maintained `len=3`
	// length check so there is ONE source of currency validity (pkg/money).
	v.RegisterValidation("currencycode", func(fl validator.FieldLevel) bool {
		s := fl.Field().String()
		if s == "" {
			return true
		}
		return money.NormalizeCurrencyCode(s).Valid()
	})

	globalValidator = v
}

// Validator returns the singleton validator instance. Use to register custom
// validation tags at boot.
func Validator() *validator.Validate {
	return globalValidator
}

// DecodeAndValidate parses the JSON request body into T, enforces the body
// size limit, and runs struct-tag validation. On any failure it writes an
// appropriate error response and returns ok=false. On success returns the
// populated struct and ok=true.
//
// maxBytes=0 means "no size limit at this call site" (rely on middleware).
// The decoder rejects unknown fields by default — clients sending extra keys
// receive 400.
func DecodeAndValidate[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (*T, bool) {
	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	}

	var out T
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&out); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, "request body too large", CodePayloadTooLarge)
			return nil, false
		}
		if errors.Is(err, io.EOF) {
			WriteError(w, http.StatusBadRequest, "request body is empty", CodeInvalidJSON)
			return nil, false
		}
		WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error(), CodeInvalidJSON)
		return nil, false
	}

	// Reject trailing data: a well-formed request must contain exactly one
	// JSON value. {"a":1}{"b":2} or {"a":1}garbage should be rejected.
	if dec.More() {
		WriteError(w, http.StatusBadRequest, "request body contains more than one JSON value", CodeInvalidJSON)
		return nil, false
	}

	if err := globalValidator.Struct(&out); err != nil {
		WriteValidationError(w, err)
		return nil, false
	}
	return &out, true
}
