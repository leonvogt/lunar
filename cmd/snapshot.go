package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	snapshotCmd = &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snap"},
		Short:   "Create a snapshot of your database",
		Run: func(_ *cobra.Command, args []string) {
			createSnapshot(args)
		},
	}
)

func createSnapshot(args []string) {
	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first.")
		return
	}

	if len(args) != 1 {
		fmt.Println("Please provide a name for the snapshot.")
		return
	}

	snapshotName := args[0]
	config, _ := internal.ReadConfig()
	snapshotDatabaseName := internal.SnapshotDatabaseName(config.DatabaseName, snapshotName)
	fmt.Println("Creating snapshot for database", config.DatabaseName, "with name", snapshotName)

	internal.TerminateAllCurrentConnections(config.DatabaseName)
	internal.TerminateAllCurrentConnections(snapshotDatabaseName)
	internal.CreateSnapshot(config.DatabaseName, snapshotDatabaseName)
}
