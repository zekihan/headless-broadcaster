package proxy

import (
	"compress/gzip"
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zekihan/headless-broadcaster/config"
)

func proxyConfig(t *testing.T) {
	t.Helper()
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "absent.yaml"))
	config.Reset()
	t.Cleanup(config.Reset)
}

func TestOutboundTLSVerification(t *testing.T) {
	var serverName string

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverName = r.TLS.ServerName
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	cert := server.Certificate()

	caFile := filepath.Join(t.TempDir(), "ca.pem")

	err := os.WriteFile(
		caFile,
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}),
		0o600,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(cert.DNSNames) == 0 {
		t.Fatal("fixture needs DNS SAN")
	}

	for _, tc := range []struct {
		name, host                 string
		trusted, insecure, success bool
	}{
		{"untrusted", cert.DNSNames[0], false, false, false},
		{"trusted", cert.DNSNames[0], true, false, true},
		{"wrong host", "wrong.invalid", true, false, false},
		{"explicit opt out", "wrong.invalid", false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			proxyConfig(t)
			t.Setenv("HB__PROXY_INSECURE_SKIP_VERIFY", strconv.FormatBool(tc.insecure))

			if tc.trusted {
				t.Setenv("HB__PROXY_CA_FILE", caFile)
			}

			target, err := url.Parse(server.URL)
			if err != nil {
				t.Fatal(err)
			}

			target.Host = tc.host + ":" + target.Port()

			result, err := RequestToIP(t.Context(), "127.0.0.1", target, http.MethodGet, nil, nil)
			if err != nil {
				t.Fatal(err)
			}

			if (result.Err == "") != tc.success {
				t.Fatalf("%+v", result)
			}

			if tc.success && serverName != tc.host {
				t.Fatalf("SNI=%q want %q", serverName, tc.host)
			}
		})
	}
}

func TestResponseBodyBoundaries(t *testing.T) {
	proxyConfig(t)
	t.Setenv("HB__PROXY_MAX_RESPONSE_BODY_BYTES", "8")

	for _, size := range []int{0, 7, 8, 9} {
		for _, chunked := range []bool{false, true} {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if chunked {
						_ = http.NewResponseController(w).Flush()
					}

					_, _ = w.Write([]byte(strings.Repeat("x", size)))
				}),
			)
			target, _ := url.Parse(server.URL)
			result, err := RequestToIP(t.Context(), "127.0.0.1", target, http.MethodGet, nil, nil)

			server.Close()

			if err != nil {
				t.Fatal(err)
			}

			if (result.Err != "") != (size > 8) {
				t.Fatalf("size=%d chunked=%v: %+v", size, chunked, result)
			}

			if size > 8 && result.Result != nil {
				t.Fatal("oversized body retained")
			}

			if size <= 8 && result.Result != strings.Repeat("x", size) {
				t.Fatalf("boundary body lost: %+v", result)
			}
		}
	}
}

func TestBoundedResponseCancellation(t *testing.T) {
	proxyConfig(t)

	started := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = http.NewResponseController(w).Flush()

		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		select {
		case <-started:
			cancel()
		case <-ctx.Done():
		}
	}()

	target, _ := url.Parse(server.URL)
	done := make(chan *Result, 1)

	go func() { result, _ := RequestToIP(ctx, "127.0.0.1", target, http.MethodGet, nil, nil); done <- result }()

	select {
	case result := <-done:
		if result == nil || result.Err == "" {
			t.Fatalf("missing cancellation: %+v", result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("body read ignored cancellation")
	}
}

// A body that never ends ensures unknown lengths cannot escape the byte budget.
type endlessBody struct {
	read   int
	closed bool
}

func (b *endlessBody) Read(buffer []byte) (int, error) {
	for i := range buffer {
		buffer[i] = 'x'
	}

	b.read += len(buffer)

	return len(buffer), nil
}

func (b *endlessBody) Close() error {
	b.closed = true

	return nil
}

func TestResponseBudgetStopsAndClosesUnknownBody(t *testing.T) {
	proxyConfig(t)
	t.Setenv("HB__PROXY_MAX_RESPONSE_BODY_BYTES", "8")

	body := &endlessBody{}
	result := &Result{}
	parseResponse(
		t.Context(),
		&http.Response{StatusCode: http.StatusOK, ContentLength: -1, Body: body},
		result,
	)

	if body.read != 9 || !body.closed || result.Err == "" || result.Result != nil {
		t.Fatalf("body=%+v result=%+v", body, result)
	}
}

func TestResponseBudgetAfterDecompression(t *testing.T) {
	proxyConfig(t)
	t.Setenv("HB__PROXY_MAX_RESPONSE_BODY_BYTES", "128")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte(strings.Repeat("x", 1024)))
		_ = writer.Close()
	}))
	defer server.Close()

	target, _ := url.Parse(server.URL)

	result, err := RequestToIP(t.Context(), "127.0.0.1", target, http.MethodGet, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	if result.Err == "" || result.Result != nil {
		t.Fatalf("expanded body escaped limit: %+v", result)
	}
}
