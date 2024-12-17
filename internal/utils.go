package internal

import (
	"strings"
)

var SEPERATOR = "____"

func SnapshotDatabaseName(databaseName, userProvidedSnapshotName string) string {
	return "lunar_snapshot" + SEPERATOR + databaseName + SEPERATOR + userProvidedSnapshotName
}

func SnapshotDatabasesForDatabase(databaseName string) []string {
	snapshots := make([]string, 0)
	allLunarSnapshotDatabases, _ := AllSnapshotDatabases()
	for _, db := range allLunarSnapshotDatabases {
		// Split the database name from the snapshot name, by splitting at the SEPERATOR
		// The first part is the database name, the second part is the snapshot name
		parts := strings.Split(db, SEPERATOR)
		if parts[1] == databaseName {
			snapshots = append(snapshots, parts[2])
		}
	}

	return snapshots
}
