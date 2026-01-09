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
			if err := removeSnapshot(args); err != nil {
				fmt.Println(err)
			}
		},
	}
)

func removeSnapshot(args []string) error {
	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		if len(args) == 1 {
			snapshotName := args[0]
			if err := manager.CheckIfSnapshotExists(snapshotName); err != nil {
				return err
			}
			return removeSnapshotByName(manager, snapshotName)
		}

		snapshots, err := manager.ListSnapshots()
		if err != nil {
			return fmt.Errorf("error listing snapshots: %v", err)
		}

		if len(snapshots) == 0 {
			fmt.Println("No snapshots found.")
			return nil
		}

		snapshotNames := make([]string, len(snapshots))
		for i, snapshot := range snapshots {
			snapshotNames[i] = snapshot.Name
		}

		prompt := selection.New("Please select a snapshot to remove:", snapshotNames)
		prompt.PageSize = 50

		selectedSnapshot, err := prompt.RunPrompt()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		return removeSnapshotByName(manager, selectedSnapshot)
	})
}

func removeSnapshotByName(manager *internal.Manager, snapshotName string) error {
	fmt.Printf("Removing snapshot %s...\n", snapshotName)

	if err := manager.RemoveSnapshot(snapshotName); err != nil {
		return fmt.Errorf("error removing snapshot: %v", err)
	}

	fmt.Println("Snapshot removed successfully")
	return nil
}
