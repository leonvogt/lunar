package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var debugLog *log.Logger

func init() {
	// Create debug log file in /tmp
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "lunar-debug.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not create debug log: %v\n", err)
		return
	}

	debugLog = log.New(logFile, "", log.LstdFlags)

	rootCmd.AddCommand(snapshotCmd)
}

func logDebug(format string, v ...interface{}) {
	if debugLog != nil {
		debugLog.Printf(format, v...)
	}
}

type SnapshotManager struct {
	mutex        sync.Mutex
	dbConnection *sql.DB
}

func NewSnapshotManager() (*SnapshotManager, error) {
	// Connect to postgres database
	db, err := sql.Open("postgres", fmt.Sprintf("postgres://localhost:5432/postgres?sslmode=disable"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return &SnapshotManager{
		dbConnection: db,
	}, nil
}

func (sm *SnapshotManager) IsSnapshotInProgress(snapshotName string) bool {
	// Use hash of snapshot name as lock ID
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	var locked bool
	// Try to get advisory lock without waiting
	err := sm.dbConnection.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	if err != nil {
		logDebug("Error checking advisory lock: %v", err)
		return true
	}

	// If we got the lock, release it immediately as this is just a check
	if locked {
		_, err := sm.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
		if err != nil {
			logDebug("Error releasing advisory lock: %v", err)
		}
		return false
	}

	return true
}

func (sm *SnapshotManager) StartSnapshot(snapshotName string) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	// Try to acquire advisory lock
	_, err := sm.dbConnection.Exec("SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire snapshot lock: %v", err)
	}

	return nil
}

func (sm *SnapshotManager) FinishSnapshot(snapshotName string) {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	_, err := sm.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		logDebug("Error releasing advisory lock: %v", err)
	}
}

func (sm *SnapshotManager) WaitForOngoingSnapshot(snapshotName string, timeout time.Duration) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to acquire lock with timeout
	_, err := sm.dbConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("timeout waiting for ongoing snapshot to complete: %v", err)
	}

	// If we get the lock, release it immediately
	_, err = sm.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		logDebug("Error releasing advisory lock: %v", err)
	}

	return nil
}

var snapshotManager, _ = NewSnapshotManager()

var snapshotCmd = &cobra.Command{
	Use:     "snapshot",
	Aliases: []string{"snap"},
	Short:   "Create a snapshot of your database",
	Run: func(_ *cobra.Command, args []string) {
		createSnapshot(args)
	},
}

func createSnapshot(args []string) {
	logDebug("Starting createSnapshot with args: %v", args)

	if !internal.DoesConfigExist() {
		fmt.Println("There seems to be no configuration file. Please run 'lunar init' first")
		return
	}

	if len(args) != 1 {
		fmt.Println("Please provide a name for the snapshot")
		return
	}

	snapshotName := args[0]
	config, _ := internal.ReadConfig()
	snapshotDatabaseName := internal.SnapshotDatabaseName(config.DatabaseName, snapshotName)

	logDebug("Config loaded. Database: %s, Snapshot: %s", config.DatabaseName, snapshotDatabaseName)

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

	logDebug("Creating initial snapshot...")
	if err := internal.CreateSnapshotWithRetry(config.DatabaseName, snapshotDatabaseName, 3); err != nil {
		fmt.Printf("Error creating initial snapshot: %v\n", err)
		snapshotManager.FinishSnapshot(snapshotName)
		return
	}
	logDebug("Initial snapshot created successfully")

	// Channel to track completion of background task
	done := make(chan bool)
	var copyErr error

	// Run the snapshot copy in the background
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logDebug("Panic in background copy: %v", r)
				copyErr = fmt.Errorf("panic in background copy: %v", r)
			}
			snapshotManager.FinishSnapshot(snapshotName)
			done <- true
		}()

		logDebug("Starting background copy process...")
		if err := internal.CreateSnapshotCopyWithRetry(snapshotDatabaseName, 3); err != nil {
			logDebug("Error in background copy: %v", err)
			copyErr = err
			return
		}
		logDebug("Background copy completed successfully")
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
	logDebug("Main process completed. Background copy is running...")
}
