package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "schema-sync",
	Short: "Database schema synchronization and migration tool",
	Long:  `SchemaSync compares database schemas across environments and generates migration files.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd, snapshotCmd, diffCmd, migrateCmd, statusCmd)
}