package config

import (
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v4"
)

// SavedQuery is a named SQL snippet for the config file.
// It maps to model.SavedQuery when loading into the app.
type SavedQuery struct {
	Name  string `yaml:"name"`
	Query string `yaml:"query"`
}

// Connection is a database connection configuration.
type Connection struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"` // "sqlite" or "postgres"
	DSN  string `yaml:"dsn"`  // postgres://... or /path/to/db.sqlite
}

// Config holds stoat configuration loaded from ~/.stoat/config.yaml.
type Config struct {
	Theme        string       `yaml:"theme"`
	SavedQueries []SavedQuery `yaml:"saved_queries"`
	Connections  []Connection `yaml:"connections"`
}

// ConfigDir returns the stoat config directory (e.g. ~/.stoat).
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".stoat"), nil
}

// ConfigPath returns the path to the config file (~/.stoat/config.yaml).
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// EnsureConfigDir creates the config directory if it does not exist.
func EnsureConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o700)
}

// DefaultConfig returns config with standard values (e.g. for first-run).
func DefaultConfig() Config {
	return Config{
		Theme:        "default",
		SavedQueries: []SavedQuery{},
	}
}

// WriteConfig writes cfg to the config file, creating the file if needed.
// A comment header with the config path is written at the top of the file.
func WriteConfig(cfg Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	header := "# " + path + "\n\n"
	return os.WriteFile(path, append([]byte(header), data...), 0o600)
}

// LoadConfig loads the config from the config file.
// If the file does not exist, it is created with default values and those are returned.
func LoadConfig() (Config, error) {
	err := EnsureConfigDir()
	if err != nil {
		return Config{}, err
	}

	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{}
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = DefaultConfig()
			if writeErr := WriteConfig(cfg); writeErr != nil {
				return Config{}, writeErr
			}
			return cfg, nil
		}
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
