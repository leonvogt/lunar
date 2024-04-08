package tests

import (
	"context"
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/uptrace/bun"
)

func SetupTestDatabase() {
	db := internal.ConnectToTemplateDatabase()
	DropDatabase("lunar_test", db)
	CreateDatabase("lunar_test", db)
	db.Close()

	db = internal.ConnectToDatabase("lunar_test")
	CreateUsersTable("lunar_test", db)
	InsertUsers("lunar_test", db)
	db.Close()
}

func DoesDatabaseExists(databaseName string) bool {
	db := internal.ConnectToTemplateDatabase()
	rows, err := db.Query("SELECT 1 FROM pg_database WHERE datname='"+databaseName+"'", context.Background())
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	return rows.Next()
}

func DropDatabase(databaseName string, db *bun.DB) {
	ctx := context.Background()

	if _, err := db.Exec("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '"+databaseName+"' AND pid <> pg_backend_pid()", ctx); err != nil {
		panic(err)
	}

	_, err := db.Exec("DROP DATABASE IF EXISTS "+databaseName, ctx)
	if err != nil {
		panic(err)
	}
}

func CreateDatabase(databaseName string, db *bun.DB) {
	_, err := db.Exec("CREATE DATABASE "+databaseName, context.Background())
	if err != nil {
		panic(err)
	}
}

func CreateUsersTable(databaseName string, db *bun.DB) {
	_, err := db.Exec("CREATE TABLE users (id serial PRIMARY KEY, firstname VARCHAR(50), lastname VARCHAR(50), email VARCHAR(100))", context.Background())
	if err != nil {
		fmt.Println(err)
	}
}

func InsertUser(databaseName string, firstname, lastname, email string, db *bun.DB) {
	_, err := db.Exec("INSERT INTO users (firstname, lastname, email) VALUES ('"+firstname+"', '"+lastname+"', '"+email+"')", context.Background())
	if err != nil {
		fmt.Println(err)
	}
}

func InsertUsers(databaseName string, db *bun.DB) {
	InsertUser(databaseName, "John", "Doe", "john.doe@example.com", db)
	InsertUser(databaseName, "Jane", "Smith", "jane.smith@example.com", db)
	InsertUser(databaseName, "Michael", "Johnson", "michael.johnson@example.com", db)
	InsertUser(databaseName, "Emily", "Brown", "emily.brown@example.com", db)
	InsertUser(databaseName, "Christopher", "Wilson", "christopher.wilson@example.com", db)
	InsertUser(databaseName, "Jessica", "Martinez", "jessica.martinez@example.com", db)
	InsertUser(databaseName, "Daniel", "Anderson", "daniel.anderson@example.com", db)
	InsertUser(databaseName, "Sarah", "Taylor", "sarah.taylor@example.com", db)
	InsertUser(databaseName, "David", "Thomas", "david.thomas@example.com", db)
	InsertUser(databaseName, "Jennifer", "Hernandez", "jennifer.hernandez@example.com", db)
}
