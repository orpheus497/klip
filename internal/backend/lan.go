package backend

import (
	"context"
	"net"
	"time"
)

// LANBackend implements direct LAN/IP connections
type LANBackend struct{}

// Name returns the backend name
func (b *LANBackend) Name() string {
	return "lan"
}

// IsAvailable checks if LAN backend is available (always true)
func (b *LANBackend) IsAvailable(ctx context.Context) bool {
	return true
}

// IsConnected checks if we have network connectivity
func (b *LANBackend) IsConnected(ctx context.Context) bool {
	// LAN is considered "connected" if we have any network interface up
	interfaces, err := net.Interfaces()
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Check if interface has addresses
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		if len(addrs) > 0 {
			return true
		}
	}

	return false
}

// GetStatus returns LAN status
func (b *LANBackend) GetStatus(ctx context.Context) (*Status, error) {
	status := &Status{
		Backend:   b.Name(),
		Connected: b.IsConnected(ctx),
		LastCheck: time.Now(),
	}

	if status.Connected {
		status.Message = "Network interfaces active"
		// Try to get local IP
		if localIP := b.getLocalIP(); localIP != "" {
			status.LocalIP = localIP
		}
	} else {
		status.Message = "No active network interfaces"
	}

	return status, nil
}

// GetPeerIP resolves a hostname to IP (uses DNS)
func (b *LANBackend) GetPeerIP(ctx context.Context, hostname string) (string, error) {
	// Check if it's already an IP address
	if ip := net.ParseIP(hostname); ip != nil {
		return hostname, nil
	}

	// Resolve hostname using DNS with context
	resolver := &net.Resolver{}
	ips, err := resolver.LookupIP(ctx, "ip4", hostname)
	if err != nil {
		return "", err
	}

	if len(ips) == 0 {
		return "", ErrPeerNotFound
	}

	return ips[0].String(), nil
}

// Priority returns the priority for auto-detection (lowest, as LAN is fallback)
func (b *LANBackend) Priority() int {
	return 10
}

// getLocalIP attempts to determine the local IP address
func (b *LANBackend) getLocalIP() string {
	// Try to get the IP by dialing a remote address (doesn't actually connect)
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return b.getFirstNonLoopbackIP()
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getFirstNonLoopbackIP returns the first non-loopback IP address
func (b *LANBackend) getFirstNonLoopbackIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			// Prefer IPv4
			if ip.To4() != nil {
				return ip.String()
			}
		}
	}

	return ""
}
