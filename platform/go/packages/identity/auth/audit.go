package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Common audit action codes. Add new codes as services grow; keep names
// stable for SIEM correlation queries.
const (
	AuditActionLoginSuccess       = "auth.login_success"
	AuditActionLoginFailure       = "auth.login_failure"
	AuditActionLogout             = "auth.logout"
	AuditActionTokenRotated       = "auth.token_rotated"
	AuditActionTokenReuseDetected = "auth.token_reuse_detected"
	// AuditActionRotationRevokeFailed records that a refresh-token rotation minted
	// a new pair but FAILED to revoke the old (parent) session (RevokeForRotation
	// errored). This is a fail-open degradation: the old refresh token stays valid
	// until its TTL and single-use reuse-detection is disarmed for that lineage.
	// Emitted so the failure is observable instead of silently swallowed. Whether
	// to instead fail closed (refuse the new pair) is a separate config/owner call.
	AuditActionRotationRevokeFailed = "auth.rotation_revoke_failed"
	// AuditActionRefreshGraceReplay records a BENIGN refresh-token replay inside
	// the rotation grace window (AuthConfig.RefreshReuseLeeway): a fresh token pair
	// was issued and the family was NOT revoked. Kept distinct from
	// token_reuse_detected so operators can tell normal concurrency (multi-tab,
	// retries, lost Set-Cookie, deploy overlap) apart from real theft. Low severity.
	AuditActionRefreshGraceReplay = "auth.refresh_grace_replay"
	// AuditActionRefreshInactiveUser records that a refresh was REJECTED because the
	// user account is no longer active (deactivated / soft-deleted). Refresh is the
	// liveness checkpoint (Auth0 / RFC 9700) — without this recheck a deactivated
	// user keeps minting fresh access tokens until every session's refresh token
	// expires, because middleware/RBAC trust the JWT claims alone.
	AuditActionRefreshInactiveUser = "auth.refresh_inactive_user"
	AuditActionSessionRevoked      = "auth.session_revoked"
	AuditActionMagicLinkGenerated  = "auth.magic_link_generated"
	AuditActionMagicLinkConsumed   = "auth.magic_link_consumed"
	AuditActionMagicLinkWrongPurp  = "auth.magic_link_wrong_purpose"
	AuditActionPasswordResetReq    = "auth.password_reset_requested"
	AuditActionPasswordResetAppr   = "auth.password_reset_approved"
	AuditActionPasswordResetRej    = "auth.password_reset_rejected"
	AuditActionRBACUnknownRole     = "rbac.unknown_role"
	AuditActionRBACPermissionDeny  = "rbac.permission_denied"

	// User management — admin actions on other users. target_type=user,
	// target_id=affected user. actor_id=the admin performing the action.
	AuditActionUserInvited             = "user.invited"
	AuditActionUserReinvited           = "user.reinvited"
	AuditActionUserRoleChanged         = "user.role_changed"
	AuditActionUserDeactivated         = "user.deactivated"
	AuditActionUserReactivated         = "user.reactivated"
	AuditActionUserPasswordResetIssued = "user.password_reset_issued"
)

// AuditTargetUser is the target_type for user-management actions. Use this
// constant rather than literal strings so callers stay aligned with the
// list endpoint's filter values.
const AuditTargetUser = "user"

// AuditEvent records a single structured audit row.
type AuditEvent struct {
	ID         uuid.UUID      `json:"id"`
	ActorID    *uuid.UUID     `json:"actor_id,omitempty"` // nil for anonymous (failed login etc.)
	Action     string         `json:"action"`
	TargetType string         `json:"target_type,omitempty"`
	TargetID   *uuid.UUID     `json:"target_id,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AuditRepository persists audit events in Postgres.
type AuditRepository interface {
	Append(ctx context.Context, event *AuditEvent) error
	ListEvents(ctx context.Context, opts AuditListOptions) ([]AuditEvent, int, error)
}

// AuditListOptions filters and pages the audit log. Zero limit means "no
// limit was requested" — the repo clamps to 50 by default and 500 max so a
// runaway query can't pull millions of rows into memory.
type AuditListOptions struct {
	Action     string     // exact match (e.g. AuditActionUserRoleChanged); empty = all
	ActorID    *uuid.UUID // filter by who performed the action
	TargetType string     // exact match (e.g. AuditTargetUser); empty = all
	TargetID   *uuid.UUID // filter by the target of the action
	Limit      int        // 0 → 50, max 500
	Offset     int        // page offset

	Query string // case-insensitive substring search across action/target_type/ip_address/actor_id/target_id; empty = no search
	Sort  string // whitelisted column key (see auditSortColumns); empty/unknown → "created_at"
	Desc  bool   // sort direction; resolves to DESC when (Desc || Sort==""), else ASC
}

// auditSortColumns is the ORDER BY whitelist. ORDER BY cannot be parameterized,
// so the SQL column is looked up by an accepted key here — user text never
// reaches the query string. An unknown/empty key falls back to "created_at".
var auditSortColumns = map[string]string{
	"created_at":  "created_at",
	"action":      "action",
	"actor_id":    "actor_id",
	"target_type": "target_type",
	"ip_address":  "ip_address",
}

// escapeLike escapes the three LIKE/ILIKE wildcard metacharacters (\ % _) with
// a leading backslash so a user-supplied search term is treated as a literal
// substring. The default ESCAPE '\' applies, so the escaped term needs no extra
// ESCAPE clause. The backslash itself is escaped first to avoid double-escaping.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

// buildAuditListSQL builds the count and list queries for the audit log. It is
// pure (no DB, no clamping) so the WHERE/ORDER BY construction — in particular
// the ORDER BY injection boundary and the ILIKE search group — is unit-testable
// without a database.
//
// Returns:
//   - listSQL  — full SELECT with WHERE + ORDER BY + LIMIT/OFFSET. The LIMIT and
//     OFFSET placeholders are the two args AFTER the filter/search args, so the
//     caller runs it with append(args, limit, offset).
//   - countSQL — SELECT COUNT(*) sharing the SAME WHERE (no LIMIT/OFFSET). The
//     caller runs it with exactly args (the filter/search args).
//   - args     — the parameterized filter + search values, in placeholder order.
func buildAuditListSQL(schema string, opts AuditListOptions) (listSQL, countSQL string, args []any) {
	args = make([]any, 0, 5)
	where := ""
	add := func(clause string, value any) {
		args = append(args, value)
		if where == "" {
			where = " WHERE " + fmt.Sprintf(clause, len(args))
		} else {
			where += " AND " + fmt.Sprintf(clause, len(args))
		}
	}
	if opts.Action != "" {
		add("action = $%d", opts.Action)
	}
	if opts.ActorID != nil {
		add("actor_id = $%d", *opts.ActorID)
	}
	if opts.TargetType != "" {
		add("target_type = $%d", opts.TargetType)
	}
	if opts.TargetID != nil {
		add("target_id = $%d", *opts.TargetID)
	}
	if opts.Query != "" {
		// One parameterized arg referenced 5× — the value is never concatenated.
		add("(action ILIKE $%[1]d OR target_type ILIKE $%[1]d OR ip_address ILIKE $%[1]d"+
			" OR actor_id::text ILIKE $%[1]d OR target_id::text ILIKE $%[1]d)",
			"%"+escapeLike(opts.Query)+"%")
	}

	// ORDER BY — injection boundary. The column is a literal from the whitelist
	// and the direction is a literal derived from a bool; raw opts.Sort text is
	// never written into the SQL string.
	col := auditSortColumns[opts.Sort]
	if col == "" {
		col = "created_at"
	}
	dir := "ASC"
	if opts.Desc || opts.Sort == "" {
		dir = "DESC"
	}
	orderBy := fmt.Sprintf("ORDER BY %s %s", col, dir)
	if col != "created_at" {
		orderBy += ", created_at DESC"
	}
	orderBy += ", id DESC"

	countSQL = fmt.Sprintf(`SELECT COUNT(*) FROM %s.audit_events%s`, schema, where)
	listSQL = fmt.Sprintf(`SELECT id, actor_id, action, target_type, target_id,
			ip_address, user_agent, payload, created_at
		FROM %s.audit_events%s
		%s
		LIMIT $%d OFFSET $%d`, schema, where, orderBy, len(args)+1, len(args)+2)
	return listSQL, countSQL, args
}

type postgresAuditRepo struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresAuditRepo creates a Postgres-backed AuditRepository.
// Panics if schema is not a valid PostgreSQL identifier (M1).
func NewPostgresAuditRepo(pool *pgxpool.Pool, schema string) AuditRepository {
	MustValidateSchema(schema)
	return &postgresAuditRepo{pool: pool, schema: schema}
}

func (r *postgresAuditRepo) ListEvents(ctx context.Context, opts AuditListOptions) ([]AuditEvent, int, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	// Build WHERE + ORDER BY + LIMIT/OFFSET via the pure helper. The count query
	// shares the same WHERE (filter + search args); the list query appends the
	// clamped limit/offset as the final two placeholders.
	listQuery, countQuery, args := buildAuditListSQL(r.schema, opts)

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("audit count: %w", err)
	}

	listArgs := append(append([]any{}, args...), limit, offset)

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit list: %w", err)
	}
	defer rows.Close()

	events := make([]AuditEvent, 0, limit)
	for rows.Next() {
		var (
			e          AuditEvent
			actorID    *uuid.UUID
			targetType *string
			targetID   *uuid.UUID
			ipAddr     *string
			userAgent  *string
			payload    []byte
		)
		if err := rows.Scan(&e.ID, &actorID, &e.Action, &targetType, &targetID,
			&ipAddr, &userAgent, &payload, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("audit scan: %w", err)
		}
		e.ActorID = actorID
		if targetType != nil {
			e.TargetType = *targetType
		}
		e.TargetID = targetID
		if ipAddr != nil {
			e.IPAddress = *ipAddr
		}
		if userAgent != nil {
			e.UserAgent = *userAgent
		}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &e.Payload); err != nil {
				return nil, 0, fmt.Errorf("audit payload unmarshal: %w", err)
			}
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("audit rows: %w", err)
	}
	return events, total, nil
}

func (r *postgresAuditRepo) Append(ctx context.Context, e *AuditEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	payload, err := json.Marshal(e.Payload)
	if err != nil {
		return fmt.Errorf("audit marshal payload: %w", err)
	}
	if len(payload) == 0 || string(payload) == "null" {
		payload = []byte("{}")
	}
	query := fmt.Sprintf(`INSERT INTO %s.audit_events
		(id, actor_id, action, target_type, target_id, ip_address, user_agent, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, r.schema)
	_, err = r.pool.Exec(ctx, query,
		e.ID, e.ActorID, e.Action, nullIfEmpty(e.TargetType), e.TargetID,
		nullIfEmpty(e.IPAddress), nullIfEmpty(e.UserAgent), payload, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("audit append: %w", err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
