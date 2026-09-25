package property

import (
	"reflect"
	"strconv"
	"testing"
)

// Locks the read-side `?tags=` normalizer contract (Decision #0280 M-B Unit 2):
// slug-normalize via TagSlug, drop empties, dedup first-seen, cap at
// MaxSearchTags. The slug FORMULA itself is covered by TagSlug's
// cross-engine golden tests; this guards the dedup/order/cap/empty-drop behavior
// that lives only here.
func TestNormalizeSearchTags(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil input", nil, nil},
		{"empty slice", []string{}, nil},
		{"all empty/whitespace -> nil", []string{"", "   ", "\t"}, nil},
		{"hyphen-only collapses to empty -> dropped", []string{"-", "  -  "}, nil},
		{"casing + whitespace folds to slug", []string{"Sea View"}, []string{"sea-view"}},
		{"dedup by slug, first-seen kept", []string{"Sea View", "sea view", "SEA VIEW"}, []string{"sea-view"}},
		{"input order preserved", []string{"B Tag", "A Tag"}, []string{"b-tag", "a-tag"}},
		{"empties dropped among valid", []string{"", "Pool", "  ", "Sea View"}, []string{"pool", "sea-view"}},
		{"already-slug is idempotent", []string{"sea-view", "pool"}, []string{"sea-view", "pool"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeSearchTags(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeSearchTags(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// The cap TRUNCATES (keeps the first MaxSearchTags distinct slugs in input order),
// it does not reject — a too-long ?tags= still returns a narrower result, never a 400.
func TestNormalizeSearchTags_capTruncates(t *testing.T) {
	in := make([]string, 0, MaxSearchTags+5)
	for i := 0; i < MaxSearchTags+5; i++ {
		in = append(in, "tag-"+strconv.Itoa(i))
	}
	got := NormalizeSearchTags(in)
	if len(got) != MaxSearchTags {
		t.Fatalf("len(got) = %d, want %d (cap must truncate)", len(got), MaxSearchTags)
	}
	if got[0] != "tag-0" || got[MaxSearchTags-1] != "tag-"+strconv.Itoa(MaxSearchTags-1) {
		t.Errorf("truncation kept wrong window: first=%q last=%q", got[0], got[len(got)-1])
	}
}
