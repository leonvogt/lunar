package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "Lunar",
	Short: "A database snapshot tool for PostgreSQL databases.",
	Long:  "Use Lunar to create and restore database snapshots for PostgreSQL databases. \nRun 'lunar --help' for more information.",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cfgFile := "lunar.yml"
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./lunar.yml)")
}
