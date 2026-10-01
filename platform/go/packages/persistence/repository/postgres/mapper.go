// Package postgres provides the PostgreSQL implementation of Repository[M].
package postgres

// Mapper defines how a domain model maps to PostgreSQL storage.
// Each entity implements this interface to declare its table structure,
// column names, and row scanning logic.
//
// The Mapper is the single source of truth for column expansion —
// it applies Type reference_semantics rules (identity, embedded, code)
// from the entity's package.yml model.
//
// Example:
//
//	type PropertyMapper struct{}
//
//	func (m PropertyMapper) TableName() string { return "bestierealestate.properties" }
//
//	func (m PropertyMapper) Columns() []string {
//	    return []string{"id", "title", "price_amount", "price_currency", ...}
//	}
//
//	func (m PropertyMapper) FieldColumn(fieldName string) string {
//	    switch fieldName {
//	    case "price_amount":  return "price_amount"
//	    case "is_published":  return "is_published"
//	    default:              return fieldName
//	    }
//	}
type Mapper[M any] interface {
	// TableName returns the fully qualified table name (schema.table).
	// Used for INSERT, UPDATE, DELETE operations.
	TableName() string

	// SelectFrom returns the FROM clause for SELECT queries.
	// For simple entities, returns the same as TableName().
	// For entities with attribute-group tables, returns the table
	// with LEFT JOIN clauses (e.g., "properties p LEFT JOIN property_locations loc ON ...").
	SelectFrom() string

	// Columns returns all column names for SELECT queries.
	// May include columns from joined tables.
	// Order must match the scan order in FromRow.
	Columns() []string

	// FieldColumn maps a logical field name (from Field[T].Name)
	// to the actual database column name. Returns the field name
	// unchanged if no mapping is needed.
	FieldColumn(fieldName string) string

	// ToRow converts a domain model to a column-value map for INSERT/UPDATE.
	// Keys are column names, values are the corresponding Go values.
	// Only includes columns for the primary table (TableName), not joined tables.
	ToRow(model M) (map[string]any, error)

	// FromRow scans a database row into a domain model.
	// The scan function is called with destination pointers matching
	// the column order from Columns().
	FromRow(scan func(dest ...any) error) (M, error)
}
