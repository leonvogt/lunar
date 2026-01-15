package sqlite

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/leonvogt/lunar/internal/provider"
)

type Config struct {
	DatabasePath      string
	SnapshotDirectory string
}

type Provider struct {
	config *Config
	mu     sync.Mutex // For synchronization (replaces PostgreSQL advisory locks)
}

func New(config *Config) (*Provider, error) {
	if config.DatabasePath == "" {
		return nil, fmt.Errorf("database_path is required for SQLite provider")
	}

	if _, err := os.Stat(config.DatabasePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("database file does not exist: %s", config.DatabasePath)
	}

	// Set default snapshot directory if not provided
	if config.SnapshotDirectory == "" {
		config.SnapshotDirectory = filepath.Join(filepath.Dir(config.DatabasePath), ".lunar_snapshots")
	}

	if err := os.MkdirAll(config.SnapshotDirectory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %v", err)
	}

	return &Provider{
		config: config,
	}, nil
}

func (p *Provider) Close() error {
	// No persistent connections to close for SQLite file-based approach
	return nil
}

func (p *Provider) GetDatabaseIdentifier() string {
	return p.config.DatabasePath
}

func (p *Provider) CheckIfSnapshotCanBeTaken(snapshotName string) error {
	snapshotPath := p.snapshotPath(snapshotName)

	if _, err := os.Stat(snapshotPath); err == nil {
		return fmt.Errorf("snapshot with name %s already exists", snapshotName)
	}

	return nil
}

func (p *Provider) CheckIfSnapshotExists(snapshotName string) error {
	snapshotPath := p.snapshotPath(snapshotName)

	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot with name %s does not exist", snapshotName)
	}

	return nil
}

func (p *Provider) CreateSnapshot(snapshotName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	snapshotPath := p.snapshotPath(snapshotName)

	if err := copyFile(p.config.DatabasePath, snapshotPath); err != nil {
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	p.copyWALFiles(p.config.DatabasePath, snapshotPath)

	return nil
}

func (p *Provider) CreateSnapshotCopy(snapshotName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	snapshotPath := p.snapshotPath(snapshotName)
	copyPath := p.snapshotCopyPath(snapshotName)

	if err := copyFile(snapshotPath, copyPath); err != nil {
		return fmt.Errorf("failed to create snapshot copy: %v", err)
	}

	// Also copy WAL files if they exist
	p.copyWALFiles(snapshotPath, copyPath)

	return nil
}

func (p *Provider) RestoreSnapshot(snapshotName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	copyPath := p.snapshotCopyPath(snapshotName)

	if _, err := os.Stat(copyPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot copy %s does not exist. The snapshot may still be initializing or was not created properly", snapshotName)
	}

	if err := os.Remove(p.config.DatabasePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove current database: %v", err)
	}

	// Remove WAL and SHM files if they exist
	os.Remove(p.config.DatabasePath + "-wal")
	os.Remove(p.config.DatabasePath + "-shm")

	if err := copyFile(copyPath, p.config.DatabasePath); err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	// Copy WAL files if they exist
	p.copyWALFiles(copyPath, p.config.DatabasePath)

	// Remove the copy file after restore (it will be recreated)
	os.Remove(copyPath)
	os.Remove(copyPath + "-wal")
	os.Remove(copyPath + "-shm")

	snapshotPath := p.snapshotPath(snapshotName)
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return fmt.Errorf("snapshot %s no longer exists after restore", snapshotName)
	}

	return nil
}

func (p *Provider) RemoveSnapshot(snapshotName string) error {
	snapshotPath := p.snapshotPath(snapshotName)
	copyPath := p.snapshotCopyPath(snapshotName)

	if err := os.Remove(snapshotPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove snapshot: %v", err)
	}

	os.Remove(snapshotPath + "-wal")
	os.Remove(snapshotPath + "-shm")
	os.Remove(copyPath)
	os.Remove(copyPath + "-wal")
	os.Remove(copyPath + "-shm")

	return nil
}

func (p *Provider) ReplaceSnapshot(snapshotName string) error {
	if err := p.CheckIfSnapshotExists(snapshotName); err != nil {
		return err
	}

	if err := p.RemoveSnapshot(snapshotName); err != nil {
		return fmt.Errorf("failed to remove existing snapshot: %v", err)
	}

	if err := p.CreateSnapshot(snapshotName); err != nil {
		return fmt.Errorf("failed to create new snapshot: %v", err)
	}

	return nil
}

func (p *Provider) ListSnapshots() ([]provider.SnapshotInfo, error) {
	entries, err := os.ReadDir(p.config.SnapshotDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return []provider.SnapshotInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read snapshot directory: %v", err)
	}

	snapshots := make([]provider.SnapshotInfo, 0)
	dbBaseName := filepath.Base(p.config.DatabasePath)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip WAL and SHM files
		if strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-shm") {
			continue
		}

		// Skip copy files
		if strings.HasSuffix(name, "_copy.db") {
			continue
		}

		// Match snapshot files: dbname_snapshotname.db
		prefix := strings.TrimSuffix(dbBaseName, filepath.Ext(dbBaseName)) + "_"
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".db") {
			continue
		}

		// Extract snapshot name
		snapshotName := strings.TrimPrefix(name, prefix)
		snapshotName = strings.TrimSuffix(snapshotName, ".db")

		info, err := entry.Info()
		if err != nil {
			snapshots = append(snapshots, provider.SnapshotInfo{Name: snapshotName, Age: 0})
			continue
		}

		age := time.Since(info.ModTime())
		snapshots = append(snapshots, provider.SnapshotInfo{Name: snapshotName, Age: age})
	}

	return snapshots, nil
}

// For SQLite, we use mutex-based locking, so we just try to acquire the lock
func (p *Provider) IsSnapshotInProgress(snapshotName string) bool {
	// Try to acquire the lock without blocking
	if p.mu.TryLock() {
		p.mu.Unlock()
		return false
	}
	return true
}

func (p *Provider) IsOperationInProgress() bool {
	if p.mu.TryLock() {
		p.mu.Unlock()
		return false
	}
	return true
}

func (p *Provider) WaitForOngoingSnapshot(snapshotName string) error {
	// Acquire and release the lock to wait
	p.mu.Lock()
	p.mu.Unlock()
	return nil
}

func (p *Provider) WaitForOngoingOperations() error {
	p.mu.Lock()
	p.mu.Unlock()
	return nil
}

func (p *Provider) snapshotPath(snapshotName string) string {
	dbBaseName := filepath.Base(p.config.DatabasePath)
	dbNameWithoutExt := strings.TrimSuffix(dbBaseName, filepath.Ext(dbBaseName))
	return filepath.Join(p.config.SnapshotDirectory, dbNameWithoutExt+"_"+snapshotName+".db")
}

func (p *Provider) snapshotCopyPath(snapshotName string) string {
	dbBaseName := filepath.Base(p.config.DatabasePath)
	dbNameWithoutExt := strings.TrimSuffix(dbBaseName, filepath.Ext(dbBaseName))
	return filepath.Join(p.config.SnapshotDirectory, dbNameWithoutExt+"_"+snapshotName+"_copy.db")
}

func (p *Provider) copyWALFiles(src, dst string) {
	// Copy WAL file if exists
	if _, err := os.Stat(src + "-wal"); err == nil {
		copyFile(src+"-wal", dst+"-wal")
	}
	// Copy SHM file if exists
	if _, err := os.Stat(src + "-shm"); err == nil {
		copyFile(src+"-shm", dst+"-shm")
	}
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}
