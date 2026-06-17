package diff

import (
	"github.com/fuleinist/schema-sync/internal/schema"
)

// ComputeDiff compares two schemas and returns the differences
func ComputeDiff(oldSchema, newSchema *schema.Schema) *schema.DiffResult {
	result := &schema.DiffResult{
		Added:    []schema.Table{},
		Removed:  []schema.Table{},
		Modified: []schema.TableDiff{},
	}

	oldTables := make(map[string]schema.Table)
	newTables := make(map[string]schema.Table)

	for _, t := range oldSchema.Tables {
		oldTables[t.Name] = t
	}
	for _, t := range newSchema.Tables {
		newTables[t.Name] = t
	}

	// Find added tables
	for name, newT := range newTables {
		if _, exists := oldTables[name]; !exists {
			result.Added = append(result.Added, newT)
		}
	}

	// Find removed tables
	for name, oldT := range oldTables {
		if _, exists := newTables[name]; !exists {
			result.Removed = append(result.Removed, oldT)
		}
	}

	// Find modified tables
	for name, newT := range newTables {
		if oldT, exists := oldTables[name]; exists {
			tableDiff := compareTables(oldT, newT)
			if len(tableDiff.AddedColumns) > 0 || len(tableDiff.DroppedColumns) > 0 ||
				len(tableDiff.ModifiedColumns) > 0 || len(tableDiff.AddedIndexes) > 0 ||
				len(tableDiff.DroppedIndexes) > 0 || len(tableDiff.AddedFKs) > 0 ||
				len(tableDiff.DroppedFKs) > 0 {
				tableDiff.TableName = name
				result.Modified = append(result.Modified, tableDiff)
			}
		}
	}

	return result
}

func compareTables(oldTable, newTable schema.Table) schema.TableDiff {
	td := schema.TableDiff{}

	oldCols := make(map[string]schema.Column)
	newCols := make(map[string]schema.Column)
	for _, c := range oldTable.Columns {
		oldCols[c.Name] = c
	}
	for _, c := range newTable.Columns {
		newCols[c.Name] = c
	}

	// Added columns
	for name, col := range newCols {
		if _, exists := oldCols[name]; !exists {
			td.AddedColumns = append(td.AddedColumns, col)
		}
	}

	// Dropped columns
	for name, col := range oldCols {
		if _, exists := newCols[name]; !exists {
			td.DroppedColumns = append(td.DroppedColumns, col)
		}
	}

	// Modified columns
	for name, newCol := range newCols {
		if oldCol, exists := oldCols[name]; exists {
			if oldCol.Type != newCol.Type || oldCol.Nullable != newCol.Nullable ||
				!EqualDefault(oldCol.Default, newCol.Default) {
				td.ModifiedColumns = append(td.ModifiedColumns, schema.ColumnChange{
					Name:      name,
					OldType:   oldCol.Type,
					NewType:   newCol.Type,
					OldNull:   oldCol.Nullable,
					NewNull:   newCol.Nullable,
					OldDefault: oldCol.Default,
					NewDefault: newCol.Default,
				})
			}
		}
	}

	// Compare indexes
	td.AddedIndexes = compareIndexes(oldTable.Indexes, newTable.Indexes)
	td.DroppedIndexes = compareIndexes(newTable.Indexes, oldTable.Indexes)

	// Compare foreign keys
	td.AddedFKs = compareFKs(oldTable.ForeignKeys, newTable.ForeignKeys)
	td.DroppedFKs = compareFKs(newTable.ForeignKeys, oldTable.ForeignKeys)

	return td
}

func compareIndexes(oldIndexes, newIndexes []schema.Index) []schema.Index {
	var added []schema.Index
	oldIdx := make(map[string]schema.Index)
	for _, idx := range oldIndexes {
		oldIdx[idx.Name] = idx
	}
	for _, idx := range newIndexes {
		if _, exists := oldIdx[idx.Name]; !exists {
			added = append(added, idx)
		}
	}
	return added
}

func compareFKs(oldFKs, newFKs []schema.ForeignKey) []schema.ForeignKey {
	var added []schema.ForeignKey
	oldFK := make(map[string]schema.ForeignKey)
	for _, fk := range oldFKs {
		oldFK[fk.Name] = fk
	}
	for _, fk := range newFKs {
		if _, exists := oldFK[fk.Name]; !exists {
			added = append(added, fk)
		}
	}
	return added
}

// EqualDefault reports whether two *string column default values are
// equal. Two nil defaults are equal; one nil and one non-nil are not.
func EqualDefault(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}