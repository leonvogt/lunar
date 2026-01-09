package internal

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func AllDatabases(db *sql.DB) []string {
	databases := make([]string, 0)

	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

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

func AllSnapshotDatabases() ([]string, error) {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	databases := AllDatabases(db)
	snapshotDatabases := make([]string, 0)

	expectedPrefix := "lunar_snapshot" + SEPARATOR

	for _, database := range databases {
		if len(database) >= len(expectedPrefix) && database[:len(expectedPrefix)] == expectedPrefix {
			snapshotDatabases = append(snapshotDatabases, database)
		}
	}

	return snapshotDatabases, nil
}

func OpenDatabaseConnection(databaseUrl string, sslMode bool) (*sql.DB, error) {
	if !sslMode {
		databaseUrl += "?sslmode=disable"
	}

	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	return db, nil
}

func ConnectToDatabase(databaseName string) *sql.DB {
	config, err := ReadConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %v", err))
	}

	databaseUrl := config.DatabaseUrl
	db, err := OpenDatabaseConnection(databaseUrl+databaseName, false)
	if err != nil {
		panic(fmt.Errorf("failed to connect to database: %v", err))
	}
	return db
}

// ConnectToMaintenanceDatabase connects to a maintenance database for administrative operations.
// It tries the configured maintenance_database first, then falls back to postgres and template1.
func ConnectToMaintenanceDatabase() *sql.DB {
	config, err := ReadConfig()
	if err != nil {
		panic(fmt.Errorf("failed to read config: %v", err))
	}

	// Build list of databases to try
	databasesToTry := []string{}
	if config.GetMaintenanceDatabase() != "" {
		databasesToTry = append(databasesToTry, config.GetMaintenanceDatabase())
	}
	databasesToTry = append(databasesToTry, DefaultMaintenanceDatabases()...)

	var lastErr error
	for _, dbName := range databasesToTry {
		db, err := OpenDatabaseConnection(config.DatabaseUrl+dbName, false)
		if err != nil {
			lastErr = err
			continue
		}

		// Test the connection
		if err := db.Ping(); err != nil {
			db.Close()
			lastErr = err
			continue
		}

		return db
	}

	panic(fmt.Errorf("failed to connect to any maintenance database (tried: %v): %v", databasesToTry, lastErr))
}

// ConnectToMaintenanceDatabaseWithUrl connects to a maintenance database using a specific URL.
// Useful during initialization when config may not exist yet.
func ConnectToMaintenanceDatabaseWithUrl(databaseUrl string) (*sql.DB, error) {
	for _, dbName := range DefaultMaintenanceDatabases() {
		db, err := OpenDatabaseConnection(databaseUrl+dbName, false)
		if err != nil {
			continue
		}

		// Test the connection
		if err := db.Ping(); err != nil {
			db.Close()
			continue
		}

		return db, nil
	}

	return nil, fmt.Errorf("failed to connect to any maintenance database (tried: %v)", DefaultMaintenanceDatabases())
}

// Deprecated: Use ConnectToMaintenanceDatabase instead
func ConnectToTemplateDatabase() *sql.DB {
	return ConnectToMaintenanceDatabase()
}

// Deprecated: Use ConnectToMaintenanceDatabase instead
func ConnectToPostgresDatabase() *sql.DB {
	return ConnectToMaintenanceDatabase()
}

func TerminateAllCurrentConnections(databaseName string) error {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	query := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1
		AND pid <> pg_backend_pid()`

	_, err := db.Exec(query, databaseName)
	if err != nil {
		return err
	}
	return nil
}

func DoesDatabaseExists(databaseName string) bool {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists)
	if err != nil {
		log.Fatal(err)
	}

	return exists
}

func TestConnection(db *sql.DB) error {
	db.SetConnMaxLifetime(5 * time.Second)

	err := db.Ping()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	return nil
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

func CreateDatabaseWithTemplate(db *sql.DB, databaseName, templateDatabaseName string) error {
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", databaseName, templateDatabaseName))
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}
	return nil
}

func DropDatabase(databaseName string) {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	// Note: PostgreSQL doesn't support parameterized DDL for database names
	// The databaseName comes from internal snapshot naming, not user input
	_, err := db.Exec("DROP DATABASE IF EXISTS " + databaseName)
	if err != nil {
		panic(err)
	}
}

func RenameDatabase(oldName, newName string) error {
	db := ConnectToTemplateDatabase()
	defer db.Close()

	// Note: PostgreSQL doesn't support parameterized DDL for database names
	_, err := db.Exec("ALTER DATABASE " + oldName + " RENAME TO " + newName)
	if err != nil {
		return fmt.Errorf("failed to rename database: %v", err)
	}
	return nil
}

func RestoreSnapshot(databaseName, snapshotCopyName string) error {
	DropDatabase(databaseName)

	// Rename the _copy database to the target database for instant restore
	if err := RenameDatabase(snapshotCopyName, databaseName); err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	return nil
}
