package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/ui"
	"github.com/spf13/cobra"
)

var (
	replaceCmd = &cobra.Command{
		Use:   "replace [snapshot]",
		Short: "Replaces a snapshot (Delete previously existing snapshot and create a new one with the same name)",
		Run: func(_ *cobra.Command, args []string) {
			if err := replaceSnapshot(args); err != nil {
				fmt.Println(err)
			}
		},
	}
)

func replaceSnapshot(args []string) error {
	return withSnapshotManager(func(manager *internal.Manager, config *internal.Config) error {
		snapshotName, err := getSnapshotNameFromArgsOrPrompt(args, manager, "Please select a snapshot to replace:")
		if err != nil {
			return err
		}

		if manager.IsWaitingForOperation() {
			stopWaitSpinner := ui.StartSpinner("Currently there is a Lunar background operation running. Waiting for it to complete before replacing the snapshot...")
			if err := manager.WaitForOngoingOperations(); err != nil {
				stopWaitSpinner()
				return fmt.Errorf("failed to wait for ongoing operation: %v", err)
			}
			stopWaitSpinner()
		}

		message := fmt.Sprintf("Replacing snapshot %s for database %s", snapshotName, manager.GetDatabaseIdentifier())
		setInfo, stopSpinner := ui.StartDynamicSpinner(message)

		if size, err := manager.GetDatabaseSize(); err == nil && size > 0 {
			setInfo(ui.FormatBytes(size))
		}

		if err := manager.ReplaceSnapshot(snapshotName); err != nil {
			stopSpinner()
			return fmt.Errorf("error replacing snapshot: %v", err)
		}

		elapsed := stopSpinner()
		fmt.Printf("Snapshot replaced successfully in %s\n", ui.FormatDuration(elapsed))

		if err := spawnBackgroundCommand("snapshot", "create-copy", snapshotName); err != nil {
			fmt.Printf("Warning: Could not prepare snapshot for fast restore: %v\n", err)
		}

		return nil
	})
}
