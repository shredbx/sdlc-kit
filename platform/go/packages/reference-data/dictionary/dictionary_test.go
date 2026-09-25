package dictionary

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileBackend_List(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "transaction-type")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	entityYml := `type: entity
name: transaction-type
type_category: dictionary
purpose: How a property is available
values:
  - code: sale
    label: Sale
    sort_order: 1
  - code: rent
    label: Rent
    sort_order: 2
  - code: lease
    label: Lease
    sort_order: 3
  - code: sale-lease
    label: Sale & Lease
    sort_order: 4
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(entityYml), 0644))

	backend := NewFileBackend(dir)
	ctx := context.Background()

	entries, err := backend.List(ctx, "transaction-type")
	require.NoError(t, err)
	assert.Len(t, entries, 4)
	assert.Equal(t, "sale", entries[0].Code)
	assert.Equal(t, "Sale", entries[0].Label)
	assert.Equal(t, "sale-lease", entries[3].Code)
	assert.Equal(t, "Sale & Lease", entries[3].Label)
}

func TestFileBackend_Get(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "land-size-unit")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	entityYml := `type: entity
name: land-size-unit
type_category: dictionary
purpose: Land measurement units
values:
  - code: sqm
    label: Square Meters
    description: Metric square meters
    sort_order: 1
  - code: rai
    label: Rai
    description: Thai unit equal to 1600 sqm
    sort_order: 2
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(entityYml), 0644))

	backend := NewFileBackend(dir)
	ctx := context.Background()

	entry, err := backend.Get(ctx, "land-size-unit", "rai")
	require.NoError(t, err)
	assert.Equal(t, "rai", entry.Code)
	assert.Equal(t, "Rai", entry.Label)
	assert.Equal(t, "Thai unit equal to 1600 sqm", entry.Description)

	_, err = backend.Get(ctx, "land-size-unit", "nonexistent")
	assert.Error(t, err)
}

func TestFileBackend_NotDictionary(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "property")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	entityYml := `type: entity
name: property
type_category: document
purpose: A property listing
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(entityYml), 0644))

	backend := NewFileBackend(dir)
	ctx := context.Background()

	_, err := backend.List(ctx, "property")
	assert.Error(t, err)
}

func TestFileBackend_ListAll(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{"property-type", "transaction-type"} {
		entityDir := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(entityDir, 0755))
		yml := `type: entity
name: ` + name + `
type_category: dictionary
purpose: Test
values:
  - code: a
    label: A
    sort_order: 1
`
		require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(yml), 0644))
	}

	propDir := filepath.Join(dir, "property")
	require.NoError(t, os.MkdirAll(propDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(propDir, "entity.yml"), []byte(`type: entity
name: property
type_category: document
purpose: A property
`), 0644))

	backend := NewFileBackend(dir)
	ctx := context.Background()

	dicts, err := backend.ListAll(ctx)
	require.NoError(t, err)
	assert.Len(t, dicts, 2)
}

func TestPostgresBackend_GenerateCreateSQL(t *testing.T) {
	backend := &PostgresBackend{schema: "bestierealestate"}

	sql := backend.GenerateCreateSQL("property-type")
	assert.Contains(t, sql, "CREATE SCHEMA IF NOT EXISTS bestierealestate")
	assert.Contains(t, sql, "CREATE TABLE IF NOT EXISTS bestierealestate.property_types")
	assert.Contains(t, sql, "code")
	assert.Contains(t, sql, "VARCHAR(50)")
	assert.Contains(t, sql, "PRIMARY KEY")
	assert.Contains(t, sql, "label")
	assert.Contains(t, sql, "NOT NULL")
	assert.Contains(t, sql, "sort_order")
	assert.Contains(t, sql, "SMALLINT")
}

func TestPostgresBackend_GenerateCreateSQL_NewColumns(t *testing.T) {
	backend := &PostgresBackend{schema: "bestierealestate"}
	sql := backend.GenerateCreateSQL("property-type")

	assert.Contains(t, sql, "color")
	assert.Contains(t, sql, "VARCHAR(7)")
	assert.Contains(t, sql, "translations")
	assert.Contains(t, sql, "JSONB")
	assert.Contains(t, sql, "image_url")
	assert.Contains(t, sql, "video_url")
	assert.Contains(t, sql, "TEXT")
}

func TestPostgresBackend_GenerateSeedSQL(t *testing.T) {
	backend := &PostgresBackend{schema: "bestierealestate"}

	entries := []Entry{
		{Code: "sale", Label: "Sale", Description: "Property available for purchase", SortOrder: 1},
		{Code: "rent", Label: "Rent", SortOrder: 2},
	}

	sql := backend.GenerateSeedSQL("transaction-type", entries)
	assert.Contains(t, sql, "INSERT INTO bestierealestate.transaction_types")
	assert.Contains(t, sql, `code, label, description, icon, color, "group", sort_order, translations, image_url, video_url, deprecated`)
	assert.Contains(t, sql, "'sale'")
	assert.Contains(t, sql, "ON CONFLICT (code) DO NOTHING")
}

func TestPostgresBackend_GenerateSeedSQL_WithTranslations(t *testing.T) {
	backend := &PostgresBackend{schema: "test"}
	entries := []Entry{
		{
			Code:         "villa",
			Label:        "Villa",
			Icon:         "Home",
			Color:        "#C8A851",
			SortOrder:    1,
			Translations: map[string]string{"en": "Villa", "th": "วิลล่า"},
			ImageURL:     "https://cdn.example.com/villa.jpg",
		},
	}
	sql := backend.GenerateSeedSQL("property-type", entries)
	assert.Contains(t, sql, "Villa")
	assert.Contains(t, sql, "::jsonb")
	assert.Contains(t, sql, "'#C8A851'")
	assert.Contains(t, sql, "cdn.example.com")
}

func TestPostgresBackend_TableName(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		input  string
		want   string
	}{
		{"with schema", "bestierealestate", "property-type", "bestierealestate.property_types"},
		{"no schema", "", "property-type", "property_types"},
		{"multi-hyphen", "bestays", "land-size-unit", "bestays.land_size_units"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &PostgresBackend{schema: tt.schema}
			assert.Equal(t, tt.want, b.tableName(tt.input))
		})
	}
}

func TestStore_List(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "title-deed")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	entityYml := `type: entity
name: title-deed
type_category: dictionary
purpose: Thai land title deeds
values:
  - code: chanote
    label: Chanote
    sort_order: 1
  - code: leasehold
    label: Leasehold
    sort_order: 5
  - code: nor-sor-3-gor
    label: Nor Sor 3 Gor
    sort_order: 2
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(entityYml), 0644))

	store := NewStore(NewFileBackend(dir))
	ctx := context.Background()

	entries, err := store.List(ctx, "title-deed")
	require.NoError(t, err)
	assert.Len(t, entries, 3)
	// Should be sorted by sort_order
	assert.Equal(t, "chanote", entries[0].Code)
	assert.Equal(t, "nor-sor-3-gor", entries[1].Code)
	assert.Equal(t, "leasehold", entries[2].Code)
}

func TestEntry_NewFields_Compile(t *testing.T) {
	e := Entry{
		Code:         "villa",
		Label:        "Villa",
		Color:        "#C8A851",
		Translations: map[string]string{"en": "Villa", "th": "วิลล่า"},
		ImageURL:     "https://cdn.example.com/villa.jpg",
		VideoURL:     "https://cdn.example.com/villa.mp4",
	}
	if e.Color == "" {
		t.Error("Color field missing from Entry")
	}
	if e.Translations["th"] != "วิลล่า" {
		t.Error("Translations field missing from Entry")
	}
	if e.ImageURL == "" {
		t.Error("ImageURL field missing from Entry")
	}
	if e.VideoURL == "" {
		t.Error("VideoURL field missing from Entry")
	}
}

func TestDictEnum_CatalogueMetadata(t *testing.T) {
	d := DictEnum{
		Name: "interior-amenities",
		Metadata: &CatalogueMetadata{
			Category:      "interior",
			Regions:       []string{"global", "thai"},
			PropertyTypes: []string{"villa", "house"},
		},
	}
	if d.Metadata == nil {
		t.Error("Metadata field missing from DictEnum")
	}
	if d.Metadata.Category != "interior" {
		t.Errorf("got %q, want %q", d.Metadata.Category, "interior")
	}
	if len(d.Metadata.Regions) != 2 {
		t.Errorf("got %d regions, want 2", len(d.Metadata.Regions))
	}
}

func TestFileBackend_NewEntryFields_FromYAML(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "property-type")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	yml := `type: entity
name: property-type
type_category: dictionary
purpose: Property types
values:
  - code: villa
    label: Villa
    icon: Home
    color: "#C8A851"
    sort_order: 1
    translations:
      en: Villa
      th: วิลล่า
    image_url: https://cdn.example.com/villa.jpg
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(yml), 0644))

	backend := NewFileBackend(dir)
	ctx := context.Background()

	entries, err := backend.List(ctx, "property-type")
	require.NoError(t, err)
	require.Len(t, entries, 1)

	e := entries[0]
	assert.Equal(t, "Home", e.Icon)
	assert.Equal(t, "#C8A851", e.Color)
	assert.Equal(t, "Villa", e.Translations["en"])
	assert.Equal(t, "วิลล่า", e.Translations["th"])
	assert.Equal(t, "https://cdn.example.com/villa.jpg", e.ImageURL)
}

func TestFileBackend_ListAll_WithMetadata(t *testing.T) {
	dir := t.TempDir()
	entityDir := filepath.Join(dir, "interior-amenities")
	require.NoError(t, os.MkdirAll(entityDir, 0755))

	yml := `type: entity
name: interior-amenities
type_category: dictionary
purpose: Interior amenities
metadata:
  category: interior
  regions: [global, thai]
  property_types: [villa, house]
values:
  - code: air_conditioning
    label: Air Conditioning
    icon: Snowflake
    sort_order: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(entityDir, "entity.yml"), []byte(yml), 0644))

	backend := NewFileBackend(dir)
	ctx := context.Background()

	dicts, err := backend.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, dicts, 1)
	require.NotNil(t, dicts[0].Metadata)
	assert.Equal(t, "interior", dicts[0].Metadata.Category)
	assert.Equal(t, []string{"global", "thai"}, dicts[0].Metadata.Regions)
	assert.Equal(t, []string{"villa", "house"}, dicts[0].Metadata.PropertyTypes)
}
