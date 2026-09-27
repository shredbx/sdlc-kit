package socialnetwork

import (
	"strings"
	"testing"
)

func TestNewSocialNetwork(t *testing.T) {
	t.Run("creates with platform and handle", func(t *testing.T) {
		s := NewSocialNetwork("instagram", "bestays_kohphangan")
		if s.Platform != "instagram" || s.Handle != "bestays_kohphangan" {
			t.Errorf("expected {instagram, bestays_kohphangan}, got {%s, %s}", s.Platform, s.Handle)
		}
		if s.URL != "" {
			t.Errorf("expected empty URL, got %q", s.URL)
		}
	})

	t.Run("creates LINE contact", func(t *testing.T) {
		s := NewSocialNetwork("line", "@bestays")
		if s.Platform != "line" || s.Handle != "@bestays" {
			t.Errorf("expected {line, @bestays}, got {%s, %s}", s.Platform, s.Handle)
		}
	})
}

func TestSocialNetwork_Validate(t *testing.T) {
	tests := []struct {
		name     string
		platform string
		handle   string
		wantErr  bool
		errMsg   string
	}{
		{"valid profile", "instagram", "bestays_kohphangan", false, ""},
		{"valid whatsapp", "whatsapp", "+66812345678", false, ""},
		{"empty platform", "", "bestays", true, "platform must not be empty"},
		{"whitespace platform", "   ", "bestays", true, "platform must not be empty"},
		{"empty handle", "instagram", "", true, "handle must not be empty"},
		{"whitespace handle", "instagram", "   ", true, "handle must not be empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSocialNetwork(tt.platform, tt.handle)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestSocialNetwork_IsZero(t *testing.T) {
	t.Run("zero value", func(t *testing.T) {
		s := SocialNetwork{}
		if !s.IsZero() {
			t.Error("expected IsZero() = true for empty SocialNetwork")
		}
	})

	t.Run("non-zero with platform and handle", func(t *testing.T) {
		s := NewSocialNetwork("instagram", "bestays")
		if s.IsZero() {
			t.Error("expected IsZero() = false for populated SocialNetwork")
		}
	})

	t.Run("non-zero with URL only", func(t *testing.T) {
		s := SocialNetwork{URL: "https://instagram.com/bestays"}
		if s.IsZero() {
			t.Error("expected IsZero() = false when URL is set")
		}
	})
}

func TestSocialNetwork_HasURL(t *testing.T) {
	t.Run("with URL", func(t *testing.T) {
		s := SocialNetwork{
			Platform: "instagram",
			Handle:   "bestays_kohphangan",
			URL:      "https://instagram.com/bestays_kohphangan",
		}
		if !s.HasURL() {
			t.Error("expected HasURL() = true when URL is set")
		}
	})

	t.Run("without URL", func(t *testing.T) {
		s := NewSocialNetwork("line", "@bestays")
		if s.HasURL() {
			t.Error("expected HasURL() = false when URL is empty")
		}
	})
}

func TestList_Value_Empty(t *testing.T) {
	v, err := List{}.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if v != nil {
		t.Errorf("empty list should Value()=nil (SQL NULL), got %v", v)
	}
}

func TestList_RoundTrip(t *testing.T) {
	original := List{
		{Platform: "line", Handle: "somchai"},
		{Platform: "instagram", Handle: "somchai_pv", URL: "https://instagram.com/somchai_pv"},
	}
	v, err := original.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	if _, ok := v.([]byte); !ok {
		t.Fatalf("non-empty list should Value() to []byte, got %T", v)
	}

	var rescanned List
	if err := rescanned.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(rescanned) != 2 {
		t.Fatalf("want 2 entries, got %d", len(rescanned))
	}
	if rescanned[0].Platform != "line" || rescanned[1].Handle != "somchai_pv" {
		t.Errorf("round-trip mismatch: %+v", rescanned)
	}
}

func TestList_Scan_NullAndEmpty(t *testing.T) {
	cases := []struct {
		name string
		src  any
	}{
		{"nil", nil},
		{"empty bytes", []byte{}},
		{"literal null", []byte("null")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := List{{Platform: "line", Handle: "x"}} // start non-nil to prove reset
			if err := l.Scan(tc.src); err != nil {
				t.Fatalf("Scan(%v): %v", tc.src, err)
			}
			if l != nil {
				t.Errorf("Scan(%v) should produce nil list, got %+v", tc.src, l)
			}
		})
	}
}

func TestList_Validate(t *testing.T) {
	t.Run("empty list is valid", func(t *testing.T) {
		if err := (List{}).Validate(); err != nil {
			t.Errorf("empty list should validate, got %v", err)
		}
	})

	t.Run("all-valid entries pass", func(t *testing.T) {
		l := List{
			{Platform: "line", Handle: "somchai"},
			{Platform: "instagram", Handle: "somchai_pv"},
		}
		if err := l.Validate(); err != nil {
			t.Errorf("expected valid, got %v", err)
		}
	})

	t.Run("bad entry fails with its index", func(t *testing.T) {
		l := List{
			{Platform: "line", Handle: "somchai"},
			{Platform: "instagram", Handle: ""}, // missing handle
		}
		err := l.Validate()
		if err == nil {
			t.Fatal("expected validation failure for bad entry")
		}
		if !strings.Contains(err.Error(), "index 1") {
			t.Errorf("error should identify the failing index, got %q", err.Error())
		}
	})
}
