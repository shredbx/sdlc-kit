package dictionary_test

import (
	"testing"

	"github.com/shredbx/sbx-core/pkg/dictionary"
	"github.com/stretchr/testify/assert"
)

// TC-001 — Registry resolves a known dictionary name to its Spec.
// Maps to SC1.1 (admin discovers manageable dictionaries from Registry).
func TestRegistry_ResolvesPropertyTypes(t *testing.T) {
	reg := dictionary.Registry{
		"property_types": dictionary.Spec{
			Name:          "property_types",
			Table:         "property_types",
			Label:         "Property Types",
			PublicExposed: true,
			FKRefs: []dictionary.FKRef{
				{Table: "properties", Column: "property_type", Nullable: true},
			},
		},
	}

	spec, ok := reg.Lookup("property_types")
	assert.True(t, ok, "Registry must resolve known dict name")
	assert.Equal(t, "property_types", spec.Table)
	assert.Equal(t, "Property Types", spec.Label)
	assert.Equal(t, 1, len(spec.FKRefs))
	assert.Equal(t, "properties", spec.FKRefs[0].Table)
	assert.True(t, spec.FKRefs[0].Nullable)
	assert.True(t, spec.PublicExposed)
}

// TC-002 — Registry returns ok=false for unknown name.
// Maps to T1 mitigation (SQL injection via {name} path parameter) — gate must reject before SQL.
func TestRegistry_UnknownDictReturnsZero(t *testing.T) {
	reg := dictionary.Registry{
		"property_types": {Name: "property_types"},
	}

	_, ok := reg.Lookup("evil; DROP TABLE properties;--")
	assert.False(t, ok, "unknown dict name must return ok=false")

	_, ok = reg.Lookup("")
	assert.False(t, ok, "empty name must return ok=false")

	_, ok = reg.Lookup("property_typess") // typo / tampered
	assert.False(t, ok, "near-miss name must return ok=false")
}
