package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresMagicLinkRepo struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresMagicLinkRepo creates a Postgres-backed MagicLinkRepository.
// Panics if schema is not a valid PostgreSQL identifier (M1).
func NewPostgresMagicLinkRepo(pool *pgxpool.Pool, schema string) MagicLinkRepository {
	MustValidateSchema(schema)
	return &postgresMagicLinkRepo{pool: pool, schema: schema}
}

func (r *postgresMagicLinkRepo) table() string {
	return r.schema + ".magic_links"
}

func (r *postgresMagicLinkRepo) Create(ctx context.Context, link *MagicLink) error {
	if link.Purpose == "" {
		link.Purpose = MagicLinkPurposeInvite
	}
	// M5: store SHA-256(token) in token_hash; clear the plaintext token column.
	// Raw token stays on the in-memory struct so the caller can build the URL,
	// but never persists to disk. A DB leak cannot be replayed as token theft.
	tokenHash := HashToken(link.Token)
	query := fmt.Sprintf(`INSERT INTO %s (id, token, token_hash, user_id, purpose, expires_at, created_at)
		VALUES ($1, '', $2, $3, $4, $5, $6)`, r.table())

	_, err := r.pool.Exec(ctx, query,
		link.ID, tokenHash, link.UserID, link.Purpose, link.ExpiresAt, link.CreatedAt)
	if err != nil {
		return fmt.Errorf("create magic link: %w", err)
	}
	return nil
}

func (r *postgresMagicLinkRepo) GetByToken(ctx context.Context, token string) (*MagicLink, error) {
	// M5: lookup by hash, never by plaintext. Hash the input before query.
	tokenHash := HashToken(token)
	query := fmt.Sprintf(`SELECT id, user_id, purpose, expires_at, used_at, created_at
		FROM %s WHERE token_hash = $1`, r.table())

	ml := &MagicLink{Token: token}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&ml.ID, &ml.UserID, &ml.Purpose, &ml.ExpiresAt, &ml.UsedAt, &ml.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMagicLinkNotFound
		}
		return nil, fmt.Errorf("get magic link: %w", err)
	}
	return ml, nil
}

func (r *postgresMagicLinkRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*MagicLink, error) {
	// Returns a link WITHOUT its plaintext token — that data is gone from disk
	// (M5). Idempotent invite return paths must use the original token from the
	// caller's request flow, not regenerate from this row.
	query := fmt.Sprintf(`SELECT id, user_id, purpose, expires_at, used_at, created_at
		FROM %s WHERE user_id = $1 AND used_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC LIMIT 1`, r.table())

	ml := &MagicLink{}
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&ml.ID, &ml.UserID, &ml.Purpose, &ml.ExpiresAt, &ml.UsedAt, &ml.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMagicLinkNotFound
		}
		return nil, fmt.Errorf("get active magic link: %w", err)
	}
	return ml, nil
}

// InvalidateUnusedForUser marks all unused links for a user as used (H5).
// Idempotent — returns count of rows affected.
func (r *postgresMagicLinkRepo) InvalidateUnusedForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := fmt.Sprintf(`UPDATE %s SET used_at = NOW() WHERE user_id = $1 AND used_at IS NULL`, r.table())
	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("invalidate unused magic links: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *postgresMagicLinkRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	// Atomic: UPDATE WHERE used_at IS NULL prevents race condition
	query := fmt.Sprintf(`UPDATE %s SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`, r.table())
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrMagicLinkUsed
	}
	return nil
}
