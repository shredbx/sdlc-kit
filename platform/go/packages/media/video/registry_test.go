package video_test

import (
	"errors"
	"testing"

	"github.com/shredbx/sbx-core/pkg/video"
)

func TestRegistry_Parse_Table(t *testing.T) {
	reg := video.NewRegistry(nil, video.NewMemoryCache())

	cases := []struct {
		name       string
		in         string
		wantPlat   video.Platform
		wantID     string
		wantOrient video.Orientation
		wantErr    bool // expect ErrUnsupportedURL
	}{
		{"youtube watch", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", video.PlatformYouTube, "dQw4w9WgXcQ", video.OrientationLandscape, false},
		{"youtu.be", "https://youtu.be/aR4Fk2", video.PlatformYouTube, "aR4Fk2", video.OrientationLandscape, false},
		{"youtube shorts → portrait", "https://www.youtube.com/shorts/tPEE9ZwTmy0", video.PlatformYouTube, "tPEE9ZwTmy0", video.OrientationPortrait, false},
		{"youtu.be query+timestamp stripped", "https://youtu.be/aR4Fk2?t=30&si=x", video.PlatformYouTube, "aR4Fk2", video.OrientationLandscape, false},
		{"m.youtube watch", "https://m.youtube.com/watch?v=abc123", video.PlatformYouTube, "abc123", video.OrientationLandscape, false},
		{"youtube embed no-www", "https://youtube.com/embed/xyz789", video.PlatformYouTube, "xyz789", video.OrientationLandscape, false},
		{"tiktok video", "https://www.tiktok.com/@scout2015/video/6718335390845095173", video.PlatformTikTok, "6718335390845095173", video.OrientationPortrait, false},
		{"instagram reel → portrait", "https://www.instagram.com/reel/CxYz123/", video.PlatformInstagram, "CxYz123", video.OrientationPortrait, false},
		{"instagram post → landscape", "https://www.instagram.com/p/CxYz123/", video.PlatformInstagram, "CxYz123", video.OrientationLandscape, false},
		{"trailing slash + whitespace trimmed", "  https://youtu.be/aR4Fk2/  ", video.PlatformYouTube, "aR4Fk2", video.OrientationLandscape, false},
		{"http accepted+normalized", "http://youtu.be/aR4Fk2", video.PlatformYouTube, "aR4Fk2", video.OrientationLandscape, false},
		{"vimeo unsupported", "https://vimeo.com/76979871", "", "", "", true},
		{"not a url", "hello world", "", "", "", true},
		{"empty", "", "", "", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reg.Parse(tc.in)
			if tc.wantErr {
				if !errors.Is(err, video.ErrUnsupportedURL) {
					t.Fatalf("Parse(%q) err = %v, want wraps ErrUnsupportedURL", tc.in, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) err = %v, want nil", tc.in, err)
			}
			if got.Platform != tc.wantPlat {
				t.Errorf("Parse(%q) platform = %q, want %q", tc.in, got.Platform, tc.wantPlat)
			}
			if got.ExternalID != tc.wantID {
				t.Errorf("Parse(%q) id = %q, want %q", tc.in, got.ExternalID, tc.wantID)
			}
			if got.Orientation != tc.wantOrient {
				t.Errorf("Parse(%q) orientation = %q, want %q", tc.in, got.Orientation, tc.wantOrient)
			}
			if got.CanonicalURL == "" {
				t.Errorf("Parse(%q) canonical URL is empty", tc.in)
			}
		})
	}
}

func TestRegistry_CanonicalURL_StripsQuery(t *testing.T) {
	reg := video.NewRegistry(nil, video.NewMemoryCache())
	p, err := reg.Parse("https://youtu.be/aR4Fk2?t=30&si=x")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := p.CanonicalURL; got != "https://www.youtube.com/watch?v=aR4Fk2" {
		t.Errorf("canonical URL = %q, want the query-stripped watch URL", got)
	}
}

func TestRegistry_Match(t *testing.T) {
	reg := video.NewRegistry(nil, video.NewMemoryCache())

	if _, ok := reg.Match("https://www.youtube.com/watch?v=x"); !ok {
		t.Error("Match should recognize a youtube URL")
	}
	if _, ok := reg.Match("https://www.tiktok.com/@u/video/1"); !ok {
		t.Error("Match should recognize a tiktok URL")
	}
	if _, ok := reg.Match("https://www.instagram.com/reel/x/"); !ok {
		t.Error("Match should recognize an instagram URL")
	}
	if _, ok := reg.Match("https://vimeo.com/123"); ok {
		t.Error("Match should reject an unsupported host")
	}
	if _, ok := reg.Match("not a url"); ok {
		t.Error("Match should reject a non-URL")
	}
}
