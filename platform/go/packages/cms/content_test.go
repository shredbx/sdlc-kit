package cms

// The former domain-level draft→publish operation (CmsPage.Publish copying
// DraftContent into PublishedContent + bumping version) was REMOVED by Decision
// #0281 (AE4): CMS pages adopt the content_entries single direct-save model — the
// section editor writes straight to published_content and a `published` boolean
// flag gates the public read. There is no longer a two-slot copy operation to
// unit-test here, so the TC-SELL-3 / SC-SELL-5 Publish tests are gone with the
// method. The direct-save + published-flag behavior is covered end-to-end in
// internal/repository/cmspage_postgres_integration_test.go (the upsert + GetBySlug
// round-trip) and internal/handler/cmspage_public_test.go (the public 404 gate).
//
// Section-list domain behavior (NewSection, kind discriminators, JSONB round-trip)
// continues to live in section_test.go / entry_test.go — unaffected by #0281.
