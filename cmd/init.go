package cmd

import (
	"fmt"
	"os"

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

	// Test the connection
	db := internal.OpenDatabaseConnection(config.DatabaseUrl, false)
	defer db.Close()
	if err := internal.TestConnection(db); err != nil {
		fmt.Printf("Could not connect to PostgreSQL with the URL %s. Error: %v\n", config.DatabaseUrl, err)
		return
	}

	if databaseNameFlag == "" {
		defaultTemplateUrl := config.DatabaseUrl + "template1"
		config.DatabaseName = askForDatabaseName(defaultTemplateUrl)
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

	databaseNames := internal.AllDatabases(internal.OpenDatabaseConnection(databaseUrl, false))
	sp := selection.New("Please enter the name of the database you want to snapshot", databaseNames)
	sp.PageSize = 50

	databaseName, err := sp.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	return databaseName
}
