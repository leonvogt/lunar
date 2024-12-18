package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	restoreCmd = &cobra.Command{
		Use:   "restore",
		Short: "Restore a snapshot of your database",
		Run: func(_ *cobra.Command, args []string) {
			restoreSnapshot(args)
		},
	}
)

func restoreSnapshot(args []string) {
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

	message := fmt.Sprintf("Restoring snapshot %s (%s) for database %s", snapshotName, snapshotDatabaseName, config.DatabaseName)
	stopSpinner := StartSpinner(message)

	internal.TerminateAllCurrentConnections(config.DatabaseName)
	internal.TerminateAllCurrentConnections(snapshotDatabaseName)
	internal.RestoreSnapshot(config.DatabaseName, snapshotDatabaseName)

	done := stopSpinner()
	<-done

	fmt.Println("Snapshot restored successfully")
}
