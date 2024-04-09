package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestInit(t *testing.T) {
	command := "cd dummy && go run ../../main.go init -u postgres://localhost:5432/ -n lunar_test"
	err := exec.Command("sh", "-c", command).Run()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if _, err := os.Stat("dummy/lunar.yml"); os.IsNotExist(err) {
		t.Errorf("Expected file 'lunar.yml' to exist but it does not.")
	}

	os.Remove("dummy/lunar.yml")
}
