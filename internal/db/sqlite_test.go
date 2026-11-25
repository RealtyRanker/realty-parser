package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/asmisnik/realty-parser/internal/sqlite"
)

// Integration tests use the exact deploy schema, rather than a test-only copy.
func testDB(t *testing.T) (*DB, string) {
	t.Helper()
	schemaPath := os.Getenv("REALTYRANKER_TEST_SCHEMA")
	if schemaPath == "" {
		schemaPath = "../../../dev-tips/migrations/sqlite/001_initial.sql"
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(schemaPath), "*.sql"))
	if len(files) == 0 && os.Getenv("REALTYRANKER_TEST_SCHEMA") == "" {
		t.Skip("deploy schema unavailable; run dev-tips/test.sh in the full workspace")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatalf("deploy schema unavailable at %s", schemaPath)
	}
	dsn := "file:" + filepath.Join(t.TempDir(), "realtyranker.db")
	pool, err := sqlite.Open(context.Background(), dsn+"?mode=rwc")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		schema, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.ExecContext(context.Background(), string(schema)); err != nil {
			t.Fatal(err)
		}
	}
	pool.Close()
	database, err := New(context.Background(), dsn+"?mode=rw")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	return database, dsn + "?mode=rw"
}
