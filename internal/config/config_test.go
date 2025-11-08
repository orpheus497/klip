package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Profiles)
	assert.Equal(t, DefaultSettings(), cfg.Settings)
}

func TestProfileValidation(t *testing.T) {
	tests := []struct {
		name      string
		profile   *Profile
		wantError bool
	}{
		{
			name: "valid profile",
			profile: &Profile{
				RemoteUser: "user",
				RemoteHost: "host",
				SSHPort:    22,
				Backend:    BackendAuto,
			},
			wantError: false,
		},
		{
			name: "missing user",
			profile: &Profile{
				RemoteHost: "host",
				SSHPort:    22,
			},
			wantError: true,
		},
		{
			name: "missing host",
			profile: &Profile{
				RemoteUser: "user",
				SSHPort:    22,
			},
			wantError: true,
		},
		{
			name: "invalid port",
			profile: &Profile{
				RemoteUser: "user",
				RemoteHost: "host",
				SSHPort:    99999,
			},
			wantError: true,
		},
		{
			name: "invalid backend",
			profile: &Profile{
				RemoteUser: "user",
				RemoteHost: "host",
				SSHPort:    22,
				Backend:    BackendType("invalid"),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.profile.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProfileClone(t *testing.T) {
	original := NewProfile("test", "user", "host")
	original.Description = "Test profile"
	original.TransferOptions.ExcludePatterns = []string{"*.tmp", "*.log"}

	clone := original.Clone()

	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.RemoteUser, clone.RemoteUser)
	assert.Equal(t, original.RemoteHost, clone.RemoteHost)
	assert.Equal(t, original.Description, clone.Description)

	// Verify deep copy of slice
	assert.Equal(t, original.TransferOptions.ExcludePatterns, clone.TransferOptions.ExcludePatterns)
	clone.TransferOptions.ExcludePatterns[0] = "*.bak"
	assert.NotEqual(t, original.TransferOptions.ExcludePatterns[0], clone.TransferOptions.ExcludePatterns[0])
}

func TestAddProfile(t *testing.T) {
	cfg := NewConfig()

	profile := NewProfile("test", "user", "host")
	err := cfg.AddProfile("test", profile)

	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1)
	assert.Equal(t, "test", cfg.CurrentProfile) // First profile becomes current

	// Add second profile
	profile2 := NewProfile("test2", "user2", "host2")
	err = cfg.AddProfile("test2", profile2)

	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 2)
	assert.Equal(t, "test", cfg.CurrentProfile) // Current should not change
}

func TestDeleteProfile(t *testing.T) {
	cfg := NewConfig()

	profile1 := NewProfile("test1", "user1", "host1")
	profile2 := NewProfile("test2", "user2", "host2")

	cfg.AddProfile("test1", profile1)
	cfg.AddProfile("test2", profile2)

	// Delete non-current profile
	err := cfg.DeleteProfile("test2")
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 1)

	// Delete current profile
	err = cfg.DeleteProfile("test1")
	require.NoError(t, err)
	assert.Len(t, cfg.Profiles, 0)
	assert.Empty(t, cfg.CurrentProfile)
}

func TestSanitizeProfile(t *testing.T) {
	profile := &Profile{
		Name:       "  test  ",
		RemoteUser: "  user  ",
		RemoteHost: "  host  ",
		TransferOptions: TransferOptions{
			CompressionLevel: -1,
		},
	}

	SanitizeProfile(profile)

	assert.Equal(t, "test", profile.Name)
	assert.Equal(t, "user", profile.RemoteUser)
	assert.Equal(t, "host", profile.RemoteHost)
	assert.Equal(t, 0, profile.TransferOptions.CompressionLevel)
	assert.Equal(t, 22, profile.SSHPort)
	assert.Equal(t, BackendAuto, profile.Backend)
}

func TestParseBashConfig(t *testing.T) {
	// Create temporary bash config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.sh")

	content := `# Test config
LAN_REMOTE_USER="testuser"
LAN_REMOTE_HOST="testhost"
TS_REMOTE_USER='tsuser'
TS_REMOTE_HOST='tshost'
INVALID_LINE
# Comment
EMPTY_VALUE=
`

	err := os.WriteFile(configPath, []byte(content), 0600)
	require.NoError(t, err)

	vars, err := parseBashConfig(configPath)
	require.NoError(t, err)

	assert.Equal(t, "testuser", vars["LAN_REMOTE_USER"])
	assert.Equal(t, "testhost", vars["LAN_REMOTE_HOST"])
	assert.Equal(t, "tsuser", vars["TS_REMOTE_USER"])
	assert.Equal(t, "tshost", vars["TS_REMOTE_HOST"])
}

func TestIsPlaceholderValue(t *testing.T) {
	tests := []struct {
		value       string
		placeholder bool
	}{
		{"", true},
		{"your_lan_user", true},
		{"localhost", true},
		{"realuser", false},
		{"192.168.1.1", false},
		{"example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			result := isPlaceholderValue(tt.value)
			assert.Equal(t, tt.placeholder, result)
		})
	}
}
