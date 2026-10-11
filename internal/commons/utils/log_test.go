package utils

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestLoggerInitializesWithNilWriter(t *testing.T) {
	cfg := &LoggerConfig{
		Level:     slog.LevelWarn,
		AddSource: true,
		Writer:    nil,
	}

	InitLogger(cfg)
}

func TestLoggerUsesDefaultConfigWhenNilPassed(t *testing.T) {
	InitLogger(nil)
}

func TestLoggerInitializesWithCustomLevelAndSource(t *testing.T) {
	cfg := &LoggerConfig{
		Level:     slog.LevelError,
		AddSource: true,
		Writer:    os.Stdout,
	}

	InitLogger(cfg)
}

func TestLoggerDoesNotWriteANSIColorCodes(t *testing.T) {
	var output bytes.Buffer
	InitLogger(&LoggerConfig{Writer: &output})

	slog.Info("Test log message")

	if strings.Contains(output.String(), "\x1b[") {
		t.Fatalf("log output contains ANSI color codes: %q", output.String())
	}
}

func TestLoggerFailsWithInvalidWriter(t *testing.T) {
	cfg := &LoggerConfig{
		Level:     slog.LevelInfo,
		AddSource: false,
		Writer:    nil,
	}

	// Simulate invalid writer by passing a closed file
	file, _ := os.CreateTemp(t.TempDir(), "test")
	_ = file.Close()

	cfg.Writer = file

	InitLogger(cfg)
}
