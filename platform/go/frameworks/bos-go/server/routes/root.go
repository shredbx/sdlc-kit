package routes

import (
	"net/http"

	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/core/config"
)

// Root serves GET /: the service name, version, environment and the endpoints the
// consumer lists. With no list it names only /health.
func Root(cfg config.Config, endpoints []string) http.HandlerFunc {
	if len(endpoints) == 0 {
		endpoints = []string{"/health"}
	}
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"service":     cfg.AppName,
			"version":     cfg.AppVersion,
			"environment": cfg.Environment,
			"endpoints":   endpoints,
		})
	}
}
