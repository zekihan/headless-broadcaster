//nolint:forbidigo // This package configures log packages.
package config

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/sethvargo/go-envconfig/v2"
	"gopkg.in/yaml.v3"
)

const (
	EnvKeyPrefix      = "HB__"
	defaultConfigFile = "config.yaml"
	defaultPort       = 8080
)

var config *Config

type Config struct {
	WebConfig   WebConfig   `yaml:"web"   env:", prefix=WEB_"`
	LogConfig   LogConfig   `yaml:"log"   env:", prefix=LOG_"`
	ProxyConfig ProxyConfig `yaml:"proxy" env:", prefix=PROXY_"`
}

const (
	DefaultMaxRequestBodyBytes  int64 = 1 << 20
	DefaultMaxResponseBodyBytes int64 = 4 << 20
)

var (
	errInvalidProxyConfig = errors.New("proxy body limits must be positive and less than MaxInt64")
	errInvalidCAFile      = errors.New("proxy CA file contains no certificates")
)

type ProxyConfig struct {
	rootCAs              *x509.CertPool
	CAFile               string `yaml:"caFile"               env:"CA_FILE"`
	MaxRequestBodyBytes  int64  `yaml:"maxRequestBodyBytes"  env:"MAX_REQUEST_BODY_BYTES"`
	MaxResponseBodyBytes int64  `yaml:"maxResponseBodyBytes" env:"MAX_RESPONSE_BODY_BYTES"`
	InsecureSkipVerify   bool   `yaml:"insecureSkipVerify"   env:"INSECURE_SKIP_VERIFY"`
}

func (cfg *ProxyConfig) TLSRootCAs() *x509.CertPool { return cfg.rootCAs }

// Validate loads additional trust without replacing the system roots.
func (cfg *ProxyConfig) Validate() error {
	const maxInt64 = int64(1<<63 - 1)
	if cfg.MaxRequestBodyBytes <= 0 || cfg.MaxRequestBodyBytes == maxInt64 ||
		cfg.MaxResponseBodyBytes <= 0 || cfg.MaxResponseBodyBytes == maxInt64 {
		return errInvalidProxyConfig
	}

	if cfg.CAFile == "" {
		return nil
	}

	pem, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return fmt.Errorf("read proxy CA file: %w", err)
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("load system trust: %w", err)
	}

	if !roots.AppendCertsFromPEM(pem) {
		return errInvalidCAFile
	}

	cfg.rootCAs = roots

	return nil
}

type WebConfig struct {
	Host   string `yaml:"host"   env:"HOST"`
	APIKey string `yaml:"apiKey" env:"API_KEY"`
	Port   int    `yaml:"port"   env:"PORT"`
}

type LogConfig struct {
	LevelStr  string     `yaml:"level"     env:"LEVEL"`
	Level     slog.Level `yaml:"-"`
	AddSource bool       `yaml:"addSource" env:"ADD_SOURCE"`
}

func GetConfig() *Config {
	if config == nil {
		cfg := GetConfigFromFile()

		ctx := context.Background()

		err := envconfig.ProcessWith(ctx, &envconfig.Config{
			Target:           cfg,
			Lookuper:         envconfig.PrefixLookuper(EnvKeyPrefix, envconfig.OsLookuper()),
			DefaultOverwrite: true,
		})
		if err != nil {
			log.Fatalf("Failed to process env config: %v", err)
		}

		if cfg.LogConfig.LevelStr != "" {
			level := slog.LevelInfo

			switch strings.ToLower(cfg.LogConfig.LevelStr) {
			case "debug":
				level = slog.LevelDebug
			case "info":
				level = slog.LevelInfo
			case "warn", "warning":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			default:
				slog.ErrorContext(
					ctx,
					"Invalid log level in config, defaulting to info",
					slog.String("level", cfg.LogConfig.LevelStr),
				)
			}

			cfg.LogConfig.Level = level
		}

		err = cfg.ProxyConfig.Validate()
		if err != nil {
			log.Fatalf("Invalid proxy config: %v", err)
		}

		config = cfg
	}

	return config
}

func GetConfigFromFile() *Config {
	newConfig := GetDefaultConfig()

	file, err := os.Open(getConfigFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file not found at %s, using default config", getConfigFilePath())

			return newConfig
		}

		log.Fatalf("Failed to open config file: %v", err)
	}
	defer func(file *os.File) {
		fileErr := file.Close()
		if fileErr != nil {
			fmt.Printf("Failed to close config file: %v", fileErr)
		}
	}(file)

	content, err := io.ReadAll(file)
	if err != nil {
		fileErr := file.Close()
		if fileErr != nil {
			fmt.Printf("Failed to close config file after read error: %v", fileErr)
		}

		log.Fatalf("Failed to read config file: %v", fileErr) //nolint:gocritic
	}

	err = yaml.Unmarshal(content, newConfig)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	return newConfig
}

func getConfigFilePath() string {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath != "" {
		return configPath
	}

	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("%v", err)
		}

		configDir = cwd
	}

	configPath = path.Join(configDir, defaultConfigFile)

	return configPath
}

func GetDefaultConfig() *Config {
	return &Config{
		ProxyConfig: ProxyConfig{
			MaxRequestBodyBytes:  DefaultMaxRequestBodyBytes,
			MaxResponseBodyBytes: DefaultMaxResponseBodyBytes,
		},
		WebConfig: WebConfig{
			Host:   "127.0.0.1",
			Port:   defaultPort,
			APIKey: "",
		},
		LogConfig: LogConfig{
			Level:     slog.LevelWarn,
			AddSource: false,
		},
	}
}

// Reset clears the cached configuration, used for testing.
func Reset() {
	config = nil
}
