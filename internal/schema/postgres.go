package schema

import (
	"database/sql"
	"fmt"
	"strings"
)

// PostgresExtractor extracts schema from PostgreSQL
type PostgresExtractor struct{}

func (e *PostgresExtractor) DBType() string {
	return "postgres"
}

func (e *PostgresExtractor) Extract(db *sql.DB) (*Schema, error) {
	tables, err := e.extractTables(db)
	if err != nil {
		return nil, err
	}
	return &Schema{Tables: tables}, nil
}

func (e *PostgresExtractor) extractTables(db *sql.DB) ([]Table, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`
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

func (e *PostgresExtractor) extractColumns(db *sql.DB, tableName string) ([]Column, error) {
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var name, dataType, nullable string
		var defaultVal sql.NullString
		if err := rows.Scan(&name, &dataType, &nullable, &defaultVal); err != nil {
			return nil, err
		}
		col := Column{
			Name:     name,
			Type:     dataType,
			Nullable: nullable == "YES",
			Primary:  e.isPrimaryKey(db, tableName, name),
			Unique:   e.isUnique(db, tableName, name),
		}
		if defaultVal.Valid {
			col.Default = &defaultVal.String
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (e *PostgresExtractor) isPrimaryKey(db *sql.DB, tableName, colName string) bool {
	query := `
		SELECT 1 FROM information_schema.key_column_usage
		WHERE table_name = $1 AND column_name = $2 AND constraint_name IN (
			SELECT constraint_name FROM information_schema.table_constraints
			WHERE table_name = $1 AND constraint_type = 'PRIMARY KEY'
		)
	`
	var exists int
	db.QueryRow(query, tableName, colName).Scan(&exists)
	return exists == 1
}

func (e *PostgresExtractor) isUnique(db *sql.DB, tableName, colName string) bool {
	query := `
		SELECT 1 FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage ccu
		ON tc.constraint_name = ccu.constraint_name
		WHERE tc.table_name = $1 AND ccu.column_name = $2 AND tc.constraint_type = 'UNIQUE'
	`
	var exists int
	db.QueryRow(query, tableName, colName).Scan(&exists)
	return exists == 1
}

func (e *PostgresExtractor) extractIndexes(db *sql.DB, tableName string) ([]Index, error) {
	query := `
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE tablename = $1 AND schemaname = 'public'
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("query indexes: %w", err)
	}
	defer rows.Close()

	var indexes []Index
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			return nil, err
		}
		// Parse column names from indexdef
		cols := parseIndexColumns(def, tableName)
		isUnique := strings.Contains(def, "UNIQUE")
		indexes = append(indexes, Index{
			Name:    name,
			Columns: cols,
			Unique:  isUnique,
		})
	}
	return indexes, nil
}

func (e *PostgresExtractor) extractForeignKeys(db *sql.DB, tableName string) ([]ForeignKey, error) {
	query := `
		SELECT
			constraint_name,
			column_name,
			foreign_table_name,
			foreign_column_name,
			DELETE_RULE,
			UPDATE_RULE
		FROM information_schema.key_column_usage
		WHERE table_name = $1 AND foreign_table_name IS NOT NULL
	`
	rows, err := db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("query foreign keys: %w", err)
	}
	defer rows.Close()

	// Group by constraint name
	fkMap := make(map[string]ForeignKey)
	for rows.Next() {
		var conName, col, refTable, refCol, onDel, onUpd string
		if err := rows.Scan(&conName, &col, &refTable, &refCol, &onDel, &onUpd); err != nil {
			return nil, err
		}
		fk, ok := fkMap[conName]
		if !ok {
			fk = ForeignKey{Name: conName, OnDelete: onDel, OnUpdate: onUpd}
		}
		fk.Columns = append(fk.Columns, col)
		fk.RefColumns = append(fk.RefColumns, refCol)
		fk.RefTable = refTable
		fkMap[conName] = fk
	}

	var fks []ForeignKey
	for _, fk := range fkMap {
		fks = append(fks, fk)
	}
	return fks, nil
}