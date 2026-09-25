package cms

import "fmt"

// Prose section adapter (Decision #0273) — the ONLY place markdown survives. The
// markup is sanitized at the render boundary on the public site, never here.

func init() {
	registerSectionKind(SectionKindProse, func() SectionPayload { return &ProseSection{} })
}

// ProseSection is long-form markdown body.
type ProseSection struct {
	Markdown string `json:"markdown,omitempty"`
}

// Validate caps the markdown at the same length as the legacy body_markdown.
func (p *ProseSection) Validate() error {
	if len(p.Markdown) > maxBodyMarkdownLen {
		return fmt.Errorf("%w: prose markdown exceeds maximum length", ErrValidation)
	}
	return nil
}
