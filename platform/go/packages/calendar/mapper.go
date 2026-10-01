package calendar

import (
	"fmt"
	"time"

	"github.com/shredbx/sbx-core/pkg/repository"
	"github.com/shredbx/sbx-core/pkg/repository/postgres"
)

// =============================================================================
// MD LAYER — PostgresMapper[Event] for the flat events row
// =============================================================================
//
// HAND-WRITTEN (not `sbx generate mapper`): the entity.yml deliberately ships NO
// `platforms.go.postgres` binding block (Phase 3a was PD-only), so there is no codegen
// source to project from — this mapper is the Phase 3b SI authoring of it. It mirrors
// the generated property/inquiry mapper shape exactly so it plugs straight into the
// generic postgres.NewPostgresStore[Event]: a flat single-table mapper aliased "e".
//
// The Reminders + References child collections are NOT columns on the events row, so —
// exactly like Property.Amenities and a contact's notes/categories — they are loaded
// by the companion ReminderStore / ReferenceStore (store.go), never by this mapper.
//
// COLUMN-NAME NOTE: the PD field End maps to the column end_at and Start to start_at
// because `end` is a reserved word in SQL; `start` is not reserved but is paired with
// end_at for symmetry. FieldColumn translates the logical names ("start"/"end") the
// typed Fields descriptors carry onto those columns for WHERE/ORDER BY.

// Compile-time check: PostgresMapper implements postgres.Mapper[Event].
var _ postgres.Mapper[Event] = PostgresMapper{}

// Fields exposes typed, compile-time-safe field descriptors for List filtering +
// sorting. The HTTP adapter builds the time-window range (Start.Gte / End.Lte),
// the owner predicate (OwnerID.Eq), and the visibility predicate (Visibility.Eq)
// from these. Each Name is the LOGICAL key FieldColumn understands (start/end map
// to the start_at/end_at columns).
var Fields = struct {
	ID         repository.Field[string]
	Start      repository.Field[time.Time]
	End        repository.Field[time.Time]
	OwnerID    repository.Field[string]
	Visibility repository.Field[string]
	Status     repository.Field[string]
	EventType  repository.Field[string]
	CreatedAt  repository.Field[string]
	DeletedAt  repository.Field[string]
}{
	ID:         repository.Field[string]{Name: "id"},
	Start:      repository.Field[time.Time]{Name: "start"},
	End:        repository.Field[time.Time]{Name: "end"},
	OwnerID:    repository.Field[string]{Name: "owner_id"},
	Visibility: repository.Field[string]{Name: "visibility"},
	Status:     repository.Field[string]{Name: "status"},
	EventType:  repository.Field[string]{Name: "event_type"},
	CreatedAt:  repository.Field[string]{Name: "created_at"},
	DeletedAt:  repository.Field[string]{Name: "deleted_at"},
}

// PostgresMapper maps Event to the {schema}.events table for any schema. Construct
// with NewPostgresMapper("<schema>").
type PostgresMapper struct {
	schema string
}

// NewPostgresMapper creates a mapper scoped to the given PostgreSQL schema.
func NewPostgresMapper(schema string) PostgresMapper {
	return PostgresMapper{schema: schema}
}

// TableName returns the fully qualified primary table for INSERT/UPDATE/DELETE.
func (m PostgresMapper) TableName() string {
	return m.schema + ".events"
}

// SelectFrom aliases the primary table — Columns() prefixes every field with the
// "e" alias, so the FROM clause must publish it.
func (m PostgresMapper) SelectFrom() string {
	return m.TableName() + " e"
}

// Columns returns the SELECT column list. Order MUST exactly match FromRow scan.
func (m PostgresMapper) Columns() []string {
	return []string{
		"e.id",
		"e.start_at",
		"e.end_at",
		"e.all_day",
		"e.title",
		"e.notes",
		"e.event_type",
		"e.status",
		"e.owner_id",
		"e.visibility",
		"e.created_at",
		"e.updated_at",
		"e.deleted_at",
	}
}

// FieldColumn maps logical field names to aliased columns for WHERE/ORDER BY. The
// "start"/"end" logical keys translate to the start_at/end_at columns; everything
// else is its own snake-case column under the "e" alias.
func (m PostgresMapper) FieldColumn(field string) string {
	switch field {
	case "id":
		return "e.id"
	case "start":
		return "e.start_at"
	case "end":
		return "e.end_at"
	case "all_day":
		return "e.all_day"
	case "event_type":
		return "e.event_type"
	case "status":
		return "e.status"
	case "owner_id":
		return "e.owner_id"
	case "visibility":
		return "e.visibility"
	case "created_at":
		return "e.created_at"
	case "updated_at":
		return "e.updated_at"
	case "deleted_at":
		return "e.deleted_at"
	default:
		return field
	}
}

// ToRow converts an Event to a column→value map for INSERT/UPDATE. Discriminators are
// serialized to their STRING form (so pgx binds a VARCHAR, never the named Go type).
// deleted_at is omitted (DB default NULL; soft-delete is a separate UPDATE), mirroring
// the inquiry mapper. The child collections (Reminders/References) are not columns and
// are written by the companion stores, never here.
func (m PostgresMapper) ToRow(e Event) (map[string]any, error) {
	return map[string]any{
		"id":         e.ID,
		"start_at":   e.Start,
		"end_at":     e.End,
		"all_day":    e.AllDay,
		"title":      e.Title,
		"notes":      e.Notes,
		"event_type": string(e.EventType),
		"status":     string(e.Status),
		"owner_id":   e.OwnerID,
		"visibility": string(e.Visibility),
		"created_at": e.CreatedAt,
		"updated_at": e.UpdatedAt,
	}, nil
}

// FromRow scans a row into an Event. Column order MUST exactly match Columns(). The
// VARCHAR discriminators scan into the named types directly (string ~ EventType etc.,
// so pgx assigns the underlying string). Reminders/References stay nil here — the
// caller hydrates them via the companion stores when needed.
func (m PostgresMapper) FromRow(scan func(dest ...any) error) (Event, error) {
	var e Event
	var eventType, status, visibility string
	err := scan(
		&e.ID,
		&e.Start,
		&e.End,
		&e.AllDay,
		&e.Title,
		&e.Notes,
		&eventType,
		&status,
		&e.OwnerID,
		&visibility,
		&e.CreatedAt,
		&e.UpdatedAt,
		&e.DeletedAt,
	)
	if err != nil {
		return e, fmt.Errorf("scan event: %w", err)
	}
	e.EventType = EventType(eventType)
	e.Status = EventStatus(status)
	e.Visibility = Visibility(visibility)
	return e, nil
}
