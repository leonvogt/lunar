package tests

import (
	"os"
	"testing"
)

// ============================================================================
// PostgreSQL Replace Tests
// ============================================================================

func TestPostgres_Replace(t *testing.T) {
	const snapshotName = "pg-replace-test"

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
		// Replace the snapshot
		_, err = RunLunarCommand("replace " + snapshotName)
		if err != nil {
			t.Errorf("Error replacing snapshot: %v", err)
		}

		// Verify snapshot still exists after replace
		os.Chdir("tests")
		exists, err = DoesDatabaseExist(SnapshotDatabaseName(snapshotName))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected database `%s` to exist after replace - but it does not", SnapshotDatabaseName(snapshotName))
		}

		CleanupSnapshot(snapshotName)
	})
}

// ============================================================================
// SQLite Replace Tests
// ============================================================================

func TestSQLite_Replace(t *testing.T) {
	const snapshotName = "sqlite-replace-test"

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

		// Replace the snapshot
		_, err = RunLunarCommand("replace " + snapshotName)
		if err != nil {
			t.Errorf("Error replacing snapshot: %v", err)
		}

		// Verify snapshot still exists after replace
		exists, err = SQLiteSnapshotExists(snapshotName)
		if err != nil {
			t.Fatalf("Error checking snapshot existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected snapshot `%s` to exist after replace - but it does not", snapshotName)
		}

		os.Chdir("tests")
		CleanupSQLiteSnapshot(snapshotName)
	})
}
