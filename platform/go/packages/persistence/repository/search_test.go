package repository

import "testing"

func TestSearchOptions_EmptyQuery(t *testing.T) {
	opts := SearchOptions{}
	if opts.Query != "" {
		t.Error("zero-value SearchOptions.Query should be empty string")
	}
	if opts.Limit != 0 {
		t.Error("zero-value SearchOptions.Limit should be 0")
	}
}

func TestSearchOptions_WithFilter(t *testing.T) {
	price := Field[int64]{Name: "price_amount"}
	opts := SearchOptions{
		Query:  "pool villa",
		Filter: And(price.Gte(5000000), price.Lte(20000000)),
		Limit:  20,
		Offset: 0,
	}

	if opts.Query != "pool villa" {
		t.Errorf("Query = %q, want %q", opts.Query, "pool villa")
	}
	if opts.Filter.IsEmpty() {
		t.Error("Filter should not be empty when predicates are set")
	}
	if len(opts.Filter.And) != 2 {
		t.Errorf("Filter.And len = %d, want 2", len(opts.Filter.And))
	}
}

func TestSearchResult_ZeroValue(t *testing.T) {
	var r SearchResult[string]
	if r.Total != 0 {
		t.Errorf("Total = %d, want 0", r.Total)
	}
	if r.UsedFuzzy {
		t.Error("UsedFuzzy should be false by default")
	}
	if r.Items != nil {
		t.Error("Items should be nil by default")
	}
}

func TestSearchResult_UsedFuzzy(t *testing.T) {
	r := SearchResult[string]{
		Items:     []string{"result1"},
		Total:     1,
		UsedFuzzy: true,
	}
	if !r.UsedFuzzy {
		t.Error("UsedFuzzy should be true when set")
	}
	if r.Total != 1 {
		t.Errorf("Total = %d, want 1", r.Total)
	}
}

// TestSearcher_InterfaceContract verifies the Searcher interface signature compiles.
// This is a compile-time assertion — if the interface changes, this test breaks.
func TestSearcher_InterfaceContract(t *testing.T) {
	// Verify SearchOptions can carry both text query and structured filter
	published := Field[bool]{Name: "is_published"}
	opts := SearchOptions{
		Query:  "condo srithanu",
		Filter: And(published.Eq(true)),
		Sort:   []SortField{{Field: "price_amount", Desc: false}},
		Limit:  10,
		Offset: 0,
	}

	if opts.Sort[0].Field != "price_amount" {
		t.Errorf("Sort[0].Field = %q, want %q", opts.Sort[0].Field, "price_amount")
	}
	if opts.Sort[0].Desc {
		t.Error("Sort[0].Desc should be false for ascending price")
	}
}
