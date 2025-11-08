package backend

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Detector handles backend auto-detection
type Detector struct {
	registry *Registry
}

// NewDetector creates a new backend detector
func NewDetector(registry *Registry) *Detector {
	return &Detector{registry: registry}
}

// DetectBest finds the best available and connected backend using parallel detection
func (d *Detector) DetectBest(ctx context.Context) (Backend, error) {
	backends := d.registry.List()

	// Sort backends by priority (highest first)
	sort.Slice(backends, func(i, j int) bool {
		return backends[i].Priority() > backends[j].Priority()
	})

	// Result structure for parallel detection
	type backendResult struct {
		backend   Backend
		available bool
		connected bool
	}

	// Channel for results
	results := make(chan backendResult, len(backends))

	// Check backends in parallel
	for _, backend := range backends {
		go func(b Backend) {
			result := backendResult{backend: b}
			result.available = b.IsAvailable(ctx)
			if result.available {
				result.connected = b.IsConnected(ctx)
			}
			results <- result
		}(backend)
	}

	// Collect results
	var availableBackends []Backend
	var connectedBackend Backend

	for i := 0; i < len(backends); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-results:
			if !result.available {
				continue
			}

			availableBackends = append(availableBackends, result.backend)

			// If we found a connected backend and don't have one yet, save it
			// We need to check priority later
			if result.connected && connectedBackend == nil {
				connectedBackend = result.backend
			} else if result.connected && connectedBackend != nil {
				// Choose higher priority backend
				if result.backend.Priority() > connectedBackend.Priority() {
					connectedBackend = result.backend
				}
			}
		}
	}

	// If we found a connected backend, return it
	if connectedBackend != nil {
		return connectedBackend, nil
	}

	// If no connected backend, but we have available ones, return the highest priority
	if len(availableBackends) > 0 {
		// Sort by priority
		sort.Slice(availableBackends, func(i, j int) bool {
			return availableBackends[i].Priority() > availableBackends[j].Priority()
		})
		return availableBackends[0], nil
	}

	return nil, fmt.Errorf("no available backends found")
}

// DetectAll returns status of all backends
func (d *Detector) DetectAll(ctx context.Context) map[string]*Status {
	results := make(map[string]*Status)
	backends := d.registry.List()

	for _, backend := range backends {
		if !backend.IsAvailable(ctx) {
			results[backend.Name()] = &Status{
				Backend:   backend.Name(),
				Connected: false,
				Message:   "Not installed",
				LastCheck: time.Now(),
			}
			continue
		}

		status, err := backend.GetStatus(ctx)
		if err != nil {
			results[backend.Name()] = &Status{
				Backend:   backend.Name(),
				Connected: false,
				Message:   err.Error(),
				LastCheck: time.Now(),
			}
			continue
		}

		results[backend.Name()] = status
	}

	return results
}

// DetectByName returns status for a specific backend
func (d *Detector) DetectByName(ctx context.Context, name string) (*Status, error) {
	backend, err := d.registry.Get(name)
	if err != nil {
		return nil, err
	}

	if !backend.IsAvailable(ctx) {
		return &Status{
			Backend:   name,
			Connected: false,
			Message:   "Not installed",
			LastCheck: time.Now(),
		}, nil
	}

	return backend.GetStatus(ctx)
}

// SelectBackend chooses the appropriate backend based on preference
// preference can be "auto", "lan", "tailscale", "headscale", or "netbird"
func (d *Detector) SelectBackend(ctx context.Context, preference string) (Backend, error) {
	if preference == "auto" || preference == "" {
		return d.DetectBest(ctx)
	}

	backend, err := d.registry.Get(preference)
	if err != nil {
		return nil, err
	}

	if !backend.IsAvailable(ctx) {
		return nil, fmt.Errorf("backend '%s' is not available (not installed)", preference)
	}

	return backend, nil
}

// ResolveHost resolves a hostname using the appropriate backend
func (d *Detector) ResolveHost(ctx context.Context, backend Backend, hostname string) (string, error) {
	if backend == nil {
		return "", fmt.Errorf("backend is nil")
	}

	ip, err := backend.GetPeerIP(ctx, hostname)
	if err != nil {
		// If resolution fails on VPN backend, try LAN as fallback
		if backend.Name() != "lan" {
			lanBackend := &LANBackend{}
			if lanIP, lanErr := lanBackend.GetPeerIP(ctx, hostname); lanErr == nil {
				return lanIP, nil
			}
		}
		return "", err
	}

	return ip, nil
}

// HealthCheck performs a health check on all backends
type HealthCheckResult struct {
	Backend   string
	Available bool
	Connected bool
	Message   string
	Duration  time.Duration
}

// HealthCheck checks the health of all backends
func (d *Detector) HealthCheck(ctx context.Context) []HealthCheckResult {
	var results []HealthCheckResult
	backends := d.registry.List()

	for _, backend := range backends {
		start := time.Now()

		result := HealthCheckResult{
			Backend: backend.Name(),
		}

		result.Available = backend.IsAvailable(ctx)
		if !result.Available {
			result.Message = "Not installed"
			result.Duration = time.Since(start)
			results = append(results, result)
			continue
		}

		result.Connected = backend.IsConnected(ctx)

		status, err := backend.GetStatus(ctx)
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Message = status.Message
		}

		result.Duration = time.Since(start)
		results = append(results, result)
	}

	return results
}
