package transfer

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// RsyncTransfer implements file transfer using rsync
type RsyncTransfer struct {
	config           *TransferConfig
	progressCallback ProgressCallback
}

// NewRsyncTransfer creates a new rsync-based transfer
func NewRsyncTransfer(cfg *TransferConfig) *RsyncTransfer {
	return &RsyncTransfer{
		config: cfg,
	}
}

// SetProgressCallback sets the progress callback
func (r *RsyncTransfer) SetProgressCallback(callback ProgressCallback) {
	r.progressCallback = callback
}

// Execute performs the rsync transfer
func (r *RsyncTransfer) Execute(ctx context.Context) error {
	// Check if rsync is available
	if _, err := exec.LookPath("rsync"); err != nil {
		return fmt.Errorf("rsync not found in PATH: %w", err)
	}

	// Build rsync command
	args := r.buildRsyncArgs()

	cmd := exec.CommandContext(ctx, "rsync", args...)

	// Capture output for progress parsing
	if r.config.ShowProgress && r.progressCallback != nil {
		return r.executeWithProgress(ctx, cmd)
	}

	// Execute without progress
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rsync failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// buildRsyncArgs builds the argument list for rsync
func (r *RsyncTransfer) buildRsyncArgs() []string {
	args := []string{}

	// Archive mode (preserves permissions, timestamps, etc.)
	if r.config.PreservePermissions {
		args = append(args, "-a")
	} else {
		args = append(args, "-r") // Recursive
	}

	// Verbose mode
	args = append(args, "-v")

	// Progress information
	if r.config.ShowProgress {
		args = append(args, "--progress")
	}

	// Compression
	if r.config.CompressionLevel > 0 {
		args = append(args, "-z")
		args = append(args, fmt.Sprintf("--compress-level=%d", r.config.CompressionLevel))
	}

	// Bandwidth limit
	if r.config.BandwidthLimit > 0 {
		args = append(args, fmt.Sprintf("--bwlimit=%d", r.config.BandwidthLimit))
	}

	// Exclude patterns - validate each pattern for security
	for _, pattern := range r.config.ExcludePatterns {
		// Validate pattern to prevent injection attacks
		if err := ValidateExcludePattern(pattern); err != nil {
			// Skip invalid patterns and log warning
			// In a production system, this should use proper logging
			continue
		}
		args = append(args, "--exclude", pattern)
	}

	// Delete source after transfer
	if r.config.DeleteAfterTransfer {
		args = append(args, "--remove-source-files")
	}

	// Dry run
	if r.config.DryRun {
		args = append(args, "--dry-run")
	}

	// Partial transfer support (resume)
	args = append(args, "--partial")

	// SSH options
	sshArgs := r.buildSSHArgs()
	if len(sshArgs) > 0 {
		args = append(args, "-e", fmt.Sprintf("ssh %s", strings.Join(sshArgs, " ")))
	}

	// Determine the host to use (resolved host takes precedence)
	remoteHost := r.config.Profile.RemoteHost
	if r.config.ResolvedHost != "" {
		remoteHost = r.config.ResolvedHost
	}

	// Source and destination
	if r.config.Direction == DirectionPush {
		// Local to remote
		args = append(args, r.config.SourcePath)
		args = append(args, fmt.Sprintf("%s@%s:%s",
			r.config.Profile.RemoteUser,
			remoteHost,
			r.config.DestPath))
	} else {
		// Remote to local
		args = append(args, fmt.Sprintf("%s@%s:%s",
			r.config.Profile.RemoteUser,
			remoteHost,
			r.config.SourcePath))
		args = append(args, r.config.DestPath)
	}

	return args
}

// buildSSHArgs builds SSH arguments for rsync
func (r *RsyncTransfer) buildSSHArgs() []string {
	args := []string{}

	// SSH port
	if r.config.Profile.SSHPort != 22 {
		args = append(args, "-p", strconv.Itoa(r.config.Profile.SSHPort))
	}

	// SSH key
	if r.config.Profile.SSHKeyPath != "" {
		args = append(args, "-i", r.config.Profile.SSHKeyPath)
	}

	// SECURITY: Never disable strict host key checking as it prevents MITM attacks
	// Host key verification is handled automatically via klip's known_hosts management
	// in ~/.config/klip/known_hosts. If you encounter host key errors, use:
	//   klip health --verify-host <profile>

	return args
}

// executeWithProgress executes rsync and parses progress output
func (r *RsyncTransfer) executeWithProgress(ctx context.Context, cmd *exec.Cmd) error {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start rsync: %w", err)
	}

	// Parse output in goroutine
	done := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Check context periodically during scanning
			select {
			case <-ctx.Done():
				return
			default:
			}

			line := scanner.Text()
			r.parseProgressLine(line)
		}

		// Also read stderr
		stderrScanner := bufio.NewScanner(stderr)
		for stderrScanner.Scan() {
			// Check context periodically
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Log stderr but don't parse for progress
			line := stderrScanner.Text()
			if r.progressCallback != nil {
				r.progressCallback(ProgressInfo{
					Message: line,
				})
			}
		}

		waitErr := cmd.Wait()

		// Non-blocking send to prevent deadlock if context was cancelled
		select {
		case done <- waitErr:
		case <-ctx.Done():
			// Context cancelled, don't block on send
		}
	}()

	select {
	case <-ctx.Done():
		// Kill the process if still running
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// parseProgressLine parses a line of rsync output for progress information
func (r *RsyncTransfer) parseProgressLine(line string) {
	if r.progressCallback == nil {
		return
	}

	// Rsync progress line format:
	// filename
	//     1,234,567  50%  123.45MB/s    0:00:12

	// Try to match progress line
	progressRegex := regexp.MustCompile(`\s+([\d,]+)\s+(\d+)%\s+([\d.]+\w+/s)\s+(\d+:\d+:\d+)`)
	matches := progressRegex.FindStringSubmatch(line)

	if len(matches) == 5 {
		// Extract transferred bytes
		bytesStr := strings.ReplaceAll(matches[1], ",", "")
		transferred, _ := strconv.ParseInt(bytesStr, 10, 64)

		// Extract percentage
		percentage, _ := strconv.Atoi(matches[2])

		// Calculate total bytes from percentage
		var total int64
		if percentage > 0 {
			total = (transferred * 100) / int64(percentage)
		}

		// Speed is in matches[3] but we'll skip parsing it for now

		r.progressCallback(ProgressInfo{
			TransferredBytes: transferred,
			TotalBytes:       total,
			Message:          line,
		})
	} else {
		// Just send the line as a message
		r.progressCallback(ProgressInfo{
			Message: line,
		})
	}
}
