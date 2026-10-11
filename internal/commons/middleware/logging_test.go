package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestLogging_HandlerCalled(t *testing.T) {
	var logBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil))) // capture logs in buffer

	called := false
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	req.RemoteAddr = "192.168.1.100:12345" // simulate a request from a specific IP
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected wrapped handler to be called")
	}

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rr.Code)
	}

	logStr := logBuffer.String()
	checkLogs(t, logStr, http.MethodPost, "192.168.1.100", http.StatusCreated, "/test")
}

func TestLogging_HandlerCalled_IPUnknown(t *testing.T) {
	var logBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil))) // capture logs in buffer

	called := false
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	req.RemoteAddr = "300.0.113.1:1234" // simulate invalid IP
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected wrapped handler to be called")
	}

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	logStr := logBuffer.String()
	checkLogs(t, logStr, http.MethodPost, "unknown", http.StatusCreated, "/test")
}

func TestLogging_SkippedPaths(t *testing.T) {
	paths := []string{"/ping", "/health", "/ready", "/api/ping", "/api/health", "/api/ready"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			var logBuffer bytes.Buffer
			slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil)))

			called := false
			handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true

				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, http.NoBody)
			req.RemoteAddr = "127.0.0.1:1234"
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if !called {
				t.Errorf("expected handler to be called for skipped path %s", path)
			}

			if strings.Contains(logBuffer.String(), "HTTP request") {
				t.Errorf(
					"expected no log entry for skipped path %s, got: %s",
					path,
					logBuffer.String(),
				)
			}
		})
	}
}

func TestLogging_SkippedContains(t *testing.T) {
	var logBuffer bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logBuffer, nil)))

	called := false
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		w.WriteHeader(http.StatusOK)
	}))

	// defaultSkippedContains matches the literal path "favicon.ico" via exact equality,
	// so we set URL.Path directly rather than passing it as the request URL.
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.URL.Path = "favicon.ico"
	req.RemoteAddr = "127.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("expected handler to be called for favicon.ico path")
	}

	if strings.Contains(logBuffer.String(), "HTTP request") {
		t.Errorf("expected no log entry for favicon.ico, got: %s", logBuffer.String())
	}
}

func checkLogs(t *testing.T, logStr, method, ip string, status int, path string) {
	t.Helper()

	if !strings.Contains(logStr, method) {
		t.Errorf("expected log to contain method '%s', got %s", method, logStr)
	}

	if !strings.Contains(logStr, ip) {
		t.Errorf("expected log to contain IP '%s', got %s", ip, logStr)
	}

	if !strings.Contains(logStr, strconv.Itoa(status)) {
		t.Errorf("expected log to contain status '%d', got %s", status, logStr)
	}

	if !strings.Contains(logStr, path) {
		t.Errorf("expected log to contain path '%s', got %s", path, logStr)
	}
}
