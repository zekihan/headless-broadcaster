package utils

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

type LoggerConfig struct {
	Writer    io.Writer
	Level     slog.Level
	AddSource bool
}

// InitLogger initializes the global slog logger.
// Passing nil uses sane defaults.
func InitLogger(cfg *LoggerConfig) {
	c := resolveLoggerConfig(cfg)

	slog.SetDefault(slog.New(tint.NewTextHandler(c.Writer, &tint.Options{
		Level:      c.Level,
		AddSource:  c.AddSource,
		TimeFormat: time.RFC3339,
		NoColor:    !writerSupportsColor(c.Writer),
	})))
}

func writerSupportsColor(writer io.Writer) bool {
	file, ok := writer.(interface{ Fd() uintptr })
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}

func resolveLoggerConfig(cfg *LoggerConfig) LoggerConfig {
	defaultCfg := LoggerConfig{
		Level:     slog.LevelInfo,
		AddSource: false,
		Writer:    os.Stdout,
	}

	if cfg == nil {
		return defaultCfg
	}

	if cfg.Writer != nil {
		defaultCfg.Writer = cfg.Writer
	}

	defaultCfg.Level = cfg.Level
	defaultCfg.AddSource = cfg.AddSource

	return defaultCfg
}
