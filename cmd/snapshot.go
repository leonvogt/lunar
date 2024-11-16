package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var debugLog *log.Logger

func init() {
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "lunar-debug.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not create debug log: %v\n", err)
		return
	}

	debugLog = log.New(logFile, "", log.LstdFlags)
	rootCmd.AddCommand(snapshotCmd)
}

var snapshotManager, _ = internal.NewSnapshotManager(debugLog)

var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Aliases: []string{"snap"},
	Short:   "Create a snapshot of your database",
	Run: func(_ *cobra.Command, args []string) {
		createSnapshot(args)
	},
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

	if err := snapshotManager.StartSnapshot(snapshotName); err != nil {
		fmt.Printf("Error starting snapshot: %v\n", err)
		return
	}

	if err := internal.CreateSnapshotWithRetry(config.DatabaseName, snapshotDatabaseName, 3); err != nil {
		fmt.Printf("Error creating initial snapshot: %v\n", err)
		snapshotManager.FinishSnapshot(snapshotName)
		return
	}

	// Channel to track completion of background task
	done := make(chan bool)
	var copyErr error

	// Run the snapshot copy in the background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				debugLog.Printf("Panic in background copy: %v", r)
				copyErr = fmt.Errorf("panic in background copy: %v", r)
			}
			snapshotManager.FinishSnapshot(snapshotName)
			done <- true
		}()

		if err := internal.CreateSnapshotCopyWithRetry(snapshotDatabaseName, 3); err != nil {
			debugLog.Printf("Error in background copy: %v", err)
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
	fmt.Println("Initial snapshot created successfully. Secondary copy is being created in the background.")
}
