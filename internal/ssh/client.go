// Package ssh provides SSH client functionality for klip
package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Client wraps SSH client functionality
type Client struct {
	config *ssh.ClientConfig
	client *ssh.Client
	host   string
	port   int
}

// Config contains SSH client configuration
type Config struct {
	Host        string
	Port        int
	User        string
	KeyPath     string
	Password    string
	UsePassword bool
	Timeout     time.Duration
}

// NewClient creates a new SSH client
func NewClient(cfg *Config) (*Client, error) {
	if cfg.Port == 0 {
		cfg.Port = 22
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	authMethods := []ssh.AuthMethod{}

	// Try key-based authentication first
	if !cfg.UsePassword && cfg.KeyPath != "" {
		keyAuth, err := publicKeyAuth(cfg.KeyPath)
		if err == nil {
			authMethods = append(authMethods, keyAuth)
		}
	}

	// Try default SSH keys if no specific key provided
	if len(authMethods) == 0 && !cfg.UsePassword {
		if defaultAuth := tryDefaultKeys(); defaultAuth != nil {
			authMethods = append(authMethods, defaultAuth...)
		}
	}

	// Add password authentication if requested or as fallback
	if cfg.UsePassword && cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	// Add keyboard-interactive for password prompt
	if cfg.UsePassword || len(authMethods) == 0 {
		authMethods = append(authMethods, ssh.KeyboardInteractive(keyboardInteractiveChallenge))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: NewHostKeyCallback(),
		Timeout:         cfg.Timeout,
	}

	return &Client{
		config: clientConfig,
		host:   cfg.Host,
		port:   cfg.Port,
	}, nil
}

// Connect establishes the SSH connection
func (c *Client) Connect(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", c.host, c.port)

	// Create a dialer with context support
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	// Wrap connection to support context cancellation
	connWithContext := &contextConn{
		Conn: conn,
		ctx:  ctx,
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(connWithContext, address, c.config)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create SSH connection: %w", err)
	}

	c.client = ssh.NewClient(sshConn, chans, reqs)
	return nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	return c.client != nil
}

// GetClient returns the underlying SSH client
func (c *Client) GetClient() *ssh.Client {
	return c.client
}

// NewSession creates a new SSH session
func (c *Client) NewSession() (*ssh.Session, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected")
	}
	return c.client.NewSession()
}

// RunCommand executes a command and returns the output
func (c *Client) RunCommand(ctx context.Context, command string) (string, error) {
	session, err := c.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	// Set up context cancellation
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			session.Close()
		case <-done:
		}
	}()
	defer close(done)

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

// InteractiveShell starts an interactive SSH shell
func (c *Client) InteractiveShell() error {
	session, err := c.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// Set up terminal
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("stdin is not a terminal")
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	// Get terminal dimensions
	width, height, err := term.GetSize(fd)
	if err != nil {
		width, height = 80, 24
	}

	// Request pseudo terminal
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", height, width, modes); err != nil {
		return fmt.Errorf("failed to request pty: %w", err)
	}

	// Connect stdin/stdout/stderr
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Start shell
	if err := session.Shell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Wait for session to complete
	return session.Wait()
}

// publicKeyAuth creates SSH auth from private key file
func publicKeyAuth(keyPath string) (ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// tryDefaultKeys tries to load default SSH keys
func tryDefaultKeys() []ssh.AuthMethod {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	sshDir := filepath.Join(homeDir, ".ssh")
	defaultKeys := []string{"id_rsa", "id_ed25519", "id_ecdsa", "id_dsa"}

	var methods []ssh.AuthMethod
	for _, keyFile := range defaultKeys {
		keyPath := filepath.Join(sshDir, keyFile)
		if auth, err := publicKeyAuth(keyPath); err == nil {
			methods = append(methods, auth)
		}
	}

	return methods
}

// keyboardInteractiveChallenge handles keyboard-interactive authentication
func keyboardInteractiveChallenge(user, instruction string, questions []string, echos []bool) ([]string, error) {
	answers := make([]string, len(questions))

	for i, question := range questions {
		fmt.Print(question)

		var answer string
		if echos[i] {
			fmt.Scanln(&answer)
		} else {
			// Read password without echo
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return nil, err
			}
			answer = string(passwordBytes)
			fmt.Println()
		}

		answers[i] = answer
	}

	return answers, nil
}

// contextConn wraps net.Conn to support context cancellation
type contextConn struct {
	net.Conn
	ctx context.Context
}

func (c *contextConn) Read(b []byte) (n int, err error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.Conn.Read(b)
	}
}

func (c *contextConn) Write(b []byte) (n int, err error) {
	select {
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		return c.Conn.Write(b)
	}
}

// CopyReader is a helper to copy from a reader with context support
func CopyReader(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	written := int64(0)
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return written, er
			}
			break
		}
	}
	return written, nil
}
