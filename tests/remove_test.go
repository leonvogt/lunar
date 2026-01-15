package tests

import (
	"os"
	"testing"
)

// ============================================================================
// PostgreSQL Remove Tests
// ============================================================================

func TestPostgres_Remove(t *testing.T) {
	const snapshotName = "pg-remove-test"

	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, snapshotName)

		os.Chdir("tests")
		exists, err := DoesDatabaseExist(SnapshotDatabaseName(snapshotName))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName(snapshotName))
		}

		os.Chdir("..")
		_, err = RunLunarCommand("remove " + snapshotName)
		if err != nil {
			t.Errorf("Error removing snapshot: %v", err)
		}

		os.Chdir("tests")
		exists, err = DoesDatabaseExist(SnapshotDatabaseName(snapshotName))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if exists {
			t.Errorf("Expected database `%s` to not exist - but it does", SnapshotDatabaseName(snapshotName))
		}
	})
}

// ============================================================================
// SQLite Remove Tests
// ============================================================================

func TestSQLite_Remove(t *testing.T) {
	const snapshotName = "sqlite-remove-test"

	config := SetupSQLiteTestDatabase(t)
	defer TeardownSQLiteTestDatabase(t)

	WithSQLiteTestDirectory(t, config, func() {
		CreateTestSnapshot(t, snapshotName)

		exists, err := SQLiteSnapshotExists(snapshotName)
		if err != nil {
			t.Fatalf("Error checking snapshot existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected snapshot `%s` to exist - but it does not", snapshotName)
		}

		_, err = RunLunarCommand("remove " + snapshotName)
		if err != nil {
			t.Errorf("Error removing snapshot: %v", err)
		}

		exists, err = SQLiteSnapshotExists(snapshotName)
		if err != nil {
			t.Fatalf("Error checking snapshot existence: %v", err)
		}
		if exists {
			t.Errorf("Expected snapshot `%s` to not exist - but it does", snapshotName)
		}
	})
}
