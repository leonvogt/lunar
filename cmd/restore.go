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
	snapshotManager, err := internal.SnapshotManager(config)
	if err != nil {
		fmt.Printf("Error initializing snapshot manager: %v\n", err)
		return
	}
	defer snapshotManager.Close()

	if err := snapshotManager.CheckIfSnapshotExists(snapshotName); err != nil {
		fmt.Println(err)
		return
	}

	message := fmt.Sprintf("Restoring snapshot %s for database %s", snapshotName, config.DatabaseName)
	stopSpinner := StartSpinner(message)

	// Restore the snapshot using the snapshot manager
	if err := snapshotManager.RestoreSnapshot(snapshotName); err != nil {
		stopSpinner()
		fmt.Printf("Error restoring snapshot: %v\n", err)
		return
	}

	stopSpinner()
	fmt.Println("Snapshot restored successfully")
}
