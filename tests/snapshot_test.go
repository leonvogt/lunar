package tests

import (
	"os"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestMain(m *testing.M) {
	// This will be called once before all tests in this package
	code := m.Run()
	os.Exit(code)
}

func TestSnapshot(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		// Verify we're in the right place
		if _, err := os.Stat("main.go"); os.IsNotExist(err) {
			wd, _ := os.Getwd()
			t.Fatalf("main.go not found in current directory. Current dir: %s", wd)
		}

		CreateTestSnapshot(t, "production")

		// Go back to tests directory to check database
		os.Chdir("tests")
		if !internal.DoesDatabaseExists(SnapshotDatabaseName("production")) {
			t.Errorf("Expected database `%s` to exist - but it does not", SnapshotDatabaseName("production"))
		}

		// Cleanup
		CleanupSnapshot("production")
	})
}

func TestSnapshotAlreadyExists(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, "production")

		// Try to create a snapshot with the same name
		out, err := RunLunarCommand("snapshot production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		expectedOutput := "snapshot with name production already exists\n"
		if string(out) != expectedOutput {
			t.Errorf("Expected output to be '%v' but got '%v'", expectedOutput, string(out))
		}

		// Go back to tests directory for cleanup
		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
