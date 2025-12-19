package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestInit(t *testing.T) {
	// Setup test container for database connection testing
	config := SetupTestContainer(t)
	defer TeardownTestContainer(t)

	// Create dummy directory in parent directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir("..")

	os.MkdirAll("dummy", 0755)
	defer os.RemoveAll("dummy")

	// Change to dummy directory and run init
	os.Chdir("dummy")

	command := "go run ../main.go init -d lunar_test -u " + config.DatabaseUrl
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Command failed with error: %v\nOutput: %s", err, string(output))
		return
	}
	t.Logf("Command output: %s", string(output))

	if _, err := os.Stat("lunar.yml"); os.IsNotExist(err) {
		t.Errorf("Expected file 'lunar.yml' to exist but it does not.")
	} else {
		os.Remove("lunar.yml")
	}
}
