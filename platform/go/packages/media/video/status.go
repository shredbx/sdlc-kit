package video

// Status is the publication state of a stored video. It is a dictionary-backed
// discriminator (never a raw string): the authoritative value set lives in the
// video.status dictionary YAML. A removed video is marked hidden (soft state),
// never hard-deleted, so it drops from public reads while staying auditable.
type Status string

const (
	// StatusPublished means the video participates in public reads.
	StatusPublished Status = "published"
	// StatusHidden means the video is withheld from public reads.
	StatusHidden Status = "hidden"
)

// ParseStatus maps a raw string to a Status, reporting whether the value is a
// known dictionary member. Unknown values map to ("", false).
func ParseStatus(s string) (Status, bool) {
	st := Status(s)
	if st.Valid() {
		return st, true
	}
	return "", false
}

// Valid reports whether s is one of the known video.status dictionary values.
func (s Status) Valid() bool {
	switch s {
	case StatusPublished, StatusHidden:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (s Status) String() string { return string(s) }
