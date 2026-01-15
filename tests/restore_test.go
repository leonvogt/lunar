package tests

import (
	"os"
	"testing"
)

// ============================================================================
// PostgreSQL Restore Tests
// ============================================================================

func TestPostgres_Restore(t *testing.T) {
	const snapshotName = "pg-restore-test"

	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		os.Chdir("tests")
		database, err := ConnectToTestDatabase("lunar_test")
		if err != nil {
			t.Fatalf("Failed to connect to database: %v", err)
		}
		defer database.Close()

		os.Chdir("..")

		CreateTestSnapshot(t, snapshotName)

		_, err = database.Exec("DROP TABLE users")
		if err != nil {
			t.Errorf("Error dropping table: %v", err)
		}

		_, err = database.Query("SELECT email FROM users")
		if err == nil {
			t.Errorf("Error: Table still exists after drop")
		}

		_, err = RunLunarCommand("restore " + snapshotName)
		if err != nil {
			t.Errorf("Error restoring snapshot: %v", err)
		}

		os.Chdir("tests")
		CleanupSnapshot(snapshotName)
	})
}

// ============================================================================
// SQLite Restore Tests
// ============================================================================

func TestSQLite_Restore(t *testing.T) {
	const snapshotName = "sqlite-restore-test"

	config := SetupSQLiteTestDatabase(t)
	defer TeardownSQLiteTestDatabase(t)

	WithSQLiteTestDirectory(t, config, func() {
		// Create a snapshot first
		CreateTestSnapshot(t, snapshotName)

		os.Chdir("tests")

		// Connect and drop the users table
		database, err := ConnectToSQLiteTestDatabase()
		if err != nil {
			t.Fatalf("Failed to connect to database: %v", err)
		}

		_, err = database.Exec("DROP TABLE users")
		if err != nil {
			t.Errorf("Error dropping table: %v", err)
		}
		database.Close()

		// Verify table is gone
		database, _ = ConnectToSQLiteTestDatabase()
		_, err = database.Query("SELECT email FROM users")
		if err == nil {
			t.Errorf("Error: Table still exists after drop")
		}
		database.Close()

		os.Chdir("..")

		// Restore the snapshot
		_, err = RunLunarCommand("restore " + snapshotName)
		if err != nil {
			t.Errorf("Error restoring snapshot: %v", err)
		}

		os.Chdir("tests")

		// Verify the table is back
		database, err = ConnectToSQLiteTestDatabase()
		if err != nil {
			t.Fatalf("Failed to connect to database after restore: %v", err)
		}
		defer database.Close()

		var count int
		err = database.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
		if err != nil {
			t.Errorf("Error querying users after restore: %v", err)
		}
		if count == 0 {
			t.Errorf("Expected users to exist after restore, but table is empty")
		}

		CleanupSQLiteSnapshot(snapshotName)
	})
}
