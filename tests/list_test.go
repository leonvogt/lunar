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

		if string(out) != "production\n" {
			t.Errorf("Expected output to be 'production' but got '%s'", string(out))
		}

		// Go back to tests directory for cleanup
		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
