package schema

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// MySQLExtractor extracts schema from MySQL
type MySQLExtractor struct{}

func (e *MySQLExtractor) DBType() string {
	return "mysql"
}

func (e *MySQLExtractor) Extract(db *sql.DB) (*Schema, error) {
	tables, err := e.extractTables(db)
	if err != nil {
		return nil, err
	}
	return &Schema{Tables: tables}, nil
}

func (e *MySQLExtractor) extractTables(db *sql.DB) ([]Table, error) {
	query := `SHOW TABLES`
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

func (e *MySQLExtractor) extractColumns(db *sql.DB, tableName string) ([]Column, error) {
	query := fmt.Sprintf("SHOW COLUMNS FROM `%s`", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	var columns []Column
	for rows.Next() {
		var field, colType, null, key, extra string
		var defaultVal sql.NullString
		if err := rows.Scan(&field, &colType, &null, &key, &defaultVal, &extra); err != nil {
			return nil, err
		}
		col := Column{
			Name:     field,
			Type:     colType,
			Nullable: null == "YES",
			Primary:  key == "PRI",
			Unique:   key == "UNI",
		}
		if defaultVal.Valid {
			col.Default = &defaultVal.String
		}
		columns = append(columns, col)
	}
	return columns, nil
}

func (e *MySQLExtractor) extractIndexes(db *sql.DB, tableName string) ([]Index, error) {
	query := fmt.Sprintf("SHOW INDEXES FROM `%s`", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query indexes: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]Index)
	for rows.Next() {
		var table, colName string
		var nonUnique int
		var idxName string
		var seq int
		if err := rows.Scan(&table, &idxName, &nonUnique, &colName, &seq, new(string), new(string), new(string), new(int), new(int)); err != nil {
			return nil, err
		}
		idx, ok := indexMap[idxName]
		if !ok {
			idx = Index{Name: idxName, Unique: nonUnique == 0}
		}
		idx.Columns = append(idx.Columns, colName)
		indexMap[idxName] = idx
	}

	var indexes []Index
	for _, idx := range indexMap {
		indexes = append(indexes, idx)
	}
	return indexes, nil
}

func (e *MySQLExtractor) extractForeignKeys(db *sql.DB, tableName string) ([]ForeignKey, error) {
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)
	var createSQL string
	if err := db.QueryRow(query).Scan(&createSQL); err != nil {
		return nil, fmt.Errorf("query foreign keys: %w", err)
	}

	return parseMySQLForeignKeys(createSQL), nil
}

// mysqlFKRe matches a MySQL FOREIGN KEY constraint line from SHOW CREATE TABLE output.
// Format:
//   CONSTRAINT `fk_name` FOREIGN KEY (`col1`, `col2`) REFERENCES `ref_table` (`ref_col1`, `ref_col2`) ON DELETE CASCADE ON UPDATE CASCADE
var mysqlFKRe = regexp.MustCompile(
	"CONSTRAINT `([^`]+)` FOREIGN KEY \\(([^)]+)\\) REFERENCES `([^`]+)` \\(([^)]+)\\)(?: ON DELETE ((?:CASCADE|RESTRICT|SET NULL|SET DEFAULT|NO ACTION)))?(?: ON UPDATE ((?:CASCADE|RESTRICT|SET NULL|SET DEFAULT|NO ACTION)))?",
)

func parseMySQLForeignKeys(createSQL string) []ForeignKey {
	var fks []ForeignKey
	lines := strings.Split(createSQL, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "FOREIGN KEY") {
			continue
		}
		m := mysqlFKRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		fk := ForeignKey{
			Name:       m[1],
			RefTable:   m[3],
			OnDelete:   m[5],
			OnUpdate:   m[6],
		}
		// Parse column lists: split on comma, trim backticks and whitespace
		for _, col := range strings.Split(m[2], ",") {
			fk.Columns = append(fk.Columns, strings.Trim(strings.TrimSpace(col), "`"))
		}
		for _, col := range strings.Split(m[4], ",") {
			fk.RefColumns = append(fk.RefColumns, strings.Trim(strings.TrimSpace(col), "`"))
		}
		fks = append(fks, fk)
	}
	return fks
}