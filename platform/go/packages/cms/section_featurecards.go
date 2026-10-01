package cms

// FeatureCards section adapter (Decision #0273) — the "Why sell with us" grid: a
// repeatable list of icon + title + body cards.

func init() {
	registerSectionKind(SectionKindFeatureCards, func() SectionPayload { return &FeatureCardsSection{} })
}

// FeatureCard is one card in the grid.
type FeatureCard struct {
	Icon  string `json:"icon,omitempty"`
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// FeatureCardsSection is the repeatable card grid.
type FeatureCardsSection struct {
	Items []FeatureCard `json:"items,omitempty"`
}

// Validate — cards are free text; empty cards are dropped by NonEmpty at the
// editor/save boundary, so no hard constraint here.
func (s *FeatureCardsSection) Validate() error { return nil }
