package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/leonvogt/lunar/internal"
	"github.com/leonvogt/lunar/internal/provider/postgres"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testContainer *testpostgres.PostgresContainer
	testConfig    *internal.Config
)

func SetupTestContainer(t *testing.T) *internal.Config {
	if testContainer != nil {
		return testConfig
	}

	ctx := context.Background()

	// Create PostgreSQL container
	container, err := testpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		testpostgres.WithDatabase("lunar_test"),
		testpostgres.WithUsername("testuser"),
		testpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	testContainer = container

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	// Create test config
	databaseURL := fmt.Sprintf("postgres://testuser:testpass@%s:%s/", host, port.Port())
	testConfig = &internal.Config{
		DatabaseUrl:  databaseURL,
		DatabaseName: "lunar_test",
	}

	err = internal.CreateConfigFile(testConfig, "lunar.yml")
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return testConfig
}

func TeardownTestContainer(t *testing.T) {
	if testContainer == nil {
		return
	}

	ctx := context.Background()
	if err := testContainer.Terminate(ctx); err != nil {
		t.Logf("Failed to terminate PostgreSQL container: %v", err)
	}

	os.Remove("lunar.yml")

	testContainer = nil
	testConfig = nil
}

func SetupTestDatabase(t *testing.T) {
	config := SetupTestContainer(t)

	os.Setenv("TEST_DATABASE_URL", config.DatabaseUrl)
	os.Setenv("TEST_DATABASE_NAME", config.DatabaseName)

	database, err := postgres.ConnectToMaintenanceDatabaseWithURL(config.DatabaseUrl)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer database.Close()

	// Connect to the specific test database
	testDB, err := sql.Open("postgres", config.DatabaseUrl+"lunar_test?sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer testDB.Close()

	CreateUsersTable("lunar_test", testDB)
	InsertUsers("lunar_test", testDB)
}

func CreateUsersTable(databaseName string, db *sql.DB) {
	_, err := db.Exec("CREATE TABLE users (id serial PRIMARY KEY, firstname VARCHAR(50), lastname VARCHAR(50), email VARCHAR(100))")
	if err != nil {
		fmt.Printf("Error creating users table: %v\n", err)
	}
}

func InsertUser(databaseName string, firstname, lastname, email string, db *sql.DB) {
	_, err := db.Exec("INSERT INTO users (firstname, lastname, email) VALUES ($1, $2, $3)", firstname, lastname, email)
	if err != nil {
		fmt.Printf("Error inserting user: %v\n", err)
	}
}

func InsertUsers(databaseName string, db *sql.DB) {
	InsertUser(databaseName, "John", "Doe", "john.doe@example.com", db)
	InsertUser(databaseName, "Jane", "Smith", "jane.smith@example.com", db)
	InsertUser(databaseName, "Michael", "Johnson", "michael.johnson@example.com", db)
	InsertUser(databaseName, "Emily", "Brown", "emily.brown@example.com", db)
	InsertUser(databaseName, "Christopher", "Wilson", "christopher.wilson@example.com", db)
	InsertUser(databaseName, "Jessica", "Martinez", "jessica.martinez@example.com", db)
	InsertUser(databaseName, "Daniel", "Anderson", "daniel.anderson@example.com", db)
	InsertUser(databaseName, "Sarah", "Taylor", "sarah.taylor@example.com", db)
	InsertUser(databaseName, "David", "Thomas", "david.thomas@example.com", db)
	InsertUser(databaseName, "Jennifer", "Hernandez", "jennifer.hernandez@example.com", db)
}

type TestDirectoryManager struct {
	originalDir   string
	hasConfigFile bool
}

func SetupTestDirectory(t *testing.T) *TestDirectoryManager {
	originalDir, _ := os.Getwd()
	os.Chdir("..")

	// Copy the test config file to the current directory
	exec.Command("cp", "tests/lunar.yml", "lunar.yml").Run()

	return &TestDirectoryManager{
		originalDir:   originalDir,
		hasConfigFile: true,
	}
}

func (dm *TestDirectoryManager) Cleanup() {
	if dm.hasConfigFile {
		os.Remove("lunar.yml")
	}
	os.Chdir(dm.originalDir)
}

// Execute a lunar command and returns output and error
func RunLunarCommand(command string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", "go run main.go "+command)
	return cmd.CombinedOutput()
}

func CreateTestSnapshot(t *testing.T, snapshotName string) {
	output, err := RunLunarCommand("snapshot " + snapshotName)
	if err != nil {
		t.Fatalf("Failed to create snapshot: %v\nOutput: %s", err, string(output))
	}
}

func SnapshotDatabaseName(snapshotName string) string {
	return "lunar_snapshot____lunar_test____" + snapshotName
}

func CleanupSnapshot(snapshotName string) {
	config, err := internal.ReadConfig()
	if err != nil {
		return
	}

	db, err := postgres.ConnectToMaintenanceDatabaseWithURL(config.DatabaseUrl)
	if err != nil {
		return
	}
	defer db.Close()

	// Drop the snapshot and its copy
	db.Exec("DROP DATABASE IF EXISTS " + SnapshotDatabaseName(snapshotName))
	db.Exec("DROP DATABASE IF EXISTS " + SnapshotDatabaseName(snapshotName) + "_copy")
}

// DoesDatabaseExist checks if a database exists (test helper)
func DoesDatabaseExist(databaseName string) (bool, error) {
	config, err := internal.ReadConfig()
	if err != nil {
		return false, err
	}

	db, err := postgres.ConnectToMaintenanceDatabaseWithURL(config.DatabaseUrl)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check database existence: %v", err)
	}
	return exists, nil
}

// ConnectToTestDatabase connects to the test database (test helper)
func ConnectToTestDatabase(databaseName string) (*sql.DB, error) {
	config, err := internal.ReadConfig()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", config.DatabaseUrl+databaseName+"?sslmode=disable")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func WithTestDirectory(t *testing.T, testFunc func()) {
	dm := SetupTestDirectory(t)
	defer dm.Cleanup()
	testFunc()
}
