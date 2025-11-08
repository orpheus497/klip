# Legacy LINK Scripts

**Created by orpheus497**

---

These are the original bash scripts that klip was built to modernize and replace.
They are preserved here for historical reference only.

## Historical Context

Before klip, remote connections were managed through simple bash scripts stored in this directory.
The klip project represents a complete modernization of this approach, providing:

- Type safety and error handling through Go
- Multi-VPN backend support with automatic detection
- Structured configuration management
- Security improvements (proper host key verification, SSH key management)
- Cross-platform binary distribution
- Professional CLI with progress tracking and interactive prompts

## Migration

If you're still using these legacy scripts, please migrate to klip:

```bash
klip init  # This will offer to migrate your configuration automatically
```

## Legacy Scripts

### Original LINK Suite

- **LINK.sh** - SSH connection script for LAN connectivity
  - Simple SSH wrapper for direct LAN connections
  - Used configuration from ~/.LINK/config.sh

- **LINKc.sh** - File copy/push script (local → remote)
  - Basic rsync wrapper for pushing files to remote
  - Limited error handling and validation

- **LINKr.sh** - File retrieve/pull script (remote → local)
  - Basic rsync wrapper for pulling files from remote
  - Limited error handling and validation

### Why These Scripts Were Replaced

**Security Concerns:**
- No host key verification
- Credentials in plain text configuration files
- No input validation or sanitization
- Direct shell command construction with user input

**Functionality Limitations:**
- Single backend support (LAN only)
- No progress tracking
- No resume support for interrupted transfers
- No profile management
- Limited error messages
- Manual configuration editing required

**Maintainability Issues:**
- Bash-specific (not cross-platform)
- No dependency management
- Difficult to test
- Hard to extend with new features

## Replacement Features in klip

**klip** provides all the functionality of the legacy scripts plus:

| Legacy Feature | klip Enhancement |
|----------------|------------------|
| SSH connection | Multi-VPN backend support (LAN, Tailscale, Headscale, NetBird) |
| Basic rsync | Choice of rsync or SFTP with progress tracking |
| Manual config | YAML configuration with validation |
| Single profile | Multiple profiles with interactive selection |
| No error handling | Comprehensive error messages and health checks |
| Bash-only | Cross-platform Go binaries (Linux, macOS, Windows) |
| No security | Host key verification, secure key permissions |

## Do Not Use These Scripts

**These scripts are not maintained and should not be used for production work.**

They exist solely for:
- Historical reference
- Understanding the evolution of klip
- Migration verification (comparing old vs new behavior)

For all actual remote connection needs, use klip instead.

---

**Migration created this directory on:** November 8, 2025

**Original scripts created by:** orpheus497
