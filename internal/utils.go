package internal

import (
	"fmt"
	"strings"
)

var SEPERATOR = "____"

func SnapshotDatabaseName(databaseName, userProvidedSnapshotName string) string {
	return "lunar_snapshot" + SEPERATOR + databaseName + SEPERATOR + userProvidedSnapshotName
}

func SnapshotDatabasesForDatabase(databaseName string) []string {
	snapshots := make([]string, 0)
	allLunarSnapshotDatabases, err := AllSnapshotDatabases()
	if err != nil {
		fmt.Printf("Error getting snapshot databases: %v\n", err)
		return snapshots
	}
	for _, db := range allLunarSnapshotDatabases {
		// Split the database name from the snapshot name, by splitting at the SEPERATOR
		// The first part is the database name, the second part is the snapshot name
		parts := strings.Split(db, SEPERATOR)

		if len(parts) >= 3 && parts[1] == databaseName {
			snapshotName := parts[2]
			if !strings.HasSuffix(snapshotName, "_copy") {
				snapshots = append(snapshots, snapshotName)
			}
		}
	}

	return snapshots
}
