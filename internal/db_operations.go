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

func AllSnapshotDatabases() ([]string, error) {
	db := ConnectToTemplateDatabase()
	databases := AllDatabases(db)
	snapshotDatabases := make([]string, 0)

	expectedPrefix := "lunar_snapshot" + SEPERATOR

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

func ConnectToTemplateDatabase() *sql.DB {
	return ConnectToDatabase("template1")
}

func ConnectToPostgresDatabase() *sql.DB {
	return ConnectToDatabase("postgres")
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

	rows, err := db.Query("SELECT 1 FROM pg_database WHERE datname='" + databaseName + "'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	return rows.Next()
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

	_, err := db.Query("DROP DATABASE IF EXISTS " + databaseName)
	if err != nil {
		panic(err)
	}
}

func RestoreSnapshot(databaseName, snapshotName string) error {
	DropDatabase(databaseName)

	db := ConnectToTemplateDatabase()
	defer db.Close()

	_, err := db.Query("CREATE DATABASE " + databaseName + " TEMPLATE " + snapshotName)
	if err != nil {
		return fmt.Errorf("failed to restore snapshot: %v", err)
	}

	return nil
}
