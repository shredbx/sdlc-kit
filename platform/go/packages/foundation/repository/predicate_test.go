package repository

import (
	"testing"
)

// ---------- Field[T] predicate methods ----------

func TestField_Eq(t *testing.T) {
	f := Field[string]{Name: "city"}
	p := f.Eq("Amsterdam")

	if p.FieldName != "city" {
		t.Errorf("FieldName = %q, want %q", p.FieldName, "city")
	}
	if p.Op != OpEq {
		t.Errorf("Op = %d, want OpEq (%d)", p.Op, OpEq)
	}
	if p.Value != "Amsterdam" {
		t.Errorf("Value = %v, want %q", p.Value, "Amsterdam")
	}
}

func TestField_Neq(t *testing.T) {
	f := Field[int]{Name: "status"}
	p := f.Neq(0)

	if p.Op != OpNeq {
		t.Errorf("Op = %d, want OpNeq (%d)", p.Op, OpNeq)
	}
	if p.Value != 0 {
		t.Errorf("Value = %v, want 0", p.Value)
	}
}

func TestField_Gt(t *testing.T) {
	f := Field[int64]{Name: "price"}
	p := f.Gt(100000)

	if p.Op != OpGt {
		t.Errorf("Op = %d, want OpGt (%d)", p.Op, OpGt)
	}
	if p.Value != int64(100000) {
		t.Errorf("Value = %v, want 100000", p.Value)
	}
}

func TestField_Gte(t *testing.T) {
	f := Field[int]{Name: "bedrooms"}
	p := f.Gte(3)

	if p.Op != OpGte {
		t.Errorf("Op = %d, want OpGte (%d)", p.Op, OpGte)
	}
}

func TestField_Lt(t *testing.T) {
	f := Field[float64]{Name: "rating"}
	p := f.Lt(4.5)

	if p.Op != OpLt {
		t.Errorf("Op = %d, want OpLt (%d)", p.Op, OpLt)
	}
}

func TestField_Lte(t *testing.T) {
	f := Field[int]{Name: "max_guests"}
	p := f.Lte(10)

	if p.Op != OpLte {
		t.Errorf("Op = %d, want OpLte (%d)", p.Op, OpLte)
	}
}

func TestField_In(t *testing.T) {
	f := Field[string]{Name: "status"}
	vals := []string{"active", "pending"}
	p := f.In(vals)

	if p.Op != OpIn {
		t.Errorf("Op = %d, want OpIn (%d)", p.Op, OpIn)
	}
	got, ok := p.Value.([]string)
	if !ok {
		t.Fatalf("Value type = %T, want []string", p.Value)
	}
	if len(got) != 2 || got[0] != "active" || got[1] != "pending" {
		t.Errorf("Value = %v, want %v", got, vals)
	}
}

func TestField_Like(t *testing.T) {
	f := Field[string]{Name: "name"}
	p := f.Like("%beach%")

	if p.Op != OpLike {
		t.Errorf("Op = %d, want OpLike (%d)", p.Op, OpLike)
	}
	if p.Value != "%beach%" {
		t.Errorf("Value = %v, want %q", p.Value, "%beach%")
	}
}

func TestField_IsNull(t *testing.T) {
	f := Field[string]{Name: "deleted_at"}
	p := f.IsNull()

	if p.Op != OpIsNull {
		t.Errorf("Op = %d, want OpIsNull (%d)", p.Op, OpIsNull)
	}
	if p.Value != nil {
		t.Errorf("Value = %v, want nil", p.Value)
	}
}

func TestField_IsNotNull(t *testing.T) {
	f := Field[string]{Name: "email"}
	p := f.IsNotNull()

	if p.Op != OpIsNotNull {
		t.Errorf("Op = %d, want OpIsNotNull (%d)", p.Op, OpIsNotNull)
	}
	if p.Value != nil {
		t.Errorf("Value = %v, want nil", p.Value)
	}
}

func TestField_preserves_field_name(t *testing.T) {
	f := Field[bool]{Name: "is_published"}

	ops := []struct {
		name string
		pred Predicate
	}{
		{"Eq", f.Eq(true)},
		{"Neq", f.Neq(false)},
		{"IsNull", f.IsNull()},
		{"IsNotNull", f.IsNotNull()},
	}

	for _, tt := range ops {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pred.FieldName != "is_published" {
				t.Errorf("FieldName = %q, want %q", tt.pred.FieldName, "is_published")
			}
		})
	}
}

// ---------- And / Or composition ----------

func TestAnd(t *testing.T) {
	t.Run("single_predicate", func(t *testing.T) {
		city := Field[string]{Name: "city"}
		q := And(city.Eq("Paris"))

		if len(q.And) != 1 {
			t.Fatalf("And len = %d, want 1", len(q.And))
		}
		if q.And[0].Pred == nil {
			t.Fatal("And[0].Pred is nil")
		}
		if q.And[0].Pred.FieldName != "city" {
			t.Errorf("FieldName = %q, want %q", q.And[0].Pred.FieldName, "city")
		}
	})

	t.Run("multiple_predicates", func(t *testing.T) {
		city := Field[string]{Name: "city"}
		published := Field[bool]{Name: "is_published"}
		q := And(city.Eq("Paris"), published.Eq(true))

		if len(q.And) != 2 {
			t.Fatalf("And len = %d, want 2", len(q.And))
		}
		if q.And[0].Pred.FieldName != "city" {
			t.Errorf("And[0] field = %q, want %q", q.And[0].Pred.FieldName, "city")
		}
		if q.And[1].Pred.FieldName != "is_published" {
			t.Errorf("And[1] field = %q, want %q", q.And[1].Pred.FieldName, "is_published")
		}
	})

	t.Run("zero_predicates", func(t *testing.T) {
		q := And()
		if len(q.And) != 0 {
			t.Errorf("And() len = %d, want 0", len(q.And))
		}
		if !q.IsEmpty() {
			t.Error("And() should be empty")
		}
	})

	t.Run("wraps_each_predicate_as_leaf", func(t *testing.T) {
		price := Field[int]{Name: "price"}
		q := And(price.Gt(100), price.Lt(500))

		for i, sub := range q.And {
			if sub.Pred == nil {
				t.Errorf("And[%d].Pred should not be nil", i)
			}
			if len(sub.And) != 0 || len(sub.Or) != 0 {
				t.Errorf("And[%d] should be a leaf node", i)
			}
		}
	})

	t.Run("does_not_set_or_or_pred", func(t *testing.T) {
		city := Field[string]{Name: "city"}
		q := And(city.Eq("Rome"))

		if len(q.Or) != 0 {
			t.Error("And should not set Or")
		}
		if q.Pred != nil {
			t.Error("And should not set Pred")
		}
	})
}

func TestOr(t *testing.T) {
	t.Run("single_predicate", func(t *testing.T) {
		status := Field[string]{Name: "status"}
		q := Or(status.Eq("active"))

		if len(q.Or) != 1 {
			t.Fatalf("Or len = %d, want 1", len(q.Or))
		}
		if q.Or[0].Pred.FieldName != "status" {
			t.Errorf("FieldName = %q, want %q", q.Or[0].Pred.FieldName, "status")
		}
	})

	t.Run("multiple_predicates", func(t *testing.T) {
		status := Field[string]{Name: "status"}
		q := Or(status.Eq("active"), status.Eq("pending"))

		if len(q.Or) != 2 {
			t.Fatalf("Or len = %d, want 2", len(q.Or))
		}
	})

	t.Run("zero_predicates", func(t *testing.T) {
		q := Or()
		if len(q.Or) != 0 {
			t.Errorf("Or() len = %d, want 0", len(q.Or))
		}
		if !q.IsEmpty() {
			t.Error("Or() should be empty")
		}
	})
}

// ---------- Query ----------

func TestQuery_IsEmpty(t *testing.T) {
	t.Run("zero_value_is_empty", func(t *testing.T) {
		var q Query
		if !q.IsEmpty() {
			t.Error("zero-value Query should be empty")
		}
	})

	t.Run("with_and_not_empty", func(t *testing.T) {
		q := And(Field[string]{Name: "x"}.Eq("y"))
		if q.IsEmpty() {
			t.Error("Query with And should not be empty")
		}
	})

	t.Run("with_or_not_empty", func(t *testing.T) {
		q := Or(Field[string]{Name: "x"}.Eq("y"))
		if q.IsEmpty() {
			t.Error("Query with Or should not be empty")
		}
	})

	t.Run("with_pred_not_empty", func(t *testing.T) {
		p := Field[int]{Name: "n"}.Eq(1)
		q := Query{Pred: &p}
		if q.IsEmpty() {
			t.Error("Query with Pred should not be empty")
		}
	})
}
