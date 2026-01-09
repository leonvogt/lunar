package internal

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

func AllDatabases(database *sql.DB) ([]string, error) {
	databases := make([]string, 0)

	rows, err := database.Query("SELECT datname FROM pg_database WHERE datistemplate = false")
	if err != nil {
		return nil, fmt.Errorf("failed to query databases: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var databaseName string
		if err := rows.Scan(&databaseName); err != nil {
			return nil, fmt.Errorf("failed to scan database name: %v", err)
		}
		databases = append(databases, databaseName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database rows: %v", err)
	}

	return databases, nil
}

func GetDatabaseAge(database *sql.DB, databaseName string) (time.Time, error) {
	var creationTime time.Time
	query := `
		SELECT (pg_stat_file('base/'|| oid ||'/PG_VERSION')).modification
		FROM pg_database
		WHERE datname = $1`

	err := database.QueryRow(query, databaseName).Scan(&creationTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get database age: %v", err)
	}

	return creationTime, nil
}

func AllSnapshotDatabases() ([]string, error) {
	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return nil, err
	}
	defer database.Close()

	databases, err := AllDatabases(database)
	if err != nil {
		return nil, err
	}

	snapshotDatabases := make([]string, 0)
	expectedPrefix := "lunar_snapshot" + SEPARATOR

	for _, databaseName := range databases {
		if len(databaseName) >= len(expectedPrefix) && databaseName[:len(expectedPrefix)] == expectedPrefix {
			snapshotDatabases = append(snapshotDatabases, databaseName)
		}
	}

	return snapshotDatabases, nil
}

func OpenDatabaseConnection(databaseUrl string, sslMode bool) (*sql.DB, error) {
	if !sslMode {
		databaseUrl += "?sslmode=disable"
	}

	database, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %v", err)
	}

	return database, nil
}

func ConnectToDatabase(databaseName string) (*sql.DB, error) {
	config, err := ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	database, err := OpenDatabaseConnection(config.DatabaseUrl+databaseName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return database, nil
}

// ConnectToMaintenanceDatabase connects to a maintenance database for administrative operations.
// It tries the configured maintenance_database first, then falls back to postgres and template1.
func ConnectToMaintenanceDatabase() (*sql.DB, error) {
	config, err := ReadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	databasesToTry := []string{}
	if config.GetMaintenanceDatabase() != "" {
		databasesToTry = append(databasesToTry, config.GetMaintenanceDatabase())
	}
	databasesToTry = append(databasesToTry, DefaultMaintenanceDatabases()...)

	var lastErr error
	for _, databaseName := range databasesToTry {
		database, err := OpenDatabaseConnection(config.DatabaseUrl+databaseName, false)
		if err != nil {
			lastErr = err
			continue
		}

		if err := database.Ping(); err != nil {
			database.Close()
			lastErr = err
			continue
		}

		return database, nil
	}

	return nil, fmt.Errorf("failed to connect to any maintenance database (tried: %v): %v", databasesToTry, lastErr)
}

// ConnectToMaintenanceDatabaseWithUrl connects to a maintenance database using a specific URL.
// Useful during initialization when config may not exist yet.
func ConnectToMaintenanceDatabaseWithUrl(databaseUrl string) (*sql.DB, error) {
	for _, databaseName := range DefaultMaintenanceDatabases() {
		database, err := OpenDatabaseConnection(databaseUrl+databaseName, false)
		if err != nil {
			continue
		}

		if err := database.Ping(); err != nil {
			database.Close()
			continue
		}

		return database, nil
	}

	return nil, fmt.Errorf("failed to connect to any maintenance database (tried: %v)", DefaultMaintenanceDatabases())
}

func TerminateAllCurrentConnections(databaseName string) error {
	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return err
	}
	defer database.Close()

	query := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1
		AND pid <> pg_backend_pid()`

	_, err = database.Exec(query, databaseName)
	return err
}

func DoesDatabaseExist(databaseName string) (bool, error) {
	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return false, err
	}
	defer database.Close()

	var exists bool
	err = database.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %v", err)
	}

	return exists, nil
}

func TestConnection(database *sql.DB) error {
	database.SetConnMaxLifetime(5 * time.Second)

	if err := database.Ping(); err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	return nil
}

func CreateDatabase(databaseName string) error {
	TerminateAllCurrentConnections("template1")

	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return err
	}
	defer database.Close()

	_, err = database.Exec("CREATE DATABASE " + databaseName)
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}

	return nil
}

func CreateDatabaseWithTemplate(database *sql.DB, databaseName, templateDatabaseName string) error {
	_, err := database.Exec(fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", databaseName, templateDatabaseName))
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}
	return nil
}

func DropDatabase(databaseName string) error {
	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return err
	}
	defer database.Close()

	// Note: PostgreSQL doesn't support parameterized DDL for database names
	// The databaseName comes from internal snapshot naming, not user input
	_, err = database.Exec("DROP DATABASE IF EXISTS " + databaseName)
	if err != nil {
		return fmt.Errorf("failed to drop database: %v", err)
	}

	return nil
}

func RenameDatabase(oldName, newName string) error {
	database, err := ConnectToMaintenanceDatabase()
	if err != nil {
		return err
	}
	defer database.Close()

	_, err = database.Exec("ALTER DATABASE " + oldName + " RENAME TO " + newName)
	if err != nil {
		return fmt.Errorf("failed to rename database: %v", err)
	}
	return nil
}

func RestoreSnapshot(databaseName, snapshotCopyName string) error {
	if err := DropDatabase(databaseName); err != nil {
		return err
	}

	if err := RenameDatabase(snapshotCopyName, databaseName); err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	return nil
}
