package tests

import (
	"os/exec"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestReplace(t *testing.T) {
	SetupTestDatabase()

	// Create a snapshot
	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if !internal.DoesDatabaseExists("lunar_snapshot__lunar_test__production") {
		t.Errorf("Expected database `lunar_snapshot__lunar_test__production` to exist - but it does not")
	}

	// Replace the snapshot
	command = "go run ../main.go replace production"
	err = exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Cleanup
	internal.DropDatabase("lunar_snapshot__lunar_test__production")
}
