package postgres

import (
	"reflect"
	"strings"
	"testing"

	"github.com/shredbx/sbx-core/pkg/repository"
)

// identity mapper — returns field name unchanged
func identity(name string) string { return name }

// ---------- NewQueryBuilder ----------

func TestNewQueryBuilder(t *testing.T) {
	qb := NewQueryBuilder()
	if qb == nil {
		t.Fatal("NewQueryBuilder returned nil")
	}
	if qb.counter != 0 {
		t.Errorf("counter = %d, want 0", qb.counter)
	}
	if len(qb.args) != 0 {
		t.Errorf("args len = %d, want 0", len(qb.args))
	}
}

// ---------- Build — empty query ----------

func TestBuild_empty_query(t *testing.T) {
	qb := NewQueryBuilder()
	clause, args := qb.Build(repository.Query{}, identity)

	if clause != "" {
		t.Errorf("clause = %q, want empty", clause)
	}
	if args != nil {
		t.Errorf("args = %v, want nil", args)
	}
}

// ---------- Build — single predicates ----------

func TestBuild_eq(t *testing.T) {
	qb := NewQueryBuilder()
	city := repository.Field[string]{Name: "city"}
	q := repository.And(city.Eq("Paris"))

	clause, args := qb.Build(q, identity)

	if clause != "city = $1" {
		t.Errorf("clause = %q, want %q", clause, "city = $1")
	}
	if len(args) != 1 || args[0] != "Paris" {
		t.Errorf("args = %v, want [Paris]", args)
	}
}

func TestBuild_neq(t *testing.T) {
	qb := NewQueryBuilder()
	status := repository.Field[string]{Name: "status"}
	q := repository.And(status.Neq("deleted"))

	clause, args := qb.Build(q, identity)

	if clause != "status != $1" {
		t.Errorf("clause = %q, want %q", clause, "status != $1")
	}
	if len(args) != 1 || args[0] != "deleted" {
		t.Errorf("args = %v, want [deleted]", args)
	}
}

func TestBuild_gt(t *testing.T) {
	qb := NewQueryBuilder()
	price := repository.Field[int]{Name: "price"}
	q := repository.And(price.Gt(100))

	clause, args := qb.Build(q, identity)

	if clause != "price > $1" {
		t.Errorf("clause = %q, want %q", clause, "price > $1")
	}
	if len(args) != 1 || args[0] != 100 {
		t.Errorf("args = %v, want [100]", args)
	}
}

func TestBuild_gte(t *testing.T) {
	qb := NewQueryBuilder()
	beds := repository.Field[int]{Name: "bedrooms"}
	q := repository.And(beds.Gte(3))

	clause, _ := qb.Build(q, identity)
	if clause != "bedrooms >= $1" {
		t.Errorf("clause = %q, want %q", clause, "bedrooms >= $1")
	}
}

func TestBuild_lt(t *testing.T) {
	qb := NewQueryBuilder()
	rating := repository.Field[float64]{Name: "rating"}
	q := repository.And(rating.Lt(4.5))

	clause, _ := qb.Build(q, identity)
	if clause != "rating < $1" {
		t.Errorf("clause = %q, want %q", clause, "rating < $1")
	}
}

func TestBuild_lte(t *testing.T) {
	qb := NewQueryBuilder()
	guests := repository.Field[int]{Name: "max_guests"}
	q := repository.And(guests.Lte(10))

	clause, _ := qb.Build(q, identity)
	if clause != "max_guests <= $1" {
		t.Errorf("clause = %q, want %q", clause, "max_guests <= $1")
	}
}

func TestBuild_like(t *testing.T) {
	qb := NewQueryBuilder()
	name := repository.Field[string]{Name: "name"}
	q := repository.And(name.Like("%beach%"))

	clause, args := qb.Build(q, identity)
	if clause != "name LIKE $1" {
		t.Errorf("clause = %q, want %q", clause, "name LIKE $1")
	}
	if len(args) != 1 || args[0] != "%beach%" {
		t.Errorf("args = %v, want [%%beach%%]", args)
	}
}

func TestBuild_is_null(t *testing.T) {
	qb := NewQueryBuilder()
	deleted := repository.Field[string]{Name: "deleted_at"}
	q := repository.And(deleted.IsNull())

	clause, args := qb.Build(q, identity)
	if clause != "deleted_at IS NULL" {
		t.Errorf("clause = %q, want %q", clause, "deleted_at IS NULL")
	}
	if len(args) != 0 {
		t.Errorf("args len = %d, want 0 (IS NULL has no params)", len(args))
	}
}

func TestBuild_is_not_null(t *testing.T) {
	qb := NewQueryBuilder()
	email := repository.Field[string]{Name: "email"}
	q := repository.And(email.IsNotNull())

	clause, args := qb.Build(q, identity)
	if clause != "email IS NOT NULL" {
		t.Errorf("clause = %q, want %q", clause, "email IS NOT NULL")
	}
	if len(args) != 0 {
		t.Errorf("args len = %d, want 0", len(args))
	}
}

// ---------- Build — IN operator ----------

func TestBuild_in_strings(t *testing.T) {
	qb := NewQueryBuilder()
	status := repository.Field[string]{Name: "status"}
	q := repository.And(status.In([]string{"active", "pending"}))

	clause, args := qb.Build(q, identity)

	if clause != "status IN ($1, $2)" {
		t.Errorf("clause = %q, want %q", clause, "status IN ($1, $2)")
	}
	if len(args) != 2 || args[0] != "active" || args[1] != "pending" {
		t.Errorf("args = %v, want [active pending]", args)
	}
}

func TestBuild_in_int64(t *testing.T) {
	qb := NewQueryBuilder()
	id := repository.Field[int64]{Name: "id"}
	q := repository.And(id.In([]int64{1, 2, 3}))

	clause, args := qb.Build(q, identity)

	if clause != "id IN ($1, $2, $3)" {
		t.Errorf("clause = %q, want %q", clause, "id IN ($1, $2, $3)")
	}
	if len(args) != 3 {
		t.Errorf("args len = %d, want 3", len(args))
	}
}

func TestBuild_in_empty_strings(t *testing.T) {
	qb := NewQueryBuilder()
	status := repository.Field[string]{Name: "status"}
	q := repository.And(status.In([]string{}))

	clause, args := qb.Build(q, identity)

	if clause != "FALSE" {
		t.Errorf("clause = %q, want %q (empty IN = FALSE)", clause, "FALSE")
	}
	if len(args) != 0 {
		t.Errorf("args len = %d, want 0", len(args))
	}
}

func TestBuild_in_empty_int64(t *testing.T) {
	qb := NewQueryBuilder()
	id := repository.Field[int64]{Name: "id"}
	q := repository.And(id.In([]int64{}))

	clause, _ := qb.Build(q, identity)
	if clause != "FALSE" {
		t.Errorf("clause = %q, want %q", clause, "FALSE")
	}
}

func TestBuild_in_empty_int(t *testing.T) {
	qb := NewQueryBuilder()
	count := repository.Field[int]{Name: "count"}
	q := repository.And(count.In([]int{}))

	clause, _ := qb.Build(q, identity)
	if clause != "FALSE" {
		t.Errorf("clause = %q, want %q", clause, "FALSE")
	}
}

// ---------- Build — array containment (@>) ----------

func TestBuild_array_contains(t *testing.T) {
	qb := NewQueryBuilder()
	tags := repository.ArrayField{Name: "tags"}
	q := repository.And(tags.Contains([]string{"a", "b"}))

	clause, args := qb.Build(q, identity)

	// Exactly one placeholder, the ::TEXT[] cast present — the array is bound
	// as ONE parameter, never expanded into per-element placeholders.
	if clause != "tags @> $1::TEXT[]" {
		t.Errorf("clause = %q, want %q", clause, "tags @> $1::TEXT[]")
	}

	// No inlined literals — the elements must NOT appear in the SQL string.
	if strings.Contains(clause, "'a'") || strings.Contains(clause, "'b'") ||
		strings.Contains(clause, "a, b") {
		t.Errorf("clause %q inlines array literals — must be parameterized", clause)
	}

	// args carries exactly one element: the bound []string{"a","b"}.
	if len(args) != 1 {
		t.Fatalf("args len = %d, want 1 (whole array is one bound param)", len(args))
	}
	if !reflect.DeepEqual(args[0], []string{"a", "b"}) {
		t.Errorf("args[0] = %#v, want []string{\"a\", \"b\"}", args[0])
	}
}

// Regression: OpArrayContains MUST take the dedicated case, never fall through
// to the opToSQL default (which returns "=" — a silent wrong-results trap).
func TestBuild_array_contains_not_eq_fallthrough(t *testing.T) {
	qb := NewQueryBuilder()
	tags := repository.ArrayField{Name: "tags"}
	q := repository.And(tags.Contains([]string{"x"}))

	clause, _ := qb.Build(q, identity)

	if strings.Contains(clause, "=") {
		t.Errorf("clause = %q — OpArrayContains compiled to '=' (default fallthrough); want '@>'", clause)
	}
	if !strings.Contains(clause, "@>") {
		t.Errorf("clause = %q, want it to contain the '@>' operator", clause)
	}
}

func TestBuild_array_contains_with_field_mapper(t *testing.T) {
	qb := NewQueryBuilder()
	tags := repository.ArrayField{Name: "tags"}

	mapper := func(name string) string {
		if name == "tags" {
			return "p.tags"
		}
		return name
	}

	q := repository.And(tags.Contains([]string{"pool"}))
	clause, args := qb.Build(q, mapper)

	if clause != "p.tags @> $1::TEXT[]" {
		t.Errorf("clause = %q, want %q", clause, "p.tags @> $1::TEXT[]")
	}
	if len(args) != 1 || !reflect.DeepEqual(args[0], []string{"pool"}) {
		t.Errorf("args = %#v, want [[]string{\"pool\"}]", args)
	}
}

// Mixed composition: array-contains alongside a scalar predicate must keep
// sequential placeholder numbering and bind the array as a single param.
func TestBuild_array_contains_mixed_with_scalar(t *testing.T) {
	qb := NewQueryBuilder()
	city := repository.Field[string]{Name: "city"}
	tags := repository.ArrayField{Name: "tags"}
	q := repository.And(city.Eq("Phuket"), tags.Contains([]string{"sea-view", "pool"}))

	clause, args := qb.Build(q, identity)

	if clause != "(city = $1 AND tags @> $2::TEXT[])" {
		t.Errorf("clause = %q, want %q", clause, "(city = $1 AND tags @> $2::TEXT[])")
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
	if args[0] != "Phuket" {
		t.Errorf("args[0] = %v, want Phuket", args[0])
	}
	if !reflect.DeepEqual(args[1], []string{"sea-view", "pool"}) {
		t.Errorf("args[1] = %#v, want []string{\"sea-view\", \"pool\"}", args[1])
	}
}

// ---------- Build — AND composition ----------

func TestBuild_and_multiple(t *testing.T) {
	qb := NewQueryBuilder()
	city := repository.Field[string]{Name: "city"}
	published := repository.Field[bool]{Name: "is_published"}
	q := repository.And(city.Eq("Paris"), published.Eq(true))

	clause, args := qb.Build(q, identity)

	if clause != "(city = $1 AND is_published = $2)" {
		t.Errorf("clause = %q, want %q", clause, "(city = $1 AND is_published = $2)")
	}
	if len(args) != 2 {
		t.Fatalf("args len = %d, want 2", len(args))
	}
	if args[0] != "Paris" {
		t.Errorf("args[0] = %v, want Paris", args[0])
	}
	if args[1] != true {
		t.Errorf("args[1] = %v, want true", args[1])
	}
}

func TestBuild_and_three_predicates(t *testing.T) {
	qb := NewQueryBuilder()
	city := repository.Field[string]{Name: "city"}
	beds := repository.Field[int]{Name: "bedrooms"}
	price := repository.Field[int]{Name: "price"}
	q := repository.And(city.Eq("Rome"), beds.Gte(2), price.Lt(500000))

	clause, args := qb.Build(q, identity)

	if clause != "(city = $1 AND bedrooms >= $2 AND price < $3)" {
		t.Errorf("clause = %q", clause)
	}
	if len(args) != 3 {
		t.Errorf("args len = %d, want 3", len(args))
	}
}

// ---------- Build — OR composition ----------

func TestBuild_or_multiple(t *testing.T) {
	qb := NewQueryBuilder()
	status := repository.Field[string]{Name: "status"}
	q := repository.Or(status.Eq("active"), status.Eq("pending"))

	clause, args := qb.Build(q, identity)

	if clause != "(status = $1 OR status = $2)" {
		t.Errorf("clause = %q, want %q", clause, "(status = $1 OR status = $2)")
	}
	if len(args) != 2 {
		t.Errorf("args len = %d, want 2", len(args))
	}
}

// ---------- Build — field mapper ----------

func TestBuild_field_mapper(t *testing.T) {
	qb := NewQueryBuilder()
	price := repository.Field[int]{Name: "price_amount"}

	mapper := func(name string) string {
		if name == "price_amount" {
			return "p.price_amount"
		}
		return name
	}

	q := repository.And(price.Gt(100000))
	clause, _ := qb.Build(q, mapper)

	if clause != "p.price_amount > $1" {
		t.Errorf("clause = %q, want %q", clause, "p.price_amount > $1")
	}
}

// ---------- Build — parameter numbering ----------

func TestBuild_param_numbering_sequential(t *testing.T) {
	qb := NewQueryBuilder()
	a := repository.Field[string]{Name: "a"}
	b := repository.Field[int]{Name: "b"}
	c := repository.Field[bool]{Name: "c"}
	q := repository.And(a.Eq("x"), b.Gt(5), c.Eq(false))

	clause, args := qb.Build(q, identity)

	if clause != "(a = $1 AND b > $2 AND c = $3)" {
		t.Errorf("clause = %q", clause)
	}
	if len(args) != 3 {
		t.Fatalf("args len = %d, want 3", len(args))
	}
	if args[0] != "x" || args[1] != 5 || args[2] != false {
		t.Errorf("args = %v, want [x 5 false]", args)
	}
}

// ---------- BuildOrderBy ----------

func TestBuildOrderBy(t *testing.T) {
	t.Run("empty_sorts", func(t *testing.T) {
		result := BuildOrderBy(nil, identity)
		if result != "" {
			t.Errorf("result = %q, want empty", result)
		}
	})

	t.Run("empty_slice", func(t *testing.T) {
		result := BuildOrderBy([]repository.SortField{}, identity)
		if result != "" {
			t.Errorf("result = %q, want empty", result)
		}
	})

	t.Run("single_asc", func(t *testing.T) {
		sorts := []repository.SortField{repository.Asc("name")}
		result := BuildOrderBy(sorts, identity)

		if result != " ORDER BY name ASC" {
			t.Errorf("result = %q, want %q", result, " ORDER BY name ASC")
		}
	})

	t.Run("single_desc", func(t *testing.T) {
		sorts := []repository.SortField{repository.DescSort("price")}
		result := BuildOrderBy(sorts, identity)

		if result != " ORDER BY price DESC" {
			t.Errorf("result = %q, want %q", result, " ORDER BY price DESC")
		}
	})

	t.Run("multiple_sorts", func(t *testing.T) {
		sorts := []repository.SortField{
			repository.DescSort("created_at"),
			repository.Asc("name"),
		}
		result := BuildOrderBy(sorts, identity)

		if result != " ORDER BY created_at DESC, name ASC" {
			t.Errorf("result = %q, want %q", result, " ORDER BY created_at DESC, name ASC")
		}
	})

	t.Run("with_field_mapper", func(t *testing.T) {
		mapper := func(name string) string {
			if name == "price" {
				return "p.price_amount"
			}
			return name
		}
		sorts := []repository.SortField{repository.DescSort("price")}
		result := BuildOrderBy(sorts, mapper)

		if result != " ORDER BY p.price_amount DESC" {
			t.Errorf("result = %q, want %q", result, " ORDER BY p.price_amount DESC")
		}
	})

	t.Run("three_sorts", func(t *testing.T) {
		sorts := []repository.SortField{
			repository.DescSort("created_at"),
			repository.Asc("city"),
			repository.DescSort("price"),
		}
		result := BuildOrderBy(sorts, identity)

		want := " ORDER BY created_at DESC, city ASC, price DESC"
		if result != want {
			t.Errorf("result = %q, want %q", result, want)
		}
	})
}
