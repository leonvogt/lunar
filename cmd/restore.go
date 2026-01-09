package cmd

import (
	"fmt"
	"os"
	"os/exec"

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

	// Hidden command to recreate snapshot copy in background
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

	if err := snapshotManager.RestoreSnapshot(snapshotName); err != nil {
		stopSpinner()
		fmt.Printf("Error restoring snapshot: %v\n", err)
		return
	}

	stopSpinner()
	fmt.Println("Snapshot restored successfully")

	// Spawn a background process to recreate the _copy database
	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Warning: Could not prepare snapshot for next restore: %v\n", err)
		return
	}

	cmd := exec.Command(executable, "restore", "recreate-copy", snapshotName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		fmt.Printf("Warning: Could not prepare snapshot for next restore: %v\n", err)
	}
}

// recreateSnapshotCopy is called as a subprocess to recreate the _copy database
func recreateSnapshotCopy(args []string) {
	if len(args) != 1 {
		return
	}

	snapshotName := args[0]
	config, err := internal.ReadConfig()
	if err != nil {
		return
	}

	snapshotManager, err := internal.SnapshotManager(config)
	if err != nil {
		return
	}
	defer snapshotManager.Close()

	snapshotManager.RecreateSnapshotCopy(snapshotName)
}
