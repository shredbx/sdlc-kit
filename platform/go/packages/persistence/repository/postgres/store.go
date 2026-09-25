package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/shredbx/sbx-core/pkg/database"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// PostgresStore implements Repository[M] using PostgreSQL via pgx.
// It uses a Mapper[M] to handle entity-specific column mapping.
type PostgresStore[M any] struct {
	db     *database.DB
	mapper Mapper[M]
}

// NewPostgresStore creates a PostgreSQL-backed Repository[M].
func NewPostgresStore[M any](db *database.DB, mapper Mapper[M]) *PostgresStore[M] {
	return &PostgresStore[M]{db: db, mapper: mapper}
}

// Get retrieves a single entity by ID.
func (s *PostgresStore[M]) Get(ctx context.Context, id string) (M, error) {
	var zero M
	cols := strings.Join(s.mapper.Columns(), ", ")
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE id = $1 AND deleted_at IS NULL",
		cols, s.mapper.SelectFrom(),
	)

	row := s.db.QueryRow(ctx, query, id)
	result, err := s.mapper.FromRow(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return zero, repository.NewNotFoundError(s.mapper.TableName(), id)
		}
		return zero, fmt.Errorf("repository get %s/%s: %w", s.mapper.TableName(), id, err)
	}

	return result, nil
}

// List retrieves entities matching the given options with total count.
func (s *PostgresStore[M]) List(ctx context.Context, opts repository.ListOptions) ([]M, int, error) {
	table := s.mapper.TableName()
	selectFrom := s.mapper.SelectFrom()
	cols := strings.Join(s.mapper.Columns(), ", ")
	fieldMapper := s.mapper.FieldColumn

	// Build WHERE clause from filter. The soft-delete gate flips to IS NOT NULL
	// when the caller asks for the archived population (ListOptions.OnlyDeleted).
	qb := NewQueryBuilder()
	deletedOp := repository.OpIsNull
	if opts.OnlyDeleted {
		deletedOp = repository.OpIsNotNull
	}
	baseFilter := repository.And(repository.Predicate{
		FieldName: "deleted_at",
		Op:        deletedOp,
	})

	var combinedFilter repository.Query
	if opts.Filter.IsEmpty() {
		combinedFilter = baseFilter
	} else {
		combinedFilter = repository.Query{
			And: []repository.Query{baseFilter, opts.Filter},
		}
	}

	whereClause, whereArgs := qb.Build(combinedFilter, fieldMapper)

	// Count query — uses primary table with same alias as data query
	// (WHERE clause references aliased columns like p.is_published)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", selectFrom)
	if whereClause != "" {
		countQuery += " WHERE " + whereClause
	}

	var total int
	if err := s.db.QueryRow(ctx, countQuery, whereArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository count %s: %w", table, err)
	}

	// Data query — uses SelectFrom (may include JOINs)
	dataQuery := fmt.Sprintf("SELECT %s FROM %s", cols, selectFrom)
	if whereClause != "" {
		dataQuery += " WHERE " + whereClause
	}

	dataQuery += BuildOrderBy(opts.Sort, fieldMapper)

	if opts.Limit > 0 {
		dataQuery += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		dataQuery += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := s.db.Query(ctx, dataQuery, whereArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository list %s: %w", table, err)
	}
	defer rows.Close()

	var items []M
	for rows.Next() {
		item, err := s.mapper.FromRow(rows.Scan)
		if err != nil {
			return nil, 0, fmt.Errorf("repository scan %s: %w", table, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository rows %s: %w", table, err)
	}

	if items == nil {
		items = make([]M, 0)
	}

	return items, total, nil
}

// facetBaseFilter combines the caller's filter with the soft-delete gate using
// EXACTLY the same composition as List, so a faceted count's base WHERE is
// produced by identical machinery to the list WHERE — they cannot drift.
func facetBaseFilter(filter repository.Query) repository.Query {
	baseFilter := repository.And(repository.Predicate{
		FieldName: "deleted_at",
		Op:        repository.OpIsNull,
	})
	if filter.IsEmpty() {
		return baseFilter
	}
	return repository.Query{And: []repository.Query{baseFilter, filter}}
}

// GroupCount implements repository.GroupCounter: COUNT(*) grouped by the column
// that groupField maps to, scoped by opts.Filter + the soft-delete gate. It
// reuses SelectFrom() (with its JOINs), FieldColumn, and the QueryBuilder, so
// the facet WHERE matches the List WHERE byte-for-byte. NULL group values are
// excluded (they have no bucket — mirrors the news category facet skipping NULL
// category_id). opts.Sort/Limit/Offset are ignored.
func (s *PostgresStore[M]) GroupCount(ctx context.Context, opts repository.ListOptions, groupField string) (map[string]int, error) {
	table := s.mapper.TableName()
	selectFrom := s.mapper.SelectFrom()
	fieldMapper := s.mapper.FieldColumn
	groupCol := fieldMapper(groupField)

	qb := NewQueryBuilder()
	whereClause, whereArgs := qb.Build(facetBaseFilter(opts.Filter), fieldMapper)

	query := fmt.Sprintf("SELECT %s, COUNT(*) FROM %s", groupCol, selectFrom)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	query += fmt.Sprintf(" GROUP BY %s", groupCol)

	rows, err := s.db.Query(ctx, query, whereArgs...)
	if err != nil {
		return nil, fmt.Errorf("repository group-count %s by %s: %w", table, groupField, err)
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var code *string
		var n int
		if err := rows.Scan(&code, &n); err != nil {
			return nil, fmt.Errorf("repository group-count scan %s: %w", table, err)
		}
		if code == nil {
			continue // NULL group value has no bucket
		}
		out[*code] = n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository group-count rows %s: %w", table, err)
	}
	return out, nil
}

// RangeCounts implements repository.RangeCounter: COUNT(*) FILTER (WHERE <bucket>)
// for each bucket in ONE query, all sharing opts.Filter + the soft-delete gate as
// the base WHERE. Every clause is compiled by the SAME QueryBuilder instance as
// the list filter, so bucket predicates can never drift from list semantics.
// opts.Sort/Limit/Offset are ignored. A nil/empty bucket list returns an empty map.
func (s *PostgresStore[M]) RangeCounts(ctx context.Context, opts repository.ListOptions, buckets []repository.RangeBucket) (map[string]int, error) {
	out := make(map[string]int)
	if len(buckets) == 0 {
		return out, nil
	}

	table := s.mapper.TableName()
	selectFrom := s.mapper.SelectFrom()
	fieldMapper := s.mapper.FieldColumn

	// ONE shared QueryBuilder so the base WHERE and every bucket FILTER share a
	// single, monotonic placeholder counter ($1, $2, …) and arg list.
	qb := NewQueryBuilder()
	whereClause, _ := qb.Build(facetBaseFilter(opts.Filter), fieldMapper)

	selects := make([]string, 0, len(buckets))
	for i, b := range buckets {
		filterClause := qb.buildNode(b.Where, fieldMapper)
		if filterClause == "" {
			// Empty bucket predicate → counts the whole base population.
			selects = append(selects, fmt.Sprintf("COUNT(*) AS c%d", i))
			continue
		}
		selects = append(selects, fmt.Sprintf("COUNT(*) FILTER (WHERE %s) AS c%d", filterClause, i))
	}

	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(selects, ", "), selectFrom)
	if whereClause != "" {
		query += " WHERE " + whereClause
	}

	dest := make([]any, len(buckets))
	vals := make([]int, len(buckets))
	for i := range vals {
		dest[i] = &vals[i]
	}
	if err := s.db.QueryRow(ctx, query, qb.args...).Scan(dest...); err != nil {
		return nil, fmt.Errorf("repository range-counts %s: %w", table, err)
	}
	for i, b := range buckets {
		out[b.Label] = vals[i]
	}
	return out, nil
}

// Create inserts a new entity.
func (s *PostgresStore[M]) Create(ctx context.Context, model M) (M, error) {
	var zero M
	row, err := s.mapper.ToRow(model)
	if err != nil {
		return zero, fmt.Errorf("repository create %s: %w", s.mapper.TableName(), err)
	}

	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	args := make([]any, 0, len(row))
	i := 1
	for col, val := range row {
		columns = append(columns, col)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		args = append(args, val)
		i++
	}

	// Two-step write: INSERT into the target table returning only the unaliased
	// "id" column, then re-fetch via Get to include joined attribute-group
	// columns. Mirrors Update's pattern — mapper.Columns() returns SELECT-aliased
	// names (e.g. p.id, loc.street) that aren't reachable from a single-table
	// INSERT, so they cannot appear in the RETURNING clause.
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		s.mapper.TableName(),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	var id string
	if err := s.db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		if isUniqueViolation(err) {
			return zero, fmt.Errorf("%w: %v", repository.ErrConflict, err)
		}
		return zero, fmt.Errorf("repository create %s: %w", s.mapper.TableName(), err)
	}

	return s.Get(ctx, id)
}

// Update modifies an existing entity by ID.
func (s *PostgresStore[M]) Update(ctx context.Context, id string, model M) (M, error) {
	var zero M
	row, err := s.mapper.ToRow(model)
	if err != nil {
		return zero, fmt.Errorf("repository update %s/%s: %w", s.mapper.TableName(), id, err)
	}

	setClauses := make([]string, 0, len(row))
	args := make([]any, 0, len(row)+1)
	i := 1
	for col, val := range row {
		if col == "id" || col == "created_at" {
			continue // Never update ID or created_at
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, i))
		args = append(args, val)
		i++
	}

	if len(setClauses) == 0 {
		return zero, fmt.Errorf("repository update %s/%s: no fields to update", s.mapper.TableName(), id)
	}

	// Add ID as last parameter for WHERE clause
	args = append(args, id)

	// Two-step write: UPDATE the target table (no RETURNING because Columns()
	// is aliased for joined SELECTs and those joined columns aren't reachable
	// from a single-table UPDATE), then re-fetch via Get to include joined
	// attribute-group columns.
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d AND deleted_at IS NULL",
		s.mapper.TableName(),
		strings.Join(setClauses, ", "),
		i,
	)

	tag, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return zero, fmt.Errorf("%w: %v", repository.ErrConflict, err)
		}
		return zero, fmt.Errorf("repository update %s/%s: %w", s.mapper.TableName(), id, err)
	}
	if tag.RowsAffected() == 0 {
		return zero, repository.NewNotFoundError(s.mapper.TableName(), id)
	}

	return s.Get(ctx, id)
}

// Delete soft-deletes an entity by setting deleted_at.
func (s *PostgresStore[M]) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL",
		s.mapper.TableName(),
	)

	tag, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository delete %s/%s: %w", s.mapper.TableName(), id, err)
	}

	if tag.RowsAffected() == 0 {
		return repository.NewNotFoundError(s.mapper.TableName(), id)
	}

	return nil
}

// Restore clears deleted_at on a soft-deleted row — the "unarchive" half of the
// archive lifecycle (2607-102). Gated on deleted_at IS NOT NULL so restoring a
// live (or unknown) id is a NotFound, never a silent no-op.
func (s *PostgresStore[M]) Restore(ctx context.Context, id string) error {
	query := fmt.Sprintf(
		"UPDATE %s SET deleted_at = NULL WHERE id = $1 AND deleted_at IS NOT NULL",
		s.mapper.TableName(),
	)

	tag, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository restore %s/%s: %w", s.mapper.TableName(), id, err)
	}

	if tag.RowsAffected() == 0 {
		return repository.NewNotFoundError(s.mapper.TableName(), id)
	}

	return nil
}

// HardDelete permanently removes a row by id — deliberately NO deleted_at filter,
// so previously soft-deleted rows are purgeable through the same path. Owner
// ruling 2026-07-18: an entity with no restore path in its UI hard-deletes,
// period. Delete (soft) stays the default for consumers that keep one. The
// caller owns cascading its owned resources (files, memberships) BEFORE this —
// FK cascades fire here and take the reference rows with them.
func (s *PostgresStore[M]) HardDelete(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.mapper.TableName())

	tag, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository hard-delete %s/%s: %w", s.mapper.TableName(), id, err)
	}

	if tag.RowsAffected() == 0 {
		return repository.NewNotFoundError(s.mapper.TableName(), id)
	}

	return nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
