// Package dictionary PostgreSQL backend — one lookup table per dictionary entity.
//
// Each dictionary entity gets its own table:
//
//	{schema}.{name_underscored}s  (e.g., bestierealestate.property_types)
//
// Table schema:
//
//	code        VARCHAR(50) PRIMARY KEY
//	label       VARCHAR(100) NOT NULL
//	description TEXT
//	sort_order  SMALLINT NOT NULL DEFAULT 0
//	icon        VARCHAR(100)
//	deprecated  BOOLEAN NOT NULL DEFAULT false
//
// Generic CRUD: the table name is derived from the dictionary name + schema,
// so the same functions work for any dictionary entity.
package dictionary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/shredbx/sbx-core/pkg/database"
)

// PostgresBackend stores dictionary entries in per-entity lookup tables.
type PostgresBackend struct {
	db       *database.DB
	schema   string            // PostgreSQL schema (e.g., "bestierealestate")
	tableMap map[string]string // optional explicit dict-name → table overrides
}

// NewPostgresBackend creates a new PostgreSQL dictionary backend.
func NewPostgresBackend(db *database.DB, schema string) *PostgresBackend {
	return &PostgresBackend{
		db:     db,
		schema: schema,
	}
}

// NewPostgresBackendFromPool creates a backend from a raw *pgxpool.Pool.
// Bridge for consumers (like BR's api-chi) that carry a pool directly rather
// than a *database.DB. Wraps the pool via database.WrapPool so all Exec/Query/
// Begin/WithTx methods work identically.
func NewPostgresBackendFromPool(pool *pgxpool.Pool, schema string) *PostgresBackend {
	return &PostgresBackend{
		db:     database.WrapPool(pool, schema),
		schema: schema,
	}
}

// WithRegistry returns the backend with explicit dict-name → table overrides
// from the Registry. Use this when dictionary names don't follow the default
// kebab-singular → plural-snake convention (e.g., names like "property_types"
// or "property_furnished" where the heuristic would produce wrong tables).
func (b *PostgresBackend) WithRegistry(registry Registry) *PostgresBackend {
	if b.tableMap == nil {
		b.tableMap = make(map[string]string, len(registry))
	}
	for name, spec := range registry {
		if spec.Table != "" {
			b.tableMap[name] = spec.Table
		}
	}
	return b
}

// tableName resolves a dictionary name to a qualified table name. Explicit
// overrides from WithRegistry take precedence; otherwise the default heuristic
// "property-type" → "{schema}.property_types" applies (replace dashes with
// underscores, append "s").
func (b *PostgresBackend) tableName(name string) string {
	var table string
	if t, ok := b.tableMap[name]; ok {
		table = t
	} else {
		table = strings.ReplaceAll(name, "-", "_") + "s"
	}
	if b.schema != "" {
		return b.schema + "." + table
	}
	return table
}

// EnsureTable creates the lookup table if it doesn't exist.
func (b *PostgresBackend) EnsureTable(ctx context.Context, name string) error {
	sql := b.GenerateCreateSQL(name)
	_, err := b.db.Exec(ctx, sql)
	return err
}

// GenerateCreateSQL generates CREATE TABLE DDL for a dictionary lookup table.
func (b *PostgresBackend) GenerateCreateSQL(name string) string {
	var sb strings.Builder
	qt := b.tableName(name)

	if b.schema != "" {
		sb.WriteString(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;\n\n", b.schema))
	}

	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", qt))
	sb.WriteString("    code         VARCHAR(50)  PRIMARY KEY,\n")
	sb.WriteString("    label        TEXT         NOT NULL,\n")
	sb.WriteString("    description  TEXT,\n")
	sb.WriteString("    icon         VARCHAR(100),\n")
	sb.WriteString("    color        VARCHAR(7),\n")
	sb.WriteString(`    "group"      VARCHAR(50)  NOT NULL DEFAULT '',` + "\n")
	sb.WriteString("    sort_order   SMALLINT     NOT NULL DEFAULT 0,\n")
	sb.WriteString("    translations JSONB,\n")
	sb.WriteString("    image_url    TEXT,\n")
	sb.WriteString("    video_url    TEXT,\n")
	sb.WriteString("    deprecated   BOOLEAN      NOT NULL DEFAULT false\n")
	sb.WriteString(");\n")

	return sb.String()
}

// GenerateSeedSQL generates INSERT statements from entries.
func (b *PostgresBackend) GenerateSeedSQL(name string, entries []Entry) string {
	if len(entries) == 0 {
		return ""
	}

	var sb strings.Builder
	qt := b.tableName(name)

	sb.WriteString(fmt.Sprintf(
		`INSERT INTO %s (code, label, description, icon, color, "group", sort_order, translations, image_url, video_url, deprecated) VALUES`+"\n",
		qt,
	))

	for i, e := range entries {
		var transLit string
		if len(e.Translations) > 0 {
			raw, _ := json.Marshal(e.Translations)
			transLit = fmt.Sprintf("'%s'::jsonb", strings.ReplaceAll(string(raw), "'", "''"))
		} else {
			transLit = "NULL"
		}

		sep := ","
		if i == len(entries)-1 {
			sep = ""
		}

		sb.WriteString(fmt.Sprintf("    ('%s', '%s', %s, %s, %s, '%s', %d, %s, %s, %s, %t)%s\n",
			strings.ReplaceAll(e.Code, "'", "''"),
			strings.ReplaceAll(e.Label, "'", "''"),
			sqlStrLit(e.Description),
			sqlStrLit(e.Icon),
			sqlStrLit(e.Color),
			strings.ReplaceAll(e.Group, "'", "''"),
			e.SortOrder,
			transLit,
			sqlStrLit(e.ImageURL),
			sqlStrLit(e.VideoURL),
			e.Deprecated,
			sep,
		))
	}

	sb.WriteString("ON CONFLICT (code) DO NOTHING;\n")

	return sb.String()
}

// List returns all entries for a dictionary, ordered by sort_order.
//
// Column set is the minimal core: code, label, description, icon, sort_order,
// deprecated. The extended fields (Color, Translations, ImageURL, VideoURL) on
// the Entry struct stay zero-valued — they were aspirational in the original
// DDL but no production consumer uses them at runtime. Consumers that need
// them should issue narrower domain-specific queries.
func (b *PostgresBackend) List(ctx context.Context, name string) ([]Entry, error) {
	qt := b.tableName(name)

	sql := fmt.Sprintf(
		`SELECT code, label, COALESCE(description,''), COALESCE(icon,''), COALESCE("group",''), sort_order, deprecated
		 FROM %s ORDER BY sort_order ASC`,
		qt,
	)

	rows, err := b.db.Query(ctx, sql)
	if err != nil {
		return nil, fmt.Errorf("%w: list %s: %v", ErrBackendError, name, err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(
			&e.Code, &e.Label, &e.Description, &e.Icon, &e.Group, &e.SortOrder, &e.Deprecated,
		); err != nil {
			return nil, fmt.Errorf("%w: scan: %v", ErrBackendError, err)
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

// Get returns a single entry by code. See List for column-set notes.
func (b *PostgresBackend) Get(ctx context.Context, name, code string) (*Entry, error) {
	qt := b.tableName(name)

	sql := fmt.Sprintf(
		`SELECT code, label, COALESCE(description,''), COALESCE(icon,''), COALESCE("group",''), sort_order, deprecated
		 FROM %s WHERE code = $1`,
		qt,
	)

	var e Entry
	err := b.db.QueryRow(ctx, sql, code).Scan(
		&e.Code, &e.Label, &e.Description, &e.Icon, &e.Group, &e.SortOrder, &e.Deprecated,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, fmt.Errorf("%w: code '%s' in %s", ErrNotFound, code, name)
		}
		return nil, fmt.Errorf("%w: get: %v", ErrBackendError, err)
	}

	return &e, nil
}

// Create inserts a new entry into the dictionary table. See List for column-set notes.
func (b *PostgresBackend) Create(ctx context.Context, name string, entry Entry) error {
	qt := b.tableName(name)

	sql := fmt.Sprintf(
		`INSERT INTO %s (code, label, description, icon, "group", sort_order, deprecated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		qt,
	)

	_, err := b.db.Exec(ctx, sql,
		entry.Code,
		entry.Label,
		nilIfEmpty(entry.Description),
		nilIfEmpty(entry.Icon),
		entry.Group,
		entry.SortOrder,
		entry.Deprecated,
	)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fmt.Errorf("%w: code '%s' already exists in %s", ErrDuplicate, entry.Code, name)
		}
		return fmt.Errorf("%w: create: %v", ErrBackendError, err)
	}

	return nil
}

// Seed inserts multiple entries, skipping duplicates (ON CONFLICT DO NOTHING).
func (b *PostgresBackend) Seed(ctx context.Context, name string, entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	sql := b.GenerateSeedSQL(name, entries)
	_, err := b.db.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("%w: seed %s: %v", ErrBackendError, name, err)
	}

	return nil
}

// Update applies a partial patch to a dictionary entry. nil-valued fields in
// UpdatePatch leave the corresponding column unchanged. An all-nil patch is
// a no-op (returns current entry). Returns ErrNotFound if (name, code) does
// not exist.
func (b *PostgresBackend) Update(ctx context.Context, name, code string, patch UpdatePatch) (*Entry, error) {
	qt := b.tableName(name)

	// Build dynamic SET clause from non-nil patch fields.
	sets := make([]string, 0, 5)
	args := make([]any, 0, 6)
	idx := 1
	if patch.Label != nil {
		sets = append(sets, fmt.Sprintf("label = $%d", idx))
		args = append(args, *patch.Label)
		idx++
	}
	if patch.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", idx))
		args = append(args, nilIfEmpty(*patch.Description))
		idx++
	}
	if patch.Group != nil {
		// group is NOT NULL DEFAULT '' — write the literal value (empty clears it).
		sets = append(sets, fmt.Sprintf(`"group" = $%d`, idx))
		args = append(args, *patch.Group)
		idx++
	}
	if patch.SortOrder != nil {
		sets = append(sets, fmt.Sprintf("sort_order = $%d", idx))
		args = append(args, *patch.SortOrder)
		idx++
	}
	if patch.Icon != nil {
		sets = append(sets, fmt.Sprintf("icon = $%d", idx))
		args = append(args, nilIfEmpty(*patch.Icon))
		idx++
	}
	if patch.Deprecated != nil {
		sets = append(sets, fmt.Sprintf("deprecated = $%d", idx))
		args = append(args, *patch.Deprecated)
		idx++
	}

	// No-op patch — return current entry without touching DB.
	if len(sets) == 0 {
		return b.Get(ctx, name, code)
	}

	args = append(args, code)
	sql := fmt.Sprintf(
		"UPDATE %s SET %s WHERE code = $%d",
		qt, strings.Join(sets, ", "), idx,
	)

	tag, err := b.db.Exec(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: update %s/%s: %v", ErrBackendError, name, code, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("%w: %s/%s", ErrNotFound, name, code)
	}

	return b.Get(ctx, name, code)
}

// Delete removes a single dictionary row. FK RESTRICT enforces orphan
// prevention at the DB level; callers should use CascadeReplaceAndDelete
// when consumer rows exist. Returns ErrNotFound if (name, code) does not exist.
func (b *PostgresBackend) Delete(ctx context.Context, name, code string) error {
	qt := b.tableName(name)
	sql := fmt.Sprintf("DELETE FROM %s WHERE code = $1", qt)

	tag, err := b.db.Exec(ctx, sql, code)
	if err != nil {
		// FK RESTRICT violation surfaces as a Postgres "violates foreign key constraint" error.
		// Callers (the handler layer) translate this into a meaningful 422 when consumers exist.
		return fmt.Errorf("%w: delete %s/%s: %v", ErrBackendError, name, code, err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: %s/%s", ErrNotFound, name, code)
	}

	return nil
}

// ConsumersCount runs one COUNT query per FKRef and returns the total + breakdown.
// Invoked at delete-click time only — NOT eager-loaded into editor view-model (SMI-007).
func (b *PostgresBackend) ConsumersCount(ctx context.Context, name, code string, refs []FKRef) (total int, byRef []ConsumerCount, err error) {
	if len(refs) == 0 {
		return 0, nil, nil
	}

	byRef = make([]ConsumerCount, 0, len(refs))
	for _, ref := range refs {
		// Schema-qualify the consumer table the same way we do for dict tables.
		qt := ref.Table
		if b.schema != "" {
			qt = b.schema + "." + ref.Table
		}
		// Use %I formatting via parameter binding — but column/table names cannot be
		// parameterized in SQL. They come from the Registry (compile-time-known list),
		// not user input — so safe to format directly. The dict `code` value IS
		// parameterized via $1.
		q := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s = $1", qt, ref.Column)
		var n int
		if err := b.db.QueryRow(ctx, q, code).Scan(&n); err != nil {
			return 0, nil, fmt.Errorf("%w: count %s.%s: %v", ErrBackendError, ref.Table, ref.Column, err)
		}
		if n > 0 {
			byRef = append(byRef, ConsumerCount{Table: ref.Table, Column: ref.Column, Count: n})
			total += n
		}
	}
	return total, byRef, nil
}

// ConsumersCountByCode returns the total consumer count for EVERY code in the
// dictionary in one pass per FKRef, keyed by code. Use this to enrich a list/
// editor view-model with per-value usage counts without issuing one query per
// (code, ref) pair. Codes with zero consumers are absent from the map (caller
// treats a missing key as 0).
//
// Unlike ConsumersCount (single-code, delete-click path), this is GROUP BY-based:
// one query per FKRef regardless of how many codes the dictionary has.
func (b *PostgresBackend) ConsumersCountByCode(ctx context.Context, refs []FKRef) (map[string]int, error) {
	counts := make(map[string]int)
	for _, ref := range refs {
		qt := ref.Table
		if b.schema != "" {
			qt = b.schema + "." + ref.Table
		}
		// table/column come from the compile-time Registry (not user input), so
		// formatting them directly is safe — same rationale as ConsumersCount.
		q := fmt.Sprintf("SELECT %s, COUNT(*) FROM %s WHERE %s IS NOT NULL GROUP BY %s",
			ref.Column, qt, ref.Column, ref.Column)
		rows, err := b.db.Query(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("%w: count-by-code %s.%s: %v", ErrBackendError, ref.Table, ref.Column, err)
		}
		for rows.Next() {
			var code string
			var n int
			if err := rows.Scan(&code, &n); err != nil {
				rows.Close()
				return nil, fmt.Errorf("%w: scan count-by-code: %v", ErrBackendError, err)
			}
			counts[code] += n
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("%w: iterate count-by-code: %v", ErrBackendError, err)
		}
		rows.Close()
	}
	return counts, nil
}

// CascadeReplaceAndDelete migrates every consumer row referencing `code` to
// `replacement`, then deletes the dictionary row. Executes inside a single
// transaction — either all consumer rows migrate and the row is gone, or
// nothing changes. FK RESTRICT belt-and-suspenders enforces ordering.
//
// If replacement is empty, the operation falls back to plain Delete (which will
// fail if consumers exist — surfaced as ErrBackendError with FK violation).
func (b *PostgresBackend) CascadeReplaceAndDelete(ctx context.Context, name, code, replacement string, refs []FKRef) (migratedCount int, err error) {
	qt := b.tableName(name)

	err = b.db.WithTx(ctx, func(tx pgx.Tx) error {
		if replacement != "" {
			for _, ref := range refs {
				cqt := ref.Table
				if b.schema != "" {
					cqt = b.schema + "." + ref.Table
				}
				// Update every row in consumer table that points at `code` to point at `replacement`.
				q := fmt.Sprintf("UPDATE %s SET %s = $1 WHERE %s = $2", cqt, ref.Column, ref.Column)
				tag, ierr := tx.Exec(ctx, q, replacement, code)
				if ierr != nil {
					return fmt.Errorf("update %s.%s: %v", ref.Table, ref.Column, ierr)
				}
				migratedCount += int(tag.RowsAffected())
			}
		}

		// Delete the dictionary row. If consumers still exist (replacement was empty
		// or migration missed any), FK RESTRICT aborts the tx and rolls back any
		// preceding consumer updates.
		dq := fmt.Sprintf("DELETE FROM %s WHERE code = $1", qt)
		tag, ierr := tx.Exec(ctx, dq, code)
		if ierr != nil {
			return fmt.Errorf("delete %s: %v", name, ierr)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: %s/%s", ErrNotFound, name, code)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("%w: cascade %s/%s→%s: %v", ErrBackendError, name, code, replacement, err)
	}

	return migratedCount, nil
}

// Reorder atomically rewrites sort_order for the dictionary's entries so each
// code in `codes` lands at position index+1. Caller must pass the FULL set of
// dictionary codes — partial reorder is rejected. The operation runs in a
// single transaction; any validation failure or DB error rolls back.
//
// Validation rules (all enforced before any UPDATE):
//   - codes must not be empty
//   - codes must not contain duplicates
//   - codes must match the dictionary's current code set EXACTLY (no extras, no missing)
//
// Returns ErrNotFound if the dictionary table is empty.
// Returns ErrInvalidPayload wrapping a specific reason on validation failure.
// Returns ErrBackendError on transaction failure.
func (b *PostgresBackend) Reorder(ctx context.Context, name string, codes []string) error {
	if len(codes) == 0 {
		return fmt.Errorf("%w: codes is empty", ErrInvalidPayload)
	}

	seen := make(map[string]struct{}, len(codes))
	for _, c := range codes {
		if _, dup := seen[c]; dup {
			return fmt.Errorf("%w: duplicate code %q", ErrInvalidPayload, c)
		}
		seen[c] = struct{}{}
	}

	qt := b.tableName(name)

	return b.db.WithTx(ctx, func(tx pgx.Tx) error {
		existing := make(map[string]struct{}, len(codes))
		rows, err := tx.Query(ctx, fmt.Sprintf("SELECT code FROM %s", qt))
		if err != nil {
			return fmt.Errorf("%w: select codes %s: %v", ErrBackendError, name, err)
		}
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				rows.Close()
				return fmt.Errorf("%w: scan code: %v", ErrBackendError, err)
			}
			existing[c] = struct{}{}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("%w: iterate codes: %v", ErrBackendError, err)
		}

		if len(existing) == 0 {
			return fmt.Errorf("%w: dictionary %s has no entries", ErrNotFound, name)
		}
		if len(existing) != len(codes) {
			return fmt.Errorf("%w: codes count %d does not match dict size %d", ErrInvalidPayload, len(codes), len(existing))
		}
		for _, c := range codes {
			if _, ok := existing[c]; !ok {
				return fmt.Errorf("%w: unknown code %q", ErrInvalidPayload, c)
			}
		}

		for i, c := range codes {
			q := fmt.Sprintf("UPDATE %s SET sort_order = $1 WHERE code = $2", qt)
			if _, err := tx.Exec(ctx, q, i+1, c); err != nil {
				return fmt.Errorf("%w: update sort_order for %s/%s: %v", ErrBackendError, name, c, err)
			}
		}
		return nil
	})
}

// nilIfEmpty returns nil for empty strings (for nullable DB columns).
func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// nilIfBytes returns nil for empty/nil byte slices (for nullable JSONB columns).
func nilIfBytes(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

// sqlStrLit returns a single-quoted SQL string literal, or NULL for empty strings.
func sqlStrLit(s string) string {
	if s == "" {
		return "NULL"
	}
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "''"))
}
