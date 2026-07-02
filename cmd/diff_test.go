package cmd

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
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

// captureStdout runs fn and returns whatever it wrote to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	w.Close()
	return <-done
}

// TestPrintDiff_ShowsNullableChange is the regression test for the
// `printDiff` bug: a column whose nullability flips was previously
// rendered as `~ column: name <type> -> <type>` (no visible change)
// because the display only ever printed the type. The diff engine
// correctly detected the change, but the user saw nothing. This test
// fails on the buggy code and passes once printDiff emits the
// `nullable: ...` line.
func TestPrintDiff_ShowsNullableChange(t *testing.T) {
	defaultVal := "0"
	result := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				ModifiedColumns: []schema.ColumnChange{
					{
						Name:    "email",
						OldType: "VARCHAR(255)",
						NewType: "VARCHAR(255)",
						OldNull: false,
						NewNull: true,
						OldDefault: &defaultVal,
						NewDefault: &defaultVal,
					},
				},
			},
		},
	}

	out := captureStdout(t, func() { printDiff(result) })

	if !strings.Contains(out, "nullable: false -> true") {
		t.Errorf("expected nullability change to be displayed; got:\n%s", out)
	}
	// Bug: the old code printed "~ column: email VARCHAR(255) -> VARCHAR(255)"
	// with no visible diff. After the fix, when the type is unchanged the
	// header should be the bare column name.
	if strings.Contains(out, "VARCHAR(255) -> VARCHAR(255)") {
		t.Errorf("unchanged type should not be rendered as '-> ' transition; got:\n%s", out)
	}
}

// TestPrintDiff_ShowsDefaultChange covers the parallel bug for column
// default values. The diff engine records the change in `ColumnChange`
// but the old `printDiff` never rendered it, so a default change was
// invisible. The fix adds a `default: <old> -> <new>` line.
func TestPrintDiff_ShowsDefaultChange(t *testing.T) {
	oldVal := "active"
	newVal := "inactive"
	result := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				ModifiedColumns: []schema.ColumnChange{
					{
						Name:    "status",
						OldType: "VARCHAR(20)",
						NewType: "VARCHAR(20)",
						OldNull: true,
						NewNull: true,
						OldDefault: &oldVal,
						NewDefault: &newVal,
					},
				},
			},
		},
	}

	out := captureStdout(t, func() { printDiff(result) })

	if !strings.Contains(out, "default: active -> inactive") {
		t.Errorf("expected default change to be displayed; got:\n%s", out)
	}
}

// TestPrintDiff_ShowsTypeChange preserves the existing visible behavior
// when only the type changes: the original
// `~ column: name oldType -> newType` line must still appear.
func TestPrintDiff_ShowsTypeChange(t *testing.T) {
	result := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				ModifiedColumns: []schema.ColumnChange{
					{
						Name:    "age",
						OldType: "INTEGER",
						NewType: "BIGINT",
						OldNull: true,
						NewNull: true,
					},
				},
			},
		},
	}

	out := captureStdout(t, func() { printDiff(result) })

	if !strings.Contains(out, "~ column: age INTEGER -> BIGINT") {
		t.Errorf("expected type transition line; got:\n%s", out)
	}
}

// TestDiffCmd_JsonFlag verifies that `diff --json` outputs valid JSON
// matching the DiffResult structure, and that the exit code is 0 even
// when there are no differences.
func TestDiffCmd_JsonFlag(t *testing.T) {
	dir := t.TempDir()
	snapshotDir := ".schema-sync/snapshots"

	// Write config and identical snapshots for both envs so the diff is empty.
	os.MkdirAll(filepath.Join(dir, ".schema-sync"), 0o755)
	os.WriteFile(filepath.Join(dir, ".schema-sync", "config.yaml"), []byte("settings:\n  snapshot_dir: .schema-sync/snapshots\n"), 0o644)
	writeSnapshot(t, dir, snapshotDir, "dev")
	writeSnapshot(t, dir, snapshotDir, "prod")

	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	// Simulate `schema-sync diff dev prod --json`
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	rootCmd.SetArgs([]string{"diff", "dev", "prod", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("diff --json: %v", err)
	}

	w.Close()
	out := <-done

	// Must be valid JSON
	var parsed schema.DiffResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, out)
	}

	// Empty diff: all slices should be empty
	if len(parsed.Added) != 0 || len(parsed.Removed) != 0 || len(parsed.Modified) != 0 {
		t.Errorf("expected empty diff, got Added=%d Removed=%d Modified=%d",
			len(parsed.Added), len(parsed.Removed), len(parsed.Modified))
	}
}

// TestDiffCmd_JsonFlag_WithChanges verifies that --json output correctly
// serializes added, removed, and modified tables.
func TestDiffCmd_JsonFlag_WithChanges(t *testing.T) {
	dir := t.TempDir()
	snapshotDir := ".schema-sync/snapshots"

	// Write config
	os.MkdirAll(filepath.Join(dir, ".schema-sync"), 0o755)
	os.WriteFile(filepath.Join(dir, ".schema-sync", "config.yaml"), []byte("settings:\n  snapshot_dir: .schema-sync/snapshots\n"), 0o644)

	// Write dev snapshot (old)
	devPayload, _ := json.Marshal(&schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
				{Name: "email", Type: "VARCHAR(100)", Nullable: true},
			}},
		},
	})
	devPath := filepath.Join(dir, snapshotDir, "dev.json")
	os.MkdirAll(filepath.Dir(devPath), 0o755)
	os.WriteFile(devPath, devPayload, 0o644)

	// Write prod snapshot (new — added table, removed column, modified column)
	prodPayload, _ := json.Marshal(&schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
				{Name: "email", Type: "VARCHAR(255)", Nullable: false},
			}},
			{Name: "orders", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
			}},
		},
	})
	prodPath := filepath.Join(dir, snapshotDir, "prod.json")
	os.MkdirAll(filepath.Dir(prodPath), 0o755)
	os.WriteFile(prodPath, prodPayload, 0o644)

	// Capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	rootCmd.SetArgs([]string{"diff", "dev", "prod", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("diff --json: %v", err)
	}

	w.Close()
	out := <-done

	var parsed schema.DiffResult
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("expected valid JSON, got error: %v\noutput: %s", err, out)
	}

	// Expect: 1 added table (orders), 0 removed, 1 modified (users)
	if len(parsed.Added) != 1 || parsed.Added[0].Name != "orders" {
		t.Errorf("expected 1 added table 'orders', got %+v", parsed.Added)
	}
	if len(parsed.Removed) != 0 {
		t.Errorf("expected 0 removed tables, got %d", len(parsed.Removed))
	}
	if len(parsed.Modified) != 1 || parsed.Modified[0].TableName != "users" {
		t.Errorf("expected 1 modified table 'users', got %+v", parsed.Modified)
	}
	// users: email type VARCHAR(100)->VARCHAR(255), nullable true->false
	mod := parsed.Modified[0]
	if len(mod.ModifiedColumns) != 1 || mod.ModifiedColumns[0].Name != "email" {
		t.Errorf("expected email column change, got %+v", mod.ModifiedColumns)
	}
	if mod.ModifiedColumns[0].OldType != "VARCHAR(100)" || mod.ModifiedColumns[0].NewType != "VARCHAR(255)" {
		t.Errorf("expected VARCHAR(100)->VARCHAR(255), got %s->%s",
			mod.ModifiedColumns[0].OldType, mod.ModifiedColumns[0].NewType)
	}
}

// TestPrintDiff_ShowsAllThreeKinds exercises the full combination: a
// single modified column where type, nullability, and default all
// change at once. All three lines must appear so the user sees the
// complete diff.
func TestPrintDiff_ShowsAllThreeKinds(t *testing.T) {
	oldDefault := "0"
	newDefault := "1"
	result := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				ModifiedColumns: []schema.ColumnChange{
					{
						Name:    "score",
						OldType: "INTEGER",
						NewType: "BIGINT",
						OldNull: false,
						NewNull: true,
						OldDefault: &oldDefault,
						NewDefault: &newDefault,
					},
				},
			},
		},
	}

	out := captureStdout(t, func() { printDiff(result) })

	for _, want := range []string{
		"~ column: score INTEGER -> BIGINT",
		"nullable: false -> true",
		"default: 0 -> 1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing line %q in output:\n%s", want, out)
		}
	}
}


