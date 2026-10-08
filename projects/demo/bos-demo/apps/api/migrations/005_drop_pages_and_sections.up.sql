-- The hand-built pages/page_sections model (001, 003) is replaced by the cms kit's
-- own cms_pages table (platform/go/packages/cms/migrations) — see
-- docs/proposals/bos-cms-port-plan.md. Nothing in these tables carries forward:
-- bos-demo is a dev demo with no production data, and the content model itself
-- changed shape (slug-keyed, typed sections) rather than migrating row-for-row.
DROP TABLE IF EXISTS page_sections;
DROP TABLE IF EXISTS pages;
