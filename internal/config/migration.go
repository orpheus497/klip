package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MigrateLegacyConfig attempts to migrate configuration from LINK bash scripts
func MigrateLegacyConfig() (*Config, error) {
	legacyPath := LegacyConfigPath()
	if legacyPath == "" {
		return nil, fmt.Errorf("unable to determine home directory")
	}

	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("legacy config file not found: %s", legacyPath)
	}

	// Parse the legacy Bash config
	variables, err := parseBashConfig(legacyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	cfg := NewConfig()

	// Migrate LAN profile
	if lanUser, lanOk := variables["LAN_REMOTE_USER"]; lanOk {
		if lanHost, hostOk := variables["LAN_REMOTE_HOST"]; hostOk {
			// Only migrate if not placeholder values
			if !isPlaceholderValue(lanUser) && !isPlaceholderValue(lanHost) {
				lanProfile := NewProfile("lan", lanUser, lanHost)
				lanProfile.Description = "Migrated from LINK LAN configuration"
				lanProfile.Backend = BackendLAN
				cfg.AddProfile("lan", lanProfile)
			}
		}
	}

	// Migrate Tailscale profile
	if tsUser, tsOk := variables["TS_REMOTE_USER"]; tsOk {
		if tsHost, hostOk := variables["TS_REMOTE_HOST"]; hostOk {
			// Only migrate if not placeholder values
			if !isPlaceholderValue(tsUser) && !isPlaceholderValue(tsHost) {
				tsProfile := NewProfile("tailscale", tsUser, tsHost)
				tsProfile.Description = "Migrated from LINK Tailscale configuration"
				tsProfile.Backend = BackendTailscale
				cfg.AddProfile("tailscale", tsProfile)
			}
		}
	}

	// Check if any profiles were migrated
	if len(cfg.Profiles) == 0 {
		return nil, fmt.Errorf("no valid profiles found in legacy config (only placeholder values detected)")
	}

	return cfg, nil
}

// parseBashConfig parses a Bash configuration file and extracts variable assignments
func parseBashConfig(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	variables := make(map[string]string)
	scanner := bufio.NewScanner(file)

	// Regex to match variable assignments: VAR="value" or VAR='value' or VAR=value
	assignmentRegex := regexp.MustCompile(`^\s*([A-Z_][A-Z0-9_]*)\s*=\s*["']?([^"']*)["']?\s*$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Match variable assignment
		matches := assignmentRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			varName := matches[1]
			varValue := strings.Trim(matches[2], `"'`)
			variables[varName] = varValue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return variables, nil
}

// isPlaceholderValue checks if a value is a placeholder (not configured by user)
func isPlaceholderValue(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))

	placeholders := []string{
		"",
		"your_lan_user",
		"your_lan_hostname_or_ip",
		"your_tailscale_user",
		"your_tailscale_hostname",
		"user",
		"hostname",
		"localhost",
		"example.com",
		"changeme",
		"replace_me",
	}

	for _, placeholder := range placeholders {
		if value == placeholder {
			return true
		}
	}

	return false
}

// BackupLegacyConfig creates a backup of the legacy configuration
func BackupLegacyConfig() (string, error) {
	legacyPath := LegacyConfigPath()
	if legacyPath == "" {
		return "", fmt.Errorf("unable to determine home directory")
	}

	if _, err := os.Stat(legacyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("legacy config file not found: %s", legacyPath)
	}

	// Create backup file path
	backupPath := legacyPath + ".backup"

	// Read original file
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read legacy config: %w", err)
	}

	// Write backup
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// CleanupLegacyConfig removes the legacy LINK configuration directory
func CleanupLegacyConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	legacyDir := filepath.Join(homeDir, LegacyConfigDir)
	if _, err := os.Stat(legacyDir); os.IsNotExist(err) {
		return nil // Already cleaned up
	}

	// Create backup before removal
	backupPath, err := BackupLegacyConfig()
	if err != nil {
		return fmt.Errorf("failed to create backup before cleanup: %w", err)
	}

	// Remove legacy directory
	if err := os.RemoveAll(legacyDir); err != nil {
		return fmt.Errorf("failed to remove legacy config directory: %w", err)
	}

	fmt.Printf("Legacy LINK configuration removed (backup saved to: %s)\n", backupPath)
	return nil
}

// MigrationStatus checks if migration is needed or has been completed
type MigrationStatus struct {
	LegacyConfigExists bool
	ModernConfigExists bool
	NeedsMigration     bool
	CanMigrate         bool
}

// CheckMigrationStatus determines if migration is needed
func CheckMigrationStatus() MigrationStatus {
	status := MigrationStatus{}

	// Check if legacy config exists
	legacyPath := LegacyConfigPath()
	if legacyPath != "" {
		if _, err := os.Stat(legacyPath); err == nil {
			status.LegacyConfigExists = true
		}
	}

	// Check if modern config exists
	modernPath, err := ConfigPath()
	if err == nil {
		if _, err := os.Stat(modernPath); err == nil {
			status.ModernConfigExists = true
		}
	}

	// Determine migration status
	status.CanMigrate = status.LegacyConfigExists
	status.NeedsMigration = status.LegacyConfigExists && !status.ModernConfigExists

	return status
}
