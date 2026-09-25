package postgres_test

import "testing"

// TC1-SC1: Full-text search returns ranked results (title > location > description)
func TestPostgresSearcher_Search_FullText(t *testing.T) {
	t.Skip("RED: PostgresSearcher not yet implemented — awaiting MD layer step")
}

// TC3-SC7: pg_trgm fallback when tsvector returns zero matches
func TestPostgresSearcher_Search_TrgmFallback(t *testing.T) {
	t.Skip("RED: PostgresSearcher not yet implemented — awaiting MD layer step")
}

// TC-SC3: Filter by bedrooms >= N and area
func TestPostgresSearcher_Search_BedroomAreaFilter(t *testing.T) {
	t.Skip("RED: PostgresSearcher not yet implemented — awaiting MD layer step")
}

// TC-SC4: Property amenities loaded from join tables
func TestPostgresSearcher_Search_WithAmenities(t *testing.T) {
	t.Skip("RED: PostgresSearcher not yet implemented — awaiting MD layer step")
}

// TC-SC5: Admin search with status=draft filter
func TestPostgresSearcher_Search_DraftFilter(t *testing.T) {
	t.Skip("RED: PostgresSearcher not yet implemented — awaiting MD layer step")
}
