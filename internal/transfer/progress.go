package transfer

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ProgressBar wraps a progress bar for file transfers
type ProgressBar struct {
	bar       *progressbar.ProgressBar
	startTime time.Time
	lastBytes int64
}

// NewProgressBar creates a new progress bar
func NewProgressBar(totalBytes int64, description string) *ProgressBar {
	bar := progressbar.DefaultBytes(
		totalBytes,
		description,
	)

	return &ProgressBar{
		bar:       bar,
		startTime: time.Now(),
	}
}

// Update updates the progress bar
func (p *ProgressBar) Update(current int64) {
	if p.bar != nil {
		p.bar.Set64(current)
		p.lastBytes = current
	}
}

// Finish completes the progress bar
func (p *ProgressBar) Finish() {
	if p.bar != nil {
		p.bar.Finish()
	}
}

// GetElapsedTime returns the elapsed time since start
func (p *ProgressBar) GetElapsedTime() time.Duration {
	return time.Since(p.startTime)
}

// GetAverageSpeed returns the average transfer speed in bytes/second
func (p *ProgressBar) GetAverageSpeed() int64 {
	elapsed := p.GetElapsedTime()
	if elapsed.Seconds() == 0 {
		return 0
	}
	return int64(float64(p.lastBytes) / elapsed.Seconds())
}

// ProgressTracker tracks progress across multiple files
type ProgressTracker struct {
	totalFiles       int
	completedFiles   int
	totalBytes       int64
	transferredBytes int64
	currentFile      string
	startTime        time.Time
	bar              *ProgressBar
}

// NewProgressTracker creates a new progress tracker
func NewProgressTracker(totalFiles int, totalBytes int64) *ProgressTracker {
	return &ProgressTracker{
		totalFiles: totalFiles,
		totalBytes: totalBytes,
		startTime:  time.Now(),
		bar:        NewProgressBar(totalBytes, "Transferring"),
	}
}

// Update updates the tracker with current progress
func (pt *ProgressTracker) Update(info ProgressInfo) {
	if info.CurrentFile != pt.currentFile {
		pt.currentFile = info.CurrentFile
		if pt.completedFiles < pt.totalFiles {
			pt.completedFiles++
		}
	}

	pt.transferredBytes = info.TransferredBytes

	if pt.bar != nil {
		pt.bar.Update(pt.transferredBytes)
	}
}

// FileCompleted marks a file as completed
func (pt *ProgressTracker) FileCompleted() {
	pt.completedFiles++
}

// Finish completes the progress tracker
func (pt *ProgressTracker) Finish() {
	if pt.bar != nil {
		pt.bar.Finish()
	}
}

// GetStats returns progress statistics
func (pt *ProgressTracker) GetStats() ProgressStats {
	elapsed := time.Since(pt.startTime)
	var speed int64
	if elapsed.Seconds() > 0 {
		speed = int64(float64(pt.transferredBytes) / elapsed.Seconds())
	}

	var eta time.Duration
	if speed > 0 && pt.totalBytes > pt.transferredBytes {
		remaining := pt.totalBytes - pt.transferredBytes
		etaSeconds := float64(remaining) / float64(speed)
		eta = time.Duration(etaSeconds) * time.Second
	}

	var percentage float64
	if pt.totalBytes > 0 {
		percentage = float64(pt.transferredBytes) / float64(pt.totalBytes) * 100
	}

	return ProgressStats{
		TotalFiles:       pt.totalFiles,
		CompletedFiles:   pt.completedFiles,
		TotalBytes:       pt.totalBytes,
		TransferredBytes: pt.transferredBytes,
		CurrentFile:      pt.currentFile,
		Speed:            speed,
		Percentage:       percentage,
		Elapsed:          elapsed,
		ETA:              eta,
	}
}

// ProgressStats contains transfer statistics
type ProgressStats struct {
	TotalFiles       int
	CompletedFiles   int
	TotalBytes       int64
	TransferredBytes int64
	CurrentFile      string
	Speed            int64
	Percentage       float64
	Elapsed          time.Duration
	ETA              time.Duration
}

// String returns a string representation of the stats
func (ps ProgressStats) String() string {
	return fmt.Sprintf(
		"Files: %d/%d | Bytes: %s/%s (%.1f%%) | Speed: %s | Elapsed: %s | ETA: %s",
		ps.CompletedFiles,
		ps.TotalFiles,
		FormatBytes(ps.TransferredBytes),
		FormatBytes(ps.TotalBytes),
		ps.Percentage,
		FormatSpeed(ps.Speed),
		ps.formatDuration(ps.Elapsed),
		ps.formatDuration(ps.ETA),
	)
}

// formatDuration formats a duration into a readable string
func (ps ProgressStats) formatDuration(d time.Duration) string {
	if d == 0 {
		return "--:--:--"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
