package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/ui"
	"github.com/spf13/cobra"
)

var (
	restoreCmd = &cobra.Command{
		Use:   "restore [snapshot]",
		Short: "Restore a snapshot of your database",
		Run: func(_ *cobra.Command, args []string) {
			if err := restoreSnapshot(args); err != nil {
				fmt.Println(err)
			}
		},
	}

	recreateCopyCmd = &cobra.Command{
		Use:    "recreate-copy",
		Hidden: true,
		Run: func(_ *cobra.Command, args []string) {
			recreateSnapshotCopy(args)
		},
	}
)

func init() {
	restoreCmd.AddCommand(recreateCopyCmd)
}

func restoreSnapshot(args []string) error {
	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		snapshotName, err := getSnapshotNameFromArgsOrPrompt(args, manager, "Please select a snapshot to restore:")
		if err != nil {
			return err
		}

		// Check and wait for any ongoing operations
		if manager.IsWaitingForOperation() {
			stopWaitSpinner := ui.StartSpinner("Currently there is a Lunar background operation running. Waiting for it to complete before restoring the snapshot...")
			if err := manager.WaitForOngoingOperations(); err != nil {
				stopWaitSpinner()
				return fmt.Errorf("failed to wait for ongoing operation: %v", err)
			}
			stopWaitSpinner()
		}

		message := fmt.Sprintf("Restoring snapshot %s for database %s", snapshotName, manager.GetDatabaseIdentifier())
		stopSpinner := ui.StartSpinner(message)

		if err := manager.RestoreSnapshot(snapshotName); err != nil {
			stopSpinner()
			return fmt.Errorf("error restoring snapshot: %v", err)
		}

		stopSpinner()
		fmt.Println("Snapshot restored successfully")

		if err := spawnBackgroundCommand("restore", "recreate-copy", snapshotName); err != nil {
			fmt.Printf("Warning: Could not prepare snapshot for next restore: %v\n", err)
		}

		return nil
	})
}

func recreateSnapshotCopy(args []string) {
	if len(args) != 1 {
		return
	}

	snapshotName := args[0]
	config, err := internal.ReadConfig()
	if err != nil {
		return
	}

	snapshotManager, err := internal.NewSnapshotManager(config)
	if err != nil {
		return
	}
	defer snapshotManager.Close()

	snapshotManager.CreateSnapshotCopy(snapshotName)
}
