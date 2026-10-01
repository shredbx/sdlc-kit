// Package server builds the HTTP app: the fixed middleware chain, the health and root
// routes, and the /api group where a consumer mounts its own routes.
package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	pkgauth "github.com/shredbx/sbx-core/pkg/auth"

	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/core/config"
	bosmiddleware "github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/server/middleware"
	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/server/routes"
)

// Deps is what a consumer hands the app. Every field is optional.
type Deps struct {
	// DB answers the health check's database ping. A nil interface reports "disconnected".
	DB routes.Pinger
	// Auth, when set, adds the AuthExtract middleware to the chain.
	Auth pkgauth.AuthService
	// API mounts the consumer's routes on the /api group, which already has the CSRF check.
	API func(r chi.Router)
	// Endpoints is the list the root route reports.
	Endpoints []string
}

// NewApp builds the router.
func NewApp(cfg config.Config, deps Deps) http.Handler {
	r := chi.NewRouter()

	// L6: RequestID FIRST so panic logs (and access logs) carry the correlation ID.
	// Order: RequestID → Logger → JSONRecoverer → SecurityHeaders → CORS → AuthExtract.
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(bosmiddleware.JSONRecoverer)
	r.Use(pkgauth.SecurityHeaders(pkgauth.DefaultSecurityHeadersConfig(cfg.Environment == "production")))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "X-Request-ID", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	if deps.Auth != nil {
		r.Use(pkgauth.AuthExtract(deps.Auth))
	}

	// Health check
	r.Get("/health", routes.Health(cfg, deps.DB))
	r.Head("/health", routes.HealthHead)

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(pkgauth.CSRFCheck)
		if deps.API != nil {
			deps.API(r)
			return
		}
		// A chi group with no routes never runs its middleware. This catch-all, only for
		// the empty group, keeps the CSRF check in front of unknown paths, as it is in an
		// app that has routes; its answer is chi's own not-found.
		r.HandleFunc("/*", http.NotFound)
	})

	// Root info
	r.Get("/", routes.Root(cfg, deps.Endpoints))

	return r
}

// Run serves h on cfg.Port until SIGINT or SIGTERM, then drains in-flight requests.
func Run(cfg config.Config, h http.Handler) {
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: h}
	go func() {
		log.Printf("Starting %s v%s on port %s (%s)", cfg.AppName, cfg.AppVersion, cfg.Port, cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown: on SIGINT/SIGTERM stop accepting new requests and drain
	// in-flight ones within a bounded timeout.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Printf("Shutdown signal received — draining...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
	log.Printf("Shutdown complete")
}
