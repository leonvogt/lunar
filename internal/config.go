package internal

import (
	"os"

	"gopkg.in/yaml.v3"
)

const (
	CONFIG_PATH = "lunar.yml"
)

type Config struct {
	DatabaseUrl  string `yaml:"database_url"`
	DatabaseName string `yaml:"database"`
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
