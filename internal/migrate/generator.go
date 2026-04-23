package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuleinist/schema-sync/internal/schema"
)

// Generator handles migration file generation
type Generator struct {
	outputDir string
}

// NewGenerator creates a new migration generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{outputDir: outputDir}
}

// GenerateMigration creates a migration file from diff results
func (g *Generator) GenerateMigration(env string, diff *schema.DiffResult) (string, error) {
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output dir: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, env)
	filepath := filepath.Join(g.outputDir, filename)

	f, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	// Write UP migration
	f.WriteString("-- +migrate Up\n")
	f.WriteString("-- +migrate StatementBegin\n")
	g.writeUpMigration(f, diff)
	f.WriteString("-- +migrate StatementEnd\n\n")

	// Write DOWN migration
	f.WriteString("-- +migrate Down\n")
	f.WriteString("-- +migrate StatementBegin\n")
	g.writeDownMigration(f, diff)
	f.WriteString("-- +migrate StatementEnd\n")

	return filepath, nil
}

func (g *Generator) writeUpMigration(f *os.File, diff *schema.DiffResult) {
	// Added tables
	for _, t := range diff.Added {
		f.WriteString(g.generateCreateTable(t))
	}

	// Modified tables
	for _, td := range diff.Modified {
		g.writeTableChangesUp(f, td)
	}

	// Removed tables (only in DOWN, not UP)
	_ = f
}

func (g *Generator) writeDownMigration(f *os.File, diff *schema.DiffResult) {
	// Removed tables
	for _, t := range diff.Removed {
		f.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", t.Name))
	}

	// Modified tables - rollback in reverse
	for _, td := range diff.Modified {
		g.writeTableChangesDown(f, td)
	}

	// Added tables - drop them
	for _, t := range diff.Added {
		f.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", t.Name))
	}
}

func (g *Generator) writeTableChangesUp(f *os.File, td schema.TableDiff) {
	// Added columns
	for _, col := range td.AddedColumns {
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", td.TableName, col.Name, col.Type)
		if !col.Nullable {
			stmt += " NOT NULL"
		}
		if col.Default != nil {
			stmt += fmt.Sprintf(" DEFAULT %s", *col.Default)
		}
		f.WriteString(stmt + ";\n")
	}

	// Modified columns
	for _, mc := range td.ModifiedColumns {
		stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", td.TableName, mc.Name, mc.NewType)
		f.WriteString(stmt + ";\n")
		if mc.OldNull != mc.NewNull {
			if mc.NewNull {
				f.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;\n", td.TableName, mc.Name))
			} else {
				f.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;\n", td.TableName, mc.Name))
			}
		}
	}

	// Added indexes
	for _, idx := range td.AddedIndexes {
		f.WriteString(g.generateCreateIndex(td.TableName, idx))
	}

	// Added foreign keys
	for _, fk := range td.AddedFKs {
		f.WriteString(g.generateAddFK(td.TableName, fk))
	}
}

func (g *Generator) writeTableChangesDown(f *os.File, td schema.TableDiff) {
	// Drop added foreign keys
	for _, fk := range td.AddedFKs {
		f.WriteString(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n", td.TableName, fk.Name))
	}

	// Drop added indexes
	for _, idx := range td.AddedIndexes {
		f.WriteString(fmt.Sprintf("DROP INDEX IF EXISTS %s;\n", idx.Name))
	}

	// Drop added columns
	for _, col := range td.AddedColumns {
		f.WriteString(fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;\n", td.TableName, col.Name))
	}
}

func (g *Generator) generateCreateTable(t schema.Table) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", t.Name))

	for i, col := range t.Columns {
		sb.WriteString(fmt.Sprintf("  %s %s", col.Name, col.Type))
		if !col.Nullable {
			sb.WriteString(" NOT NULL")
		}
		if col.Default != nil {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", *col.Default))
		}
		if col.Primary {
			sb.WriteString(" PRIMARY KEY")
		}
		if i < len(t.Columns)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString(");\n")
	return sb.String()
}

func (g *Generator) generateCreateIndex(tableName string, idx schema.Index) string {
	unique := ""
	if idx.Unique {
		unique = "UNIQUE "
	}
	cols := strings.Join(idx.Columns, ", ")
	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s);\n", unique, idx.Name, tableName, cols)
}

func (g *Generator) generateAddFK(tableName string, fk schema.ForeignKey) string {
	cols := strings.Join(fk.Columns, ", ")
	refCols := strings.Join(fk.RefColumns, ", ")
	return fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s) ON DELETE %s ON UPDATE %s;\n",
		tableName, fk.Name, cols, fk.RefTable, refCols, fk.OnDelete, fk.OnUpdate)
}