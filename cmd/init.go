package cmd

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	initCmd = &cobra.Command{
		Use:     "init",
		Aliases: []string{"initialize", "initialise", "create"},
		Short:   "Initialize Lunar for the current directory",
		Run: func(_ *cobra.Command, args []string) {
			initializeProject()
		},
	}
)

func initializeProject() {
	if internal.DoesConfigExist() {
		fmt.Println("There already is a lunar.yml file in this directory. Please remove it if you want to start over.")
		return
	}

	fmt.Println("Welcome to Lunar! Let's get started.")
	config := internal.Config{}
	config.DatabaseUrl = askForDatabaseUrl()
	config.DatabaseName = askForDatabaseName()
	internal.CreateConfigFile(&config, internal.CONFIG_PATH)

	fmt.Println("Intialization complete. You may now run 'lunar snapshot production' to create a snapshot of your database.")
}

func askForDatabaseUrl() string {
	fmt.Println("Please enter the connection URL to your PostgreSQL database.")
	fmt.Println("Example: postgres://localhost:5432/")

	var databaseUrl string
	fmt.Print("\nPostgreSQL URL: ")
	fmt.Scanln(&databaseUrl)

	return databaseUrl
}

func askForDatabaseName() string {
	fmt.Println("Please enter the name of the database you want to snapshot.")

	fmt.Println("")
	fmt.Println(internal.AllDatabases())
	fmt.Println("")

	var databaseName string
	fmt.Print("Database name: ")
	fmt.Scanln(&databaseName)

	return databaseName
}