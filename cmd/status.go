package cmd

import (
	"fmt"
	"os"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show tracked environments and last sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, _ := os.Getwd()
		cfg, err := config.Load(dir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No schema-sync configuration found. Run 'schema-sync init' first.")
				return nil
			}
			return fmt.Errorf("load config: %w", err)
		}

		fmt.Println("Tracked Environments:")
		if len(cfg.Environments) == 0 {
			fmt.Println("  (none)")
		} else {
			for name, env := range cfg.Environments {
				fmt.Printf("  %s: %s\n", name, env.DBType)
			}
		}

		fmt.Printf("\nSnapshot dir: %s\n", cfg.Settings.SnapshotDir)
		fmt.Printf("Output dir: %s\n", cfg.Settings.OutputDir)

		return nil
	},
}