package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// TailscaleBackend implements Tailscale VPN backend
type TailscaleBackend struct{}

// Name returns the backend name
func (b *TailscaleBackend) Name() string {
	return "tailscale"
}

// IsAvailable checks if Tailscale is installed
func (b *TailscaleBackend) IsAvailable(ctx context.Context) bool {
	_, err := exec.LookPath("tailscale")
	return err == nil
}

// IsConnected checks if Tailscale is running and connected
func (b *TailscaleBackend) IsConnected(ctx context.Context) bool {
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
	return status.BackendState == "Running"
}

// GetStatus returns Tailscale status
func (b *TailscaleBackend) GetStatus(ctx context.Context) (*Status, error) {
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
		return status, fmt.Errorf("failed to parse Tailscale status: %w", err)
	}

	status.Connected = tsStatus.BackendState == "Running"
	status.Message = tsStatus.BackendState

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

// GetPeerIP resolves a Tailscale hostname to IP
func (b *TailscaleBackend) GetPeerIP(ctx context.Context, hostname string) (string, error) {
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

// Priority returns the priority for auto-detection (high priority)
func (b *TailscaleBackend) Priority() int {
	return 40
}

// tailscaleStatus represents the Tailscale status JSON output
type tailscaleStatus struct {
	BackendState string                       `json:"BackendState"`
	Self         tailscaleSelf                `json:"Self"`
	Peer         map[string]tailscalePeerInfo `json:"Peer"`
}

// tailscaleSelf represents information about the local Tailscale node
type tailscaleSelf struct {
	HostName     string   `json:"HostName"`
	TailscaleIPs []string `json:"TailscaleIPs"`
}

// tailscalePeerInfo represents information about a Tailscale peer
type tailscalePeerInfo struct {
	HostName     string   `json:"HostName"`
	TailscaleIPs []string `json:"TailscaleIPs"`
	Online       bool     `json:"Online"`
	LastSeen     string   `json:"LastSeen"`
}
