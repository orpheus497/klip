// Package logger provides structured logging for klip
// Copyright (c) 2025 orpheus497
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// Logger wraps slog.Logger with klip-specific functionality
type Logger struct {
	slog   *slog.Logger
	level  slog.Level
	output io.Writer
}

// New creates a new logger with the specified verbosity
func New(verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		slog:   slog.New(handler),
		level:  level,
		output: os.Stderr,
	}
}

// NewWithJSON creates a logger with JSON output format
func NewWithJSON(verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		slog:   slog.New(handler),
		level:  level,
		output: os.Stderr,
	}
}

// NewWithOutput creates a logger with custom output writer
func NewWithOutput(w io.Writer, verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		slog:   slog.New(handler),
		level:  level,
		output: w,
	}
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level slog.Level) {
	l.level = level
	handler := slog.NewTextHandler(l.output, &slog.HandlerOptions{
		Level: level,
	})
	l.slog = slog.New(handler)
}

// SetOutput changes the output writer
func (l *Logger) SetOutput(w io.Writer) {
	l.output = w
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: l.level,
	})
	l.slog = slog.New(handler)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.slog.Debug(msg, args...)
}

// Info logs an informational message
func (l *Logger) Info(msg string, args ...any) {
	l.slog.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.slog.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.slog.Error(msg, args...)
}

// DebugContext logs a debug message with context
func (l *Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.slog.DebugContext(ctx, msg, args...)
}

// InfoContext logs an informational message with context
func (l *Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.slog.InfoContext(ctx, msg, args...)
}

// WarnContext logs a warning message with context
func (l *Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.slog.WarnContext(ctx, msg, args...)
}

// ErrorContext logs an error message with context
func (l *Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.slog.ErrorContext(ctx, msg, args...)
}

// With returns a new logger with the given attributes
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slog:   l.slog.With(args...),
		level:  l.level,
		output: l.output,
	}
}

// WithGroup returns a new logger with the given group name
func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{
		slog:   l.slog.WithGroup(name),
		level:  l.level,
		output: l.output,
	}
}

// GetLogFilePath returns the XDG-compliant path for log files
func GetLogFilePath(filename string) (string, error) {
	logDir := filepath.Join(xdg.StateHome, "klip", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(logDir, filename), nil
}

// NewFileLogger creates a logger that writes to a file
func NewFileLogger(filename string, verbose bool) (*Logger, error) {
	logPath, err := GetLogFilePath(filename)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		slog:   slog.New(handler),
		level:  level,
		output: file,
	}, nil
}

// Default returns a default logger instance
func Default() *Logger {
	return New(false)
}
