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

	if err := manager.MarkSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}

	if err := manager.CreateSnapshot(databaseName, databaseNameFromSnapshot); err != nil {
		fmt.Printf("Error creating snapshot: %v\n", err)
		manager.MarkSnapshotFinish(snapshotName)
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
