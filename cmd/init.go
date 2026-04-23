package cmd

import (
	"fmt"
	"os"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize schema-sync configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()

		cfg := config.DefaultConfig()
		if err := config.Save(dir, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Create output directories
		os.MkdirAll(cfg.Settings.OutputDir, 0755)
		os.MkdirAll(cfg.Settings.SnapshotDir, 0755)

		fmt.Println("Initialized schema-sync configuration")
		fmt.Println("  Config: .schema-sync/config.yaml")
		fmt.Println("  Migrations: " + cfg.Settings.OutputDir)
		fmt.Println("  Snapshots: " + cfg.Settings.SnapshotDir)

		return nil
	},
}