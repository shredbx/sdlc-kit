package cms

// FAQ section adapter (Decision #0273) — the FAQ accordion: a repeatable list of
// question + answer pairs.

func init() {
	registerSectionKind(SectionKindFAQ, func() SectionPayload { return &FAQSection{} })
}

// FAQItem is one question/answer pair.
type FAQItem struct {
	Q string `json:"q,omitempty"`
	A string `json:"a,omitempty"`
}

// FAQSection is the repeatable Q/A list.
type FAQSection struct {
	Items []FAQItem `json:"items,omitempty"`
}

// Validate — Q/A are free text; no hard constraint.
func (s *FAQSection) Validate() error { return nil }
