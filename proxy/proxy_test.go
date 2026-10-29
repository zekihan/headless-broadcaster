package proxy

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

var errRead = errors.New("read error")

// errorReader Simulates a Body that always fails on Read().
type errorReader struct{}

func (e errorReader) Read(p []byte) (int, error) { return 0, errRead }
func (e errorReader) Close() error               { return nil }

func TestParseResponse_JSON(t *testing.T) {
	body := `{"msg":"hello"}`
	resp := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
	}
	result := &Result{}
	parseResponse(t.Context(), resp, result)

	if result.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", result.StatusCode)
	}

	m, ok := result.Result.(map[string]any)
	if !ok || m["msg"] != "hello" {
		t.Errorf("expected JSON map with msg=hello, got %+v", result.Result)
	}

	if result.Err != "" {
		t.Errorf("unexpected error: %s", result.Err)
	}
}

func TestParseResponse_Text(t *testing.T) {
	body := "plain text"
	resp := &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	result := &Result{}
	parseResponse(t.Context(), resp, result)

	if result.StatusCode != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", result.StatusCode)
	}

	if result.Result != body {
		t.Errorf("expected result %q, got %v", body, result.Result)
	}

	if result.Err != "" {
		t.Errorf("unexpected error: %s", result.Err)
	}
}

func TestParseResponse_ReadError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       errorReader{},
	}
	result := &Result{}
	parseResponse(t.Context(), resp, result)

	if result.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", result.StatusCode)
	}

	if !strings.Contains(result.Err, "read error") {
		t.Errorf("expected read error, got %s", result.Err)
	}
}

func TestRequestToIP_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		_, err := w.Write([]byte(`{"ok":true}`))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)

			return
		}
	}))
	defer srv.Close()

	pathURL, _ := url.Parse(srv.URL + "/path")

	ip, _, err := net.SplitHostPort(pathURL.Host)
	if err != nil {
		ip = pathURL.Host // Handle cases where no port is present
	}

	res, err := RequestToIP(t.Context(), ip, pathURL, http.MethodGet, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	m, _ := res.Result.(map[string]any)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %+v", res.Result)
	}
}

func TestRequestToIP_Text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("no content"))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)

			return
		}
	}))
	defer srv.Close()

	helloURL, _ := url.Parse(srv.URL + "/hello")

	ip, _, err := net.SplitHostPort(helloURL.Host)
	if err != nil {
		ip = helloURL.Host // Handle cases where no port is present
	}

	res, err := RequestToIP(t.Context(), ip, helloURL, http.MethodGet, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	if res.Result != "no content" {
		t.Errorf("expected result 'no content', got %v", res.Result)
	}
}

func TestParseResponse_IPv6(t *testing.T) {
	srv := httptest.NewUnstartedServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")

			_, err := w.Write([]byte(`{"ok":true}`))
			if err != nil {
				http.Error(w, "failed to write response", http.StatusInternalServerError)

				return
			}
		}),
	)

	err := srv.Listener.Close()
	if err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}

	listener := net.ListenConfig{}
	srv.Listener, _ = listener.Listen(t.Context(), "tcp6", "[::1]:0") // Use IPv6 address

	srv.Start()
	defer srv.Close()

	pathURL, _ := url.Parse(srv.URL + "/path")

	ip, _, err := net.SplitHostPort(pathURL.Host)
	if err != nil {
		ip = pathURL.Host // Handle cases where no port is present
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil || parsedIP.To16() == nil ||
		parsedIP.To4() != nil { // Ensure we have a valid IPv6 address
		t.Skip("Skipping IPv6 test as server is not using IPv6")
	}

	res, err := RequestToIP(t.Context(), ip, pathURL, http.MethodGet, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	m, _ := res.Result.(map[string]any)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %+v", res.Result)
	}
}

func TestRequestToIP_Header(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		_, err := w.Write([]byte(`{"ok":true}`))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)

			return
		}
	}))
	defer srv.Close()

	pathURL, _ := url.Parse(srv.URL + "/path")

	ip, _, err := net.SplitHostPort(pathURL.Host)
	if err != nil {
		ip = pathURL.Host // Handle cases where no port is present
	}

	res, err := RequestToIP(t.Context(), ip, pathURL, http.MethodGet, nil, http.Header{
		"X-Custom-Header": {"test-value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	m, _ := res.Result.(map[string]any)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %+v", res.Result)
	}
}

func TestRequestToIP_Post(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")

		_, err := w.Write([]byte(`{"ok":true}`))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)

			return
		}
	}))
	defer srv.Close()

	pathURL, _ := url.Parse(srv.URL + "/path")

	ip, _, err := net.SplitHostPort(pathURL.Host)
	if err != nil {
		ip = pathURL.Host // Handle cases where no port is present
	}

	res, err := RequestToIP(
		t.Context(),
		ip,
		pathURL,
		http.MethodPost,
		[]byte(`{"data":"test"}`),
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}

	m, _ := res.Result.(map[string]any)
	if m["ok"] != true {
		t.Errorf("expected ok=true, got %+v", res.Result)
	}
}
