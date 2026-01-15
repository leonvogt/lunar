package tests

import (
	"os"
	"testing"
)

func TestReplace(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, "production")

		// Go back to tests directory to check database exists
		os.Chdir("tests")
		exists, err := DoesDatabaseExist(SnapshotDatabaseName("production"))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName("production"))
		}

		// Go back to parent directory for replace command
		os.Chdir("..")
		// Replace the snapshot
		_, err = RunLunarCommand("replace production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		// Cleanup
		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
