package transfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"
)

// SFTPTransfer implements file transfer using SFTP
type SFTPTransfer struct {
	config           *TransferConfig
	progressCallback ProgressCallback
}

// NewSFTPTransfer creates a new SFTP-based transfer
func NewSFTPTransfer(cfg *TransferConfig) *SFTPTransfer {
	return &SFTPTransfer{
		config: cfg,
	}
}

// SetProgressCallback sets the progress callback
func (s *SFTPTransfer) SetProgressCallback(callback ProgressCallback) {
	s.progressCallback = callback
}

// Execute performs the SFTP transfer
func (s *SFTPTransfer) Execute(ctx context.Context) error {
	if s.config.SSHClient == nil || !s.config.SSHClient.IsConnected() {
		return fmt.Errorf("SSH client not connected")
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(s.config.SSHClient.GetClient())
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Execute transfer based on direction
	if s.config.Direction == DirectionPush {
		return s.push(ctx, sftpClient)
	}
	return s.pull(ctx, sftpClient)
}

// push transfers files from local to remote
func (s *SFTPTransfer) push(ctx context.Context, client *sftp.Client) error {
	srcInfo, err := os.Stat(s.config.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	if srcInfo.IsDir() {
		return s.pushDirectory(ctx, client, s.config.SourcePath, s.config.DestPath)
	}
	return s.pushFile(ctx, client, s.config.SourcePath, s.config.DestPath)
}

// pull transfers files from remote to local
func (s *SFTPTransfer) pull(ctx context.Context, client *sftp.Client) error {
	srcInfo, err := client.Stat(s.config.SourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat remote source: %w", err)
	}

	if srcInfo.IsDir() {
		return s.pullDirectory(ctx, client, s.config.SourcePath, s.config.DestPath)
	}
	return s.pullFile(ctx, client, s.config.SourcePath, s.config.DestPath)
}

// pushFile transfers a single file to remote
func (s *SFTPTransfer) pushFile(ctx context.Context, client *sftp.Client, localPath, remotePath string) error {
	if s.config.DryRun {
		s.notifyProgress(ProgressInfo{
			CurrentFile: localPath,
			Message:     fmt.Sprintf("Would transfer: %s -> %s", localPath, remotePath),
		})
		return nil
	}

	// Open local file
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer localFile.Close()

	// Get file size for progress
	stat, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Create remote directory if needed
	remoteDir := path.Dir(remotePath)
	if err := client.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Create remote file
	remoteFile, err := client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Copy with progress
	return s.copyWithProgress(ctx, remoteFile, localFile, stat.Size(), localPath)
}

// pullFile transfers a single file from remote
func (s *SFTPTransfer) pullFile(ctx context.Context, client *sftp.Client, remotePath, localPath string) error {
	if s.config.DryRun {
		s.notifyProgress(ProgressInfo{
			CurrentFile: remotePath,
			Message:     fmt.Sprintf("Would transfer: %s -> %s", remotePath, localPath),
		})
		return nil
	}

	// Open remote file
	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Get file size for progress
	stat, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat remote file: %w", err)
	}

	// Create local directory if needed
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy with progress
	return s.copyWithProgress(ctx, localFile, remoteFile, stat.Size(), remotePath)
}

// pushDirectory recursively transfers a directory to remote
func (s *SFTPTransfer) pushDirectory(ctx context.Context, client *sftp.Client, localPath, remotePath string) error {
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		remoteDest := filepath.Join(remotePath, relPath)

		if info.IsDir() {
			if !s.config.DryRun {
				return client.MkdirAll(remoteDest)
			}
			return nil
		}

		return s.pushFile(ctx, client, path, remoteDest)
	})
}

// pullDirectory recursively transfers a directory from remote
func (s *SFTPTransfer) pullDirectory(ctx context.Context, client *sftp.Client, remotePath, localPath string) error {
	walker := client.Walk(remotePath)

	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		path := walker.Path()
		info := walker.Stat()

		// Calculate relative path
		relPath, err := filepath.Rel(remotePath, path)
		if err != nil {
			return err
		}

		localDest := filepath.Join(localPath, relPath)

		if info.IsDir() {
			if !s.config.DryRun {
				if err := os.MkdirAll(localDest, 0755); err != nil {
					return err
				}
			}
			continue
		}

		if err := s.pullFile(ctx, client, path, localDest); err != nil {
			return err
		}
	}

	return nil
}

// copyWithProgress copies data with progress reporting
func (s *SFTPTransfer) copyWithProgress(ctx context.Context, dst io.Writer, src io.Reader, total int64, filename string) error {
	var written int64
	buf := make([]byte, 32*1024)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)

				// Report progress
				s.notifyProgress(ProgressInfo{
					TotalBytes:       total,
					TransferredBytes: written,
					CurrentFile:      filename,
				})
			}
			if ew != nil {
				return ew
			}
			if nr != nw {
				return io.ErrShortWrite
			}
		}
		if er != nil {
			if er != io.EOF {
				return er
			}
			break
		}
	}

	return nil
}

// notifyProgress sends progress information to the callback
func (s *SFTPTransfer) notifyProgress(info ProgressInfo) {
	if s.progressCallback != nil {
		s.progressCallback(info)
	}
}
