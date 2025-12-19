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
	snapshotManager, err := internal.SnapshotManager(config)
	if err != nil {
		fmt.Printf("Error initializing snapshot manager: %v\n", err)
		return
	}
	defer snapshotManager.Close()

	message := fmt.Sprintf("Replacing snapshot %s for database %s", snapshotName, config.DatabaseName)
	stopSpinner := StartSpinner(message)

	// Replace the snapshot using the snapshot manager
	if err := snapshotManager.ReplaceSnapshot(snapshotName); err != nil {
		stopSpinner()
		fmt.Printf("Error replacing snapshot: %v\n", err)
		return
	}

	stopSpinner()
	fmt.Println("Snapshot replaced successfully")
}
