// klipr - Retrieve files from remote machines
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
	"github.com/orpheus497/klip/internal/transfer"
	"github.com/orpheus497/klip/internal/ui"
	"github.com/orpheus497/klip/internal/version"
	"github.com/spf13/cobra"
)

var (
	profileName      string
	backendName      string
	destPath         string
	method           string
	compressionLevel int
	dryRun           bool
	verbose          bool
	timeout          int
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "klipr <remote-source> [local-destination]",
		Short: "Retrieve files from remote machines",
		Long: `klipr retrieves files from remote machines to local machine via SSH,
with support for multiple VPN backends.

Created by orpheus497.`,
		Args: cobra.MinimumNArgs(1),
		Run:  runRetrieve,
	}

	rootCmd.Flags().StringVarP(&profileName, "profile", "p", "", "Connection profile to use")
	rootCmd.Flags().StringVarP(&backendName, "backend", "b", "", "VPN backend (auto, lan, tailscale, headscale, netbird)")
	rootCmd.Flags().StringVarP(&destPath, "dest", "d", "", "Local destination path (defaults to current directory)")
	rootCmd.Flags().StringVarP(&method, "method", "m", "rsync", "Transfer method (rsync, sftp)")
	rootCmd.Flags().IntVarP(&compressionLevel, "compress", "z", 6, "Compression level (0-9, 0=disabled)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be transferred without actually doing it")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "Connection timeout in seconds")

	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	})

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runRetrieve(cmd *cobra.Command, args []string) {
	remotePath := args[0]

	// Determine local destination path
	if len(args) > 1 {
		destPath = args[1]
	}
	if destPath == "" {
		// Default to current directory
		cwd, err := os.Getwd()
		if err != nil {
			ui.PrintError("Failed to get current directory: %v", err)
			os.Exit(1)
		}
		destPath = cwd
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		ui.PrintError("Failed to load configuration: %v", err)
		ui.PrintInfo("Run 'klip init' to create initial configuration")
		os.Exit(1)
	}

	// Select profile
	var profile *config.Profile

	if profileName != "" {
		profile, err = cfg.GetProfile(profileName)
		if err != nil {
			ui.PrintError("Profile not found: %s", profileName)
			os.Exit(1)
		}
	} else {
		// Use current profile or interactive selection
		if cfg.CurrentProfile != "" {
			profile, err = cfg.GetCurrentProfile()
			if err != nil {
				profile = nil
			}
		}

		if profile == nil {
			selector := ui.NewProfileSelector(cfg)
			profile, _, err = selector.SelectProfile()
			if err != nil {
				ui.PrintError("Failed to select profile: %v", err)
				os.Exit(1)
			}
		}
	}

	// Override backend if specified
	if backendName != "" {
		profile = profile.Clone()
		profile.Backend = config.BackendType(backendName)
	}

	// Override method if specified
	if method != "" {
		profile.TransferOptions.Method = method
	}

	// Override compression if specified
	if cmd.Flags().Changed("compress") {
		profile.TransferOptions.CompressionLevel = compressionLevel
	}

	ui.PrintInfo("Retrieving from: %s@%s:%s", profile.RemoteUser, profile.RemoteHost, remotePath)
	ui.PrintInfo("Destination: %s", destPath)
	if dryRun {
		ui.PrintWarning("DRY RUN - No files will be transferred")
	}

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

	// Resolve host
	resolvedHost := profile.RemoteHost

	if selectedBackend.Name() != "lan" {
		if verbose {
			ui.PrintInfo("Resolving host via %s...", selectedBackend.Name())
		}

		ip, err := detector.ResolveHost(ctx, selectedBackend, profile.RemoteHost)
		if err != nil {
			ui.PrintWarning("Failed to resolve via %s, using hostname", selectedBackend.Name())
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

	// Connect if using SFTP
	if profile.TransferOptions.Method == "sftp" {
		if err := client.Connect(ctx); err != nil {
			ui.PrintError("Connection failed: %v", err)
			os.Exit(1)
		}
		defer client.Close()
	}

	// Configure transfer
	transferConfig := &transfer.TransferConfig{
		SSHClient:           client,
		Profile:             profile,
		SourcePath:          remotePath,
		DestPath:            destPath,
		Direction:           transfer.DirectionPull,
		Method:              profile.TransferOptions.Method,
		CompressionLevel:    profile.TransferOptions.CompressionLevel,
		ExcludePatterns:     profile.TransferOptions.ExcludePatterns,
		BandwidthLimit:      profile.TransferOptions.BandwidthLimit,
		PreservePermissions: profile.TransferOptions.PreservePermissions,
		DeleteAfterTransfer: profile.TransferOptions.DeleteAfterTransfer,
		DryRun:              dryRun,
		ShowProgress:        true,
	}

	// Create transfer
	xfer, err := transfer.NewTransfer(transferConfig)
	if err != nil {
		ui.PrintError("Failed to create transfer: %v", err)
		os.Exit(1)
	}

	// Set progress callback
	if verbose || dryRun {
		xfer.SetProgressCallback(func(info transfer.ProgressInfo) {
			if info.Message != "" {
				fmt.Println(info.Message)
			}
		})
	}

	// Execute transfer
	startTime := time.Now()

	if err := xfer.Execute(ctx); err != nil {
		ui.PrintError("Transfer failed: %v", err)
		os.Exit(1)
	}

	elapsed := time.Since(startTime)

	if dryRun {
		ui.PrintSuccess("Dry run completed in %.2fs", elapsed.Seconds())
	} else {
		ui.PrintSuccess("Transfer completed in %.2fs", elapsed.Seconds())
	}
}
