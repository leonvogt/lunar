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

func NewSnapshotManager(config *Config) (*Manager, error) {
	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return nil, err
	}

	return &Manager{
		dbConnection: database,
		config:       config,
	}, nil
}

func (manager *Manager) Close() error {
	return manager.dbConnection.Close()
}

func (manager *Manager) CheckIfSnapshotCanBeTaken(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)

	exists, err := DoesDatabaseExist(snapshotDatabaseName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("snapshot with name %s already exists", snapshotName)
	}

	return nil
}

func (manager *Manager) CreateMainSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)

	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	if err := manager.MarkSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}
	defer manager.MarkSnapshotFinish(snapshotName)

	if err := manager.createDatabaseCopy(databaseName, snapshotDatabaseName); err != nil {
		return fmt.Errorf("error creating snapshot: %v", err)
	}

	return nil
}

func (manager *Manager) createDatabaseCopy(sourceDatabaseName, targetDatabaseName string) error {
	if err := TerminateAllCurrentConnections(sourceDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections: %v", err)
	}

	if err := CreateDatabaseWithTemplate(manager.dbConnection, targetDatabaseName, sourceDatabaseName); err != nil {
		return fmt.Errorf("failed to create database copy: %v", err)
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
func (manager *Manager) operationLockID(databaseName string) int64 {
	return int64(crc32.ChecksumIEEE([]byte("op:" + databaseName)))
}

// IsWaitingForOperation checks if there's an ongoing operation that we'd need to wait for.
func (manager *Manager) IsWaitingForOperation() bool {
	return manager.IsOperationInProgress(manager.config.DatabaseName)
}

// WaitForOngoingOperations waits for any ongoing operations to complete.
func (manager *Manager) WaitForOngoingOperations() error {
	databaseName := manager.config.DatabaseName
	if err := manager.WaitForOngoingOperation(databaseName, 30*time.Minute); err != nil {
		return fmt.Errorf("failed to wait for ongoing operation: %v", err)
	}
	return nil
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
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)

	exists, err := DoesDatabaseExist(snapshotDatabaseName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("snapshot with name %s does not exist", snapshotName)
	}

	return nil
}

func (manager *Manager) RestoreSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDatabaseName := SnapshotCopyDatabaseName(databaseName, snapshotName)

	copyExists, err := DoesDatabaseExist(snapshotCopyDatabaseName)
	if err != nil {
		return err
	}
	if !copyExists {
		return fmt.Errorf("snapshot copy %s does not exist. The snapshot may still be initializing or was not created properly", snapshotName)
	}

	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	if err := TerminateAllCurrentConnections(databaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to database: %v", err)
	}

	if err := TerminateAllCurrentConnections(snapshotCopyDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot copy: %v", err)
	}

	if err := RestoreSnapshot(databaseName, snapshotCopyDatabaseName); err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	// Verify the main snapshot still exists after restore
	snapshotExists, err := DoesDatabaseExist(snapshotDatabaseName)
	if err != nil {
		return fmt.Errorf("failed to verify snapshot: %v", err)
	}
	if !snapshotExists {
		return fmt.Errorf("snapshot %s no longer exists after restore", snapshotName)
	}

	return nil
}

func (manager *Manager) CreateSnapshotCopy(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDatabaseName := SnapshotCopyDatabaseName(databaseName, snapshotName)

	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	if err := TerminateAllCurrentConnections(snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot: %v", err)
	}

	if err := manager.createDatabaseCopy(snapshotDatabaseName, snapshotCopyDatabaseName); err != nil {
		return fmt.Errorf("failed to create snapshot copy: %v", err)
	}

	return nil
}

func (manager *Manager) RemoveSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName
	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDatabaseName := SnapshotCopyDatabaseName(databaseName, snapshotName)

	if err := TerminateAllCurrentConnections(snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot: %v", err)
	}

	if err := DropDatabase(snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to drop snapshot database: %v", err)
	}

	// Also remove the _copy database if it exists
	copyExists, err := DoesDatabaseExist(snapshotCopyDatabaseName)
	if err != nil {
		return fmt.Errorf("failed to check if snapshot copy exists: %v", err)
	}

	if copyExists {
		if err := TerminateAllCurrentConnections(snapshotCopyDatabaseName); err != nil {
			return fmt.Errorf("failed to terminate connections to snapshot copy: %v", err)
		}
		if err := DropDatabase(snapshotCopyDatabaseName); err != nil {
			return fmt.Errorf("failed to drop snapshot copy database: %v", err)
		}
	}

	return nil
}

func (manager *Manager) ReplaceSnapshot(snapshotName string) error {
	databaseName := manager.config.DatabaseName

	if err := manager.CheckIfSnapshotExists(snapshotName); err != nil {
		return err
	}

	if err := manager.MarkOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer manager.MarkOperationFinish(databaseName)

	if err := manager.RemoveSnapshot(snapshotName); err != nil {
		return fmt.Errorf("failed to remove existing snapshot: %v", err)
	}

	snapshotDatabaseName := SnapshotDatabaseName(databaseName, snapshotName)

	if err := manager.MarkSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}
	defer manager.MarkSnapshotFinish(snapshotName)

	if err := manager.createDatabaseCopy(databaseName, snapshotDatabaseName); err != nil {
		return fmt.Errorf("failed to create new snapshot: %v", err)
	}

	return nil
}

func (manager *Manager) ListSnapshots() ([]string, error) {
	databaseName := manager.config.DatabaseName
	return SnapshotDatabasesForDatabase(databaseName)
}
