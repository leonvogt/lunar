package tests

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

// ============================================================================
// PostgreSQL Snapshot Tests
// ============================================================================

func TestPostgres_Snapshot(t *testing.T) {
	const snapshotName = "pg-snapshot-test"

	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			wd, _ := os.Getwd()
			t.Fatalf("main.go not found in current directory. Current dir: %s", wd)
		}

		CreateTestSnapshot(t, snapshotName)

		os.Chdir("tests")
		exists, err := DoesDatabaseExist(SnapshotDatabaseName(snapshotName))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName(snapshotName))
		}

		CleanupSnapshot(snapshotName)
	})
}

func TestPostgres_SnapshotAlreadyExists(t *testing.T) {
	const snapshotName = "pg-duplicate-test"

	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, snapshotName)

		// Try to create a snapshot with the same name
		out, err := RunLunarCommand("snapshot " + snapshotName)
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		expectedOutput := "snapshot with name " + snapshotName + " already exists\n"
		if string(out) != expectedOutput {
			t.Errorf("Expected output to be '%v' but got '%v'", expectedOutput, string(out))
		}

		os.Chdir("tests")
		CleanupSnapshot(snapshotName)
	})
}

// ============================================================================
// SQLite Snapshot Tests
// ============================================================================

func TestSQLite_Snapshot(t *testing.T) {
	const snapshotName = "sqlite-snapshot-test"

	config := SetupSQLiteTestDatabase(t)
	defer TeardownSQLiteTestDatabase(t)

	WithSQLiteTestDirectory(t, config, func() {
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			wd, _ := os.Getwd()
			t.Fatalf("main.go not found in current directory. Current dir: %s", wd)
		}

		CreateTestSnapshot(t, snapshotName)

		exists, err := SQLiteSnapshotExists(snapshotName)
		if err != nil {
			t.Fatalf("Error checking snapshot existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected snapshot `%s` to exist - but it does not", snapshotName)
		}

		os.Chdir("tests")
		CleanupSQLiteSnapshot(snapshotName)
	})
}

func TestSQLite_SnapshotAlreadyExists(t *testing.T) {
	const snapshotName = "sqlite-duplicate-test"

	config := SetupSQLiteTestDatabase(t)
	defer TeardownSQLiteTestDatabase(t)

	WithSQLiteTestDirectory(t, config, func() {
		CreateTestSnapshot(t, snapshotName)

		// Try to create a snapshot with the same name
		out, err := RunLunarCommand("snapshot " + snapshotName)
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		expectedOutput := "snapshot with name " + snapshotName + " already exists\n"
		if string(out) != expectedOutput {
			t.Errorf("Expected output to be '%v' but got '%v'", expectedOutput, string(out))
		}

		os.Chdir("tests")
		CleanupSQLiteSnapshot(snapshotName)
	})
}
