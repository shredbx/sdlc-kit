package repository

import "testing"

func TestAscSort(t *testing.T) {
	t.Run("creates_ascending_sort", func(t *testing.T) {
		sf := Asc("created_at")
		if sf.Field != "created_at" {
			t.Errorf("Field = %q, want %q", sf.Field, "created_at")
		}
		if sf.Desc {
			t.Error("Asc should set Desc = false")
		}
	})

	t.Run("empty_field_name", func(t *testing.T) {
		sf := Asc("")
		if sf.Field != "" {
			t.Errorf("Field = %q, want empty", sf.Field)
		}
		if sf.Desc {
			t.Error("Asc should set Desc = false even for empty field")
		}
	})
}

func TestDescSort(t *testing.T) {
	t.Run("creates_descending_sort", func(t *testing.T) {
		sf := DescSort("price")
		if sf.Field != "price" {
			t.Errorf("Field = %q, want %q", sf.Field, "price")
		}
		if !sf.Desc {
			t.Error("DescSort should set Desc = true")
		}
	})

	t.Run("empty_field_name", func(t *testing.T) {
		sf := DescSort("")
		if sf.Field != "" {
			t.Errorf("Field = %q, want empty", sf.Field)
		}
		if !sf.Desc {
			t.Error("DescSort should set Desc = true even for empty field")
		}
	})
}

func TestListOptions_defaults(t *testing.T) {
	t.Run("zero_value_has_empty_filter", func(t *testing.T) {
		var opts ListOptions
		if !opts.Filter.IsEmpty() {
			t.Error("default Filter should be empty")
		}
		if opts.Sort != nil {
			t.Error("default Sort should be nil")
		}
		if opts.Limit != 0 {
			t.Errorf("default Limit = %d, want 0", opts.Limit)
		}
		if opts.Offset != 0 {
			t.Errorf("default Offset = %d, want 0", opts.Offset)
		}
	})

	t.Run("compose_filter_sort_pagination", func(t *testing.T) {
		city := Field[string]{Name: "city"}
		opts := ListOptions{
			Filter: And(city.Eq("Paris")),
			Sort:   []SortField{DescSort("price"), Asc("name")},
			Limit:  20,
			Offset: 40,
		}

		if opts.Filter.IsEmpty() {
			t.Error("Filter should not be empty")
		}
		if len(opts.Sort) != 2 {
			t.Fatalf("Sort len = %d, want 2", len(opts.Sort))
		}
		if opts.Sort[0].Field != "price" || !opts.Sort[0].Desc {
			t.Errorf("Sort[0] = %+v, want price DESC", opts.Sort[0])
		}
		if opts.Sort[1].Field != "name" || opts.Sort[1].Desc {
			t.Errorf("Sort[1] = %+v, want name ASC", opts.Sort[1])
		}
		if opts.Limit != 20 {
			t.Errorf("Limit = %d, want 20", opts.Limit)
		}
		if opts.Offset != 40 {
			t.Errorf("Offset = %d, want 40", opts.Offset)
		}
	})
}
