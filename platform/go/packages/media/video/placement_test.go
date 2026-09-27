package video_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/video"
)

func TestValidatePlacement(t *testing.T) {
	cases := []struct {
		name    string
		o       video.Orientation
		p       video.Placement
		wantErr bool
	}{
		{"portrait→shorts_rail allowed", video.OrientationPortrait, video.PlacementShortsRail, false},
		{"landscape→tours_grid allowed", video.OrientationLandscape, video.PlacementToursGrid, false},
		{"portrait→tours_grid allowed (short can be a tour)", video.OrientationPortrait, video.PlacementToursGrid, false},
		{"landscape→shorts_rail rejected (drives 422)", video.OrientationLandscape, video.PlacementShortsRail, true},
		{"invalid orientation rejected", video.Orientation("square"), video.PlacementToursGrid, true},
		{"invalid placement rejected", video.OrientationPortrait, video.Placement("hero"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := video.ValidatePlacement(tc.o, tc.p)
			if tc.wantErr && err == nil {
				t.Errorf("ValidatePlacement(%q,%q) = nil, want error", tc.o, tc.p)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidatePlacement(%q,%q) = %v, want nil", tc.o, tc.p, err)
			}
		})
	}
}
