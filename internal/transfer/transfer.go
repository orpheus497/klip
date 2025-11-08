// Package transfer provides file transfer functionality for klip
package transfer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orpheus497/klip/internal/config"
	"github.com/orpheus497/klip/internal/ssh"
)

// TransferDirection indicates the direction of file transfer
type TransferDirection int

const (
	// DirectionPush transfers files from local to remote
	DirectionPush TransferDirection = iota

	// DirectionPull transfers files from remote to local
	DirectionPull
)

// Transfer represents a file transfer operation
type Transfer interface {
	// Execute performs the transfer
	Execute(ctx context.Context) error

	// SetProgressCallback sets a callback for progress updates
	SetProgressCallback(callback ProgressCallback)
}

// TransferConfig contains configuration for a transfer operation
type TransferConfig struct {
	// SSHClient is the SSH connection to use
	SSHClient *ssh.Client

	// Profile contains connection profile information
	Profile *config.Profile

	// SourcePath is the source file or directory path
	SourcePath string

	// DestPath is the destination file or directory path
	DestPath string

	// Direction indicates push or pull
	Direction TransferDirection

	// Method specifies transfer method (rsync, sftp)
	Method string

	// CompressionLevel for rsync (0-9)
	CompressionLevel int

	// ExcludePatterns for rsync
	ExcludePatterns []string

	// BandwidthLimit in KB/s (0=unlimited)
	BandwidthLimit int

	// PreservePermissions maintains file permissions
	PreservePermissions bool

	// DeleteAfterTransfer removes source after successful transfer
	DeleteAfterTransfer bool

	// DryRun performs a trial run without making changes
	DryRun bool

	// ShowProgress displays progress information
	ShowProgress bool
}

// ProgressInfo contains transfer progress information
type ProgressInfo struct {
	// TotalBytes is the total size in bytes
	TotalBytes int64

	// TransferredBytes is the number of bytes transferred
	TransferredBytes int64

	// CurrentFile is the file currently being transferred
	CurrentFile string

	// FilesTotal is the total number of files
	FilesTotal int

	// FilesTransferred is the number of files completed
	FilesTransferred int

	// Speed is the current transfer speed in bytes/second
	Speed int64

	// Message is a status message
	Message string
}

// ProgressCallback is called to report transfer progress
type ProgressCallback func(info ProgressInfo)

// NewTransfer creates a new transfer based on the configured method
func NewTransfer(cfg *TransferConfig) (Transfer, error) {
	if cfg.Method == "" {
		cfg.Method = "rsync"
	}

	// Validate source path for push operations
	if cfg.Direction == DirectionPush {
		if _, err := os.Stat(cfg.SourcePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("source path does not exist: %s", cfg.SourcePath)
		}
	}

	// Normalize paths
	cfg.SourcePath = normalizePath(cfg.SourcePath)
	cfg.DestPath = normalizePath(cfg.DestPath)

	switch cfg.Method {
	case "rsync":
		return NewRsyncTransfer(cfg), nil
	case "sftp":
		return NewSFTPTransfer(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported transfer method: %s", cfg.Method)
	}
}

// normalizePath normalizes a file path
func normalizePath(path string) string {
	// Expand ~ to home directory
	if path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}

	// Clean the path
	return filepath.Clean(path)
}

// isDirectory checks if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// FormatBytes formats bytes into a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatSpeed formats transfer speed into a human-readable string
func FormatSpeed(bytesPerSecond int64) string {
	return FormatBytes(bytesPerSecond) + "/s"
}
