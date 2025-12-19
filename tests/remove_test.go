package tests

import (
	"os"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestRemove(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, "production")

		// Go back to tests directory to check database exists
		os.Chdir("tests")
		if !internal.DoesDatabaseExists(SnapshotDatabaseName("production")) {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName("production"))
		}

		// Go back to parent directory for remove command
		os.Chdir("..")
		// Remove the snapshot
		_, err := RunLunarCommand("remove production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		// Go back to tests directory to check database is removed
		os.Chdir("tests")
		if internal.DoesDatabaseExists(SnapshotDatabaseName("production")) {
			t.Errorf("Expected database `%s` to not exist - but it does", SnapshotDatabaseName("production"))
		}
	})
}
