package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	snapshotCmd = &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snap"},
		Short:   "Create a snapshot of your database",
		Run: func(_ *cobra.Command, args []string) {
			createSnapshot(args)
		},
	}

	// Hidden command to create snapshot copy in background
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

func createSnapshot(args []string) {
	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first")
		return
	}

	if len(args) != 1 {
		fmt.Println("Please provide a name for the snapshot. Like `lunar snapshot production` or `lunar snapshot staging`")
		return
	}

	snapshotName := args[0]
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

	if err := snapshotManager.CheckIfSnapshotCanBeTaken(snapshotName); err != nil {
		fmt.Println(err)
		return
	}

	message := fmt.Sprintf("Creating a snapshot for the database %s", config.DatabaseName)
	stopSpinner := StartSpinner(message)

	if err := snapshotManager.StartSnapshotprocess(snapshotName); err != nil {
		stopSpinner()
		fmt.Printf("Error creating snapshot: %v\n", err)
		return
	}

	stopSpinner()
	fmt.Println("Snapshot created successfully")

	// Spawn a background process to create the _copy database
	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Warning: Could not prepare snapshot for fast restore: %v\n", err)
		return
	}

	cmd := exec.Command(executable, "snapshot", "create-copy", snapshotName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		fmt.Printf("Warning: Could not prepare snapshot for fast restore: %v\n", err)
	}
}

// createSnapshotCopy is called as a subprocess to create the _copy database
func createSnapshotCopy(args []string) {
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
