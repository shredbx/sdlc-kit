package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresSessionRepo struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresSessionRepo creates a Postgres-backed SessionRepository.
// Panics if schema is not a valid PostgreSQL identifier (M1).
func NewPostgresSessionRepo(pool *pgxpool.Pool, schema string) SessionRepository {
	MustValidateSchema(schema)
	return &postgresSessionRepo{pool: pool, schema: schema}
}

func (r *postgresSessionRepo) table() string {
	return r.schema + ".sessions"
}

const sessionColumns = `id, user_id, refresh_token, jti, user_agent, ip_address,
	expires_at, revoked_at, replaced_by_session_id, created_at`

func (r *postgresSessionRepo) Create(ctx context.Context, session *Session) error {
	query := fmt.Sprintf(`INSERT INTO %s (id, user_id, refresh_token, jti, user_agent, ip_address, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, r.table())

	_, err := r.pool.Exec(ctx, query,
		session.ID, session.UserID, session.RefreshToken, session.JTI,
		session.UserAgent, session.IPAddress, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *postgresSessionRepo) Get(ctx context.Context, id uuid.UUID) (*Session, error) {
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1`, sessionColumns, r.table())
	return r.scanSession(r.pool.QueryRow(ctx, query, id))
}

func (r *postgresSessionRepo) GetByRefreshToken(ctx context.Context, token string) (*Session, error) {
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE refresh_token = $1`, sessionColumns, r.table())
	return r.scanSession(r.pool.QueryRow(ctx, query, token))
}

func (r *postgresSessionRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	query := fmt.Sprintf(`SELECT %s FROM %s
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC`, sessionColumns, r.table())

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions by user: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		s := Session{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.RefreshToken, &s.JTI,
			&s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.ReplacedBySessionID, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		// Never expose refresh token in list
		s.RefreshToken = ""
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *postgresSessionRepo) Revoke(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET revoked_at = NOW() WHERE id = $1`, r.table())
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

// RevokeForRotation atomically marks oldID as revoked and points it at newID.
// Used by RefreshToken so a replayed old refresh token can be distinguished from
// an admin-revoked session (theft signal — see C4).
func (r *postgresSessionRepo) RevokeForRotation(ctx context.Context, oldID, newID uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET revoked_at = NOW(), replaced_by_session_id = $2 WHERE id = $1`, r.table())
	tag, err := r.pool.Exec(ctx, query, oldID, newID)
	if err != nil {
		return fmt.Errorf("revoke for rotation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *postgresSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1`, r.table())
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *postgresSessionRepo) DeleteExpired(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(`DELETE FROM %s WHERE expires_at < NOW() AND revoked_at IS NULL`, r.table())
	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *postgresSessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := fmt.Sprintf(`UPDATE %s SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, r.table())
	tag, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return 0, fmt.Errorf("revoke all sessions for user: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *postgresSessionRepo) scanSession(row pgx.Row) (*Session, error) {
	s := &Session{}
	err := row.Scan(&s.ID, &s.UserID, &s.RefreshToken, &s.JTI,
		&s.UserAgent, &s.IPAddress, &s.ExpiresAt, &s.RevokedAt, &s.ReplacedBySessionID, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return s, nil
}
