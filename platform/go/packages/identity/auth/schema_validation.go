package auth

import (
	"fmt"
	"regexp"
)

// schemaNamePattern enforces PostgreSQL identifier rules for schema names:
// lowercase letter or underscore, followed by up to 62 alphanumerics/underscores.
// M1: prevents SQL injection if a future code path lets user input flow into
// the schema parameter of any NewPostgres* constructor.
var schemaNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// MustValidateSchema panics with a clear message if schema doesn't match the
// allowlist pattern. Constructors call this at init time, so a misconfigured
// schema fails loudly at startup instead of corrupting queries at runtime.
func MustValidateSchema(schema string) {
	if !schemaNamePattern.MatchString(schema) {
		panic(fmt.Sprintf("auth: invalid schema name %q — must match %s", schema, schemaNamePattern.String()))
	}
}
