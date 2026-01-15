package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/erikgeiser/promptkit/selection"
	"github.com/erikgeiser/promptkit/textinput"
	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/provider"
	"github.com/leonvogt/lunar/internal/provider/postgres"
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
	if internal.DoesConfigExistInCurrentDir() {
		fmt.Println("There already is a lunar.yml file in this directory. Please remove it if you want to start over.")
		return
	}

	var providerType provider.ProviderType
	if providerFlag != "" {
		switch strings.ToLower(providerFlag) {
		case "postgres", "postgresql":
			providerType = provider.ProviderTypePostgres
		case "sqlite":
			providerType = provider.ProviderTypeSQLite
		default:
			fmt.Printf("Unknown provider: %s. Must be 'postgres' or 'sqlite'.\n", providerFlag)
			os.Exit(1)
		}
	} else {
		fmt.Println("Welcome to Lunar! Let's get started.")
		fmt.Println("")
		providerType = askForProviderType()
	}

	var config internal.Config
	config.ProviderType = providerType

	switch providerType {
	case provider.ProviderTypePostgres:
		initializePostgres(&config)
	case provider.ProviderTypeSQLite:
		initializeSQLite(&config)
	}

	internal.CreateConfigFile(&config, internal.CONFIG_PATH)
	fmt.Println("Initialization complete. You may now run 'lunar snapshot production' to create a snapshot of your database.")
}

func askForProviderType() provider.ProviderType {
	choices := []string{"PostgreSQL", "SQLite"}

	prompt := selection.New("What type of database do you want to snapshot?", choices)
	prompt.PageSize = 10

	choice, err := prompt.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("")

	if choice == "SQLite" {
		return provider.ProviderTypeSQLite
	}
	return provider.ProviderTypePostgres
}

func initializePostgres(config *internal.Config) {
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

	db, err := postgres.ConnectToMaintenanceDatabaseWithURL(testUrl)
	if err != nil {
		fmt.Printf("Could not connect to PostgreSQL with the URL %s. Error: %v\n", config.DatabaseUrl, err)
		fmt.Println("Hint: Make sure at least one of the following databases exists and is accessible: postgres, template1")
		os.Exit(1)
	}
	defer db.Close()

	if databaseNameFlag == "" {
		config.DatabaseName = askForDatabaseName(config.DatabaseUrl)
	} else {
		config.DatabaseName = databaseNameFlag
	}
}

func initializeSQLite(config *internal.Config) {
	if databasePathFlag != "" {
		config.DatabasePath = databasePathFlag
	} else {
		config.DatabasePath = askForDatabasePath()
	}

	// Verify the file exists
	if _, err := os.Stat(config.DatabasePath); os.IsNotExist(err) {
		fmt.Printf("Database file does not exist: %s\n", config.DatabasePath)
		os.Exit(1)
	}

	if snapshotDirectoryFlag != "" {
		config.SnapshotDirectory = snapshotDirectoryFlag
	} else {
		config.SnapshotDirectory = askForSnapshotDirectory(config.DatabasePath)
	}
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

	database, err := postgres.ConnectToMaintenanceDatabaseWithURL(databaseUrl)
	if err != nil {
		fmt.Printf("Could not connect to PostgreSQL. Error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	databaseNames, err := postgres.AllDatabasesWithConnection(database)
	if err != nil {
		fmt.Printf("Could not list databases. Error: %v\n", err)
		os.Exit(1)
	}

	filteredDatabaseNames := make([]string, 0)
	for _, name := range databaseNames {
		if name == "postgres" {
			continue
		}
		if strings.HasPrefix(name, "lunar_snapshot____") {
			continue
		}
		filteredDatabaseNames = append(filteredDatabaseNames, name)
	}

	prompt := selection.New("Please select the database you want to snapshot", filteredDatabaseNames)
	prompt.PageSize = 50

	databaseName, err := prompt.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	return databaseName
}

func askForDatabasePath() string {
	// Try to find .db files in current directory as suggestions
	currentDir, _ := os.Getwd()

	input := textinput.New("Path to SQLite database file (relative to this directory)")
	input.Placeholder = "e.g., ./myapp.db or data/database.sqlite"

	// Look for existing SQLite files in current directory and common folders
	var matches []string
	extensions := []string{"*.db", "*.sqlite", "*.sqlite3"}

	// Search in current directory
	for _, ext := range extensions {
		found, _ := filepath.Glob(filepath.Join(currentDir, ext))
		matches = append(matches, found...)
	}

	// Search in common directories
	commonDirs := []string{"storage", "data", "db", "database", "sqlite"}
	for _, dir := range commonDirs {
		dirPath := filepath.Join(currentDir, dir)
		if _, err := os.Stat(dirPath); err == nil {
			for _, ext := range extensions {
				found, _ := filepath.Glob(filepath.Join(dirPath, ext))
				matches = append(matches, found...)
			}
		}
	}

	if len(matches) > 0 {
		// Use relative path for initial value
		relPath, err := filepath.Rel(currentDir, matches[0])
		if err == nil {
			input.InitialValue = "./" + relPath
		} else {
			input.InitialValue = matches[0]
		}
	}

	dbPath, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Keep as relative path - will be resolved at runtime
	return dbPath
}

func askForSnapshotDirectory(databasePath string) string {
	// Default to .lunar_snapshots in the same directory as the database
	dbDir := filepath.Dir(databasePath)
	defaultDir := filepath.Join(dbDir, ".lunar_snapshots")

	if !strings.HasPrefix(defaultDir, "./") && !strings.HasPrefix(defaultDir, "/") {
		defaultDir = "./" + defaultDir
	}

	input := textinput.New("Directory to store snapshots (relative to this directory)")
	input.InitialValue = defaultDir
	input.Placeholder = "Directory path for snapshots"

	snapshotDir, err := input.RunPrompt()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Keep as relative path - will be resolved at runtime
	return snapshotDir
}
