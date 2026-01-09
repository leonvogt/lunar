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
			if err := replaceSnapshot(args); err != nil {
				fmt.Println(err)
			}
		},
	}
)

func replaceSnapshot(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please provide a snapshot name")
	}

	snapshotName := args[0]

	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		if manager.IsWaitingForOperation() {
			stopWaitSpinner := StartSpinner("Currently there is a Lunar background operation running. Waiting for it to complete before replacing the snapshot...")
			if err := manager.WaitForOngoingOperations(); err != nil {
				stopWaitSpinner()
				return fmt.Errorf("failed to wait for ongoing operation: %v", err)
			}
			stopWaitSpinner()
		}

		message := fmt.Sprintf("Replacing snapshot %s for database %s", snapshotName, config.DatabaseName)
		stopSpinner := StartSpinner(message)

		if err := manager.ReplaceSnapshot(snapshotName); err != nil {
			stopSpinner()
			return fmt.Errorf("error replacing snapshot: %v", err)
		}

		stopSpinner()
		fmt.Println("Snapshot replaced successfully")

		if err := spawnBackgroundCommand("snapshot", "create-copy", snapshotName); err != nil {
			fmt.Printf("Warning: Could not prepare snapshot for fast restore: %v\n", err)
		}

		return nil
	})
}
