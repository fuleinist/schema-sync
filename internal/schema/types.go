package schema

// ChangeType represents the type of schema change
type ChangeType string

const (
	ChangeAdd    ChangeType = "ADD"
	ChangeDrop   ChangeType = "DROP"
	ChangeModify ChangeType = "MODIFY"
	ChangeRename ChangeType = "RENAME"
)

// ConstraintType represents the type of constraint
type ConstraintType string

const (
	ConstraintNotNull ConstraintType = "NOT NULL"
	ConstraintUnique  ConstraintType = "UNIQUE"
	ConstraintDefault ConstraintType = "DEFAULT"
	ConstraintCheck   ConstraintType = "CHECK"
	ConstraintPrimary ConstraintType = "PRIMARY KEY"
	ConstraintForeign ConstraintType = "FOREIGN KEY"
)

// Column represents a database column
type Column struct {
	Name     string
	Type     string
	Nullable bool
	Default  *string
	Primary  bool
	Unique   bool
}

// Index represents a database index
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name         string
	Columns      []string
	RefTable     string
	RefColumns   []string
	OnDelete     string
	OnUpdate     string
}

// Table represents a database table
type Table struct {
	Name        string
	Columns     []Column
	Indexes     []Index
	ForeignKeys []ForeignKey
}

// Schema represents a full database schema
type Schema struct {
	Tables []Table
}

// DiffItem represents a single schema change
type DiffItem struct {
	Type      ChangeType
	Entity    string // "table", "column", "index", "fk", "constraint"
	Table     string
	Name      string
	OldValue  interface{}
	NewValue  interface{}
}

// DiffResult holds the complete diff between two schemas
type DiffResult struct {
	Added    []Table
	Removed  []Table
	Modified []TableDiff
}

// TableDiff holds column/index/fk changes for a modified table
type TableDiff struct {
	TableName    string
	AddedColumns []Column
	DroppedColumns []Column
	ModifiedColumns []ColumnChange
	AddedIndexes   []Index
	DroppedIndexes []Index
	AddedFKs       []ForeignKey
	DroppedFKs     []ForeignKey
}

// ColumnChange represents a column modification
type ColumnChange struct {
	Name      string
	OldType   string
	NewType   string
	OldNull   bool
	NewNull   bool
	OldDefault *string
	NewDefault *string
}