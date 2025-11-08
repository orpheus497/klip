package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/orpheus497/klip/internal/backend"
	"github.com/orpheus497/klip/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigurationWorkflow tests the full configuration workflow
func TestConfigurationWorkflow(t *testing.T) {
	// Skip in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI")
	}

	// Create temporary directory for test config
	tmpDir := t.TempDir()
	oldXDGConfig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldXDGConfig)

	// Test new configuration creation
	cfg := config.NewConfig()
	require.NotNil(t, cfg)

	// Add a test profile
	profile := config.NewProfile("test-server", "testuser", "testhost")
	profile.Backend = config.BackendLAN
	profile.Description = "Test server profile"

	err := cfg.AddProfile("test-server", profile)
	require.NoError(t, err)

	// Test configuration save
	err = cfg.Save()
	require.NoError(t, err)

	// Verify file was created
	configPath, err := config.ConfigPath()
	require.NoError(t, err)
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Test configuration load
	loadedCfg, err := config.Load()
	require.NoError(t, err)
	assert.Len(t, loadedCfg.Profiles, 1)

	// Test profile retrieval
	loadedProfile, err := loadedCfg.GetProfile("test-server")
	require.NoError(t, err)
	assert.Equal(t, "testuser", loadedProfile.RemoteUser)
	assert.Equal(t, "testhost", loadedProfile.RemoteHost)
	assert.Equal(t, config.BackendLAN, loadedProfile.Backend)

	// Test profile validation
	err = loadedProfile.Validate()
	assert.NoError(t, err)
}

// TestBackendRegistry tests the backend registry system
func TestBackendRegistry(t *testing.T) {
	registry := backend.NewRegistry()
	require.NotNil(t, registry)

	// Test that all backends are registered
	backends := registry.List()
	assert.NotEmpty(t, backends)

	// Verify specific backends exist
	backendNames := make(map[string]bool)
	for _, b := range backends {
		backendNames[b.Name()] = true
	}

	assert.True(t, backendNames["lan"])
	assert.True(t, backendNames["tailscale"])
	assert.True(t, backendNames["headscale"])
	assert.True(t, backendNames["netbird"])

	// Test backend retrieval
	lanBackend, err := registry.Get("lan")
	require.NoError(t, err)
	assert.Equal(t, "lan", lanBackend.Name())

	// Test non-existent backend
	_, err = registry.Get("nonexistent")
	assert.Error(t, err)
}

// TestBackendDetection tests basic backend detection
func TestBackendDetection(t *testing.T) {
	ctx := context.Background()
	registry := backend.NewRegistry()

	// Test LAN backend (should always be available)
	lanBackend, err := registry.Get("lan")
	require.NoError(t, err)

	assert.True(t, lanBackend.IsAvailable(ctx))

	status, err := lanBackend.GetStatus(ctx)
	require.NoError(t, err)
	assert.Equal(t, "lan", status.Backend)
}

// TestProfileValidationRules tests profile validation edge cases
func TestProfileValidationRules(t *testing.T) {
	tests := []struct {
		name      string
		profile   *config.Profile
		wantError bool
	}{
		{
			name:      "nil profile",
			profile:   nil,
			wantError: true,
		},
		{
			name: "valid minimal profile",
			profile: &config.Profile{
				RemoteUser: "user",
				RemoteHost: "host",
				SSHPort:    22,
				Backend:    config.BackendAuto,
			},
			wantError: false,
		},
		{
			name: "profile with all options",
			profile: &config.Profile{
				Name:        "full",
				Description: "Full profile",
				RemoteUser:  "user",
				RemoteHost:  "host",
				SSHPort:     2222,
				SSHKeyPath:  "/path/to/key",
				Backend:     config.BackendTailscale,
				TransferOptions: config.TransferOptions{
					Method:              "rsync",
					CompressionLevel:    9,
					ExcludePatterns:     []string{"*.log"},
					BandwidthLimit:      1000,
					PreservePermissions: true,
				},
			},
			wantError: false,
		},
		{
			name: "invalid SSH port",
			profile: &config.Profile{
				RemoteUser: "user",
				RemoteHost: "host",
				SSHPort:    -1,
				Backend:    config.BackendAuto,
			},
			wantError: true,
		},
		{
			name: "invalid compression level",
			profile: &config.Profile{
				RemoteUser: "user",
				RemoteHost: "host",
				SSHPort:    22,
				Backend:    config.BackendAuto,
				TransferOptions: config.TransferOptions{
					CompressionLevel: 15,
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := config.ValidateProfile(tt.profile)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLegacyConfigMigration tests migration from bash config
func TestLegacyConfigMigration(t *testing.T) {
	// Create temporary legacy config
	tmpDir := t.TempDir()
	legacyDir := filepath.Join(tmpDir, ".LINK")
	err := os.MkdirAll(legacyDir, 0755)
	require.NoError(t, err)

	legacyConfig := `# Legacy LINK config
LAN_REMOTE_USER="olduser"
LAN_REMOTE_HOST="oldhost"
TS_REMOTE_USER="tsuser"
TS_REMOTE_HOST="tshost"
`

	legacyPath := filepath.Join(legacyDir, "config.sh")
	err = os.WriteFile(legacyPath, []byte(legacyConfig), 0600)
	require.NoError(t, err)

	// Set home directory for test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Test migration
	migrated, err := config.MigrateLegacyConfig()
	require.NoError(t, err)
	assert.Len(t, migrated.Profiles, 2)

	// Verify LAN profile
	lanProfile, err := migrated.GetProfile("lan")
	require.NoError(t, err)
	assert.Equal(t, "olduser", lanProfile.RemoteUser)
	assert.Equal(t, "oldhost", lanProfile.RemoteHost)

	// Verify Tailscale profile
	tsProfile, err := migrated.GetProfile("tailscale")
	require.NoError(t, err)
	assert.Equal(t, "tsuser", tsProfile.RemoteUser)
	assert.Equal(t, "tshost", tsProfile.RemoteHost)
}
