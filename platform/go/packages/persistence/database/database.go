// Package database provides PostgreSQL adapter with connection pooling and schema isolation.
//
// This package implements the database package specification from
// .sbx/workspace/packages/core/go/database/package.yml
//
// Key Features:
//   - Connection pooling with pgxpool
//   - Schema-per-project isolation (hub, shredbx, bestays)
//   - Transaction support with auto-commit/rollback
//   - Migration runner
//   - Audit logging to shared.audit_log
//
// Example:
//
//	cfg := database.Config{
//	    URL:    "postgres://hub:password@localhost:54320/workspace",
//	    Schema: "hub",
//	}
//	db, err := database.New(ctx, cfg)
//	defer db.Close()
package database

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Common errors
var (
	ErrConnectionFailed  = errors.New("database connection failed")
	ErrMigrationFailed   = errors.New("migration failed")
	ErrTransactionFailed = errors.New("transaction failed")
	ErrAuditLogFailed    = errors.New("audit log failed")
)

// Config holds database configuration.
type Config struct {
	URL      string // PostgreSQL connection URL
	Schema   string // Project schema name (hub, shredbx, bestays)
	MaxConns int32  // Maximum pool connections (default: 10)
	MinConns int32  // Minimum idle connections (default: 2)
}

// DB is the database connection wrapper with pool and schema context.
type DB struct {
	pool   *pgxpool.Pool
	schema string
	config Config
}

// Migration represents a migration file.
type Migration struct {
	Version  string // Migration version (e.g., "001", "002")
	Name     string // Migration description
	Up       string // SQL to apply migration
	Down     string // SQL to rollback migration
	FilePath string // Absolute path to the .up.sql or .sql file
}

// MigrationStatus represents the combined state of a migration (file + DB).
type MigrationStatus struct {
	Version    string     `json:"version"`
	Name       string     `json:"name"`
	State      string     `json:"state"`
	AppliedAt  *time.Time `json:"applied_at,omitempty"`
	Checksum   string     `json:"checksum,omitempty"`
	DurationMs *int       `json:"duration_ms,omitempty"`
	Dirty      bool       `json:"dirty"`
}

// =============================================================================
// CONNECTION
// =============================================================================

// New creates a new database connection with pool.
func New(ctx context.Context, cfg Config) (*DB, error) {
	// Apply defaults
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 10
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = 2
	}

	// Parse config
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Set pool limits
	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns

	// Set search_path for schema isolation
	if cfg.Schema != "" {
		poolCfg.ConnConfig.RuntimeParams["search_path"] = cfg.Schema + ",public,shared"
	}

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	return &DB{
		pool:   pool,
		schema: cfg.Schema,
		config: cfg,
	}, nil
}

// WrapPool wraps an existing pgxpool.Pool into a DB. Use this when a consumer
// already has a *pgxpool.Pool (e.g., via main.go connection logic) and wants
// to use pkg/database helpers without re-opening the connection. The wrapped
// DB does not own the pool — Close() will close the underlying pool.
func WrapPool(pool *pgxpool.Pool, schema string) *DB {
	return &DB{
		pool:   pool,
		schema: schema,
	}
}

// Close closes all connections in the pool.
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Pool returns the underlying connection pool.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

// Schema returns the current schema.
func (db *DB) Schema() string {
	return db.schema
}

// =============================================================================
// QUERY EXECUTION
// =============================================================================

// Query executes a query and returns rows.
func (db *DB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

// QueryRow executes a query and returns a single row.
func (db *DB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}

// Exec executes a statement (INSERT, UPDATE, DELETE).
func (db *DB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// =============================================================================
// TRANSACTIONS
// =============================================================================

// Begin starts a new transaction.
func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

// WithTx executes a function within a transaction.
// Automatically commits on success or rolls back on error.
func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%w: begin: %v", ErrTransactionFailed, err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("%w: rollback after error: %v (original: %v)", ErrTransactionFailed, rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%w: commit: %v", ErrTransactionFailed, err)
	}

	return nil
}

// =============================================================================
// MIGRATIONS
// =============================================================================

// Migrate runs all pending migrations from the given directory.
// Uses pg_advisory_lock to prevent concurrent migration runs.
func Migrate(ctx context.Context, db *DB, migrationsPath string) ([]Migration, error) {
	// Acquire advisory lock
	lockID := advisoryLockID(db.schema)
	if _, err := db.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, fmt.Errorf("%w: acquire advisory lock: %v", ErrMigrationFailed, err)
	}
	defer db.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockID)

	// The engine guarantees its own schema before anything else. Without this, a
	// database whose volume was initialized while broken (initdb is empty-volume-only)
	// has no schema, and the unqualified CREATE TABLE below would silently land
	// schema_migrations in `public` (postgres skips missing search_path entries) —
	// stranding migration tracking outside every schema-scoped dump and check.
	if err := db.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("%w: ensure schema %s: %v", ErrMigrationFailed, db.schema, err)
	}

	// Ensure migrations table exists
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return nil, err
	}

	// Load migration files
	migrations, err := loadMigrations(migrationsPath)
	if err != nil {
		return nil, err
	}

	// Apply pending migrations
	var appliedMigrations []Migration
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		if err := applyMigration(ctx, db, m); err != nil {
			return appliedMigrations, fmt.Errorf("%w: version %s: %v", ErrMigrationFailed, m.Version, err)
		}
		appliedMigrations = append(appliedMigrations, m)
	}

	return appliedMigrations, nil
}

// MigrateFS runs all pending migrations from an fs.FS rooted at subdir — for a kit
// that ships its own migrations embedded in its binary (embed.FS), so applying them
// never depends on the source tree being present at runtime (a consumer's Docker
// bundle image copies only the built binary, not the kit's source folder; a plain
// OS path would silently find nothing there). platform/CLAUDE.md D15: a kit "later
// also carries its bos wiring (handlers, migrations, routes, admin pages)" — this is
// the migrations half of that. Same locking/tracking as Migrate, against the SAME
// schema_migrations table: a kit's own migrations and the consumer's own coexist
// there, so a kit's versions must never collide with a consumer's (or another
// kit's) — this is why loadMigrationsFS expects long timestamp versions
// (YYYYMMDDHHmmss, matching the real source's own convention) rather than a
// consumer's short sequential ones. Added as a new function rather than
// generalizing Migrate itself, to carry zero risk to the already-proven path-based
// flow every existing caller (and its tests) depends on.
func MigrateFS(ctx context.Context, db *DB, fsys fs.FS, subdir string) ([]Migration, error) {
	lockID := advisoryLockID(db.schema)
	if _, err := db.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, fmt.Errorf("%w: acquire advisory lock: %v", ErrMigrationFailed, err)
	}
	defer db.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockID)

	if err := db.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("%w: ensure schema %s: %v", ErrMigrationFailed, db.schema, err)
	}
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}

	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return nil, err
	}

	migrations, err := loadMigrationsFS(fsys, subdir)
	if err != nil {
		return nil, err
	}

	var appliedMigrations []Migration
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if err := applyMigration(ctx, db, m); err != nil {
			return appliedMigrations, fmt.Errorf("%w: version %s: %v", ErrMigrationFailed, m.Version, err)
		}
		appliedMigrations = append(appliedMigrations, m)
	}
	return appliedMigrations, nil
}

// loadMigrationsFS reads migration files from an fs.FS, rooted at subdir. Supports
// exactly one naming pattern (NNN_description.up.sql, optionally paired with a
// .down.sql) — a kit's migrations are authored fresh, not inherited from a legacy
// tool, so the two backward-compat patterns loadMigrations also accepts do not
// apply here. checksum is left empty for an FS-sourced migration (applyMigration's
// os.ReadFile(m.FilePath) call cannot resolve an fs.FS-relative path to a real OS
// file, and fails silently into an empty checksum by design — see applyMigration);
// this only weakens dirty-migration detection for kit-shipped files, it does not
// affect whether they apply correctly.
func loadMigrationsFS(fsys fs.FS, subdir string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, subdir)
	if err != nil {
		return nil, fmt.Errorf("%w: read fs dir: %v", ErrMigrationFailed, err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fname := entry.Name()
		if !strings.HasSuffix(fname, ".up.sql") {
			continue
		}
		base := strings.TrimSuffix(fname, ".up.sql")
		parts := strings.SplitN(base, "_", 2)
		if len(parts) != 2 {
			continue
		}
		version, description := parts[0], parts[1]
		upPath := path.Join(subdir, fname)

		upSQL, err := fs.ReadFile(fsys, upPath)
		if err != nil {
			return nil, fmt.Errorf("%w: read %s: %v", ErrMigrationFailed, fname, err)
		}
		downPath := path.Join(subdir, strings.Replace(fname, ".up.sql", ".down.sql", 1))
		downSQL, _ := fs.ReadFile(fsys, downPath) // optional

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     description,
			Up:       string(upSQL),
			Down:     string(downSQL),
			FilePath: upPath,
		})
	}

	seenVersions := make(map[string]string, len(migrations))
	for _, m := range migrations {
		if prev, dup := seenVersions[m.Version]; dup {
			return nil, fmt.Errorf("%w: duplicate migration version %s: %q and %q — rename one to a unique version",
				ErrMigrationFailed, m.Version, prev, m.FilePath)
		}
		seenVersions[m.Version] = m.FilePath
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

// MigrateDown rolls back the last migration.
func MigrateDown(ctx context.Context, db *DB, migrationsPath string) error {
	// Get last applied migration
	var version, name string
	err := db.QueryRow(ctx, `
		SELECT version, name FROM schema_migrations
		ORDER BY applied_at DESC LIMIT 1
	`).Scan(&version, &name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // No migrations to rollback
		}
		return fmt.Errorf("%w: get last migration: %v", ErrMigrationFailed, err)
	}

	// Load migrations to find the down SQL
	migrations, err := loadMigrations(migrationsPath)
	if err != nil {
		return err
	}

	var migration *Migration
	for i := range migrations {
		if migrations[i].Version == version {
			migration = &migrations[i]
			break
		}
	}

	if migration == nil || migration.Down == "" {
		return fmt.Errorf("%w: no down migration for version %s", ErrMigrationFailed, version)
	}

	// Execute down migration in transaction
	return db.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, migration.Down); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version)
		return err
	})
}

// txControlLineRe matches ONE full line (terminator excluded) that is entirely a
// standalone transaction-control statement — BEGIN / START TRANSACTION / COMMIT /
// ROLLBACK / END, case-insensitive, optional trailing semicolon. NB: [ \t] (NOT
// \s) so a CRLF line's trailing \r keeps the line unmatched, same as the old
// multiline regex behaved.
var txControlLineRe = regexp.MustCompile(`^[ \t]*(?i:BEGIN|START[ \t]+TRANSACTION|COMMIT|ROLLBACK|END)[ \t]*;?[ \t]*$`)

// dollarTagRe matches a dollar-quote delimiter at the start of the input:
// $$ or $tag$ with an identifier-shaped tag. ($1-style positional params do
// not match — no closing $ on an all-digit tag.)
var dollarTagRe = regexp.MustCompile(`^\$[A-Za-z_][A-Za-z0-9_]*\$|^\$\$`)

// sqlRegion tracks whether the scan position is inside a region that spans
// lines: a single-quoted string (” escape; backslash escapes only in E'…'
// strings — standard_conforming_strings semantics), a double-quoted identifier,
// a dollar-quoted body, or a (nestable) block comment. Line comments (--) never
// span lines, so scanLine simply stops at them.
type sqlRegion struct {
	inSingle   bool
	eString    bool // the open single-quoted string is E'…' (backslash escapes)
	inDouble   bool
	dollarTag  string // "" = not inside a dollar-quoted body
	blockDepth int    // /* … */ nesting depth (PostgreSQL block comments nest)
}

// clean reports that no multi-line region is open at the current position.
func (r *sqlRegion) clean() bool {
	return !r.inSingle && !r.inDouble && r.dollarTag == "" && r.blockDepth == 0
}

// isIdentByte reports whether b can be part of a SQL identifier/keyword — used
// to tell a standalone E'…' string prefix from a word that merely ends in E
// (e.g. LIKE'x'), so plain strings never get backslash-escape semantics.
func isIdentByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// scanLine advances the region state across one line's bytes.
func (r *sqlRegion) scanLine(line string) {
	i := 0
	for i < len(line) {
		switch {
		case r.blockDepth > 0:
			if strings.HasPrefix(line[i:], "*/") {
				r.blockDepth--
				i += 2
			} else if strings.HasPrefix(line[i:], "/*") {
				r.blockDepth++
				i += 2
			} else {
				i++
			}
		case r.inSingle:
			c := line[i]
			if r.eString && c == '\\' {
				i += 2 // backslash escapes the next byte (E-strings only)
				continue
			}
			if c == '\'' {
				if i+1 < len(line) && line[i+1] == '\'' {
					i += 2 // '' — escaped quote, string continues
					continue
				}
				r.inSingle, r.eString = false, false
			}
			i++
		case r.inDouble:
			if line[i] == '"' {
				if i+1 < len(line) && line[i+1] == '"' {
					i += 2 // "" — escaped quote inside the identifier
					continue
				}
				r.inDouble = false
			}
			i++
		case r.dollarTag != "":
			if strings.HasPrefix(line[i:], r.dollarTag) {
				i += len(r.dollarTag)
				r.dollarTag = ""
			} else {
				i++
			}
		default: // clean — look for region openers
			c := line[i]
			if c == '-' && strings.HasPrefix(line[i:], "--") {
				return // line comment: the rest of the line is inert
			}
			if c == '/' && strings.HasPrefix(line[i:], "/*") {
				r.blockDepth++
				i += 2
				continue
			}
			if c == '\'' {
				r.inSingle = true
				r.eString = i >= 1 && (line[i-1] == 'E' || line[i-1] == 'e') &&
					(i == 1 || !isIdentByte(line[i-2]))
				i++
				continue
			}
			if c == '"' {
				r.inDouble = true
				i++
				continue
			}
			if c == '$' {
				if m := dollarTagRe.FindString(line[i:]); m != "" {
					r.dollarTag = m
					i += len(m)
					continue
				}
			}
			i++
		}
	}
}

// stripTxControl removes standalone transaction-control LINES so the caller's
// OUTER transaction owns commit/rollback. This is required for ExecFile dry-run
// correctness: a SQL file's own COMMIT; executed mid-outer-tx would persist the
// changes and defeat the rollback. A line is stripped ONLY when it sits outside
// every string literal, dollar-quoted body, quoted identifier, and comment — a
// bare BEGIN/END inside a DO $$ body or a multi-line literal is content, not
// transaction control, and passes through byte-intact (2607-014 SC02/SC03).
// SET / SET LOCAL and statements sharing a line with other tokens are preserved
// (standalone-line scope is the safe contract). Stripped lines keep their line
// terminator, matching the old regex-replace output shape.
func stripTxControl(sql string) string {
	var b strings.Builder
	b.Grow(len(sql))
	var region sqlRegion
	rest := sql
	for len(rest) > 0 {
		line := rest
		hasNL := false
		if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
			line, rest, hasNL = rest[:nl], rest[nl+1:], true
		} else {
			rest = ""
		}
		if region.clean() && txControlLineRe.MatchString(line) {
			if hasNL {
				b.WriteByte('\n')
			}
			continue
		}
		region.scanLine(line)
		b.WriteString(line)
		if hasNL {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ResolveSchema resolves the target schema for the one-shot CLIs (cmd/migrate,
// cmd/sqlexec): DATABASE_SCHEMA env when set, else the DATABASE_URL user
// (workspace convention: schema = DB user = project name). Empty when neither
// resolves — callers decide whether that is fatal.
func ResolveSchema(dbURL string) string {
	if s := os.Getenv("DATABASE_SCHEMA"); s != "" {
		return s
	}
	if u, err := url.Parse(dbURL); err == nil && u.User != nil {
		return u.User.Username()
	}
	return ""
}

// ExecFileResult reports the outcome of an ExecFile run.
type ExecFileResult struct {
	Bytes      int   // size of the source file in bytes
	DurationMs int64 // wall-clock spent inside the transaction
	DryRun     bool  // true → transaction was rolled back
}

// ExecFile reads a SQL file and executes it in a single transaction against db.
// If dryRun is true the transaction is rolled back (nothing persisted); otherwise
// it is committed. Standalone transaction-control statements (BEGIN/COMMIT/…)
// are stripped first (see stripTxControl) so the outer transaction owns the
// outcome. The SQL is executed as one whole-string multi-statement query — the
// same pattern applyMigration uses. NOT recorded in schema_migrations: this is
// for one-off SQL (data cutover, backfills, manual ops), not versioned schema
// migrations. Re-runs are the caller's responsibility (ExecFile itself is not
// idempotent — the SQL must be, e.g. via ON CONFLICT).
func ExecFile(ctx context.Context, db *DB, path string, dryRun bool) (ExecFileResult, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ExecFileResult{}, fmt.Errorf("%w: read %s: %v", ErrMigrationFailed, path, err)
	}
	body := stripTxControl(string(raw))

	start := time.Now()
	tx, err := db.Begin(ctx)
	if err != nil {
		return ExecFileResult{}, fmt.Errorf("%w: begin: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx) // safe no-op after Commit; dry-run lands here too
		}
	}()

	// Empty after stripping (e.g. a BEGIN/COMMIT-only file) → nothing to exec.
	if strings.TrimSpace(body) != "" {
		if _, err := tx.Exec(ctx, body); err != nil {
			return ExecFileResult{}, fmt.Errorf("%w: exec %s: %v", ErrMigrationFailed, path, err)
		}
	}

	res := ExecFileResult{Bytes: len(raw), DurationMs: time.Since(start).Milliseconds(), DryRun: dryRun}
	if dryRun {
		return res, nil // defer rolls the open tx back
	}
	if err := tx.Commit(ctx); err != nil {
		return ExecFileResult{}, fmt.Errorf("%w: commit: %v", ErrTransactionFailed, err)
	}
	committed = true
	return res, nil
}

func ensureMigrationsTable(ctx context.Context, db *DB) error {
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum VARCHAR(64),
			duration_ms INTEGER
		)
	`)
	if err != nil {
		return err
	}
	// Add columns if they don't exist (for existing tables)
	db.Exec(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS checksum VARCHAR(64)`)
	db.Exec(ctx, `ALTER TABLE schema_migrations ADD COLUMN IF NOT EXISTS duration_ms INTEGER`)
	return nil
}

func getAppliedMigrations(ctx context.Context, db *DB) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

// loadMigrations reads migration files from a directory.
// Supports three naming patterns:
//   - NNN_description.up.sql (standard, with optional .down.sql pair)
//   - NNN_description.sql    (bare, backward-compat with bestays)
//   - NNN-description.sql    (hyphen-separated, backward-compat with bestierealestate)
func loadMigrations(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("%w: read dir: %v", ErrMigrationFailed, err)
	}

	var migrations []Migration
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fname := entry.Name()

		// Skip non-SQL files and .down.sql files
		if !strings.HasSuffix(fname, ".sql") || strings.HasSuffix(fname, ".down.sql") {
			continue
		}

		var version, description, upPath string

		if strings.HasSuffix(fname, ".up.sql") {
			// Pattern: NNN_description.up.sql
			base := strings.TrimSuffix(fname, ".up.sql")
			parts := strings.SplitN(base, "_", 2)
			if len(parts) != 2 {
				continue
			}
			version = parts[0]
			description = parts[1]
			upPath = filepath.Join(dir, fname)
		} else {
			// Pattern: NNN_description.sql or NNN-description.sql
			base := strings.TrimSuffix(fname, ".sql")
			// Try underscore first, then hyphen
			var parts []string
			if idx := strings.IndexByte(base, '_'); idx > 0 {
				parts = []string{base[:idx], base[idx+1:]}
			} else if idx := strings.IndexByte(base, '-'); idx > 0 {
				parts = []string{base[:idx], base[idx+1:]}
			}
			if len(parts) != 2 {
				continue
			}
			version = parts[0]
			description = parts[1]
			upPath = filepath.Join(dir, fname)
		}

		// Read up SQL
		upSQL, err := os.ReadFile(upPath)
		if err != nil {
			return nil, fmt.Errorf("%w: read %s: %v", ErrMigrationFailed, fname, err)
		}

		// Read down SQL if exists (only for .up.sql pattern)
		var downSQL []byte
		if strings.HasSuffix(fname, ".up.sql") {
			downPath := filepath.Join(dir, strings.Replace(fname, ".up.sql", ".down.sql", 1))
			downSQL, _ = os.ReadFile(downPath) // Optional
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     description,
			Up:       string(upSQL),
			Down:     string(downSQL),
			FilePath: upPath,
		})
	}

	// Guard against duplicate version timestamps. Two migration files sharing a
	// version silently corrupt the ledger: the apply high-water-mark records one
	// and skips the other forever — it never shows as pending to `up` (it can
	// read "pending" in `status` yet `up` reports "No pending migrations"). Fail
	// loudly at load time so up/down/status all surface it instead of dropping a
	// migration. (Regression: BR 20260527400000 had two files; widen_phone_country_code
	// never applied.)
	seenVersions := make(map[string]string, len(migrations))
	for _, m := range migrations {
		if prev, dup := seenVersions[m.Version]; dup {
			return nil, fmt.Errorf("%w: duplicate migration version %s: %q and %q — rename one to a unique version",
				ErrMigrationFailed, m.Version, filepath.Base(prev), filepath.Base(m.FilePath))
		}
		seenVersions[m.Version] = m.FilePath
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// checksumFile computes SHA-256 hex digest of a file's contents.
func checksumFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file for checksum: %w", err)
	}
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum), nil
}

// NextMigrationVersion returns a version (YYYYMMDDHHmmss, 14 digits, UTC) for the
// next migration: max(wall clock, latest existing version + 1).
//
// A bare wall-clock stamp is unsafe when the existing sequence is FUTURE-DATED —
// e.g. a parallel sprint stamped migrations ahead of today's date. A clock-based
// version would then sort BELOW already-applied migrations and silently never run
// (it reads "pending" in status yet `up` skips it because its version is under the
// high-water mark). So when the clock is at or below the highest version already
// in `dir`, bump that latest version by 1 instead, guaranteeing the new file sorts
// strictly after every existing migration. (Cross-worktree note: `dir` only sees
// THIS worktree's files; a sibling worktree with a higher unmerged version is not
// visible here, but its versions interleave correctly on merge — only an identical
// version collides, which loadMigrations rejects at load time.)
func NextMigrationVersion(dir string) (string, error) {
	now := time.Now().UTC().Format("20060102150405")

	migrations, err := loadMigrations(dir)
	if err != nil {
		return "", err
	}
	if len(migrations) == 0 {
		return now, nil
	}
	latest := migrations[len(migrations)-1].Version // loadMigrations sorts ascending
	if now > latest {
		return now, nil
	}
	// Clock is at/behind the highest existing version (future-dated sequence).
	// Bump it by 1 so the new version sorts strictly after every existing one.
	// Falls back to the clock for legacy non-numeric versions (001/002...), which
	// the 14-digit clock already out-sorts.
	if next, ok := incrementNumericVersion(latest); ok && next > now {
		return next, nil
	}
	return now, nil
}

// incrementNumericVersion adds 1 to a zero-padded numeric version string,
// preserving its width. Returns ok=false if v is not purely numeric.
func incrementNumericVersion(v string) (string, bool) {
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf("%0*d", len(v), n+1), true
}

// advisoryLockID generates a consistent advisory lock ID from a schema name.
func advisoryLockID(schema string) int64 {
	h := fnv.New64a()
	h.Write([]byte("sbx-migrate:" + schema))
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF) // Ensure positive
}

func applyMigration(ctx context.Context, db *DB, m Migration) error {
	// Compute checksum if file path is available
	var checksum string
	if m.FilePath != "" {
		cs, err := checksumFile(m.FilePath)
		if err == nil {
			checksum = cs
		}
	}

	start := time.Now()
	err := db.WithTx(ctx, func(tx pgx.Tx) error {
		// Execute migration
		if _, err := tx.Exec(ctx, m.Up); err != nil {
			return err
		}

		durationMs := int(time.Since(start).Milliseconds())

		// Record migration with checksum and duration
		_, err := tx.Exec(ctx, `
			INSERT INTO schema_migrations (version, name, checksum, duration_ms) VALUES ($1, $2, $3, $4)
		`, m.Version, m.Name, checksum, durationMs)
		return err
	})
	return err
}

// MigrateBaseline marks all migration files as applied without executing their SQL.
// Used to bootstrap tracking for databases where migrations were applied manually.
// Uses pg_advisory_lock to prevent concurrent runs.
func MigrateBaseline(ctx context.Context, db *DB, migrationsPath string) ([]Migration, error) {
	// Acquire advisory lock
	lockID := advisoryLockID(db.schema)
	if _, err := db.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockID); err != nil {
		return nil, fmt.Errorf("%w: acquire advisory lock: %v", ErrMigrationFailed, err)
	}
	defer db.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockID)

	// Same schema guarantee as Migrate — baseline tracking must never strand
	// schema_migrations in `public` via a missing search_path schema.
	if err := db.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("%w: ensure schema %s: %v", ErrMigrationFailed, db.schema, err)
	}

	// Ensure migrations table exists
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}

	// Get already applied migrations
	applied, err := getAppliedMigrations(ctx, db)
	if err != nil {
		return nil, err
	}

	// Load migration files
	migrations, err := loadMigrations(migrationsPath)
	if err != nil {
		return nil, err
	}

	// Record each pending migration as applied (without executing SQL)
	var baselined []Migration
	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}

		var checksum string
		if m.FilePath != "" {
			cs, err := checksumFile(m.FilePath)
			if err == nil {
				checksum = cs
			}
		}

		_, err := db.Exec(ctx, `
			INSERT INTO schema_migrations (version, name, checksum, duration_ms) VALUES ($1, $2, $3, 0)
		`, m.Version, m.Name, checksum)
		if err != nil {
			return baselined, fmt.Errorf("%w: baseline version %s: %v", ErrMigrationFailed, m.Version, err)
		}
		baselined = append(baselined, m)
	}

	return baselined, nil
}

// MigrateStatus returns the status of all migrations (applied + pending).
func MigrateStatus(ctx context.Context, db *DB, migrationsPath string) ([]MigrationStatus, error) {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}

	// Load applied migrations from DB
	type appliedRecord struct {
		Version    string
		Name       string
		AppliedAt  time.Time
		Checksum   *string
		DurationMs *int
	}
	rows, err := db.Query(ctx, `
		SELECT version, name, applied_at, checksum, duration_ms
		FROM schema_migrations ORDER BY version
	`)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	appliedMap := make(map[string]appliedRecord)
	for rows.Next() {
		var r appliedRecord
		if err := rows.Scan(&r.Version, &r.Name, &r.AppliedAt, &r.Checksum, &r.DurationMs); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		appliedMap[r.Version] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Load migration files
	migrations, err := loadMigrations(migrationsPath)
	if err != nil {
		return nil, err
	}

	// Build combined status
	var statuses []MigrationStatus
	for _, m := range migrations {
		currentChecksum := ""
		if m.FilePath != "" {
			cs, err := checksumFile(m.FilePath)
			if err == nil {
				currentChecksum = cs
			}
		}

		if applied, ok := appliedMap[m.Version]; ok {
			dirty := false
			if applied.Checksum != nil && currentChecksum != "" && *applied.Checksum != currentChecksum {
				dirty = true
			}
			statuses = append(statuses, MigrationStatus{
				Version:    m.Version,
				Name:       m.Name,
				State:      "applied",
				AppliedAt:  &applied.AppliedAt,
				Checksum:   currentChecksum,
				DurationMs: applied.DurationMs,
				Dirty:      dirty,
			})
			delete(appliedMap, m.Version)
		} else {
			statuses = append(statuses, MigrationStatus{
				Version:  m.Version,
				Name:     m.Name,
				State:    "pending",
				Checksum: currentChecksum,
			})
		}
	}

	// Any applied migrations not in files (orphaned)
	for _, r := range appliedMap {
		statuses = append(statuses, MigrationStatus{
			Version:    r.Version,
			Name:       r.Name,
			State:      "applied",
			AppliedAt:  &r.AppliedAt,
			DurationMs: r.DurationMs,
			Dirty:      true, // File missing = dirty
		})
	}

	return statuses, nil
}

// =============================================================================
// AUDIT LOG
// =============================================================================

// AuditLog writes an entry to shared.audit_log.
func (db *DB) AuditLog(ctx context.Context, entityType, entityID, action, actor string, changes any) error {
	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return fmt.Errorf("%w: marshal changes: %v", ErrAuditLogFailed, err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO shared.audit_log (entity_type, entity_id, action, actor, changes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, entityType, entityID, action, actor, changesJSON, time.Now().UTC())

	if err != nil {
		return fmt.Errorf("%w: insert: %v", ErrAuditLogFailed, err)
	}

	return nil
}

// =============================================================================
// HELPERS
// =============================================================================

// TableExists checks if a table exists in the current schema.
func (db *DB) TableExists(ctx context.Context, tableName string) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)
	`, db.schema, tableName).Scan(&exists)
	return exists, err
}

// EnsureSchema creates the schema if it doesn't exist.
func (db *DB) EnsureSchema(ctx context.Context) error {
	_, err := db.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, db.schema))
	return err
}
