package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	removeCmd = &cobra.Command{
		Use:     "remove [snapshot]",
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
		snapshotName, err := getSnapshotNameFromArgsOrPrompt(args, manager, "Please select a snapshot to remove:")
		if err != nil {
			return err
		}

		return removeSnapshotByName(manager, snapshotName)
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
