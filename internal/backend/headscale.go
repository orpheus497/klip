package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// HeadscaleBackend implements Headscale VPN backend
// Note: Headscale clients use the Tailscale client, but connect to a self-hosted control server
type HeadscaleBackend struct{}

// Name returns the backend name
func (b *HeadscaleBackend) Name() string {
	return "headscale"
}

// IsAvailable checks if Tailscale client is installed (used by Headscale)
func (b *HeadscaleBackend) IsAvailable(ctx context.Context) bool {
	_, err := exec.LookPath("tailscale")
	return err == nil
}

// IsConnected checks if Tailscale is connected to a Headscale server
func (b *HeadscaleBackend) IsConnected(ctx context.Context) bool {
	if !b.IsAvailable(ctx) {
		return false
	}

	cmd := exec.CommandContext(ctx, "tailscale", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	var status tailscaleStatus
	if err := json.Unmarshal(output, &status); err != nil {
		return false
	}

	// Check if backend state is Running
	// Note: We can't reliably distinguish between Tailscale and Headscale from the client
	// This backend is for explicit Headscale configurations
	return status.BackendState == "Running"
}

// GetStatus returns Headscale status
func (b *HeadscaleBackend) GetStatus(ctx context.Context) (*Status, error) {
	if !b.IsAvailable(ctx) {
		return nil, ErrNotAvailable
	}

	status := &Status{
		Backend:   b.Name(),
		LastCheck: time.Now(),
		Peers:     []PeerInfo{},
	}

	cmd := exec.CommandContext(ctx, "tailscale", "status", "--json")
	output, err := cmd.Output()
	if err != nil {
		status.Connected = false
		status.Message = "Failed to get status"
		return status, ErrCommandFailed
	}

	var tsStatus tailscaleStatus
	if err := json.Unmarshal(output, &tsStatus); err != nil {
		status.Connected = false
		status.Message = "Failed to parse status"
		return status, fmt.Errorf("failed to parse Headscale status: %w", err)
	}

	status.Connected = tsStatus.BackendState == "Running"
	status.Message = fmt.Sprintf("Headscale (%s)", tsStatus.BackendState)

	// Get local IP
	if tsStatus.Self.TailscaleIPs != nil && len(tsStatus.Self.TailscaleIPs) > 0 {
		status.LocalIP = tsStatus.Self.TailscaleIPs[0]
	}

	// Parse peers
	for _, peer := range tsStatus.Peer {
		peerInfo := PeerInfo{
			Hostname: peer.HostName,
			Online:   peer.Online,
		}

		if peer.TailscaleIPs != nil && len(peer.TailscaleIPs) > 0 {
			peerInfo.IP = peer.TailscaleIPs[0]
		}

		if peer.LastSeen != "" {
			if t, err := time.Parse(time.RFC3339, peer.LastSeen); err == nil {
				peerInfo.LastSeen = t
			}
		}

		status.Peers = append(status.Peers, peerInfo)
	}

	return status, nil
}

// GetPeerIP resolves a Headscale hostname to IP
func (b *HeadscaleBackend) GetPeerIP(ctx context.Context, hostname string) (string, error) {
	if !b.IsConnected(ctx) {
		return "", ErrNotConnected
	}

	// Use tailscale ip command to resolve hostname
	cmd := exec.CommandContext(ctx, "tailscale", "ip", "-4", hostname)
	output, err := cmd.Output()
	if err != nil {
		// If tailscale ip fails, try to find it in status
		status, statusErr := b.GetStatus(ctx)
		if statusErr != nil {
			return "", ErrPeerNotFound
		}

		// Search for peer by hostname
		for _, peer := range status.Peers {
			if strings.EqualFold(peer.Hostname, hostname) {
				if peer.IP != "" {
					return peer.IP, nil
				}
			}
		}

		return "", ErrPeerNotFound
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "", ErrPeerNotFound
	}

	return ip, nil
}

// Priority returns the priority for auto-detection (same as Tailscale)
// Note: When using "auto" backend, the profile configuration should specify
// whether it's Tailscale or Headscale, as they use the same client
func (b *HeadscaleBackend) Priority() int {
	return 40
}
