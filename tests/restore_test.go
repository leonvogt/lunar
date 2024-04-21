package tests

import (
	"os/exec"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestRestore(t *testing.T) {
	SetupTestDatabase()
	lunarTestdb := internal.ConnectToDatabase("lunar_test")

	// Create a snapshot
	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Manipulate the database
	_, err = lunarTestdb.Exec("DROP TABLE users")
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// make sure the table is dropped
	_, err = lunarTestdb.Query("SELECT email FROM users")
	if err == nil {
		t.Errorf("Error: Table still exists")
	}

	// Restore the snapshot
	command = "go run ../main.go restore production"
	err = exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Check if the table exists
	// lunarTestdb.Close()
	// internal.TerminateAllCurrentConnections("lunar_test")
	// lunarTestdb = internal.ConnectToDatabase("lunar_test")
	// _, err = lunarTestdb.Query("SELECT email FROM users")
	// if err != nil {
	// 	t.Errorf("Error: Table does not exist")
	// }

	// Cleanup
	DropDatabase("lunar_snapshot__lunar_test__production")
}
