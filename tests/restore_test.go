package tests

import (
	"os/exec"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestRestore(t *testing.T) {
	SetupTestDatabase()
	db := internal.ConnectToDatabase("lunar_test")

	// Create a snapshot
	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Manipulate the database
	_, err = db.Exec("DROP TABLE users")
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// make sure the table is dropped
	_, err = db.Query("SELECT email FROM users")
	if err == nil {
		t.Errorf("Error: Table still exists")
	}
	db.Close()

	// Restore the snapshot
	command = "go run ../main.go restore production"
	err = exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Check if the table exists
	db = internal.ConnectToDatabase("lunar_test")
	_, err = db.Query("SELECT email FROM users")
	if err != nil {
		t.Errorf("Error: Table does not exist")
	}
	db.Close()

	// Cleanup
	db = internal.ConnectToTemplateDatabase()
	DropDatabase("lunar_snapshot_lunar_test_production", db)
	db.Close()
}
