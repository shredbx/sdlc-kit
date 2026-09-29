// Command api is the API of bos-demo, the bos product's own reference build, on the bos Go framework.
package main

import (
	"context"
	"log"

	"github.com/go-chi/chi/v5"
	pkgauth "github.com/shredbx/sbx-core/pkg/auth"
	"github.com/shredbx/sbx-core/pkg/cms"
	"github.com/shredbx/sbx-core/pkg/database"

	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/authfake"
	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/core/config"
	"github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/server"
	bosmw "github.com/shredbx/sdlc-kit/platform/go/frameworks/bos-go/server/middleware"

	"github.com/shredbx/sdlc-kit/projects/demo/bos-demo/apps/api/internal/settings"
)

func main() {
	cfg := config.Load("BOSDEMO", config.Defaults{
		AppName: "bos-demo-api",
		Port:    "5010",
		BaseURL: "http://localhost:4010",
	})

	var deps server.Deps
	var cmsRepo *cms.Repository
	var settingsStore *settings.Store

	// database_url empty means no database: /health still reports it as disconnected (bos-go's
	// own M0 behavior, unchanged) and the admin/content routes below simply do not mount.
	if cfg.DatabaseURL != "" {
		ctx := context.Background()
		// Schema "public": bos-demo has its own dedicated bos-postgres database (not a shared
		// multi-project instance), so the package's schema-per-project isolation is not needed —
		// EnsureSchema still requires a non-empty name to run CREATE SCHEMA IF NOT EXISTS.
		db, err := database.New(ctx, database.Config{URL: cfg.DatabaseURL, Schema: "public"})
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		if _, err := database.Migrate(ctx, db, "migrations"); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		// The cms kit ships its own migrations (embedded — platform/CLAUDE.md D15:
		// a kit "later also carries its bos wiring... migrations"), applied against
		// the SAME schema_migrations table via a separate fs.FS-backed call. See
		// MigrateFS's own doc comment for why a kit's migration versions must never
		// collide with this app's own "00N" ones (they're long timestamps).
		if _, err := database.MigrateFS(ctx, db, cms.MigrationsFS, "migrations"); err != nil {
			log.Fatalf("migrate (cms kit): %v", err)
		}
		deps.DB = db.Pool()
		cmsRepo = cms.NewRepository(db, "public")
		settingsStore = settings.NewStore(db)
	}

	// authfake preserves the RBAC seam (RequireAuth, Claims.Role) before real authentication
	// (roadmap M6): swapping it for the real AuthService later touches only this line.
	authSvc := authfake.New("admin")

	deps.API = func(r chi.Router) {
		if cmsRepo != nil {
			cms.RegisterPublic(r, cmsRepo)
		}
		if settingsStore != nil {
			settings.RegisterPublic(r, settingsStore)
		}
		r.Route("/admin", func(r chi.Router) {
			r.Use(authfake.Always(authSvc))
			r.Use(pkgauth.RequireAuth)
			r.Get("/whoami", bosmw.WhoAmI)
			// CMS pages: both admin and content-manager reach these (RequireAuth above is
			// enough — any authenticated role manages content).
			if cmsRepo != nil {
				cms.RegisterAdmin(r, cmsRepo)
			}
			// Site configuration: administrators only (docs/proposals/bos-site-chrome.md's
			// content-manager vs administrator split).
			if settingsStore != nil {
				r.Group(func(r chi.Router) {
					r.Use(bosmw.RequireRole("admin"))
					settings.RegisterAdmin(r, settingsStore)
				})
			}
		})
	}

	server.Run(cfg, server.NewApp(cfg, deps))
}
