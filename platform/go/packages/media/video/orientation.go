package video

// Orientation is the aspect-ratio class of a video, derived at parse time from
// the URL shape (e.g. a YouTube /shorts/ path ⇒ portrait). It is a dictionary-
// backed discriminator (never a raw string): the authoritative value set lives
// in the video.orientation dictionary YAML.
type Orientation string

const (
	// OrientationPortrait is a vertical video (9:16-ish): YouTube Shorts,
	// TikTok, Instagram Reels.
	OrientationPortrait Orientation = "portrait"
	// OrientationLandscape is a horizontal video (16:9-ish): a standard
	// YouTube watch link, an Instagram post.
	OrientationLandscape Orientation = "landscape"
)

// ParseOrientation maps a raw string to an Orientation, reporting whether the
// value is a known dictionary member. Unknown values map to ("", false).
func ParseOrientation(s string) (Orientation, bool) {
	o := Orientation(s)
	if o.Valid() {
		return o, true
	}
	return "", false
}

// Valid reports whether o is one of the known video.orientation dictionary values.
func (o Orientation) Valid() bool {
	switch o {
	case OrientationPortrait, OrientationLandscape:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (o Orientation) String() string { return string(o) }
