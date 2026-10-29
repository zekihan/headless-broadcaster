package config

import (
	"net"
	"testing"
)

func TestWebConfigAddress(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "fe80::1%lo", "localhost", ""} {
		t.Run(host, func(t *testing.T) {
			cfg := WebConfig{Host: host, Port: 8080}
			address := cfg.Address()

			gotHost, gotPort, err := net.SplitHostPort(address)
			if err != nil {
				t.Fatalf(
					"configured host %q produces invalid listener address %q: %v",
					host,
					address,
					err,
				)
			}

			if gotHost != host || gotPort != "8080" {
				t.Fatalf("listener address changed host or port: %q, %q", gotHost, gotPort)
			}
		})
	}
}
