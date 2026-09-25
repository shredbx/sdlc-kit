package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shredbx/sbx-core/pkg/repository"
)

// schemaNamePattern is duplicated locally instead of imported from pkg/auth to
// keep pkg/user free of pkg/auth dependency (correct architectural direction).
var schemaNamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

// mustValidateSchema panics if schema is not a valid PostgreSQL identifier (M1).
func mustValidateSchema(schema string) {
	if !schemaNamePattern.MatchString(schema) {
		panic(fmt.Sprintf("user: invalid schema name %q — must match %s", schema, schemaNamePattern.String()))
	}
}

type postgresStore struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresStore creates a Postgres-backed UserStore.
// Panics if schema is not a valid PostgreSQL identifier (M1).
func NewPostgresStore(pool *pgxpool.Pool, schema string) UserStore {
	mustValidateSchema(schema)
	return &postgresStore{pool: pool, schema: schema}
}

func (s *postgresStore) table() string {
	return s.schema + ".users"
}

func (s *postgresStore) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	u := &User{
		ID:        uuid.New(),
		Email:     input.Email,
		FullName:  input.FullName,
		Role:      input.Role,
		Status:    UserStatusInvited,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if u.Role == "" {
		u.Role = "team-member"
	}

	query := fmt.Sprintf(`INSERT INTO %s (id, email, full_name, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, s.table())

	_, err := s.pool.Exec(ctx, query,
		u.ID, u.Email, u.FullName, u.Role, u.Status, u.CreatedAt, u.UpdatedAt)
	if err != nil {
		// M2: detect unique-violation via typed pgx error code 23505, not
		// fragile string-matching on err.Error().
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (s *postgresStore) Get(ctx context.Context, id uuid.UUID) (*User, error) {
	query := fmt.Sprintf(`SELECT id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at FROM %s WHERE id = $1`, s.table())

	return s.scanUser(s.pool.QueryRow(ctx, query, id))
}

func (s *postgresStore) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := fmt.Sprintf(`SELECT id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at FROM %s
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`, s.table())

	return s.scanUser(s.pool.QueryRow(ctx, query, email))
}

func (s *postgresStore) Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (*User, error) {
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{id}
	idx := 2

	if input.FullName != nil {
		sets = append(sets, fmt.Sprintf("full_name = $%d", idx))
		args = append(args, *input.FullName)
		idx++
	}
	if input.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", idx))
		args = append(args, *input.AvatarURL)
		idx++
	}

	query := fmt.Sprintf(`UPDATE %s SET %s WHERE id = $1
		RETURNING id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at`, s.table(), strings.Join(sets, ", "))

	return s.scanUser(s.pool.QueryRow(ctx, query, args...))
}

// Activate transitions an INVITED user to active, setting password + full name.
// Fail-closed: WHERE status='invited' guards against re-activating an active
// user (which would overwrite their existing password — H4 defense-in-depth).
// Distinguishes "user not found" from "user not invited" via a follow-up Get.
func (s *postgresStore) Activate(ctx context.Context, id uuid.UUID, input ActivateUserInput) (*User, error) {
	query := fmt.Sprintf(`UPDATE %s SET password_hash = $2, full_name = $3, status = 'active', updated_at = NOW()
		WHERE id = $1 AND status = 'invited' AND deleted_at IS NULL
		RETURNING id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at`, s.table())

	u, err := s.scanUser(s.pool.QueryRow(ctx, query, id, input.PasswordHash, input.FullName))
	if err == ErrUserNotFound {
		// 0 rows matched — either the user doesn't exist OR they're not invited.
		// Disambiguate so the caller gets the right error.
		if _, getErr := s.Get(ctx, id); getErr == nil {
			return nil, ErrUserAlreadyActive
		}
		return nil, ErrUserNotFound
	}
	return u, err
}

// ResetPassword updates the password hash for an ACTIVE user. Refuses to update
// invited/deleted users (those flow through Activate or are tombstones). Returns
// ErrUserNotActive when the WHERE guard matches no rows — fixes C1.
func (s *postgresStore) ResetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := fmt.Sprintf(`UPDATE %s SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND status = 'active' AND deleted_at IS NULL`, s.table())
	tag, err := s.pool.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return fmt.Errorf("reset password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotActive
	}
	return nil
}

func (s *postgresStore) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, s.table())
	_, err := s.pool.Exec(ctx, query, id)
	return err
}

func (s *postgresStore) UpdateRole(ctx context.Context, id uuid.UUID, role string) (*User, error) {
	query := fmt.Sprintf(`UPDATE %s SET role = $2, updated_at = NOW() WHERE id = $1
		RETURNING id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at`, s.table())

	return s.scanUser(s.pool.QueryRow(ctx, query, id, role))
}

func (s *postgresStore) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s SET deleted_at = NOW(), status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, s.table())
	_, err := s.pool.Exec(ctx, query, id)
	return err
}

// GetByEmailIncludingDeleted mirrors GetByEmail but does NOT filter
// deleted_at — used by invite/reactivate flows that need to surface "this
// email belongs to a deactivated user" instead of opaquely failing on the
// postgres unique constraint.
func (s *postgresStore) GetByEmailIncludingDeleted(ctx context.Context, email string) (*User, error) {
	query := fmt.Sprintf(`SELECT id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at FROM %s
		WHERE LOWER(email) = LOWER($1)`, s.table())
	return s.scanUser(s.pool.QueryRow(ctx, query, email))
}

// Reactivate flips a soft-deleted user back to `invited` status, clears the
// deleted_at tombstone, and resets password_hash so the user must complete the
// magic-link onboarding flow again. Returns error if the user is not currently
// in `deleted` state.
func (s *postgresStore) Reactivate(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`UPDATE %s
		SET deleted_at = NULL, status = 'invited', password_hash = NULL, updated_at = NOW()
		WHERE id = $1 AND status = 'deleted'`, s.table())
	tag, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("reactivate user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("reactivate user: not in deleted state (id=%s)", id)
	}
	return nil
}

func (s *postgresStore) List(ctx context.Context, opts repository.ListOptions) ([]User, int, error) {
	return s.list(ctx, opts, false)
}

// ListIncludingDeleted returns all users (including soft-deleted) so the admin
// UI can surface deactivated rows for reactivation.
func (s *postgresStore) ListIncludingDeleted(ctx context.Context, opts repository.ListOptions) ([]User, int, error) {
	return s.list(ctx, opts, true)
}

func (s *postgresStore) list(ctx context.Context, opts repository.ListOptions, includeDeleted bool) ([]User, int, error) {
	where := "WHERE deleted_at IS NULL"
	if includeDeleted {
		where = ""
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s %s`, s.table(), where)
	var total int
	if err := s.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	query := fmt.Sprintf(`SELECT id, email, full_name, password_hash, avatar_url, role, status,
		last_login_at, created_at, updated_at, deleted_at FROM %s
		%s ORDER BY created_at DESC LIMIT $1 OFFSET $2`, s.table(), where)

	rows, err := s.pool.Query(ctx, query, limit, opts.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		u, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, *u)
	}
	return users, total, nil
}

func (s *postgresStore) scanUser(row pgx.Row) (*User, error) {
	u := &User{}
	var passwordHash, avatarURL *string
	err := row.Scan(&u.ID, &u.Email, &u.FullName, &passwordHash, &avatarURL,
		&u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if passwordHash != nil {
		u.PasswordHash = *passwordHash
	}
	u.AvatarURL = avatarURL
	return u, nil
}

func (s *postgresStore) scanUserFromRows(rows pgx.Rows) (*User, error) {
	u := &User{}
	var passwordHash, avatarURL *string
	err := rows.Scan(&u.ID, &u.Email, &u.FullName, &passwordHash, &avatarURL,
		&u.Role, &u.Status, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, fmt.Errorf("scan user row: %w", err)
	}
	if passwordHash != nil {
		u.PasswordHash = *passwordHash
	}
	u.AvatarURL = avatarURL
	return u, nil
}
