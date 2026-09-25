package calendar

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// COMPANION STORES — child collections the generated mapper cannot map
// =============================================================================
//
// An Event's Reminders (containment child) and References (independent-lifecycle
// link) are child tables, NOT columns on the events row, so the flat PostgresMapper
// cannot map them — exactly the constraint that drives the contact NoteStore /
// CategoryStore companions and Property.Amenities' Load/Save. These two stores are
// that companion pair: a batch Load (no N+1), an Add, and a Remove against
// {schema}.event_reminders and {schema}.event_references.
//
// TODO(coordination): refactor to pkg/repository.ChildStore[T] once property-units
// lands it. That generic child-store is in-flight on the property-units worktree and
// is NOT on this branch; hand-rolling these two companions here (mirroring the proven
// contact stores) avoids a merge collision with that work. When it merges, both
// stores collapse onto the generic ChildStore[Reminder] / ChildStore[Reference].

// queryQuerier is the row-set surface Load needs; rowQuerier is the single-row
// surface Add needs for its INSERT … RETURNING; execQuerier is the command-only
// surface Remove needs. Both *pgxpool.Pool and pgx.Tx satisfy all three, so Load runs
// on the pool while Add/Remove accept EITHER a pool or a tx — the handler passes a
// pgx.Tx to make a check-then-delete atomic; tests may pass the pool directly.
type queryQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type execQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// pgxQuerier is the read surface the stores hold on the pool (Query for batch Load,
// QueryRow for the Add RETURNING). *pgxpool.Pool satisfies it.
type pgxQuerier interface {
	queryQuerier
	rowQuerier
}

// Compile-time checks that the concrete pgx types satisfy the query surfaces.
var (
	_ pgxQuerier  = (*pgxpool.Pool)(nil)
	_ rowQuerier  = (pgx.Tx)(nil)
	_ execQuerier = (*pgxpool.Pool)(nil)
	_ execQuerier = (pgx.Tx)(nil)
)

// =============================================================================
// REMINDER STORE — companion for the event_reminders containment child table
// =============================================================================

// ReminderStore reads and writes an event's reminder collection against the
// {schema}.event_reminders table. Construct with NewReminderStore.
type ReminderStore struct {
	pool   pgxQuerier
	schema string
}

// NewReminderStore wires the store to a pool + schema. Pass a nil pool only in tests
// that exercise the nil-pool-safe Load without a DB.
func NewReminderStore(pool pgxQuerier, schema string) ReminderStore {
	return ReminderStore{pool: pool, schema: schema}
}

// Load fetches the reminders for one or more events in a SINGLE query (no N+1),
// returning a map of eventID → []Reminder, ordered by lead_minutes ascending (then id
// as a stable tie-breaker). An event with no reminders is simply absent from the map.
// An empty id list (or a nil pool) yields an empty map.
func (s ReminderStore) Load(ctx context.Context, eventIDs ...string) (map[string][]Reminder, error) {
	out := make(map[string][]Reminder, len(eventIDs))
	if s.pool == nil || len(eventIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, event_id, lead_minutes, channel
		   FROM `+s.schema+`.event_reminders
		  WHERE event_id = ANY($1)
		  ORDER BY event_id, lead_minutes, id`,
		eventIDs)
	if err != nil {
		return nil, fmt.Errorf("load event reminders: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rem Reminder
		var channel string
		if err := rows.Scan(&rem.ID, &rem.EventID, &rem.LeadMinutes, &channel); err != nil {
			return nil, fmt.Errorf("scan event reminder: %w", err)
		}
		rem.Channel = ReminderChannel(channel)
		out[rem.EventID] = append(out[rem.EventID], rem)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load event reminders: rows: %w", err)
	}
	return out, nil
}

// Add appends one reminder to an event and returns the created Reminder (with its
// server-assigned id), so the caller can echo it. The reminder is validated
// (non-negative lead, known channel) before insert. Pass a pgx.Tx to keep the append
// in a larger transaction.
func (s ReminderStore) Add(ctx context.Context, q rowQuerier, eventID string, leadMinutes int, channel ReminderChannel) (Reminder, error) {
	rem := Reminder{EventID: eventID, LeadMinutes: leadMinutes, Channel: channel}
	if err := rem.Validate(); err != nil {
		return Reminder{}, err
	}

	var channelStr string
	err := q.QueryRow(ctx,
		`INSERT INTO `+s.schema+`.event_reminders (event_id, lead_minutes, channel)
		 VALUES ($1, $2, $3)
		 RETURNING id, event_id, lead_minutes, channel`,
		eventID, leadMinutes, string(channel)).Scan(&rem.ID, &rem.EventID, &rem.LeadMinutes, &channelStr)
	if err != nil {
		return Reminder{}, fmt.Errorf("add event reminder: %w", err)
	}
	rem.Channel = ReminderChannel(channelStr)
	return rem, nil
}

// Remove deletes one reminder from an event by id, scoped to the event so a wrong
// (id, event) pair is a no-op. Returns whether a row was deleted. Pass a pgx.Tx to
// keep the delete in a larger transaction.
func (s ReminderStore) Remove(ctx context.Context, q execQuerier, eventID, reminderID string) (bool, error) {
	tag, err := q.Exec(ctx,
		`DELETE FROM `+s.schema+`.event_reminders WHERE id = $1 AND event_id = $2`,
		reminderID, eventID)
	if err != nil {
		return false, fmt.Errorf("remove event reminder %s: %w", reminderID, err)
	}
	return tag.RowsAffected() > 0, nil
}

// =============================================================================
// REFERENCE STORE — companion for the event_references link table
// =============================================================================

// ReferenceStore reads and writes an event's reference collection against the
// {schema}.event_references table. Construct with NewReferenceStore.
//
// References are an INDEPENDENT-lifecycle association (Decision #0019 three-map model):
// deleting a link row never touches the referenced target (ref_id is a plain value,
// not an FK), and the cached label + subtitle survive the target's delete (D11).
type ReferenceStore struct {
	pool   pgxQuerier
	schema string
}

// NewReferenceStore wires the store to a pool + schema. Pass a nil pool only in tests
// that exercise the nil-pool-safe Load without a DB.
func NewReferenceStore(pool pgxQuerier, schema string) ReferenceStore {
	return ReferenceStore{pool: pool, schema: schema}
}

// Load fetches the references for one or more events in a SINGLE query (no N+1),
// returning a map of eventID → []Reference, ordered by id (stable insert-ish order).
// An event with no references is simply absent from the map. An empty id list (or a
// nil pool) yields an empty map. relation/as_type/subtitle are nullable columns,
// scanned through *string and coalesced to "" so the wire shape (RefChip) omits them
// when blank.
func (s ReferenceStore) Load(ctx context.Context, eventIDs ...string) (map[string][]Reference, error) {
	out := make(map[string][]Reference, len(eventIDs))
	if s.pool == nil || len(eventIDs) == 0 {
		return out, nil
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, event_id, ref_type, ref_id, relation, as_type, label, subtitle
		   FROM `+s.schema+`.event_references
		  WHERE event_id = ANY($1)
		  ORDER BY event_id, id`,
		eventIDs)
	if err != nil {
		return nil, fmt.Errorf("load event references: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			ref               Reference
			eventID           string
			relation, asType  *string
			subtitle          *string
		)
		if err := rows.Scan(&ref.ID, &eventID, &ref.RefType, &ref.RefID, &relation, &asType, &ref.Label, &subtitle); err != nil {
			return nil, fmt.Errorf("scan event reference: %w", err)
		}
		ref.Relation = deref(relation)
		ref.AsType = deref(asType)
		ref.Subtitle = deref(subtitle)
		out[eventID] = append(out[eventID], ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load event references: rows: %w", err)
	}
	return out, nil
}

// Add links one reference to an event and returns the created Reference (with its
// server-assigned id), so the caller can echo it in the mutation result. The reference
// is validated (ref_type, ref_id, label required — the cached label is mandatory so the
// chip survives the target's delete) before insert. Blank relation/as_type/subtitle are
// stored as SQL NULL. Pass a pgx.Tx to keep the add in a larger transaction.
func (s ReferenceStore) Add(ctx context.Context, q rowQuerier, eventID string, ref Reference) (Reference, error) {
	ref.RefType = strings.TrimSpace(ref.RefType)
	ref.RefID = strings.TrimSpace(ref.RefID)
	ref.Label = strings.TrimSpace(ref.Label)
	if err := ref.Validate(); err != nil {
		return Reference{}, err
	}

	var relation, asType, subtitle *string
	err := q.QueryRow(ctx,
		`INSERT INTO `+s.schema+`.event_references (event_id, ref_type, ref_id, relation, as_type, label, subtitle)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, ref_type, ref_id, relation, as_type, label, subtitle`,
		eventID, ref.RefType, ref.RefID, nilIfBlank(ref.Relation), nilIfBlank(ref.AsType), ref.Label, nilIfBlank(ref.Subtitle),
	).Scan(&ref.ID, &ref.RefType, &ref.RefID, &relation, &asType, &ref.Label, &subtitle)
	if err != nil {
		return Reference{}, fmt.Errorf("add event reference: %w", err)
	}
	ref.Relation = deref(relation)
	ref.AsType = deref(asType)
	ref.Subtitle = deref(subtitle)
	return ref, nil
}

// Remove deletes one reference link from an event by id, scoped to the event so a
// wrong (id, event) pair is a no-op. Deleting the link NEVER touches the referenced
// target (independent lifecycle). Returns whether a row was deleted. Pass a pgx.Tx to
// keep the delete in a larger transaction.
func (s ReferenceStore) Remove(ctx context.Context, q execQuerier, eventID, referenceID string) (bool, error) {
	tag, err := q.Exec(ctx,
		`DELETE FROM `+s.schema+`.event_references WHERE id = $1 AND event_id = $2`,
		referenceID, eventID)
	if err != nil {
		return false, fmt.Errorf("remove event reference %s: %w", referenceID, err)
	}
	return tag.RowsAffected() > 0, nil
}

// deref returns the pointed-to string or "" for a nil pointer — collapses a nullable
// VARCHAR column into the omitempty-friendly value the Reference wire shape wants.
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// nilIfBlank maps an empty/whitespace string to a nil *string so a blank optional
// column is stored as SQL NULL rather than "".
func nilIfBlank(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}
