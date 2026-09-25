// Package socialnetwork provides a reusable embedded type for social media
// profile references.
//
// SocialNetwork is a value object with embedded reference semantics — consuming
// entities gain typed columns ({attr}_platform, {attr}_handle, {attr}_url) via
// 3-column expansion, following the same pattern as the GeoCoordinate type.
//
// Example:
//
//	ig := socialnetwork.NewSocialNetwork("instagram", "bestays_kohphangan")
//	ig.URL = "https://instagram.com/bestays_kohphangan"
//	if err := ig.Validate(); err != nil { ... }
package socialnetwork

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// SocialNetwork represents a social media profile reference.
// It is an immutable value object — create new instances rather than mutating.
type SocialNetwork struct {
	Platform string `json:"platform" yaml:"platform"`
	Handle   string `json:"handle" yaml:"handle"`
	URL      string `json:"url,omitempty" yaml:"url,omitempty"`
}

// NewSocialNetwork creates a SocialNetwork with the given platform and handle.
func NewSocialNetwork(platform, handle string) SocialNetwork {
	return SocialNetwork{Platform: platform, Handle: handle}
}

// Validate checks that the social network profile has required fields.
func (s SocialNetwork) Validate() error {
	if strings.TrimSpace(s.Platform) == "" {
		return fmt.Errorf("platform must not be empty")
	}
	if strings.TrimSpace(s.Handle) == "" {
		return fmt.Errorf("handle must not be empty")
	}
	return nil
}

// IsZero returns true if all social network fields are empty.
func (s SocialNetwork) IsZero() bool {
	return s.Platform == "" && s.Handle == "" && s.URL == ""
}

// HasURL returns true if the profile has a URL set.
func (s SocialNetwork) HasURL() bool {
	return s.URL != ""
}

// =============================================================================
// LIST — JSONB array round-trip
// =============================================================================

// List is a slice of SocialNetwork with Scan/Value methods so it round-trips
// through a JSONB column. It is the single reusable list wrapper shared by every
// consumer that stores a set of social-profile references (contacts, agents, the
// CMS business-contact block) — promote here rather than redefining per entity.
//
// An empty list serializes as SQL NULL (Value() returns nil); a NULL/empty/`null`
// source Scans back to a nil list, never an error.
type List []SocialNetwork

// Value implements driver.Valuer for JSONB columns. An empty list → SQL NULL.
func (l List) Value() (driver.Value, error) {
	if len(l) == 0 {
		return nil, nil
	}
	return json.Marshal(l)
}

// Scan implements sql.Scanner. Accepts []byte (pgx default) or string. A NULL,
// empty, or `null` source yields a nil list, not an error.
func (l *List) Scan(src any) error {
	if src == nil {
		*l = nil
		return nil
	}
	var data []byte
	switch v := src.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("socialnetwork.List.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*l = nil
		return nil
	}
	return json.Unmarshal(data, l)
}

// Validate checks every entry via SocialNetwork.Validate, returning the first
// failure annotated with its index. An empty list is valid.
func (l List) Validate() error {
	for i, sn := range l {
		if err := sn.Validate(); err != nil {
			return fmt.Errorf("social network entry index %d: %w", i, err)
		}
	}
	return nil
}
