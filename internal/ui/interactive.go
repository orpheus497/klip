package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/orpheus497/klip/internal/config"
)

// ProfileSelector provides interactive profile selection
type ProfileSelector struct {
	config *config.Config
}

// NewProfileSelector creates a new profile selector
func NewProfileSelector(cfg *config.Config) *ProfileSelector {
	return &ProfileSelector{
		config: cfg,
	}
}

// SelectProfile interactively selects a profile
func (ps *ProfileSelector) SelectProfile() (*config.Profile, string, error) {
	profiles := ps.config.ListProfiles()

	if len(profiles) == 0 {
		return nil, "", fmt.Errorf("no profiles configured")
	}

	if len(profiles) == 1 {
		// Only one profile, use it automatically
		profileName := profiles[0]
		profile, err := ps.config.GetProfile(profileName)
		return profile, profileName, err
	}

	PrintHeader("Select Connection Profile")
	PrintEmptyLine()

	// Display profiles
	for i, name := range profiles {
		profile, err := ps.config.GetProfile(name)
		if err != nil {
			continue
		}

		isCurrent := (name == ps.config.CurrentProfile)
		marker := " "
		if isCurrent {
			marker = Success("●")
		}

		fmt.Printf("  %s %d. %s\n", marker, i+1, Bold(name))
		fmt.Printf("      %s@%s (%s)\n", profile.RemoteUser, profile.RemoteHost, profile.Backend)

		if profile.Description != "" {
			fmt.Printf("      %s\n", Dim(profile.Description))
		}
		PrintEmptyLine()
	}

	// Prompt for selection
	fmt.Print(Info("Select profile number (or press Enter for current): "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}

	input = strings.TrimSpace(input)

	// If empty, use current profile
	if input == "" {
		if ps.config.CurrentProfile == "" {
			return nil, "", fmt.Errorf("no current profile set")
		}
		profile, err := ps.config.GetCurrentProfile()
		return profile, ps.config.CurrentProfile, err
	}

	// Parse selection
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(profiles) {
		return nil, "", fmt.Errorf("invalid selection")
	}

	profileName := profiles[selection-1]
	profile, err := ps.config.GetProfile(profileName)
	if err != nil {
		return nil, "", err
	}

	return profile, profileName, nil
}

// CreateProfileInteractive creates a profile interactively
func CreateProfileInteractive() (*config.Profile, string, error) {
	PrintHeader("Create New Profile")
	PrintEmptyLine()

	reader := bufio.NewReader(os.Stdin)

	// Profile name
	fmt.Print(Bold("Profile name: "))
	name, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	name = strings.TrimSpace(name)

	if name == "" {
		return nil, "", fmt.Errorf("profile name cannot be empty")
	}

	// Remote user
	fmt.Print(Bold("Remote username: "))
	user, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	user = strings.TrimSpace(user)

	if user == "" {
		return nil, "", fmt.Errorf("username cannot be empty")
	}

	// Remote host
	fmt.Print(Bold("Remote hostname or IP: "))
	host, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	host = strings.TrimSpace(host)

	if host == "" {
		return nil, "", fmt.Errorf("hostname cannot be empty")
	}

	// Create profile
	profile := config.NewProfile(name, user, host)

	// Backend selection
	PrintEmptyLine()
	PrintInfo("Select VPN backend:")
	backends := []string{"auto", "lan", "tailscale", "headscale", "netbird"}
	PrintNumberedList(backends)

	fmt.Print(Bold("Backend [1-5] (default: 1): "))
	backendInput, _ := reader.ReadString('\n')
	backendInput = strings.TrimSpace(backendInput)

	if backendInput == "" {
		backendInput = "1"
	}

	backendIdx, err := strconv.Atoi(backendInput)
	if err != nil || backendIdx < 1 || backendIdx > len(backends) {
		PrintWarning("Invalid backend, using 'auto'")
		backendIdx = 1
	}

	profile.Backend = config.BackendType(backends[backendIdx-1])

	// SSH port (optional)
	PrintEmptyLine()
	fmt.Print(Bold("SSH port (default: 22): "))
	portInput, _ := reader.ReadString('\n')
	portInput = strings.TrimSpace(portInput)

	if portInput != "" {
		port, err := strconv.Atoi(portInput)
		if err != nil || port < 1 || port > 65535 {
			PrintWarning("Invalid port, using default (22)")
		} else {
			profile.SSHPort = port
		}
	}

	// SSH key path (optional)
	PrintEmptyLine()
	fmt.Print(Bold("SSH key path (optional, press Enter to skip): "))
	keyPath, _ := reader.ReadString('\n')
	keyPath = strings.TrimSpace(keyPath)

	if keyPath != "" {
		profile.SSHKeyPath = keyPath
	}

	// Description (optional)
	PrintEmptyLine()
	fmt.Print(Bold("Description (optional): "))
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)

	if desc != "" {
		profile.Description = desc
	}

	return profile, name, nil
}

// EditProfileInteractive edits a profile interactively
func EditProfileInteractive(profile *config.Profile) error {
	PrintHeader(fmt.Sprintf("Edit Profile: %s", profile.Name))
	PrintEmptyLine()

	reader := bufio.NewReader(os.Stdin)

	// Show current values and allow editing
	fmt.Printf("Remote user [%s]: ", profile.RemoteUser)
	user, _ := reader.ReadString('\n')
	user = strings.TrimSpace(user)
	if user != "" {
		profile.RemoteUser = user
	}

	fmt.Printf("Remote host [%s]: ", profile.RemoteHost)
	host, _ := reader.ReadString('\n')
	host = strings.TrimSpace(host)
	if host != "" {
		profile.RemoteHost = host
	}

	fmt.Printf("Backend [%s]: ", profile.Backend)
	backend, _ := reader.ReadString('\n')
	backend = strings.TrimSpace(backend)
	if backend != "" {
		profile.Backend = config.BackendType(backend)
	}

	fmt.Printf("SSH port [%d]: ", profile.SSHPort)
	portInput, _ := reader.ReadString('\n')
	portInput = strings.TrimSpace(portInput)
	if portInput != "" {
		port, err := strconv.Atoi(portInput)
		if err == nil && port > 0 && port <= 65535 {
			profile.SSHPort = port
		}
	}

	fmt.Printf("SSH key path [%s]: ", profile.SSHKeyPath)
	keyPath, _ := reader.ReadString('\n')
	keyPath = strings.TrimSpace(keyPath)
	if keyPath != "" {
		profile.SSHKeyPath = keyPath
	}

	fmt.Printf("Description [%s]: ", profile.Description)
	desc, _ := reader.ReadString('\n')
	desc = strings.TrimSpace(desc)
	if desc != "" {
		profile.Description = desc
	}

	return nil
}

// SelectBackend interactively selects a backend
func SelectBackend() (string, error) {
	PrintInfo("Select VPN backend:")

	backends := []string{"auto", "lan", "tailscale", "headscale", "netbird"}
	PrintNumberedList(backends)

	fmt.Print(Bold("Backend [1-5]: "))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSpace(input)
	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(backends) {
		return "", fmt.Errorf("invalid selection")
	}

	return backends[selection-1], nil
}
