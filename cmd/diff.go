package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/fuleinist/schema-sync/internal/diff"
	"github.com/fuleinist/schema-sync/internal/schema"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <env1> <env2>",
	Short: "Compare schemas between two environments",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		env1, env2 := args[0], args[1]

		dir, _ := os.Getwd()
		cfg, err := config.Load(dir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		// Load snapshots
		schema1, err := loadSnapshot(dir, cfg.Settings.SnapshotDir, env1)
		if err != nil {
			return err
		}
		schema2, err := loadSnapshot(dir, cfg.Settings.SnapshotDir, env2)
		if err != nil {
			return err
		}

		// Compute diff
		result := diff.ComputeDiff(schema1, schema2)

		// Print diff
		printDiff(result)
		return nil
	},
}

func loadSnapshot(dir, snapshotDir, env string) (*schema.Schema, error) {
	// `dir` is the project root (typically `os.Getwd()`). The snapshot
	// writer in `cmd/snapshot.go` writes to `filepath.Join(dir, snapshotDir, ...)`,
	// so the reader must join the same way; using a bare relative path
	// (`snapshotDir + "/" + env + ".json"`) only works by accident when
	// the caller's CWD happens to equal `dir`.
	snapshotFile := filepath.Join(dir, snapshotDir, env+".json")
	data, err := os.ReadFile(snapshotFile)
	if err != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", env, err)
	}

	var s schema.Schema
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse snapshot %s: %w", env, err)
	}

	return &s, nil
}

func printDiff(result *schema.DiffResult) {
	if len(result.Added) == 0 && len(result.Removed) == 0 && len(result.Modified) == 0 {
		fmt.Println("No schema differences found.")
		return
	}

	if len(result.Added) > 0 {
		fmt.Println("=== Added Tables ===")
		for _, t := range result.Added {
			fmt.Printf("  + %s\n", t.Name)
		}
	}

	if len(result.Removed) > 0 {
		fmt.Println("=== Removed Tables ===")
		for _, t := range result.Removed {
			fmt.Printf("  - %s\n", t.Name)
		}
	}

	if len(result.Modified) > 0 {
		fmt.Println("=== Modified Tables ===")
		for _, td := range result.Modified {
			fmt.Printf("  ~ %s\n", td.TableName)
			for _, c := range td.AddedColumns {
				fmt.Printf("      + column: %s %s\n", c.Name, c.Type)
			}
			for _, c := range td.DroppedColumns {
				fmt.Printf("      - column: %s %s\n", c.Name, c.Type)
			}
			for _, mc := range td.ModifiedColumns {
				fmt.Printf("      ~ column: %s %s -> %s\n", mc.Name, mc.OldType, mc.NewType)
			}
		}
	}
}