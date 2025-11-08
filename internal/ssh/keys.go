package ssh

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// KeyType represents the type of SSH key
type KeyType string

const (
	// KeyTypeRSA represents RSA keys
	KeyTypeRSA KeyType = "rsa"

	// KeyTypeED25519 represents ED25519 keys
	KeyTypeED25519 KeyType = "ed25519"
)

// GenerateKeyPair generates an SSH key pair
func GenerateKeyPair(keyType KeyType, bits int) (privateKey, publicKey []byte, err error) {
	switch keyType {
	case KeyTypeRSA:
		return generateRSAKeyPair(bits)
	case KeyTypeED25519:
		return generateED25519KeyPair()
	default:
		return nil, nil, fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// generateRSAKeyPair generates an RSA key pair
func generateRSAKeyPair(bits int) ([]byte, []byte, error) {
	if bits == 0 {
		bits = 4096
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Generate public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(pub)

	return privateKeyPEM, publicKeyBytes, nil
}

// generateED25519KeyPair generates an ED25519 key pair
func generateED25519KeyPair() ([]byte, []byte, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ED25519 key: %w", err)
	}

	// Encode private key
	privateKeyBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := pem.EncodeToMemory(privateKeyBytes)

	// Generate public key
	pub, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate public key: %w", err)
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(pub)

	return privateKeyPEM, publicKeyBytes, nil
}

// SaveKeyPair saves a key pair to files
func SaveKeyPair(privateKeyPath, publicKeyPath string, privateKey, publicKey []byte) error {
	// Ensure directory exists
	dir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save private key with restricted permissions
	if err := os.WriteFile(privateKeyPath, privateKey, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key
	if err := os.WriteFile(publicKeyPath, publicKey, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// DeployPublicKey deploys a public key to a remote host using SFTP for security
func DeployPublicKey(ctx context.Context, cfg *Config, publicKey []byte) error {
	client, err := NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Create SFTP client for secure file operations
	sftpClient, err := sftp.NewClient(client.GetClient())
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Ensure .ssh directory exists with correct permissions
	sshDir := ".ssh"
	if err := sftpClient.MkdirAll(sshDir); err != nil {
		// Directory might already exist, try to continue
		if !os.IsExist(err) {
			return fmt.Errorf("failed to create .ssh directory: %w", err)
		}
	}

	// Set .ssh directory permissions (0700)
	if err := sftpClient.Chmod(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to set .ssh directory permissions: %w", err)
	}

	// Open authorized_keys file for append
	authKeysPath := filepath.Join(sshDir, "authorized_keys")
	f, err := sftpClient.OpenFile(authKeysPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return fmt.Errorf("failed to open authorized_keys: %w", err)
	}
	defer f.Close()

	// Write public key atomically
	if _, err := f.Write(publicKey); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Ensure newline at end if not present
	if len(publicKey) > 0 && publicKey[len(publicKey)-1] != '\n' {
		if _, err := f.Write([]byte("\n")); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	// Set correct permissions on authorized_keys (0600)
	if err := sftpClient.Chmod(authKeysPath, 0600); err != nil {
		return fmt.Errorf("failed to set authorized_keys permissions: %w", err)
	}

	return nil
}

// GetDefaultKeyPath returns the default SSH key path for a given key type
func GetDefaultKeyPath(keyType KeyType) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	sshDir := filepath.Join(homeDir, ".ssh")

	switch keyType {
	case KeyTypeRSA:
		return filepath.Join(sshDir, "id_rsa"), nil
	case KeyTypeED25519:
		return filepath.Join(sshDir, "id_ed25519"), nil
	default:
		return "", fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// KeyExists checks if an SSH key exists at the given path
func KeyExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// ValidateKeyPair validates that a private/public key pair is valid
func ValidateKeyPair(privateKeyPath, publicKeyPath string) error {
	// Check if files exist
	if !KeyExists(privateKeyPath) {
		return fmt.Errorf("private key not found: %s", privateKeyPath)
	}

	if !KeyExists(publicKeyPath) {
		return fmt.Errorf("public key not found: %s", publicKeyPath)
	}

	// Try to load private key
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	_, err = ssh.ParsePrivateKey(privateKeyData)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Try to load public key
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	_, _, _, _, err = ssh.ParseAuthorizedKey(publicKeyData)
	if err != nil {
		return fmt.Errorf("invalid public key: %w", err)
	}

	return nil
}
