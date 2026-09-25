package cms

// Steps section adapter (Decision #0273) — the "How it works" list: ordered steps
// whose number is POSITIONAL (the renderer numbers them 1..n), so the payload
// carries only title + body.

func init() {
	registerSectionKind(SectionKindSteps, func() SectionPayload { return &StepsSection{} })
}

// Step is one ordered step (number is positional, not stored).
type Step struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
}

// StepsSection is the ordered step list.
type StepsSection struct {
	Items []Step `json:"items,omitempty"`
}

// Validate — steps are free text; positional numbering needs no constraint.
func (s *StepsSection) Validate() error { return nil }
