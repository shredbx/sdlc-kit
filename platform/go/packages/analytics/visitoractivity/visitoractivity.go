// Package visitoractivity is a reusable, anonymous visitor-view tracking
// primitive for SBX projects. It records lightweight engagement events
// (property/listing views, clicks) keyed by a daily-rotated, salted HMAC
// "visitor id" that is intentionally NOT durable PII: it cannot be reversed to
// an IP/user-agent and rotates every UTC day so the same visitor maps to a
// different id the next day.
//
// The package never stores raw IP addresses or raw user-agent strings. The IP
// and user-agent flow into ComputeVisitorID (hashing) and ClassifyClient (coarse
// device/os/browser buckets) only; neither is persisted. Country (ISO-2) and
// language (ISO-639 subtag) are likewise coarse, derived-then-discarded
// attributes — never the raw IP nor the full Accept-Language header.
//
// Layering:
//   - visitoractivity.go  — domain types + validators (this file)
//   - identity.go         — HMAC visitor id + client/language/channel classification
//   - salt.go             — daily-rotated salt providers (memory + redis)
//   - schema.go           — schema-name allowlist (SQL-injection guard)
//   - store.go            — Postgres append + generic breakdown aggregation reads
//   - service.go          — Record (validate/derive/sanitize) + read pass-throughs
//   - http.go             — router-agnostic track + insights handlers
//
// Schema contract — the store expects this visitor_activity shape (each consumer
// project runs an additive, all-nullable migration; the store uses COALESCE so a
// missing/NULL value is safe):
//
//	id, created_at, event_type, target_type, target_id, visitor_id, is_staff,
//	path, referer_origin, device_type,
//	os_family      VARCHAR(16)  NULL,   -- coarse OS bucket (OSFamily)
//	browser_family VARCHAR(24)  NULL,   -- coarse browser bucket (BrowserFamily)
//	country_code   VARCHAR(2)   NULL,   -- ISO-3166-α2 (never the raw IP)
//	language       VARCHAR(8)   NULL,   -- ISO-639 primary subtag
//	is_bot, props
//
// channel/hour/weekday/utm_* are READ-SIDE breakdown dimensions — they have no
// stored column and are computed at query time from referer_origin/created_at/props.
//
// The Postgres store is hand-written (mirrors pkg/auth/audit.go) and is NOT a
// `sbx generate mapper` source — the entity.yml for this package is
// documentation-only.
package visitoractivity

import (
	"time"

	"github.com/google/uuid"
)

// EventType is the kind of engagement event recorded. The string values match
// the DB CHECK constraint on visitor_activity.event_type exactly.
type EventType string

// Allowed event types. Keep these aligned with the visitor_activity_event_type_chk
// constraint in the table DDL.
const (
	EventPropertyView EventType = "property_view"
	EventListingView  EventType = "listing_view"
	EventPageView     EventType = "page_view"
	EventClick        EventType = "click"
)

// Valid reports whether e is one of the recognised event types.
func (e EventType) Valid() bool {
	switch e {
	case EventPropertyView, EventListingView, EventPageView, EventClick:
		return true
	default:
		return false
	}
}

// TargetType is the kind of entity an event points at. The string values match
// the DB CHECK constraint on visitor_activity.target_type exactly.
type TargetType string

// Allowed target types. Keep these aligned with the
// visitor_activity_target_type_chk constraint in the table DDL.
const (
	TargetProperty TargetType = "property"
	TargetListing  TargetType = "listing"
	TargetPage     TargetType = "page"
)

// Valid reports whether t is one of the recognised target types.
func (t TargetType) Valid() bool {
	switch t {
	case TargetProperty, TargetListing, TargetPage:
		return true
	default:
		return false
	}
}

// DeviceType is a coarse device bucket derived from the user-agent. It is the
// only thing kept about the user-agent — the raw UA is never stored.
type DeviceType string

// Device buckets. Stored in visitor_activity.device_type (default "unknown").
const (
	DeviceMobile  DeviceType = "mobile"
	DeviceTablet  DeviceType = "tablet"
	DeviceDesktop DeviceType = "desktop"
	DeviceBot     DeviceType = "bot"
	DeviceUnknown DeviceType = "unknown"
)

// OSFamily is a coarse operating-system bucket derived from the user-agent. Like
// DeviceType it is the only thing kept about the OS — the raw UA is never stored.
type OSFamily string

// OS buckets. Stored in visitor_activity.os_family (default "" → OSOther on read
// via COALESCE). Closed set; unknowns fall into OSOther.
const (
	OSiOS     OSFamily = "ios"
	OSAndroid OSFamily = "android"
	OSWindows OSFamily = "windows"
	OSMacOS   OSFamily = "macos"
	OSLinux   OSFamily = "linux"
	OSOther   OSFamily = "other"
)

// Valid reports whether o is one of the recognised OS families.
func (o OSFamily) Valid() bool {
	switch o {
	case OSiOS, OSAndroid, OSWindows, OSMacOS, OSLinux, OSOther:
		return true
	default:
		return false
	}
}

// BrowserFamily is a coarse browser bucket derived from the user-agent.
type BrowserFamily string

// Browser buckets. Stored in visitor_activity.browser_family. Closed set;
// unknowns fall into BrowserOther.
const (
	BrowserChrome  BrowserFamily = "chrome"
	BrowserSafari  BrowserFamily = "safari"
	BrowserFirefox BrowserFamily = "firefox"
	BrowserEdge    BrowserFamily = "edge"
	BrowserSamsung BrowserFamily = "samsung"
	BrowserOther   BrowserFamily = "other"
)

// Valid reports whether b is one of the recognised browser families.
func (b BrowserFamily) Valid() bool {
	switch b {
	case BrowserChrome, BrowserSafari, BrowserFirefox, BrowserEdge, BrowserSamsung, BrowserOther:
		return true
	default:
		return false
	}
}

// Channel is a traffic-acquisition bucket classified from the stored
// referer_origin. It is a READ-SIDE dimension — never a stored column — so the
// Go classifier (ClassifyChannel) and the SQL CASE in the dimension registry
// must stay in lockstep.
type Channel string

// Traffic channels. Closed set.
const (
	ChannelDirect   Channel = "direct"
	ChannelSearch   Channel = "search"
	ChannelSocial   Channel = "social"
	ChannelReferral Channel = "referral"
)

// Valid reports whether c is one of the recognised traffic channels.
func (c Channel) Valid() bool {
	switch c {
	case ChannelDirect, ChannelSearch, ChannelSocial, ChannelReferral:
		return true
	default:
		return false
	}
}

// Event is a single persisted visitor-activity row. It deliberately carries NO
// raw IP and NO raw user-agent — those are reduced to VisitorID (HMAC),
// DeviceType/OSFamily/BrowserFamily/IsBot (coarse buckets) before an Event is
// ever constructed. Country/Language are likewise coarse, derived-then-discarded
// attributes (ISO-2 country code, ISO-639 language subtag) — never the raw IP or
// the full Accept-Language header.
type Event struct {
	ID            uuid.UUID      `json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	EventType     EventType      `json:"event_type"`
	TargetType    TargetType     `json:"target_type"`
	TargetID      *uuid.UUID     `json:"target_id,omitempty"`
	VisitorID     string         `json:"visitor_id"`
	IsStaff       bool           `json:"is_staff"`
	Path          string         `json:"path"`
	RefererOrigin string         `json:"referer_origin,omitempty"`
	DeviceType    DeviceType     `json:"device_type"`
	OSFamily      OSFamily       `json:"os_family,omitempty"`
	BrowserFamily BrowserFamily  `json:"browser_family,omitempty"`
	Country       string         `json:"country,omitempty"`  // ISO-3166-α2, "" = unknown
	Language      string         `json:"language,omitempty"` // ISO-639 primary subtag, "" = unknown
	IsBot         bool           `json:"is_bot"`
	Props         map[string]any `json:"props,omitempty"`
}

// RecordInput is the raw, pre-derivation input to Service.Record. The IP,
// UserAgent and AcceptLanguage fields are used for hashing/classification ONLY
// and are never persisted: the service derives VisitorID, DeviceType, OSFamily,
// BrowserFamily, IsBot and Language from them and then discards them. Country is
// the already-derived ISO-2 code (the handler resolves it from the trusted edge
// header), never the raw IP.
type RecordInput struct {
	EventType  EventType
	TargetType TargetType
	TargetID   *uuid.UUID
	Path       string
	RefererRaw string
	IsStaff    bool

	// IP is used solely as HMAC input for the visitor id. NEVER stored.
	IP string
	// UserAgent is used for HMAC input + device/os/browser classification. NEVER stored raw.
	UserAgent string
	// AcceptLanguage is the raw Accept-Language header. Reduced to a primary
	// language subtag (ParseLanguage) then discarded — never stored raw.
	AcceptLanguage string
	// Country is the already-resolved ISO-3166-α2 code (or "" for unknown). The
	// handler derives it from the trusted edge header; the raw IP is never seen here.
	Country string

	Props map[string]any
}
