package internal

import (
	"fmt"

	"github.com/leonvogt/lunar/internal/provider"
	"github.com/leonvogt/lunar/internal/provider/postgres"
)

type Manager struct {
	provider provider.Provider
	config   *Config
}

func NewSnapshotManager(config *Config) (*Manager, error) {
	p, err := createProvider(config)
	if err != nil {
		return nil, err
	}

	return &Manager{
		provider: p,
		config:   config,
	}, nil
}

func createProvider(config *Config) (provider.Provider, error) {
	switch config.GetProviderType() {
	case provider.ProviderTypePostgres:
		return postgres.New(&postgres.Config{
			DatabaseURL:         config.DatabaseUrl,
			DatabaseName:        config.DatabaseName,
			MaintenanceDatabase: config.MaintenanceDatabase,
		})
	case provider.ProviderTypeSQLite:
		return nil, fmt.Errorf("SQLite provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider type: %s", config.GetProviderType())
	}
}

func (m *Manager) Close() error {
	return m.provider.Close()
}

func (m *Manager) GetDatabaseIdentifier() string {
	return m.provider.GetDatabaseIdentifier()
}

// --- Snapshot operations

func (m *Manager) CheckIfSnapshotCanBeTaken(snapshotName string) error {
	return m.provider.CheckIfSnapshotCanBeTaken(snapshotName)
}

func (m *Manager) CheckIfSnapshotExists(snapshotName string) error {
	return m.provider.CheckIfSnapshotExists(snapshotName)
}

func (m *Manager) CreateMainSnapshot(snapshotName string) error {
	return m.provider.CreateSnapshot(snapshotName)
}

func (m *Manager) CreateSnapshotCopy(snapshotName string) error {
	return m.provider.CreateSnapshotCopy(snapshotName)
}

func (m *Manager) RestoreSnapshot(snapshotName string) error {
	return m.provider.RestoreSnapshot(snapshotName)
}

func (m *Manager) RemoveSnapshot(snapshotName string) error {
	return m.provider.RemoveSnapshot(snapshotName)
}

func (m *Manager) ReplaceSnapshot(snapshotName string) error {
	return m.provider.ReplaceSnapshot(snapshotName)
}

func (m *Manager) ListSnapshots() ([]provider.SnapshotInfo, error) {
	return m.provider.ListSnapshots()
}

// --- Locking/synchronization

func (m *Manager) IsSnapshotInProgress(snapshotName string) bool {
	return m.provider.IsSnapshotInProgress(snapshotName)
}

func (m *Manager) IsWaitingForOperation() bool {
	return m.provider.IsOperationInProgress()
}

func (m *Manager) WaitForOngoingSnapshot(snapshotName string) error {
	return m.provider.WaitForOngoingSnapshot(snapshotName)
}

func (m *Manager) WaitForOngoingOperations() error {
	return m.provider.WaitForOngoingOperations()
}
