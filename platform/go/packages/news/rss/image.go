package rss

import "regexp"

// imgSrcRe captures the src attribute of the first <img> tag, regardless of
// attribute order or quote style (single or double). The (?s) flag lets the
// tag span newlines.
var imgSrcRe = regexp.MustCompile(`(?is)<img\b[^>]*?\bsrc\s*=\s*["']([^"']+)["']`)

// ExtractImageFromContent returns the src of the first <img> in an HTML
// fragment (typically content:encoded), or "" when none is present. This is
// step 2 of the WordPress adapter's 3-step image fallback (media:* handled by
// the parser; the branded placeholder applied by the UI when both yield "").
func ExtractImageFromContent(html string) string {
	m := imgSrcRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// ogPropContentRe matches an Open Graph <meta> tag where property/name="og:image"
// precedes content="…". The \bog:image\b boundary excludes og:image:secure_url
// and og:image:width (those have a colon after "image", not a word boundary).
var ogPropContentRe = regexp.MustCompile(
	`(?is)<meta\b[^>]*?\b(?:property|name)\s*=\s*["']og:image["'][^>]*?\bcontent\s*=\s*["']([^"']+)["']`)

// ogContentPropRe matches the reverse attribute order (content="…" before
// property/name="og:image"). Both orders appear in the wild.
var ogContentPropRe = regexp.MustCompile(
	`(?is)<meta\b[^>]*?\bcontent\s*=\s*["']([^"']+)["'][^>]*?\b(?:property|name)\s*=\s*["']og:image["']`)

// ExtractOGImage returns the og:image URL from an article's HTML <head>, or ""
// when absent. It is the P3 step-3 image fallback: when the feed item carried no
// media:*/enclosure/content <img>, the refresh job fetches the article link and
// reads its Open Graph image. It deliberately matches only the canonical
// "og:image" (never og:image:secure_url / :width / :height) and tolerates either
// attribute order (property-then-content or content-then-property).
func ExtractOGImage(html string) string {
	if m := ogPropContentRe.FindStringSubmatch(html); len(m) >= 2 {
		return m[1]
	}
	if m := ogContentPropRe.FindStringSubmatch(html); len(m) >= 2 {
		return m[1]
	}
	return ""
}
