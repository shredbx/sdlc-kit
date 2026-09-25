package dictionary

// Registry maps a dictionary name (URL slug + table identifier) to its Spec.
// Consumers (BR API handler) construct a Registry in their `internal/registry`
// package with project-specific FKRefs. The shared pkg/dictionary doesn't know
// which project tables reference which dictionaries.
//
// Lookup is the only operation needed: the gate before any SQL is "does this
// name resolve to a known Spec?" If not, the caller MUST return 404 / refuse
// to proceed. This is the single source of truth that kills duplicate
// allowlists across handlers (per task 2605-058 motivation).
type Registry map[string]Spec

// Lookup returns the Spec for name and ok=true; or zero Spec + false if the
// name is not registered. Callers MUST check ok before using the result.
func (r Registry) Lookup(name string) (Spec, bool) {
	if name == "" {
		return Spec{}, false
	}
	spec, ok := r[name]
	return spec, ok
}

// Names returns the registered dictionary names in declaration order is NOT
// guaranteed — callers needing stable order should sort. Used by the public
// bundle resolver and the Content Management landing page.
func (r Registry) Names() []string {
	out := make([]string, 0, len(r))
	for name := range r {
		out = append(out, name)
	}
	return out
}

// PublicNames returns only the names with PublicExposed=true. Drives the
// public bundle endpoint allowlist.
func (r Registry) PublicNames() []string {
	out := make([]string, 0, len(r))
	for name, spec := range r {
		if spec.PublicExposed {
			out = append(out, name)
		}
	}
	return out
}

// Spec describes one dictionary's storage and consumer contracts.
//
// Storage:
//   - Name: URL slug + lookup key (e.g., "property_types")
//   - Table: schema-qualified table name in PostgreSQL (PostgresBackend prepends schema)
//   - Label / Description: presentation
//
// Consumers:
//   - FKRefs: every table+column that has FOREIGN KEY into this dictionary.
//     Drives the ConsumersCount + CascadeReplaceAndDelete operations.
//
// Exposure:
//   - PublicExposed: included in /api/content/public-bundle response if true.
//     Admin-only dictionaries set this to false.
type Spec struct {
	Name          string
	Table         string
	Label         string
	Description   string
	FKRefs        []FKRef
	PublicExposed bool
}

// FKRef declares a single consumer reference: a (table, column) pair in the
// owning project that has a foreign key into the dictionary's code column.
// Nullable controls whether CascadeReplaceAndDelete accepts a NULL replacement.
type FKRef struct {
	Table    string
	Column   string
	Nullable bool
}

// UpdatePatch carries the optional fields for Update. nil pointer = field
// unchanged. All zero / nil = no-op.
type UpdatePatch struct {
	Label       *string
	Description *string
	Group       *string
	SortOrder   *int
	Icon        *string
	Deprecated  *bool
}

// ConsumerCount is one row of the ConsumersCount breakdown: how many rows in
// (table, column) currently reference the dictionary code being inspected.
//
// JSON tags are explicit because this struct travels over the wire in
// /api/manage/content/dictionaries/{name}/values/{code}/consumers — the Svelte
// editor depends on the lower-case shape via ConsumerCheck.by_ref.
type ConsumerCount struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	Count  int    `json:"count"`
}
