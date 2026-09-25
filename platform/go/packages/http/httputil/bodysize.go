package httputil

import (
	"net/http"
	"strings"
)

// BodySizeLimit returns middleware that caps request body size to maxBytes
// using http.MaxBytesReader. When the limit is exceeded, the next read from
// r.Body returns *http.MaxBytesError, which DecodeAndValidate translates to
// 413 Payload Too Large.
//
// Recommended workspace-wide policy:
//   - 1MB default for all JSON endpoints (auth, dictionary, view-model APIs)
//   - 10MB override for single-file upload endpoints (image/asset upload)
//
// Multi-image upload uses N sequential 10MB-capped requests, not one giant
// multipart payload, so the per-request cap stays the per-file cap.
//
// Multipart requests SKIP this cap: a multipart upload carries its own
// handler-level MaxBytesReader on the file field (the 10MB per-file ceiling),
// and a route-group JSON cap must NOT pre-wrap a multipart body — the smaller
// group cap would fire first and silently shrink the upload ceiling. Concrete
// case (task 2607-008): /manage/properties' 1MB JSON cap (NFR-001) blocked the
// 10MB gallery upload because this middleware pre-wrapped the multipart body at
// 1MB before gallery.UploadImage's own 10MB MaxBytesReader ran. Let the handler
// cap multipart; this middleware only caps JSON/Application bodies.
func BodySizeLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}
