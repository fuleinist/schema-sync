package cmd

import (
	"fmt"
	"os"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/fuleinist/schema-sync/internal/diff"
	"github.com/fuleinist/schema-sync/internal/migrate"
	"github.com/fuleinist/schema-sync/internal/schema"
	"github.com/spf13/cobra"
)

// dryRun outputs migration SQL to stdout instead of writing to a file
var dryRun bool

// fromEnv is the reference environment to compare against
var fromEnv string

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

		// Load target snapshot
		targetSchema, err := loadSnapshot(dir, cfg.Settings.SnapshotDir, env)
		if err != nil {
			return fmt.Errorf("load target snapshot %s: %w", env, err)
		}

		// Load reference snapshot
		refEnv := fromEnv
		if refEnv == "" {
			return fmt.Errorf("--from flag is required (e.g., --from dev); compare environments with 'schema-sync diff <env1> <env2>' first")
		}

		refSchema, err := loadSnapshot(dir, cfg.Settings.SnapshotDir, refEnv)
		if err != nil {
			return fmt.Errorf("load reference snapshot %s: %w", refEnv, err)
		}

		// Compute diff: ref -> target (changes to apply to ref to get target)
		diffResult := diff.ComputeDiff(refSchema, targetSchema)

		// Check if there are any changes
		if len(diffResult.Added) == 0 && len(diffResult.Removed) == 0 && len(diffResult.Modified) == 0 {
			if dryRun {
				fmt.Println("[DRY-RUN] No schema changes detected.")
			} else {
				fmt.Println("No schema changes detected.")
			}
			return nil
		}

		gen := migrate.NewGenerator(cfg.Settings.OutputDir, dryRun)
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
	migrateCmd.Flags().StringVar(&fromEnv, "from", "", "Reference environment snapshot to compare (required)")
}

// loadPreviousSnapshot is a stub — previous snapshot tracking not yet implemented
func loadPreviousSnapshot(env string) (*schema.Schema, error) {
	return nil, fmt.Errorf("previous snapshot not available")
}
