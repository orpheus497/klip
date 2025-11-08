// Package backend provides VPN backend abstraction for klip
package backend

import (
	"context"
	"fmt"
	"time"
)

// Backend represents a VPN backend interface
type Backend interface {
	// Name returns the backend name (lan, tailscale, headscale, netbird)
	Name() string

	// IsAvailable checks if the backend is installed and available
	IsAvailable(ctx context.Context) bool

	// IsConnected checks if the backend is currently connected
	IsConnected(ctx context.Context) bool

	// GetStatus returns detailed status information
	GetStatus(ctx context.Context) (*Status, error)

	// GetPeerIP resolves a hostname to an IP address through this backend
	GetPeerIP(ctx context.Context, hostname string) (string, error)

	// Priority returns the priority for auto-detection (higher = preferred)
	Priority() int
}

// Status represents the connection status of a backend
type Status struct {
	// Backend is the backend name
	Backend string

	// Connected indicates if the backend is connected
	Connected bool

	// Message provides additional status information
	Message string

	// LocalIP is the local IP address on this network
	LocalIP string

	// Peers contains information about connected peers
	Peers []PeerInfo

	// LastCheck is when the status was retrieved
	LastCheck time.Time
}

// PeerInfo represents information about a peer in the network
type PeerInfo struct {
	// Hostname is the peer's hostname
	Hostname string

	// IP is the peer's IP address
	IP string

	// Online indicates if the peer is currently online
	Online bool

	// LastSeen is when the peer was last seen
	LastSeen time.Time
}

// Error types
var (
	// ErrNotAvailable indicates the backend is not installed or unavailable
	ErrNotAvailable = fmt.Errorf("backend not available")

	// ErrNotConnected indicates the backend is not connected
	ErrNotConnected = fmt.Errorf("backend not connected")

	// ErrPeerNotFound indicates the requested peer was not found
	ErrPeerNotFound = fmt.Errorf("peer not found")

	// ErrCommandFailed indicates a backend command failed
	ErrCommandFailed = fmt.Errorf("backend command failed")

	// ErrTimeout indicates an operation timed out
	ErrTimeout = fmt.Errorf("operation timed out")
)

// Registry maintains all available backends
type Registry struct {
	backends map[string]Backend
}

// NewRegistry creates a new backend registry with all supported backends
func NewRegistry() *Registry {
	r := &Registry{
		backends: make(map[string]Backend),
	}

	// Register all backends
	r.Register(&LANBackend{})
	r.Register(&TailscaleBackend{})
	r.Register(&HeadscaleBackend{})
	r.Register(&NetBirdBackend{})

	return r
}

// Register adds a backend to the registry
func (r *Registry) Register(backend Backend) {
	r.backends[backend.Name()] = backend
}

// Get retrieves a backend by name
func (r *Registry) Get(name string) (Backend, error) {
	backend, exists := r.backends[name]
	if !exists {
		return nil, fmt.Errorf("backend '%s' not found", name)
	}
	return backend, nil
}

// List returns all registered backends
func (r *Registry) List() []Backend {
	backends := make([]Backend, 0, len(r.backends))
	for _, backend := range r.backends {
		backends = append(backends, backend)
	}
	return backends
}

// DetectBest finds the best available and connected backend
func (r *Registry) DetectBest(ctx context.Context) (Backend, error) {
	detector := &Detector{registry: r}
	return detector.DetectBest(ctx)
}
