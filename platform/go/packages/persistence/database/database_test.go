package database

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ==============================================================================
// TEST HELPERS
// ==============================================================================

// createTestMigrationsDir creates a temp directory with migration files.
func createTestMigrationsDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", name, err)
		}
	}
	return dir
}

// ==============================================================================
// TEST: loadMigrations
// ==============================================================================

func TestLoadMigrations_UpSqlFiles(t *testing.T) {
	dir := createTestMigrationsDir(t, map[string]string{
		"001_create_users.up.sql":   "CREATE TABLE users (id INT);",
		"001_create_users.down.sql": "DROP TABLE users;",
		"002_add_email.up.sql":      "ALTER TABLE users ADD COLUMN email TEXT;",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("Expected 2 migrations, got %d", len(migrations))
	}

	// Check first migration
	if migrations[0].Version != "001" {
		t.Errorf("Expected version '001', got '%s'", migrations[0].Version)
	}
	if migrations[0].Name != "create_users" {
		t.Errorf("Expected name 'create_users', got '%s'", migrations[0].Name)
	}
	if migrations[0].Up != "CREATE TABLE users (id INT);" {
		t.Errorf("Expected UP SQL content, got '%s'", migrations[0].Up)
	}
	if migrations[0].Down != "DROP TABLE users;" {
		t.Errorf("Expected DOWN SQL content, got '%s'", migrations[0].Down)
	}

	// Check second migration has no down
	if migrations[1].Version != "002" {
		t.Errorf("Expected version '002', got '%s'", migrations[1].Version)
	}
	if migrations[1].Down != "" {
		t.Errorf("Expected empty down SQL, got '%s'", migrations[1].Down)
	}
}

func TestLoadMigrations_BareSqlFiles(t *testing.T) {
	// Backward compatibility: NNN_name.sql (no .up suffix)
	dir := createTestMigrationsDir(t, map[string]string{
		"001_create_properties.sql": "CREATE TABLE properties (id UUID);",
		"002_seed_properties.sql":   "INSERT INTO properties VALUES ('abc');",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("Expected 2 migrations, got %d", len(migrations))
	}

	if migrations[0].Version != "001" {
		t.Errorf("Expected version '001', got '%s'", migrations[0].Version)
	}
	if migrations[0].Name != "create_properties" {
		t.Errorf("Expected name 'create_properties', got '%s'", migrations[0].Name)
	}
	if migrations[0].Up != "CREATE TABLE properties (id UUID);" {
		t.Errorf("Unexpected UP SQL: %s", migrations[0].Up)
	}
}

func TestLoadMigrations_HyphenSeparator(t *testing.T) {
	// bestierealestate uses NNN-name.sql pattern
	dir := createTestMigrationsDir(t, map[string]string{
		"001-schema.sql":   "CREATE TABLE agents (id UUID);",
		"002-triggers.sql": "CREATE TRIGGER updated_at;",
		"003-seed.sql":     "INSERT INTO agents VALUES ('x');",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 3 {
		t.Fatalf("Expected 3 migrations, got %d", len(migrations))
	}

	if migrations[0].Version != "001" {
		t.Errorf("Expected version '001', got '%s'", migrations[0].Version)
	}
	if migrations[0].Name != "schema" {
		t.Errorf("Expected name 'schema', got '%s'", migrations[0].Name)
	}
}

func TestLoadMigrations_MixedNaming(t *testing.T) {
	// Mix of .up.sql and bare .sql in same dir
	dir := createTestMigrationsDir(t, map[string]string{
		"001_create_users.up.sql": "CREATE TABLE users (id INT);",
		"002_seed_data.sql":       "INSERT INTO users VALUES (1);",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("Expected 2 migrations, got %d", len(migrations))
	}

	if migrations[0].Version != "001" {
		t.Errorf("Expected version '001', got '%s'", migrations[0].Version)
	}
	if migrations[1].Version != "002" {
		t.Errorf("Expected version '002', got '%s'", migrations[1].Version)
	}
}

func TestLoadMigrations_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 0 {
		t.Errorf("Expected 0 migrations, got %d", len(migrations))
	}
}

func TestLoadMigrations_NonExistentDirectory(t *testing.T) {
	_, err := loadMigrations("/nonexistent/path/migrations")
	if err == nil {
		t.Error("Expected error for non-existent directory, got nil")
	}
}

func TestLoadMigrations_NonSqlFilesIgnored(t *testing.T) {
	dir := createTestMigrationsDir(t, map[string]string{
		"001_create_users.up.sql": "CREATE TABLE users (id INT);",
		"README.md":               "# Migrations",
		".gitkeep":                "",
		"notes.txt":               "some notes",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 1 {
		t.Errorf("Expected 1 migration (non-SQL ignored), got %d", len(migrations))
	}
}

func TestLoadMigrations_SortedByVersion(t *testing.T) {
	dir := createTestMigrationsDir(t, map[string]string{
		"003_third.up.sql":  "SELECT 3;",
		"001_first.up.sql":  "SELECT 1;",
		"002_second.up.sql": "SELECT 2;",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 3 {
		t.Fatalf("Expected 3 migrations, got %d", len(migrations))
	}

	for i, expected := range []string{"001", "002", "003"} {
		if migrations[i].Version != expected {
			t.Errorf("Migration %d: expected version '%s', got '%s'", i, expected, migrations[i].Version)
		}
	}
}

func TestLoadMigrations_DuplicateVersionErrors(t *testing.T) {
	// Two files sharing a version must fail loudly — otherwise the apply
	// high-water-mark records one and silently skips the other forever (the
	// BR 20260527400000 widen_phone_country_code regression).
	dir := createTestMigrationsDir(t, map[string]string{
		"20260527400000_search_vector.up.sql": "SELECT 1;",
		"20260527400000_widen_phone.up.sql":   "SELECT 2;",
	})

	_, err := loadMigrations(dir)
	if err == nil {
		t.Fatal("Expected an error for duplicate migration version, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate migration version 20260527400000") {
		t.Errorf("Expected error to name the duplicate version, got: %v", err)
	}
	// Both conflicting filenames must appear so the operator knows what to rename.
	for _, want := range []string{"search_vector", "widen_phone"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Expected error to name file %q, got: %v", want, err)
		}
	}
}

func TestLoadMigrations_DownSqlIgnoredAsStandalone(t *testing.T) {
	// .down.sql files should not be loaded as standalone migrations
	dir := createTestMigrationsDir(t, map[string]string{
		"001_create_users.up.sql":   "CREATE TABLE users (id INT);",
		"001_create_users.down.sql": "DROP TABLE users;",
	})

	migrations, err := loadMigrations(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(migrations) != 1 {
		t.Errorf("Expected 1 migration (.down.sql not standalone), got %d", len(migrations))
	}
}

// ==============================================================================
// TEST: checksumFile
// ==============================================================================

func TestChecksumFile(t *testing.T) {
	content := "CREATE TABLE users (id INT);"
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	dir := createTestMigrationsDir(t, map[string]string{
		"test.sql": content,
	})

	checksum, err := checksumFile(filepath.Join(dir, "test.sql"))
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if checksum != expected {
		t.Errorf("Expected checksum '%s', got '%s'", expected, checksum)
	}
}

func TestChecksumFile_NonExistent(t *testing.T) {
	_, err := checksumFile("/nonexistent/file.sql")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// ==============================================================================
// TEST: NextMigrationVersion — must return 14-digit UTC timestamp (YYYYMMDDHHmmss)
// ==============================================================================

var timestampVersionRe = regexp.MustCompile(`^\d{14}$`)

func TestNextMigrationVersion_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	version, err := NextMigrationVersion(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !timestampVersionRe.MatchString(version) {
		t.Errorf("Expected 14-digit timestamp version, got '%s'", version)
	}
}

func TestNextMigrationVersion_ExistingSequentialFiles(t *testing.T) {
	// Legacy sequential files must not break timestamp generation
	dir := createTestMigrationsDir(t, map[string]string{
		"001_first.up.sql":  "SELECT 1;",
		"002_second.up.sql": "SELECT 2;",
	})

	version, err := NextMigrationVersion(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !timestampVersionRe.MatchString(version) {
		t.Errorf("Expected 14-digit timestamp version, got '%s'", version)
	}
}

func TestNextMigrationVersion_BareSqlFiles(t *testing.T) {
	dir := createTestMigrationsDir(t, map[string]string{
		"001_first.sql":  "SELECT 1;",
		"002_second.sql": "SELECT 2;",
		"003_third.sql":  "SELECT 3;",
	})

	version, err := NextMigrationVersion(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !timestampVersionRe.MatchString(version) {
		t.Errorf("Expected 14-digit timestamp version, got '%s'", version)
	}
}

func TestNextMigrationVersion_HyphenSeparator(t *testing.T) {
	dir := createTestMigrationsDir(t, map[string]string{
		"001-schema.sql":   "SELECT 1;",
		"002-triggers.sql": "SELECT 2;",
	})

	version, err := NextMigrationVersion(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !timestampVersionRe.MatchString(version) {
		t.Errorf("Expected 14-digit timestamp version, got '%s'", version)
	}
}

func TestNextMigrationVersion_TimestampSortsAfterSequential(t *testing.T) {
	// Timestamp version must lexicographically sort after existing sequential versions
	// so that new migrations always run last
	dir := createTestMigrationsDir(t, map[string]string{
		"007-dict.sql": "SELECT 1;",
	})

	version, err := NextMigrationVersion(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if version <= "007" {
		t.Errorf("Timestamp version %q must sort after '007'", version)
	}
}

func TestNextMigrationVersion_FutureDatedSequenceBumpsLatest(t *testing.T) {
	// Regression: when the existing sequence is FUTURE-DATED (a parallel sprint
	// stamped versions ahead of today's date), a wall-clock version sorts BELOW
	// the max and silently never runs. The next version must be latest+1.
	dir := createTestMigrationsDir(t, map[string]string{
		"20260101000000_first.up.sql":  "SELECT 1;",
		"29991231000000_future.up.sql": "SELECT 1;",
	})

	version, err := NextMigrationVersion(dir)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// latest is 29991231000000; +1 = 29991231000001 (integer increment of the
	// sortable version string, NOT a clock — it just bumps the last digit).
	if version != "29991231000001" {
		t.Errorf("Expected latest+1 '29991231000001', got %q", version)
	}
	if version <= "29991231000000" {
		t.Errorf("Version %q must sort strictly after the future-dated max", version)
	}
}

func TestIncrementNumericVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"20260623000001", "20260623000002", true},
		{"007", "008", true},
		{"not-numeric", "", false},
	}
	for _, c := range cases {
		got, ok := incrementNumericVersion(c.in)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("incrementNumericVersion(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// ==============================================================================
// TEST: stripTxControl (ExecFile preprocessing)
// ==============================================================================

func TestStripTxControl(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "begin/commit block stripped",
			in:   "BEGIN;\nINSERT INTO t VALUES (1);\nCOMMIT;\n",
			want: "\nINSERT INTO t VALUES (1);\n\n",
		},
		{
			name: "start transaction + end stripped",
			in:   "START TRANSACTION;\nSELECT 1;\nEND;\n",
			want: "\nSELECT 1;\n\n",
		},
		{
			name: "rollback stripped",
			in:   "INSERT INTO t VALUES (2);\nROLLBACK;\n",
			want: "INSERT INTO t VALUES (2);\n\n",
		},
		{
			name: "case-insensitive + optional semicolon",
			in:   "begin\ncommit\nBegin;\n",
			want: "\n\n\n",
		},
		{
			name: "SET LOCAL preserved (needed inside the outer tx)",
			in:   "BEGIN;\nSET LOCAL search_path TO bestierealestate;\nSELECT 1;\nCOMMIT;\n",
			want: "\nSET LOCAL search_path TO bestierealestate;\nSELECT 1;\n\n",
		},
		{
			name: "real statements preserved verbatim",
			in:   "INSERT INTO properties (id) VALUES ('x') ON CONFLICT (id) DO NOTHING;\nUPDATE properties SET title='y' WHERE id='x';\n",
			want: "INSERT INTO properties (id) VALUES ('x') ON CONFLICT (id) DO NOTHING;\nUPDATE properties SET title='y' WHERE id='x';\n",
		},
		{
			name: "mid-line begin keyword not matched (standalone-line scope)",
			in:   "INSERT INTO logs (msg) VALUES ('BEGIN work now');\n",
			want: "INSERT INTO logs (msg) VALUES ('BEGIN work now');\n",
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripTxControl(c.in)
			if got != c.want {
				t.Errorf("stripTxControl(%q)\n got: %q\nwant: %q", c.in, got, c.want)
			}
		})
	}
}

// A standalone BEGIN/END line inside a dollar-quoted body is CONTENT (plpgsql),
// not transaction control — stripping it corrupts the block (2607-014 SC02).
func TestStripTxControl_DollarQuotedBodies(t *testing.T) {
	fnBody := "CREATE FUNCTION f() RETURNS trigger AS $fn$\nBEGIN\n  RETURN NEW;\nEND;\n$fn$ LANGUAGE plpgsql;\n"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "DO block body preserved, outer wrapper stripped",
			in:   "BEGIN;\nDO $$\nBEGIN\n  UPDATE t SET n = n + 1;\nEND\n$$;\nCOMMIT;\n",
			want: "\nDO $$\nBEGIN\n  UPDATE t SET n = n + 1;\nEND\n$$;\n\n",
		},
		{
			name: "tagged function body preserved verbatim",
			in:   fnBody,
			want: fnBody,
		},
		{
			name: "END inside body preserved, END outside stripped",
			in:   "DO $$\nBEGIN\n  NULL;\nEND;\n$$;\nEND;\n",
			want: "DO $$\nBEGIN\n  NULL;\nEND;\n$$;\n\n",
		},
		{
			name: "tag with underscore and digits",
			in:   "SELECT $tag_1$\nCOMMIT;\n$tag_1$;\nCOMMIT;\n",
			want: "SELECT $tag_1$\nCOMMIT;\n$tag_1$;\n\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripTxControl(c.in)
			if got != c.want {
				t.Errorf("stripTxControl(%q)\n got: %q\nwant: %q", c.in, got, c.want)
			}
		})
	}
}

// Standalone tx-control lines inside string literals, quoted identifiers, and
// block comments are content too — only lines OUTSIDE every quoted/commented
// region are stripped (2607-014 SC03).
func TestStripTxControl_QuotedAndCommentedRegions(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "multi-line single-quoted literal keeps bare BEGIN line",
			in:   "INSERT INTO logs (msg) VALUES ('first\nBEGIN\nlast');\nCOMMIT;\n",
			want: "INSERT INTO logs (msg) VALUES ('first\nBEGIN\nlast');\n\n",
		},
		{
			name: "doubled quote does not close the literal",
			in:   "INSERT INTO t VALUES ('it''s\nROLLBACK\nstill string');\n",
			want: "INSERT INTO t VALUES ('it''s\nROLLBACK\nstill string');\n",
		},
		{
			name: "E-string backslash-escaped quote does not close the literal",
			in:   "INSERT INTO t VALUES (E'a\\'\nCOMMIT\nb');\n",
			want: "INSERT INTO t VALUES (E'a\\'\nCOMMIT\nb');\n",
		},
		{
			name: "plain string backslash is literal (standard_conforming_strings)",
			in:   "INSERT INTO t VALUES ('a\\');\nCOMMIT;\n",
			want: "INSERT INTO t VALUES ('a\\');\n\n",
		},
		{
			name: "multi-line block comment keeps BEGIN line",
			in:   "/* wrapper\nBEGIN\n*/\nBEGIN;\nSELECT 1;\nCOMMIT;\n",
			want: "/* wrapper\nBEGIN\n*/\n\nSELECT 1;\n\n",
		},
		{
			name: "nested block comments stay one region",
			in:   "/* a /* b\nCOMMIT\n*/ c\nCOMMIT\n*/\nCOMMIT;\n",
			want: "/* a /* b\nCOMMIT\n*/ c\nCOMMIT\n*/\n\n",
		},
		{
			name: "double-quoted identifier spanning lines",
			in:   "SELECT 1 AS \"weird\nBEGIN\nname\";\nCOMMIT;\n",
			want: "SELECT 1 AS \"weird\nBEGIN\nname\";\n\n",
		},
		{
			name: "line comment never opens a region",
			in:   "-- note: 'unclosed quote\nBEGIN;\nSELECT 1;\n",
			want: "-- note: 'unclosed quote\n\nSELECT 1;\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := stripTxControl(c.in)
			if got != c.want {
				t.Errorf("stripTxControl(%q)\n got: %q\nwant: %q", c.in, got, c.want)
			}
		})
	}
}

// ==============================================================================
// TEST: ResolveSchema — one shared resolver for the migrate + sqlexec one-shots
// ==============================================================================

func TestResolveSchema(t *testing.T) {
	t.Setenv("DATABASE_SCHEMA", "")
	if got := ResolveSchema("postgres://bestierealestate:p@h:5432/db"); got != "bestierealestate" {
		t.Errorf("URL-user fallback: got %q, want %q", got, "bestierealestate")
	}
	if got := ResolveSchema("postgres://h/db"); got != "" {
		t.Errorf("userless URL: got %q, want empty", got)
	}
	if got := ResolveSchema("://not-a-url"); got != "" {
		t.Errorf("unparseable URL: got %q, want empty", got)
	}
	t.Setenv("DATABASE_SCHEMA", "custom")
	if got := ResolveSchema("postgres://other:p@h/db"); got != "custom" {
		t.Errorf("env precedence: got %q, want %q", got, "custom")
	}
}
