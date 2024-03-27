package main

import (
	"fmt"

	"github.com/leonvogt/lunar/internal"
)

func main() {
	config, err := internal.ReadConfig()
	if err != nil {
		startOnboarding()
		config, _ = internal.ReadConfig()
	}

	internal.ConnectToDatabaseAndQuery(config.Database)
}

func startOnboarding() {
	fmt.Println("Welcome to Lunar! Let's get started.")

	internal.ListAllDatabases()

	var database string
	fmt.Print("Enter database name: ")
	fmt.Scanln(&database)

	internal.StoreConfig(database)
}
