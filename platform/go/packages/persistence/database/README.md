# database

A PostgreSQL adapter over pgx: a schema-scoped connection pool, transactions, a file-based migration runner, one-off SQL files and an audit-log helper.

```bash
go test ./...    # this module's tests
```

## Overview

`database` wraps a `pgxpool` connection pool for one project's schema. `New(ctx, Config)` parses the URL, sets the pool limits (`MaxConns` 10 and `MinConns` 2 when left at 0), puts `<schema>,public,shared` on the `search_path`, and pings the database, so it fails with `ErrConnectionFailed` unless a server is reachable. `WrapPool` wraps a pool you already have. A `DB` then offers `Query`, `QueryRow`, `Exec`, `Begin`, `Pool` and `Close`, and:

- **`WithTx(ctx, fn)`** runs `fn` in a transaction: it commits when `fn` returns nil, rolls back when `fn` returns an error (your own error comes back), and rolls back and re-panics on a panic.
- **Migrations** are SQL files in one directory, named `NNN_name.up.sql` (with an optional `NNN_name.down.sql`), `NNN_name.sql` or `NNN-name.sql`. `Migrate` takes a PostgreSQL advisory lock, creates the schema and a `schema_migrations` table if they are missing, and applies each pending file in version order, each in its own transaction, recording its version, name, SHA-256 checksum and duration. It stops at the first failure and returns the migrations applied so far. `MigrateStatus` lists each file as `applied` or `pending`, and marks an entry `Dirty` when its file changed after it ran or is missing. `MigrateDown` undoes the last applied migration using its `.down.sql`. `MigrateBaseline` records every pending file as applied without running it, for a database that was migrated by hand. A duplicate version number is refused when the directory is read. `NextMigrationVersion(dir)` returns a 14-digit UTC timestamp, or one past the highest existing version when that is ahead of the clock.
- **`ExecFile(ctx, db, path, dryRun)`** runs one SQL file in a single transaction and commits it, or rolls it back when `dryRun` is true. Lines that are only `BEGIN`, `COMMIT`, `ROLLBACK` or `END` are removed first (outside quoted text, dollar-quoted bodies and comments), so the outer transaction decides the outcome. The run is not recorded in `schema_migrations`.
- **`AuditLog`** inserts a row into `shared.audit_log` (entity type, id, action, actor and a JSON `changes` value); this package does not create that table. `TableExists` and `EnsureSchema` are small helpers, and `ResolveSchema(url)` picks the schema from `DATABASE_SCHEMA`, or else from the user name in the URL.

Things to know, from reading the code. The functions that talk to a database have no tests here:

- The schema name is put into `search_path` and into `CREATE SCHEMA IF NOT EXISTS <schema>` without quoting. Pass a constant, never request input.
- Errors carry a sentinel (`ErrConnectionFailed`, `ErrMigrationFailed`, `ErrTransactionFailed`, `ErrAuditLogFailed`) that `errors.Is` finds, but the underlying pgx error is turned into text, so `errors.As` cannot reach it.
- Migration versions sort as text, so keep them one width: zero-padded numbers, or timestamps.
- `Migrate` and `MigrateBaseline` take the advisory lock and release it with `db.Exec`, which goes through the pool. A session-level advisory lock belongs to one connection, and the pool does not promise that the unlock runs on the same one.
- `MigrateDown` picks the migration with the latest `applied_at`, and it does not take the advisory lock.
- The comment on `WrapPool` says the `DB` does not own the pool, but `Close` closes it.

The `repository/postgres` module builds on this one.

## Install

`database` is a Go module in the `platform/go` workspace and is not published. Inside the workspace it is listed in `platform/go/go.work`, so there is nothing to install: import it.

From another module, require it and point the `replace` at this folder (the path is relative to your `go.mod`), then run `go mod tidy` to pull in its one third-party requirement, `github.com/jackc/pgx/v5`:

```
require github.com/shredbx/sbx-core/pkg/database v0.0.0

replace github.com/shredbx/sbx-core/pkg/database => <path to platform/go/packages/persistence/database>
```

The import path keeps its original name, `github.com/shredbx/sbx-core/pkg/database`, until the naming pass; only the folder was regrouped.

## Usage

Some functions need no server. Resolve the schema, see how `New` fails on a malformed URL, and get the next migration version:

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shredbx/sbx-core/pkg/database"
)

func main() {
	// The schema is the DATABASE_SCHEMA variable when set, otherwise the user in the URL.
	dsn := "postgres://hub:secret@localhost:5432/workspace"
	_ = os.Unsetenv("DATABASE_SCHEMA")
	fmt.Println("schema from the URL:", database.ResolveSchema(dsn))
	_ = os.Setenv("DATABASE_SCHEMA", "tenant")
	fmt.Println("schema from the environment:", database.ResolveSchema(dsn))

	// New parses the URL first, so a malformed one fails before any connection is tried.
	_, err := database.New(context.Background(), database.Config{URL: "not a url"})
	fmt.Println("malformed URL:", errors.Is(err, database.ErrConnectionFailed))

	// Migration files live in one directory. The next version is the UTC clock,
	// or one past the highest existing version when that is in the future.
	dir, err := os.MkdirTemp("", "migrations")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)
	_ = os.WriteFile(filepath.Join(dir, "99990101000000_create_items.up.sql"), []byte("CREATE TABLE items (id int);"), 0o644)
	next, _ := database.NextMigrationVersion(dir)
	fmt.Println("next version:", next)
}
```

It prints:

```
schema from the URL: hub
schema from the environment: tenant
malformed URL: true
next version: 99990101000001
```

Against a running PostgreSQL, connect, migrate and run a transaction. This second program is compiled and vetted with the other checks on this README, but not run, because it needs a server:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/shredbx/sbx-core/pkg/database"
)

func main() {
	ctx := context.Background()

	db, err := database.New(ctx, database.Config{
		URL:    "postgres://app:secret@localhost:5432/appdb",
		Schema: "app", // goes on the search_path: a trusted constant, never request input
	})
	if err != nil {
		log.Fatal(err) // errors.Is(err, database.ErrConnectionFailed)
	}
	defer db.Close()

	// Apply the pending files in ./migrations, one transaction each.
	applied, err := database.Migrate(ctx, db, "./migrations")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("applied:", len(applied))

	// Commit when the function returns nil; roll back when it returns an error or panics.
	err = db.WithTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO items (name) VALUES ($1)`, "first")
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Configuration

`Config` has `URL`, `Schema`, `MaxConns` (10 when 0) and `MinConns` (2 when 0). `ResolveSchema` reads `DATABASE_SCHEMA`; nothing else is read from the environment.

## Tests

`go test ./...` in this folder, or `go -C platform/go/packages/persistence/database test ./...` from the repository root. The tests cover reading migration files, version numbering, the transaction-line stripping and `ResolveSchema`. None of them touches a database, so `New`, `WithTx`, the `Migrate…` runners, `ExecFile` and `AuditLog` are not tested here.
