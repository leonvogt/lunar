package internal

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"log"
	"time"
)

// Manager handles database snapshot operations and locking
type Manager struct {
	dbConnection *sql.DB
	logger       *log.Logger
}

func NewSnapshotManager(logger *log.Logger) (*Manager, error) {
	// Connect to postgres database
	db, err := sql.Open("postgres", "postgres://localhost:5432/postgres?sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return &Manager{
		dbConnection: db,
		logger:       logger,
	}, nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	return m.dbConnection.Close()
}

func (m *Manager) logDebug(format string, v ...interface{}) {
	if m.logger != nil {
		m.logger.Printf(format, v...)
	}
}

// IsSnapshotInProgress checks if a snapshot operation is currently running
func (m *Manager) IsSnapshotInProgress(snapshotName string) bool {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	var locked bool
	err := m.dbConnection.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	if err != nil {
		m.logDebug("Error checking advisory lock: %v", err)
		return true
	}

	if locked {
		_, err := m.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
		if err != nil {
			m.logDebug("Error releasing advisory lock: %v", err)
		}
		return false
	}

	return true
}

// StartSnapshot attempts to start a new snapshot operation
func (m *Manager) StartSnapshot(snapshotName string) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	_, err := m.dbConnection.Exec("SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire snapshot lock: %v", err)
	}

	return nil
}

// FinishSnapshot marks a snapshot operation as complete
func (m *Manager) FinishSnapshot(snapshotName string) {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	_, err := m.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		m.logDebug("Error releasing advisory lock: %v", err)
	}
}

// WaitForOngoingSnapshot waits for an ongoing snapshot to complete
func (m *Manager) WaitForOngoingSnapshot(snapshotName string, timeout time.Duration) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := m.dbConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("timeout waiting for ongoing snapshot to complete: %v", err)
	}

	_, err = m.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	if err != nil {
		m.logDebug("Error releasing advisory lock: %v", err)
	}

	return nil
}
