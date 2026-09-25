package rss_test

import (
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/rss"
)

func TestExcerpt_StripsTags(t *testing.T) {
	got := rss.Excerpt("<p>Hello <strong>world</strong></p>", 100)
	if strings.ContainsAny(got, "<>") {
		t.Errorf("excerpt still has angle brackets: %q", got)
	}
	if got != "Hello world" {
		t.Errorf("excerpt = %q, want %q", got, "Hello world")
	}
}

func TestExcerpt_UnescapesEntities(t *testing.T) {
	got := rss.Excerpt("HONG KONG &amp;mdash; capital flows &amp; trends", 100)
	// &amp;mdash; -> &mdash; -> — ; &amp; -> &
	if strings.Contains(got, "&amp;") || strings.Contains(got, "&mdash;") {
		t.Errorf("excerpt still contains escaped entities: %q", got)
	}
	if !strings.Contains(got, "—") {
		t.Errorf("excerpt should decode &mdash; to em dash: %q", got)
	}
}

func TestExcerpt_CollapsesWhitespace(t *testing.T) {
	got := rss.Excerpt("<p>line one</p>\n\n<p>line   two</p>", 100)
	if strings.Contains(got, "\n") {
		t.Errorf("excerpt should not contain newlines: %q", got)
	}
	if strings.Contains(got, "  ") {
		t.Errorf("excerpt should collapse runs of spaces: %q", got)
	}
	if got != "line one line two" {
		t.Errorf("excerpt = %q, want %q", got, "line one line two")
	}
}

func TestExcerpt_TruncatesOnWordBoundary(t *testing.T) {
	in := "the quick brown fox jumps over the lazy dog"
	got := rss.Excerpt(in, 20)
	if !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated excerpt should end with ellipsis: %q", got)
	}
	// The visible text (minus ellipsis) must be a whole-word prefix of input —
	// i.e. the original char right after it is a space (or we hit the end).
	body := strings.TrimRight(strings.TrimSuffix(got, "…"), " ")
	if !strings.HasPrefix(in, body) {
		t.Fatalf("truncated body %q is not a prefix of input", body)
	}
	if len(body) > 0 && len(body) < len(in) && in[len(body)] != ' ' {
		t.Errorf("truncation did not land on a word boundary: body=%q next=%q", body, string(in[len(body)]))
	}
	if len([]rune(got)) > 21 { // 20 + ellipsis rune
		t.Errorf("excerpt exceeds maxLen+ellipsis: %q (%d runes)", got, len([]rune(got)))
	}
}

func TestExcerpt_ShortInputUnchanged(t *testing.T) {
	got := rss.Excerpt("short text", 100)
	if got != "short text" {
		t.Errorf("excerpt = %q, want unchanged short text", got)
	}
	if strings.HasSuffix(got, "…") {
		t.Error("short input should not get an ellipsis")
	}
}

func TestExcerpt_RemovesLeakedEditorMarkup(t *testing.T) {
	// Prachachat economy items leak ChatGPT/Claude editor block wrappers and
	// stray "Copy code" labels. The excerpt must be clean prose only.
	in := `<div class="chatgpt-leaked-block" data-message-author-role="assistant"><p>ธุรกิจรับสร้างบ้านปี 2569 เผชิญวิกฤตแรงงานขาดแคลน</p></div><p style="color:#ccc">Copy code</p>`
	got := rss.Excerpt(in, 500)
	if strings.ContainsAny(got, "<>") {
		t.Errorf("excerpt leaked HTML: %q", got)
	}
	if strings.Contains(got, "chatgpt") || strings.Contains(got, "message-author-role") {
		t.Errorf("excerpt leaked editor class/attr markup: %q", got)
	}
	if strings.Contains(got, "Copy code") {
		t.Errorf("excerpt leaked 'Copy code' editor label: %q", got)
	}
	if !strings.Contains(got, "ธุรกิจรับสร้างบ้าน") {
		t.Errorf("excerpt dropped the real article text: %q", got)
	}
}

func TestExcerpt_EmptyAndDefaultLen(t *testing.T) {
	if got := rss.Excerpt("", 100); got != "" {
		t.Errorf("empty input should give empty excerpt, got %q", got)
	}
	if got := rss.Excerpt("<p></p>   ", 100); got != "" {
		t.Errorf("markup-only input should give empty excerpt, got %q", got)
	}
	// maxLen <= 0 falls back to DefaultExcerptLen (no panic, no zero-length cut)
	long := strings.Repeat("word ", 200)
	got := rss.Excerpt(long, 0)
	if len([]rune(got)) == 0 {
		t.Error("maxLen=0 should fall back to DefaultExcerptLen, not produce empty")
	}
	if len([]rune(got)) > rss.DefaultExcerptLen+1 { // +1 for ellipsis rune
		t.Errorf("excerpt longer than DefaultExcerptLen: %d runes", len([]rune(got)))
	}
}
