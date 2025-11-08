package ssh

import (
	"context"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/ssh"
)

// Session wraps SSH session functionality
type Session struct {
	client     *Client
	sshSession *ssh.Session
}

// SessionConfig contains session configuration
type SessionConfig struct {
	// Command to execute (if empty, starts shell)
	Command string

	// Stdin, Stdout, Stderr streams
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// RequestPTY requests a pseudo-terminal
	RequestPTY bool

	// Env contains environment variables to set
	Env map[string]string
}

// NewSession creates a new session
func (c *Client) NewSessionWithConfig(cfg *SessionConfig) (*Session, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}

	sshSession, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set up streams
	if cfg.Stdin != nil {
		sshSession.Stdin = cfg.Stdin
	}
	if cfg.Stdout != nil {
		sshSession.Stdout = cfg.Stdout
	}
	if cfg.Stderr != nil {
		sshSession.Stderr = cfg.Stderr
	}

	// Set environment variables
	for key, value := range cfg.Env {
		if err := sshSession.Setenv(key, value); err != nil {
			// Some SSH servers don't allow setting environment variables
			// Log this but don't fail
			continue
		}
	}

	// Request PTY if needed
	if cfg.RequestPTY {
		modes := ssh.TerminalModes{
			ssh.ECHO:          1,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}

		if err := sshSession.RequestPty("xterm-256color", 40, 80, modes); err != nil {
			sshSession.Close()
			return nil, fmt.Errorf("failed to request PTY: %w", err)
		}
	}

	return &Session{
		client:     c,
		sshSession: sshSession,
	}, nil
}

// Run executes the configured command or shell
func (s *Session) Run(ctx context.Context, command string) error {
	if s.sshSession == nil {
		return fmt.Errorf("session is closed")
	}

	// Set up context cancellation
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-done:
		}
	}()
	defer close(done)

	if command != "" {
		return s.sshSession.Run(command)
	}

	return s.sshSession.Shell()
}

// Start starts the command but doesn't wait for it to complete
func (s *Session) Start(command string) error {
	if s.sshSession == nil {
		return fmt.Errorf("session is closed")
	}

	if command != "" {
		return s.sshSession.Start(command)
	}

	return s.sshSession.Shell()
}

// Wait waits for the session to complete
func (s *Session) Wait() error {
	if s.sshSession == nil {
		return fmt.Errorf("session is closed")
	}

	return s.sshSession.Wait()
}

// Close closes the session
func (s *Session) Close() error {
	if s.sshSession != nil {
		return s.sshSession.Close()
	}
	return nil
}

// StdinPipe returns a pipe that will be connected to the session's stdin
func (s *Session) StdinPipe() (io.WriteCloser, error) {
	if s.sshSession == nil {
		return nil, fmt.Errorf("session is closed")
	}
	return s.sshSession.StdinPipe()
}

// StdoutPipe returns a pipe that will be connected to the session's stdout
func (s *Session) StdoutPipe() (io.Reader, error) {
	if s.sshSession == nil {
		return nil, fmt.Errorf("session is closed")
	}
	return s.sshSession.StdoutPipe()
}

// StderrPipe returns a pipe that will be connected to the session's stderr
func (s *Session) StderrPipe() (io.Reader, error) {
	if s.sshSession == nil {
		return nil, fmt.Errorf("session is closed")
	}
	return s.sshSession.StderrPipe()
}

// SessionManager manages multiple SSH sessions
type SessionManager struct {
	client   *Client
	sessions []*Session
}

// NewSessionManager creates a new session manager
func NewSessionManager(client *Client) *SessionManager {
	return &SessionManager{
		client:   client,
		sessions: make([]*Session, 0),
	}
}

// CreateSession creates and tracks a new session
func (sm *SessionManager) CreateSession(cfg *SessionConfig) (*Session, error) {
	session, err := sm.client.NewSessionWithConfig(cfg)
	if err != nil {
		return nil, err
	}

	sm.sessions = append(sm.sessions, session)
	return session, nil
}

// CloseAll closes all tracked sessions
func (sm *SessionManager) CloseAll() error {
	var lastErr error

	for _, session := range sm.sessions {
		if err := session.Close(); err != nil {
			lastErr = err
		}
	}

	sm.sessions = nil
	return lastErr
}

// Count returns the number of active sessions
func (sm *SessionManager) Count() int {
	return len(sm.sessions)
}

// ExecuteWithTimeout executes a command with a timeout
func ExecuteWithTimeout(client *Client, command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return client.RunCommand(ctx, command)
}
