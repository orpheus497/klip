package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}
	messages := make([]string, len(ve))
	for i, err := range ve {
		messages[i] = err.Error()
	}
	return strings.Join(messages, "; ")
}

// Validate performs comprehensive validation of the configuration
func (c *Config) Validate() error {
	var errors ValidationErrors

	// Validate settings
	if err := c.validateSettings(); err != nil {
		if ve, ok := err.(ValidationErrors); ok {
			errors = append(errors, ve...)
		} else {
			errors = append(errors, ValidationError{Field: "settings", Message: err.Error()})
		}
	}

	// Validate current profile reference
	if c.CurrentProfile != "" {
		if _, exists := c.Profiles[c.CurrentProfile]; !exists {
			errors = append(errors, ValidationError{
				Field:   "current_profile",
				Message: fmt.Sprintf("references non-existent profile '%s'", c.CurrentProfile),
			})
		}
	}

	// Validate all profiles
	for name, profile := range c.Profiles {
		if profile == nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("profiles.%s", name),
				Message: "profile is nil",
			})
			continue
		}

		if err := profile.Validate(); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("profiles.%s", name),
				Message: err.Error(),
			})
		}

		// Check SSH key path exists if specified
		if profile.SSHKeyPath != "" {
			if _, err := os.Stat(profile.SSHKeyPath); os.IsNotExist(err) {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("profiles.%s.ssh_key_path", name),
					Message: fmt.Sprintf("SSH key file does not exist: %s", profile.SSHKeyPath),
				})
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateSettings validates the settings section
func (c *Config) validateSettings() error {
	var errors ValidationErrors

	// Validate default backend
	validBackends := map[string]bool{
		"auto":      true,
		"lan":       true,
		"tailscale": true,
		"headscale": true,
		"netbird":   true,
	}
	if !validBackends[c.Settings.DefaultBackend] {
		errors = append(errors, ValidationError{
			Field:   "settings.default_backend",
			Message: fmt.Sprintf("invalid backend '%s', must be one of: auto, lan, tailscale, headscale, netbird", c.Settings.DefaultBackend),
		})
	}

	// Validate SSH timeout
	if c.Settings.SSHTimeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "settings.ssh_timeout",
			Message: "must be greater than 0",
		})
	}
	if c.Settings.SSHTimeout > 300 {
		errors = append(errors, ValidationError{
			Field:   "settings.ssh_timeout",
			Message: "must be 300 seconds or less",
		})
	}

	// Validate transfer method
	validMethods := map[string]bool{"rsync": true, "sftp": true}
	if !validMethods[c.Settings.TransferMethod] {
		errors = append(errors, ValidationError{
			Field:   "settings.transfer_method",
			Message: fmt.Sprintf("invalid method '%s', must be 'rsync' or 'sftp'", c.Settings.TransferMethod),
		})
	}

	// Validate compression level
	if c.Settings.CompressionLevel < 0 || c.Settings.CompressionLevel > 9 {
		errors = append(errors, ValidationError{
			Field:   "settings.compression_level",
			Message: "must be between 0 and 9",
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// ValidateProfile validates a single profile without checking the full config
func ValidateProfile(profile *Profile) error {
	if profile == nil {
		return fmt.Errorf("profile is nil")
	}
	return profile.Validate()
}

// SanitizeProfile sanitizes profile values and applies defaults
func SanitizeProfile(profile *Profile) {
	// Ensure SSH port is set
	if profile.SSHPort == 0 {
		profile.SSHPort = 22
	}

	// Ensure backend is set
	if profile.Backend == "" {
		profile.Backend = BackendAuto
	}

	// Ensure transfer method is set
	if profile.TransferOptions.Method == "" {
		profile.TransferOptions.Method = "rsync"
	}

	// Ensure compression level is valid
	if profile.TransferOptions.CompressionLevel < 0 {
		profile.TransferOptions.CompressionLevel = 0
	}
	if profile.TransferOptions.CompressionLevel > 9 {
		profile.TransferOptions.CompressionLevel = 9
	}

	// Trim whitespace from string fields
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Description = strings.TrimSpace(profile.Description)
	profile.RemoteUser = strings.TrimSpace(profile.RemoteUser)
	profile.RemoteHost = strings.TrimSpace(profile.RemoteHost)
	profile.SSHKeyPath = strings.TrimSpace(profile.SSHKeyPath)
}

// ValidatePort checks if port is in valid range
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return &ValidationError{
			Field:   "ssh_port",
			Message: fmt.Sprintf("port must be between 1 and 65535, got %d", port),
		}
	}
	return nil
}

// ValidateHostname checks if hostname is valid format (RFC 1123)
func ValidateHostname(hostname string) error {
	if hostname == "" {
		return &ValidationError{
			Field:   "remote_host",
			Message: "hostname cannot be empty",
		}
	}

	// Maximum length check (RFC 1123)
	if len(hostname) > 253 {
		return &ValidationError{
			Field:   "remote_host",
			Message: "hostname exceeds maximum length of 253 characters",
		}
	}

	// Check for valid characters: alphanumeric, hyphens, dots
	// Allow IP addresses and hostnames
	validHostname := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-\.]*[a-zA-Z0-9])?$`)
	if !validHostname.MatchString(hostname) {
		return &ValidationError{
			Field:   "remote_host",
			Message: "hostname contains invalid characters (only alphanumeric, hyphens, dots allowed)",
		}
	}

	// No consecutive dots
	if strings.Contains(hostname, "..") {
		return &ValidationError{
			Field:   "remote_host",
			Message: "hostname cannot contain consecutive dots",
		}
	}

	return nil
}

// ValidateUsername checks if username is valid (POSIX-style)
func ValidateUsername(username string) error {
	if username == "" {
		return &ValidationError{
			Field:   "remote_user",
			Message: "username cannot be empty",
		}
	}

	// Maximum length check (POSIX limit is 32 chars)
	if len(username) > 32 {
		return &ValidationError{
			Field:   "remote_user",
			Message: "username exceeds maximum length of 32 characters",
		}
	}

	// POSIX username rules: start with letter or underscore,
	// contain only lowercase alphanumeric, underscore, hyphen
	validUsername := regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	if !validUsername.MatchString(username) {
		return &ValidationError{
			Field:   "remote_user",
			Message: "username must start with letter or underscore, contain only lowercase alphanumeric, underscore, hyphen",
		}
	}

	return nil
}

// ValidateSSHKeyPath checks if SSH key exists and has correct permissions
func ValidateSSHKeyPath(keyPath string) error {
	if keyPath == "" {
		return nil // Empty is OK, will use default keys
	}

	// Expand tilde to home directory
	if strings.HasPrefix(keyPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return &ValidationError{
				Field:   "ssh_key_path",
				Message: "cannot determine home directory",
			}
		}
		keyPath = filepath.Join(home, keyPath[2:])
	}

	// Check existence
	info, err := os.Stat(keyPath)
	if os.IsNotExist(err) {
		return &ValidationError{
			Field:   "ssh_key_path",
			Message: fmt.Sprintf("SSH key not found: %s", keyPath),
		}
	}
	if err != nil {
		return &ValidationError{
			Field:   "ssh_key_path",
			Message: fmt.Sprintf("cannot access SSH key: %v", err),
		}
	}

	// Check it's a regular file
	if !info.Mode().IsRegular() {
		return &ValidationError{
			Field:   "ssh_key_path",
			Message: "SSH key path is not a regular file",
		}
	}

	// Check permissions (should be 0600 or stricter on Unix systems)
	// On Windows, permission checks are different, so we skip this
	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		return &ValidationError{
			Field:   "ssh_key_path",
			Message: fmt.Sprintf("SSH key has overly permissive permissions %#o (should be 0600)", mode),
		}
	}

	// Try to parse key to ensure it's valid
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return &ValidationError{
			Field:   "ssh_key_path",
			Message: fmt.Sprintf("cannot read SSH key: %v", err),
		}
	}

	// Attempt to parse the private key
	_, err = ssh.ParsePrivateKey(keyData)
	if err != nil {
		// If it's encrypted, that's OK - we'll prompt for passphrase at connection time
		if !strings.Contains(err.Error(), "encrypted") && !strings.Contains(err.Error(), "passphrase") {
			return &ValidationError{
				Field:   "ssh_key_path",
				Message: fmt.Sprintf("invalid SSH key format: %v", err),
			}
		}
		// Encrypted key is acceptable
	}

	return nil
}

// ValidateBandwidthLimit checks bandwidth limit is non-negative
func ValidateBandwidthLimit(limit int) error {
	if limit < 0 {
		return &ValidationError{
			Field:   "bandwidth_limit",
			Message: "bandwidth limit cannot be negative",
		}
	}
	return nil
}

// ValidateCompressionLevel checks compression level is in range 0-9
func ValidateCompressionLevel(level int) error {
	if level < 0 || level > 9 {
		return &ValidationError{
			Field:   "compression_level",
			Message: fmt.Sprintf("compression level must be between 0 and 9, got %d", level),
		}
	}
	return nil
}
