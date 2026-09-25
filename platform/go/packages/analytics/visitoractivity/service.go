package visitoractivity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// Sentinel errors returned by Service.Record. Callers map these to HTTP 4xx.
var (
	// ErrInvalidEvent — the event type or target type is not recognised.
	ErrInvalidEvent = errors.New("visitoractivity: invalid event or target type")
	// ErrTargetRequired — a property event was recorded without a target id.
	ErrTargetRequired = errors.New("visitoractivity: target id required for property events")
	// ErrPathRequired — a page event was recorded without a path (the URL path
	// IS the page identifier, so it must be non-empty) or carried a target id
	// (a page is keyed by its path, never a UUID).
	ErrPathRequired = errors.New("visitoractivity: page events require a path and no target id")
	// ErrPropsTooLarge — the marshalled props exceed maxPropsBytes.
	ErrPropsTooLarge = errors.New("visitoractivity: props payload too large")
)

// maxPropsBytes caps the marshalled props JSON. Keeps a single tracking call
// from storing an unbounded blob (DoS / storage abuse guard).
const maxPropsBytes = 2048

// Service validates, derives, and sanitizes a RecordInput before appending it.
// It is the only writer of Events — raw IP and user-agent enter here and are
// reduced to a VisitorID + DeviceType/IsBot, never persisted.
type Service struct {
	store Store
	salt  SaltProvider
}

// NewService wires a Service over a Store and a SaltProvider.
func NewService(store Store, salt SaltProvider) *Service {
	return &Service{store: store, salt: salt}
}

// Record validates the input, derives the anonymous visitor id + device bucket,
// sanitizes the referer, and appends an Event. The raw IP and user-agent are
// used only for derivation and are never written to the Event.
func (s *Service) Record(ctx context.Context, in RecordInput) error {
	if !in.EventType.Valid() || !in.TargetType.Valid() {
		return ErrInvalidEvent
	}
	if in.TargetType == TargetProperty && in.TargetID == nil {
		return ErrTargetRequired
	}
	// A page is keyed by its URL path (not a UUID): the path must be non-empty
	// and a target id must NOT be supplied.
	if in.TargetType == TargetPage && (in.Path == "" || in.TargetID != nil) {
		return ErrPathRequired
	}

	// Cap props before doing any work that touches the salt/DB.
	props, err := json.Marshal(in.Props)
	if err != nil {
		return fmt.Errorf("visitoractivity: marshal props: %w", err)
	}
	if len(props) > maxPropsBytes {
		return ErrPropsTooLarge
	}

	salt, err := s.salt.Salt(ctx)
	if err != nil {
		return fmt.Errorf("visitoractivity: salt: %w", err)
	}

	device, os, browser, isBot := ClassifyClient(in.UserAgent)

	ev := &Event{
		EventType:     in.EventType,
		TargetType:    in.TargetType,
		TargetID:      in.TargetID,
		VisitorID:     ComputeVisitorID(salt, in.IP, in.UserAgent),
		IsStaff:       in.IsStaff,
		Path:          in.Path,
		RefererOrigin: sanitizeReferer(in.RefererRaw),
		DeviceType:    device,
		OSFamily:      os,
		BrowserFamily: browser,
		Country:       normalizeCountry(in.Country),
		Language:      ParseLanguage(in.AcceptLanguage),
		IsBot:         isBot,
		Props:         in.Props,
		CreatedAt:     time.Now(),
	}
	return s.store.Append(ctx, ev)
}

// TopProperties — thin pass-through to the store.
func (s *Service) TopProperties(ctx context.Context, since time.Time, limit int) ([]TargetStat, error) {
	return s.store.TopProperties(ctx, since, limit)
}

// PropertyStats — thin pass-through to the store. Optional dims restricts which
// breakdown dimensions are computed (empty = the full registry).
func (s *Service) PropertyStats(ctx context.Context, targetID uuid.UUID, since time.Time, dims ...string) (*PropertyStats, error) {
	return s.store.PropertyStats(ctx, targetID, since, dims...)
}

// Overview — thin pass-through to the store. Optional dims restricts which
// breakdown dimensions are computed (empty = the full registry).
func (s *Service) Overview(ctx context.Context, since time.Time, dims ...string) (*Overview, error) {
	return s.store.Overview(ctx, since, dims...)
}

// PathStats — thin pass-through to the store. Returns the same per-page
// breakdown shape as PropertyStats, keyed by URL path instead of a target id.
// Optional dims restricts which breakdown dimensions are computed (empty = the
// full registry).
func (s *Service) PathStats(ctx context.Context, path string, since time.Time, dims ...string) (*PropertyStats, error) {
	return s.store.PathStats(ctx, path, since, dims...)
}

// PathSummary — thin pass-through to the store. Returns the all-time and
// today-distinct page-view counts for one URL path.
func (s *Service) PathSummary(ctx context.Context, path string) (PathSummary, error) {
	return s.store.PathSummary(ctx, path)
}

// sanitizeReferer reduces a raw Referer header to "scheme://host", dropping the
// path and query (which may carry tokens / PII). Only http/https origins are
// kept; anything unparseable, schemeless, hostless, or non-http returns "".
func sanitizeReferer(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	if u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}
