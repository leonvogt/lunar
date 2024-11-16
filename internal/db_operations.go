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
