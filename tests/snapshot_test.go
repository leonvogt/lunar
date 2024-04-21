package tests

import (
	"os/exec"
	"testing"
)

func TestSnapshot(t *testing.T) {
	SetupTestDatabase()

	command := "go run ../main.go snapshot production"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if !DoesDatabaseExists("lunar_snapshot__lunar_test__production") {
		t.Errorf("Expected database `lunar_snapshot__lunar_test__production` to exist - but it does not")
	}

	DropDatabase("lunar_snapshot__lunar_test__production")
}
