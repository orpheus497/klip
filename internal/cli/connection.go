// Package cli - Common CLI utilities and connection helpers
// Copyright (c) 2025 orpheus497
package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/orpheus497/klip/internal/backend"
	"github.com/orpheus497/klip/internal/config"
	"github.com/orpheus497/klip/internal/logger"
	"github.com/orpheus497/klip/internal/ssh"
	"github.com/orpheus497/klip/internal/ui"
)

// ConnectionConfig holds configuration for establishing connections
type ConnectionConfig struct {
	ProfileName string
	BackendName string
	Timeout     int
	Verbose     bool
}

// ConnectionHelper assists with connection setup and management
// This eliminates code duplication across klip, klipc, and klipr commands
type ConnectionHelper struct {
	Config       *config.Config
	Profile      *config.Profile
	Backend      backend.Backend
	Log          *logger.Logger
	ResolvedHost string // The resolved hostname/IP after backend resolution
}

// NewConnectionHelper creates a connection helper with profile selection
// This centralizes the connection setup logic used by all three commands
func NewConnectionHelper(cfg ConnectionConfig) (*ConnectionHelper, error) {
	// Initialize logger
	log := logger.New(cfg.Verbose)

	// Load configuration
	appConfig, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Determine and select profile
	profile, err := selectProfile(appConfig, cfg.ProfileName)
	if err != nil {
		return nil, fmt.Errorf("failed to select profile: %w", err)
	}

	// Override backend if specified via command line
	if cfg.BackendName != "" {
		profile = profile.Clone()
		profile.Backend = config.BackendType(cfg.BackendName)
	}

	// Detect and select appropriate backend
	registry := backend.NewRegistry()
	detector := backend.NewDetector(registry)
	selectedBackend, err := detector.SelectBackend(context.Background(), string(profile.Backend))
	if err != nil {
		return nil, fmt.Errorf("failed to detect backend: %w", err)
	}

	log.Debug("Backend selected", "backend", selectedBackend.Name(), "profile", profile.Name)

	return &ConnectionHelper{
		Config:  appConfig,
		Profile: profile,
		Backend: selectedBackend,
		Log:     log,
	}, nil
}

// CreateSSHClient creates and connects an SSH client with proper error handling
// Returns a connected SSH client ready for use
func (h *ConnectionHelper) CreateSSHClient(ctx context.Context, timeout int) (*ssh.Client, error) {
	// Create context with timeout if specified
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Resolve hostname via backend
	hostname, err := h.resolveHostname(ctx)
	if err != nil {
		return nil, err
	}

	// Store the resolved hostname for later use (e.g., rsync transfers)
	h.ResolvedHost = hostname

	h.Log.Debug("Resolved hostname", "backend", h.Backend.Name(), "hostname", hostname)

	// Create SSH configuration
	sshConfig := &ssh.Config{
		Host:        hostname,
		Port:        h.Profile.SSHPort,
		User:        h.Profile.RemoteUser,
		KeyPath:     h.Profile.SSHKeyPath,
		UsePassword: h.Profile.UsePassword,
		Timeout:     time.Duration(timeout) * time.Second,
	}

	// Create SSH client
	client, err := ssh.NewClient(sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}

	// Connect to remote host
	h.Log.Info("Connecting to remote host",
		"user", sshConfig.User,
		"host", hostname,
		"port", sshConfig.Port,
		"backend", h.Backend.Name())

	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}

	h.Log.Info("Connected successfully", "host", hostname)

	return client, nil
}

// resolveHostname resolves the hostname via the selected backend
// For VPN backends (tailscale, headscale, netbird), this queries the VPN network
// to resolve the hostname to an internal IP. For LAN backend, the hostname is
// used directly and DNS resolution happens at connection time.
func (h *ConnectionHelper) resolveHostname(ctx context.Context) (string, error) {
	// Use the actual backend name (which may be auto-detected)
	// not the profile setting (which could be "auto")
	backendName := h.Backend.Name()

	// For LAN backend, use hostname directly (DNS resolution will happen at connection time)
	if backendName == "lan" {
		return h.Profile.RemoteHost, nil
	}

	// For VPN backends (tailscale, headscale, netbird), resolve hostname to IP via backend
	// This ensures we connect through the VPN network rather than attempting direct DNS resolution
	resolvedHost, err := h.Backend.GetPeerIP(ctx, h.Profile.RemoteHost)
	if err != nil {
		// Return the error for VPN backends since hostname resolution is critical
		// for proper routing through the VPN network
		return "", fmt.Errorf("failed to resolve hostname via %s: %w (hint: ensure the host is reachable via %s)", backendName, err, backendName)
	}

	return resolvedHost, nil
}

// GetResolvedHost returns the resolved hostname without creating a connection
// Useful for validation and dry-run operations
func (h *ConnectionHelper) GetResolvedHost(ctx context.Context) (string, error) {
	return h.resolveHostname(ctx)
}

// ValidateConnection validates the connection configuration without actually connecting
// Returns detailed validation errors if any issues are found
func (h *ConnectionHelper) ValidateConnection(ctx context.Context) error {
	// Validate profile configuration
	if err := h.Profile.Validate(); err != nil {
		return fmt.Errorf("profile validation failed: %w", err)
	}

	// Check backend availability
	if !h.Backend.IsAvailable(ctx) {
		return fmt.Errorf("backend %s is not available", h.Backend.Name())
	}

	// Check backend connectivity
	if !h.Backend.IsConnected(ctx) {
		return fmt.Errorf("backend %s is not connected", h.Backend.Name())
	}

	// Try to resolve hostname
	if _, err := h.resolveHostname(ctx); err != nil {
		return fmt.Errorf("hostname resolution failed: %w", err)
	}

	return nil
}

// selectProfile selects a profile either by name or interactively
func selectProfile(cfg *config.Config, profileName string) (*config.Profile, error) {
	if profileName != "" {
		// Profile name specified, retrieve it
		profile, err := cfg.GetProfile(profileName)
		if err != nil {
			return nil, fmt.Errorf("profile %q not found: %w", profileName, err)
		}
		return profile, nil
	}

	// No profile specified, use interactive selection
	selector := ui.NewProfileSelector(cfg)
	profile, _, err := selector.SelectProfile()
	if err != nil {
		return nil, fmt.Errorf("profile selection failed: %w", err)
	}

	return profile, nil
}

// PrintConnectionInfo prints connection information to the user
func (h *ConnectionHelper) PrintConnectionInfo() {
	ui.PrintInfo("Profile: %s", h.Profile.Name)
	if h.Profile.Description != "" {
		ui.PrintInfo("Description: %s", h.Profile.Description)
	}
	ui.PrintInfo("Backend: %s", h.Backend.Name())
	ui.PrintInfo("Remote: %s@%s:%d", h.Profile.RemoteUser, h.Profile.RemoteHost, h.Profile.SSHPort)
}
