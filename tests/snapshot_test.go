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

	if !DoesDatabaseExists("lunar_snapshot_lunar_test_production") {
		t.Errorf("Database does not exist")
	}

	DropDatabase("lunar_snapshot_lunar_test_production", internal.ConnectToDatabase())
}
