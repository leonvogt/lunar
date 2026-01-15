package tests

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/provider"
	_ "github.com/mattn/go-sqlite3"
)

var (
	sqliteTestDir    string
	sqliteTestConfig *internal.Config
)

// SetupSQLiteTestDatabase creates a test SQLite database with sample data
func SetupSQLiteTestDatabase(t *testing.T) *internal.Config {
	tmpDir, err := os.MkdirTemp("", "lunar_sqlite_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	sqliteTestDir = tmpDir

	dbPath := filepath.Join(tmpDir, "test.db")
	snapshotDir := filepath.Join(tmpDir, "snapshots")

	// Create the SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to create SQLite database: %v", err)
	}
	defer db.Close()

	// Create test table and insert data
	_, err = db.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		firstname TEXT,
		lastname TEXT,
		email TEXT
	)`)
	if err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	InsertSQLiteUsers(db)

	// Create test config
	sqliteTestConfig = &internal.Config{
		ProviderType:      provider.ProviderTypeSQLite,
		DatabasePath:      dbPath,
		SnapshotDirectory: snapshotDir,
	}

	return sqliteTestConfig
}

func TeardownSQLiteTestDatabase(t *testing.T) {
	if sqliteTestDir != "" {
		os.RemoveAll(sqliteTestDir)
		sqliteTestDir = ""
		sqliteTestConfig = nil
	}
}

func InsertSQLiteUsers(db *sql.DB) {
	users := []struct {
		firstname, lastname, email string
	}{
		{"John", "Doe", "john.doe@example.com"},
		{"Jane", "Smith", "jane.smith@example.com"},
		{"Michael", "Johnson", "michael.johnson@example.com"},
		{"Emily", "Brown", "emily.brown@example.com"},
		{"Christopher", "Wilson", "christopher.wilson@example.com"},
	}

	for _, u := range users {
		_, err := db.Exec("INSERT INTO users (firstname, lastname, email) VALUES (?, ?, ?)",
			u.firstname, u.lastname, u.email)
		if err != nil {
			fmt.Printf("Error inserting user: %v\n", err)
		}
	}
}

func SetupSQLiteTestDirectory(t *testing.T, config *internal.Config) *TestDirectoryManager {
	originalDir, _ := os.Getwd()
	os.Chdir("..")

	err := internal.CreateConfigFile(config, internal.CONFIG_PATH)
	if err != nil {
		t.Fatalf("Failed to create SQLite config file: %v", err)
	}

	return &TestDirectoryManager{
		originalDir:   originalDir,
		hasConfigFile: true,
	}
}

func SQLiteSnapshotExists(snapshotName string) (bool, error) {
	if sqliteTestConfig == nil {
		return false, fmt.Errorf("SQLite test config not initialized")
	}

	// Snapshot naming follows the pattern: dbNameWithoutExt_snapshotName.db
	dbBaseName := filepath.Base(sqliteTestConfig.DatabasePath)
	dbNameWithoutExt := dbBaseName[:len(dbBaseName)-len(filepath.Ext(dbBaseName))]
	snapshotPath := filepath.Join(sqliteTestConfig.SnapshotDirectory, dbNameWithoutExt+"_"+snapshotName+".db")

	_, err := os.Stat(snapshotPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func CleanupSQLiteSnapshot(snapshotName string) {
	if sqliteTestConfig == nil {
		return
	}

	// Snapshot naming follows the pattern: dbNameWithoutExt_snapshotName.db
	dbBaseName := filepath.Base(sqliteTestConfig.DatabasePath)
	dbNameWithoutExt := dbBaseName[:len(dbBaseName)-len(filepath.Ext(dbBaseName))]
	snapshotPath := filepath.Join(sqliteTestConfig.SnapshotDirectory, dbNameWithoutExt+"_"+snapshotName+".db")
	snapshotCopyPath := filepath.Join(sqliteTestConfig.SnapshotDirectory, dbNameWithoutExt+"_"+snapshotName+"_copy.db")

	os.Remove(snapshotPath)
	os.Remove(snapshotCopyPath)
	// Also remove WAL files if they exist
	os.Remove(snapshotPath + "-wal")
	os.Remove(snapshotPath + "-shm")
	os.Remove(snapshotCopyPath + "-wal")
	os.Remove(snapshotCopyPath + "-shm")
}

func ConnectToSQLiteTestDatabase() (*sql.DB, error) {
	if sqliteTestConfig == nil {
		return nil, fmt.Errorf("SQLite test config not initialized")
	}

	return sql.Open("sqlite3", sqliteTestConfig.DatabasePath)
}

func WithSQLiteTestDirectory(t *testing.T, config *internal.Config, testFunc func()) {
	dm := SetupSQLiteTestDirectory(t, config)
	defer dm.Cleanup()
	testFunc()
}
