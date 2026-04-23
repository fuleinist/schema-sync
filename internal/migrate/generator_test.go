package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuleinist/schema-sync/internal/schema"
)

func TestGenerator_GenerateMigration_AddedTable(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator(tmpDir)

	diff := &schema.DiffResult{
		Added: []schema.Table{
			{
				Name: "users",
				Columns: []schema.Column{
					{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
					{Name: "email", Type: "VARCHAR(255)", Nullable: false},
				},
			},
		},
	}

	path, err := gen.GenerateMigration("dev", diff)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("Migration file not created at %s", path)
	}

	// Read and verify content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)

	// Check for UP migration
	if !strings.Contains(contentStr, "-- +migrate Up") {
		t.Error("Missing UP migration marker")
	}
	if !strings.Contains(contentStr, "CREATE TABLE users") {
		t.Error("Missing CREATE TABLE statement")
	}
	if !strings.Contains(contentStr, "id INTEGER") {
		t.Error("Missing id column definition")
	}
	if !strings.Contains(contentStr, "email VARCHAR(255)") {
		t.Error("Missing email column definition")
	}
	if !strings.Contains(contentStr, "NOT NULL") {
		t.Error("Missing NOT NULL constraint")
	}
	if !strings.Contains(contentStr, "PRIMARY KEY") {
		t.Error("Missing PRIMARY KEY constraint")
	}

	// Check for DOWN migration
	if !strings.Contains(contentStr, "-- +migrate Down") {
		t.Error("Missing DOWN migration marker")
	}
	if !strings.Contains(contentStr, "DROP TABLE IF EXISTS users") {
		t.Error("Missing DROP TABLE in DOWN migration")
	}
}

func TestGenerator_GenerateMigration_ModifiedColumns(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator(tmpDir)

	diff := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				ModifiedColumns: []schema.ColumnChange{
					{Name: "email", OldType: "VARCHAR(100)", NewType: "VARCHAR(255)", OldNull: true, NewNull: false},
				},
				AddedColumns: []schema.Column{
					{Name: "created_at", Type: "TIMESTAMP", Nullable: false},
				},
			},
		},
	}

	path, err := gen.GenerateMigration("prod", diff)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)

	// Check for ALTER COLUMN
	if !strings.Contains(contentStr, "ALTER TABLE users") {
		t.Error("Missing ALTER TABLE statement")
	}
	if !strings.Contains(contentStr, "created_at TIMESTAMP") {
		t.Error("Missing ADD COLUMN for created_at")
	}
}

func TestGenerator_GenerateMigration_AddedIndex(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator(tmpDir)

	diff := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				AddedIndexes: []schema.Index{
					{Name: "idx_email", Columns: []string{"email"}, Unique: false},
				},
			},
		},
	}

	path, err := gen.GenerateMigration("prod", diff)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "CREATE INDEX idx_email") {
		t.Error("Missing CREATE INDEX statement")
	}
	if !strings.Contains(contentStr, "ON users (email)") {
		t.Error("Missing index columns")
	}
}

func TestGenerator_GenerateMigration_RemovedTable(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator(tmpDir)

	diff := &schema.DiffResult{
		Removed: []schema.Table{
			{Name: "deprecated_table", Columns: []schema.Column{}},
		},
	}

	path, err := gen.GenerateMigration("prod", diff)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "DROP TABLE IF EXISTS deprecated_table") {
		t.Error("Missing DROP TABLE in DOWN migration")
	}
}

func TestGenerator_GenerateMigration_FilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator(tmpDir)

	diff := &schema.DiffResult{}

	path, err := gen.GenerateMigration("staging", diff)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	filename := filepath.Base(path)
	if !strings.HasPrefix(filename, "20") {
		t.Errorf("Filename should start with timestamp, got: %s", filename)
	}
	if !strings.Contains(filename, "staging") {
		t.Errorf("Filename should contain environment name, got: %s", filename)
	}
	if !strings.HasSuffix(filename, ".sql") {
		t.Errorf("Filename should have .sql extension, got: %s", filename)
	}
}

func TestGenerator_GenerateMigration_DefaultValue(t *testing.T) {
	tmpDir := t.TempDir()
	gen := NewGenerator(tmpDir)

	defaultVal := "false"
	diff := &schema.DiffResult{
		Modified: []schema.TableDiff{
			{
				TableName: "users",
				AddedColumns: []schema.Column{
					{Name: "active", Type: "BOOLEAN", Nullable: false, Default: &defaultVal},
				},
			},
		},
	}

	path, err := gen.GenerateMigration("dev", diff)
	if err != nil {
		t.Fatalf("GenerateMigration failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "DEFAULT false") {
		t.Error("Missing DEFAULT value in column definition")
	}
}
