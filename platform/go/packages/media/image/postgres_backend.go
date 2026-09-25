package image

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresBackend stores image metadata in a PostgreSQL images table.
// Schema is injected per-app (bestierealestate, bestays, shredbx).
type PostgresBackend struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresBackend creates a Postgres-backed image metadata store.
func NewPostgresBackend(pool *pgxpool.Pool, schema string) *PostgresBackend {
	return &PostgresBackend{pool: pool, schema: schema}
}

func (b *PostgresBackend) table() string {
	return b.schema + ".images"
}

const imageColumns = `id, key, url, format, size, alt_text, purpose,
	primary_color, created_by, created_at, updated_at, deleted_at`

// Create inserts a new image metadata row.
func (b *PostgresBackend) Create(ctx context.Context, img *Image) error {
	q := fmt.Sprintf(`
		INSERT INTO %s (
			id, key, url, format, size, alt_text, purpose,
			primary_color, created_by, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,$11
		)`, b.table())

	var createdBy *string
	if img.CreatedBy != "" {
		createdBy = &img.CreatedBy
	}
	var primaryColor *string
	if img.PrimaryColor != "" {
		primaryColor = &img.PrimaryColor
	}

	_, err := b.pool.Exec(ctx, q,
		img.ID, img.Key, img.URL,
		string(img.Format), img.Size, img.AltText, string(img.Purpose),
		primaryColor, createdBy,
		img.CreatedAt, img.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%w: create image: %v", ErrBackendError, err)
	}
	return nil
}

// Get retrieves a non-deleted image by ID.
func (b *PostgresBackend) Get(ctx context.Context, id string) (*Image, error) {
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1 AND deleted_at IS NULL`,
		imageColumns, b.table())
	row := b.pool.QueryRow(ctx, q, id)
	img, err := scanImage(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrImageNotFound
		}
		return nil, fmt.Errorf("%w: get image: %v", ErrBackendError, err)
	}
	return img, nil
}

// Update replaces all mutable fields for an existing image.
func (b *PostgresBackend) Update(ctx context.Context, img *Image) error {
	q := fmt.Sprintf(`
		UPDATE %s SET
			key=$2, url=$3, format=$4, size=$5, alt_text=$6,
			purpose=$7, primary_color=$8, updated_at=$9
		WHERE id=$1 AND deleted_at IS NULL`, b.table())

	var primaryColor *string
	if img.PrimaryColor != "" {
		primaryColor = &img.PrimaryColor
	}

	tag, err := b.pool.Exec(ctx, q,
		img.ID, img.Key, img.URL,
		string(img.Format), img.Size, img.AltText, string(img.Purpose),
		primaryColor, img.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%w: update image: %v", ErrBackendError, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrImageNotFound
	}
	return nil
}

// Delete soft-deletes an image by setting deleted_at.
func (b *PostgresBackend) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`UPDATE %s SET deleted_at=$2 WHERE id=$1 AND deleted_at IS NULL`,
		b.table())
	tag, err := b.pool.Exec(ctx, q, id, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("%w: delete image: %v", ErrBackendError, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrImageNotFound
	}
	return nil
}

// DeleteByKey soft-deletes the image metadata row matching the given object key,
// if any. Idempotent + best-effort: a missing row (e.g. a key-addressed system
// asset with no metadata row, like a baked logo tint) is a no-op, not an error.
// The metadata counterpart of Store.ReclaimByKey (consumer-driven replace-reclaim).
func (b *PostgresBackend) DeleteByKey(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	q := fmt.Sprintf(`UPDATE %s SET deleted_at=$2 WHERE key=$1 AND deleted_at IS NULL`,
		b.table())
	if _, err := b.pool.Exec(ctx, q, key, time.Now().UTC()); err != nil {
		return fmt.Errorf("%w: delete image by key: %v", ErrBackendError, err)
	}
	return nil
}

// ListBySlot returns the non-deleted images stored under the owner-scoped slot prefix
// {owner}/{ownerId}/{facet}/ (or system/{facet}/). The prefix is the package's own key
// scheme (slotPrefix), so this matches on key LIKE — no owner/owner_id columns needed.
func (b *PostgresBackend) ListBySlot(ctx context.Context, owner ImageOwner, ownerID string, facet ImageFacet) ([]Image, error) {
	prefix := slotPrefix(owner, ownerID, facet)
	q := fmt.Sprintf(`SELECT %s FROM %s WHERE key LIKE $1 AND deleted_at IS NULL`,
		imageColumns, b.table())
	rows, err := b.pool.Query(ctx, q, prefix+"%")
	if err != nil {
		return nil, fmt.Errorf("%w: list images by slot: %v", ErrBackendError, err)
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		img, err := scanImage(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: scan image row: %v", ErrBackendError, err)
		}
		images = append(images, *img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: rows error: %v", ErrBackendError, err)
	}
	return images, nil
}

// List returns non-deleted images filtered and paginated by opts.
func (b *PostgresBackend) List(ctx context.Context, opts ListOpts) ([]Image, error) {
	args := []any{}
	argN := 1

	where := "deleted_at IS NULL"
	if opts.Facet != "" {
		where += fmt.Sprintf(" AND purpose=$%d", argN)
		args = append(args, string(opts.Facet))
		argN++
	}

	q := fmt.Sprintf(`SELECT %s FROM %s WHERE %s ORDER BY created_at DESC`,
		imageColumns, b.table(), where)

	if opts.Limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, opts.Limit)
		argN++
	}
	if opts.Offset > 0 {
		q += fmt.Sprintf(" OFFSET $%d", argN)
		args = append(args, opts.Offset)
	}

	rows, err := b.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: list images: %v", ErrBackendError, err)
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		img, err := scanImage(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: scan image row: %v", ErrBackendError, err)
		}
		images = append(images, *img)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: rows error: %v", ErrBackendError, err)
	}
	return images, nil
}

// =============================================================================
// SCANNER
// =============================================================================

type pgxRow interface {
	Scan(dest ...any) error
}

func scanImage(row pgxRow) (*Image, error) {
	var (
		img          Image
		format       string
		purpose      string
		altText      *string
		primaryColor *string
		createdBy    *string
		deletedAt    *time.Time
	)

	// alt_text / primary_color / created_by are nullable columns — scan into pointers
	// so a NULL row value becomes the zero string rather than a scan error (alt_text is
	// optional: most images carry none).
	err := row.Scan(
		&img.ID, &img.Key, &img.URL,
		&format, &img.Size, &altText, &purpose,
		&primaryColor, &createdBy,
		&img.CreatedAt, &img.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	img.Format = ImageFormat(format)
	img.Purpose = ImageFacet(purpose)

	if altText != nil {
		img.AltText = *altText
	}
	if primaryColor != nil {
		img.PrimaryColor = *primaryColor
	}
	if createdBy != nil {
		img.CreatedBy = *createdBy
	}
	img.DeletedAt = deletedAt

	return &img, nil
}
