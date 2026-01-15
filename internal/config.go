package internal

import (
	"os"
	"path/filepath"

	"github.com/leonvogt/lunar/internal/provider"
	"gopkg.in/yaml.v3"
)

const (
	CONFIG_PATH = "lunar.yml"
)

type Config struct {
	// Provider type: "postgres" (default) or "sqlite"
	ProviderType provider.ProviderType `yaml:"provider,omitempty"`

	// PostgreSQL configuration
	DatabaseUrl         string `yaml:"database_url,omitempty"`
	DatabaseName        string `yaml:"database,omitempty"`
	MaintenanceDatabase string `yaml:"maintenance_database,omitempty"`

	// SQLite configuration
	DatabasePath      string `yaml:"database_path,omitempty"`
	SnapshotDirectory string `yaml:"snapshot_directory,omitempty"`
}

func (c *Config) GetProviderType() provider.ProviderType {
	if c.ProviderType == "" {
		return provider.ProviderTypePostgres
	}
	return c.ProviderType
}

func DefaultMaintenanceDatabases() []string {
	return []string{"postgres", "template1"}
}

func (c *Config) GetMaintenanceDatabase() string {
	if c.MaintenanceDatabase != "" {
		return c.MaintenanceDatabase
	}
	return ""
}

func (c *Config) GetDatabaseIdentifier() string {
	switch c.GetProviderType() {
	case provider.ProviderTypeSQLite:
		return c.DatabasePath
	default:
		return c.DatabaseName
	}
}

func (c *Config) GetResolvedDatabasePath() string {
	return resolvePath(c.DatabasePath)
}

func (c *Config) GetResolvedSnapshotDirectory() string {
	return resolvePath(c.SnapshotDirectory)
}

// Returns an absolute path for the given path
func resolvePath(path string) string {
	if path == "" {
		return path
	}

	// If already absolute, return as-is
	if filepath.IsAbs(path) {
		return path
	}

	// Get the directory containing the config file
	configDir, err := os.Getwd()
	if err != nil {
		return path
	}

	return filepath.Join(configDir, path)
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
