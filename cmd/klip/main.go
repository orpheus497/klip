// klip - Remote connection tool with multi-VPN support
// Copyright (c) 2025 orpheus497
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/orpheus497/klip/internal/backend"
	"github.com/orpheus497/klip/internal/config"
	"github.com/orpheus497/klip/internal/ssh"
	"github.com/orpheus497/klip/internal/ui"
	"github.com/orpheus497/klip/internal/version"
	"github.com/spf13/cobra"
)

var (
	profileName     string
	backendName     string
	verbose         bool
	timeout         int
	showVersionFlag bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "klip [profile]",
		Short: "Connect to remote machines via SSH over VPN networks",
		Long: `klip is a remote connection tool that simplifies SSH access across
LAN, Tailscale, Headscale, and NetBird networks.

Created by orpheus497.`,
		Run: runConnect,
	}

	rootCmd.Flags().StringVarP(&profileName, "profile", "p", "", "Connection profile to use")
	rootCmd.Flags().StringVarP(&backendName, "backend", "b", "", "VPN backend (auto, lan, tailscale, headscale, netbird)")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "Connection timeout in seconds")
	rootCmd.Flags().BoolVar(&showVersionFlag, "version", false, "Show version information")

	// Subcommands
	rootCmd.AddCommand(profileCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(healthCmd())
	rootCmd.AddCommand(initCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runConnect(cmd *cobra.Command, args []string) {
	// Handle version flag
	if showVersionFlag {
		fmt.Println(version.String())
		return
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		ui.PrintInfo("Run 'klip init' to create initial configuration")
		os.Exit(1)
	}

	// Determine profile
	var profile *config.Profile
	var selectedProfileName string

	if len(args) > 0 {
		profileName = args[0]
	}

	if profileName != "" {
		profile, err = cfg.GetProfile(profileName)
		if err != nil {
			ui.PrintError("Profile not found: %s", profileName)
			os.Exit(1)
		}
		selectedProfileName = profileName
	} else {
		// Interactive selection
		selector := ui.NewProfileSelector(cfg)
		profile, selectedProfileName, err = selector.SelectProfile()
		if err != nil {
			ui.PrintError("Failed to select profile: %v", err)
			os.Exit(1)
		}
	}

	// Override backend if specified
	if backendName != "" {
		profile = profile.Clone()
		profile.Backend = config.BackendType(backendName)
	}

	// Validate profile
	if err := profile.Validate(); err != nil {
		ui.PrintError("Invalid profile configuration: %v", err)
		os.Exit(1)
	}

	ui.PrintInfo("Connecting to: %s (%s)", selectedProfileName, profile.Backend)

	// Select backend
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	registry := backend.NewRegistry()
	detector := backend.NewDetector(registry)

	selectedBackend, err := detector.SelectBackend(ctx, string(profile.Backend))
	if err != nil {
		ui.PrintError("Failed to select backend: %v", err)
		os.Exit(1)
	}

	if verbose {
		ui.PrintInfo("Using backend: %s", selectedBackend.Name())
	}

	// Resolve host
	resolvedHost := profile.RemoteHost

	if selectedBackend.Name() != "lan" {
		if verbose {
			ui.PrintInfo("Resolving host via %s...", selectedBackend.Name())
		}

		ip, err := detector.ResolveHost(ctx, selectedBackend, profile.RemoteHost)
		if err != nil {
			ui.PrintWarning("Failed to resolve via %s, using hostname: %v", selectedBackend.Name(), err)
		} else {
			resolvedHost = ip
			if verbose {
				ui.PrintInfo("Resolved to: %s", resolvedHost)
			}
		}
	}

	// Create SSH client
	sshConfig := &ssh.Config{
		Host:        resolvedHost,
		Port:        profile.SSHPort,
		User:        profile.RemoteUser,
		KeyPath:     profile.SSHKeyPath,
		UsePassword: profile.UsePassword,
		Timeout:     time.Duration(timeout) * time.Second,
	}

	client, err := ssh.NewClient(sshConfig)
	if err != nil {
		ui.PrintError("Failed to create SSH client: %v", err)
		os.Exit(1)
	}

	if verbose {
		ui.PrintInfo("Connecting via SSH...")
	}

	// Connect
	if err := client.Connect(ctx); err != nil {
		ui.PrintError("Connection failed: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	ui.PrintSuccess("Connected to %s@%s", profile.RemoteUser, resolvedHost)

	// Start interactive shell
	if err := client.InteractiveShell(); err != nil {
		ui.PrintError("Shell error: %v", err)
		os.Exit(1)
	}
}

func profileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage connection profiles",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Run:   runProfileList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "add",
		Short: "Add a new profile",
		Run:   runProfileAdd,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "remove <profile>",
		Short: "Remove a profile",
		Args:  cobra.ExactArgs(1),
		Run:   runProfileRemove,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set-current <profile>",
		Short: "Set the current profile",
		Args:  cobra.ExactArgs(1),
		Run:   runProfileSetCurrent,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "validate <profile>",
		Short: "Validate a profile configuration",
		Long:  "Validates profile settings and tests connectivity without establishing a full connection",
		Args:  cobra.ExactArgs(1),
		Run:   runProfileValidate,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "edit <profile>",
		Short: "Edit a profile interactively",
		Long:  "Opens an interactive editor to modify profile settings",
		Args:  cobra.ExactArgs(1),
		Run:   runProfileEdit,
	})

	return cmd
}

func runProfileList(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	profiles := cfg.ListProfiles()
	if len(profiles) == 0 {
		ui.PrintInfo("No profiles configured")
		return
	}

	ui.PrintHeader("Connection Profiles")

	for _, name := range profiles {
		profile, err := cfg.GetProfile(name)
		if err != nil {
			continue
		}

		marker := " "
		if name == cfg.CurrentProfile {
			marker = ui.Success("●")
		}

		fmt.Printf("%s %s\n", marker, ui.Bold(name))
		fmt.Printf("  User: %s\n", profile.RemoteUser)
		fmt.Printf("  Host: %s\n", profile.RemoteHost)
		fmt.Printf("  Backend: %s\n", profile.Backend)
		if profile.Description != "" {
			fmt.Printf("  Description: %s\n", ui.Dim(profile.Description))
		}
		ui.PrintEmptyLine()
	}
}

func runProfileAdd(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	profile, name, err := ui.CreateProfileInteractive()
	if err != nil {
		ui.PrintError("Failed to create profile: %v", err)
		os.Exit(1)
	}

	if err := cfg.AddProfile(name, profile); err != nil {
		ui.PrintError("Failed to add profile: %v", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		ui.PrintError("Failed to save configuration: %v", err)
		os.Exit(1)
	}

	ui.PrintSuccess("Profile '%s' added successfully", name)
}

func runProfileRemove(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	name := args[0]

	if !ui.ConfirmDefaultNo(fmt.Sprintf("Remove profile '%s'?", name)) {
		ui.PrintInfo("Cancelled")
		return
	}

	if err := cfg.DeleteProfile(name); err != nil {
		ui.PrintError("Failed to remove profile: %v", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		ui.PrintError("Failed to save configuration: %v", err)
		os.Exit(1)
	}

	ui.PrintSuccess("Profile '%s' removed", name)
}

func runProfileSetCurrent(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	name := args[0]

	if err := cfg.SetCurrentProfile(name); err != nil {
		ui.PrintError("Failed to set current profile: %v", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		ui.PrintError("Failed to save configuration: %v", err)
		os.Exit(1)
	}

	ui.PrintSuccess("Current profile set to '%s'", name)
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show backend status",
		Run:   runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	registry := backend.NewRegistry()
	detector := backend.NewDetector(registry)

	allStatus := detector.DetectAll(ctx)

	ui.PrintHeader("VPN Backend Status")

	headers := []string{"Backend", "Status", "IP Address", "Message"}
	var rows [][]string

	for name, status := range allStatus {
		statusStr := ui.Error("✗ Disconnected")
		if status.Connected {
			statusStr = ui.Success("✓ Connected")
		}

		rows = append(rows, []string{
			name,
			statusStr,
			status.LocalIP,
			status.Message,
		})
	}

	ui.PrintTable(headers, rows)
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			info := version.GetInfo()
			ui.PrintHeader("klip Version Information")
			ui.PrintKeyValue("Version", info.Version)
			ui.PrintKeyValue("Git Commit", info.GitCommit)
			ui.PrintKeyValue("Build Date", info.BuildDate)
			ui.PrintKeyValue("Go Version", info.GoVersion)
			ui.PrintKeyValue("Platform", info.Platform)
		},
	}
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check backend and connection health",
		Run:   runHealth,
	}
}

func runHealth(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	registry := backend.NewRegistry()
	detector := backend.NewDetector(registry)

	ui.PrintHeader("Health Check")
	ui.PrintEmptyLine()

	results := detector.HealthCheck(ctx)

	for _, result := range results {
		status := ui.Error("✗")
		if result.Available && result.Connected {
			status = ui.Success("✓")
		} else if result.Available {
			status = ui.Warning("○")
		}

		fmt.Printf("%s %s: %s (%.2fs)\n",
			status,
			ui.Bold(result.Backend),
			result.Message,
			result.Duration.Seconds())
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize klip configuration",
		Run:   runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) {
	// Check migration status
	migrationStatus := config.CheckMigrationStatus()

	if migrationStatus.ModernConfigExists {
		ui.PrintInfo("Configuration already exists")

		if !ui.Confirm("Re-initialize configuration?") {
			return
		}
	}

	cfg := config.NewConfig()

	// Attempt migration if legacy config exists
	if migrationStatus.CanMigrate {
		ui.PrintInfo("Found legacy LINK configuration")

		if ui.Confirm("Migrate existing profiles?") {
			migrated, err := config.MigrateLegacyConfig()
			if err != nil {
				ui.PrintWarning("Migration failed: %v", err)
			} else {
				cfg = migrated
				ui.PrintSuccess("Migrated %d profile(s)", len(cfg.Profiles))
			}
		}
	}

	// Create first profile if none exist
	if len(cfg.Profiles) == 0 {
		ui.PrintInfo("No profiles found. Let's create your first profile.")

		profile, name, err := ui.CreateProfileInteractive()
		if err != nil {
			ui.PrintError("Failed to create profile: %v", err)
			os.Exit(1)
		}

		cfg.AddProfile(name, profile)
	}

	// Save configuration
	if err := cfg.Save(); err != nil {
		ui.PrintError("Failed to save configuration: %v", err)
		os.Exit(1)
	}

	configPath, _ := config.ConfigPath()
	ui.PrintSuccess("Configuration initialized: %s", configPath)
}

func runProfileValidate(cmd *cobra.Command, args []string) {
	profileName := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Get profile
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		ui.PrintError("Profile not found: %s", profileName)
		os.Exit(1)
	}

	ui.PrintHeader(fmt.Sprintf("Validating Profile: %s", profileName))
	ui.PrintEmptyLine()

	// Validate profile configuration
	ui.PrintInfo("Checking profile configuration...")
	if err := profile.Validate(); err != nil {
		ui.PrintError("Profile validation failed: %v", err)
		os.Exit(1)
	}
	ui.PrintSuccess("Profile configuration is valid")

	// Validate port
	ui.PrintInfo("Validating SSH port...")
	if err := config.ValidatePort(profile.SSHPort); err != nil {
		ui.PrintError("Invalid port: %v", err)
		os.Exit(1)
	}
	ui.PrintSuccess("Port %d is valid", profile.SSHPort)

	// Validate hostname
	ui.PrintInfo("Validating hostname...")
	if err := config.ValidateHostname(profile.RemoteHost); err != nil {
		ui.PrintError("Invalid hostname: %v", err)
		os.Exit(1)
	}
	ui.PrintSuccess("Hostname %s is valid", profile.RemoteHost)

	// Validate username
	ui.PrintInfo("Validating username...")
	if err := config.ValidateUsername(profile.RemoteUser); err != nil {
		ui.PrintError("Invalid username: %v", err)
		os.Exit(1)
	}
	ui.PrintSuccess("Username %s is valid", profile.RemoteUser)

	// Validate SSH key if specified
	if profile.SSHKeyPath != "" {
		ui.PrintInfo("Validating SSH key...")
		if err := config.ValidateSSHKeyPath(profile.SSHKeyPath); err != nil {
			ui.PrintError("Invalid SSH key: %v", err)
			os.Exit(1)
		}
		ui.PrintSuccess("SSH key is valid")
	}

	// Check backend availability
	ui.PrintInfo("Checking backend availability...")
	ctx := context.Background()
	registry := backend.NewRegistry()
	detector := backend.NewDetector(registry)
	selectedBackend, err := detector.SelectBackend(ctx, string(profile.Backend))
	if err != nil {
		ui.PrintError("Backend detection failed: %v", err)
		os.Exit(1)
	}

	if !selectedBackend.IsAvailable(ctx) {
		ui.PrintWarning("Backend %s is not available", selectedBackend.Name())
	} else {
		ui.PrintSuccess("Backend %s is available", selectedBackend.Name())

		// Check if backend is connected
		if selectedBackend.IsConnected(ctx) {
			ui.PrintSuccess("Backend %s is connected", selectedBackend.Name())

			// Try to resolve hostname
			ui.PrintInfo("Resolving hostname via %s...", selectedBackend.Name())
			resolvedHost, err := selectedBackend.GetPeerIP(ctx, profile.RemoteHost)
			if err != nil {
				ui.PrintWarning("Failed to resolve hostname: %v", err)
			} else {
				ui.PrintSuccess("Hostname resolved to: %s", resolvedHost)
			}
		} else {
			ui.PrintWarning("Backend %s is not connected", selectedBackend.Name())
		}
	}

	ui.PrintEmptyLine()
	ui.PrintSuccess("Profile validation complete!")
	ui.PrintInfo("Profile %s appears to be properly configured", profileName)
}

func runProfileEdit(cmd *cobra.Command, args []string) {
	profileName := args[0]

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		os.Exit(1)
	}

	// Get profile
	profile, err := cfg.GetProfile(profileName)
	if err != nil {
		ui.PrintError("Profile not found: %s", profileName)
		os.Exit(1)
	}

	// Edit profile interactively (modifies profile in place)
	if err := ui.EditProfileInteractive(profile); err != nil {
		ui.PrintError("Failed to edit profile: %v", err)
		os.Exit(1)
	}

	// Validate updated profile
	if err := profile.Validate(); err != nil {
		ui.PrintError("Updated profile is invalid: %v", err)
		ui.PrintWarning("Changes not saved")
		os.Exit(1)
	}

	// Update profile in configuration
	cfg.Profiles[profileName] = profile

	// Save configuration
	if err := cfg.Save(); err != nil {
		ui.PrintError("Failed to save configuration: %v", err)
		os.Exit(1)
	}

	ui.PrintSuccess("Profile %s updated successfully", profileName)
}
