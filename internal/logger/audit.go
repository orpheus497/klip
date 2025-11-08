// Package logger - Audit logging for security events
// Copyright (c) 2025 orpheus497
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
)

// AuditEvent represents a security audit event
// All events are logged in JSON format for machine parsing
type AuditEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	EventType   string            `json:"event_type"`
	Profile     string            `json:"profile"`
	User        string            `json:"user"`
	Host        string            `json:"host"`
	Backend     string            `json:"backend"`
	Operation   string            `json:"operation,omitempty"`
	Source      string            `json:"source,omitempty"`
	Destination string            `json:"destination,omitempty"`
	Status      string            `json:"status"`
	Error       string            `json:"error,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// AuditLogger logs security and operational events
// Thread-safe implementation with JSON output
type AuditLogger struct {
	file    *os.File
	encoder *json.Encoder
	enabled bool
	mu      sync.Mutex
}

// NewAuditLogger creates a new audit logger
// If enabled is false, the logger is a no-op (for performance)
func NewAuditLogger(enabled bool) (*AuditLogger, error) {
	if !enabled {
		return &AuditLogger{enabled: false}, nil
	}

	// Get XDG-compliant state directory for audit log
	auditPath := filepath.Join(xdg.StateHome, "klip", "audit.log")

	// Ensure directory exists with secure permissions
	if err := os.MkdirAll(filepath.Dir(auditPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	// Open audit log file (append mode, create if not exists)
	file, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}

	return &AuditLogger{
		file:    file,
		encoder: json.NewEncoder(file),
		enabled: true,
	}, nil
}

// Log logs a generic audit event
// Thread-safe operation
func (a *AuditLogger) Log(event AuditEvent) error {
	if !a.enabled {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Set timestamp to current UTC time
	event.Timestamp = time.Now().UTC()

	// Encode and write to file
	if err := a.encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to write audit event: %w", err)
	}

	return nil
}

// LogConnection logs a connection event (success or failure)
func (a *AuditLogger) LogConnection(profile, user, host, backend, status string, err error) error {
	event := AuditEvent{
		EventType: "connection",
		Profile:   profile,
		User:      user,
		Host:      host,
		Backend:   backend,
		Status:    status,
	}

	if err != nil {
		event.Error = err.Error()
	}

	return a.Log(event)
}

// LogTransfer logs a file transfer event
func (a *AuditLogger) LogTransfer(profile, user, host, backend, operation, source, dest, status string, err error) error {
	event := AuditEvent{
		EventType:   "transfer",
		Profile:     profile,
		User:        user,
		Host:        host,
		Backend:     backend,
		Operation:   operation,
		Source:      source,
		Destination: dest,
		Status:      status,
	}

	if err != nil {
		event.Error = err.Error()
	}

	return a.Log(event)
}

// LogProfileChange logs profile creation, modification, or deletion
func (a *AuditLogger) LogProfileChange(profile, operation, status string, err error) error {
	event := AuditEvent{
		EventType: "profile_change",
		Profile:   profile,
		Operation: operation,
		Status:    status,
	}

	if err != nil {
		event.Error = err.Error()
	}

	return a.Log(event)
}

// LogSSHKeyDeployment logs SSH key deployment events
func (a *AuditLogger) LogSSHKeyDeployment(profile, user, host, backend, status string, err error) error {
	event := AuditEvent{
		EventType: "ssh_key_deployment",
		Profile:   profile,
		User:      user,
		Host:      host,
		Backend:   backend,
		Operation: "deploy_public_key",
		Status:    status,
	}

	if err != nil {
		event.Error = err.Error()
	}

	return a.Log(event)
}

// LogHealthCheck logs health check operations
func (a *AuditLogger) LogHealthCheck(profile, backend, status string, metadata map[string]string, err error) error {
	event := AuditEvent{
		EventType: "health_check",
		Profile:   profile,
		Backend:   backend,
		Operation: "health_check",
		Status:    status,
		Metadata:  metadata,
	}

	if err != nil {
		event.Error = err.Error()
	}

	return a.Log(event)
}

// Close closes the audit log file
// Should be called when the application exits
func (a *AuditLogger) Close() error {
	if !a.enabled || a.file == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	return a.file.Close()
}

// GetAuditLogPath returns the path to the audit log file
func GetAuditLogPath() (string, error) {
	return filepath.Join(xdg.StateHome, "klip", "audit.log"), nil
}

// IsEnabled returns whether audit logging is enabled
func (a *AuditLogger) IsEnabled() bool {
	return a.enabled
}
