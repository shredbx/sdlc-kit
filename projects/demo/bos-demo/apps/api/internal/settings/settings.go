// Package settings is bos-demo's site-settings singleton: a title/tagline, plus the site's chrome
// (header/footer) — whether each renders, which chrome preset draws it, and its links
// (docs/proposals/bos-site-chrome.md). Unlike a page section's opaque content, chrome fields are
// known and validated here: the admin and the web-side chrome registry both need to agree on them.
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shredbx/sbx-core/pkg/database"
)

// NavItem is one entry in the header's navigation — a link (to a page or a literal URL) or a visual
// separator. web/mobile mirror bos-consumer-plan.md §13's header_nav[] shape; page_id/custom_path is
// the same content-type-reference duality bos-constructor.md uses elsewhere.
type NavItem struct {
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	PageID     *uuid.UUID `json:"page_id,omitempty"`
	CustomPath string     `json:"custom_path,omitempty"`
	Icon       string     `json:"icon,omitempty"`
	Enabled    bool       `json:"enabled"`
	Web        bool       `json:"web"`
	Mobile     bool       `json:"mobile"`
	NewTab     bool       `json:"new_tab"`
}

// Header is the site's header/topbar config.
type Header struct {
	Enabled bool      `json:"enabled"`
	Preset  string    `json:"preset"`
	Nav     []NavItem `json:"nav"`
}

// FooterLink is one link in a footer section — same page_id/custom_path duality as a NavItem, plus
// web/mobile visibility applied symmetrically (BR's own shape only has that on nav items).
type FooterLink struct {
	Title      string     `json:"title"`
	PageID     *uuid.UUID `json:"page_id,omitempty"`
	CustomPath string     `json:"custom_path,omitempty"`
	Enabled    bool       `json:"enabled"`
	Web        bool       `json:"web"`
	Mobile     bool       `json:"mobile"`
}

// FooterSection is one column of the footer — a heading and its links.
type FooterSection struct {
	Heading string       `json:"heading"`
	Links   []FooterLink `json:"links"`
}

// Footer is the site's footer config.
type Footer struct {
	Enabled  bool            `json:"enabled"`
	Preset   string          `json:"preset"`
	Sections []FooterSection `json:"sections"`
}

// validCustomPath rejects anything that isn't a site-relative path or an http(s)/mailto link —
// notably a "javascript:" scheme, which would otherwise execute for any visitor who clicks a nav or
// footer link built from this admin-entered value (found by automated security review).
func validCustomPath(path string) bool {
	if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") {
		return true
	}
	u, err := url.Parse(path)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "http", "https", "mailto":
		return true
	default:
		return false
	}
}

func validateNav(nav []NavItem) error {
	for i, n := range nav {
		if n.Type == "separator" {
			continue
		}
		if n.PageID == nil && n.CustomPath == "" {
			return fmt.Errorf("nav item %d (%q): needs a page_id or a custom_path", i, n.Title)
		}
		if n.CustomPath != "" && !validCustomPath(n.CustomPath) {
			return fmt.Errorf("nav item %d (%q): custom_path %q is not a valid link", i, n.Title, n.CustomPath)
		}
	}
	return nil
}

func validateFooter(sections []FooterSection) error {
	for si, sec := range sections {
		for li, l := range sec.Links {
			if l.PageID == nil && l.CustomPath == "" {
				return fmt.Errorf("footer section %d (%q) link %d (%q): needs a page_id or a custom_path", si, sec.Heading, li, l.Title)
			}
			if l.CustomPath != "" && !validCustomPath(l.CustomPath) {
				return fmt.Errorf("footer section %d (%q) link %d (%q): custom_path %q is not a valid link", si, sec.Heading, li, l.Title, l.CustomPath)
			}
		}
	}
	return nil
}

// Settings is the one row of the settings table (id = 1), the full admin-facing shape.
type Settings struct {
	SiteTitle string `json:"site_title"`
	Tagline   string `json:"tagline"`
	Header    Header `json:"header"`
	Footer    Footer `json:"footer"`
}

// PublicSettings is the trimmed shape the public web app needs to render a page's chrome — no
// admin-only fields.
type PublicSettings struct {
	SiteTitle string `json:"site_title"`
	Header    Header `json:"header"`
	Footer    Footer `json:"footer"`
}

// Store is a direct Postgres repository for the settings singleton.
type Store struct {
	db *database.DB
}

// NewStore wraps db for the settings table.
func NewStore(db *database.DB) *Store {
	return &Store{db: db}
}

// Get returns the singleton row.
func (s *Store) Get(ctx context.Context) (Settings, error) {
	var out Settings
	var headerRaw, footerRaw []byte
	row := s.db.QueryRow(ctx, "SELECT site_title, tagline, header, footer FROM settings WHERE id = 1")
	if err := row.Scan(&out.SiteTitle, &out.Tagline, &headerRaw, &footerRaw); err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(headerRaw, &out.Header); err != nil {
		return Settings{}, err
	}
	if err := json.Unmarshal(footerRaw, &out.Footer); err != nil {
		return Settings{}, err
	}
	return out, nil
}

// General is the site_title/tagline slice of Settings — returned on its own so a general-only
// update never implies anything about header/footer, which it does not touch.
type General struct {
	SiteTitle string `json:"site_title"`
	Tagline   string `json:"tagline"`
}

// UpdateGeneral writes the site title and tagline into the singleton row.
func (s *Store) UpdateGeneral(ctx context.Context, siteTitle, tagline string) (General, error) {
	var out General
	row := s.db.QueryRow(ctx,
		"UPDATE settings SET site_title = $1, tagline = $2 WHERE id = 1 RETURNING site_title, tagline",
		siteTitle, tagline,
	)
	err := row.Scan(&out.SiteTitle, &out.Tagline)
	return out, err
}

// UpdateHeader writes a new header config into the singleton row and returns what was stored.
func (s *Store) UpdateHeader(ctx context.Context, h Header) (Header, error) {
	raw, err := json.Marshal(h)
	if err != nil {
		return Header{}, err
	}
	var out []byte
	row := s.db.QueryRow(ctx, "UPDATE settings SET header = $1 WHERE id = 1 RETURNING header", json.RawMessage(raw))
	if err := row.Scan(&out); err != nil {
		return Header{}, err
	}
	var result Header
	err = json.Unmarshal(out, &result)
	return result, err
}

// UpdateFooter writes a new footer config into the singleton row and returns what was stored.
func (s *Store) UpdateFooter(ctx context.Context, f Footer) (Footer, error) {
	raw, err := json.Marshal(f)
	if err != nil {
		return Footer{}, err
	}
	var out []byte
	row := s.db.QueryRow(ctx, "UPDATE settings SET footer = $1 WHERE id = 1 RETURNING footer", json.RawMessage(raw))
	if err := row.Scan(&out); err != nil {
		return Footer{}, err
	}
	var result Footer
	err = json.Unmarshal(out, &result)
	return result, err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// RegisterAdmin mounts GET/PATCH /settings, PATCH /settings/header and PATCH /settings/footer on r.
// Each PATCH updates only its own concern — a header edit cannot accidentally wipe the footer, and
// vice versa, since each is decoded into its own typed request body.
func RegisterAdmin(r chi.Router, store *Store) {
	r.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
		out, err := store.Get(req.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Patch("/settings", func(w http.ResponseWriter, req *http.Request) {
		var in struct {
			SiteTitle string `json:"site_title"`
			Tagline   string `json:"tagline"`
		}
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		out, err := store.UpdateGeneral(req.Context(), in.SiteTitle, in.Tagline)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Patch("/settings/header", func(w http.ResponseWriter, req *http.Request) {
		var in Header
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := validateNav(in.Nav); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := store.UpdateHeader(req.Context(), in)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	})

	r.Patch("/settings/footer", func(w http.ResponseWriter, req *http.Request) {
		var in Footer
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if err := validateFooter(in.Sections); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		out, err := store.UpdateFooter(req.Context(), in)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	})
}

// RegisterPublic mounts the public read route (GET /settings, trimmed to PublicSettings) on r.
func RegisterPublic(r chi.Router, store *Store) {
	r.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
		full, err := store.Get(req.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, PublicSettings{SiteTitle: full.SiteTitle, Header: full.Header, Footer: full.Footer})
	})
}
