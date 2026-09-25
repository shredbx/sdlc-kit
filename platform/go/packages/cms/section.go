// Structured page sections — Decision #0273.
//
// A section-capable CmsPage is an ORDERED list of TYPED, closed-kind sections
// (SectionList) with a draft/published split (see cmspage.go DraftContent /
// PublishedContent). Each kind is an ADAPTER: a payload struct in its own
// section_<kind>.go file that registers a constructor in init(). A new kind is a
// new file + one registerSectionKind call — ZERO change to this core (project governance
// rule #9: new variant = new adapter). Free-form / unknown kinds are rejected on
// decode, so the set stays closed.
//
// Wire shape: {"kind":"hero","hero":{...}} — the payload key EQUALS the
// discriminator, so the Svelte renderer reads section[section.kind] directly.
//
// SectionList round-trips through JSONB via driver.Valuer/sql.Scanner exactly
// like cms.PageDetails: a nil list (caller never touched sections) Values to SQL
// NULL; a present-but-empty list ([]) Values to a JSON array — the field-presence
// signal that makes "clear all sections" expressible (mirrors the PageDetails
// replaceDetails flag).
package cms

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// SectionKind is the closed, named discriminator for a Section (never a raw
// string in signatures). The valid set is exactly the kinds registered by the
// section_<kind>.go adapter files.
type SectionKind string

const (
	SectionKindHero         SectionKind = "hero"
	SectionKindProse        SectionKind = "prose"
	SectionKindFeatureCards SectionKind = "featurecards"
	SectionKindSteps        SectionKind = "steps"
	SectionKindFAQ          SectionKind = "faq"
)

// SectionPayload is the kind-specific typed body of a Section. Each adapter file
// defines one payload type implementing this interface and registers its
// constructor — the only contract a new kind must satisfy.
type SectionPayload interface {
	// Validate enforces the kind's payload invariants (wrap cms.ErrValidation).
	Validate() error
}

// sectionRegistry maps each registered kind to a constructor for its zero
// payload. Populated by the section_<kind>.go init() functions.
var sectionRegistry = map[SectionKind]func() SectionPayload{}

// registerSectionKind wires a kind's payload constructor. Called from each
// adapter file's init(); duplicate registration is a programming error.
func registerSectionKind(kind SectionKind, ctor func() SectionPayload) {
	if _, dup := sectionRegistry[kind]; dup {
		panic("cms: duplicate section kind registered: " + string(kind))
	}
	sectionRegistry[kind] = ctor
}

// Registered reports whether the kind has an adapter (is in the closed set).
func (k SectionKind) Registered() bool {
	_, ok := sectionRegistry[k]
	return ok
}

// Validate requires a non-empty, registered kind.
func (k SectionKind) Validate() error {
	if k == "" {
		return fmt.Errorf("%w: section kind is required", ErrValidation)
	}
	if !k.Registered() {
		return fmt.Errorf("%w: unknown section kind %q", ErrValidation, k)
	}
	return nil
}

// Section is one entry in a page's content — a kind plus its typed payload.
type Section struct {
	Kind    SectionKind
	Payload SectionPayload
	// Hidden marks a section authored-but-not-rendered: it stays in the editor and
	// round-trips through storage, but the public loader filters it out (2607-041
	// scope-06). Envelope-level (any kind may carry it) and optional; default false
	// = visible, and omitted from the wire when false (omitempty via MarshalJSON) so
	// pre-flag payloads stay byte-identical.
	Hidden bool `json:"hidden,omitempty"`
}

// NewSection builds a section from a kind + its payload (the construction path
// used by seeds, the renderer, and tests).
func NewSection(kind SectionKind, payload SectionPayload) Section {
	return Section{Kind: kind, Payload: payload}
}

// Validate checks the kind is in the closed set and the payload is present + valid.
func (s Section) Validate() error {
	if err := s.Kind.Validate(); err != nil {
		return err
	}
	if s.Payload == nil {
		return fmt.Errorf("%w: section %q has no payload", ErrValidation, s.Kind)
	}
	return s.Payload.Validate()
}

// MarshalJSON emits {"kind":K,"<K>":payload}. The payload key equals the kind so
// the client reads section[section.kind].
func (s Section) MarshalJSON() ([]byte, error) {
	payload, err := json.Marshal(s.Payload)
	if err != nil {
		return nil, err
	}
	kindJSON, err := json.Marshal(string(s.Kind))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString(`{"kind":`)
	buf.Write(kindJSON)
	buf.WriteByte(',')
	buf.Write(kindJSON) // payload key == kind
	buf.WriteByte(':')
	if len(payload) == 0 {
		buf.WriteString("null")
	} else {
		buf.Write(payload)
	}
	// omitempty: the flag rides the envelope only when set, so a visible section is
	// byte-identical to the pre-flag wire.
	if s.Hidden {
		buf.WriteString(`,"hidden":true`)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON reads the kind, rejects any unregistered kind (closed set), then
// decodes the payload sub-object into the kind's registered type.
func (s *Section) UnmarshalJSON(b []byte) error {
	var head struct {
		Kind   SectionKind `json:"kind"`
		Hidden bool        `json:"hidden"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return err
	}
	if err := head.Kind.Validate(); err != nil {
		return err
	}
	payload := sectionRegistry[head.Kind]()

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if pj, ok := raw[string(head.Kind)]; ok && len(pj) > 0 && string(pj) != "null" {
		if err := json.Unmarshal(pj, payload); err != nil {
			return fmt.Errorf("%w: section %q payload: %v", ErrValidation, head.Kind, err)
		}
	}
	s.Kind = head.Kind
	s.Payload = payload
	s.Hidden = head.Hidden
	return nil
}

// SectionList is the ordered content of a page (draft or published copy),
// persisted as a JSONB array.
type SectionList []Section

// Value implements driver.Valuer. A nil list (untouched) Values to SQL NULL; a
// present-but-empty list Values to "[]" (the explicit-clear write path).
func (l SectionList) Value() (driver.Value, error) {
	if l == nil {
		return nil, nil
	}
	return json.Marshal(l)
}

// Scan implements sql.Scanner. NULL/empty/`null` -> nil; "[]" -> a non-nil empty
// list; a populated array -> the decoded sections.
func (l *SectionList) Scan(src any) error {
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
		return fmt.Errorf("SectionList.Scan: unsupported source type %T", src)
	}
	if len(data) == 0 || string(data) == "null" {
		*l = nil
		return nil
	}
	var out []Section
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	if out == nil { // a present "[]" decodes to nil — keep it non-nil (present empty)
		out = []Section{}
	}
	*l = out
	return nil
}

// Validate checks every section in the list.
func (l SectionList) Validate() error {
	for i, s := range l {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("section %d: %w", i, err)
		}
	}
	return nil
}
