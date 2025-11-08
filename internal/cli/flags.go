// Package cli provides common CLI utilities for klip commands
// Copyright (c) 2025 orpheus497
package cli

import (
	"github.com/spf13/cobra"
)

// Common flag variables shared across klip commands
var (
	// Profile flags
	ProfileName string

	// Backend flags
	BackendName string

	// Connection flags
	Verbose bool
	Timeout int
	DryRun  bool

	// Transfer flags
	DestPath         string
	Method           string
	CompressionLevel int
)

// AddProfileFlags adds profile-related flags to a command
func AddProfileFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&ProfileName, "profile", "p", "", "Connection profile to use")
}

// AddBackendFlags adds backend-related flags to a command
func AddBackendFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&BackendName, "backend", "b", "", "VPN backend (auto, lan, tailscale, headscale, netbird)")
}

// AddConnectionFlags adds connection-related flags to a command
func AddConnectionFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&Verbose, "verbose", "v", false, "Enable verbose output")
	cmd.Flags().IntVarP(&Timeout, "timeout", "t", 30, "Connection timeout in seconds")
}

// AddDryRunFlag adds the dry-run flag to a command
func AddDryRunFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&DryRun, "dry-run", false, "Show what would be done without actually doing it")
}

// AddTransferFlags adds file transfer-related flags to a command
func AddTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&DestPath, "dest", "d", "", "Destination path")
	cmd.Flags().StringVarP(&Method, "method", "m", "rsync", "Transfer method (rsync, sftp)")
	cmd.Flags().IntVarP(&CompressionLevel, "compress", "z", 6, "Compression level (0-9, 0=disabled)")
}

// AddCommonFlags adds all common flags to a command (profile, backend, connection)
func AddCommonFlags(cmd *cobra.Command) {
	AddProfileFlags(cmd)
	AddBackendFlags(cmd)
	AddConnectionFlags(cmd)
}

// AddAllFlags adds all available flags to a command
func AddAllFlags(cmd *cobra.Command) {
	AddCommonFlags(cmd)
	AddDryRunFlag(cmd)
	AddTransferFlags(cmd)
}

// ResetFlags resets all flag variables to their defaults
// This is primarily useful for testing
func ResetFlags() {
	ProfileName = ""
	BackendName = ""
	Verbose = false
	Timeout = 30
	DryRun = false
	DestPath = ""
	Method = "rsync"
	CompressionLevel = 6
}
