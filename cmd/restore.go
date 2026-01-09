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
	if len(args) != 1 {
		return fmt.Errorf("please provide a snapshot name")
	}

	snapshotName := args[0]

	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		if err := manager.CheckIfSnapshotExists(snapshotName); err != nil {
			return err
		}

		message := fmt.Sprintf("Restoring snapshot %s for database %s", snapshotName, config.DatabaseName)
		stopSpinner := StartSpinner(message)

		status, err := manager.RestoreSnapshot(snapshotName)
		if err != nil {
			stopSpinner()
			return fmt.Errorf("error restoring snapshot: %v", err)
		}
		printWaitingStatus(status)

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
