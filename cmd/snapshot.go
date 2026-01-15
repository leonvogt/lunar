package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	snapshotCmd = &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snap"},
		Short:   "Create a snapshot of your database",
		Run: func(_ *cobra.Command, args []string) {
			if err := createSnapshot(args); err != nil {
				fmt.Println(err)
			}
		},
	}

	createCopyCmd = &cobra.Command{
		Use:    "create-copy",
		Hidden: true,
		Run: func(_ *cobra.Command, args []string) {
			createSnapshotCopy(args)
		},
	}
)

func init() {
	snapshotCmd.AddCommand(createCopyCmd)
}

func createSnapshot(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("please provide a name for the snapshot. Like `lunar snapshot production` or `lunar snapshot staging`")
	}

	snapshotName := args[0]

	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		if err := manager.CheckIfSnapshotCanBeTaken(snapshotName); err != nil {
			return err
		}

		// Check and wait for any ongoing operations
		if manager.IsWaitingForOperation() {
			stopWaitSpinner := StartSpinner("Currently there is a Lunar background operation running. Waiting for it to complete before creating the snapshot...")
			if err := manager.WaitForOngoingOperations(); err != nil {
				stopWaitSpinner()
				return fmt.Errorf("failed to wait for ongoing operation: %v", err)
			}
			stopWaitSpinner()
		}

		message := fmt.Sprintf("Creating a snapshot for the database %s", manager.GetDatabaseIdentifier())
		stopSpinner := StartSpinner(message)

		if err := manager.CreateMainSnapshot(snapshotName); err != nil {
			stopSpinner()
			return fmt.Errorf("error creating snapshot: %v", err)
		}

		stopSpinner()
		fmt.Println("Snapshot created successfully")

		if err := spawnBackgroundCommand("snapshot", "create-copy", snapshotName); err != nil {
			fmt.Printf("Warning: Could not prepare snapshot for fast restore: %v\n", err)
		}

		return nil
	})
}

func createSnapshotCopy(args []string) {
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
