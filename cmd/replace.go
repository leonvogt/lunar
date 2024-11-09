package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	replaceCmd = &cobra.Command{
		Use:   "replace",
		Short: "Replaces a snapshot",
		Run: func(_ *cobra.Command, args []string) {
			replaceSnapshot(args)
		},
	}
)

func replaceSnapshot(args []string) {
	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first.")
		return
	}

	if len(args) != 1 {
		fmt.Println("Please provide a snapshot name.")
		return
	}

	snapshotName := args[0]
	config, _ := internal.ReadConfig()
	snapshotDatabaseName := internal.SnapshotDatabaseName(config.DatabaseName, snapshotName)

	// Remove the existing snapshot
	fmt.Println("Removing snapshot ", snapshotName)
	internal.TerminateAllCurrentConnections(snapshotDatabaseName)
	internal.DropDatabase(snapshotDatabaseName)

	// Create a new snapshot
	message := fmt.Sprintf("Creating a snapshot for the database %s", config.DatabaseName)
	stopSpinner := StartSpinner(message)

	internal.TerminateAllCurrentConnections(snapshotDatabaseName)
	internal.TerminateAllCurrentConnections(config.DatabaseName)
	internal.CreateSnapshot(config.DatabaseName, snapshotDatabaseName)

	done := stopSpinner()
	<-done

	fmt.Println("Snapshot created successfully")
}
