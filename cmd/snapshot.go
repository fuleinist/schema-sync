package cmd

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fuleinist/schema-sync/internal/config"
	"github.com/fuleinist/schema-sync/internal/schema"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot <env>",
	Short: "Capture current database schema snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		env := args[0]

		// Load config
		dir, _ := os.Getwd()
		cfg, err := config.Load(dir)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		envConfig, exists := cfg.Environments[env]
		if !exists {
			return fmt.Errorf("environment %q not found in config", env)
		}

		// Connect to database
		db, err := sql.Open(envConfig.DBType, envConfig.DSN)
		if err != nil {
			return fmt.Errorf("connect to database: %w", err)
		}
		defer db.Close()

		if err := db.Ping(); err != nil {
			return fmt.Errorf("ping database: %w", err)
		}

		// Extract schema
		extractor := schema.NewExtractor(envConfig.DBType)
		if extractor == nil {
			return fmt.Errorf("unsupported database type: %s", envConfig.DBType)
		}

		s, err := extractor.Extract(db)
		if err != nil {
			return fmt.Errorf("extract schema: %w", err)
		}

		// Save snapshot
		snapshotDir := filepath.Join(dir, cfg.Settings.SnapshotDir)
		os.MkdirAll(snapshotDir, 0755)

		snapshotFile := filepath.Join(snapshotDir, fmt.Sprintf("%s.json", env))
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal schema: %w", err)
		}

		if err := os.WriteFile(snapshotFile, data, 0644); err != nil {
			return fmt.Errorf("write snapshot: %w", err)
		}

		fmt.Printf("Snapshot saved: %s\n", snapshotFile)
		return nil
	},
}