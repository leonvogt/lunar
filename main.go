package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"gopkg.in/yaml.v3"
)

const (
	CONFIG_PATH = "lunar.yml"
)

type User struct {
	ID int64 `bun:",pk,autoincrement"`
}

type Config struct {
	Database string `yaml:"database"`
}

func main() {
	config, err := ReadConfig()
	if err != nil {
		startOnboarding()
		config, _ = ReadConfig()
	}

	connectToDatabaseAndQuery(config.Database)
}

func listAllDatabases() {
	db := connectToDatabase("postgres://postgres:@localhost:5432/template1?sslmode=disable")

	ctx := context.Background()
	databases := make([]string, 0)
	if err := db.NewSelect().Column("datname").Model(&databases).Table("pg_database").Where("datistemplate = false").Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("all databases: %v\n\n", databases)
}

func connectToDatabase(databaseUrl string) *bun.DB {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseUrl)))
	db := bun.NewDB(sqldb, pgdialect.New())
	return db
}

func WriteConfig(config *Config, path string) error {
	// Create a new file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(config)
}

func ReadConfig() (*Config, error) {
	config := &Config{}

	file, err := os.Open(CONFIG_PATH)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func storeConfig(database string) {
	config := &Config{}
	config.Database = database
	configPath := CONFIG_PATH
	if err := WriteConfig(config, configPath); err != nil {
		panic(err)
	}
}

func startOnboarding() {
	fmt.Println("Welcome to Lunar! Let's get started.")

	listAllDatabases()

	var database string
	fmt.Print("Enter database name: ")
	fmt.Scanln(&database)

	storeConfig(database)
}

func connectToDatabaseAndQuery(database string) {
	db := connectToDatabase(fmt.Sprintf("postgres://postgres:@localhost:5432/%s?sslmode=disable", database))

	// Sample query
	users := make([]User, 0)
	ctx := context.Background()
	if err := db.NewSelect().Model(&users).OrderExpr("id ASC").Scan(ctx); err != nil {
		panic(err)
	}
	fmt.Printf("all users: %v\n\n", users)
}
