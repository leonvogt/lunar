package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var databaseUrlFlag string
var databaseNameFlag string
var databasePathFlag string
var snapshotDirectoryFlag string
var providerFlag string

var rootCmd = &cobra.Command{
	Use:     "lunar",
	Version: "0.1.0-rc.2",
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
	initCmd.Flags().StringVar(&providerFlag, "provider", "", "Database provider to use: 'postgres' or 'sqlite'.")
	initCmd.Flags().StringVarP(&databaseUrlFlag, "database-url", "u", "", "The connection URL to your PostgreSQL database.")
	initCmd.Flags().StringVarP(&databaseNameFlag, "database-name", "d", "", "The name of the database you want to snapshot.")
	initCmd.Flags().StringVar(&databasePathFlag, "database-path", "", "Path to the SQLite database file.")
	initCmd.Flags().StringVar(&snapshotDirectoryFlag, "snapshot-directory", "", "Directory to store SQLite snapshots.")

	rootCmd.AddCommand(snapshotCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(replaceCmd)
}
