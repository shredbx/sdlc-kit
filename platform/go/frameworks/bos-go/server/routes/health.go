// Package routes holds the routes every bos API serves: the health check and the root info.
package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/core/config"
)

// HealthResponse is the standard healthcheck response.
type HealthResponse struct {
	Status      string `json:"status"`
	Service     string `json:"service"`
	Version     string `json:"version"`
	Environment string `json:"environment,omitempty"`
	Database    string `json:"database,omitempty"`
}

// Pinger is what the health check asks of a database: one round trip. A pgx pool
// satisfies it. Pass a nil interface, not a nil pool, when the app has no database.
type Pinger interface {
	Ping(ctx context.Context) error
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Health serves GET /health.
func Health(cfg config.Config, db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "disconnected"
		if db != nil {
			pingCtx, pingCancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer pingCancel()
			if err := db.Ping(pingCtx); err == nil {
				dbStatus = "connected"
			}
		}
		writeJSON(w, http.StatusOK, HealthResponse{
			Status:      "healthy",
			Service:     cfg.AppName,
			Version:     cfg.AppVersion,
			Environment: cfg.Environment,
			Database:    dbStatus,
		})
	}
}

// HealthHead serves HEAD /health.
func HealthHead(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
