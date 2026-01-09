package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/erikgeiser/promptkit/selection"
	"github.com/erikgeiser/promptkit/textinput"
	"github.com/leonvogt/lunar/internal"
	"github.com/spf13/cobra"
)

var (
	initCmd = &cobra.Command{
		Use:     "init",
		Aliases: []string{"initialize", "initialise"},
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
	fmt.Println("")
	config := internal.Config{}

	if databaseUrlFlag == "" {
		config.DatabaseUrl = askForDatabaseUrl()
	} else {
		config.DatabaseUrl = databaseUrlFlag
	}

	// Test the connection by trying multiple maintenance databases
	testUrl := config.DatabaseUrl
	if !strings.HasSuffix(testUrl, "/") {
		testUrl += "/"
	}

	db, err := internal.ConnectToMaintenanceDatabaseWithUrl(testUrl)
	if err != nil {
		fmt.Printf("Could not connect to PostgreSQL with the URL %s. Error: %v\n", config.DatabaseUrl, err)
		fmt.Println("Hint: Make sure at least one of the following databases exists and is accessible: postgres, template1")
		return
	}
	defer db.Close()

	if databaseNameFlag == "" {
		config.DatabaseName = askForDatabaseName(config.DatabaseUrl)
	} else {
		config.DatabaseName = databaseNameFlag
	}
	internal.CreateConfigFile(&config, internal.CONFIG_PATH)

	fmt.Println("Intialization complete. You may now run 'lunar snapshot production' to create a snapshot of your database.")
}

func askForDatabaseUrl() string {
	input := textinput.New("PostgreSQL URL")
	input.InitialValue = "postgres://localhost:5432/"
	input.Placeholder = "PostgreSQL URL cannot be empty"

	databaseUrl, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	return databaseUrl
}

func askForDatabaseName(databaseUrl string) string {
	fmt.Println("")

	database, err := internal.ConnectToMaintenanceDatabaseWithUrl(databaseUrl)
	if err != nil {
		fmt.Printf("Could not connect to PostgreSQL. Error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	databaseNames, err := internal.AllDatabases(database)
	if err != nil {
		fmt.Printf("Could not list databases. Error: %v\n", err)
		os.Exit(1)
	}

	prompt := selection.New("Please enter the name of the database you want to snapshot", databaseNames)
	prompt.PageSize = 50

	databaseName, err := prompt.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	return databaseName
}
