# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
