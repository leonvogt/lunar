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

	configPath string `yaml:"-"`
	configDir  string `yaml:"-"`
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
	return resolvePath(c.DatabasePath, c.configDir)
}

func (c *Config) GetResolvedSnapshotDirectory() string {
	return resolvePath(c.SnapshotDirectory, c.configDir)
}

// Returns an absolute path for the given path
func resolvePath(path string, baseDir string) string {
	if path == "" {
		return path
	}

	// If already absolute, return as-is
	if filepath.IsAbs(path) {
		return path
	}

	if baseDir == "" {
		configDir, err := os.Getwd()
		if err != nil {
			return path
		}
		baseDir = configDir
	}

	return filepath.Join(baseDir, path)
}

func CreateConfigFile(config *Config, path string) error {
	if config != nil {
		setConfigBasePath(config, path)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return yaml.NewEncoder(file).Encode(config)
}

func ReadConfig() (*Config, error) {
	config := &Config{}

	configPath, err := findConfigPath()
	if err != nil {
		return nil, err
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	setConfigBasePath(config, configPath)

	d := yaml.NewDecoder(file)
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func setConfigBasePath(config *Config, path string) {
	if config == nil {
		return
	}

	configPath := path
	if absPath, err := filepath.Abs(path); err == nil {
		configPath = absPath
	}

	config.configPath = configPath
	config.configDir = filepath.Dir(configPath)
}

func DoesConfigExist() bool {
	_, err := findConfigPath()
	return err == nil
}

func DoesConfigExistInCurrentDir() bool {
	_, err := os.Stat(CONFIG_PATH)
	return !os.IsNotExist(err)
}

func findConfigPath() (string, error) {
	startDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	currentDir := startDir
	for {
		candidate := filepath.Join(currentDir, CONFIG_PATH)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", os.ErrNotExist
}
