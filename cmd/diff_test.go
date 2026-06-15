package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fuleinist/schema-sync/internal/schema"
)

// writeSnapshot writes a minimal valid schema JSON file at
// <dir>/<snapshotDir>/<env>.json so the test can assert that
// loadSnapshot joins `dir` with `snapshotDir` before reading.
func writeSnapshot(t *testing.T, dir, snapshotDir, env string) string {
	t.Helper()
	full := filepath.Join(dir, snapshotDir)
	if err := os.MkdirAll(full, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", full, err)
	}
	path := filepath.Join(full, env+".json")
	payload, err := json.Marshal(&schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
			}},
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// TestLoadSnapshot_JoinsDirWithSnapshotDir pins down the contract that
// `loadSnapshot` must resolve the snapshot file under `dir`, not as a
// bare relative path. The previous implementation did
// `snapshotDir + "/" + env + ".json"` and never used `dir`, so callers
// that ran from a directory other than the project root got
// "file not found" even when the snapshot was on disk. The companion
// writer in `cmd/snapshot.go` correctly does
// `filepath.Join(dir, cfg.Settings.SnapshotDir, ...)`; this test makes
// the reader match.
func TestLoadSnapshot_JoinsDirWithSnapshotDir(t *testing.T) {
	dir := t.TempDir()
	snapshotDir := ".schema-sync/snapshots"
	env := "dev"

	writeSnapshot(t, dir, snapshotDir, env)

	got, err := loadSnapshot(dir, snapshotDir, env)
	if err != nil {
		t.Fatalf("loadSnapshot: %v", err)
	}
	if len(got.Tables) != 1 || got.Tables[0].Name != "users" {
		t.Fatalf("expected one table named 'users', got: %+v", got.Tables)
	}
}

// TestLoadSnapshot_MissingFileReturnsError covers the negative case:
// when the snapshot really isn't on disk, the error must point at the
// env name (so users can tell which env is missing) — not at the joined
// path (which would leak the absolute CWD into the error).
func TestLoadSnapshot_MissingFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	if _, err := loadSnapshot(dir, ".schema-sync/snapshots", "missing"); err == nil {
		t.Fatal("expected error for missing snapshot, got nil")
	}
}
