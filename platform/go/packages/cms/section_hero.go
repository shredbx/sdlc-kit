package cms

// Hero section adapter (Decision #0273) — the split-hero band: a mono eyebrow, an
// H1 headline, and an optional sub-headline. Registering here is the ONLY wiring a
// kind needs; the core (section.go) never changes.

func init() {
	registerSectionKind(SectionKindHero, func() SectionPayload { return &HeroSection{} })
}

// HeroSection is the hero payload.
type HeroSection struct {
	Eyebrow  string `json:"eyebrow,omitempty"`
	Headline string `json:"headline,omitempty"`
	Sub      string `json:"sub,omitempty"`
}

// Validate — all fields optional; a hero with no text is allowed (renders nothing).
func (h *HeroSection) Validate() error { return nil }
