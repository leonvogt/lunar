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
	internal.ListAllDatabases()

	var database string
	fmt.Print("Enter a database name: ")
	fmt.Scanln(&database)

	internal.StoreConfig(database)

	fmt.Println("Intialization complete. You may now run 'lunar snapshot production' to create a snapshot of your database.")
}
