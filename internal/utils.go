package internal

import (
	"strings"
)

var SEPARATOR = "____"

func SnapshotDatabaseName(databaseName, userProvidedSnapshotName string) string {
	return "lunar_snapshot" + SEPARATOR + databaseName + SEPARATOR + userProvidedSnapshotName
}

func SnapshotCopyDatabaseName(databaseName, snapshotName string) string {
	return SnapshotDatabaseName(databaseName, snapshotName) + "_copy"
}

func SnapshotDatabasesForDatabase(databaseName string) ([]string, error) {
	allLunarSnapshotDatabases, err := AllSnapshotDatabases()
	if err != nil {
		return nil, err
	}

	snapshots := make([]string, 0)
	for _, snapshotDatabase := range allLunarSnapshotDatabases {
		parts := strings.Split(snapshotDatabase, SEPARATOR)

		if len(parts) >= 3 && parts[1] == databaseName {
			snapshotName := parts[2]
			if !strings.HasSuffix(snapshotName, "_copy") {
				snapshots = append(snapshots, snapshotName)
			}
		}
	}

	return snapshots, nil
}
