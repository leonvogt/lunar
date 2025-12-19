package cmd

import (
	"fmt"
	"os"

	"github.com/erikgeiser/promptkit/selection"
	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	removeCmd = &cobra.Command{
		Use:     "remove",
		Aliases: []string{"drop", "delete"},
		Short:   "Removes a snapshot",
		Run: func(_ *cobra.Command, args []string) {
			removeSnapshot(args)
		},
	}
)

func removeSnapshot(args []string) {
	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first.")
		return
	}

	config, err := internal.ReadConfig()
	if err != nil {
		fmt.Printf("Error reading config: %v\n", err)
		return
	}

	snapshotManager, err := internal.SnapshotManager(config)
	if err != nil {
		fmt.Printf("Error initializing snapshot manager: %v\n", err)
		return
	}
	defer snapshotManager.Close()

	// If a snapshot name is provided as argument, remove it directly
	if len(args) == 1 {
		snapshotName := args[0]
		if err := snapshotManager.CheckIfSnapshotExists(snapshotName); err != nil {
			fmt.Println(err)
			return
		}
		removeSnapshotByName(snapshotManager, snapshotName)
		return
	}

	// Otherwise, show selection interface
	snapshots, err := snapshotManager.ListSnapshots()
	if err != nil {
		fmt.Printf("Error listing snapshots: %v\n", err)
		return
	}

	if len(snapshots) == 0 {
		fmt.Println("No snapshots found.")
		return
	}

	sp := selection.New("Please select a snapshot to remove:", snapshots)
	sp.PageSize = 50

	selectedSnapshot, err := sp.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	removeSnapshotByName(snapshotManager, selectedSnapshot)
}

func removeSnapshotByName(snapshotManager *internal.Manager, snapshotName string) {
	fmt.Printf("Removing snapshot %s...\n", snapshotName)

	// Remove the main snapshot
	if err := snapshotManager.RemoveSnapshot(snapshotName); err != nil {
		fmt.Printf("Error removing snapshot: %v\n", err)
		return
	}

	// Also remove the _copy version if it exists
	copySnapshotName := snapshotName + "_copy"
	if err := snapshotManager.CheckIfSnapshotExists(copySnapshotName); err == nil {
		fmt.Printf("Also removing temporary snapshot %s...\n", copySnapshotName)
		if err := snapshotManager.RemoveSnapshot(copySnapshotName); err != nil {
			fmt.Printf("Warning: Could not remove temporary snapshot: %v\n", err)
		}
	}

	fmt.Println("Snapshot removed successfully")
}
