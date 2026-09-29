package cms

import "embed"

// MigrationsFS embeds this kit's own schema migrations, so a consumer applies them
// with database.MigrateFS(ctx, db, cms.MigrationsFS, "migrations") — no dependency
// on this kit's source tree being present at runtime (platform/CLAUDE.md D15: a kit
// "later also carries its bos wiring", migrations included).
//
//go:embed migrations/*.sql
var MigrationsFS embed.FS
