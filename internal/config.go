package internal

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	CONFIG_PATH = "lunar.yml"
)

type Config struct {
	DatabaseUrl         string `yaml:"database_url"`
	DatabaseName        string `yaml:"database"`
	MaintenanceDatabase string `yaml:"maintenance_database,omitempty"`
}

// DefaultMaintenanceDatabases returns the list of databases to try for maintenance operations
// in order of preference
func DefaultMaintenanceDatabases() []string {
	return []string{"postgres", "template1"}
}

// GetMaintenanceDatabase returns the configured maintenance database or empty string if not set
func (c *Config) GetMaintenanceDatabase() string {
	if c.MaintenanceDatabase != "" {
		return c.MaintenanceDatabase
	}
	return ""
}

func CreateConfigFile(config *Config, path string) error {
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

func DoesConfigExist() bool {
	_, err := os.Stat(CONFIG_PATH)
	return !os.IsNotExist(err)
}
