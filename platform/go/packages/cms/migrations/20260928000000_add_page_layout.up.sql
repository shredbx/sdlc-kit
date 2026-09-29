-- Adds the layout selector (docs/proposals/bos-layout-presets.md, corrected: placement of
-- sections within a layout is by SectionKind, not a separate slot column — this migration
-- only adds WHICH registered layout preset (bos-svelte's layout registry) a page uses).
-- Fresh column, not part of the original create-table migration: that one already applied
-- against the running bos-demo dev database, so it can't be edited in place.
ALTER TABLE cms_pages ADD COLUMN IF NOT EXISTS layout TEXT NOT NULL DEFAULT 'default';
