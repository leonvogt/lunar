package tests

import (
	"os"
	"testing"
)

func TestSnapshotList(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		CreateTestSnapshot(t, "production")

		out, err := RunLunarCommand("list")
		if err != nil {
			t.Errorf("Error running list command: %v", err)
		}

		// Test if the output starts with "production"
		if string(out[:10]) != "production" {
			t.Errorf("Expected output to be 'production' but got '%s'", string(out))
		}

		// Go back to tests directory for cleanup
		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
