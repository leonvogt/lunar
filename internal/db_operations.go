package internal

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func AllDatabases() []string {
	db := ConnectToTemplateDatabase()

	ctx := context.Background()
	databases := make([]string, 0)
	if err := db.NewSelect().Column("datname").Model(&databases).Table("pg_database").Where("datistemplate = false").Scan(ctx); err != nil {
		panic(err)
	}

	return databases
}

func ConnectToDatabase(databaseName string) *bun.DB {
	config, err := ReadConfig()
	if err != nil {
		panic(err)
	}

	databaseUrl := config.DatabaseUrl
	if databaseName == "template1" {
		databaseUrl += "template1?sslmode=disable"
	} else {
		databaseUrl += databaseName + "?sslmode=disable"
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseUrl)))
	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}

func ConnectToTemplateDatabase() *bun.DB {
	return ConnectToDatabase("template1")
}

func ConnectToDatabaseFromConfig() *bun.DB {
	return ConnectToDatabase("")
}

func CreateSnapshot(databaseName, snapshotName string) {
	db := ConnectToTemplateDatabase()

	ctx := context.Background()
	if _, err := db.Exec("CREATE DATABASE "+snapshotName+" TEMPLATE "+databaseName, ctx); err != nil {
		panic(err)
	}
}

func RestoreSnapshot(databaseName, snapshotName string) {
	db := ConnectToTemplateDatabase()

	ctx := context.Background()
	if _, err := db.Exec("DROP DATABASE IF EXISTS "+databaseName, ctx); err != nil {
		panic(err)
	}
	if _, err := db.Exec("CREATE DATABASE "+databaseName+" TEMPLATE "+snapshotName, ctx); err != nil {
		panic(err)
	}
}

func TerminateAllCurrentConnections(databaseName string) {
	db := ConnectToTemplateDatabase()

	ctx := context.Background()
	if _, err := db.Exec("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '"+databaseName+"' AND pid <> pg_backend_pid()", ctx); err != nil {
		panic(err)
	}
}
