package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all snapshots",
		Run: func(_ *cobra.Command, args []string) {
			listSnapshots()
		},
	}
)

func listSnapshots() {
	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first.")
		return
	}

	config, _ := internal.ReadConfig()
	snapshots := internal.SnapshotDatabasesForDatabase(config.DatabaseName)
	if len(snapshots) == 0 {
		fmt.Println("No snapshots found.")
		return
	}

	for _, snapshot := range snapshots {
		fmt.Println(snapshot)
	}
}
