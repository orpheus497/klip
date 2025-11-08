package config

import (
	"fmt"
	"os"
	"strings"
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
