package config

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path"
	"strconv"
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

// Address returns the configured HTTP listener address.
func (cfg WebConfig) Address() string {
	return net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
}

type LogConfig struct {
	LevelStr  string     `yaml:"level"     env:"LEVEL"`
	Level     slog.Level `yaml:"-"`
	AddSource bool       `yaml:"addSource" env:"ADD_SOURCE"`
}

func GetConfig() (*Config, error) {
	if config == nil {
		cfg, err := GetConfigFromFile()
		if err != nil {
			return nil, err
		}

		ctx := context.Background()

		err = envconfig.ProcessWith(ctx, &envconfig.Config{
			Target:           cfg,
			Lookuper:         envconfig.PrefixLookuper(EnvKeyPrefix, envconfig.OsLookuper()),
			DefaultOverwrite: true,
		})
		if err != nil {
			return nil, fmt.Errorf("process env config: %w", err)
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
			return nil, fmt.Errorf("invalid proxy config: %w", err)
		}

		config = cfg
	}

	return config, nil
}

func GetConfigFromFile() (*Config, error) {
	newConfig := GetDefaultConfig()

	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info(
				"Config file not found, using default config",
				slog.String("path", configPath),
			)

			return newConfig, nil
		}

		return nil, fmt.Errorf("read config file: %w", err)
	}

	err = yaml.Unmarshal(content, newConfig)
	if err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	return newConfig, nil
}

func getConfigFilePath() (string, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath != "" {
		return configPath, nil
	}

	configDir := os.Getenv("CONFIG_DIR")
	if configDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}

		configDir = cwd
	}

	return path.Join(configDir, defaultConfigFile), nil
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
