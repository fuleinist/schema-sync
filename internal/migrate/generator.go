package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fuleinist/schema-sync/internal/diff"
	"github.com/fuleinist/schema-sync/internal/schema"
)

// Generator handles migration file generation
type Generator struct {
	outputDir string
	dryRun    bool
}

// NewGenerator creates a new migration generator
func NewGenerator(outputDir string, dryRun bool) *Generator {
	return &Generator{outputDir: outputDir, dryRun: dryRun}
}

// GenerateMigration creates a migration file from diff results
func (g *Generator) GenerateMigration(env string, diff *schema.DiffResult) (string, error) {
	var sb strings.Builder

	// Build migration SQL
	sb.WriteString("-- +migrate Up\n")
	sb.WriteString("-- +migrate StatementBegin\n")
	g.writeUpMigration(&sb, diff)
	sb.WriteString("-- +migrate StatementEnd\n\n")

	sb.WriteString("-- +migrate Down\n")
	sb.WriteString("-- +migrate StatementBegin\n")
	g.writeDownMigration(&sb, diff)
	sb.WriteString("-- +migrate StatementEnd\n")

	output := sb.String()

	if g.dryRun {
		fmt.Print(output)
		return "", nil
	}

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

	f.WriteString(output)

	return filepath, nil
}

func (g *Generator) writeUpMigration(sb *strings.Builder, diff *schema.DiffResult) {
	// Added tables
	for _, t := range diff.Added {
		sb.WriteString(g.generateCreateTable(t))
	}

	// Modified tables
	for _, td := range diff.Modified {
		g.writeTableChangesUp(sb, td)
	}

	// Removed tables (only in DOWN, not UP)
	_ = sb
}

func (g *Generator) writeDownMigration(sb *strings.Builder, diff *schema.DiffResult) {
	// Removed tables
	for _, t := range diff.Removed {
		sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", t.Name))
	}

	// Modified tables - rollback in reverse
	for _, td := range diff.Modified {
		g.writeTableChangesDown(sb, td)
	}

	// Added tables - drop them
	for _, t := range diff.Added {
		sb.WriteString(fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", t.Name))
	}
}

func (g *Generator) writeTableChangesUp(sb *strings.Builder, td schema.TableDiff) {
	// Added columns
	for _, col := range td.AddedColumns {
		stmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", td.TableName, col.Name, col.Type)
		if !col.Nullable {
			stmt += " NOT NULL"
		}
		if col.Default != nil {
			stmt += fmt.Sprintf(" DEFAULT %s", *col.Default)
		}
		sb.WriteString(stmt + ";\n")
	}

	// Modified columns
	for _, mc := range td.ModifiedColumns {
		stmt := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s", td.TableName, mc.Name, mc.NewType)
		sb.WriteString(stmt + ";\n")
		if mc.OldNull != mc.NewNull {
			if mc.NewNull {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;\n", td.TableName, mc.Name))
			} else {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;\n", td.TableName, mc.Name))
			}
		}
		// Default-value changes were previously dropped here: the diff
		// records them (diff.compareTables) and cmd.printDiff renders
		// them, but the UP migration never emitted a SET/DROP DEFAULT
		// statement, so applying a migration whose only column change
		// was the default would silently leave the column's default
		// untouched. Use the canonical equality check so a nil default
		// is treated the same way as in the diff path.
		if !diff.EqualDefault(mc.OldDefault, mc.NewDefault) {
			if mc.NewDefault == nil {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;\n", td.TableName, mc.Name))
			} else {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;\n", td.TableName, mc.Name, *mc.NewDefault))
			}
		}
	}

	// Added indexes
	for _, idx := range td.AddedIndexes {
		sb.WriteString(g.generateCreateIndex(td.TableName, idx))
	}

	// Added foreign keys
	for _, fk := range td.AddedFKs {
		sb.WriteString(g.generateAddFK(td.TableName, fk))
	}
}

func (g *Generator) writeTableChangesDown(sb *strings.Builder, td schema.TableDiff) {
	// Drop added foreign keys
	for _, fk := range td.AddedFKs {
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s;\n", td.TableName, fk.Name))
	}

	// Drop added indexes
	for _, idx := range td.AddedIndexes {
		sb.WriteString(fmt.Sprintf("DROP INDEX IF EXISTS %s;\n", idx.Name))
	}

	// Drop added columns
	for _, col := range td.AddedColumns {
		sb.WriteString(fmt.Sprintf("ALTER TABLE %s DROP COLUMN IF EXISTS %s;\n", td.TableName, col.Name))
	}

	// Revert modified columns. The UP migration applies Type, Nullable,
	// and Default changes via writeTableChangesUp; DOWN must apply the
	// inverse of each. Without this a rollback left the schema drifted
	// from the pre-migration shape even though no other diff entries
	// were present.
	for _, mc := range td.ModifiedColumns {
		if mc.OldType != mc.NewType {
			sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s;\n", td.TableName, mc.Name, mc.OldType))
		}
		if mc.OldNull != mc.NewNull {
			if mc.OldNull {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP NOT NULL;\n", td.TableName, mc.Name))
			} else {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET NOT NULL;\n", td.TableName, mc.Name))
			}
		}
		if !diff.EqualDefault(mc.OldDefault, mc.NewDefault) {
			if mc.OldDefault == nil {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s DROP DEFAULT;\n", td.TableName, mc.Name))
			} else {
				sb.WriteString(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s SET DEFAULT %s;\n", td.TableName, mc.Name, *mc.OldDefault))
			}
		}
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