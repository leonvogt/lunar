package internal

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type User struct {
	ID int64 `bun:",pk,autoincrement"`
}

func ListAllDatabases() {
	db := ConnectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")

	ctx := context.Background()
	databases := make([]string, 0)
	if err := db.NewSelect().Column("datname").Model(&databases).Table("pg_database").Where("datistemplate = false").Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("all databases: %v\n\n", databases)
}

func ConnectToDatabase(databaseUrl string) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseUrl)))
	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}

func ConnectToDatabaseAndQuery(database string) {
	db := ConnectToDatabase(fmt.Sprintf("postgres://postgres:@localhost:5432/%s?sslmode=disable", database))

	// Sample query
	users := make([]User, 0)
	ctx := context.Background()
	if err := db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("all users: %v\n\n", users)
}
