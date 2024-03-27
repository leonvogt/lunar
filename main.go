package main

import (
	"fmt"
	"os"

	"github.com/leonvogt/lunar/internal"

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

	internal.ConnectToDatabaseAndQuery(config.Database)
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

	internal.ListAllDatabases()

	var database string
	fmt.Print("Enter database name: ")
	fmt.Scanln(&database)

	storeConfig(database)
}
