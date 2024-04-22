package tests

import (
	"os/exec"
	"testing"
)

func TestRemove(t *testing.T) {
	SetupTestDatabase()

	// Create a snapshot
	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if !DoesDatabaseExists("lunar_snapshot__lunar_test__production") {
		t.Errorf("Expected database `lunar_snapshot__lunar_test__production` to exist - but it does not")
	}

	// Remove the snapshot
	command = "go run ../main.go remove production"
	err = exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if DoesDatabaseExists("lunar_snapshot__lunar_test__production") {
		t.Errorf("Expected database `lunar_snapshot__lunar_test__production` to not exist - but it does")
	}
}
