package cms

import (
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

// EntryKind.Validate — the structural discriminator accepts only guide|service.
func TestEntryKind_Validate(t *testing.T) {
	tests := []struct {
		name    string
		kind    EntryKind
		wantErr bool
	}{
		{"guide ok", EntryKindGuide, false},
		{"service ok", EntryKindService, false},
		{"blog ok", EntryKindBlog, false},
		{"empty rejected", EntryKind(""), true},
		{"unknown rejected", EntryKind("article"), true},
		{"case-sensitive rejected", EntryKind("Guide"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.kind.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("EntryKind(%q).Validate() = nil, want error", tt.kind)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("EntryKind(%q).Validate() = %v, want nil", tt.kind, err)
			}
		})
	}
}

func TestEntryKind_String(t *testing.T) {
	if EntryKindGuide.String() != "guide" {
		t.Fatalf("EntryKindGuide.String() = %q, want guide", EntryKindGuide.String())
	}
}

// EntryDetails Value/Scan round-trip — zero → SQL NULL; service + media blocks
// survive a marshal/unmarshal cycle; NULL/empty/"null" scan back to zero.
func TestEntryDetails_ValueScan_RoundTrip(t *testing.T) {
	t.Run("zero values to NULL", func(t *testing.T) {
		var d EntryDetails
		if !d.IsZero() {
			t.Fatal("zero EntryDetails must report IsZero")
		}
		v, err := d.Value()
		if err != nil {
			t.Fatalf("Value() error: %v", err)
		}
		if v != nil {
			t.Fatalf("zero EntryDetails Value() = %v, want nil (SQL NULL)", v)
		}
	})

	t.Run("service block round-trips", func(t *testing.T) {
		in := EntryDetails{Service: &ServiceDetails{
			IncludedItems: []IncludedItem{{Title: "Tenant sourcing", Description: "Vetted tenants."}},
			Cta:           &CtaConfig{Label: "Message us", Href: "https://wa.me/66", Phone: "+66 98"},
		}}
		if in.IsZero() {
			t.Fatal("service block must not be zero")
		}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("Value() error: %v", err)
		}
		raw, ok := v.([]byte)
		if !ok {
			t.Fatalf("Value() type = %T, want []byte", v)
		}
		var out EntryDetails
		if err := out.Scan(raw); err != nil {
			t.Fatalf("Scan() error: %v", err)
		}
		if out.Service == nil || len(out.Service.IncludedItems) != 1 {
			t.Fatalf("service round-trip lost data: %+v", out)
		}
		if out.Service.IncludedItems[0].Title != "Tenant sourcing" {
			t.Fatalf("included item title = %q", out.Service.IncludedItems[0].Title)
		}
		if out.Service.Cta == nil || out.Service.Cta.Label != "Message us" {
			t.Fatalf("cta lost: %+v", out.Service.Cta)
		}
	})

	t.Run("media block round-trips", func(t *testing.T) {
		in := EntryDetails{Media: &MediaDetails{
			Gallery: []string{"https://img/a.jpg"},
			Reels:   []string{"https://tiktok/r"},
			Videos:  []string{"https://youtu.be/v"},
		}}
		v, err := in.Value()
		if err != nil {
			t.Fatalf("Value() error: %v", err)
		}
		var out EntryDetails
		if err := out.Scan(v.([]byte)); err != nil {
			t.Fatalf("Scan() error: %v", err)
		}
		if out.Media == nil || len(out.Media.Gallery) != 1 || out.Media.Videos[0] != "https://youtu.be/v" {
			t.Fatalf("media round-trip lost data: %+v", out.Media)
		}
	})

	t.Run("null and empty scan to zero", func(t *testing.T) {
		for _, src := range []any{nil, []byte(""), []byte("null"), "null"} {
			var d EntryDetails
			if err := d.Scan(src); err != nil {
				t.Fatalf("Scan(%v) error: %v", src, err)
			}
			if !d.IsZero() {
				t.Fatalf("Scan(%v) → not zero: %+v", src, d)
			}
		}
	})
}

// CmsEntry.Validate — positive + each negative case; errors are ErrValidation.
func TestCmsEntry_Validate(t *testing.T) {
	cat := "buying"
	long := strings.Repeat("x", maxEntryBodyLen+1)
	tests := []struct {
		name    string
		entry   CmsEntry
		wantErr bool
	}{
		{
			name:  "valid guide with category",
			entry: CmsEntry{Slug: "buying-a-condo", Kind: EntryKindGuide, Title: "Buying a Condo", Category: &cat},
		},
		{
			name:  "valid service no category",
			entry: CmsEntry{Slug: "property-management", Kind: EntryKindService, Title: "Property Management"},
		},
		{
			name:  "valid blog with category",
			entry: CmsEntry{Slug: "hello-world", Kind: EntryKindBlog, Title: "Hello World", Category: &cat},
		},
		{
			name:    "missing title",
			entry:   CmsEntry{Slug: "x", Kind: EntryKindGuide, Title: "   "},
			wantErr: true,
		},
		{
			name:    "bad slug",
			entry:   CmsEntry{Slug: "Bad Slug", Kind: EntryKindGuide, Title: "T"},
			wantErr: true,
		},
		{
			name:    "bad kind",
			entry:   CmsEntry{Slug: "x", Kind: EntryKind("article"), Title: "T"},
			wantErr: true,
		},
		{
			name:    "category on service rejected",
			entry:   CmsEntry{Slug: "x", Kind: EntryKindService, Title: "T", Category: &cat},
			wantErr: true,
		},
		{
			name:    "oversized body",
			entry:   CmsEntry{Slug: "x", Kind: EntryKindGuide, Title: "T", BodyMarkdown: &long},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Validate() = nil, want error")
				}
				if !errors.Is(err, ErrValidation) {
					t.Fatalf("Validate() error not ErrValidation: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() = %v, want nil", err)
			}
		})
	}
}

// PostgresMapper.Columns() count must equal the FromRow scan-destination count,
// and SelectFrom must alias the table `p` so p.search_vector resolves.
func TestEntryPostgresMapper_ColumnsMatchFromRow(t *testing.T) {
	m := NewEntryPostgresMapper("bestierealestate")

	if !strings.HasSuffix(m.SelectFrom(), " p") {
		t.Fatalf("SelectFrom must alias the table p: %q", m.SelectFrom())
	}
	for _, c := range m.Columns() {
		if !strings.HasPrefix(c, "p.") {
			t.Fatalf("every column must be p.-prefixed: %q", c)
		}
	}

	// Count scan destinations by running FromRow with a counting scanner.
	var n int
	_, err := m.FromRow(func(dest ...any) error {
		n = len(dest)
		return nil
	})
	if err != nil {
		t.Fatalf("FromRow counting scan error: %v", err)
	}
	if n != len(m.Columns()) {
		t.Fatalf("Columns() count %d != FromRow dest count %d", len(m.Columns()), n)
	}

	// TextSearchable contract.
	if m.SearchVectorColumn() != "p.search_vector" {
		t.Fatalf("SearchVectorColumn = %q", m.SearchVectorColumn())
	}
	if got := m.FuzzyTextFields(); len(got) != 1 || got[0] != "p.title" {
		t.Fatalf("FuzzyTextFields = %v, want [p.title]", got)
	}
}

// ToRow → (fake scan) FromRow round-trip for a guide and a service. The fake
// scanner replays the ToRow values in Columns() order, proving the column
// ordering + value mapping are mutually consistent. search_vector + deleted_at
// are NOT in ToRow (trigger-maintained / default), so they scan as zero.
func TestEntryPostgresMapper_ToRowFromRow_RoundTrip(t *testing.T) {
	m := NewEntryPostgresMapper("bestierealestate")
	cat := "buying"
	cover := "https://r2/cover.jpg"
	excerpt := "An excerpt"
	body := "## Body"
	author := "Bestie editorial"

	guide := CmsEntry{
		ID:            "11111111-1111-1111-1111-111111111111",
		Slug:          "buying-a-condo",
		Kind:          EntryKindGuide,
		Category:      &cat,
		Title:         "Buying a Condo",
		CoverImageURL: &cover,
		Excerpt:       &excerpt,
		BodyMarkdown:  &body,
		Published:     true,
		SortOrder:     5,
		Tags:          []string{"condo", "phuket"},
		Featured:      true,
		Author:        &author,
		Version:       1,
	}
	roundTrip(t, m, guide)

	service := CmsEntry{
		ID:        "22222222-2222-2222-2222-222222222222",
		Slug:      "property-management",
		Kind:      EntryKindService,
		Title:     "Property Management",
		Published: true,
		SortOrder: 10,
		Details: EntryDetails{Service: &ServiceDetails{
			IncludedItems: []IncludedItem{{Title: "Tenant sourcing", Description: "Vetted."}},
			Cta:           &CtaConfig{Label: "Message us", Href: "https://wa.me/66"},
		}},
		Version: 2,
	}
	roundTrip(t, m, service)
}

// roundTrip asserts ToRow → FromRow reproduces the durable fields. The replay
// scanner maps each Columns() name to its ToRow value (or zero for the
// trigger-maintained search_vector + the default deleted_at).
func roundTrip(t *testing.T, m EntryPostgresMapper, in CmsEntry) {
	t.Helper()
	row, err := m.ToRow(in)
	if err != nil {
		t.Fatalf("ToRow error: %v", err)
	}
	cols := m.Columns()
	out, err := m.FromRow(func(dest ...any) error {
		if len(dest) != len(cols) {
			t.Fatalf("dest %d != cols %d", len(dest), len(cols))
		}
		for i, c := range cols {
			name := strings.TrimPrefix(c, "p.")
			assignScan(t, dest[i], row[name])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FromRow error: %v", err)
	}
	if out.ID != in.ID || out.Slug != in.Slug || out.Kind != in.Kind || out.Title != in.Title {
		t.Fatalf("core fields diverged: %+v vs %+v", out, in)
	}
	if out.Published != in.Published || out.SortOrder != in.SortOrder || out.Featured != in.Featured {
		t.Fatalf("flags diverged: %+v vs %+v", out, in)
	}
	if in.Kind == EntryKindService {
		if out.Details.Service == nil || out.Details.Service.Cta == nil {
			t.Fatalf("service details lost: %+v", out.Details)
		}
	}
}

// assignScan emulates a sql driver assigning a stored value to a scan dest
// pointer. It handles the pointer + named types this mapper uses plus the
// EntryDetails Valuer/Scanner pair.
func assignScan(t *testing.T, dest, val any) {
	t.Helper()
	switch d := dest.(type) {
	case *string:
		if val != nil {
			*d = val.(string)
		}
	case *Slug:
		if val != nil {
			*d = Slug(val.(string))
		}
	case *EntryKind:
		if val != nil {
			*d = EntryKind(val.(string))
		}
	case **string:
		if val != nil {
			s := val.(*string)
			*d = s
		}
	case *bool:
		if val != nil {
			*d = val.(bool)
		}
	case *int:
		if val != nil {
			*d = val.(int)
		}
	case *[]string:
		if val != nil {
			*d = val.([]string)
		}
	case *EntryDetails:
		// ToRow stores Details as the EntryDetails value (a driver.Valuer). Marshal
		// then Scan to emulate the JSONB round-trip.
		v, ok := val.(EntryDetails)
		if !ok {
			return
		}
		dv, err := v.Value()
		if err != nil {
			t.Fatalf("details value: %v", err)
		}
		if dv == nil {
			return
		}
		if err := d.Scan(dv); err != nil {
			t.Fatalf("details scan: %v", err)
		}
	default:
		// time.Time / *time.Time and any nil-only columns — ignore for round-trip.
		_ = driver.Value(nil)
	}
}
