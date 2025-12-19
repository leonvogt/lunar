package cmd

import (
	"fmt"
	"time"

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
)

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
	snapshotManager.StartSnapshotprocess(snapshotName)

	// Channel to track completion of background task
	done := make(chan bool)
	var copyErr error

	// Run the snapshot copy in the background
	snapshotDatabaseName := internal.SnapshotDatabaseName(config.DatabaseName, snapshotName)
	snapshotCopyDatabaseName := snapshotDatabaseName + "_copy"
	go func() {
		defer func() {
			if r := recover(); r != nil {
				copyErr = fmt.Errorf("panic in background copy: %v", r)
			}
			snapshotManager.MarkSnapshotFinish(snapshotName)
			done <- true
		}()

		if err := snapshotManager.CreateSnapshot(config.DatabaseName, snapshotCopyDatabaseName); err != nil {
			copyErr = err
			return
		}
	}()

	// Wait a short time to catch immediate failures
	select {
	case <-done:
		if copyErr != nil {
			fmt.Printf("Background copy failed immediately: %v\n", copyErr)
		}
	case <-time.After(2 * time.Second):
		// Continue if the background process hasn't failed quickly
	}

	stopSpinner()
}
