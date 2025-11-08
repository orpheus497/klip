# klip - Remote Connection Tool with Multi-VPN Support

**klip** is a modern, production-ready remote connection and file transfer tool built in Go. It simplifies SSH access and file synchronization across multiple VPN backends including LAN, Tailscale, Headscale, and NetBird.

**Created by orpheus497**

## Features

- **Multi-VPN Backend Support**: Seamlessly connect via LAN, Tailscale, Headscale, or NetBird
- **Automatic Backend Detection**: Intelligently selects the best available VPN backend
- **Profile-Based Configuration**: Manage multiple remote connections with named profiles
- **Interactive Mode**: User-friendly interactive prompts for profile selection
- **Dual Transfer Methods**: Choose between rsync (fast) or SFTP (reliable)
- **Progress Tracking**: Real-time progress indicators for file transfers
- **Resume Support**: Partial transfer support for interrupted operations
- **Health Checks**: Verify backend connectivity and SSH accessibility
- **Configuration Migration**: Automatic migration from legacy LINK bash scripts
- **Cross-Platform**: Native binaries for Linux, macOS, Windows, and more

## Quick Start

### Installation

#### From Source

```bash
# Navigate to the klip directory
cd klip

# Build the binaries
make build

# Install to /usr/local/bin (requires sudo)
sudo make install

# Or use the installation script
chmod +x scripts/install.sh
./scripts/install.sh
```

#### Pre-built Binaries

Download pre-built binaries for your platform from the releases page.

#### Shell Completion (Optional)

Enable command-line completion for easier usage:

```bash
# Build first
make build

# Generate completions
./scripts/generate-completions.sh

# Bash
sudo cp scripts/completion/klip.bash /etc/bash_completion.d/klip

# Zsh
mkdir -p ~/.zsh/completion
cp scripts/completion/_klip ~/.zsh/completion/
# Add to ~/.zshrc: fpath=(~/.zsh/completion $fpath)

# Fish
cp scripts/completion/klip.fish ~/.config/fish/completions/

# PowerShell
# Add to profile: . /path/to/scripts/completion/klip.ps1
```

### Initial Setup

```bash
# Initialize configuration
klip init

# Create your first profile
# Follow the interactive prompts
```

### Basic Usage

#### Connect via SSH

```bash
# Connect using current profile
klip

# Connect using specific profile
klip myserver

# Connect with backend override
klip --backend tailscale myserver
```

#### Copy Files TO Remote

```bash
# Copy to remote (preserves path structure)
klipc ~/Documents/project.txt

# Copy to specific destination
klipc ~/Documents/project.txt /remote/path/

# Copy with specific profile
klipc --profile workserver ~/project/
```

#### Retrieve Files FROM Remote

```bash
# Retrieve to current directory
klipr ~/remote/file.txt

# Retrieve to specific destination
klipr ~/remote/project/ ~/local/backup/

# Dry run (preview without transferring)
klipr --dry-run ~/remote/data/
```

## Commands

### klip - SSH Connection

Connect to remote machines via SSH.

**Usage:**
```bash
klip [profile] [flags]
```

**Flags:**
- `-p, --profile <name>`: Specify connection profile
- `-b, --backend <backend>`: Override VPN backend (auto, lan, tailscale, headscale, netbird)
- `-v, --verbose`: Enable verbose output
- `-t, --timeout <seconds>`: Connection timeout (default: 30)

**Subcommands:**
- `klip profile list`: List all profiles
- `klip profile add`: Add new profile
- `klip profile remove <name>`: Remove profile
- `klip profile set-current <name>`: Set default profile
- `klip status`: Show VPN backend status
- `klip health`: Perform health checks
- `klip version`: Show version information
- `klip init`: Initialize configuration

### klipc - Copy to Remote

Transfer files from local to remote machines.

**Usage:**
```bash
klipc <source> [destination] [flags]
```

**Flags:**
- `-p, --profile <name>`: Connection profile
- `-d, --dest <path>`: Destination path on remote
- `-m, --method <method>`: Transfer method (rsync, sftp)
- `-z, --compress <level>`: Compression level 0-9 (default: 6)
- `--dry-run`: Preview without transferring
- `-v, --verbose`: Verbose output

### klipr - Retrieve from Remote

Transfer files from remote to local machines.

**Usage:**
```bash
klipr <remote-source> [local-destination] [flags]
```

**Flags:**
- Same as `klipc`

## Configuration

Configuration is stored in XDG-compliant locations:
- Linux: `~/.config/klip/config.yaml`
- macOS: `~/Library/Application Support/klip/config.yaml`
- Windows: `%APPDATA%\klip\config.yaml`

### Example Configuration

```yaml
current_profile: myserver

profiles:
  myserver:
    name: myserver
    description: My development server
    backend: auto
    remote_user: username
    remote_host: server.example.com
    ssh_port: 22
    ssh_key_path: ~/.ssh/id_ed25519
    transfer_options:
      method: rsync
      compression_level: 6
      preserve_permissions: true

  tailscale-server:
    name: tailscale-server
    description: Server via Tailscale
    backend: tailscale
    remote_user: admin
    remote_host: myserver
    ssh_port: 22

settings:
  verbose: false
  default_backend: auto
  ssh_timeout: 30
  transfer_method: rsync
  compression_level: 6
  show_progress: true
```

## VPN Backend Support

### LAN (Direct)
Direct IP/hostname connections without VPN.

**Requirements:** Network connectivity

### Tailscale
Official Tailscale VPN service.

**Requirements:**
- Tailscale installed and running
- `tailscale` command in PATH

**Installation:**
```bash
# See: https://tailscale.com/download
```

### Headscale
Self-hosted Tailscale control server.

**Requirements:**
- Tailscale client installed
- Connected to Headscale server
- `tailscale` command in PATH

**Installation:**
```bash
# Client: https://tailscale.com/download
# Server: https://headscale.net/
```

### NetBird
Open-source WireGuard-based mesh VPN.

**Requirements:**
- NetBird installed and running
- `netbird` command in PATH

**Installation:**
```bash
# See: https://netbird.io/docs/getting-started/installation
```

## Migration from Legacy LINK

If you have existing LINK bash scripts, klip can automatically migrate your configuration:

```bash
klip init
# Choose "yes" when prompted to migrate
```

Your old configuration at `~/.LINK/config.sh` will be imported as profiles.

## Documentation

- [Technical Documentation](DOCUMENTATION.md) - Comprehensive technical reference
- [Changelog](CHANGELOG.md) - Version history and release notes

## Dependencies

All dependencies are FOSS with permissive licenses:

- [github.com/spf13/cobra](https://github.com/spf13/cobra) - CLI framework (Apache-2.0)
- [github.com/adrg/xdg](https://github.com/adrg/xdg) - XDG support (MIT)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto) - SSH client (BSD-3-Clause)
- [golang.org/x/term](https://pkg.go.dev/golang.org/x/term) - Terminal operations (BSD-3-Clause)
- [github.com/pkg/sftp](https://github.com/pkg/sftp) - SFTP (BSD-2-Clause)
- [github.com/fatih/color](https://github.com/fatih/color) - Terminal colors (MIT)
- [github.com/schollz/progressbar/v3](https://github.com/schollz/progressbar) - Progress bars (MIT)
- [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml) - YAML parsing (MIT)

**External Tools Required:**
- `rsync` - Fast file transfer program (GPL-3.0, external binary)
- `ssh` - SSH client for connections (BSD, system package)

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](CONTRIBUTING.md) for details on our code of conduct, development process, and how to submit pull requests.

## License

MIT License - Copyright (c) 2025 orpheus497

See [LICENSE.md](LICENSE.md) for details.

## Acknowledgments

This project makes use of excellent FOSS libraries from the Go community. Special thanks to all contributors and maintainers of the dependencies listed above.

**Created by orpheus497**
