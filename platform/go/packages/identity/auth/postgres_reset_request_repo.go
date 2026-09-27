package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresResetRequestRepo struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresResetRequestRepo creates a Postgres-backed ResetRequestRepository.
// Panics if schema is not a valid PostgreSQL identifier (M1).
func NewPostgresResetRequestRepo(pool *pgxpool.Pool, schema string) ResetRequestRepository {
	MustValidateSchema(schema)
	return &postgresResetRequestRepo{pool: pool, schema: schema}
}

func (r *postgresResetRequestRepo) table() string {
	return r.schema + ".password_reset_requests"
}

func (r *postgresResetRequestRepo) Create(ctx context.Context, req *PasswordResetRequest) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, status, reason, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`, r.table())

	_, err := r.pool.Exec(ctx, query,
		req.ID, req.UserID, req.Status, req.Reason, req.ExpiresAt, req.CreatedAt)
	if err != nil {
		return fmt.Errorf("create reset request: %w", err)
	}
	return nil
}

func (r *postgresResetRequestRepo) Get(ctx context.Context, id uuid.UUID) (*PasswordResetRequest, error) {
	query := fmt.Sprintf(`SELECT r.id, r.user_id, u.email, r.status, r.reason,
		r.resolved_by, r.resolved_at, r.magic_link_id, r.expires_at, r.created_at
		FROM %s r JOIN %s.users u ON u.id = r.user_id
		WHERE r.id = $1`, r.table(), r.schema)

	return r.scanRequest(r.pool.QueryRow(ctx, query, id))
}

func (r *postgresResetRequestRepo) GetPendingByUser(ctx context.Context, userID uuid.UUID) (*PasswordResetRequest, error) {
	query := fmt.Sprintf(`SELECT r.id, r.user_id, u.email, r.status, r.reason,
		r.resolved_by, r.resolved_at, r.magic_link_id, r.expires_at, r.created_at
		FROM %s r JOIN %s.users u ON u.id = r.user_id
		WHERE r.user_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC LIMIT 1`, r.table(), r.schema)

	return r.scanRequest(r.pool.QueryRow(ctx, query, userID))
}

func (r *postgresResetRequestRepo) ListPending(ctx context.Context) ([]PasswordResetRequest, error) {
	query := fmt.Sprintf(`SELECT r.id, r.user_id, u.email, r.status, r.reason,
		r.resolved_by, r.resolved_at, r.magic_link_id, r.expires_at, r.created_at
		FROM %s r JOIN %s.users u ON u.id = r.user_id
		WHERE r.status = 'pending'
		ORDER BY r.created_at ASC`, r.table(), r.schema)

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list pending reset requests: %w", err)
	}
	defer rows.Close()

	var requests []PasswordResetRequest
	for rows.Next() {
		var req PasswordResetRequest
		err := rows.Scan(&req.ID, &req.UserID, &req.Email, &req.Status, &req.Reason,
			&req.ResolvedBy, &req.ResolvedAt, &req.MagicLinkID, &req.ExpiresAt, &req.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan reset request row: %w", err)
		}
		requests = append(requests, req)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reset request rows: %w", err)
	}
	return requests, nil
}

func (r *postgresResetRequestRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ResetRequestStatus, resolvedBy uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET status = $1, resolved_by = $2, resolved_at = NOW()
		WHERE id = $3`, r.table())

	tag, err := r.pool.Exec(ctx, query, status, resolvedBy, id)
	if err != nil {
		return fmt.Errorf("update reset request status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResetRequestNotFound
	}
	return nil
}

func (r *postgresResetRequestRepo) SetMagicLink(ctx context.Context, id, magicLinkID uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET magic_link_id = $1 WHERE id = $2`, r.table())

	tag, err := r.pool.Exec(ctx, query, magicLinkID, id)
	if err != nil {
		return fmt.Errorf("set magic link on reset request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrResetRequestNotFound
	}
	return nil
}

func (r *postgresResetRequestRepo) ExpireOld(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`UPDATE %s SET status = 'expired'
		WHERE status = 'pending' AND expires_at < NOW()`, r.table())

	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("expire old reset requests: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *postgresResetRequestRepo) scanRequest(row pgx.Row) (*PasswordResetRequest, error) {
	req := &PasswordResetRequest{}
	err := row.Scan(&req.ID, &req.UserID, &req.Email, &req.Status, &req.Reason,
		&req.ResolvedBy, &req.ResolvedAt, &req.MagicLinkID, &req.ExpiresAt, &req.CreatedAt)
	if err != nil {
		// M2: use errors.Is for pgx.ErrNoRows — robust to wrapping.
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrResetRequestNotFound
		}
		return nil, fmt.Errorf("scan reset request: %w", err)
	}
	return req, nil
}
