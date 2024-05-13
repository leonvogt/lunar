package tests

import (
	"database/sql"
	"fmt"

	"github.com/leonvogt/lunar/internal"
)

func SetupTestDatabase() {
	internal.DropDatabase("lunar_test")
	internal.CreateDatabase("lunar_test")

	db := internal.ConnectToDatabase("lunar_test")
	defer db.Close()
	CreateUsersTable("lunar_test", db)
	InsertUsers("lunar_test", db)
}

func CreateUsersTable(databaseName string, db *sql.DB) {
	_, err := db.Exec("CREATE TABLE users (id serial PRIMARY KEY, firstname VARCHAR(50), lastname VARCHAR(50), email VARCHAR(100))")
	if err != nil {
		fmt.Println(err)
	}
}

func InsertUser(databaseName string, firstname, lastname, email string, db *sql.DB) {
	_, err := db.Exec("INSERT INTO users (firstname, lastname, email) VALUES ('" + firstname + "', '" + lastname + "', '" + email + "')")
	if err != nil {
		fmt.Println(err)
	}
}

func InsertUsers(databaseName string, db *sql.DB) {
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
