package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockBackend is a mock implementation for testing
type MockBackend struct {
	name      string
	available bool
	connected bool
	priority  int
	status    *Status
}

func (m *MockBackend) Name() string {
	return m.name
}

func (m *MockBackend) IsAvailable(ctx context.Context) bool {
	return m.available
}

func (m *MockBackend) IsConnected(ctx context.Context) bool {
	return m.connected
}

func (m *MockBackend) GetStatus(ctx context.Context) (*Status, error) {
	if m.status != nil {
		return m.status, nil
	}
	return &Status{
		Backend:   m.name,
		Connected: m.connected,
		Message:   "Mock status",
		LastCheck: time.Now(),
	}, nil
}

func (m *MockBackend) GetPeerIP(ctx context.Context, hostname string) (string, error) {
	if !m.connected {
		return "", ErrNotConnected
	}
	return "192.168.1.1", nil
}

func (m *MockBackend) Priority() int {
	return m.priority
}

func TestRegistry(t *testing.T) {
	registry := &Registry{
		backends: make(map[string]Backend),
	}

	// Test registration
	mockBackend := &MockBackend{name: "mock", available: true}
	registry.Register(mockBackend)

	// Test retrieval
	backend, err := registry.Get("mock")
	require.NoError(t, err)
	assert.Equal(t, "mock", backend.Name())

	// Test non-existent backend
	_, err = registry.Get("nonexistent")
	assert.Error(t, err)

	// Test list
	backends := registry.List()
	assert.Len(t, backends, 1)
}

func TestDetectorSelectBest(t *testing.T) {
	tests := []struct {
		name     string
		backends []Backend
		expected string
	}{
		{
			name: "selects highest priority connected backend",
			backends: []Backend{
				&MockBackend{name: "low", available: true, connected: true, priority: 10},
				&MockBackend{name: "high", available: true, connected: true, priority: 50},
				&MockBackend{name: "medium", available: true, connected: true, priority: 30},
			},
			expected: "high",
		},
		{
			name: "ignores unavailable backends",
			backends: []Backend{
				&MockBackend{name: "unavailable", available: false, priority: 100},
				&MockBackend{name: "available", available: true, connected: true, priority: 10},
			},
			expected: "available",
		},
		{
			name: "selects highest priority available when none connected",
			backends: []Backend{
				&MockBackend{name: "low", available: true, connected: false, priority: 10},
				&MockBackend{name: "high", available: true, connected: false, priority: 50},
			},
			expected: "high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := &Registry{
				backends: make(map[string]Backend),
			}

			for _, b := range tt.backends {
				registry.Register(b)
			}

			detector := &Detector{registry: registry}
			ctx := context.Background()

			backend, err := detector.DetectBest(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, backend.Name())
		})
	}
}

func TestDetectorSelectBackend(t *testing.T) {
	registry := &Registry{
		backends: make(map[string]Backend),
	}

	mockLAN := &MockBackend{name: "lan", available: true, connected: true, priority: 10}
	mockTS := &MockBackend{name: "tailscale", available: true, connected: false, priority: 40}

	registry.Register(mockLAN)
	registry.Register(mockTS)

	detector := &Detector{registry: registry}
	ctx := context.Background()

	// Test auto selection
	backend, err := detector.SelectBackend(ctx, "auto")
	require.NoError(t, err)
	assert.Equal(t, "lan", backend.Name())

	// Test specific backend selection
	backend, err = detector.SelectBackend(ctx, "tailscale")
	require.NoError(t, err)
	assert.Equal(t, "tailscale", backend.Name())

	// Test unavailable backend
	_, err = detector.SelectBackend(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestDetectorHealthCheck(t *testing.T) {
	registry := &Registry{
		backends: make(map[string]Backend),
	}

	mockBackends := []*MockBackend{
		{name: "available-connected", available: true, connected: true, priority: 10},
		{name: "available-disconnected", available: true, connected: false, priority: 10},
		{name: "unavailable", available: false, connected: false, priority: 10},
	}

	for _, b := range mockBackends {
		registry.Register(b)
	}

	detector := &Detector{registry: registry}
	ctx := context.Background()

	results := detector.HealthCheck(ctx)

	assert.Len(t, results, 3)

	for _, result := range results {
		switch result.Backend {
		case "available-connected":
			assert.True(t, result.Available)
			assert.True(t, result.Connected)
		case "available-disconnected":
			assert.True(t, result.Available)
			assert.False(t, result.Connected)
		case "unavailable":
			assert.False(t, result.Available)
			assert.False(t, result.Connected)
		}
	}
}

func TestDetectorResolveHost(t *testing.T) {
	registry := &Registry{
		backends: make(map[string]Backend),
	}

	mockBackend := &MockBackend{
		name:      "test",
		available: true,
		connected: true,
		priority:  10,
	}

	registry.Register(mockBackend)

	detector := &Detector{registry: registry}
	ctx := context.Background()

	ip, err := detector.ResolveHost(ctx, mockBackend, "testhost")
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.1", ip)
}

func TestDetectorDetectAll(t *testing.T) {
	registry := &Registry{
		backends: make(map[string]Backend),
	}

	mockBackends := []*MockBackend{
		{name: "backend1", available: true, connected: true, priority: 10},
		{name: "backend2", available: false, connected: false, priority: 20},
	}

	for _, b := range mockBackends {
		registry.Register(b)
	}

	detector := &Detector{registry: registry}
	ctx := context.Background()

	statuses := detector.DetectAll(ctx)

	assert.Len(t, statuses, 2)
	assert.True(t, statuses["backend1"].Connected)
	assert.False(t, statuses["backend2"].Connected)
}
