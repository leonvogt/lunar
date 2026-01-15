package provider

import "time"

type SnapshotInfo struct {
	Name string
	Age  time.Duration
}

type Provider interface {
	// Close releases any resources held by the provider
	Close() error

	// (database name for PostgreSQL, file path for SQLite)
	GetDatabaseIdentifier() string

	// Snapshot operations
	CheckIfSnapshotCanBeTaken(snapshotName string) error
	CheckIfSnapshotExists(snapshotName string) error
	CreateSnapshot(snapshotName string) error
	CreateSnapshotCopy(snapshotName string) error
	RestoreSnapshot(snapshotName string) error
	RemoveSnapshot(snapshotName string) error
	ReplaceSnapshot(snapshotName string) error
	ListSnapshots() ([]SnapshotInfo, error)

	// Locking/synchronization operations
	IsSnapshotInProgress(snapshotName string) bool
	IsOperationInProgress() bool
	WaitForOngoingSnapshot(snapshotName string) error
	WaitForOngoingOperations() error
}

type ProviderType string

const (
	ProviderTypePostgres ProviderType = "postgres"
	ProviderTypeSQLite   ProviderType = "sqlite"
)
