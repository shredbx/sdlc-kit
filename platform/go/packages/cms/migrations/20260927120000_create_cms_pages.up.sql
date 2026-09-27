-- The cms_pages table — editable static content pages (About, Contact, Terms, …),
-- keyed by a unique slug. Column list matches this kit's own cmspage.go
-- (PostgresMapper.Columns()/FromRow()) exactly.
--
-- Consolidated from the real source's own incremental history (migrations
-- 20260524223616, 20260528120000, 20260529200001, 20260603000000, 20260616000000
-- in bestierealestate) into one CREATE, since a bos consumer starts fresh and has
-- no prior rows to carry forward.
--
-- Unqualified table name: this kit's migrations run via database.MigrateFS against
-- a connection whose search_path already selects the consumer's own schema
-- (database.New sets it from Config.Schema) — the kit names no schema itself, so it
-- works unmodified for any consumer's schema name.
--
-- Version note: this file's version (20260927120000) is a long timestamp, not a
-- consumer's short sequential number, on purpose — every kit's migrations and every
-- consumer's own migrations land in the SAME schema_migrations table, so a kit's
-- version must never collide with a consumer's "001".."00N" style or another kit's
-- own timestamp. Every kit shipping its own migrations should follow this same
-- convention.
CREATE TABLE IF NOT EXISTS cms_pages (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              TEXT NOT NULL,
    title             TEXT,
    body_markdown     TEXT,
    details           JSONB,
    draft_content     JSONB,
    published_content JSONB,
    published         BOOLEAN NOT NULL DEFAULT false,
    seo_meta          JSONB,
    version           INTEGER NOT NULL DEFAULT 1,
    created_by        UUID,
    updated_by        UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ
);

-- Slug is the public URL key — unique among live (non-deleted) rows.
CREATE UNIQUE INDEX IF NOT EXISTS uq_cms_pages_slug
    ON cms_pages (slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_cms_pages_deleted_at
    ON cms_pages (deleted_at);

CREATE OR REPLACE FUNCTION cms_pages_update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_cms_pages_updated_at ON cms_pages;
CREATE TRIGGER trg_cms_pages_updated_at
    BEFORE UPDATE ON cms_pages
    FOR EACH ROW
    EXECUTE FUNCTION cms_pages_update_updated_at();
