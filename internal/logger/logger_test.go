// Package logger tests
// Copyright (c) 2025 orpheus497
package logger

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("creates logger with info level", func(t *testing.T) {
		logger := New(false)
		assert.NotNil(t, logger)
		assert.Equal(t, slog.LevelInfo, logger.level)
	})

	t.Run("creates logger with debug level when verbose", func(t *testing.T) {
		logger := New(true)
		assert.NotNil(t, logger)
		assert.Equal(t, slog.LevelDebug, logger.level)
	})
}

func TestNewWithJSON(t *testing.T) {
	logger := NewWithJSON(false)
	assert.NotNil(t, logger)
	assert.Equal(t, slog.LevelInfo, logger.level)
}

func TestNewWithOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, true)

	assert.NotNil(t, logger)
	assert.Equal(t, slog.LevelDebug, logger.level)
	assert.Equal(t, buf, logger.output)
}

func TestSetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)

	// Initially info level
	assert.Equal(t, slog.LevelInfo, logger.level)

	// Change to debug
	logger.SetLevel(slog.LevelDebug)
	assert.Equal(t, slog.LevelDebug, logger.level)

	// Change to warn
	logger.SetLevel(slog.LevelWarn)
	assert.Equal(t, slog.LevelWarn, logger.level)
}

func TestSetOutput(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	logger := NewWithOutput(buf1, false)
	assert.Equal(t, buf1, logger.output)

	logger.SetOutput(buf2)
	assert.Equal(t, buf2, logger.output)

	// Verify output goes to new writer
	logger.Info("test message")
	assert.Contains(t, buf2.String(), "test message")
	assert.Empty(t, buf1.String())
}

func TestDebug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, true)

	logger.Debug("debug message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "key=value")
}

func TestInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)

	logger.Info("info message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "key=value")
}

func TestWarn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)

	logger.Warn("warning message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "warning message")
	assert.Contains(t, output, "key=value")
}

func TestError(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)

	logger.Error("error message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "error message")
	assert.Contains(t, output, "key=value")
}

func TestDebugContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, true)
	ctx := context.Background()

	logger.DebugContext(ctx, "debug message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "debug message")
}

func TestInfoContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)
	ctx := context.Background()

	logger.InfoContext(ctx, "info message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "info message")
}

func TestWarnContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)
	ctx := context.Background()

	logger.WarnContext(ctx, "warning message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "warning message")
}

func TestErrorContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)
	ctx := context.Background()

	logger.ErrorContext(ctx, "error message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "error message")
}

func TestWith(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)

	childLogger := logger.With("component", "test")
	childLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "component=test")
}

func TestWithGroup(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false)

	groupLogger := logger.WithGroup("mygroup")
	groupLogger.Info("test message", "key", "value")

	output := buf.String()
	assert.Contains(t, output, "test message")
	assert.Contains(t, output, "mygroup")
}

func TestLogLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false) // Info level

	// Debug should not appear
	logger.Debug("debug message")
	assert.Empty(t, buf.String())

	// Info should appear
	buf.Reset()
	logger.Info("info message")
	assert.Contains(t, buf.String(), "info message")

	// Warn should appear
	buf.Reset()
	logger.Warn("warn message")
	assert.Contains(t, buf.String(), "warn message")

	// Error should appear
	buf.Reset()
	logger.Error("error message")
	assert.Contains(t, buf.String(), "error message")
}

func TestGetLogFilePath(t *testing.T) {
	path, err := GetLogFilePath("test.log")
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.True(t, strings.HasSuffix(path, "test.log"))
	assert.Contains(t, path, "klip")
	assert.Contains(t, path, "logs")
}

func TestDefault(t *testing.T) {
	logger := Default()
	assert.NotNil(t, logger)
	assert.Equal(t, slog.LevelInfo, logger.level)
}

func TestVerboseDebugLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, true) // Verbose mode

	logger.Debug("debug message")
	assert.Contains(t, buf.String(), "debug message")
}

func TestNonVerboseDebugFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(buf, false) // Non-verbose mode

	logger.Debug("debug message")
	assert.Empty(t, buf.String())
}
