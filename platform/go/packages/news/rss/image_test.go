package rss_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

func TestExtractImageFromContent(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string
	}{
		{
			name: "img inside figure",
			html: `<figure><img src="https://x.test/a.jpg" class="wp-post-image" loading="lazy" /></figure><p>body</p>`,
			want: "https://x.test/a.jpg",
		},
		{
			name: "img after paragraph",
			html: `<p>intro</p><img src="https://x.test/b.jpg" width="1280" />`,
			want: "https://x.test/b.jpg",
		},
		{
			name: "first of multiple",
			html: `<img src="https://x.test/first.jpg"><img src="https://x.test/second.jpg">`,
			want: "https://x.test/first.jpg",
		},
		{
			name: "single quotes",
			html: `<img src='https://x.test/c.jpg'>`,
			want: "https://x.test/c.jpg",
		},
		{
			name: "src not first attribute",
			html: `<img loading="lazy" decoding="async" src="https://x.test/d.jpg" width="100">`,
			want: "https://x.test/d.jpg",
		},
		{
			name: "no image",
			html: `<p>just text, no media here</p>`,
			want: "",
		},
		{
			name: "empty",
			html: "",
			want: "",
		},
		{
			name: "img without src",
			html: `<img alt="broken">`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rss.ExtractImageFromContent(tc.html); got != tc.want {
				t.Errorf("ExtractImageFromContent() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExtractOGImage(t *testing.T) {
	cases := []struct {
		name string
		html string
		want string
	}{
		{
			name: "property then content",
			html: `<meta property="og:image" content="https://x.test/og.jpg">`,
			want: "https://x.test/og.jpg",
		},
		{
			name: "content then property",
			html: `<meta content="https://x.test/og2.jpg" property="og:image"/>`,
			want: "https://x.test/og2.jpg",
		},
		{
			name: "single quotes",
			html: `<meta property='og:image' content='https://x.test/og3.png'>`,
			want: "https://x.test/og3.png",
		},
		{
			name: "name attribute variant",
			html: `<meta name="og:image" content="https://x.test/og4.jpg">`,
			want: "https://x.test/og4.jpg",
		},
		{
			name: "prefer og:image over secure_url",
			html: `<meta property="og:image" content="https://x.test/main.jpg">` +
				`<meta property="og:image:secure_url" content="https://x.test/secure.jpg">`,
			want: "https://x.test/main.jpg",
		},
		{
			name: "none",
			html: `<html><head><title>x</title></head></html>`,
			want: "",
		},
		{
			name: "empty",
			html: "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rss.ExtractOGImage(tc.html); got != tc.want {
				t.Errorf("ExtractOGImage() = %q, want %q", got, tc.want)
			}
		})
	}
}
