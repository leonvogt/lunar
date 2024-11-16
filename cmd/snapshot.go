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

var snapshotManager, _ = internal.NewSnapshotManager()

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
	config, _ := internal.ReadConfig()
	snapshotDatabaseName := internal.SnapshotDatabaseName(config.DatabaseName, snapshotName)

	if internal.DoesDatabaseExists(snapshotDatabaseName) {
		fmt.Println("Snapshot with name", snapshotName, "already exists")
		return
	}

	if snapshotManager.IsSnapshotInProgress(snapshotName) {
		fmt.Println("Waiting for ongoing snapshot to complete...")
		if err := snapshotManager.WaitForOngoingSnapshot(snapshotName, 30*time.Minute); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	message := fmt.Sprintf("Creating a snapshot for the database %s", config.DatabaseName)
	stopSpinner := StartSpinner(message)

	if err := snapshotManager.MarkSnapshotStart(snapshotName); err != nil {
		fmt.Printf("Error starting snapshot: %v\n", err)
		return
	}

	if err := internal.CreateSnapshot(config.DatabaseName, snapshotDatabaseName); err != nil {
		fmt.Printf("Error creating initial snapshot: %v\n", err)
		snapshotManager.MarkSnapshotFinish(snapshotName)
		return
	}

	// Channel to track completion of background task
	done := make(chan bool)
	var copyErr error

	// Run the snapshot copy in the background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				copyErr = fmt.Errorf("panic in background copy: %v", r)
			}
			snapshotManager.MarkSnapshotFinish(snapshotName)
			done <- true
		}()

		if err := internal.CreateSnapshot(config.DatabaseName, snapshotDatabaseName+"_copy"); err != nil {
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
	fmt.Println("Snapshot created successfully")
}
