package tests

import (
	"os"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestReplace(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, "production")

		// Go back to tests directory to check database exists
		os.Chdir("tests")
		if !internal.DoesDatabaseExists(SnapshotDatabaseName("production")) {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName("production"))
		}

		// Go back to parent directory for replace command
		os.Chdir("..")
		// Replace the snapshot
		_, err := RunLunarCommand("replace production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		// Cleanup
		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
