package schema

import (
	"database/sql"
)

// Extractor defines the interface for extracting schema from a database
type Extractor interface {
	Extract(db *sql.DB) (*Schema, error)
	DBType() string
}

// NewExtractor creates an extractor for the given database type
func NewExtractor(dbType string) Extractor {
	switch dbType {
	case "postgres":
		return &PostgresExtractor{}
	case "mysql":
		return &MySQLExtractor{}
	case "sqlite":
		return &SQLiteExtractor{}
	default:
		return nil
	}
}