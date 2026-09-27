package cms

import (
	"encoding/json"
	"errors"
	"testing"
)

// =============================================================================
// TC-SELL-1 — every section kind round-trips through JSON intact (closed-kind
// adapter; SC-SELL-4). The wire shape is {"kind":K,"<K>":{payload}} so the
// payload key equals the discriminator — a new kind is a new adapter, never a
// core change.
// =============================================================================
func TestSectionList_RoundTripPerKind(t *testing.T) {
	original := SectionList{
		NewSection(SectionKindHero, &HeroSection{
			Eyebrow:  "SELL WITH BESTIE",
			Headline: "Sell your Koh Phangan property the honest way",
			Sub:      "We handle it end to end.",
		}),
		NewSection(SectionKindProse, &ProseSection{Markdown: "## Why\n\nLong-form body."}),
		NewSection(SectionKindFeatureCards, &FeatureCardsSection{Items: []FeatureCard{
			{Icon: "island", Title: "Island specialists", Body: "Born and raised on Phangan."},
			{Icon: "buyers", Title: "A vetted buyer pool", Body: "Qualified buyers only."},
		}}),
		NewSection(SectionKindSteps, &StepsSection{Items: []Step{
			{Title: "Tell us about it", Body: "Share the details."},
			{Title: "We visit & value", Body: "Honest valuation."},
		}}),
		NewSection(SectionKindFAQ, &FAQSection{Items: []FAQItem{
			{Q: "What does it cost?", A: "A simple commission."},
		}}),
	}

	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var back SectionList
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(back) != len(original) {
		t.Fatalf("len = %d; want %d", len(back), len(original))
	}

	// Hero
	if back[0].Kind != SectionKindHero {
		t.Errorf("section 0 kind = %q; want hero", back[0].Kind)
	}
	hero, ok := back[0].Payload.(*HeroSection)
	if !ok {
		t.Fatalf("section 0 payload = %T; want *HeroSection", back[0].Payload)
	}
	if hero.Headline != "Sell your Koh Phangan property the honest way" || hero.Eyebrow != "SELL WITH BESTIE" {
		t.Errorf("hero payload not preserved: %+v", hero)
	}

	// Prose
	if prose, ok := back[1].Payload.(*ProseSection); !ok || prose.Markdown != "## Why\n\nLong-form body." {
		t.Errorf("prose payload not preserved: %+v (%T)", back[1].Payload, back[1].Payload)
	}

	// FeatureCards
	fc, ok := back[2].Payload.(*FeatureCardsSection)
	if !ok || len(fc.Items) != 2 || fc.Items[0].Title != "Island specialists" {
		t.Errorf("featurecards payload not preserved: %+v (%T)", back[2].Payload, back[2].Payload)
	}

	// Steps
	st, ok := back[3].Payload.(*StepsSection)
	if !ok || len(st.Items) != 2 || st.Items[1].Title != "We visit & value" {
		t.Errorf("steps payload not preserved: %+v (%T)", back[3].Payload, back[3].Payload)
	}

	// FAQ
	fq, ok := back[4].Payload.(*FAQSection)
	if !ok || len(fq.Items) != 1 || fq.Items[0].Q != "What does it cost?" {
		t.Errorf("faq payload not preserved: %+v (%T)", back[4].Payload, back[4].Payload)
	}
}

// TestSection_WireShape pins the JSON envelope so the Svelte side can rely on it:
// {"kind":"hero","hero":{...}}.
func TestSection_WireShape(t *testing.T) {
	raw, err := json.Marshal(NewSection(SectionKindHero, &HeroSection{Headline: "H"}))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("Unmarshal to map: %v", err)
	}
	if _, ok := m["kind"]; !ok {
		t.Errorf("envelope missing kind: %s", raw)
	}
	if _, ok := m["hero"]; !ok {
		t.Errorf("envelope missing payload key matching kind: %s", raw)
	}
}

// =============================================================================
// TC-SELL-2 — an unknown / unregistered kind is rejected on decode (SC-SELL-4).
// Closed set: free-form kinds never silently produce an empty section.
// =============================================================================
func TestSection_UnknownKindRejected(t *testing.T) {
	var s Section
	err := json.Unmarshal([]byte(`{"kind":"banner","banner":{}}`), &s)
	if err == nil {
		t.Fatal("Unmarshal unknown kind = nil; want error")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("error = %v; want errors.Is ErrValidation", err)
	}
}

// =============================================================================
// TC-SELL-4 — nil SectionList (untouched) Values to SQL NULL; a present-but-empty
// list Values to a JSON array (the explicit-clear write path). Distinguishing the
// two is what makes clearing a section expressible (SC-SELL-5), exactly like the
// PageDetails replaceDetails flag.
// =============================================================================
func TestSectionList_ClearPersistsEmpty(t *testing.T) {
	// nil -> NULL (caller did not touch sections)
	v, err := SectionList(nil).Value()
	if err != nil {
		t.Fatalf("nil Value: %v", err)
	}
	if v != nil {
		t.Errorf("nil SectionList Value = %v; want nil (SQL NULL)", v)
	}

	// present-but-empty -> "[]" (caller cleared all sections)
	v, err = SectionList{}.Value()
	if err != nil {
		t.Fatalf("empty Value: %v", err)
	}
	b, ok := v.([]byte)
	if !ok || string(b) != "[]" {
		t.Errorf("empty SectionList Value = %v; want []byte(\"[]\")", v)
	}

	// Scan round-trips both: NULL -> nil, "[]" -> non-nil empty
	var l SectionList
	if err := l.Scan(nil); err != nil || l != nil {
		t.Errorf("Scan(nil) -> %v, %v; want nil,nil", l, err)
	}
	if err := l.Scan([]byte("[]")); err != nil {
		t.Fatalf("Scan([]): %v", err)
	}
	if l == nil || len(l) != 0 {
		t.Errorf("Scan([]) -> %v; want non-nil empty", l)
	}
}

// =============================================================================
// TC-SELL-6 (2607-041 scope-06) — a section's optional `hidden` flag round-trips
// through decode → validate → encode. The flag lives on the ENVELOPE (beside
// kind), so any kind carries it; default false is omitted from the wire
// (omitempty) so pre-flag payloads stay byte-identical.
// =============================================================================
func TestSection_HiddenFlagRoundTrips(t *testing.T) {
	// A featurecards section marked hidden decodes with Hidden = true...
	raw := []byte(`{"kind":"featurecards","hidden":true,"featurecards":{"items":[{"title":"Why","body":"B"}]}}`)
	var s Section
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !s.Hidden {
		t.Errorf("Hidden = false; want true (decoded from %s)", raw)
	}
	// Hidden is presentation, never a validity rule — a hidden section is still valid.
	if err := s.Validate(); err != nil {
		t.Errorf("Validate hidden section = %v; want nil", err)
	}

	// ...and re-encodes with "hidden":true preserved on the envelope.
	out, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("Unmarshal re-encoded to map: %v", err)
	}
	if string(m["hidden"]) != "true" {
		t.Errorf("re-encoded envelope hidden = %s; want true (full wire: %s)", m["hidden"], out)
	}

	// A visible (default) section omits the flag entirely (omitempty) so pre-flag
	// payloads round-trip byte-identically.
	visible, err := json.Marshal(NewSection(SectionKindFAQ, &FAQSection{Items: []FAQItem{{Q: "Q", A: "A"}}}))
	if err != nil {
		t.Fatalf("Marshal visible: %v", err)
	}
	// Fresh map — json.Unmarshal MERGES into an existing map, so reusing `m` would
	// carry the earlier "hidden":true key across and mask an omitempty regression.
	var vm map[string]json.RawMessage
	if err := json.Unmarshal(visible, &vm); err != nil {
		t.Fatalf("Unmarshal visible to map: %v", err)
	}
	if _, present := vm["hidden"]; present {
		t.Errorf("visible section emitted a hidden key: %s", visible)
	}
}

// TestSection_ValidateRejectsKindPayloadMismatch — a Section whose payload does
// not match its Kind is invalid (defensive: construction outside NewSection).
func TestSection_ValidateRejectsEmptyKind(t *testing.T) {
	s := Section{Kind: SectionKind(""), Payload: &HeroSection{}}
	if err := s.Validate(); err == nil {
		t.Error("Validate empty kind = nil; want error")
	}
}
