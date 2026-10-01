package visitoractivity

import "regexp"

// schemaRe enforces PostgreSQL identifier rules for schema names: a lowercase
// letter or underscore, followed by up to 62 alphanumerics/underscores.
//
// This is co-located (rather than importing pkg/auth.MustValidateSchema) to
// avoid a cross-package dependency — the rule is small and identical. It is the
// SQL-injection guard for the schema parameter that gets interpolated into the
// store's parameterized queries (the schema is an identifier, so it cannot be a
// bound parameter; the allowlist is the defence).
var schemaRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// mustValidateSchema panics if s is not a valid PostgreSQL identifier.
// Constructors call this at init time so a misconfigured schema fails loudly at
// startup instead of corrupting queries at runtime.
func mustValidateSchema(s string) {
	if !schemaRe.MatchString(s) {
		panic("visitoractivity: invalid schema name: " + s)
	}
}
