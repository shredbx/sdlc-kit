package video

import "fmt"

// Placement is where a video is surfaced in a consumer UI. It is a dictionary-
// backed discriminator (never a raw string): the authoritative value set lives
// in the video.placement dictionary YAML.
type Placement string

const (
	// PlacementShortsRail is the horizontal rail of vertical short clips.
	PlacementShortsRail Placement = "shorts_rail"
	// PlacementToursGrid is the grid of (typically landscape) property tours;
	// a portrait short may also be placed here.
	PlacementToursGrid Placement = "tours_grid"
)

// ParsePlacement maps a raw string to a Placement, reporting whether the value
// is a known dictionary member. Unknown values map to ("", false).
func ParsePlacement(s string) (Placement, bool) {
	p := Placement(s)
	if p.Valid() {
		return p, true
	}
	return "", false
}

// Valid reports whether p is one of the known video.placement dictionary values.
func (p Placement) Valid() bool {
	switch p {
	case PlacementShortsRail, PlacementToursGrid:
		return true
	}
	return false
}

// String returns the raw dictionary code.
func (p Placement) String() string { return string(p) }

// ValidatePlacement enforces the orientation↔placement compatibility rule that
// drives the API's 422 on an invalid pairing:
//
//   - portrait → shorts_rail: OK (the canonical short).
//   - landscape → tours_grid: OK (the canonical tour).
//   - portrait → tours_grid: OK (a short may also be featured as a tour).
//   - landscape → shorts_rail: REJECTED (a horizontal video does not belong in
//     the vertical shorts rail).
//
// An unknown orientation or placement is also rejected.
func ValidatePlacement(o Orientation, p Placement) error {
	if !o.Valid() {
		return fmt.Errorf("video: invalid orientation %q", o)
	}
	if !p.Valid() {
		return fmt.Errorf("video: invalid placement %q", p)
	}
	if o == OrientationLandscape && p == PlacementShortsRail {
		return fmt.Errorf("video: landscape video cannot be placed in %s", PlacementShortsRail)
	}
	return nil
}
