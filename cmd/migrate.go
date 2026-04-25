package cmd

import (
	"fmt"
	"os"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/fuleinist/schema-sync/internal/migrate"
	"github.com/fuleinist/schema-sync/internal/schema"
	"github.com/spf13/cobra"
)

var (
	dryRun bool
)

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

		// Placeholder: create empty migration if needed
		gen := migrate.NewGenerator(cfg.Settings.OutputDir)
		diffResult := &schema.DiffResult{}

		if dryRun {
			// Write SQL to stdout instead of file
			sql, err := gen.GenerateMigrationSQL(env, diffResult)
			if err != nil {
				return fmt.Errorf("generate migration SQL: %w", err)
			}
			fmt.Println(sql)
			return nil
		}

		path, err := gen.GenerateMigration(env, diffResult)
		if err != nil {
			return fmt.Errorf("generate migration: %w", err)
		}

		fmt.Printf("Migration file: %s\n", path)
		return nil
	},
}

func init() {
	migrateCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview migration SQL to stdout instead of writing to file")
}