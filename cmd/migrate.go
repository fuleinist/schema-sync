package cmd

import (
	"fmt"
	"os"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/fuleinist/schema-sync/internal/migrate"
	"github.com/fuleinist/schema-sync/internal/schema"
	"github.com/spf13/cobra"
)

// dryRun outputs migration SQL to stdout instead of writing to a file
var dryRun bool

var migrateCmd = &cobra.Command{
	Use:   "migrate <env>",
	Short: "Generate migration file for target environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env := args[0]

		dir, _ := os.Getwd()
		cfg, err := config.Load(dir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// TODO: Load previous snapshot for comparison
		// Get current snapshot
		_, err = loadSnapshot(dir, cfg.Settings.SnapshotDir, env)
		if err != nil {
			return fmt.Errorf("load current snapshot: %w", err)
		}

		gen := migrate.NewGenerator(cfg.Settings.OutputDir, dryRun)
		diffResult := &schema.DiffResult{}
		path, err := gen.GenerateMigration(env, diffResult)
		if err != nil {
			return fmt.Errorf("generate migration: %w", err)
		}

		if dryRun {
			fmt.Println("\n[DRY-RUN] Migration SQL above. No file written.")
		} else {
			fmt.Printf("Migration file: %s\n", path)
		}
		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Output migration SQL to stdout instead of writing to file")
}

func loadPreviousSnapshot(env string) (*schema.Schema, error) {
	return nil, fmt.Errorf("previous snapshot not available")
}