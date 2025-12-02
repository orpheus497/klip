// klipc - Copy files to remote machines
// Copyright (c) 2025 orpheus497
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/orpheus497/klip/internal/cli"
	"github.com/orpheus497/klip/internal/logger"
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
		Use:   "klipc <source> [destination]",
		Short: "Copy files to remote machines",
		Long: `klipc copies files from local machine to remote machines via SSH,
with support for multiple VPN backends.

Created by orpheus497.`,
		Args: cobra.MinimumNArgs(1),
		Run:  runCopy,
	}

	rootCmd.Flags().StringVarP(&profileName, "profile", "p", "", "Connection profile to use")
	rootCmd.Flags().StringVarP(&backendName, "backend", "b", "", "VPN backend (auto, lan, tailscale, headscale, netbird)")
	rootCmd.Flags().StringVarP(&destPath, "dest", "d", "", "Destination path on remote (defaults to same as source)")
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

func runCopy(cmd *cobra.Command, args []string) {
	sourcePath := args[0]

	// Check if source exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		ui.PrintError("Source path does not exist: %s", sourcePath)
		os.Exit(1)
	}

	// Determine destination path
	if len(args) > 1 {
		destPath = args[1]
	}
	if destPath == "" {
		// Default to same path as source (relative to home directory)
		destPath = sourcePath
	}

	// Initialize audit logger (enabled by default for security tracking)
	auditLogger, err := logger.NewAuditLogger(true)
	if err != nil {
		ui.PrintWarning("Failed to initialize audit logger: %v", err)
		// Create disabled logger as fallback
		auditLogger, _ = logger.NewAuditLogger(false)
	}
	defer auditLogger.Close()

	// Create connection helper (centralizes connection setup)
	helper, err := cli.NewConnectionHelper(cli.ConnectionConfig{
		ProfileName: profileName,
		BackendName: backendName,
		Timeout:     timeout,
		Verbose:     verbose,
	})
	if err != nil {
		ui.PrintError("Failed to initialize connection: %v", err)
		ui.PrintInfo("Run 'klip init' to create initial configuration")
		os.Exit(1)
	}

	// Override transfer method if specified
	if method != "" {
		helper.Profile.TransferOptions.Method = method
	}

	// Override compression if specified
	if cmd.Flags().Changed("compress") {
		helper.Profile.TransferOptions.CompressionLevel = compressionLevel
	}

	ui.PrintInfo("Copying to: %s@%s:%s", helper.Profile.RemoteUser, helper.Profile.RemoteHost, destPath)
	if dryRun {
		ui.PrintWarning("DRY RUN - No files will be transferred")
	}

	// Create context with timeout
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Create SSH client using connection helper
	client, err := helper.CreateSSHClient(ctx, timeout)
	if err != nil {
		// Log failed connection attempt
		_ = auditLogger.LogTransfer(
			helper.Profile.Name,
			helper.Profile.RemoteUser,
			helper.Profile.RemoteHost,
			helper.Backend.Name(),
			"push",
			sourcePath,
			destPath,
			"failed",
			err,
		)
		ui.PrintError("Connection failed: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	// Configure transfer
	transferConfig := &transfer.TransferConfig{
		SSHClient:           client,
		Profile:             helper.Profile,
		ResolvedHost:        helper.ResolvedHost,
		SourcePath:          sourcePath,
		DestPath:            destPath,
		Direction:           transfer.DirectionPush,
		Method:              helper.Profile.TransferOptions.Method,
		CompressionLevel:    helper.Profile.TransferOptions.CompressionLevel,
		ExcludePatterns:     helper.Profile.TransferOptions.ExcludePatterns,
		BandwidthLimit:      helper.Profile.TransferOptions.BandwidthLimit,
		PreservePermissions: helper.Profile.TransferOptions.PreservePermissions,
		DeleteAfterTransfer: helper.Profile.TransferOptions.DeleteAfterTransfer,
		DryRun:              dryRun,
		ShowProgress:        true,
	}

	// Create transfer
	xfer, err := transfer.NewTransfer(transferConfig)
	if err != nil {
		// Log failed transfer setup
		_ = auditLogger.LogTransfer(
			helper.Profile.Name,
			helper.Profile.RemoteUser,
			helper.Profile.RemoteHost,
			helper.Backend.Name(),
			"push",
			sourcePath,
			destPath,
			"failed",
			err,
		)
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

	transferErr := xfer.Execute(ctx)
	elapsed := time.Since(startTime)

	// Determine transfer status for audit log
	status := "success"
	if transferErr != nil {
		status = "failed"
	}
	if dryRun {
		status = "dry_run"
	}

	// Log transfer result
	_ = auditLogger.LogTransfer(
		helper.Profile.Name,
		helper.Profile.RemoteUser,
		helper.Profile.RemoteHost,
		helper.Backend.Name(),
		"push",
		sourcePath,
		destPath,
		status,
		transferErr,
	)

	if transferErr != nil {
		ui.PrintError("Transfer failed: %v", transferErr)
		os.Exit(1)
	}

	if dryRun {
		ui.PrintSuccess("Dry run completed in %.2fs", elapsed.Seconds())
	} else {
		ui.PrintSuccess("Transfer completed in %.2fs", elapsed.Seconds())
	}
}
