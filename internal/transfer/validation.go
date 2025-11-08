// Package transfer - Path validation and security
// Copyright (c) 2025 orpheus497
package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath validates a file path for security issues
func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null byte")
	}

	// Check for path traversal attempts
	if !IsPathSafe(path) {
		return fmt.Errorf("path contains potentially unsafe traversal components")
	}

	return nil
}

// SanitizePath sanitizes a path by resolving it and checking for traversal
func SanitizePath(path string) (string, error) {
	if err := ValidatePath(path); err != nil {
		return "", err
	}

	// Expand tilde to home directory
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(homeDir, path[2:])
	}

	// Clean the path to remove redundant separators and . or .. elements
	cleaned := filepath.Clean(path)

	return cleaned, nil
}

// IsPathSafe checks if a path is safe from directory traversal attacks
func IsPathSafe(path string) bool {
	// Clean the path first
	cleaned := filepath.Clean(path)

	// Absolute paths are allowed
	if filepath.IsAbs(cleaned) {
		return true
	}

	// Check for parent directory references that escape the current context
	// This is a simplified check - in practice, the actual base path matters
	parts := strings.Split(filepath.ToSlash(cleaned), "/")
	depth := 0
	for _, part := range parts {
		if part == ".." {
			depth--
			if depth < 0 {
				// Attempting to go above the base directory
				return false
			}
		} else if part != "." && part != "" {
			depth++
		}
	}

	return true
}

// ResolveAbsolutePath resolves a path to an absolute path
func ResolveAbsolutePath(path string) (string, error) {
	sanitized, err := SanitizePath(path)
	if err != nil {
		return "", err
	}

	// If already absolute, return it
	if filepath.IsAbs(sanitized) {
		return sanitized, nil
	}

	// Make it absolute relative to current working directory
	absPath, err := filepath.Abs(sanitized)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	return absPath, nil
}

// ValidateSourcePath validates a source path for reading
func ValidateSourcePath(path string) error {
	if err := ValidatePath(path); err != nil {
		return fmt.Errorf("invalid source path: %w", err)
	}

	sanitized, err := SanitizePath(path)
	if err != nil {
		return fmt.Errorf("failed to sanitize source path: %w", err)
	}

	// For local sources, verify the file/directory exists
	// For remote sources, this check happens on the remote side
	if !strings.Contains(path, ":") {
		// This is a local path
		if _, err := os.Stat(sanitized); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("source path does not exist: %s", path)
			}
			return fmt.Errorf("cannot access source path: %w", err)
		}
	}

	return nil
}

// ValidateDestPath validates a destination path for writing
func ValidateDestPath(path string) error {
	if err := ValidatePath(path); err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	sanitized, err := SanitizePath(path)
	if err != nil {
		return fmt.Errorf("failed to sanitize destination path: %w", err)
	}

	// For local destinations, verify the parent directory exists and is writable
	// For remote destinations, this check happens on the remote side
	if !strings.Contains(path, ":") {
		// This is a local path
		// Check if it exists
		if info, err := os.Stat(sanitized); err == nil {
			// Path exists - if it's a directory, that's fine
			// If it's a file, we'll overwrite it (rsync/sftp behavior)
			if info.IsDir() {
				// Verify we can write to the directory
				testFile := filepath.Join(sanitized, ".klip_write_test")
				f, err := os.Create(testFile)
				if err != nil {
					return fmt.Errorf("destination directory is not writable: %w", err)
				}
				f.Close()
				os.Remove(testFile)
			}
		} else if os.IsNotExist(err) {
			// Path doesn't exist - verify parent directory exists and is writable
			parentDir := filepath.Dir(sanitized)
			if info, err := os.Stat(parentDir); err != nil {
				return fmt.Errorf("destination parent directory does not exist: %s", parentDir)
			} else if !info.IsDir() {
				return fmt.Errorf("destination parent is not a directory: %s", parentDir)
			}

			// Test write permission
			testFile := filepath.Join(parentDir, ".klip_write_test")
			f, err := os.Create(testFile)
			if err != nil {
				return fmt.Errorf("destination parent directory is not writable: %w", err)
			}
			f.Close()
			os.Remove(testFile)
		} else {
			return fmt.Errorf("cannot access destination path: %w", err)
		}
	}

	return nil
}

// ValidateTransferPaths validates both source and destination paths for a transfer
func ValidateTransferPaths(sourcePath, destPath string, direction TransferDirection) error {
	if direction == DirectionPush {
		// Pushing: source is local, dest is remote
		if err := ValidateSourcePath(sourcePath); err != nil {
			return err
		}
		// Remote destination validation happens on remote side
		if err := ValidatePath(destPath); err != nil {
			return fmt.Errorf("invalid destination path: %w", err)
		}
	} else {
		// Pulling: source is remote, dest is local
		// Remote source validation happens on remote side
		if err := ValidatePath(sourcePath); err != nil {
			return fmt.Errorf("invalid source path: %w", err)
		}
		if err := ValidateDestPath(destPath); err != nil {
			return err
		}
	}

	return nil
}

// IsWithinDirectory checks if a path is within a given base directory
// This helps prevent directory traversal attacks
func IsWithinDirectory(basePath, targetPath string) (bool, error) {
	// Resolve both paths to absolute paths
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve base path: %w", err)
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Evaluate symlinks
	absBase, err = filepath.EvalSymlinks(absBase)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to evaluate base path symlinks: %w", err)
	}

	absTarget, err = filepath.EvalSymlinks(absTarget)
	if err != nil && !os.IsNotExist(err) {
		// If target doesn't exist yet, that's okay for destination paths
		absTarget, _ = filepath.Abs(targetPath)
	}

	// Check if target is within base
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false, err
	}

	// If the relative path starts with "..", it's outside the base directory
	if strings.HasPrefix(rel, "..") {
		return false, nil
	}

	return true, nil
}
