package tests

import (
	"os/exec"
	"testing"
)

func TestSnapshotList(t *testing.T) {
	SetupTestDatabase()

	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	command = "go run ../main.go list"
	out, err := exec.Command("sh", "-c", command).Output()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if string(out) != "production\n" {
		t.Errorf("Expected output to be 'production' but got '%s'", string(out))
	}

	DropDatabase("lunar_snapshot__lunar_test__production")
}
