package internal

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/lib/pq"
)

var dbDebugLog *log.Logger

func init() {
	// Create debug log file in /tmp
	logFile, err := os.OpenFile(filepath.Join(os.TempDir(), "lunar-db-debug.log"),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Warning: Could not create db debug log: %v\n", err)
		return
	}

	dbDebugLog = log.New(logFile, "", log.LstdFlags)
}

func logDbDebug(format string, v ...interface{}) {
	if dbDebugLog != nil {
		dbDebugLog.Printf(format, v...)
	}
}

func CreateSnapshotWithRetry(databaseName, snapshotName string, maxRetries int) error {
	logDbDebug("Starting CreateSnapshotWithRetry: database=%s snapshot=%s retries=%d",
		databaseName, snapshotName, maxRetries)

	var err error
	for i := 0; i < maxRetries; i++ {
		if err = createSnapshotOnce(databaseName, snapshotName); err == nil {
			logDbDebug("CreateSnapshotWithRetry succeeded on attempt %d", i+1)
			return nil
		}
		logDbDebug("Attempt %d failed: %v", i+1, err)
		time.Sleep(time.Second * 2)
	}
	return fmt.Errorf("failed to create snapshot after %d attempts: %v", maxRetries, err)
}

func CreateSnapshotCopyWithRetry(snapshotName string, maxRetries int) error {
	logDbDebug("Starting CreateSnapshotCopyWithRetry: snapshot=%s retries=%d",
		snapshotName, maxRetries)

	var err error
	for i := 0; i < maxRetries; i++ {
		if err = createSnapshotCopyOnce(snapshotName); err == nil {
			logDbDebug("CreateSnapshotCopyWithRetry succeeded on attempt %d", i+1)
			return nil
		}
		logDbDebug("Copy attempt %d failed: %v", i+1, err)
		time.Sleep(time.Second * 2)
	}
	return fmt.Errorf("failed to create snapshot copy after %d attempts: %v", maxRetries, err)
}

func createSnapshotOnce(databaseName, snapshotName string) error {
	logDbDebug("Starting createSnapshotOnce: database=%s snapshot=%s",
		databaseName, snapshotName)

	if err := terminateConnections(databaseName); err != nil {
		logDbDebug("Failed to terminate connections: %v", err)
		return fmt.Errorf("failed to terminate connections: %v", err)
	}

	db := ConnectToTemplateDatabase()
	defer db.Close()

	logDbDebug("Creating snapshot...")
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", snapshotName, databaseName))
	if err != nil {
		logDbDebug("Failed to create snapshot: %v", err)
		return fmt.Errorf("failed to create snapshot: %v", err)
	}

	logDbDebug("Snapshot created successfully")
	return nil
}

func createSnapshotCopyOnce(snapshotName string) error {
	snapshotNameCopy := snapshotName + "_copy"
	logDbDebug("Starting createSnapshotCopyOnce: snapshot=%s copy=%s",
		snapshotName, snapshotNameCopy)

	if err := dropDatabaseIfExists(snapshotNameCopy); err != nil {
		logDbDebug("Failed to drop existing copy: %v", err)
		return fmt.Errorf("failed to drop existing copy: %v", err)
	}

	if err := terminateConnections(snapshotName); err != nil {
		logDbDebug("Failed to terminate connections: %v", err)
		return fmt.Errorf("failed to terminate connections: %v", err)
	}

	db := ConnectToTemplateDatabase()
	defer db.Close()

	logDbDebug("Creating snapshot copy...")
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", snapshotNameCopy, snapshotName))
	if err != nil {
		logDbDebug("Failed to create snapshot copy: %v", err)
		return fmt.Errorf("failed to create snapshot copy: %v", err)
	}

	logDbDebug("Snapshot copy created successfully")
	return nil
}

func dropDatabaseIfExists(databaseName string) error {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	_, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	return err
}

// Add debug logging to existing functions...
func terminateConnections(databaseName string) error {
	logDbDebug("Terminating connections for database: %s", databaseName)

	db := ConnectToTemplateDatabase()
	defer db.Close()

	query := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1
		AND pid <> pg_backend_pid()`

	result, err := db.Exec(query, databaseName)
	if err != nil {
		logDbDebug("Error terminating connections: %v", err)
		return err
	}

	affected, _ := result.RowsAffected()
	logDbDebug("Terminated %d connections", affected)
	return nil
}

func AllDatabases(db *sql.DB) []string {
	databases := make([]string, 0)

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		panic(err)
	}
	db.Close()

	for rows.Next() {
		var database string
		err := rows.Scan(&database)
		if err != nil {
			panic(err)
		}
		databases = append(databases, database)
	}

	return databases
}

func AllSnapshotDatabases() []string {
	db := ConnectToTemplateDatabase()
	databases := AllDatabases(db)
	snapshotDatabases := make([]string, 0)
	for _, database := range databases {
		if len(database) >= 16 && database[:16] == "lunar_snapshot__" {
			snapshotDatabases = append(snapshotDatabases, database)
		}
	}
	return snapshotDatabases
}

func ConnectToDatabase(databaseName string) *sql.DB {
	config, err := ReadConfig()
	if err != nil {
		panic(err)
	}

	databaseUrl := config.DatabaseUrl
	return OpenDatabaseConnection(databaseUrl, false)
}

func OpenDatabaseConnection(databaseUrl string, sslMode bool) *sql.DB {
	if !sslMode {
		databaseUrl += "?sslmode=disable"
	}

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		panic(err)
	}

	return db
}

func ConnectToTemplateDatabase() *sql.DB {
	return ConnectToDatabase("template1")
}

func ConnectToDatabaseFromConfig() *sql.DB {
	return ConnectToDatabase("")
}

func CreateSnapshot(databaseName, snapshotName string) {
	db := ConnectToTemplateDatabase()

	if _, err := db.Exec("CREATE DATABASE " + snapshotName + " TEMPLATE " + databaseName); err != nil {
		panic(err)
	}
	db.Close()
}

func CreateSnapshotCopy(snapshotName string) {
	db := ConnectToTemplateDatabase()
	snapshotNameCopy := snapshotName + "_copy"
	DropDatabase(snapshotNameCopy)

	if _, err := db.Exec("CREATE DATABASE " + snapshotNameCopy + " TEMPLATE " + snapshotName); err != nil {
		panic(err)
	}
}

func RestoreSnapshot(databaseName, snapshotName string) {
	DropDatabase(databaseName)

	db := ConnectToTemplateDatabase()
	defer db.Close()

	_, err := db.Query("CREATE DATABASE " + databaseName + " TEMPLATE " + snapshotName)
	if err != nil {
		panic(err)
	}
}

func DoesDatabaseExists(databaseName string) bool {
	db := ConnectToTemplateDatabase()

	rows, err := db.Query("SELECT 1 FROM pg_database WHERE datname='" + databaseName + "'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	return rows.Next()
}

func CreateDatabase(databaseName string) {
	TerminateAllCurrentConnections("template1")
	db := ConnectToTemplateDatabase()
	defer db.Close()

	_, err := db.Exec("CREATE DATABASE " + databaseName)
	if err != nil {
		panic(err)
	}
}

func DropDatabase(databaseName string) {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	_, err := db.Query("DROP DATABASE IF EXISTS " + databaseName)
	if err != nil {
		panic(err)
	}
}

func TerminateAllCurrentConnections(databaseName string) {
	db := ConnectToTemplateDatabase()

	_, err := db.Exec("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '" + databaseName + "' AND pid <> pg_backend_pid()")
	if err != nil {
		panic(err)
	}
	db.Close()
}

func TestConnection(db *sql.DB) error {
	db.SetConnMaxLifetime(5 * time.Second)

	err := db.Ping()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	return nil
}
