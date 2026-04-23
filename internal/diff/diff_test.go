package diff

import (
	"testing"

	"github.com/fuleinist/schema-sync/internal/schema"
)

func TestComputeDiff_AddedTables(t *testing.T) {
	oldSchema := &schema.Schema{Tables: []schema.Table{}}
	newSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
			}},
		},
	}

	result := ComputeDiff(oldSchema, newSchema)

	if len(result.Added) != 1 {
		t.Errorf("Expected 1 added table, got %d", len(result.Added))
	}
	if result.Added[0].Name != "users" {
		t.Errorf("Expected added table name 'users', got '%s'", result.Added[0].Name)
	}
	if len(result.Removed) != 0 {
		t.Errorf("Expected 0 removed tables, got %d", len(result.Removed))
	}
	if len(result.Modified) != 0 {
		t.Errorf("Expected 0 modified tables, got %d", len(result.Modified))
	}
}

func TestComputeDiff_RemovedTables(t *testing.T) {
	oldSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
			}},
		},
	}
	newSchema := &schema.Schema{Tables: []schema.Table{}}

	result := ComputeDiff(oldSchema, newSchema)

	if len(result.Removed) != 1 {
		t.Errorf("Expected 1 removed table, got %d", len(result.Removed))
	}
	if result.Removed[0].Name != "users" {
		t.Errorf("Expected removed table name 'users', got '%s'", result.Removed[0].Name)
	}
}

func TestComputeDiff_ModifiedColumns(t *testing.T) {
	oldSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
				{Name: "email", Type: "VARCHAR(100)", Nullable: true},
			}},
		},
	}
	newSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
				{Name: "email", Type: "VARCHAR(255)", Nullable: false}, // type and null changed
				{Name: "created_at", Type: "TIMESTAMP", Nullable: false}, // added
			}},
		},
	}

	result := ComputeDiff(oldSchema, newSchema)

	if len(result.Modified) != 1 {
		t.Fatalf("Expected 1 modified table, got %d", len(result.Modified))
	}

	mod := result.Modified[0]
	if len(mod.ModifiedColumns) != 1 {
		t.Errorf("Expected 1 modified column, got %d", len(mod.ModifiedColumns))
	}
	if mod.ModifiedColumns[0].Name != "email" {
		t.Errorf("Expected modified column 'email', got '%s'", mod.ModifiedColumns[0].Name)
	}
	if mod.ModifiedColumns[0].OldType != "VARCHAR(100)" {
		t.Errorf("Expected old type 'VARCHAR(100)', got '%s'", mod.ModifiedColumns[0].OldType)
	}
	if mod.ModifiedColumns[0].NewType != "VARCHAR(255)" {
		t.Errorf("Expected new type 'VARCHAR(255)', got '%s'", mod.ModifiedColumns[0].NewType)
	}

	if len(mod.AddedColumns) != 1 {
		t.Errorf("Expected 1 added column, got %d", len(mod.AddedColumns))
	}
	if mod.AddedColumns[0].Name != "created_at" {
		t.Errorf("Expected added column 'created_at', got '%s'", mod.AddedColumns[0].Name)
	}
}

func TestComputeDiff_AddedIndexes(t *testing.T) {
	oldSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
				{Name: "email", Type: "VARCHAR(255)", Nullable: false},
			}},
		},
	}
	newSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
				{Name: "email", Type: "VARCHAR(255)", Nullable: false},
			}, Indexes: []schema.Index{
				{Name: "idx_email", Columns: []string{"email"}, Unique: false},
			}},
		},
	}

	result := ComputeDiff(oldSchema, newSchema)

	if len(result.Modified) != 1 {
		t.Fatalf("Expected 1 modified table, got %d", len(result.Modified))
	}
	if len(result.Modified[0].AddedIndexes) != 1 {
		t.Errorf("Expected 1 added index, got %d", len(result.Modified[0].AddedIndexes))
	}
	if result.Modified[0].AddedIndexes[0].Name != "idx_email" {
		t.Errorf("Expected added index 'idx_email', got '%s'", result.Modified[0].AddedIndexes[0].Name)
	}
}

func TestComputeDiff_NoChanges(t *testing.T) {
	oldSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
			}},
		},
	}
	newSchema := &schema.Schema{
		Tables: []schema.Table{
			{Name: "users", Columns: []schema.Column{
				{Name: "id", Type: "INTEGER", Nullable: false, Primary: true},
			}},
		},
	}

	result := ComputeDiff(oldSchema, newSchema)

	if len(result.Added) != 0 || len(result.Removed) != 0 || len(result.Modified) != 0 {
		t.Errorf("Expected no changes, got Added=%d, Removed=%d, Modified=%d",
			len(result.Added), len(result.Removed), len(result.Modified))
	}
}
