# Makefile for klip
# Copyright (c) 2025 orpheus497

# Binary names
KLIP_BIN := klip
KLIPC_BIN := klipc
KLIPR_BIN := klipr

# Build information
VERSION := 2.2.0
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION := $(shell go version | awk '{print $$3}')

# Build flags
LDFLAGS := -ldflags "\
	-X 'github.com/orpheus497/klip/internal/version.Version=$(VERSION)' \
	-X 'github.com/orpheus497/klip/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/orpheus497/klip/internal/version.BuildDate=$(BUILD_DATE)' \
	-X 'github.com/orpheus497/klip/internal/version.GoVersion=$(GO_VERSION)' \
	"

# Installation directories
PREFIX ?= /usr/local
BINDIR := $(PREFIX)/bin

# Build directory
BUILD_DIR := build

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

.PHONY: all build clean install uninstall test fmt vet deps help

## all: Build all binaries
all: build

## build: Build all binaries
build: build-klip build-klipc build-klipr

## build-klip: Build klip binary
build-klip:
	@echo "Building $(KLIP_BIN)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(KLIP_BIN) ./cmd/klip

## build-klipc: Build klipc binary
build-klipc:
	@echo "Building $(KLIPC_BIN)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(KLIPC_BIN) ./cmd/klipc

## build-klipr: Build klipr binary
build-klipr:
	@echo "Building $(KLIPR_BIN)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(KLIPR_BIN) ./cmd/klipr

## install: Install binaries to $(BINDIR)
install: build
	@echo "Installing to $(BINDIR)..."
	@install -d $(BINDIR)
	@install -m 755 $(BUILD_DIR)/$(KLIP_BIN) $(BINDIR)/$(KLIP_BIN)
	@install -m 755 $(BUILD_DIR)/$(KLIPC_BIN) $(BINDIR)/$(KLIPC_BIN)
	@install -m 755 $(BUILD_DIR)/$(KLIPR_BIN) $(BINDIR)/$(KLIPR_BIN)
	@echo "Installation complete!"
	@echo "Run 'klip init' to get started"

## uninstall: Remove installed binaries
uninstall:
	@echo "Uninstalling from $(BINDIR)..."
	@rm -f $(BINDIR)/$(KLIP_BIN)
	@rm -f $(BINDIR)/$(KLIPC_BIN)
	@rm -f $(BINDIR)/$(KLIPR_BIN)
	@echo "Uninstallation complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

## check: Run fmt, vet, and test
check: fmt vet test

## version: Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

## help: Show this help message
help:
	@echo "klip - Makefile commands:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

.DEFAULT_GOAL := help
