// Package cms is the BR-local domain for the CmsPage entity — an editable
// static content page (About, Contact, …) whose body is authored as markdown in
// the admin and rendered on the public site, keyed by a unique slug.
//
// CmsPage is BR-project-scoped (not a shared sbx-core primitive), so it lives in
// the app's internal/ tree alongside internal/agent. The struct, its validation,
// and the postgres Mapper are a projection of
// entities/cms-page/platforms/api-chi/md.yml — keep the column list here in exact
// sync with that md.yml and the migration (cms_pages).
//
// Nullable text columns are *string (NULL-safe scan + omitempty JSON), matching
// internal/agent and pkg/property's convention. The repository
// (internal/repository.CmsPageRepository) drives reads via the generic
// postgres.Store plus a pool-level upsert (UpsertBySlug) that the generic store
// cannot express.
package cms

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/shredbx/sbx-core/pkg/address"
	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
	"github.com/shredbx/sbx-core/pkg/seo"
	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

// ErrValidation wraps every validation failure. Callers test with
// errors.Is(err, cms.ErrValidation). Mirrors internal/agent.ErrValidation.
var ErrValidation = errors.New("validation")

// =============================================================================
// SLUG — the page URL key (a named type, never a raw string)
// =============================================================================

// slugPattern is the kebab-case constraint from entity.yml attribute `slug`:
// lowercase alphanumerics in dash-joined groups (e.g. "about", "contact-us").
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Slug is the unique, immutable URL key identifying a CmsPage (e.g. "about").
// It is a named type — never a raw string in signatures — so the kebab-case
// invariant travels with the value (standing rule: normalize discriminators).
type Slug string

// String renders the slug for SQL params and URLs.
func (s Slug) String() string { return string(s) }

// Validate enforces the entity.yml `slug` constraints: required (non-empty after
// trim) and kebab-case (^[a-z0-9]+(?:-[a-z0-9]+)*$).
func (s Slug) Validate() error {
	v := strings.TrimSpace(string(s))
	if v == "" {
		return errors.New("slug is required")
	}
	if !slugPattern.MatchString(v) {
		return errors.New("slug must be kebab-case (lowercase letters, digits, single dashes)")
	}
	return nil
}

// =============================================================================
// PAGE DETAILS — typed structured payload (JSONB), beyond the markdown body
// =============================================================================

// PageContact is the business contact block a content page may carry — the
// website's OWN contact info (the 'contact' page populates it; the public footer
// and the /contact page both render it). It is NOT an agent: this is the agency's
// own contact, editable independently of any property's assigned agent.
//
// Phone/WhatsApp/Email/Hours are display strings (the client formats the phone +
// normalizes WhatsApp to wa.me). Address reuses the canonical pkg/address value
// object (postal + optional WGS84 geo → a static map pin on /contact), so no
// parallel address shape is re-derived here.
//
// Name/Bio/ProfileImageURL/SocialNetworks/QRURL/QRImageURL converge this block
// onto the same contact-profile shape as the Agent card (A2): a business contact
// is presented like an agent (a name, a photo, a bio, social links, a scannable
// QR) without being one. SocialNetworks uses the promoted socialnetwork.List
// wrapper shared with agents + contacts. All fields stay in the existing details
// JSONB blob — adding them needs NO migration.
type PageContact struct {
	Name            string             `json:"name,omitempty" yaml:"name,omitempty"`
	Phone           string             `json:"phone,omitempty" yaml:"phone,omitempty"`
	WhatsApp        string             `json:"whatsapp,omitempty" yaml:"whatsapp,omitempty"`
	Email           string             `json:"email,omitempty" yaml:"email,omitempty"`
	Hours           string             `json:"hours,omitempty" yaml:"hours,omitempty"`
	Bio             string             `json:"bio,omitempty" yaml:"bio,omitempty"`
	ProfileImageURL string             `json:"profile_image_url,omitempty" yaml:"profile_image_url,omitempty"`
	Address         address.Address    `json:"address,omitempty" yaml:"address,omitempty"`
	SocialNetworks  socialnetwork.List `json:"social_networks,omitempty" yaml:"social_networks,omitempty"`
	QRURL           string             `json:"qr_url,omitempty" yaml:"qr_url,omitempty"`
	QRImageURL      string             `json:"qr_image_url,omitempty" yaml:"qr_image_url,omitempty"`
	// Public-render OPT-OUTS (zero value = shown, so existing contacts and every
	// prior save keep their current behaviour with no data change). HideMap drops the
	// office static-map even when the Address carries a pin — the pin can be set for
	// directions/geo without forcing the map onto the public page (there was no way to
	// unset a shown map once a location was pinned). HideSocialAccounts drops the
	// public "social account cards" band even when SocialNetworks is populated — the
	// links can drive footer/contact icons without the full card grid. Both stay in the
	// same details JSONB → NO migration.
	HideMap            bool `json:"hide_map,omitempty" yaml:"hide_map,omitempty"`
	HideSocialAccounts bool `json:"hide_social_accounts,omitempty" yaml:"hide_social_accounts,omitempty"`
}

// PageDetails is the OPTIONAL typed structured payload on a CmsPage, persisted as
// a single JSONB column. It is a TYPED value (never map[string]any): adding a
// future section (a hero, a stats block) is an additive struct field — not a
// free-form bag, not a migration. Today it carries an optional contact block plus
// an optional hero (a headline + a cover image, e.g. the /sell page) and an
// optional search prompt (the managed helper copy on the public /search hero).
//
// JSONB round-trip mirrors pkg/contact (SocialNetworkList / ExtensionData): a
// zero PageDetails Values to SQL NULL, and a NULL/empty source Scans back to zero
// — so markdown-only pages store NULL, never an empty blob.
type PageDetails struct {
	Contact       *PageContact `json:"contact,omitempty" yaml:"contact,omitempty"`
	Eyebrow       *string      `json:"eyebrow,omitempty" yaml:"eyebrow,omitempty"`
	Headline      *string      `json:"headline,omitempty" yaml:"headline,omitempty"`
	CoverImageURL *string      `json:"cover_image_url,omitempty" yaml:"cover_image_url,omitempty"`
	SearchPrompt  *string      `json:"search_prompt,omitempty" yaml:"search_prompt,omitempty"`

	// Body image (ABOUT-1, 2607-108) — an OPTIONAL image rendered BESIDE the markdown
	// body in a two-column layout on body-image-capable pages (About today).
	// BodyImagePosition picks the side the image sits on (left|right); a set URL with
	// nil position defaults to left in the reader. Absent URL → the page renders
	// text-only as before. ADDITIVE in the same details JSONB — no column, no
	// migration (mirrors CoverImageURL above).
	BodyImageURL      *string            `json:"body_image_url,omitempty" yaml:"body_image_url,omitempty"`
	BodyImagePosition *BodyImagePosition `json:"body_image_position,omitempty" yaml:"body_image_position,omitempty"`

	// CoverImageURLs is the optional ORDERED list of hero background images (2606-128).
	// >1 → the public hero renders a slideshow (crossfade); 1 → a single cover; empty →
	// falls back to CoverImageURL (legacy single), then the bg_color/gradient base. Each
	// entry is one of our R2 image URLs. ADDITIVE in the same details JSONB — no column,
	// no migration (mirrors the hero tint block below and SearchPrompt above).
	CoverImageURLs []string `json:"cover_image_urls,omitempty" yaml:"cover_image_urls,omitempty"`

	// Hero slideshow behavior (2606-128 tail) — surface-generic controls for the
	// CoverImageURLs slideshow, edited beside the image manager. All OPTIONAL
	// pointers; nil means "component default" (5s interval, zoom off). ADDITIVE in
	// the same details JSONB — no column, no migration. Ranges are enforced in
	// Validate (defense-in-depth; the admin form clamps to the same bounds).
	SlideDurationS *int           `json:"slide_duration_s,omitempty" yaml:"slide_duration_s,omitempty"`
	ZoomEnabled    *bool          `json:"zoom_enabled,omitempty" yaml:"zoom_enabled,omitempty"`
	ZoomDirection  *ZoomDirection `json:"zoom_direction,omitempty" yaml:"zoom_direction,omitempty"`
	ZoomScale      *float64       `json:"zoom_scale,omitempty" yaml:"zoom_scale,omitempty"`

	// Closing band (HOME-CMS, 2607-004 batch 3 #5) — the get-in-touch band above
	// the public footer (title · supporting line · the two CTA labels; the CTA
	// DESTINATIONS stay code-owned: contact → /about#contact, sell → /sell gated
	// on its publish flag). All OPTIONAL *string; nil → the shipped default copy.
	// ADDITIVE in the same details JSONB — no column, no migration. Length caps
	// in Validate.
	ClosingTitle        *string `json:"closing_title,omitempty" yaml:"closing_title,omitempty"`
	ClosingText         *string `json:"closing_text,omitempty" yaml:"closing_text,omitempty"`
	ClosingContactLabel *string `json:"closing_contact_label,omitempty" yaml:"closing_contact_label,omitempty"`
	ClosingSellLabel    *string `json:"closing_sell_label,omitempty" yaml:"closing_sell_label,omitempty"`

	// Homepage section order + FAQ block (2607-041 scope-08) — the homepage lets a
	// manager order its published-collection rails alongside the closing band and an
	// optional FAQ block. SectionsOrder is an ORDERED token list
	// (`collection:<id>` · `closing` · `faq`); empty/absent → the web derives the legacy
	// order on read (collections by sort_order, then closing), so a pre-migration home
	// keeps rendering unchanged with no migration. Token GRAMMAR is web-owned — Go stores
	// opaque strings and never rejects an unknown token (the reader skips it). FaqHome is
	// the optional homepage FAQ-block config (nil → no FAQ block). Both ADDITIVE in the
	// same details JSONB — no column, no migration (mirrors the closing-band block above).
	SectionsOrder []string       `json:"sections_order,omitempty" yaml:"sections_order,omitempty"`
	FaqHome       *FaqHomeConfig `json:"faq_home,omitempty" yaml:"faq_home,omitempty"`
	MapHome       *MapHomeConfig `json:"map_home,omitempty" yaml:"map_home,omitempty"`

	// BrowseFeaturedTypes (B5-6 / B4-1) — which "Browse by" property-type chips the
	// public homepage promotes into the highlighted featured slot (after the intent
	// pills). POINTER-to-slice for three-state field-presence: nil = unset → the
	// consumer's shipped default (BR: business); non-nil empty = the owner explicitly
	// un-featured everything (renders no featured chips — a plain []string+omitempty
	// could never round-trip this state); non-nil list = the featured codes in order.
	// Entries are property_types dictionary CODES (charset-gated in Validate);
	// a code with no published inventory is inert (the public band renders the
	// intersection with the live type list). ADDITIVE in the same details JSONB.
	BrowseFeaturedTypes *[]string `json:"browse_featured_types,omitempty" yaml:"browse_featured_types,omitempty"`

	// BgGradient (RP-7, 2607-004 batch 3 #7) — when true AND the hero has no
	// cover image, the public hero renders a generated diagonal gradient DERIVED
	// from bg_color (lighten → base → darken via CSS color-mix) instead of the
	// flat fill. A cover image always wins (its auto-scrim owns legibility,
	// RP-8). Plain optional bool — no recipe/values stored (the ramp is a code-
	// owned style, never manager-supplied CSS).
	BgGradient *bool `json:"bg_gradient,omitempty" yaml:"bg_gradient,omitempty"`

	// Optional hero-shell tint controls — each tints one slot of the per-page
	// hero (shell background, body text, title, subtitle, eyebrow). All OPTIONAL
	// *string (a hex/CSS color); a nil field means "inherit the brand default".
	// Like SearchPrompt these are ADDITIVE in the same details JSONB — no column,
	// no migration.
	BgColor       *string `json:"bg_color,omitempty" yaml:"bg_color,omitempty"`
	TextColor     *string `json:"text_color,omitempty" yaml:"text_color,omitempty"`
	TitleColor    *string `json:"title_color,omitempty" yaml:"title_color,omitempty"`
	SubtitleColor *string `json:"subtitle_color,omitempty" yaml:"subtitle_color,omitempty"`
	EyebrowColor  *string `json:"eyebrow_color,omitempty" yaml:"eyebrow_color,omitempty"`
}

// FaqHomeConfig is the optional homepage FAQ-block configuration (2607-041 scope-08).
// A nil *FaqHomeConfig on PageDetails means the homepage renders NO FAQ block. When
// present, the public home fetches the FAQ tree and renders the picked category's
// published items (CategorySlug empty → all categories, flattened), capped at Limit
// (0 → the consumer default), under Title (empty → the consumer's default heading).
// The FAQ CONTENT lives in pkg/faq — this only picks + caps it for the home surface,
// so no FAQ data is duplicated here. Every field omitempty so a bare enable
// ({}) round-trips as an all-defaults block.
type FaqHomeConfig struct {
	CategorySlug string `json:"category_slug,omitempty" yaml:"category_slug,omitempty"`
	Limit        int    `json:"limit,omitempty" yaml:"limit,omitempty"`
	Title        string `json:"title,omitempty" yaml:"title,omitempty"`
}

// MapHomeLevel is the homepage map section's grouping-level discriminator (ruling D1,
// task 2607-133) — a named type per the modeling standard (discriminators are never raw
// strings). It caps how deep the band groups listings into named areas.
type MapHomeLevel string

// The two permitted grouping levels (mirrored by the web's 'district' | 'sub_district' union).
const (
	MapHomeLevelDistrict    MapHomeLevel = "district"
	MapHomeLevelSubDistrict MapHomeLevel = "sub_district"
)

// MapHomeConfig is the optional homepage map-section configuration (task 2607-133, R1).
// A nil *MapHomeConfig on PageDetails means the homepage renders NO map section. When
// present, the public home renders the coverage map band (named area pills over the keyless
// public map) under Title (empty → the consumer's default heading), grouped at Level
// (empty → the consumer default). Storing Level here is what ruling D1 requires: managers
// switch the grouping from page management without a code change. The map DATA (listings +
// area centroids) comes from the property API — nothing geographic is duplicated here.
// Every field omitempty so a bare enable ({}) round-trips as an all-defaults block.
type MapHomeConfig struct {
	Title string       `json:"title,omitempty" yaml:"title,omitempty"`
	Level MapHomeLevel `json:"level,omitempty" yaml:"level,omitempty"`
}

// ZoomDirection is the hero-slideshow Ken Burns direction discriminator — a
// named type per the modeling standard (discriminators are never raw strings).
// "in" scales 1→zoom_scale over the slide interval; "out" the reverse.
type ZoomDirection string

// The two permitted zoom directions (mirrored by the web's 'in' | 'out' union).
const (
	ZoomDirectionIn  ZoomDirection = "in"
	ZoomDirectionOut ZoomDirection = "out"
)

// BodyImagePosition is the About body-image side discriminator (ABOUT-1) — a named
// type per the modeling standard (discriminators are never raw strings). It places
// the optional body image LEFT or RIGHT of the prose in the two-column body layout.
type BodyImagePosition string

// The two permitted body-image positions (mirrored by the web's 'left' | 'right' union).
const (
	BodyImagePositionLeft  BodyImagePosition = "left"
	BodyImagePositionRight BodyImagePosition = "right"
)

// cssColorPattern is the ALLOWED stored-color shape for every hero tint slot.
// The hero colors are rendered verbatim into a CSS custom-property value on the
// PUBLIC hero (e.g. `--hero-bg: <value>`), so an unvalidated value is a stored
// CSS-injection surface — `red;}` would terminate the declaration and let an
// editor smuggle a second rule, `url(...)` an external fetch, `<script>` markup.
// The editor is RBAC-gated, but this is defense-in-depth at the persistence
// boundary (NFR — the value renders to anonymous visitors).
//
// Only the three machine-friendly color forms are permitted (named colors like
// "red" and CSS keywords are intentionally NOT allowed — the picker emits hex):
//   - hex:  #rgb | #rrggbb | #rrggbbaa  (3, 6, or 8 hex digits)
//   - rgb:  rgb(…) | rgba(…)            (digits, commas, %, dots, spaces only)
//   - hsl:  hsl(…) | hsla(…)            (same restricted inner charset)
//
// The inner charset of the functional forms excludes every CSS-breaking char
// (; { } < > and parentheses other than the single wrapping pair), so a value
// matching the pattern cannot carry a smuggled declaration or markup.
// The functional-form name is matched case-insensitively ((?i:…)) because CSS
// color functions are case-insensitive (RGB(…) == rgb(…)); the hex digit classes
// already accept both cases.
var cssColorPattern = regexp.MustCompile(
	`^(?:#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})` +
		`|(?i:rgba?|hsla?)\([0-9.,%\s]+\))$`,
)

// validateColor checks one optional hero-color field. A nil pointer means
// "inherit the brand default" and is always allowed. A non-nil value is trimmed
// then matched against cssColorPattern; anything that does not match (a hostile
// `red;}`, a `url(...)`, a `<script>`, a named color, a malformed hex) is rejected
// with a clear error naming the offending field.
func validateColor(field string, v *string) error {
	if v == nil {
		return nil
	}
	if !cssColorPattern.MatchString(strings.TrimSpace(*v)) {
		return fmt.Errorf("%s must be a hex (#rgb/#rrggbb/#rrggbbaa), rgb()/rgba(), or hsl()/hsla() color", field)
	}
	return nil
}

// maxHeroImages caps the hero background slideshow list (NFR length-bound — the
// public hero renders one <img> layer per entry on every view).
const maxHeroImages = 24

// maxFeaturedBrowseTypes caps the featured browse-chip list (B5-6) — the featured
// slot is a marquee, not a second full type roster.
const maxFeaturedBrowseTypes = 6

// maxHomeSectionTokens caps the homepage section-order token list (2607-041 scope-08)
// — NFR length bound: the reader renders one block per token. Ample for every
// published collection plus the closing + faq blocks.
const maxHomeSectionTokens = 64

// maxHomeFaqItems caps the homepage FAQ block's item limit (2607-041 scope-08) — the
// home FAQ is a teaser onto /faq, not the full tree.
const maxHomeFaqItems = 24

// browseTypeCodePattern is the allowed shape of one featured browse entry: a
// dictionary CODE (lowercase alnum, hyphen/underscore separators, ≤50 chars —
// the property_types VARCHAR(50) grammar). Codes flow into public hrefs via the
// slugifying facet builders, so this is defense-in-depth, not the render gate.
var browseTypeCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,49}$`)

// Validate enforces the stored-color invariant for the optional hero tint slots:
// each non-nil color MUST be a well-formed CSS color (hex / rgb / hsl). It rejects
// (rather than silently strips) so the editor sees a clear error, and so a hostile
// value can never reach the publicly rendered CSS custom property. A nil color is
// allowed (inherit). Called by CmsPage.Validate on the upsert path.
func (d PageDetails) Validate() error {
	// Hero background slideshow list (2606-128): cap the count and reject empty
	// entries — each renders as an <img src> layer on the public hero.
	if len(d.CoverImageURLs) > maxHeroImages {
		return fmt.Errorf("cover_image_urls cannot exceed %d images", maxHeroImages)
	}
	for i, u := range d.CoverImageURLs {
		if strings.TrimSpace(u) == "" {
			return fmt.Errorf("cover_image_urls[%d] must not be empty", i)
		}
	}
	for _, c := range []struct {
		field string
		val   *string
	}{
		{"bg_color", d.BgColor},
		{"text_color", d.TextColor},
		{"title_color", d.TitleColor},
		{"subtitle_color", d.SubtitleColor},
		{"eyebrow_color", d.EyebrowColor},
	} {
		if err := validateColor(c.field, c.val); err != nil {
			return err
		}
	}
	// Slideshow behavior bounds (2606-128 tail) — mirror the admin form's clamps so a
	// bypassed client can never persist a degenerate interval or zoom.
	if d.SlideDurationS != nil && (*d.SlideDurationS < 2 || *d.SlideDurationS > 30) {
		return fmt.Errorf("slide_duration_s must be between 2 and 30 seconds")
	}
	if d.ZoomDirection != nil && *d.ZoomDirection != ZoomDirectionIn && *d.ZoomDirection != ZoomDirectionOut {
		return fmt.Errorf("zoom_direction must be \"in\" or \"out\"")
	}
	if d.ZoomScale != nil && (*d.ZoomScale < 1.01 || *d.ZoomScale > 1.5) {
		return fmt.Errorf("zoom_scale must be between 1.01 and 1.5")
	}
	// Body image side (ABOUT-1) — the discriminator must be one of the two known
	// positions when set (defense-in-depth; the reader also defaults nil → left).
	if d.BodyImagePosition != nil && *d.BodyImagePosition != BodyImagePositionLeft && *d.BodyImagePosition != BodyImagePositionRight {
		return fmt.Errorf("body_image_position must be \"left\" or \"right\"")
	}
	// Featured browse chips (B5-6) — cap the count and gate each entry to the
	// dictionary-code grammar. A nil pointer (unset → consumer default) and a
	// non-nil empty list (explicitly none) are both valid.
	if d.BrowseFeaturedTypes != nil {
		if len(*d.BrowseFeaturedTypes) > maxFeaturedBrowseTypes {
			return fmt.Errorf("browse_featured_types cannot exceed %d entries", maxFeaturedBrowseTypes)
		}
		for i, code := range *d.BrowseFeaturedTypes {
			if !browseTypeCodePattern.MatchString(code) {
				return fmt.Errorf("browse_featured_types[%d] is not a valid type code", i)
			}
		}
	}
	// Closing-band copy caps (HOME-CMS) — free text rendered as TEXT (Svelte
	// escapes), so length is the only boundary concern.
	if d.ClosingTitle != nil && len(*d.ClosingTitle) > 200 {
		return fmt.Errorf("closing_title exceeds 200 characters")
	}
	if d.ClosingText != nil && len(*d.ClosingText) > 500 {
		return fmt.Errorf("closing_text exceeds 500 characters")
	}
	if d.ClosingContactLabel != nil && len(*d.ClosingContactLabel) > 80 {
		return fmt.Errorf("closing_contact_label exceeds 80 characters")
	}
	if d.ClosingSellLabel != nil && len(*d.ClosingSellLabel) > 80 {
		return fmt.Errorf("closing_sell_label exceeds 80 characters")
	}
	// Home section order (2607-041 scope-08) — cap the token count (NFR length bound;
	// the reader renders one block per token). Tokens are opaque strings (web-owned
	// grammar), never grammar-checked here — an unknown token is skipped on read.
	if len(d.SectionsOrder) > maxHomeSectionTokens {
		return fmt.Errorf("sections_order cannot exceed %d entries", maxHomeSectionTokens)
	}
	// Homepage FAQ block (2607-041 scope-08) — bound the item cap and the copy lengths.
	if d.FaqHome != nil {
		if d.FaqHome.Limit < 0 || d.FaqHome.Limit > maxHomeFaqItems {
			return fmt.Errorf("faq_home.limit must be between 0 and %d", maxHomeFaqItems)
		}
		if len(d.FaqHome.Title) > 200 {
			return fmt.Errorf("faq_home.title exceeds 200 characters")
		}
		if len(d.FaqHome.CategorySlug) > 100 {
			return fmt.Errorf("faq_home.category_slug exceeds 100 characters")
		}
	}
	// Homepage map section (task 2607-133 R1) — bound the copy, gate the level discriminator.
	if d.MapHome != nil {
		if len(d.MapHome.Title) > 200 {
			return fmt.Errorf("map_home.title exceeds 200 characters")
		}
		switch d.MapHome.Level {
		case "", MapHomeLevelDistrict, MapHomeLevelSubDistrict:
		default:
			return fmt.Errorf("map_home.level must be %q or %q", MapHomeLevelDistrict, MapHomeLevelSubDistrict)
		}
	}
	return nil
}

// IsZero reports whether the details carry no structured data (→ stored as NULL).
func (d PageDetails) IsZero() bool {
	return d.Contact == nil && d.Eyebrow == nil && d.Headline == nil && d.CoverImageURL == nil &&
		len(d.CoverImageURLs) == 0 && d.SearchPrompt == nil &&
		d.BodyImageURL == nil && d.BodyImagePosition == nil &&
		d.BgColor == nil && d.TextColor == nil && d.TitleColor == nil && d.SubtitleColor == nil && d.EyebrowColor == nil &&
		d.SlideDurationS == nil && d.ZoomEnabled == nil && d.ZoomDirection == nil && d.ZoomScale == nil &&
		d.ClosingTitle == nil && d.ClosingText == nil && d.ClosingContactLabel == nil && d.ClosingSellLabel == nil &&
		d.BgGradient == nil && d.BrowseFeaturedTypes == nil &&
		len(d.SectionsOrder) == 0 && d.FaqHome == nil && d.MapHome == nil
}

// Value implements driver.Valuer for the JSONB column. A zero PageDetails returns
// nil → SQL NULL (no empty blob persisted).
func (d PageDetails) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return json.Marshal(d)
}

// Scan implements sql.Scanner. Accepts []byte (pgx default) or string; a NULL or
// empty/`null` source yields a zero PageDetails (Contact nil), not an error.
func (d *PageDetails) Scan(src any) error {
	if src == nil {
		*d = PageDetails{}
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("PageDetails.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*d = PageDetails{}
		return nil
	}
	return json.Unmarshal(data, d)
}

// =============================================================================
// CMSPAGE — the entity
// =============================================================================

// CmsPage is a single editable static page. Document-type: UUID identity,
// timestamps, soft delete, optimistic version. Field order in the struct is
// cosmetic; the storage column order is fixed by PostgresMapper.Columns()/FromRow().
type CmsPage struct {
	ID string `json:"id" yaml:"id"`

	Slug         Slug    `json:"slug" yaml:"slug"`
	Title        *string `json:"title,omitempty" yaml:"title,omitempty"`
	BodyMarkdown *string `json:"body_markdown,omitempty" yaml:"body_markdown,omitempty"`

	// Details is the optional typed structured payload (JSONB). Zero value (no
	// Contact) round-trips as SQL NULL. The public projection (handler) omits it
	// when empty; the admin shape carries it for round-trip editing.
	Details PageDetails `json:"details,omitempty" yaml:"details,omitempty"`

	// DraftContent / PublishedContent are the structured-section slots (JSONB).
	// Decision #0273 split them (draft working copy + published live copy); Decision
	// #0281 (AE4) SUPERSEDES that workflow with one DIRECT-SAVE: Save writes the
	// section content straight to PublishedContent (gated live by Published), so
	// DraftContent is retained for the expand phase but is no longer the source of
	// truth and is dropped in the #0281 contract cleanup. A nil list round-trips as
	// SQL NULL. Sections are ADDITIVE: Title / BodyMarkdown / Details remain for
	// pages with no sections and for the public fallback when PublishedContent is NULL.
	DraftContent     SectionList `json:"draft_content,omitempty" yaml:"draft_content,omitempty"`
	PublishedContent SectionList `json:"published_content,omitempty" yaml:"published_content,omitempty"`

	// Published is the live gate (Decision #0281, supersedes #0273 publish
	// workflow): the public read serves only published=true rows. Default false
	// (a new page is a draft). Mirrors content_entries.published.
	Published bool `json:"published" yaml:"published"`

	// SeoMeta is the optional per-page SEO + Open Graph override payload (P0-A,
	// Site hub) — the SHARED seo.SeoMeta value type (the same one property.seo_meta
	// carries). A zero value round-trips as SQL NULL (Value returns nil), so a page
	// without overrides derives its meta/OG from title + content at render time. The
	// public projection attaches it only when set; the admin shape carries it for
	// round-trip editing in the SEO facet.
	//
	// VALUE field (not the *seo.SeoMeta POINTER property uses): property routes its
	// SeoMeta through LoadSeoMeta/SaveSeoMeta companions OUTSIDE its GENERATED mapper
	// because codegen's `&p.Field` would yield an unscannable **SeoMeta. This mapper
	// is hand-written, so a value field scanned directly via `&p.SeoMeta` (a plain
	// *seo.SeoMeta — a valid sql.Scanner) is sound and needs no companion dance.
	//
	// Validate() length-gates meta_title (<=60) / meta_description (<=160) at save
	// (they render in the SERP). og_image/canonical_url are NOT escaped here — that is
	// the SeoHead renderer's job (HTML-attribute escaping), same as property; unlike
	// the hero-color path in PageDetails which DOES validate at persistence because it
	// renders into raw CSS (a more dangerous sink than an escaped <meta> attribute).
	SeoMeta seo.SeoMeta `json:"seo_meta,omitempty" yaml:"seo_meta,omitempty"`

	Version int `json:"version" yaml:"version"`

	CreatedBy *string `json:"created_by,omitempty" yaml:"created_by,omitempty"`
	UpdatedBy *string `json:"updated_by,omitempty" yaml:"updated_by,omitempty"`

	CreatedAt time.Time  `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" yaml:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" yaml:"deleted_at,omitempty"`
}

// maxBodyMarkdownLen is the entity.yml `body_markdown` maxLength constraint.
const maxBodyMarkdownLen = 100000

// Validate enforces the entity.yml constraints: slug is required + kebab-case,
// and body_markdown (when present) is within the 100k length cap. Title is
// optional free text.
func (p CmsPage) Validate() error {
	if err := p.Slug.Validate(); err != nil {
		return err
	}
	if p.BodyMarkdown != nil && len(*p.BodyMarkdown) > maxBodyMarkdownLen {
		return errors.New("body_markdown exceeds maximum length")
	}
	// Each non-nil hero tint color must be a well-formed CSS color — these render
	// verbatim into a publicly served CSS custom property, so a malformed/hostile
	// value is a stored-CSS-injection surface (defense-in-depth on the save path).
	if err := p.Details.Validate(); err != nil {
		return fmt.Errorf("details: %w", err)
	}
	// A structured contact address (when present) must honor the never-skip
	// geometry invariant — both coordinates or neither (ValidateGeometry). Postal
	// completeness is intentionally lenient: an office may list a city + country
	// without a full street line. The caller wraps the result as ErrValidation.
	if p.Details.Contact != nil {
		if err := p.Details.Contact.Address.ValidateGeometry(); err != nil {
			return fmt.Errorf("contact address: %w", err)
		}
	}
	if err := p.DraftContent.Validate(); err != nil {
		return fmt.Errorf("draft content: %w", err)
	}
	if err := p.PublishedContent.Validate(); err != nil {
		return fmt.Errorf("published content: %w", err)
	}
	// SEO overrides render publicly (title/description in the SERP, OG tags in
	// shares), so the SERP length invariants (meta_title <=60, meta_description
	// <=160 runes) are enforced at the save boundary — same as property.seo_meta.
	if err := p.SeoMeta.Validate(); err != nil {
		return fmt.Errorf("seo meta: %w", err)
	}
	return nil
}

// =============================================================================
// POSTGRES MAPPER — projection of entities/cms-page/platforms/api-chi/md.yml
// =============================================================================

// Fields exposes typed, compile-time-safe field descriptors for List filtering
// (e.g. cms.Fields.Slug.Eq("about")).
var Fields = struct {
	ID        repository.Field[string]
	Slug      repository.Field[string]
	DeletedAt repository.Field[string]
	CreatedAt repository.Field[string]
}{
	ID:        repository.Field[string]{Name: "id"},
	Slug:      repository.Field[string]{Name: "slug"},
	DeletedAt: repository.Field[string]{Name: "deleted_at"},
	CreatedAt: repository.Field[string]{Name: "created_at"},
}

// Compile-time check: PostgresMapper implements postgres.Mapper[CmsPage].
var _ postgres.Mapper[CmsPage] = PostgresMapper{}

// PostgresMapper maps CmsPage to the {schema}.cms_pages table for any schema.
type PostgresMapper struct {
	schema string
}

// NewPostgresMapper creates a mapper scoped to the given PostgreSQL schema.
func NewPostgresMapper(schema string) PostgresMapper {
	return PostgresMapper{schema: schema}
}

// TableName returns the fully qualified primary table for INSERT/UPDATE/DELETE.
func (m PostgresMapper) TableName() string {
	return m.schema + ".cms_pages"
}

// SelectFrom aliases the primary table — Columns() prefixes every field with the
// "c" alias, so the FROM clause must publish it.
func (m PostgresMapper) SelectFrom() string {
	return m.TableName() + " c"
}

// Columns returns the SELECT column list. Order MUST exactly match FromRow scan.
func (m PostgresMapper) Columns() []string {
	return []string{
		"c.id",
		"c.slug",
		"c.title",
		"c.body_markdown",
		"c.details",
		"c.draft_content",
		"c.published_content",
		"c.published",
		"c.seo_meta",
		"c.version",
		"c.created_by",
		"c.updated_by",
		"c.created_at",
		"c.updated_at",
		"c.deleted_at",
	}
}

// FieldColumn maps logical field names to aliased columns for WHERE clauses.
func (m PostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id":
		return "c.id"
	case "slug":
		return "c.slug"
	case "created_at":
		return "c.created_at"
	case "deleted_at":
		return "c.deleted_at"
	default:
		return field
	}
}

// ToRow converts a CmsPage to a column→value map for INSERT/UPDATE. deleted_at is
// omitted (DB default NULL). slug is stored as its string form.
func (m PostgresMapper) ToRow(p CmsPage) (map[string]any, error) {
	return map[string]any{
		"id":                p.ID,
		"slug":              p.Slug.String(),
		"title":             p.Title,
		"body_markdown":     p.BodyMarkdown,
		"details":           p.Details,
		"draft_content":     p.DraftContent,
		"published_content": p.PublishedContent,
		"published":         p.Published,
		"seo_meta":          p.SeoMeta,
		"version":           p.Version,
		"created_by":        p.CreatedBy,
		"updated_by":        p.UpdatedBy,
		"created_at":        p.CreatedAt,
		"updated_at":        p.UpdatedAt,
	}, nil
}

// FromRow scans a row into a CmsPage. Column order MUST exactly match Columns().
func (m PostgresMapper) FromRow(scan func(dest ...any) error) (CmsPage, error) {
	var p CmsPage
	err := scan(
		&p.ID,
		&p.Slug,
		&p.Title,
		&p.BodyMarkdown,
		&p.Details,
		&p.DraftContent,
		&p.PublishedContent,
		&p.Published,
		&p.SeoMeta,
		&p.Version,
		&p.CreatedBy,
		&p.UpdatedBy,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.DeletedAt,
	)
	if err != nil {
		return p, err
	}
	return p, nil
}
