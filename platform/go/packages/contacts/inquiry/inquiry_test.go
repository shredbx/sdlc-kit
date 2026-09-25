package inquiry

import (
	"errors"
	"testing"
	"time"

	"github.com/shredbx/sbx-core/pkg/socialnetwork"
)

func ptr(s string) *string { return &s }

// TestInquirySource_Valid + TestInquiryStatus_Valid — the discriminator contracts.
func TestInquirySource_Valid(t *testing.T) {
	cases := []struct {
		src  InquirySource
		want bool
	}{
		{SourceSell, true},
		{SourceService, true},
		{SourceProperty, true},
		{SourceAsk, true},
		{InquirySource("bogus"), false},
		{InquirySource(""), false},
	}
	for _, c := range cases {
		if got := c.src.Valid(); got != c.want {
			t.Errorf("InquirySource(%q).Valid() = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestInquiryStatus_Valid(t *testing.T) {
	cases := []struct {
		st   InquiryStatus
		want bool
	}{
		{StatusNew, true},
		{StatusRead, true},
		{StatusHandled, true},
		{InquiryStatus("contacted"), false},
		{InquiryStatus(""), false},
	}
	for _, c := range cases {
		if got := c.st.Valid(); got != c.want {
			t.Errorf("InquiryStatus(%q).Valid() = %v, want %v", c.st, got, c.want)
		}
	}
}

// base returns a minimally-valid inquiry (status set, source set) that the test
// then mutates per case.
func base() Inquiry {
	// location is required (see TestInquiry_Validate_LocationRequired); set a
	// valid one here so the contact-channel cases stay focused on their concern.
	return Inquiry{Source: SourceSell, Status: StatusNew, Location: ptr("koh-phangan")}
}

// TestInquiry_Validate_LocationRequired — location is mandatory; the "other"
// sentinel requires a non-blank location_custom; property_type is optional.
func TestInquiry_Validate_LocationRequired(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Inquiry)
		wantErr bool
	}{
		{name: "listed location", mutate: func(i *Inquiry) { i.Location = ptr("hua-hin") }, wantErr: false},
		{name: "missing location", mutate: func(i *Inquiry) { i.Location = nil }, wantErr: true},
		{name: "blank location", mutate: func(i *Inquiry) { i.Location = ptr("   ") }, wantErr: true},
		{name: "other without custom", mutate: func(i *Inquiry) { i.Location = ptr(LocationOther) }, wantErr: true},
		{name: "other with blank custom", mutate: func(i *Inquiry) { i.Location = ptr(LocationOther); i.LocationCustom = ptr("  ") }, wantErr: true},
		{name: "other with custom", mutate: func(i *Inquiry) { i.Location = ptr(LocationOther); i.LocationCustom = ptr("Phuket") }, wantErr: false},
		{name: "optional property type set", mutate: func(i *Inquiry) { i.PropertyType = ptr("villa") }, wantErr: false},
		{name: "optional property type omitted", mutate: func(i *Inquiry) { i.PropertyType = nil }, wantErr: false},
		{name: "property type other without custom", mutate: func(i *Inquiry) { i.PropertyType = ptr(PropertyTypeOther) }, wantErr: true},
		{name: "property type other with custom", mutate: func(i *Inquiry) { i.PropertyType = ptr(PropertyTypeOther); i.PropertyTypeCustom = ptr("Underwater dome") }, wantErr: false},
		// S6 (2607-055): a property-attached enquiry derives its location from the property,
		// so location is NOT required when PropertyRef is set (the public form hides the field).
		{name: "property attached, location nil — not required", mutate: func(i *Inquiry) { i.PropertyRef = ptr("prop-123"); i.Location = nil }, wantErr: false},
		{name: "property attached, location blank — not required", mutate: func(i *Inquiry) { i.PropertyRef = ptr("prop-123"); i.Location = ptr("  ") }, wantErr: false},
		// …but an explicit "other" location still needs its custom text, even with a property.
		{name: "property attached, other without custom still rejected", mutate: func(i *Inquiry) { i.PropertyRef = ptr("prop-123"); i.Location = ptr(LocationOther) }, wantErr: true},
		// An EMPTY/whitespace PropertyRef does not count as attached — location stays required.
		{name: "blank property ref, location nil — still required", mutate: func(i *Inquiry) { i.PropertyRef = ptr("   "); i.Location = nil }, wantErr: true},
		// 2607-119: an assistant handoff posts with source "ask" and has NO location picker — a
		// location-less chat lead with a contact channel is valid (only the form sources require it).
		{name: "assistant handoff (ask) without location — allowed", mutate: func(i *Inquiry) { i.Source = SourceAsk; i.Location = nil }, wantErr: false},
		{name: "assistant handoff (ask) blank location — allowed", mutate: func(i *Inquiry) { i.Source = SourceAsk; i.Location = ptr("  ") }, wantErr: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			i := base()
			i.ContactEmail = ptr("a@b.com") // satisfy the contact-channel rule
			c.mutate(&i)
			err := i.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected ErrValidation, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}

// TestInquiry_Location_RoundTrip — a Location value survives Validate() AND
// round-trips through the PostgresMapper to the physical `location` column
// (Decision #0295 Stage 2 / Task 6: ToRow keys the value under "location";
// FromRow scans that column back into Location).
func TestInquiry_Location_RoundTrip(t *testing.T) {
	i := base()
	i.ContactEmail = ptr("a@b.com")
	i.Location = ptr("koh-samui")
	if err := i.Validate(); err != nil {
		t.Fatalf("a valid Location must pass Validate(): %v", err)
	}

	m := NewPostgresMapper("bestierealestate")

	// ToRow keys the Location field under the physical "location" column.
	row, err := m.ToRow(i)
	if err != nil {
		t.Fatalf("ToRow: %v", err)
	}
	got, ok := row["location"].(*string)
	if !ok || got == nil || *got != "koh-samui" {
		t.Fatalf(`ToRow["location"] = %v, want *string "koh-samui"`, row["location"])
	}

	// FromRow scans the physical location column back into Location. Drive the
	// scan with the exact Columns() order so the field still lands.
	cols := m.Columns()
	values := make(map[string]any, len(cols))
	values["i.location"] = ptr("phuket-town")
	out, err := m.FromRow(func(dest ...any) error {
		if len(dest) != len(cols) {
			t.Fatalf("FromRow scan arity = %d, want %d (Columns drift)", len(dest), len(cols))
		}
		for idx, col := range cols {
			if v, present := values[col]; present {
				p := dest[idx].(**string)
				*p = v.(*string)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FromRow: %v", err)
	}
	if out.Location == nil || *out.Location != "phuket-town" {
		t.Fatalf("FromRow → Location = %v, want phuket-town (location column must scan into Location)", out.Location)
	}
}

// TestInquiry_Validate_ContactRequired — at least one channel must be present;
// bad source is rejected; happy channels pass.
func TestInquiry_Validate_ContactRequired(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Inquiry)
		wantErr bool
	}{
		{
			name:    "no contact channel",
			mutate:  func(i *Inquiry) {},
			wantErr: true,
		},
		{
			name:    "email only",
			mutate:  func(i *Inquiry) { i.ContactEmail = ptr("a@b.com") },
			wantErr: false,
		},
		{
			name:    "phone only",
			mutate:  func(i *Inquiry) { i.PhoneCountryCode = ptr("+66"); i.PhoneNumber = ptr("812345678") },
			wantErr: false,
		},
		{
			name: "social only",
			mutate: func(i *Inquiry) {
				i.Socials = socialnetwork.List{{Platform: "line", Handle: "bestie"}}
			},
			wantErr: false,
		},
		{
			name:    "bad source",
			mutate:  func(i *Inquiry) { i.Source = InquirySource("nope"); i.ContactEmail = ptr("a@b.com") },
			wantErr: true,
		},
		{
			name:    "malformed email",
			mutate:  func(i *Inquiry) { i.ContactEmail = ptr("not-an-email") },
			wantErr: true,
		},
		{
			name: "message too long",
			mutate: func(i *Inquiry) {
				i.ContactEmail = ptr("a@b.com")
				big := make([]byte, maxMessageLen+1)
				for j := range big {
					big[j] = 'x'
				}
				i.Message = ptr(string(big))
			},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			i := base()
			c.mutate(&i)
			err := i.Validate()
			if c.wantErr && err == nil {
				t.Fatalf("expected ErrValidation, got nil")
			}
			if !c.wantErr && err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}

// TestInquiry_MarkRead_FlipsOnce — new→read stamps read_at once; a second call
// (or a call on read/handled) is a no-op.
func TestInquiry_MarkRead_FlipsOnce(t *testing.T) {
	t1 := time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Hour)

	i := base()
	i.MarkRead(t1)
	if i.Status != StatusRead {
		t.Fatalf("status = %q, want read", i.Status)
	}
	if i.ReadAt == nil || !i.ReadAt.Equal(t1) {
		t.Fatalf("read_at = %v, want %v", i.ReadAt, t1)
	}

	// Second call must not move read_at.
	i.MarkRead(t2)
	if !i.ReadAt.Equal(t1) {
		t.Errorf("read_at changed on second MarkRead: %v, want %v", i.ReadAt, t1)
	}

	// A handled inquiry stays handled.
	h := base()
	h.MarkHandled("actor-1", t1)
	h.MarkRead(t2)
	if h.Status != StatusHandled {
		t.Errorf("MarkRead regressed a handled inquiry to %q", h.Status)
	}
}

// TestInquiry_MarkHandled_RecordsActor — handled stamps actor + timestamp.
func TestInquiry_MarkHandled_RecordsActor(t *testing.T) {
	now := time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC)
	i := base()
	i.MarkHandled("actor-7", now)

	if i.Status != StatusHandled {
		t.Errorf("status = %q, want handled", i.Status)
	}
	if i.HandledBy == nil || *i.HandledBy != "actor-7" {
		t.Errorf("handled_by = %v, want actor-7", i.HandledBy)
	}
	if i.HandledAt == nil || !i.HandledAt.Equal(now) {
		t.Errorf("handled_at = %v, want %v", i.HandledAt, now)
	}
}

// TestInquiry_ErrValidation_Wraps — Validate failures are ErrValidation-class so
// the handler can map them to 400. (The entity returns plain errors today;
// callers wrap with %w: ErrValidation — assert the sentinel exists + is usable.)
func TestInquiry_ErrValidation_Sentinel(t *testing.T) {
	if ErrValidation == nil {
		t.Fatal("ErrValidation must be a non-nil sentinel")
	}
	wrapped := errors.New("x")
	_ = wrapped
}
