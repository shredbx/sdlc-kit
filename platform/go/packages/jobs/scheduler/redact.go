package scheduler

import "regexp"

// DefaultMaxOutputBytes is the cap on a persisted run output. A run's log can be
// arbitrarily large (a streaming backup pipe), so FinishRun keeps only the TAIL up
// to this many bytes — the recent lines are what matter for a post-mortem, and an
// unbounded blob is both a storage and a DoS concern. 8 KiB comfortably holds the
// closing summary + error context of any in-tree job.
const DefaultMaxOutputBytes = 8 * 1024

// truncationMarker prefixes a bounded output when content was cut, so a reader
// knows the head is missing (the tail is what survived).
const truncationMarker = "…[truncated]"

var (
	// reDBPassword masks the password in a URL userinfo (scheme://user:PASSWORD@host).
	// It keeps the scheme + user + host so the connection target stays legible. The
	// password class excludes only '@' (the host delimiter) and whitespace — generated
	// Postgres passwords routinely contain '/' and '+', which MUST still be masked.
	reDBPassword = regexp.MustCompile(`(://[^:/@\s]+:)[^@\s]+(@)`)

	// reSigQuery masks the VALUE of any signature/credential query parameter
	// (presigned URL sigs: X-Amz-Signature, X-Amz-Credential, X-Amz-Security-Token,
	// and the generic sig=/signature=). Keeps the key so the shape is still readable.
	reSigQuery = regexp.MustCompile(`(?i)([?&](?:x-amz-signature|x-amz-credential|x-amz-security-token|signature|sig)=)[^&\s]+`)

	// reSigHeader masks the VALUE of a signature/credential HTTP HEADER (the colon
	// form tools log with -v/-vv: "Authorization: …", "X-Amz-Security-Token: …",
	// "X-Amz-Signature: …"). The whole rest of the line is the sensitive value, so it
	// is masked to end-of-line. Distinct from reSigQuery (the '?key=value' form) so a
	// query string's trailing params are not over-redacted.
	reSigHeader = regexp.MustCompile(`(?i)(\b(?:authorization|x-amz-security-token|x-amz-signature|x-amz-credential)\s*:\s*)[^\r\n]+`)

	// reBearer masks a bare bearer token not behind an Authorization: header.
	reBearer = regexp.MustCompile(`(?i)(\bbearer\s+)[A-Za-z0-9._~+/\-]+=*`)

	// reSecretAssign masks the VALUE of a secret-looking KEY=VALUE assignment:
	// RCLONE_CONFIG_*, anything containing SECRET, or *_TOKEN / *_PASSWORD / *_KEY
	// (access/secret keys, tokens). Keeps the key name so the log still says WHICH
	// secret without leaking it.
	reSecretAssign = regexp.MustCompile(`(?i)\b((?:RCLONE_CONFIG_[A-Z0-9_]*|[A-Z0-9_]*(?:SECRET|TOKEN|PASSWORD)[A-Z0-9_]*|[A-Z0-9_]*ACCESS_KEY[A-Z0-9_]*)=)\S+`)
)

// Redact strips secrets from a run's output before it is persisted:
//   - URL userinfo passwords (postgres://user:pw@ → postgres://user:***@)
//   - presigned-URL / signature query-param values (X-Amz-*, sig=, signature=)
//   - secret-looking KEY=VALUE env assignments (RCLONE_CONFIG_*, *SECRET*, *TOKEN*,
//     *PASSWORD*, *ACCESS_KEY*)
//
// It is conservative — it masks the VALUE and keeps the KEY/host so the log stays
// diagnosable — and idempotent (re-redacting redacted text is a no-op).
func Redact(s string) string {
	s = reDBPassword.ReplaceAllString(s, "${1}***${2}")
	s = reSigQuery.ReplaceAllString(s, "${1}***")
	s = reSigHeader.ReplaceAllString(s, "${1}***")
	s = reBearer.ReplaceAllString(s, "${1}***")
	s = reSecretAssign.ReplaceAllString(s, "${1}***")
	return s
}

// BoundOutput caps s to at most maxBytes, keeping the TAIL (the most recent
// content) and prefixing a truncation marker when it cuts. A non-positive
// maxBytes defaults to DefaultMaxOutputBytes (defensive — a caller bug must not
// produce an empty or panicking result). When maxBytes is smaller than the marker
// itself, the marker is itself truncated to stay within the cap.
func BoundOutput(s string, maxBytes int) string {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxOutputBytes
	}
	if len(s) <= maxBytes {
		return s
	}
	// Reserve room for the marker, then keep that many trailing bytes.
	keep := maxBytes - len(truncationMarker)
	if keep <= 0 {
		// Cap is tighter than the marker — return a truncated marker only.
		return truncationMarker[:maxBytes]
	}
	return truncationMarker + s[len(s)-keep:]
}
