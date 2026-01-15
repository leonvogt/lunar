package tests

import (
	"os"
	"os/exec"
	"testing"
)

func TestInit(t *testing.T) {
	config := SetupTestContainer(t)
	defer TeardownTestContainer(t)

	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir("..")
	os.MkdirAll("dummy", 0755)
	defer os.RemoveAll("dummy")
	os.Chdir("dummy")

	// Test Postgres init
	pgCmd := "go run ../main.go init --provider postgres -d lunar_test -u '" + config.DatabaseUrl + "'"
	cmd := exec.Command("sh", "-c", pgCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Postgres init failed: %v\nOutput: %s", err, string(output))
	}
	if _, err := os.Stat("lunar.yml"); os.IsNotExist(err) {
		t.Errorf("[Postgres] Expected file 'lunar.yml' to exist but it does not.")
	} else {
		os.Remove("lunar.yml")
	}

	// Test SQLite init
	sqlitePath := "test.db"
	f, ferr := os.Create(sqlitePath)
	if ferr != nil {
		t.Fatalf("Failed to create dummy sqlite file: %v", ferr)
	}
	f.Close()

	sqliteCmd := "go run ../main.go init --provider sqlite --database-path '" + sqlitePath + "' --snapshot-directory './.lunar_snapshots'"
	cmd = exec.Command("sh", "-c", sqliteCmd)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("SQLite init failed: %v\nOutput: %s", err, string(output))
	}
	if _, err := os.Stat("lunar.yml"); os.IsNotExist(err) {
		t.Errorf("[SQLite] Expected file 'lunar.yml' to exist but it does not.")
	} else {
		os.Remove("lunar.yml")
	}

	os.Remove(sqlitePath)
}
