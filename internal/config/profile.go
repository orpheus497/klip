package config

import (
	"fmt"
	"strings"
)

// BackendType represents the VPN backend type
type BackendType string

const (
	// BackendAuto automatically detects the best available backend
	BackendAuto BackendType = "auto"

	// BackendLAN uses direct IP/hostname connection
	BackendLAN BackendType = "lan"

	// BackendTailscale uses Tailscale VPN
	BackendTailscale BackendType = "tailscale"

	// BackendHeadscale uses Headscale (self-hosted Tailscale)
	BackendHeadscale BackendType = "headscale"

	// BackendNetBird uses NetBird VPN
	BackendNetBird BackendType = "netbird"
)

// Profile represents a connection profile for a remote machine
type Profile struct {
	// Name is a descriptive name for this profile
	Name string `yaml:"name"`

	// Description provides details about this profile
	Description string `yaml:"description,omitempty"`

	// Backend specifies which VPN backend to use
	Backend BackendType `yaml:"backend"`

	// RemoteUser is the SSH username on the remote machine
	RemoteUser string `yaml:"remote_user"`

	// RemoteHost is the hostname or IP address of the remote machine
	RemoteHost string `yaml:"remote_host"`

	// SSHPort is the SSH port (default: 22)
	SSHPort int `yaml:"ssh_port,omitempty"`

	// SSHKeyPath is the path to the SSH private key
	SSHKeyPath string `yaml:"ssh_key_path,omitempty"`

	// UsePassword enables password authentication instead of key-based
	UsePassword bool `yaml:"use_password,omitempty"`

	// TransferOptions contains transfer-specific settings
	TransferOptions TransferOptions `yaml:"transfer_options,omitempty"`
}

// TransferOptions contains options for file transfers
type TransferOptions struct {
	// Method specifies the transfer method (rsync, sftp)
	Method string `yaml:"method,omitempty"`

	// CompressionLevel specifies the compression level (0-9)
	CompressionLevel int `yaml:"compression_level,omitempty"`

	// ExcludePatterns contains rsync exclude patterns
	ExcludePatterns []string `yaml:"exclude_patterns,omitempty"`

	// BandwidthLimit limits transfer speed in KB/s (0=unlimited)
	BandwidthLimit int `yaml:"bandwidth_limit,omitempty"`

	// PreservePermissions preserves file permissions during transfer
	PreservePermissions bool `yaml:"preserve_permissions,omitempty"`

	// DeleteAfterTransfer deletes source files after successful transfer
	DeleteAfterTransfer bool `yaml:"delete_after_transfer,omitempty"`
}

// NewProfile creates a new profile with defaults
func NewProfile(name, user, host string) *Profile {
	return &Profile{
		Name:       name,
		Backend:    BackendAuto,
		RemoteUser: user,
		RemoteHost: host,
		SSHPort:    22,
		TransferOptions: TransferOptions{
			Method:              "rsync",
			CompressionLevel:    6,
			PreservePermissions: true,
			DeleteAfterTransfer: false,
		},
	}
}

// Validate checks if the profile configuration is valid
func (p *Profile) Validate() error {
	if p.RemoteUser == "" {
		return fmt.Errorf("remote_user is required")
	}

	if p.RemoteHost == "" {
		return fmt.Errorf("remote_host is required")
	}

	if p.SSHPort <= 0 || p.SSHPort > 65535 {
		return fmt.Errorf("ssh_port must be between 1 and 65535")
	}

	validBackends := map[BackendType]bool{
		BackendAuto:      true,
		BackendLAN:       true,
		BackendTailscale: true,
		BackendHeadscale: true,
		BackendNetBird:   true,
	}

	if !validBackends[p.Backend] {
		return fmt.Errorf("invalid backend '%s', must be one of: auto, lan, tailscale, headscale, netbird", p.Backend)
	}

	validMethods := map[string]bool{"rsync": true, "sftp": true}
	if p.TransferOptions.Method != "" && !validMethods[p.TransferOptions.Method] {
		return fmt.Errorf("invalid transfer method '%s', must be 'rsync' or 'sftp'", p.TransferOptions.Method)
	}

	if p.TransferOptions.CompressionLevel < 0 || p.TransferOptions.CompressionLevel > 9 {
		return fmt.Errorf("compression_level must be between 0 and 9")
	}

	return nil
}

// SSHAddress returns the SSH connection address
func (p *Profile) SSHAddress() string {
	if p.SSHPort != 22 {
		return fmt.Sprintf("%s@%s:%d", p.RemoteUser, p.RemoteHost, p.SSHPort)
	}
	return fmt.Sprintf("%s@%s", p.RemoteUser, p.RemoteHost)
}

// String returns a string representation of the profile
func (p *Profile) String() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Profile: %s", p.Name))
	if p.Description != "" {
		parts = append(parts, fmt.Sprintf("  Description: %s", p.Description))
	}
	parts = append(parts, fmt.Sprintf("  Backend: %s", p.Backend))
	parts = append(parts, fmt.Sprintf("  Remote: %s", p.SSHAddress()))
	if p.SSHKeyPath != "" {
		parts = append(parts, fmt.Sprintf("  SSH Key: %s", p.SSHKeyPath))
	}
	return strings.Join(parts, "\n")
}

// Clone creates a deep copy of the profile
func (p *Profile) Clone() *Profile {
	clone := *p
	clone.TransferOptions.ExcludePatterns = make([]string, len(p.TransferOptions.ExcludePatterns))
	copy(clone.TransferOptions.ExcludePatterns, p.TransferOptions.ExcludePatterns)
	return &clone
}
