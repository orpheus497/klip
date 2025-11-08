# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- Fixed critical command injection vulnerability in SSH key deployment (DeployPublicKey function now uses SFTP instead of shell commands)
- Removed dangerous StrictHostKeyChecking suggestion from rsync implementation
- Added comprehensive input validation for rsync exclude patterns to prevent injection attacks
- Implemented input sanitization for all interactive prompts to prevent terminal injection
- Fixed TOCTOU race condition in path validation logic with improved atomic operations
- Added comprehensive validation for SSH port ranges, hostnames, and usernames
- Added SSH key path validation with permission and format checking

### Added

- Created connection helper module (internal/cli/connection.go) to eliminate code duplication across commands
- Created audit logging system (internal/logger/audit.go) for security event tracking
- Added profile validation command (klip profile validate <profile>) to test configurations without connecting
- Added profile editing command (klip profile edit <profile>) for interactive profile modification
- Added ValidateExcludePattern function for secure rsync pattern validation
- Added ValidatePort, ValidateHostname, ValidateUsername validation functions
- Added ValidateSSHKeyPath with permission checks (0600) and format validation
- Added ValidateBandwidthLimit and ValidateCompressionLevel functions
- Added sanitizeInput function for filtering terminal control characters and ANSI sequences
- Added toUnixPath helper function for consistent remote path handling
- Created legacy/ directory structure for historical script preservation
- Added comprehensive security-focused path validation throughout transfer operations
- Added ConnectionHelper for centralized SSH client creation and backend management
- Added AuditLogger for JSON-formatted security event logging with XDG compliance

### Fixed

- Fixed command injection vulnerability in internal/ssh/keys.go DeployPublicKey function (replaced shell echo with SFTP file operations)
- Fixed incorrect use of path package instead of filepath in SFTP operations (internal/transfer/sftp.go)
- Fixed potential goroutine deadlock in rsync progress parser with non-blocking channel sends
- Fixed missing test file cleanup in path validation functions (added defer cleanup)
- Fixed path handling to ensure remote paths use Unix-style forward slashes regardless of local OS
- Fixed exclude pattern handling to validate patterns before use in rsync

### Changed

- Refactored klipc command to use ConnectionHelper, eliminating ~100 lines of duplicated connection code
- Refactored klipr command to use ConnectionHelper, eliminating ~100 lines of duplicated connection code
- Integrated audit logging into klipc for tracking all file push operations (success, failure, dry-run)
- Integrated audit logging into klipr for tracking all file pull operations (success, failure, dry-run)
- Replaced shell-based SSH public key deployment with secure SFTP-based atomic file writes
- Enhanced rsync implementation with context-aware non-blocking goroutines to prevent deadlocks
- Updated all interactive input functions to use sanitization for security
- Improved rsync security comment to explain why strict host key checking should never be disabled
- Reorganized legacy bash scripts from lan/ to legacy/lan/ directory for historical preservation
- Updated SFTP path operations to use filepath package for correct cross-platform behavior
- Enhanced transfer path validation with additional security checks

### Removed

- Removed dangerous StrictHostKeyChecking comment from rsync implementation (replaced with security warning)
- Removed duplicate installation scripts from root directory (config.sh, install.sh, uninstall.sh)
- Removed legacy lan/ directory (moved contents to legacy/lan/)

### Documentation

- Created legacy/README.md explaining historical context and migration path
- Updated validation documentation with new security-focused functions
- Added comprehensive comments explaining security rationale for key changes

### Internal

- Created internal/cli/connection.go for shared connection setup logic
- Created internal/logger/audit.go for security audit logging
- Integrated ConnectionHelper into cmd/klipc/main.go and cmd/klipr/main.go
- Integrated AuditLogger into cmd/klipc/main.go and cmd/klipr/main.go for transfer event tracking
- Enhanced internal/config/validation.go with comprehensive validation functions
- Enhanced internal/transfer/validation.go with ValidateExcludePattern and improved cleanup
- Enhanced internal/ui/prompts.go with sanitization for all user inputs
- Updated internal/ssh/keys.go to use github.com/pkg/sftp for secure key deployment
- Improved error handling and logging throughout validation functions
- Added NewConnectionHelper to centralize profile selection and backend detection
- Added CreateSSHClient to standardize SSH client creation across all commands
- Fixed unused imports and type conversions in internal/cli/connection.go and internal/transfer/sftp.go

## [2.1.0] - 2025-11-08

### Security

- Implemented proper SSH host key verification system to replace InsecureIgnoreHostKey
- Added known_hosts file management with XDG Base Directory compliance
- Added interactive host key acceptance prompts with fingerprint display
- Implemented path traversal protection for file transfers
- Added comprehensive path validation and sanitization for source and destination paths

### Added

- Created structured logging system using log/slog standard library
- Added common CLI flags package to eliminate code duplication across commands
- Added shell completion generation script for Bash, Zsh, Fish, and PowerShell
- Added CONTRIBUTING.md with comprehensive contribution guidelines
- Added .editorconfig for consistent code formatting across editors
- Added .gitignore file to protect repository from accidental commits
- Implemented parallel backend detection for faster startup
- Added GetLogFilePath function for XDG-compliant log file locations
- Added comprehensive path validation functions (ValidatePath, SanitizePath, IsPathSafe)
- Added path validation for transfer operations (ValidateTransferPaths)
- Added host key management functions (AddKnownHost, RemoveKnownHost, VerifyHostKey)
- Added FormatFingerprint function for human-readable SSH key fingerprints
- Added shell completion installation instructions to README

### Fixed

- Fixed PrintJSON function to use proper JSON encoding instead of fmt.Printf
- Fixed potential panic in normalizePath when processing paths shorter than 2 characters
- Corrected README dependency list to remove non-existent gokrazy/rsync
- Added external tools documentation (rsync, ssh) as system requirements
- Fixed stderr handling in rsync transfer to properly capture error messages

### Changed

- Updated SSH client to use NewHostKeyCallback for secure host key verification
- Enhanced backend detector to check backends in parallel using goroutines
- Improved transfer path validation with security-focused checks
- Refactored normalizePath to include bounds checking before string slicing
- Updated README with accurate dependency information and versions
- Enhanced README Contributing section with link to CONTRIBUTING.md
- Improved project documentation structure

### Documentation

- Added comprehensive CONTRIBUTING.md covering development workflow and code style
- Added .editorconfig for editor configuration consistency
- Updated README with shell completion installation instructions
- Corrected dependency attribution to reflect actual go.mod contents
- Added external tools section documenting rsync and SSH requirements
- Created detailed audit documentation in .dev-docs/01-Initial_Audit.md
- Created remediation blueprint in .dev-docs/02-Remediation_Blueprint.md

### Internal

- Created internal/logger package for structured logging
- Created internal/cli package for common flag definitions
- Created internal/ssh/hostkeys.go for host key management
- Created internal/transfer/validation.go for path security
- Added comprehensive test coverage for logger package
- Improved error messages throughout codebase

## [2.0.0] - 2025-01-08

### Added

#### Core Features
- Modern Go-based remote connection and file transfer tool
- Support for multiple VPN backends: LAN, Tailscale, Headscale, and NetBird
- Automatic VPN backend detection with intelligent priority-based selection
- Profile-based configuration system for managing multiple remote connections
- Interactive profile selection and creation interface
- XDG Base Directory Specification compliance for configuration storage
- Structured YAML configuration format with validation

#### Connection Features
- SSH client with multiple authentication methods (key-based, password, keyboard-interactive)
- Automatic SSH key detection from standard locations (~/.ssh/)
- SSH key generation and deployment utilities
- Connection health check system with timeout support
- Context-based operation cancellation and timeout enforcement
- Verbose logging mode for troubleshooting

#### File Transfer Features
- Dual transfer methods: rsync (fast) and SFTP (reliable)
- Real-time progress tracking with transfer speed and ETA
- Transfer resume support using rsync partial transfer capability
- Configurable compression levels (0-9) for rsync transfers
- Bandwidth limiting support for controlled transfer speeds
- File exclusion patterns for selective transfers
- Dry-run mode for previewing transfers without execution
- Batch file transfer support for multiple sources
- Permission preservation during transfers

#### VPN Backend Support
- LAN backend with automatic network interface detection
- Tailscale integration with status checking and peer IP resolution
- Headscale integration for self-hosted Tailscale alternative
- NetBird integration with mesh VPN support
- Backend priority system for optimal connection selection
- Status display showing all VPN connection states

#### User Interface
- Terminal color output with success/error/warning indicators
- Interactive prompts with input validation
- Formatted table output for status displays
- Progress bars for file transfers
- Clear error messages with actionable guidance

#### Command-Line Interface
- Three main commands: klip (connect), klipc (copy to remote), klipr (retrieve from remote)
- Profile management subcommands (list, add, remove, set-current)
- Backend status and health check commands
- Version information display with build metadata
- Shell completion generation (bash, zsh, fish, powershell)
- Flexible command-line flags for all operations

#### Build and Installation
- Makefile with build, install, test, and clean targets
- Cross-platform binary compilation support
- Version information injection at build time
- Automated installation and uninstallation scripts
- Dependency management with Go modules

#### Documentation
- Comprehensive README with quick start guide and examples
- Technical documentation covering architecture and internals
- Troubleshooting guide with common issues and solutions
- FOSS dependency attribution with license information
- Migration guide from legacy systems

#### Testing
- Unit tests for configuration management
- Backend detection and selection tests
- Integration tests for end-to-end workflows
- Mock backend implementations for isolated testing
- Test coverage for validation logic

### Technical Details

#### Dependencies (All FOSS)
- github.com/spf13/cobra v1.8.1 - CLI framework (Apache-2.0)
- github.com/spf13/viper v1.19.0 - Configuration management (MIT)
- github.com/adrg/xdg v0.5.3 - XDG Base Directory support (MIT)
- golang.org/x/crypto/ssh - SSH client implementation (BSD-3-Clause)
- github.com/pkg/sftp v1.13.7 - SFTP protocol (BSD-2-Clause)
- github.com/gokrazy/rsync v0.1.0 - Rsync implementation (BSD-3-Clause)
- github.com/fatih/color v1.18.0 - Terminal colors (MIT)
- github.com/schollz/progressbar/v3 v3.17.1 - Progress indicators (MIT)
- github.com/stretchr/testify v1.10.0 - Testing framework (MIT)

#### Architecture
- Modular backend abstraction layer with plugin-like design
- Configuration layer with automatic migration support
- SSH layer with connection pooling and session management
- Transfer layer with method abstraction (rsync/SFTP)
- UI layer with formatted output and interactive prompts
- Comprehensive error handling with typed errors
- Context support throughout for cancellation and timeouts

#### Security
- SSH key-based authentication as default
- Private keys stored with restricted permissions (0600)
- No plaintext password storage in configuration
- Path validation to prevent traversal attacks
- Input sanitization for all user-provided values
- Secure credential handling with passphrase prompting

### Notes

This is the initial release of klip, a complete rewrite and modernization of remote connection tooling. The project is designed to provide a robust, secure, and user-friendly experience for SSH connections and file transfers across multiple VPN backends.

klip is created by orpheus497 and is released under the MIT License.
