package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCloseReleasesSQLiteFilesForDirectoryCleanup(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "sqlite-artifacts")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root: %v", err)
	}

	database, err := New(filepath.Join(root, "emerald.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	closed := false
	t.Cleanup(func() {
		if !closed {
			_ = database.Close()
		}
	})

	if err := Migrate(database); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if _, err := database.Exec(`CREATE TABLE close_test (id INTEGER PRIMARY KEY, value TEXT)`); err != nil {
		t.Fatalf("create close_test: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO close_test (value) VALUES ('value')`); err != nil {
		t.Fatalf("seed close_test: %v", err)
	}

	if err := database.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	closed = true

	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("RemoveAll after Close: %v", err)
	}
}
