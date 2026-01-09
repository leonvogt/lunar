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
		os.Chdir("tests")
		database, err := internal.ConnectToDatabase("lunar_test")
		if err != nil {
			t.Fatalf("Failed to connect to database: %v", err)
		}
		defer database.Close()

		os.Chdir("..")

		CreateTestSnapshot(t, "production")

		_, err = database.Exec("DROP TABLE users")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		_, err = database.Query("SELECT email FROM users")
		if err == nil {
			t.Errorf("Error: Table still exists")
		}

		_, err = RunLunarCommand("restore production")
		if err != nil {
			t.Errorf("Error: %v", err)
		}

		os.Chdir("tests")
		CleanupSnapshot("production")
	})
}
