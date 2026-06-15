package tests

import (
	"strings"
	"testing"

	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/provider/postgres"
)

// Using the legacy "____" naming scheme, simulating a snapshot taken by an older lunar version.
func createLegacySnapshot(t *testing.T, snapshotName string) (snapshotDB, copyDB string) {
	t.Helper()

	config, err := internal.ReadConfig()
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	db, err := postgres.ConnectToMaintenanceDatabaseWithURL(config.DatabaseUrl)
	if err != nil {
		t.Fatalf("Failed to connect to maintenance database: %v", err)
	}
	defer db.Close()

	snapshotDB = "lunar_snapshot____lunar_test____" + snapshotName
	copyDB = snapshotDB + "_copy"
	for _, name := range []string{snapshotDB, copyDB} {
		if _, err := db.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
			t.Fatalf("Failed to create legacy database %s: %v", name, err)
		}
	}

	return snapshotDB, copyDB
}

func dropDatabases(names ...string) {
	config, err := internal.ReadConfig()
	if err != nil {
		return
	}

	db, err := postgres.ConnectToMaintenanceDatabaseWithURL(config.DatabaseUrl)
	if err != nil {
		return
	}
	defer db.Close()

	for _, name := range names {
		db.Exec(`DROP DATABASE IF EXISTS "` + name + `"`)
	}
}

// Snapshots created by older lunar versions use the legacy naming scheme and must
// remain listable and restorable after upgrading (dual-read, no breaking change).
func TestPostgres_DualReadLegacySnapshot(t *testing.T) {
	const snapshotName = "legacy_snap"

	SetupTestDatabase(t)
	defer TeardownTestContainer(t)

	WithTestDirectory(t, func() {
		snapshotDB, copyDB := createLegacySnapshot(t, snapshotName)
		defer dropDatabases(snapshotDB, copyDB)

		// A legacy snapshot must show up in `list`.
		out, err := RunLunarCommand("list")
		if err != nil {
			t.Fatalf("Error listing snapshots: %v\nOutput: %s", err, string(out))
		}
		if !strings.Contains(string(out), snapshotName) {
			t.Errorf("Expected `list` to show legacy snapshot %q, got: %s", snapshotName, string(out))
		}

		// ...and must be restorable. (The `restore` command exits 0 even on a
		// logical failure, so assert on the success message, not the exit code.)
		out, err = RunLunarCommand("restore " + snapshotName)
		if err != nil {
			t.Fatalf("Error restoring legacy snapshot: %v\nOutput: %s", err, string(out))
		}
		if !strings.Contains(string(out), "restored successfully") {
			t.Fatalf("Expected first restore to succeed, got: %s", string(out))
		}

		if out, err = RunLunarCommand("restore recreate-copy " + snapshotName); err != nil {
			t.Fatalf("Error recreating legacy snapshot copy: %v\nOutput: %s", err, string(out))
		}
		copyExists, err := DoesDatabaseExist(copyDB)
		if err != nil {
			t.Fatalf("Error checking legacy copy existence: %v", err)
		}
		if !copyExists {
			t.Fatalf("Expected legacy copy %q to be regenerated after restore", copyDB)
		}

		// The regenerated copy must let the legacy snapshot be restored again.
		out, err = RunLunarCommand("restore " + snapshotName)
		if err != nil {
			t.Fatalf("Error restoring legacy snapshot a second time: %v\nOutput: %s", err, string(out))
		}
		if !strings.Contains(string(out), "restored successfully") {
			t.Fatalf("Expected second restore to succeed, got: %s", string(out))
		}
	})
}
