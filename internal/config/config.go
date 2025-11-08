// Package config provides configuration management for klip
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

const (
	// AppName is the application name used for config paths
	AppName = "klip"

	// ConfigFileName is the name of the configuration file
	ConfigFileName = "config.yaml"

	// LegacyConfigDir is the old LINK config directory for migration
	LegacyConfigDir = ".LINK"
)

// Config represents the application configuration
type Config struct {
	// CurrentProfile is the name of the currently active profile
	CurrentProfile string `yaml:"current_profile,omitempty"`

	// Profiles contains all connection profiles
	Profiles map[string]*Profile `yaml:"profiles"`

	// Settings contains global application settings
	Settings Settings `yaml:"settings"`

	// configPath stores the path where config was loaded from
	configPath string
}

// Settings contains global application settings
type Settings struct {
	// Verbose enables verbose logging output
	Verbose bool `yaml:"verbose"`

	// DefaultBackend specifies the preferred VPN backend (auto, lan, tailscale, headscale, netbird)
	DefaultBackend string `yaml:"default_backend"`

	// SSHTimeout is the SSH connection timeout in seconds
	SSHTimeout int `yaml:"ssh_timeout"`

	// TransferMethod specifies the preferred transfer method (rsync, sftp)
	TransferMethod string `yaml:"transfer_method"`

	// CompressionLevel specifies the rsync compression level (0-9, 0=disabled)
	CompressionLevel int `yaml:"compression_level"`

	// ShowProgress enables progress bars for transfers
	ShowProgress bool `yaml:"show_progress"`
}

// DefaultSettings returns settings with sensible defaults
func DefaultSettings() Settings {
	return Settings{
		Verbose:          false,
		DefaultBackend:   "auto",
		SSHTimeout:       30,
		TransferMethod:   "rsync",
		CompressionLevel: 6,
		ShowProgress:     true,
	}
}

// NewConfig creates a new configuration with defaults
func NewConfig() *Config {
	return &Config{
		Profiles: make(map[string]*Profile),
		Settings: DefaultSettings(),
	}
}

// ConfigPath returns the path to the configuration file
func ConfigPath() (string, error) {
	configDir := filepath.Join(xdg.ConfigHome, AppName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return filepath.Join(configDir, ConfigFileName), nil
}

// LegacyConfigPath returns the path to the old LINK configuration
func LegacyConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, LegacyConfigDir, "config.sh")
}

// Load reads the configuration from disk
func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// If config doesn't exist, check for legacy config to migrate
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if legacyPath := LegacyConfigPath(); legacyPath != "" {
			if _, err := os.Stat(legacyPath); err == nil {
				// Legacy config exists, attempt migration
				cfg, migrateErr := MigrateLegacyConfig()
				if migrateErr == nil {
					// Save migrated config
					if saveErr := cfg.Save(); saveErr == nil {
						return cfg, nil
					}
				}
			}
		}
		// No legacy config or migration failed, return new config
		cfg := NewConfig()
		cfg.configPath = configPath
		return cfg, nil
	}

	// Load existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := NewConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.configPath = configPath
	return cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	if c.configPath == "" {
		path, err := ConfigPath()
		if err != nil {
			return err
		}
		c.configPath = path
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(c.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProfile retrieves a profile by name
func (c *Config) GetProfile(name string) (*Profile, error) {
	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile '%s' not found", name)
	}
	return profile, nil
}

// GetCurrentProfile returns the currently active profile
func (c *Config) GetCurrentProfile() (*Profile, error) {
	if c.CurrentProfile == "" {
		return nil, fmt.Errorf("no current profile set")
	}
	return c.GetProfile(c.CurrentProfile)
}

// AddProfile adds or updates a profile
func (c *Config) AddProfile(name string, profile *Profile) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}

	c.Profiles[name] = profile

	// If this is the first profile, make it current
	if c.CurrentProfile == "" {
		c.CurrentProfile = name
	}

	return nil
}

// DeleteProfile removes a profile
func (c *Config) DeleteProfile(name string) error {
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}

	delete(c.Profiles, name)

	// If we deleted the current profile, clear it
	if c.CurrentProfile == name {
		c.CurrentProfile = ""
		// Set to first available profile if any exist
		for profileName := range c.Profiles {
			c.CurrentProfile = profileName
			break
		}
	}

	return nil
}

// SetCurrentProfile sets the active profile
func (c *Config) SetCurrentProfile(name string) error {
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("profile '%s' not found", name)
	}
	c.CurrentProfile = name
	return nil
}

// ListProfiles returns all profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}
