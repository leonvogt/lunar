package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"hash/crc32"
	"net/url"
	"strings"
	"time"

	"github.com/leonvogt/lunar/internal/provider"
	_ "github.com/lib/pq"
)

const separator = "____"

type Config struct {
	DatabaseURL         string
	DatabaseName        string
	MaintenanceDatabase string
}

type Provider struct {
	config       *Config
	dbConnection *sql.DB
}

func New(config *Config) (*Provider, error) {
	db, err := connectToMaintenanceDatabase(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to maintenance database: %v", err)
	}

	return &Provider{
		config:       config,
		dbConnection: db,
	}, nil
}

func (p *Provider) Close() error {
	if p.dbConnection != nil {
		return p.dbConnection.Close()
	}
	return nil
}

func (p *Provider) GetDatabaseIdentifier() string {
	return p.config.DatabaseName
}

func (p *Provider) CheckIfSnapshotCanBeTaken(snapshotName string) error {
	snapshotDBName := snapshotDatabaseName(p.config.DatabaseName, snapshotName)

	exists, err := p.doesDatabaseExist(snapshotDBName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("snapshot with name %s already exists", snapshotName)
	}

	return nil
}

func (p *Provider) CheckIfSnapshotExists(snapshotName string) error {
	snapshotDBName := snapshotDatabaseName(p.config.DatabaseName, snapshotName)

	exists, err := p.doesDatabaseExist(snapshotDBName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("snapshot with name %s does not exist", snapshotName)
	}

	return nil
}

func (p *Provider) CreateSnapshot(snapshotName string) error {
	databaseName := p.config.DatabaseName
	snapshotDBName := snapshotDatabaseName(databaseName, snapshotName)

	if err := p.markOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer p.markOperationFinish(databaseName)

	if err := p.markSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}
	defer p.markSnapshotFinish(snapshotName)

	if err := p.createDatabaseCopy(databaseName, snapshotDBName); err != nil {
		return fmt.Errorf("error creating snapshot: %v", err)
	}

	return nil
}

func (p *Provider) CreateSnapshotCopy(snapshotName string) error {
	databaseName := p.config.DatabaseName
	snapshotDBName := snapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDBName := snapshotCopyDatabaseName(databaseName, snapshotName)

	if err := p.markOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer p.markOperationFinish(databaseName)

	if err := p.terminateConnections(snapshotDBName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot: %v", err)
	}

	if err := p.createDatabaseCopy(snapshotDBName, snapshotCopyDBName); err != nil {
		return fmt.Errorf("failed to create snapshot copy: %v", err)
	}

	return nil
}

func (p *Provider) RestoreSnapshot(snapshotName string) error {
	databaseName := p.config.DatabaseName
	snapshotDBName := snapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDBName := snapshotCopyDatabaseName(databaseName, snapshotName)

	copyExists, err := p.doesDatabaseExist(snapshotCopyDBName)
	if err != nil {
		return err
	}
	if !copyExists {
		return fmt.Errorf("snapshot copy %s does not exist. The snapshot may still be initializing or was not created properly", snapshotName)
	}

	if err := p.markOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer p.markOperationFinish(databaseName)

	if err := p.terminateConnections(databaseName); err != nil {
		return fmt.Errorf("failed to terminate connections to database: %v", err)
	}

	if err := p.terminateConnections(snapshotCopyDBName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot copy: %v", err)
	}

	// Drop the current database and rename the copy to take its place
	if err := p.dropDatabase(databaseName); err != nil {
		return err
	}

	if err := p.renameDatabase(snapshotCopyDBName, databaseName); err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	snapshotExists, err := p.doesDatabaseExist(snapshotDBName)
	if err != nil {
		return fmt.Errorf("failed to verify snapshot: %v", err)
	}
	if !snapshotExists {
		return fmt.Errorf("snapshot %s no longer exists after restore", snapshotName)
	}

	return nil
}

func (p *Provider) RemoveSnapshot(snapshotName string) error {
	databaseName := p.config.DatabaseName
	snapshotDBName := snapshotDatabaseName(databaseName, snapshotName)
	snapshotCopyDBName := snapshotCopyDatabaseName(databaseName, snapshotName)

	if err := p.terminateConnections(snapshotDBName); err != nil {
		return fmt.Errorf("failed to terminate connections to snapshot: %v", err)
	}

	if err := p.dropDatabase(snapshotDBName); err != nil {
		return fmt.Errorf("failed to drop snapshot database: %v", err)
	}

	// Also remove the _copy database if it exists
	copyExists, err := p.doesDatabaseExist(snapshotCopyDBName)
	if err != nil {
		return fmt.Errorf("failed to check if snapshot copy exists: %v", err)
	}

	if copyExists {
		if err := p.terminateConnections(snapshotCopyDBName); err != nil {
			return fmt.Errorf("failed to terminate connections to snapshot copy: %v", err)
		}
		if err := p.dropDatabase(snapshotCopyDBName); err != nil {
			return fmt.Errorf("failed to drop snapshot copy database: %v", err)
		}
	}

	return nil
}

func (p *Provider) ReplaceSnapshot(snapshotName string) error {
	databaseName := p.config.DatabaseName

	if err := p.CheckIfSnapshotExists(snapshotName); err != nil {
		return err
	}

	if err := p.markOperationStart(databaseName); err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	defer p.markOperationFinish(databaseName)

	if err := p.RemoveSnapshot(snapshotName); err != nil {
		return fmt.Errorf("failed to remove existing snapshot: %v", err)
	}

	snapshotDBName := snapshotDatabaseName(databaseName, snapshotName)

	if err := p.markSnapshotStart(snapshotName); err != nil {
		return fmt.Errorf("failed to mark snapshot start: %v", err)
	}
	defer p.markSnapshotFinish(snapshotName)

	if err := p.createDatabaseCopy(databaseName, snapshotDBName); err != nil {
		return fmt.Errorf("failed to create new snapshot: %v", err)
	}

	return nil
}

func (p *Provider) ListSnapshots() ([]provider.SnapshotInfo, error) {
	databaseName := p.config.DatabaseName
	snapshotNames, err := p.snapshotDatabasesForDatabase(databaseName)
	if err != nil {
		return nil, err
	}

	snapshots := make([]provider.SnapshotInfo, 0, len(snapshotNames))
	for _, name := range snapshotNames {
		snapshotDBName := snapshotDatabaseName(databaseName, name)
		creationTime, err := p.getDatabaseAge(snapshotDBName)
		if err != nil {
			snapshots = append(snapshots, provider.SnapshotInfo{Name: name, Age: 0})
			continue
		}

		age := time.Since(creationTime)
		snapshots = append(snapshots, provider.SnapshotInfo{Name: name, Age: age})
	}

	return snapshots, nil
}

func (p *Provider) IsSnapshotInProgress(snapshotName string) bool {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))

	var locked bool
	err := p.dbConnection.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	if err != nil {
		return true
	}

	if locked {
		_, _ = p.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
		return false
	}

	return true
}

func (p *Provider) IsOperationInProgress() bool {
	lockID := p.operationLockID(p.config.DatabaseName)

	var locked bool
	err := p.dbConnection.QueryRow("SELECT pg_try_advisory_lock($1)", lockID).Scan(&locked)
	if err != nil {
		return true
	}

	if locked {
		_, _ = p.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
		return false
	}

	return true
}

func (p *Provider) WaitForOngoingSnapshot(snapshotName string) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))
	timeout := 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := p.dbConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("timeout waiting for ongoing snapshot to complete: %v", err)
	}

	_, _ = p.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	return nil
}

func (p *Provider) WaitForOngoingOperations() error {
	lockID := p.operationLockID(p.config.DatabaseName)
	timeout := 30 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := p.dbConnection.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("timeout waiting for ongoing operation to complete: %v", err)
	}

	_, _ = p.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
	return nil
}

func (p *Provider) operationLockID(databaseName string) int64 {
	return int64(crc32.ChecksumIEEE([]byte("op:" + databaseName)))
}

func (p *Provider) markOperationStart(databaseName string) error {
	lockID := p.operationLockID(databaseName)
	_, err := p.dbConnection.Exec("SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire operation lock: %v", err)
	}
	return nil
}

func (p *Provider) markOperationFinish(databaseName string) {
	lockID := p.operationLockID(databaseName)
	_, _ = p.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
}

func (p *Provider) markSnapshotStart(snapshotName string) error {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))
	_, err := p.dbConnection.Exec("SELECT pg_advisory_lock($1)", lockID)
	if err != nil {
		return fmt.Errorf("failed to acquire snapshot lock: %v", err)
	}
	return nil
}

func (p *Provider) markSnapshotFinish(snapshotName string) {
	lockID := int64(crc32.ChecksumIEEE([]byte(snapshotName)))
	_, _ = p.dbConnection.Exec("SELECT pg_advisory_unlock($1)", lockID)
}

func (p *Provider) createDatabaseCopy(sourceDB, targetDB string) error {
	if err := p.terminateConnections(sourceDB); err != nil {
		return fmt.Errorf("failed to terminate connections: %v", err)
	}

	_, err := p.dbConnection.Exec(fmt.Sprintf("CREATE DATABASE \"%s\" TEMPLATE \"%s\"", targetDB, sourceDB))
	if err != nil {
		return fmt.Errorf("failed to create database copy: %v", err)
	}

	return nil
}

func (p *Provider) terminateConnections(databaseName string) error {
	query := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1
		AND pid <> pg_backend_pid()`

	_, err := p.dbConnection.Exec(query, databaseName)
	return err
}

func (p *Provider) doesDatabaseExist(databaseName string) (bool, error) {
	var exists bool
	err := p.dbConnection.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %v", err)
	}
	return exists, nil
}

func (p *Provider) dropDatabase(databaseName string) error {
	_, err := p.dbConnection.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", databaseName))
	if err != nil {
		return fmt.Errorf("failed to drop database: %v", err)
	}
	return nil
}

func (p *Provider) renameDatabase(oldName, newName string) error {
	_, err := p.dbConnection.Exec(fmt.Sprintf("ALTER DATABASE \"%s\" RENAME TO \"%s\"", oldName, newName))
	if err != nil {
		return fmt.Errorf("failed to rename database: %v", err)
	}
	return nil
}

func (p *Provider) getDatabaseAge(databaseName string) (time.Time, error) {
	var creationTime time.Time
	query := `
		SELECT (pg_stat_file('base/'|| oid ||'/PG_VERSION')).modification
		FROM pg_database
		WHERE datname = $1`

	err := p.dbConnection.QueryRow(query, databaseName).Scan(&creationTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get database age: %v", err)
	}

	return creationTime, nil
}

func (p *Provider) allDatabases() ([]string, error) {
	databases := make([]string, 0)

	rows, err := p.dbConnection.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var databaseName string
		if err := rows.Scan(&databaseName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %v", err)
		}
		databases = append(databases, databaseName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database rows: %v", err)
	}

	return databases, nil
}

func (p *Provider) allSnapshotDatabases() ([]string, error) {
	databases, err := p.allDatabases()
	if err != nil {
		return nil, err
	}

	snapshotDatabases := make([]string, 0)
	expectedPrefix := "lunar_snapshot" + separator

	for _, databaseName := range databases {
		if len(databaseName) >= len(expectedPrefix) && databaseName[:len(expectedPrefix)] == expectedPrefix {
			snapshotDatabases = append(snapshotDatabases, databaseName)
		}
	}

	return snapshotDatabases, nil
}

func (p *Provider) snapshotDatabasesForDatabase(databaseName string) ([]string, error) {
	allSnapshots, err := p.allSnapshotDatabases()
	if err != nil {
		return nil, err
	}

	snapshots := make([]string, 0)
	for _, snapshotDB := range allSnapshots {
		parts := strings.Split(snapshotDB, separator)

		if len(parts) >= 3 && parts[1] == databaseName {
			snapshotName := parts[2]
			if !strings.HasSuffix(snapshotName, "_copy") {
				snapshots = append(snapshots, snapshotName)
			}
		}
	}

	return snapshots, nil
}

func snapshotDatabaseName(databaseName, snapshotName string) string {
	return "lunar_snapshot" + separator + databaseName + separator + snapshotName
}

func snapshotCopyDatabaseName(databaseName, snapshotName string) string {
	return snapshotDatabaseName(databaseName, snapshotName) + "_copy"
}

func defaultMaintenanceDatabases() []string {
	return []string{"postgres", "template1"}
}

func connectToMaintenanceDatabase(config *Config) (*sql.DB, error) {
	databasesToTry := []string{}
	if config.MaintenanceDatabase != "" {
		databasesToTry = append(databasesToTry, config.MaintenanceDatabase)
	}
	databasesToTry = append(databasesToTry, defaultMaintenanceDatabases()...)

	var lastErr error
	for _, dbName := range databasesToTry {
		db, err := openDatabaseConnection(config.DatabaseURL + dbName)
		if err != nil {
			lastErr = err
			continue
		}

		if err := db.Ping(); err != nil {
			db.Close()
			lastErr = err
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to any maintenance database (tried: %v): %v", databasesToTry, lastErr)
}

func openDatabaseConnection(databaseURL string) (*sql.DB, error) {
	databaseURL = ensureSSLModeDefault(databaseURL)

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	return db, nil
}

func ensureSSLModeDefault(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		if strings.Contains(databaseURL, "sslmode=") {
			return databaseURL
		}
		if strings.Contains(databaseURL, "?") {
			return databaseURL + "&sslmode=disable"
		}
		return databaseURL + "?sslmode=disable"
	}

	query := parsed.Query()
	if query.Get("sslmode") == "" {
		query.Set("sslmode", "disable")
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}

	return databaseURL
}

// ConnectToMaintenanceDatabaseWithURL connects to a maintenance database using a specific URL.
// Useful during initialization when config may not exist yet.
func ConnectToMaintenanceDatabaseWithURL(databaseURL string) (*sql.DB, error) {
	for _, dbName := range defaultMaintenanceDatabases() {
		db, err := openDatabaseConnection(databaseURL + dbName)
		if err != nil {
			continue
		}

		if err := db.Ping(); err != nil {
			db.Close()
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to any maintenance database (tried: %v)", defaultMaintenanceDatabases())
}

func AllDatabasesWithConnection(db *sql.DB) ([]string, error) {
	databases := make([]string, 0)

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var databaseName string
		if err := rows.Scan(&databaseName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %v", err)
		}
		databases = append(databases, databaseName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database rows: %v", err)
	}

	return databases, nil
}

func TestConnection(db *sql.DB) error {
	db.SetConnMaxLifetime(5 * time.Second)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	return nil
}
