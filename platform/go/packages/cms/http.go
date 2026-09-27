// CMS static-page HTTP handlers — ported from bestierealestate's own
// internal/handler/cmspage.go (CmsPageHandler), merged into this package the same
// way postgres.go merges the repository (see its own doc comment): the original
// split handler/repository/entity across three packages only because the app had
// no shared framework to place them in (platform/CLAUDE.md's D15 — a kit "later
// also carries its bos wiring (handlers, migrations, routes, admin pages)").
//
// RegisterAdmin/RegisterPublic are this kit's public wiring surface — a consumer's
// main.go calls them the same two-function way it already calls
// settings.RegisterAdmin/RegisterPublic; it never constructs a handler itself.
//
// A CmsPage is editable static content (About, Contact, …) authored as markdown
// and keyed by a unique slug. RegisterAdmin's routes carry no RBAC of their own —
// same as the original's route group being gated at the call site, not the
// handler — the consumer wraps them in whatever role/auth middleware it uses (see
// bos-demo's main.go: RequireAuth is enough, any authenticated role manages
// content, matching the original's own agent-vs-manager authority split being a
// separate, later concern).
//
// RegisterPublic mounts the public read: GetPublic serves the public projection,
// gated on the page's published flag (Decision #0281). Content and Details use
// JSON field-presence semantics on Save — see postgres.go's UpsertBySlug doc.
package cms

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	pkgauth "github.com/shredbx/sbx-core/pkg/auth"
	"github.com/shredbx/sbx-core/pkg/httputil"
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/seo"
)

// pageHandler is the unexported handler set RegisterAdmin/RegisterPublic build —
// a consumer never constructs one directly, matching this kit's two-function
// public surface (see settings.RegisterAdmin/RegisterPublic for the same shape).
type pageHandler struct {
	repo *Repository
}

// savePageRequest is the PUT body: the editable fields. slug comes from the URL,
// never the body (it is the immutable page key). All fields are optional — a
// blank page is a valid (if empty) page.
//
// Details and Content use JSON field-presence semantics:
//   - absent / null  → caller did not touch the structured block (markdown-only
//     edit); the repository KEEPS the existing value (COALESCE), never wiping it.
//   - present (object/array, even empty) → caller authored the block and it
//     REPLACES the stored one. A hero editor sends {} to fully clear a hero; a
//     section editor sends [] to clear all sections, so the removal persists
//     instead of being COALESCE-preserved.
//
// Content writes straight to published_content (the public-served slot) — there is
// no separate Publish step (Decision #0281). Published is the live gate: a *bool,
// so an absent value (a simple-page save that doesn't carry the toggle) KEEPS the
// stored flag and never silently unpublishes; a present value sets it.
type savePageRequest struct {
	Title        *string      `json:"title"`
	BodyMarkdown *string      `json:"body_markdown"`
	Details      *PageDetails `json:"details"`
	Content      *SectionList `json:"content"`
	Published    *bool        `json:"published"`
	// SeoMeta carries the per-page SEO/OG overrides. Same field-presence semantics
	// as Details: absent/null → KEEP the stored block; present (even {}) → REPLACE
	// it (an empty payload clears overrides to NULL).
	SeoMeta *seo.SeoMeta `json:"seo_meta"`
	// Layout selects the registered layout preset (bos-svelte's layout registry).
	// Scalar field-presence, same as Title/BodyMarkdown: absent/null → KEEP the
	// stored value; present → replace it.
	Layout *string `json:"layout"`
}

// slugMaxLen caps the slug param length before the kebab-case check (NFR-001).
const slugMaxLen = 100

// parseSlug pulls {slug} from the URL, enforces the param charset/length, and
// re-validates it as a kebab-case Slug. Writes the error response and returns
// ok=false on any failure.
func parseSlug(w http.ResponseWriter, r *http.Request) (Slug, bool) {
	raw, ok := httputil.ParseCode(w, chi.URLParam(r, "slug"), "slug", slugMaxLen)
	if !ok {
		return "", false
	}
	slug := Slug(raw)
	if err := slug.Validate(); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, err.Error(), httputil.CodeInvalidParam)
		return "", false
	}
	return slug, true
}

// fetchBySlug parses {slug}, guards a nil repo, and loads the live page — writing
// the error response and returning ok=false on any failure. Shared by the admin
// Get and the public GetPublic so the read + error mapping (404 absent/soft-deleted,
// 503 no repo, 500 unexpected) has a single definition.
func (h *pageHandler) fetchBySlug(w http.ResponseWriter, r *http.Request) (CmsPage, bool) {
	slug, ok := parseSlug(w, r)
	if !ok {
		return CmsPage{}, false
	}
	if h.repo == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "cms service unavailable", "service_unavailable")
		return CmsPage{}, false
	}
	p, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httputil.WriteError(w, http.StatusNotFound, "page not found", httputil.CodeNotFound)
			return CmsPage{}, false
		}
		log.Printf("cms.Get error slug=%s: %v", slug, err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to get page", httputil.CodeInternal)
		return CmsPage{}, false
	}
	return p, true
}

// Get returns the live CMS page for {slug} in the FULL admin shape (incl. audit
// fields). 404 when the slug has no live row.
func (h *pageHandler) Get(w http.ResponseWriter, r *http.Request) {
	p, ok := h.fetchBySlug(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// pageListItem is one row of the admin Pages list — enough to identify and open a
// page, not its full content (List callers load the full page via Get once a slug
// is picked, same as every other admin list-then-open pattern in this app).
type pageListItem struct {
	Slug      Slug      `json:"slug"`
	Title     *string   `json:"title,omitempty"`
	Published bool      `json:"published"`
	UpdatedAt time.Time `json:"updated_at"`
}

// List returns every live CMS page as a pageListItem, ordered by slug. Backs the
// admin Pages list (see Repository.List's own doc comment for why this exists).
func (h *pageHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.repo == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "cms service unavailable", "service_unavailable")
		return
	}
	pages, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("cms.List error: %v", err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to list pages", httputil.CodeInternal)
		return
	}
	items := make([]pageListItem, len(pages))
	for i, p := range pages {
		items[i] = pageListItem{Slug: p.Slug, Title: p.Title, Published: p.Published, UpdatedAt: p.UpdatedAt}
	}
	writeJSON(w, http.StatusOK, items)
}

// publicCmsPage is the PUBLIC projection of a CmsPage — only the content fields a
// visitor needs. It deliberately OMITS the audit/internal fields (id, created_by,
// updated_by, deleted_at) so the unauthenticated GET /pages/{slug} never leaks
// internal staff user ids. Details is a *PageDetails so it is omitted entirely from
// the JSON when the page has none. PublishedContent is the served structured-section
// list, attached ONLY when the page has been published (non-nil), so a page that
// has never published sections carries no "published_content" key and the reader
// falls back to body_markdown / details. The draft copy is NEVER exposed publicly.
type publicCmsPage struct {
	Slug             Slug         `json:"slug"`
	Title            *string      `json:"title,omitempty"`
	BodyMarkdown     *string      `json:"body_markdown,omitempty"`
	Details          *PageDetails `json:"details,omitempty"`
	PublishedContent *SectionList `json:"published_content,omitempty"`
	// Layout is always present (unlike the optional blocks above) — the public
	// renderer needs it to resolve which registered layout preset renders this
	// page's regions.
	Layout string `json:"layout"`
	// SeoMeta is attached only when the page carries overrides, so a page without
	// SEO settings emits no "seo_meta" key. Pointer + omitempty mirrors Details.
	SeoMeta   *seo.SeoMeta `json:"seo_meta,omitempty"`
	Version   int          `json:"version"`
	UpdatedAt time.Time    `json:"updated_at"`
}

// GetPublic returns the live CMS page for {slug} as the public projection (no audit
// fields). 404 when the slug is absent, soft-deleted, or not published.
func (h *pageHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	p, ok := h.fetchBySlug(w, r)
	if !ok {
		return
	}
	// Decision #0281 live gate: an unpublished page is invisible to the public —
	// 404 (indistinguishable from absent, so a draft never leaks its existence). The
	// shared fetchBySlug stays UNFILTERED so the admin Get can still load drafts to
	// edit; only this public projection enforces the gate.
	if !p.Published {
		httputil.WriteError(w, http.StatusNotFound, "page not found", httputil.CodeNotFound)
		return
	}
	proj := publicCmsPage{
		Slug:         p.Slug,
		Title:        p.Title,
		BodyMarkdown: p.BodyMarkdown,
		Layout:       p.Layout,
		Version:      p.Version,
		UpdatedAt:    p.UpdatedAt,
	}
	// Only attach details when the page actually carries a structured block, so a
	// markdown-only page renders no empty "details" object.
	if !p.Details.IsZero() {
		d := p.Details
		proj.Details = &d
	}
	// Attach the published sections only when the page has been published (non-nil),
	// so a page that never published carries no key and the loader uses the
	// body_markdown / details fallback.
	if p.PublishedContent != nil {
		pc := p.PublishedContent
		proj.PublishedContent = &pc
	}
	// Attach SEO overrides only when set, same omit-when-zero rule as Details.
	if !p.SeoMeta.IsZero() {
		sm := p.SeoMeta
		proj.SeoMeta = &sm
	}
	writeJSON(w, http.StatusOK, proj)
}

// Save upserts the CMS page for {slug}: creates it on first edit, else updates
// title + body_markdown + details + section content + published flag + seo_meta
// (bumping version). Each optional block is field-presence governed (see
// savePageRequest + Repository.UpsertBySlug). Returns the saved page. The author is
// stamped from the authenticated claims (updated_by).
func (h *pageHandler) Save(w http.ResponseWriter, r *http.Request) {
	slug, ok := parseSlug(w, r)
	if !ok {
		return
	}
	req, ok := httputil.DecodeAndValidate[savePageRequest](w, r, 0)
	if !ok {
		return
	}
	if h.repo == nil {
		httputil.WriteError(w, http.StatusServiceUnavailable, "cms service unavailable", "service_unavailable")
		return
	}

	var actorID string
	if claims := pkgauth.ClaimsFromContext(r.Context()); claims != nil {
		actorID = claims.Sub.String()
	}

	// Field-presence: a present details object (even {}) / content array (even [])
	// is authoritative and replaces the stored block; nil keeps it (markdown-only
	// edit). Published (*bool) flows through as the live gate — nil keeps the stored
	// value (never silently unpublishes). See savePageRequest + UpsertBySlug.
	replaceDetails := req.Details != nil
	replaceContent := req.Content != nil
	replaceSeoMeta := req.SeoMeta != nil
	p, err := h.repo.UpsertBySlug(r.Context(), slug, req.Title, req.BodyMarkdown, req.Details, replaceDetails, req.Content, replaceContent, req.Published, req.SeoMeta, replaceSeoMeta, req.Layout, actorID)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			httputil.WriteError(w, http.StatusUnprocessableEntity, err.Error(), httputil.CodeValidationFailed)
			return
		}
		log.Printf("cms.Save error slug=%s: %v", slug, err)
		httputil.WriteError(w, http.StatusInternalServerError, "failed to save page", httputil.CodeInternal)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// writeJSON writes a success-path JSON body. httputil only exports an error-shaped
// writer (WriteError) — this mirrors the same small helper bos-demo's own
// (pre-existing) handlers already use for the success path.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// RegisterAdmin mounts the admin CMS page routes on r: GET /cms (list), GET/PUT
// /cms/{slug}. The consumer wraps r in whatever auth/RBAC middleware it uses
// before calling this — these routes carry none of their own (mirrors
// settings.RegisterAdmin).
func RegisterAdmin(r chi.Router, repo *Repository) {
	h := &pageHandler{repo: repo}
	r.Get("/cms", h.List)
	r.Get("/cms/{slug}", h.Get)
	r.Put("/cms/{slug}", h.Save)
}

// RegisterPublic mounts the UNAUTHENTICATED public content read on r: GET
// /pages/{slug}. Published content is world-readable, no RBAC gate (mirrors
// settings.RegisterPublic).
func RegisterPublic(r chi.Router, repo *Repository) {
	h := &pageHandler{repo: repo}
	r.Get("/pages/{slug}", h.GetPublic)
}
