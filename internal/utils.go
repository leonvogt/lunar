package internal

import (
	"strings"
)

func SnapshotDatabaseName(databaseName, userProvidedSnapshotName string) string {
	return "lunar_snapshot__" + databaseName + "__" + userProvidedSnapshotName
}

func SnapshotDatabasesForDatabase(databaseName string) []string {
	snapshots := make([]string, 0)
	allLunarSnapshotDatabases := AllSnapshotDatabases()
	for _, db := range allLunarSnapshotDatabases {
		// split the database name from the snapshot name, by splitting at the "__"
		// the first part is the database name, the second part is the snapshot name
		parts := strings.Split(db, "__")
		if parts[1] == databaseName {
			snapshots = append(snapshots, parts[2])
		}
	}

	return snapshots
}
