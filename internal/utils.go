package internal

func SnapshotDatabaseName(databaseName, userProvidedSnapshotName string) string {
	return "lunar_snapshot_" + databaseName + "_" + userProvidedSnapshotName
}
