package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// setup resets global state and schedules cleanup of env vars.
func setup(t *testing.T) {
	t.Helper()
	Reset()
}

func TestGetDefaultConfig(t *testing.T) {
	setup(t)

	cfg := GetDefaultConfig()
	if cfg.WebConfig.Host != "127.0.0.1" {
		t.Errorf("expected default Host 127.0.0.1, got %s", cfg.WebConfig.Host)
	}

	if cfg.WebConfig.Port != 8080 {
		t.Errorf("expected default Port 8080, got %d", cfg.WebConfig.Port)
	}

	if cfg.WebConfig.APIKey != "" {
		t.Errorf("expected default APIKey empty, got %s", cfg.WebConfig.APIKey)
	}

	if cfg.LogConfig.Level != slog.LevelWarn {
		t.Errorf("expected default log level Warn, got %d", cfg.LogConfig.Level)
	}

	if cfg.LogConfig.AddSource {
		t.Errorf("expected default AddSource false, got true")
	}
}

func TestGetConfigFilePath(t *testing.T) {
	setup(t)

	t.Run("CONFIG_PATH", func(t *testing.T) {
		custom := "/tmp/custom.yaml"
		t.Setenv("CONFIG_PATH", custom)

		got := getConfigFilePath()
		if got != custom {
			t.Errorf("expected %s, got %s", custom, got)
		}
	})

	t.Run("CONFIG_DIR", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("CONFIG_DIR", dir)

		got := getConfigFilePath()

		want := filepath.Join(dir, defaultConfigFile)
		if got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	})

	t.Run("Default cwd", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CONFIG_DIR", "")
		t.Setenv("CONFIG_PATH", "")
		t.Chdir(tmpDir)

		got := getConfigFilePath()

		want := filepath.Join(tmpDir, defaultConfigFile)
		if got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
	})
}

func TestGetConfigFromFile(t *testing.T) {
	setup(t)

	content := `
web:
  host: "localhost"
  port: 9000
  apiKey: "abc123"
log:
  level: "info"
  addSource: true
  path: "/var/log/app.log"
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "cfg*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	err = tmpFile.Close()
	if err != nil {
		t.Fatalf("failed to close temp file: %v", err)
	}

	t.Setenv("CONFIG_PATH", tmpFile.Name())

	cfg := GetConfigFromFile()

	if cfg.WebConfig.Host != "localhost" {
		t.Errorf("expected host localhost, got %s", cfg.WebConfig.Host)
	}

	if cfg.WebConfig.Port != 9000 {
		t.Errorf("expected port 9000, got %d", cfg.WebConfig.Port)
	}

	if cfg.WebConfig.APIKey != "abc123" {
		t.Errorf("expected api_key abc123, got %s", cfg.WebConfig.APIKey)
	}

	if cfg.LogConfig.LevelStr != "info" {
		t.Errorf("expected level string info, got %s", cfg.LogConfig.LevelStr)
	}

	if !cfg.LogConfig.AddSource {
		t.Errorf("expected add_method_name true, got %t", cfg.LogConfig.AddSource)
	}
}

func TestGetConfigEnvOverrides(t *testing.T) {
	setup(t)

	content := `
web:
  host: "fromfile"
  port: 1234
  apiKey: "filekey"
log:
  level: "warn"
  addSource: false
  path: "/file.log"
`

	tmpFile, err := os.CreateTemp(t.TempDir(), "cfg*.yaml")
	if err != nil {
		t.Fatal(err)
	}

	_, err = tmpFile.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	err = tmpFile.Close()
	if err != nil {
		return
	}

	t.Setenv("CONFIG_PATH", tmpFile.Name())
	t.Setenv(EnvKeyPrefix+"WEB_HOST", "envhost")
	t.Setenv(EnvKeyPrefix+"WEB_PORT", "4321")
	t.Setenv(EnvKeyPrefix+"WEB_API_KEY", "envkey")
	t.Setenv(EnvKeyPrefix+"LOG_LEVEL", "error")
	t.Setenv(EnvKeyPrefix+"LOG_ADD_SOURCE", "true")
	t.Setenv(EnvKeyPrefix+"LOG_PATH", "/env.log")

	cfg := GetConfig()
	if cfg.WebConfig.Host != "envhost" {
		t.Errorf("expected env override host envhost, got %s", cfg.WebConfig.Host)
	}

	if cfg.WebConfig.Port != 4321 {
		t.Errorf("expected env override port 4321, got %d", cfg.WebConfig.Port)
	}

	if cfg.WebConfig.APIKey != "envkey" {
		t.Errorf("expected env override api_key envkey, got %s", cfg.WebConfig.APIKey)
	}

	if cfg.LogConfig.Level != slog.LevelError {
		t.Errorf("expected log level error, got %d", cfg.LogConfig.Level)
	}

	if !cfg.LogConfig.AddSource {
		t.Errorf("expected AddSource true, got %t", cfg.LogConfig.AddSource)
	}
}
