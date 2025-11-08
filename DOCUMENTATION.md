# klip - Technical Documentation

**Created by orpheus497**

## Overview

klip is a modern Go-based remote connection and file transfer tool that provides seamless SSH access across multiple VPN backends. This document provides technical details for developers and advanced users.

## Architecture

### Core Components

#### 1. Configuration Layer (`internal/config/`)
- **config.go**: Configuration management with XDG Base Directory support
- **profile.go**: Profile data structures and validation
- **validation.go**: Comprehensive configuration validation
- **migration.go**: Automatic migration from legacy LINK bash scripts

#### 2. Backend Abstraction (`internal/backend/`)
- **backend.go**: Backend interface and registry
- **lan.go**: Direct IP/hostname connections
- **tailscale.go**: Tailscale VPN integration
- **headscale.go**: Headscale (self-hosted Tailscale) integration
- **netbird.go**: NetBird mesh VPN integration
- **detector.go**: Automatic backend detection and selection

#### 3. SSH Layer (`internal/ssh/`)
- **client.go**: SSH client with context support and reconnection
- **session.go**: SSH session lifecycle management
- **keys.go**: SSH key generation, validation, and deployment
- **health.go**: Connection health checks and diagnostics

#### 4. Transfer Layer (`internal/transfer/`)
- **transfer.go**: Transfer interface and common functionality
- **rsync.go**: Rsync-based file transfers with progress parsing
- **sftp.go**: SFTP-based transfers with resume support
- **progress.go**: Progress tracking and reporting

#### 5. User Interface (`internal/ui/`)
- **output.go**: Formatted, colored terminal output
- **interactive.go**: Interactive profile selection and creation
- **prompts.go**: User input prompts with validation

#### 6. Version (`internal/version/`)
- **version.go**: Version information and build metadata

### Command Binaries

- **cmd/klip**: Main SSH connection command
- **cmd/klipc**: File copy to remote command
- **cmd/klipr**: File retrieve from remote command

## Backend System

### Backend Interface

All backends implement the `Backend` interface:

```go
type Backend interface {
    Name() string
    IsAvailable(ctx context.Context) bool
    IsConnected(ctx context.Context) bool
    GetStatus(ctx context.Context) (*Status, error)
    GetPeerIP(ctx context.Context, hostname string) (string, error)
    Priority() int
}
```

### Backend Priority System

Backends have priority values for automatic selection:
- NetBird: 50 (highest)
- Tailscale: 40
- Headscale: 40
- LAN: 10 (lowest, fallback)

### Backend Detection Flow

1. Registry lists all available backends
2. Detector checks `IsAvailable()` for each
3. For available backends, checks `IsConnected()`
4. Selects first connected backend by priority
5. If no connected backend, returns highest priority available
6. Falls back to LAN if all else fails

## Configuration Format

### Profile Structure

```yaml
profiles:
  profile_name:
    name: string              # Profile name
    description: string       # Optional description
    backend: string           # auto|lan|tailscale|headscale|netbird
    remote_user: string       # SSH username
    remote_host: string       # Hostname or IP
    ssh_port: int             # SSH port (default: 22)
    ssh_key_path: string      # Path to SSH private key
    use_password: bool        # Use password auth instead of keys
    transfer_options:
      method: string          # rsync|sftp
      compression_level: int  # 0-9 (rsync only)
      exclude_patterns: []    # Patterns to exclude
      bandwidth_limit: int    # KB/s (0=unlimited)
      preserve_permissions: bool
      delete_after_transfer: bool
```

### Settings Structure

```yaml
settings:
  verbose: bool               # Enable verbose output
  default_backend: string     # Preferred backend
  ssh_timeout: int            # Seconds
  transfer_method: string     # rsync|sftp
  compression_level: int      # 0-9
  show_progress: bool         # Show progress bars
```

## Transfer System

### Transfer Methods

#### Rsync
- **Advantages**: Fast, efficient delta transfers, compression, exclusion patterns
- **Requirements**: `rsync` command available on both local and remote
- **Best for**: Large directories, frequent synchronization, bandwidth-limited connections

#### SFTP
- **Advantages**: Pure SSH protocol, no external dependencies, reliable
- **Requirements**: SSH server with SFTP subsystem
- **Best for**: Systems without rsync, simple file transfers, guaranteed compatibility

### Transfer Flow

1. **Connection Establishment**
   - Resolve backend
   - Resolve hostname to IP
   - Establish SSH connection (for SFTP)

2. **Transfer Configuration**
   - Determine source and destination paths
   - Apply compression settings
   - Configure exclusion patterns

3. **Execution**
   - For rsync: Build command arguments, execute with progress parsing
   - For SFTP: Walk directory tree, transfer files with progress callbacks

4. **Progress Reporting**
   - Real-time progress bars
   - Transfer speed calculation
   - ETA estimation

## SSH Connection Management

### Authentication Methods

Tried in order:
1. Specified SSH key (if `ssh_key_path` set)
2. Default SSH keys (`~/.ssh/id_ed25519`, `~/.ssh/id_rsa`, etc.)
3. Password authentication (if `use_password` is true)
4. Keyboard-interactive authentication

### Connection Lifecycle

```
NewClient() -> Connect() -> [Operations] -> Close()
```

### Context Support

All SSH operations support context cancellation:
- Timeout enforcement
- User cancellation (Ctrl+C)
- Graceful shutdown

## Error Handling

### Error Types

- `ErrNotAvailable`: Backend not installed
- `ErrNotConnected`: Backend not connected
- `ErrPeerNotFound`: Peer hostname not found
- `ErrCommandFailed`: Backend command failed
- `ErrTimeout`: Operation timed out

### Validation Errors

Structured validation errors with field-level details:

```go
type ValidationError struct {
    Field   string
    Message string
}
```

## Security Considerations

### SSH Key Management

- Private keys stored with 0600 permissions
- Public keys stored with 0644 permissions
- No plaintext password storage in configuration
- Support for encrypted SSH keys (passphrase prompted)

### Host Key Verification

**Current Implementation**: Uses `ssh.InsecureIgnoreHostKey()`

**TODO for Production**:
- Implement proper host key checking
- Store known hosts in `~/.config/klip/known_hosts`
- Prompt user on first connection (SSH-style)

### Path Validation

- Source paths validated before transfer
- Path traversal protection
- Tilde expansion (`~/`) handled securely

## Testing

### Unit Tests

```bash
make test
```

Tests cover:
- Configuration loading and validation
- Backend detection logic
- SSH client operations
- Transfer configuration

### Integration Tests

Located in `test/integration_test.go`:
- End-to-end connection flows
- File transfer operations
- Backend switching

## Build System

### Build Flags

Version information injected at build time:

```bash
-X 'github.com/orpheus497/klip/internal/version.Version=2.0.0'
-X 'github.com/orpheus497/klip/internal/version.GitCommit=abc123'
-X 'github.com/orpheus497/klip/internal/version.BuildDate=2025-01-08'
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build ./cmd/klip

# macOS ARM64
GOOS=darwin GOARCH=arm64 go build ./cmd/klip

# Windows AMD64
GOOS=windows GOARCH=amd64 go build ./cmd/klip
```

## Performance Considerations

### Backend Detection

- Cached for session duration
- Parallel availability checks possible
- Timeout protection (default: 10s)

### Transfer Optimization

- Rsync compression reduces bandwidth (level 6 default)
- Partial transfer support for resume
- Buffer size: 32KB for optimal throughput

### Memory Usage

- Streaming transfers (no full file buffering)
- Progress tracking minimal overhead
- Configuration lazy-loaded

## Troubleshooting

### Verbose Mode

Enable detailed logging:

```bash
klip --verbose myserver
klipc --verbose ~/file.txt
```

### Health Checks

Diagnose connectivity:

```bash
klip health     # Check all backends
klip status     # Backend status summary
```

### Configuration Validation

```bash
# View current configuration
cat ~/.config/klip/config.yaml

# Test profile
klip --profile myprofile --verbose
```

### Common Issues

**"Profile not found"**
- Run `klip profile list` to see available profiles
- Check configuration file exists
- Run `klip init` if no configuration

**"Backend not available"**
- Ensure VPN client installed (e.g., `which tailscale`)
- Check VPN service running (e.g., `tailscale status`)
- Try explicit backend: `--backend lan`

**"Connection timeout"**
- Increase timeout: `--timeout 60`
- Check network connectivity
- Verify remote host accessible
- Check SSH port correct

**"Transfer failed"**
- Verify source path exists
- Check destination writable
- Ensure sufficient disk space
- Try alternative method: `--method sftp`

## Development

### Project Structure

```
klip/
├── cmd/               # Command binaries
├── internal/          # Internal packages
│   ├── backend/       # VPN backend implementations
│   ├── config/        # Configuration management
│   ├── ssh/           # SSH client
│   ├── transfer/      # File transfer
│   ├── ui/            # User interface
│   └── version/       # Version info
├── test/              # Integration tests
├── docs/              # Additional documentation
├── Makefile           # Build automation
├── go.mod             # Go module definition
└── README.md          # User documentation
```

### Adding a New Backend

1. Create `internal/backend/newbackend.go`
2. Implement `Backend` interface
3. Register in `NewRegistry()` (backend.go)
4. Update README.md with backend information
5. Add tests

### Contributing

See [README.md](README.md#contributing) for contribution guidelines.

## License

MIT License - Copyright (c) 2025 orpheus497

## Credits

**Created by orpheus497**

This project builds upon excellent FOSS libraries:
- Cobra and Viper by Steve Francia (spf13)
- Go SSH library by the Go team
- SFTP library by pkg
- Progress bar library by schollz
- And many more (see go.mod)
