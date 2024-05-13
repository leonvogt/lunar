package tests

import (
	"os/exec"
	"testing"

	"github.com/leonvogt/lunar/internal"
)

func TestSnapshot(t *testing.T) {
	SetupTestDatabase()

	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if !internal.DoesDatabaseExists("lunar_snapshot__lunar_test__production") {
		t.Errorf("Expected database `lunar_snapshot__lunar_test__production` to exist - but it does not")
	}

	internal.DropDatabase("lunar_snapshot__lunar_test__production")
}

func TestSnapshotAlreadyExists(t *testing.T) {
	SetupTestDatabase()

	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	// Try to create a snapshot with the same name
	command = "go run ../main.go snapshot production"
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	expectedOutput := "Snapshot with name production already exists\n"
	if string(out) != expectedOutput {
		t.Errorf("Expected output to be '%v' but got '%v'", expectedOutput, string(out))
	}

	internal.DropDatabase("lunar_snapshot__lunar_test__production")
}
