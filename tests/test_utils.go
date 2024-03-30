package tests

import (
	"context"
	"os/exec"

	"github.com/leonvogt/lunar/internal"
)

func SetupTestDatabase() {
	DropDatabase("lunar_test")
	err := exec.Command("sh", "-c", "psql -U postgres -d template1 -f lunar_test_db.sql").Run()
	if err != nil {
		panic(err)
	}
}

func DoesDatabaseExists(databaseName string) bool {
	db := internal.ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")
	ctx := context.Background()
	rows, err := db.Query("SELECT 1 FROM pg_database WHERE datname='"+databaseName+"'", ctx)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	return rows.Next()
}

func DropDatabase(databaseName string) {
	db := internal.ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")
	ctx := context.Background()
	_, err := db.Exec("DROP DATABASE IF EXISTS "+databaseName, ctx)
	if err != nil {
		panic(err)
	}
}
