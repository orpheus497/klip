# Contributing to klip

**Created by orpheus497**

Thank you for your interest in contributing to klip! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Commit Messages](#commit-messages)
- [License](#license)

## Code of Conduct

Be respectful, professional, and constructive in all interactions. We aim to create a welcoming environment for all contributors.

## Getting Started

### Prerequisites

- Go 1.22 or later
- Git
- rsync (for rsync transfer method)
- SSH client
- Optional: golangci-lint for code quality checks

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR-USERNAME/klip.git
   cd klip
   ```
3. Add the upstream repository:
   ```bash
   git remote add upstream https://github.com/orpheus497/klip.git
   ```

## Development Setup

### Install Dependencies

```bash
# Download Go dependencies
go mod download

# Verify dependencies
go mod verify
```

### Build the Project

```bash
# Build all binaries
make build

# Build a specific binary
go build -o build/klip ./cmd/klip
```

### Run Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

## Making Changes

### Creating a Branch

```bash
# Update your main branch
git checkout main
git pull upstream main

# Create a feature branch
git checkout -b feature/your-feature-name
```

### Development Workflow

1. **Make your changes**
   - Write clean, well-documented code
   - Follow the existing code style
   - Add tests for new functionality
   - Update documentation as needed

2. **Test your changes**
   ```bash
   # Run tests
   make test

   # Run linter (if available)
   make lint

   # Build binaries
   make build

   # Manual testing
   ./build/klip --help
   ```

3. **Commit your changes**
   ```bash
   git add .
   git commit -m "Add feature: description"
   ```

## Testing

### Writing Tests

- Place test files next to the code they test (`file.go` → `file_test.go`)
- Use table-driven tests for multiple test cases
- Aim for at least 75% code coverage
- Test both success and failure cases
- Use meaningful test names that describe what is being tested

Example test structure:

```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "result", false},
        {"invalid input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FeatureFunc(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FeatureFunc() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("FeatureFunc() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Running Specific Tests

```bash
# Run tests for a specific package
go test ./internal/config

# Run a specific test function
go test -run TestProfileValidation ./internal/config

# Run with verbose output
go test -v ./...
```

## Submitting Changes

### Before Submitting

- [ ] All tests pass
- [ ] Code is formatted (`gofmt` or `make fmt`)
- [ ] Code is linted (if golangci-lint available)
- [ ] Documentation is updated
- [ ] CHANGELOG.md is updated (if applicable)
- [ ] Commit messages follow guidelines

### Creating a Pull Request

1. Push your branch to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Go to GitHub and create a Pull Request from your fork to the main repository

3. Fill out the PR template with:
   - Description of changes
   - Related issues (if any)
   - Testing performed
   - Screenshots (if applicable)

4. Wait for review and address any feedback

### PR Review Process

- Maintainers will review your PR
- Address any requested changes
- Once approved, your PR will be merged
- Your contribution will be credited in the CHANGELOG

## Code Style

### Go Code

Follow standard Go conventions:

- Use `gofmt` to format code
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use meaningful variable and function names
- Keep functions small and focused
- Add comments for exported functions and types
- Use error wrapping (`fmt.Errorf("context: %w", err)`)

### Package Documentation

Every package should have a package comment:

```go
// Package config provides configuration management for klip.
// It supports XDG Base Directory compliance and profile-based configurations.
package config
```

### Function Documentation

Export functions should have documentation comments:

```go
// ValidatePath validates a file path for security issues.
// It checks for null bytes, path traversal attempts, and other unsafe patterns.
// Returns an error if the path is invalid or potentially unsafe.
func ValidatePath(path string) error {
    // ...
}
```

### Error Handling

- Always check errors
- Provide context when wrapping errors
- Use meaningful error messages
- Don't ignore errors (use `_ = err` if intentional)

```go
// Good
data, err := os.ReadFile(path)
if err != nil {
    return fmt.Errorf("failed to read config file: %w", err)
}

// Bad
data, _ := os.ReadFile(path)
```

## Commit Messages

### Format

```
<type>: <short summary>

<detailed description (optional)>

<footer (optional)>
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Adding or updating tests
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build process or auxiliary tool changes

### Examples

```
feat: add SSH host key verification system

Implements proper host key verification to replace InsecureIgnoreHostKey.
Adds known_hosts file management with XDG compliance.
Includes interactive host key acceptance prompts.

Fixes #123
```

```
fix: prevent panic in path normalization

Check path length before slicing to avoid index out of bounds error
when processing paths shorter than 2 characters.
```

## Project Structure

```
klip/
├── cmd/                  # Command binaries
│   ├── klip/            # Main SSH connection command
│   ├── klipc/           # Copy to remote command
│   └── klipr/           # Retrieve from remote command
├── internal/            # Internal packages
│   ├── backend/         # VPN backend implementations
│   ├── cli/             # Common CLI utilities
│   ├── config/          # Configuration management
│   ├── logger/          # Structured logging
│   ├── ssh/             # SSH client implementation
│   ├── transfer/        # File transfer methods
│   ├── ui/              # User interface components
│   └── version/         # Version information
├── scripts/             # Build and utility scripts
├── test/                # Integration tests
├── .dev-docs/           # AI-generated development documentation (gitignored)
├── Makefile             # Build automation
├── go.mod               # Go module definition
└── README.md            # Project documentation
```

## Adding New Features

### Backend Implementation

To add a new VPN backend:

1. Create `internal/backend/newbackend.go`
2. Implement the `Backend` interface
3. Register in `NewRegistry()` in `internal/backend/backend.go`
4. Add tests
5. Update README.md with backend information

### Transfer Method

To add a new transfer method:

1. Create `internal/transfer/newmethod.go`
2. Implement the `Transfer` interface
3. Add case in `NewTransfer()` in `internal/transfer/transfer.go`
4. Add tests
5. Update documentation

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions in existing issues
- Ask questions in pull request comments

## License

By contributing to klip, you agree that your contributions will be licensed under the MIT License.

## Attribution

**klip is created and maintained by orpheus497.**

All contributions are credited in the CHANGELOG and remain under the MIT License.

---

Thank you for contributing to klip!
