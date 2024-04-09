package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "lunar",
	Version: "0.0.1",
	Short:   "A database snapshot tool for PostgreSQL databases.",
	Long:    "Use Lunar to create and restore database snapshots for PostgreSQL databases. \nRun 'lunar --help' for more information.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(restoreCmd)
}
