package tests

import (
	"os"
	"testing"
)

func TestRemove(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, "production")

		os.Chdir("tests")
		exists, err := DoesDatabaseExist(SnapshotDatabaseName("production"))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if !exists {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName("production"))
		}

		os.Chdir("..")
		_, err = RunLunarCommand("remove production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		os.Chdir("tests")
		exists, err = DoesDatabaseExist(SnapshotDatabaseName("production"))
		if err != nil {
			t.Fatalf("Error checking database existence: %v", err)
		}
		if exists {
			t.Errorf("Expected database `%s` to not exist - but it does", SnapshotDatabaseName("production"))
		}
	})
}
