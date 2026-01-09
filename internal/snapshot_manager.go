package internal

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"time"
)

// Manager handles database snapshot operations and locking
type Manager struct {
	dbConnection *sql.DB
	config       *Config
}

func SnapshotManager(config *Config) (*Manager, error) {
	db := ConnectToPostgresDatabase()

	return &Manager{
		dbConnection: db,
		config:       config,
	}, nil
}

// Close closes the database connection
func (manager *Manager) Close() error {
	return manager.dbConnection.Close()
}

func (manager *Manager) CheckIfSnapshotCanBeTaken(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	databaseNameFromSnapshot := SnapshotDatabaseName(databaseName, snapshotName)

	if DoesDatabaseExists(databaseNameFromSnapshot) {
		return fmt.Errorf("snapshot with name %s already exists", snapshotName)
	}

	// Check for ongoing operations on this database (e.g., restore in progress)
	if manager.IsOperationInProgress(databaseName) {
		fmt.Println("Waiting for ongoing operation to complete...")
		if err := manager.WaitForOngoingOperation(databaseName, 30*time.Minute); err != nil {
			return fmt.Errorf("failed to wait for ongoing operation: %v", err)
		}
	}

	// Check for ongoing snapshot operations with the same name
	if manager.IsSnapshotInProgress(snapshotName) {
		fmt.Println("Waiting for ongoing snapshot to complete...")
		if err := manager.WaitForOngoingSnapshot(snapshotName, 30*time.Minute); err != nil {
			return fmt.Errorf("failed to wait for ongoing snapshot: %v", err)
		}
	}
	return nil
}

func (manager *Manager) StartSnapshotprocess(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	databaseNameFromSnapshot := SnapshotDatabaseName(databaseName, snapshotName)

	// Acquire operation lock to block restores during snapshot
	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	if err := manager.MarkSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}
	defer manager.MarkSnapshotFinish(snapshotName)

	if err := manager.CreateSnapshot(databaseName, databaseNameFromSnapshot); err != nil {
		return fmt.Errorf("error creating snapshot: %v", err)
	}

	fmt.Println("Snapshot created successfully")
	return nil
}

func (manager *Manager) CreateSnapshot(databaseName, snapshotName string) error {
	if err := TerminateAllCurrentConnections(databaseName); err != nil {
		return fmt.Errorf("failed to terminate connections: %v", err)
	}

	err := CreateDatabaseWithTemplate(manager.dbConnection, snapshotName, databaseName)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	return nil
}

// Checks if a snapshot operation is currently running
func (manager *Manager) IsSnapshotInProgress(snapshotName string) bool {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	var locked bool
	err := manager.dbConnection.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	if err != nil {
		fmt.Printf("Error checking advisory lock: %v", err)
		return true
	}

	if locked {
		// Release the lock
		_, err := manager.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
		if err != nil {
			fmt.Printf("Error releasing advisory lock: %v", err)
		}
		return false
	}

	return true
}

// Creates an advisory lock for the snapshot
func (manager *Manager) MarkSnapshotStart(snapshotName string) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	_, err := manager.dbConnection.Exec("SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire snapshot lock: %v", err)
	}

	return nil
}

// Releases the advisory lock
func (manager *Manager) MarkSnapshotFinish(snapshotName string) {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	_, err := manager.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		fmt.Printf("Error releasing advisory lock: %v", err)
	}
}

// Waits for an ongoing snapshot to complete
func (manager *Manager) WaitForOngoingSnapshot(snapshotName string, timeout time.Duration) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := manager.dbConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("timeout waiting for ongoing snapshot to complete: %v", err)
	}

	_, err = manager.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		fmt.Printf("Error releasing advisory lock: %v", err)
	}

	return nil
}

// operationLockID generates a lock ID for database-level operations (restore, etc.)
// Uses a different prefix to avoid collision with snapshot locks
func (manager *Manager) operationLockID(databaseName string) int64 {
	return int64(crc32.ChecksumIEEE([]byte("op:" + databaseName)))
}

// IsOperationInProgress checks if any operation is currently running on the database
func (manager *Manager) IsOperationInProgress(databaseName string) bool {
	lockID := manager.operationLockID(databaseName)

	var locked bool
	err := manager.dbConnection.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	if err != nil {
		fmt.Printf("Error checking advisory lock: %v", err)
		return true
	}

	if locked {
		_, err := manager.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
		if err != nil {
			fmt.Printf("Error releasing advisory lock: %v", err)
		}
		return false
	}

	return true
}

// MarkOperationStart acquires an advisory lock for database operations
func (manager *Manager) MarkOperationStart(databaseName string) error {
	lockID := manager.operationLockID(databaseName)

	_, err := manager.dbConnection.Exec("SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}

	return nil
}

// MarkOperationFinish releases the advisory lock for database operations
func (manager *Manager) MarkOperationFinish(databaseName string) {
	lockID := manager.operationLockID(databaseName)

	_, err := manager.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		fmt.Printf("Error releasing advisory lock: %v", err)
	}
}

// WaitForOngoingOperation waits for any ongoing operation on the database to complete
func (manager *Manager) WaitForOngoingOperation(databaseName string, timeout time.Duration) error {
	lockID := manager.operationLockID(databaseName)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := manager.dbConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("timeout waiting for ongoing operation to complete: %v", err)
	}

	_, err = manager.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		fmt.Printf("Error releasing advisory lock: %v", err)
	}

	return nil
}

func (manager *Manager) CheckIfSnapshotExists(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	databaseNameFromSnapshot := SnapshotDatabaseName(databaseName, snapshotName)

	if !DoesDatabaseExists(databaseNameFromSnapshot) {
		return fmt.Errorf("snapshot with name %s does not exist", snapshotName)
	}

	return nil
}

func (manager *Manager) RestoreSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDatabaseName := snapshotDatabaseName + "_copy"

	// Check for ongoing operations on this database (e.g., _copy being created)
	if manager.IsOperationInProgress(databaseName) {
		fmt.Println("Currently there is a Lunar background operation running. Waiting for it to complete before restoring the snapshot...")
		if err := manager.WaitForOngoingOperation(databaseName, 30*time.Minute); err != nil {
			return fmt.Errorf("failed to wait for ongoing operation: %v", err)
		}
	}

	// Check if the _copy database exists (required for fast restore)
	if !DoesDatabaseExists(snapshotCopyDatabaseName) {
		return fmt.Errorf("snapshot copy %s does not exist. The snapshot may still be initializing or was not created properly", snapshotName)
	}

	// Acquire lock for the restore operation
	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	// Terminate all connections to the target database and the _copy database
	if err := TerminateAllCurrentConnections(databaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to database: %v", err)
	}

	if err := TerminateAllCurrentConnections(snapshotCopyDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot copy: %v", err)
	}

	// Fast restore: drop target DB and rename _copy to target
	if err := RestoreSnapshot(databaseName, snapshotCopyDatabaseName); err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	return nil
}

// RecreateSnapshotCopy creates a new _copy database from the snapshot in the background
// This prepares for the next restore operation
func (manager *Manager) RecreateSnapshotCopy(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDatabaseName := snapshotDatabaseName + "_copy"

	// Acquire operation lock so other operations (like restore) wait for us
	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	// Terminate any connections to the snapshot (needed for template usage)
	if err := TerminateAllCurrentConnections(snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot: %v", err)
	}

	// Create a new _copy from the snapshot
	if err := manager.CreateSnapshot(snapshotDatabaseName, snapshotCopyDatabaseName); err != nil {
		return fmt.Errorf("failed to recreate snapshot copy: %v", err)
	}

	return nil
}

// RemoveSnapshot removes an existing snapshot database
func (manager *Manager) RemoveSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)

	// Terminate all connections to the snapshot database
	if err := TerminateAllCurrentConnections(snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot: %v", err)
	}

	// Drop the snapshot database
	DropDatabase(snapshotDatabaseName)
	return nil
}

// ReplaceSnapshot removes an existing snapshot and creates a new one with the same name
func (manager *Manager) ReplaceSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName

	// Check if the snapshot exists
	if err := manager.CheckIfSnapshotExists(snapshotName); err != nil {
		return err
	}

	// Check for ongoing operations on this database
	if manager.IsOperationInProgress(databaseName) {
		fmt.Println("Currently there is a Lunar background operation running. Waiting for it to complete before replacing the snapshot...")
		if err := manager.WaitForOngoingOperation(databaseName, 30*time.Minute); err != nil {
			return fmt.Errorf("failed to wait for ongoing operation: %v", err)
		}
	}

	// Also check for ongoing snapshot operations
	if manager.IsSnapshotInProgress(snapshotName) {
		fmt.Println("Waiting for ongoing snapshot to complete...")
		if err := manager.WaitForOngoingSnapshot(snapshotName, 30*time.Minute); err != nil {
			return fmt.Errorf("failed to wait for ongoing snapshot: %v", err)
		}
	}

	// Acquire lock for the operation
	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	// Remove the existing snapshot
	if err := manager.RemoveSnapshot(snapshotName); err != nil {
		return fmt.Errorf("failed to remove existing snapshot: %v", err)
	}

	// Create a new snapshot with the same name
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)

	if err := manager.MarkSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}
	defer manager.MarkSnapshotFinish(snapshotName)

	if err := manager.CreateSnapshot(databaseName, snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to create new snapshot: %v", err)
	}

	return nil
}

func (manager *Manager) ListSnapshots() ([]string, error) {
	databaseName := manager.config.DatabaseName
	return SnapshotDatabasesForDatabase(databaseName), nil
}
