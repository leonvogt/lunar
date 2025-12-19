package tests

import (
	"os"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestRestore(t *testing.T) {
	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		// Connect to database in tests directory context
		os.Chdir("tests")
		lunarTestdb := internal.ConnectToDatabase("lunar_test")
		defer lunarTestdb.Close()

		// Go back to parent directory for snapshot creation
		os.Chdir("..")

		// Create a snapshot
		CreateTestSnapshot(t, "production")

		// Manipulate the database
		_, err := lunarTestdb.Exec("DROP TABLE users")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		// make sure the table is dropped
		_, err = lunarTestdb.Query("SELECT email FROM users")
		if err == nil {
			t.Errorf("Error: Table still exists")
		}

		// Restore the snapshot
		_, err = RunLunarCommand("restore production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		// Cleanup
		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
