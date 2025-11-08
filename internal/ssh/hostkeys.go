// Package ssh - Host key verification and management
// Copyright (c) 2025 orpheus497
package ssh

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// GetKnownHostsPath returns the XDG-compliant path to the known_hosts file
func GetKnownHostsPath() (string, error) {
	configDir := filepath.Join(xdg.ConfigHome, "klip")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	return filepath.Join(configDir, "known_hosts"), nil
}

// LoadKnownHosts loads the known_hosts file and returns a host key callback
func LoadKnownHosts() (ssh.HostKeyCallback, error) {
	knownHostsPath, err := GetKnownHostsPath()
	if err != nil {
		return nil, err
	}

	// Create the file if it doesn't exist
	if _, err := os.Stat(knownHostsPath); os.IsNotExist(err) {
		file, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_RDONLY, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to create known_hosts file: %w", err)
		}
		file.Close()
	}

	// Load known hosts
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load known_hosts: %w", err)
	}

	return callback, nil
}

// NewHostKeyCallback creates a host key callback with interactive verification
func NewHostKeyCallback() ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Try to load known hosts
		knownHostsCallback, err := LoadKnownHosts()
		if err != nil {
			// If we can't load known hosts, fail securely
			return fmt.Errorf("failed to load known hosts: %w", err)
		}

		// Check against known hosts
		err = knownHostsCallback(hostname, remote, key)
		if err == nil {
			// Host key is known and matches
			return nil
		}

		// Check if this is a key mismatch (potential MITM) or unknown host
		if knownHostsErr, ok := err.(*knownhosts.KeyError); ok {
			if len(knownHostsErr.Want) > 0 {
				// Host key has changed - this is dangerous!
				return fmt.Errorf("WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!\n"+
					"IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!\n"+
					"Someone could be eavesdropping on you right now (man-in-the-middle attack)!\n"+
					"The fingerprint for the %s key sent by the remote host is\n%s\n"+
					"Please contact your system administrator or remove the old key from known_hosts.",
					key.Type(), FormatFingerprint(key))
			}

			// Unknown host - ask user
			fmt.Printf("\n")
			fmt.Printf("The authenticity of host '%s (%s)' can't be established.\n", hostname, remote)
			fmt.Printf("%s key fingerprint is %s\n", key.Type(), FormatFingerprint(key))
			fmt.Printf("Are you sure you want to continue connecting (yes/no)? ")

			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			response = strings.ToLower(strings.TrimSpace(response))
			if response != "yes" {
				return fmt.Errorf("host key verification failed: user rejected")
			}

			// User accepted, add to known hosts
			if err := AddKnownHost(hostname, key); err != nil {
				return fmt.Errorf("failed to add host to known_hosts: %w", err)
			}

			fmt.Printf("Warning: Permanently added '%s' (%s) to the list of known hosts.\n", hostname, key.Type())
			return nil
		}

		// Other error
		return fmt.Errorf("host key verification failed: %w", err)
	}
}

// AddKnownHost adds a host and its public key to the known_hosts file
func AddKnownHost(hostname string, key ssh.PublicKey) error {
	knownHostsPath, err := GetKnownHostsPath()
	if err != nil {
		return err
	}

	// Open file for appending
	file, err := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to open known_hosts for writing: %w", err)
	}
	defer file.Close()

	// Format the line
	line := knownhosts.Line([]string{hostname}, key)

	// Write to file
	if _, err := file.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to write to known_hosts: %w", err)
	}

	return nil
}

// FormatFingerprint returns a human-readable fingerprint of the public key
// Returns both SHA256 and MD5 formats
func FormatFingerprint(key ssh.PublicKey) string {
	// SHA256 fingerprint (modern standard)
	sha256sum := sha256.Sum256(key.Marshal())
	sha256hex := base64.RawStdEncoding.EncodeToString(sha256sum[:])

	// MD5 fingerprint (legacy, but still commonly shown)
	md5sum := md5.Sum(key.Marshal())
	md5hex := fmt.Sprintf("%x", md5sum)
	md5formatted := ""
	for i := 0; i < len(md5hex); i += 2 {
		if i > 0 {
			md5formatted += ":"
		}
		md5formatted += md5hex[i : i+2]
	}

	return fmt.Sprintf("SHA256:%s (MD5:%s)", sha256hex, md5formatted)
}

// GetKeyFingerprint returns the SSH fingerprint of a key file
func GetKeyFingerprint(keyPath string) (string, error) {
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read key file: %w", err)
	}

	key, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	return FormatFingerprint(key.PublicKey()), nil
}

// VerifyHostKey verifies a host key against known_hosts without connecting
func VerifyHostKey(hostname string, key ssh.PublicKey) error {
	callback, err := LoadKnownHosts()
	if err != nil {
		return err
	}

	// Use a dummy address since we're just checking the file
	addr := &net.TCPAddr{IP: net.IPv4(0, 0, 0, 0), Port: 22}

	return callback(hostname, addr, key)
}

// RemoveKnownHost removes all entries for a hostname from known_hosts
func RemoveKnownHost(hostname string) error {
	knownHostsPath, err := GetKnownHostsPath()
	if err != nil {
		return err
	}

	// Read all lines
	file, err := os.Open(knownHostsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to remove
		}
		return fmt.Errorf("failed to open known_hosts: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Skip lines containing the hostname
		if !strings.Contains(line, hostname) {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read known_hosts: %w", err)
	}

	// Write back filtered lines
	if err := os.WriteFile(knownHostsPath, []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write known_hosts: %w", err)
	}

	return nil
}
