package rss

import (
	"html"
	"regexp"
	"strings"
)

// DefaultExcerptLen is the fallback excerpt length when a caller passes 0.
const DefaultExcerptLen = 280

var (
	// tagRe matches any HTML tag (greedy within a single tag).
	tagRe = regexp.MustCompile(`(?s)<[^>]*>`)
	// wsRe collapses any run of whitespace (incl. newlines) to one space.
	wsRe = regexp.MustCompile(`\s+`)
	// editorNoiseRe strips leaked rich-editor labels seen in some feeds
	// (ChatGPT/Claude code blocks). Matched case-insensitively on the
	// already-tag-stripped text so we only remove the stray label, not prose.
	editorNoiseRe = regexp.MustCompile(`(?i)\bcopy code\b`)
)

// Excerpt converts feed HTML into a clean, plain-text summary: strip tags,
// unescape HTML entities, collapse whitespace, drop leaked editor labels, then
// truncate on a word boundary with a trailing ellipsis when over maxLen.
//
// maxLen counts runes (not bytes) so multibyte Thai text truncates correctly.
// A maxLen <= 0 falls back to DefaultExcerptLen.
func Excerpt(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = DefaultExcerptLen
	}

	// 1. strip tags. Class/attribute soup (incl. leaked editor block wrappers
	//    like <div class="chatgpt-..." data-message-author-role>) lives inside
	//    tags, so tag-stripping removes it wholesale.
	text := tagRe.ReplaceAllString(s, " ")

	// 2. unescape entities. Feeds frequently double-encode (Bangkok Post emits
	//    "&amp;mdash;" for an em dash), so unescape until the result stabilises,
	//    bounded to a few passes to avoid pathological loops.
	text = unescapeFully(text)

	// 3. strip tags again: full unescaping can reveal tag-like sequences from
	//    double-encoded markup (e.g. "&amp;lt;b&amp;gt;"). A second pass keeps
	//    the excerpt strictly plain text — no '<' may survive.
	text = tagRe.ReplaceAllString(text, " ")
	text = strings.NewReplacer("<", " ", ">", " ").Replace(text)

	// 4. drop leaked editor labels that survive as plain text.
	text = editorNoiseRe.ReplaceAllString(text, " ")

	// 5. collapse whitespace and trim.
	text = strings.TrimSpace(wsRe.ReplaceAllString(text, " "))
	if text == "" {
		return ""
	}

	// 6. truncate on a word boundary if needed.
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	cut := string(runes[:maxLen])
	// Back off to the last space so we never cut a word in half. If the window
	// has no space (one very long token), keep the hard cut.
	if idx := strings.LastIndex(cut, " "); idx > 0 {
		cut = cut[:idx]
	}
	return strings.TrimRight(cut, " ") + "…"
}

// unescapeFully repeatedly HTML-unescapes until the string stops changing,
// capped at maxUnescapePasses to neutralise double-encoded feed content
// without risking an unbounded loop.
func unescapeFully(s string) string {
	const maxUnescapePasses = 3
	for i := 0; i < maxUnescapePasses; i++ {
		next := html.UnescapeString(s)
		if next == s {
			break
		}
		s = next
	}
	return s
}
