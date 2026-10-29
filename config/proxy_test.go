package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProxyConfigValidation(t *testing.T) {
	for _, limit := range []int64{-1, 0, 1<<63 - 1} {
		for _, request := range []bool{false, true} {
			cfg := GetDefaultConfig().ProxyConfig
			if request {
				cfg.MaxRequestBodyBytes = limit
			} else {
				cfg.MaxResponseBodyBytes = limit
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("accepted invalid limit %d", limit)
			}
		}
	}

	cfg := GetDefaultConfig().ProxyConfig

	err := cfg.Validate()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.InsecureSkipVerify || cfg.TLSRootCAs() != nil {
		t.Fatal("defaults must verify using system trust")
	}

	cfg.CAFile = filepath.Join(t.TempDir(), "ca.pem")

	err = cfg.Validate()
	if err == nil {
		t.Fatal("accepted missing CA file")
	}

	err = os.WriteFile(cfg.CAFile, []byte("not a certificate"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	err = cfg.Validate()
	if err == nil {
		t.Fatal("accepted invalid CA file")
	}
}

func TestProxyConfigFileAndEnvironment(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	file := filepath.Join(t.TempDir(), "config.yaml")

	err := os.WriteFile(file, []byte(`proxy:
  maxRequestBodyBytes: 123
  maxResponseBodyBytes: 456
  insecureSkipVerify: true
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	t.Setenv("CONFIG_PATH", file)

	cfg := GetConfigFromFile().ProxyConfig
	if cfg.MaxRequestBodyBytes != 123 || cfg.MaxResponseBodyBytes != 456 ||
		!cfg.InsecureSkipVerify {
		t.Fatalf("YAML: %+v", cfg)
	}

	t.Setenv("HB__PROXY_MAX_REQUEST_BODY_BYTES", "789")
	t.Setenv("HB__PROXY_MAX_RESPONSE_BODY_BYTES", "987")
	t.Setenv("HB__PROXY_INSECURE_SKIP_VERIFY", "false")

	cfg = GetConfig().ProxyConfig
	if cfg.MaxRequestBodyBytes != 789 || cfg.MaxResponseBodyBytes != 987 || cfg.InsecureSkipVerify {
		t.Fatalf("env: %+v", cfg)
	}
}
