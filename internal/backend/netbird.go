package backend

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// NetBirdBackend implements NetBird VPN backend
type NetBirdBackend struct{}

// Name returns the backend name
func (b *NetBirdBackend) Name() string {
	return "netbird"
}

// IsAvailable checks if NetBird is installed
func (b *NetBirdBackend) IsAvailable(ctx context.Context) bool {
	_, err := exec.LookPath("netbird")
	return err == nil
}

// IsConnected checks if NetBird is connected
func (b *NetBirdBackend) IsConnected(ctx context.Context) bool {
	if !b.IsAvailable(ctx) {
		return false
	}

	cmd := exec.CommandContext(ctx, "netbird", "status")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse status output to check if connected
	status := parseNetBirdStatus(string(output))
	return status.connected
}

// GetStatus returns NetBird status
func (b *NetBirdBackend) GetStatus(ctx context.Context) (*Status, error) {
	if !b.IsAvailable(ctx) {
		return nil, ErrNotAvailable
	}

	status := &Status{
		Backend:   b.Name(),
		LastCheck: time.Now(),
		Peers:     []PeerInfo{},
	}

	cmd := exec.CommandContext(ctx, "netbird", "status")
	output, err := cmd.Output()
	if err != nil {
		status.Connected = false
		status.Message = "Failed to get status"
		return status, ErrCommandFailed
	}

	nbStatus := parseNetBirdStatus(string(output))
	status.Connected = nbStatus.connected
	status.Message = nbStatus.state
	status.LocalIP = nbStatus.localIP

	// Get peer list if connected
	if status.Connected {
		peers, err := b.getPeerList(ctx)
		if err == nil {
			status.Peers = peers
		}
	}

	return status, nil
}

// GetPeerIP resolves a NetBird peer hostname to IP
func (b *NetBirdBackend) GetPeerIP(ctx context.Context, hostname string) (string, error) {
	if !b.IsConnected(ctx) {
		return "", ErrNotConnected
	}

	// Get all peers
	peers, err := b.getPeerList(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get peer list: %w", err)
	}

	// Search for peer by hostname
	for _, peer := range peers {
		if strings.EqualFold(peer.Hostname, hostname) {
			if peer.IP != "" {
				return peer.IP, nil
			}
		}
	}

	return "", ErrPeerNotFound
}

// Priority returns the priority for auto-detection (high priority)
func (b *NetBirdBackend) Priority() int {
	return 50
}

// getPeerList retrieves the list of NetBird peers
func (b *NetBirdBackend) getPeerList(ctx context.Context) ([]PeerInfo, error) {
	// NetBird doesn't have a built-in peer list command in older versions
	// Try using 'netbird status' verbose output or 'netbird up' output
	cmd := exec.CommandContext(ctx, "netbird", "status", "-d")
	output, err := cmd.Output()
	if err != nil {
		// If verbose status fails, return empty list
		return []PeerInfo{}, nil
	}

	return parseNetBirdPeers(string(output)), nil
}

// netBirdStatusInfo contains parsed NetBird status information
type netBirdStatusInfo struct {
	connected bool
	state     string
	localIP   string
}

// parseNetBirdStatus parses the output of 'netbird status'
func parseNetBirdStatus(output string) netBirdStatusInfo {
	info := netBirdStatusInfo{}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Check for connection status
		if strings.Contains(strings.ToLower(line), "status:") {
			if strings.Contains(strings.ToLower(line), "connected") {
				info.connected = true
				info.state = "Connected"
			} else if strings.Contains(strings.ToLower(line), "disconnected") {
				info.connected = false
				info.state = "Disconnected"
			} else {
				// Extract state after "Status:"
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					info.state = strings.TrimSpace(parts[1])
					info.connected = strings.Contains(strings.ToLower(info.state), "connected")
				}
			}
		}

		// Extract local IP
		if strings.Contains(strings.ToLower(line), "netbird ip:") ||
			strings.Contains(strings.ToLower(line), "local ip:") ||
			strings.Contains(strings.ToLower(line), "interface ip:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.localIP = strings.TrimSpace(parts[1])
			}
		}

		// Alternative format: "Management: Connected"
		if strings.Contains(strings.ToLower(line), "management:") {
			if strings.Contains(strings.ToLower(line), "connected") {
				info.connected = true
				if info.state == "" {
					info.state = "Connected"
				}
			}
		}
	}

	// Default state if not found
	if info.state == "" {
		if info.connected {
			info.state = "Connected"
		} else {
			info.state = "Disconnected"
		}
	}

	return info
}

// parseNetBirdPeers parses peer information from NetBird status output
func parseNetBirdPeers(output string) []PeerInfo {
	var peers []PeerInfo

	// Regex patterns for peer information
	peerRegex := regexp.MustCompile(`(?i)peer\s+([^\s:]+)[:\s]+([0-9\.]+)`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Try to match peer lines
		matches := peerRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			peer := PeerInfo{
				Hostname: matches[1],
				IP:       matches[2],
				Online:   true, // Assume online if listed
			}
			peers = append(peers, peer)
		}
	}

	return peers
}
