// faq_test.go exercises the Problem-Domain layer of pkg/faq (FDD4.PD.TEST_RED,
// SDLC 2607-001). PD is PURE: zero storage imports — only validation rules, the
// Publishable Status lifecycle guard, and the derived display label. These are the
// acceptance tests for test-plan PD cases 1-4 (NFR-015: PD >90%).
//
// Mirrors pkg/cms/entry_test.go + cmspage_test.go conventions: table-driven, every
// validation failure wraps faq.ErrValidation (callers test errors.Is).
package faq

import (
	"errors"
	"testing"
)

// =============================================================================
// SLUG — the category URL key (/faq/<slug>); mirrors cms.Slug's kebab invariant.
// =============================================================================

func TestSlug_Validate(t *testing.T) {
	tests := []struct {
		name    string
		slug    Slug
		wantErr bool
	}{
		{name: "kebab", slug: Slug("buyers-guide"), wantErr: false},
		{name: "single word", slug: Slug("about"), wantErr: false},
		{name: "digits", slug: Slug("step-1"), wantErr: false},
		{name: "empty", slug: Slug(""), wantErr: true},
		{name: "whitespace", slug: Slug("   "), wantErr: true},
		{name: "uppercase", slug: Slug("Buyers-Guide"), wantErr: true},
		{name: "underscore", slug: Slug("buyers_guide"), wantErr: true},
		{name: "leading dash", slug: Slug("-buyers"), wantErr: true},
		{name: "trailing dash", slug: Slug("buyers-"), wantErr: true},
		{name: "double dash", slug: Slug("buyers--guide"), wantErr: true},
		{name: "space inside", slug: Slug("buyers guide"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.slug.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Slug(%q).Validate() err = %v, wantErr %v", tt.slug, err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// STATUS — the Publishable lifecycle (draft|published). PD case 3.
// =============================================================================

// TestStatus_Validate enforces the closed set: exactly draft|published. An
// unknown/empty value is the corrupt-row guard; the transition RULES live in
// TransitionTo below (the Service.Publish facade delegates to it).
func TestStatus_Validate(t *testing.T) {
	tests := []struct {
		name    string
		status  Status
		wantErr bool
	}{
		{name: "draft", status: StatusDraft, wantErr: false},
		{name: "published", status: StatusPublished, wantErr: false},
		{name: "empty", status: Status(""), wantErr: true},
		{name: "unknown", status: Status("archived"), wantErr: true},
		{name: "wrong case", status: Status("Draft"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.status.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Status(%q).Validate() err = %v, wantErr %v", tt.status, err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Fatalf("Status(%q).Validate() err = %v, want it wrapped in ErrValidation", tt.status, err)
			}
		})
	}
}

// TestStatus_TransitionTo is the per-row state-machine guard (test-plan case 3):
// draft->published is the headline lifecycle move; any transition involving an
// invalid (non draft/published) status is rejected.
func TestStatus_TransitionTo(t *testing.T) {
	t.Run("draft to published ok", func(t *testing.T) {
		got, err := StatusDraft.TransitionTo(StatusPublished)
		if err != nil {
			t.Fatalf("TransitionTo err = %v, want nil", err)
		}
		if got != StatusPublished {
			t.Fatalf("TransitionTo got = %q, want published", got)
		}
	})
	t.Run("published to draft ok (unpublish)", func(t *testing.T) {
		got, err := StatusPublished.TransitionTo(StatusDraft)
		if err != nil {
			t.Fatalf("TransitionTo err = %v, want nil", err)
		}
		if got != StatusDraft {
			t.Fatalf("got = %q, want draft", got)
		}
	})
	t.Run("invalid source rejected", func(t *testing.T) {
		_, err := Status("archived").TransitionTo(StatusPublished)
		if err == nil {
			t.Fatal("TransitionTo err = nil, want validation error")
		}
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
	})
	t.Run("invalid target rejected", func(t *testing.T) {
		_, err := StatusDraft.TransitionTo(Status("archived"))
		if err == nil {
			t.Fatal("TransitionTo err = nil, want validation error")
		}
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("err = %v, want ErrValidation", err)
		}
	})
}

// =============================================================================
// CATEGORY — test-plan case 1: slug required + kebab, title required, status valid.
// =============================================================================

func TestCategory_Validate(t *testing.T) {
	valid := Category{Slug: Slug("buyers-guide"), Title: "Buyers Guide", Status: StatusDraft}

	tests := []struct {
		name    string
		mutate  func(c *Category)
		wantErr bool
	}{
		{name: "valid draft", mutate: func(c *Category) {}, wantErr: false},
		{name: "valid published", mutate: func(c *Category) { c.Status = StatusPublished }, wantErr: false},
		{name: "missing title", mutate: func(c *Category) { c.Title = "" }, wantErr: true},
		{name: "blank title", mutate: func(c *Category) { c.Title = "   " }, wantErr: true},
		{name: "empty slug", mutate: func(c *Category) { c.Slug = Slug("") }, wantErr: true},
		{name: "bad slug uppercase", mutate: func(c *Category) { c.Slug = Slug("Buyers-Guide") }, wantErr: true},
		{name: "bad slug space", mutate: func(c *Category) { c.Slug = Slug("buyers guide") }, wantErr: true},
		{name: "invalid status", mutate: func(c *Category) { c.Status = Status("archived") }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid
			tt.mutate(&c)
			err := c.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Fatalf("Validate() err = %v, want ErrValidation wrap", err)
			}
		})
	}
}

// =============================================================================
// ITEM — test-plan case 2: empty question rejected (answer also required; tip opt).
// =============================================================================

func TestItem_Validate(t *testing.T) {
	valid := Item{
		Question: "Can foreigners buy property in Thailand?",
		Answer:   "Yes, foreigners may own a condominium unit freehold.",
		Status:   StatusDraft,
	}

	tests := []struct {
		name    string
		mutate  func(i *Item)
		wantErr bool
	}{
		{name: "valid with tip", mutate: func(i *Item) { i.Tip = "Always use an independent lawyer." }, wantErr: false},
		{name: "valid no tip", mutate: func(i *Item) {}, wantErr: false},
		{name: "empty question", mutate: func(i *Item) { i.Question = "" }, wantErr: true},
		{name: "blank question", mutate: func(i *Item) { i.Question = "  " }, wantErr: true},
		{name: "empty answer", mutate: func(i *Item) { i.Answer = "" }, wantErr: true},
		{name: "blank answer", mutate: func(i *Item) { i.Answer = "   " }, wantErr: true},
		{name: "invalid status", mutate: func(i *Item) { i.Status = Status("archived") }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := valid
			tt.mutate(&i)
			err := i.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, ErrValidation) {
				t.Fatalf("Validate() err = %v, want ErrValidation wrap", err)
			}
		})
	}
}

// =============================================================================
// DISPLAY LABEL — test-plan case 4: derived "{step}.{item}", never stored twice.
// Step position is 0-based (Step 0 = plan-before-buy); the label is a direct
// projection of the two positions.
// =============================================================================

func TestDisplayLabel(t *testing.T) {
	tests := []struct {
		name       string
		step, item int
		want       string
	}{
		{name: "step 1 item 2", step: 1, item: 2, want: "1.2"},
		{name: "step 0 item 1", step: 0, item: 1, want: "0.1"},
		{name: "step 0 item 0", step: 0, item: 0, want: "0.0"},
		{name: "step 6 item 3", step: 6, item: 3, want: "6.3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DisplayLabel(tt.step, tt.item); got != tt.want {
				t.Fatalf("DisplayLabel(%d,%d) = %q, want %q", tt.step, tt.item, got, tt.want)
			}
		})
	}
}
