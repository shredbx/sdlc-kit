package repository

// Op represents a comparison operator for predicates.
type Op int

const (
	OpEq        Op = iota // =
	OpNeq                 // !=
	OpGt                  // >
	OpGte                 // >=
	OpLt                  // <
	OpLte                 // <=
	OpIn                  // IN (...)
	OpLike                // LIKE
	OpIsNull              // IS NULL
	OpIsNotNull           // IS NOT NULL
	// OpArrayContains: array column contains ALL given elements (Postgres `@>`).
	OpArrayContains // col @> $N::TEXT[]
)

// Predicate is a single field comparison — storage-agnostic.
// Constructed via Field[T] methods which enforce type safety at compile time.
type Predicate struct {
	FieldName string
	Op        Op
	Value     any // typed at construction via Field[T], nil for IsNull/IsNotNull
}

// Query is a composable filter tree. Exactly one of And, Or, or Pred should be set.
//
//   - And: all sub-queries must match
//   - Or: any sub-query must match
//   - Pred: leaf predicate node
//
// An empty Query matches everything.
type Query struct {
	And  []Query
	Or   []Query
	Pred *Predicate
}

// IsEmpty returns true if the query has no filters.
func (q Query) IsEmpty() bool {
	return len(q.And) == 0 && len(q.Or) == 0 && q.Pred == nil
}

// And combines predicates into a Query where all must match.
func And(predicates ...Predicate) Query {
	subs := make([]Query, len(predicates))
	for i, p := range predicates {
		p := p
		subs[i] = Query{Pred: &p}
	}
	return Query{And: subs}
}

// Or combines predicates into a Query where any can match.
func Or(predicates ...Predicate) Query {
	subs := make([]Query, len(predicates))
	for i, p := range predicates {
		p := p
		subs[i] = Query{Pred: &p}
	}
	return Query{Or: subs}
}

// Field is a typed column descriptor. Field[T] methods produce Predicate
// values where the comparison value is guaranteed to be type T at compile time.
//
// Usage:
//
//	var Fields = struct {
//	    PriceAmount Field[int64]
//	    City        Field[string]
//	    IsPublished Field[bool]
//	}{
//	    PriceAmount: Field[int64]{Name: "price_amount"},
//	    City:        Field[string]{Name: "city"},
//	    IsPublished: Field[bool]{Name: "is_published"},
//	}
//
//	// Compiles:
//	Fields.PriceAmount.Gte(500000)
//
//	// Won't compile:
//	Fields.PriceAmount.Gte("wrong type")
type Field[T any] struct {
	Name string // Logical field name (maps to column via Mapper)
}

// Eq creates an equality predicate.
func (f Field[T]) Eq(val T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpEq, Value: val}
}

// Neq creates a not-equal predicate.
func (f Field[T]) Neq(val T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpNeq, Value: val}
}

// Gt creates a greater-than predicate.
func (f Field[T]) Gt(val T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpGt, Value: val}
}

// Gte creates a greater-than-or-equal predicate.
func (f Field[T]) Gte(val T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpGte, Value: val}
}

// Lt creates a less-than predicate.
func (f Field[T]) Lt(val T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpLt, Value: val}
}

// Lte creates a less-than-or-equal predicate.
func (f Field[T]) Lte(val T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpLte, Value: val}
}

// In creates an IN-list predicate.
func (f Field[T]) In(vals []T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpIn, Value: vals}
}

// Like creates a pattern-matching predicate.
func (f Field[T]) Like(pattern T) Predicate {
	return Predicate{FieldName: f.Name, Op: OpLike, Value: pattern}
}

// IsNull creates a NULL check predicate.
func (f Field[T]) IsNull() Predicate {
	return Predicate{FieldName: f.Name, Op: OpIsNull}
}

// IsNotNull creates a NOT NULL check predicate.
func (f Field[T]) IsNotNull() Predicate {
	return Predicate{FieldName: f.Name, Op: OpIsNotNull}
}

// ArrayField is a typed column descriptor for a Postgres TEXT[] column.
// It mirrors Field's idiom but produces array-membership predicates whose
// comparison value is guaranteed to be []string at compile time. Go does not
// permit methods on a specialized generic instantiation (e.g. Field[string]),
// so array operators live on this dedicated descriptor.
//
// Usage:
//
//	var Fields = struct {
//	    Tags ArrayField
//	}{
//	    Tags: ArrayField{Name: "tags"},
//	}
//
//	// Compiles — bound as ONE TEXT[] parameter, never inlined:
//	Fields.Tags.Contains([]string{"pool", "sea-view"})
type ArrayField struct {
	Name string // Logical field name (maps to column via Mapper)
}

// Contains creates an array-containment predicate: the TEXT[] column must
// contain ALL of vals (Postgres `col @> $N::TEXT[]`). The slice is carried as
// the predicate Value and bound later as exactly one parameter — exactly as
// Field[T].In carries its slice — never stringified or inlined.
func (f ArrayField) Contains(vals []string) Predicate {
	return Predicate{FieldName: f.Name, Op: OpArrayContains, Value: vals}
}
