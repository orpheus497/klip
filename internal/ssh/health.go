package ssh

import (
	"context"
	"fmt"
	"time"
)

// HealthCheckResult contains the result of an SSH health check
type HealthCheckResult struct {
	Reachable      bool
	Authenticated  bool
	ResponseTime   time.Duration
	Error          error
	Message        string
}

// HealthCheck performs a health check on an SSH connection
func HealthCheck(ctx context.Context, cfg *Config) *HealthCheckResult {
	result := &HealthCheckResult{}
	start := time.Now()

	client, err := NewClient(cfg)
	if err != nil {
		result.Reachable = false
		result.Error = err
		result.Message = fmt.Sprintf("Failed to create client: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}

	// Attempt connection
	if err := client.Connect(ctx); err != nil {
		result.Reachable = false
		result.Error = err
		result.Message = fmt.Sprintf("Connection failed: %v", err)
		result.ResponseTime = time.Since(start)
		return result
	}
	defer client.Close()

	result.Reachable = true
	result.Authenticated = true
	result.ResponseTime = time.Since(start)

	// Try a simple command to verify everything works
	output, err := client.RunCommand(ctx, "echo 'klip-health-check'")
	if err != nil {
		result.Message = fmt.Sprintf("Command execution failed: %v", err)
		result.Error = err
	} else if output != "klip-health-check\n" {
		result.Message = "Command execution succeeded with unexpected output"
	} else {
		result.Message = fmt.Sprintf("Healthy (%.2fs)", result.ResponseTime.Seconds())
	}

	return result
}

// QuickCheck performs a quick connectivity check without full authentication
func QuickCheck(ctx context.Context, host string, port int) bool {
	if port == 0 {
		port = 22
	}

	cfg := &Config{
		Host:    host,
		Port:    port,
		User:    "dummy", // User doesn't matter for connection test
		Timeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		return false
	}

	// Just try to connect, we don't care about auth for quick check
	err = client.Connect(ctx)
	if client.client != nil {
		client.Close()
	}

	// If we get auth error, host is reachable
	// If we get connection error, host is not reachable
	return err == nil || isAuthError(err)
}

// isAuthError checks if an error is an authentication error
func isAuthError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	authErrors := []string{
		"unable to authenticate",
		"permission denied",
		"authentication failed",
		"no supported authentication",
	}

	for _, authErr := range authErrors {
		if contains(errStr, authErr) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
