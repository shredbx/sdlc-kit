package postgres

import (
	"fmt"
	"strings"

	"github.com/shredbx/sbx-core/pkg/repository"
)

// QueryBuilder compiles a repository.Query AST into a PostgreSQL WHERE clause
// with parameterized placeholders ($1, $2, ...).
type QueryBuilder struct {
	args    []any
	counter int
}

// NewQueryBuilder creates a new QueryBuilder.
func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{}
}

// Build compiles a Query into a WHERE clause and parameter list.
// Returns empty string and nil args for an empty query.
func (qb *QueryBuilder) Build(q repository.Query, fieldMapper func(string) string) (string, []any) {
	clause := qb.buildNode(q, fieldMapper)
	if clause == "" {
		return "", nil
	}
	return clause, qb.args
}

func (qb *QueryBuilder) buildNode(q repository.Query, fieldMapper func(string) string) string {
	if q.Pred != nil {
		return qb.buildPredicate(*q.Pred, fieldMapper)
	}

	if len(q.And) > 0 {
		parts := make([]string, 0, len(q.And))
		for _, sub := range q.And {
			clause := qb.buildNode(sub, fieldMapper)
			if clause != "" {
				parts = append(parts, clause)
			}
		}
		if len(parts) == 0 {
			return ""
		}
		if len(parts) == 1 {
			return parts[0]
		}
		return "(" + strings.Join(parts, " AND ") + ")"
	}

	if len(q.Or) > 0 {
		parts := make([]string, 0, len(q.Or))
		for _, sub := range q.Or {
			clause := qb.buildNode(sub, fieldMapper)
			if clause != "" {
				parts = append(parts, clause)
			}
		}
		if len(parts) == 0 {
			return ""
		}
		if len(parts) == 1 {
			return parts[0]
		}
		return "(" + strings.Join(parts, " OR ") + ")"
	}

	return ""
}

func (qb *QueryBuilder) buildPredicate(p repository.Predicate, fieldMapper func(string) string) string {
	col := fieldMapper(p.FieldName)

	switch p.Op {
	case repository.OpIsNull:
		return col + " IS NULL"
	case repository.OpIsNotNull:
		return col + " IS NOT NULL"
	case repository.OpIn:
		return qb.buildIn(col, p.Value)
	case repository.OpArrayContains:
		return qb.buildArrayContains(col, p.Value)
	default:
		qb.counter++
		qb.args = append(qb.args, p.Value)
		return fmt.Sprintf("%s %s $%d", col, opToSQL(p.Op), qb.counter)
	}
}

func (qb *QueryBuilder) buildIn(col string, value any) string {
	// Handle slice types — reflect-free by checking common types
	switch vals := value.(type) {
	case []string:
		if len(vals) == 0 {
			return "FALSE"
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			qb.counter++
			qb.args = append(qb.args, v)
			placeholders[i] = fmt.Sprintf("$%d", qb.counter)
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))
	case []int64:
		if len(vals) == 0 {
			return "FALSE"
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			qb.counter++
			qb.args = append(qb.args, v)
			placeholders[i] = fmt.Sprintf("$%d", qb.counter)
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))
	case []int:
		if len(vals) == 0 {
			return "FALSE"
		}
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			qb.counter++
			qb.args = append(qb.args, v)
			placeholders[i] = fmt.Sprintf("$%d", qb.counter)
		}
		return fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", "))
	default:
		// Fallback: single-value IN
		qb.counter++
		qb.args = append(qb.args, value)
		return fmt.Sprintf("%s IN ($%d)", col, qb.counter)
	}
}

// buildArrayContains emits a parameterized Postgres array-containment predicate
// `col @> $N::TEXT[]`. The whole []string is bound as ONE parameter (pgx maps
// []string to TEXT[] natively — no pq.Array), so the placeholder counter is
// incremented exactly once. Mirrors buildIn's arg-binding discipline.
//
// A dedicated builder (rather than the opToSQL/default infix path) is required:
// the operator needs a static `::TEXT[]` cast and single-parameter binding,
// and opToSQL's default returns "=" which would silently produce wrong results.
func (qb *QueryBuilder) buildArrayContains(col string, value any) string {
	qb.counter++
	qb.args = append(qb.args, value)
	return fmt.Sprintf("%s @> $%d::TEXT[]", col, qb.counter)
}

func opToSQL(op repository.Op) string {
	switch op {
	case repository.OpEq:
		return "="
	case repository.OpNeq:
		return "!="
	case repository.OpGt:
		return ">"
	case repository.OpGte:
		return ">="
	case repository.OpLt:
		return "<"
	case repository.OpLte:
		return "<="
	case repository.OpLike:
		return "LIKE"
	default:
		return "="
	}
}

// BuildOrderBy generates an ORDER BY clause from SortFields.
// Returns empty string if no sort fields are provided.
func BuildOrderBy(sorts []repository.SortField, fieldMapper func(string) string) string {
	if len(sorts) == 0 {
		return ""
	}

	parts := make([]string, len(sorts))
	for i, s := range sorts {
		col := fieldMapper(s.Field)
		if s.Desc {
			parts[i] = col + " DESC"
		} else {
			parts[i] = col + " ASC"
		}
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}
