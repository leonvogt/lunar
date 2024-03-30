package internal

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func AllDatabases() []string {
	db := ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")

	ctx := context.Background()
	databases := make([]string, 0)
	if err := db.NewSelect().Column("datname").Model(&databases).Table("pg_database").Where("datistemplate = false").Scan(ctx); err != nil {
		panic(err)
	}

	return databases
}

func ConnectToDatabase(databaseUrl string) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseUrl)))
	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}

func CreateSnapshot(databaseName, snapshotName string) {
	db := ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")

	snapshotName = "lunar_snapshot_" + databaseName + "_" + snapshotName
	ctx := context.Background()
	if _, err := db.Exec("CREATE DATABASE "+snapshotName+" TEMPLATE "+databaseName, ctx); err != nil {
		panic(err)
	}
}

func RestoreSnapshot() {
	db := ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")

	ctx := context.Background()
	if _, err := db.Exec("DROP DATABASE IF EXISTS dev_box_development", ctx); err != nil {
		panic(err)
	}
	if _, err := db.Exec("CREATE DATABASE dev_box_development TEMPLATE snapshot_template1", ctx); err != nil {
		panic(err)
	}
}

func TerminateAllCurrentConnections(databaseName string) {
	db := ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")

	ctx := context.Background()
	if _, err := db.Exec("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = 'dev_box_development' AND pid <> pg_backend_pid()", ctx); err != nil {
		panic(err)
	}
}
