package video_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/video"
)

func TestPlatform_Codes_And_Parse(t *testing.T) {
	codes := map[video.Platform]string{
		video.PlatformYouTube:   "youtube",
		video.PlatformTikTok:    "tiktok",
		video.PlatformInstagram: "instagram",
		video.PlatformFacebook:  "facebook",
	}
	for p, want := range codes {
		if string(p) != want {
			t.Errorf("platform code: got %q, want %q", string(p), want)
		}
		if !p.Valid() {
			t.Errorf("platform %q should be valid", p)
		}
	}
	cases := []struct {
		in     string
		want   video.Platform
		wantOK bool
	}{
		{"youtube", video.PlatformYouTube, true},
		{"tiktok", video.PlatformTikTok, true},
		{"instagram", video.PlatformInstagram, true},
		{"facebook", video.PlatformFacebook, true},
		{"vimeo", "", false},
		{"", "", false},
		{"YouTube", "", false}, // case-sensitive dictionary code
	}
	for _, tc := range cases {
		got, ok := video.ParsePlatform(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParsePlatform(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestOrientation_Codes_And_Parse(t *testing.T) {
	if string(video.OrientationPortrait) != "portrait" {
		t.Errorf("OrientationPortrait: got %q", video.OrientationPortrait)
	}
	if string(video.OrientationLandscape) != "landscape" {
		t.Errorf("OrientationLandscape: got %q", video.OrientationLandscape)
	}
	cases := []struct {
		in     string
		want   video.Orientation
		wantOK bool
	}{
		{"portrait", video.OrientationPortrait, true},
		{"landscape", video.OrientationLandscape, true},
		{"square", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := video.ParseOrientation(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParseOrientation(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestPlacement_Codes_And_Parse(t *testing.T) {
	if string(video.PlacementShortsRail) != "shorts_rail" {
		t.Errorf("PlacementShortsRail: got %q", video.PlacementShortsRail)
	}
	if string(video.PlacementToursGrid) != "tours_grid" {
		t.Errorf("PlacementToursGrid: got %q", video.PlacementToursGrid)
	}
	cases := []struct {
		in     string
		want   video.Placement
		wantOK bool
	}{
		{"shorts_rail", video.PlacementShortsRail, true},
		{"tours_grid", video.PlacementToursGrid, true},
		{"hero", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := video.ParsePlacement(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParsePlacement(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestStatus_Codes_And_Parse(t *testing.T) {
	if string(video.StatusPublished) != "published" {
		t.Errorf("StatusPublished: got %q", video.StatusPublished)
	}
	if string(video.StatusHidden) != "hidden" {
		t.Errorf("StatusHidden: got %q", video.StatusHidden)
	}
	cases := []struct {
		in     string
		want   video.Status
		wantOK bool
	}{
		{"published", video.StatusPublished, true},
		{"hidden", video.StatusHidden, true},
		{"draft", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := video.ParseStatus(tc.in)
		if got != tc.want || ok != tc.wantOK {
			t.Errorf("ParseStatus(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}
