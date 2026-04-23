package schema

import (
	"database/sql"
	"fmt"
	"strings"
)

// SQLiteExtractor extracts schema from SQLite
type SQLiteExtractor struct{}

func (e *SQLiteExtractor) DBType() string {
	return "sqlite"
}

func (e *SQLiteExtractor) Extract(db *sql.DB) (*Schema, error) {
	tables, err := e.extractTables(db)
	if err != nil {
		return nil, err
	}
	return &Schema{Tables: tables}, nil
}

func (e *SQLiteExtractor) extractTables(db *sql.DB) ([]Table, error) {
	query := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query tables: %w", err)
	}
	defer rows.Close()

	var tables []Table
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		columns, err := e.extractColumns(db, name)
		if err != nil {
			return nil, err
		}

		indexes, err := e.extractIndexes(db, name)
		if err != nil {
			return nil, err
		}

		fks, err := e.extractForeignKeys(db, name)
		if err != nil {
			return nil, err
		}

		tables = append(tables, Table{
			Name:        name,
			Columns:     columns,
			Indexes:     indexes,
			ForeignKeys: fks,
		})
	}
	return tables, nil
}

func (e *SQLiteExtractor) extractColumns(db *sql.DB, tableName string) ([]Column, error) {
	query := fmt.Sprintf("PRAGMA table_info('%s')", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var cid int
		var name, colType string
		var notnull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		col := Column{
			Name:     name,
			Type:     colType,
			Nullable: notnull == 0,
			Primary:  pk == 1,
		}
		if dfltValue != nil {
			s := fmt.Sprintf("%v", dfltValue)
			col.Default = &s
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (e *SQLiteExtractor) extractIndexes(db *sql.DB, tableName string) ([]Index, error) {
	query := fmt.Sprintf("PRAGMA index_list('%s')", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var name string
		var unique bool
		var origin string
		if err := rows.Scan(&name, &unique, &origin); err != nil {
			return nil, err
		}

		// Get columns for this index
		colQuery := fmt.Sprintf("PRAGMA index_info('%s')", name)
		colRows, err := db.Query(colQuery)
		if err != nil {
			return nil, err
		}

		var cols []string
		for colRows.Next() {
			var seq int
			var colName string
			if err := colRows.Scan(&seq, &colName); err != nil {
				colRows.Close()
				return nil, err
			}
			cols = append(cols, colName)
		}
		colRows.Close()

		indexes = append(indexes, Index{
			Name:    name,
			Columns: cols,
			Unique:  unique,
		})
	}
	return indexes, nil
}

func (e *SQLiteExtractor) extractForeignKeys(db *sql.DB, tableName string) ([]ForeignKey, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list('%s')", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query foreign keys: %w", err)
	}
	defer rows.Close()

	var fks []ForeignKey
	for rows.Next() {
		var id, seq int
		var col, refTable, refCol string
		var onUpdate, onDelete string
		if err := rows.Scan(&id, &seq, &col, &refTable, &refCol, &onUpdate, &onDelete); err != nil {
			return nil, err
		}
		fk := ForeignKey{
			Name:       fmt.Sprintf("fk_%s_%d", tableName, id),
			Columns:    []string{col},
			RefTable:   refTable,
			RefColumns: []string{refCol},
			OnDelete:   onDelete,
			OnUpdate:   onUpdate,
		}
		fks = append(fks, fk)
	}
	return fks, nil
}

// Helper function
func parseIndexColumns(indexDef, tableName string) []string {
	var cols []string
	// Extract columns from index definition
	// Format: CREATE [UNIQUE] INDEX name ON table (col1, col2, ...)
	start := strings.Index(indexDef, "(")
	end := strings.LastIndex(indexDef, ")")
	if start != -1 && end != -1 {
		colStr := indexDef[start+1 : end]
		parts := strings.Split(colStr, ",")
		for _, p := range parts {
			cols = append(cols, strings.TrimSpace(p))
		}
	}
	return cols
}