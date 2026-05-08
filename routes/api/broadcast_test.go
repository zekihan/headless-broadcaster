package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/zekihan/headless-broadcaster/proxy"
)

// Test NewResult, Success, and Failed for all-success scenario.
func TestNewResult_AllSuccess(t *testing.T) {
	results := []*proxy.Result{
		{StatusCode: 200},
		{StatusCode: 200},
	}

	r := NewResult("example.com", results)
	if r.Counts.Total != 2 || r.Counts.Success != 2 || r.Counts.Failed != 0 {
		t.Errorf("expected total=2, success=2, failed=0; got total=%d, success=%d, failed=%d",
			r.Counts.Total, r.Counts.Success, r.Counts.Failed)
	}

	if !r.Success() || r.Failed() {
		t.Error("expected Success()=true and Failed()=false for all-success")
	}
}

// Test NewResult, Success, and Failed for all-failed scenario.
func TestNewResult_AllFailed(t *testing.T) {
	results := []*proxy.Result{
		{StatusCode: 500},
		{StatusCode: 404},
	}

	r := NewResult("badhost", results)
	if r.Counts.Total != 2 || r.Counts.Success != 0 || r.Counts.Failed != 2 {
		t.Errorf("expected total=2, success=0, failed=2; got total=%d, success=%d, failed=%d",
			r.Counts.Total, r.Counts.Success, r.Counts.Failed)
	}

	if r.Success() || !r.Failed() {
		t.Error("expected Success()=false and Failed()=true for all-failed")
	}
}

// Test NewResult, Success, Failed for mixed scenario.
func TestNewResult_Mixed(t *testing.T) {
	results := []*proxy.Result{
		{StatusCode: 200},
		{StatusCode: 503},
		{StatusCode: 200},
	}

	r := NewResult("mixed", results)
	if r.Counts.Total != 3 || r.Counts.Success != 2 || r.Counts.Failed != 1 {
		t.Errorf("expected total=3, success=2, failed=1; got total=%d, success=%d, failed=%d",
			r.Counts.Total, r.Counts.Success, r.Counts.Failed)
	}

	if r.Success() || !r.Failed() {
		t.Error("expected Success()=false and Failed()=true for mixed results")
	}
}

// Test JSON marshaling of Result.
func TestResult_JSONMarshaling(t *testing.T) {
	results := []*proxy.Result{
		{StatusCode: 200},
	}
	r := NewResult("jsonhost", results)

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("failed to marshal Result: %v", err)
	}

	var out struct {
		Host   string `json:"host"`
		Counts struct {
			Total   int `json:"total"`
			Success int `json:"success"`
			Failed  int `json:"failed"`
		} `json:"counts"`
	}

	err = json.Unmarshal(data, &out)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if out.Host != "jsonhost" || out.Counts.Total != 1 || out.Counts.Success != 1 ||
		out.Counts.Failed != 0 {
		t.Errorf("unexpected JSON content: %+v", out)
	}
}

// Test Broadcast handler error on empty URL.
func TestBroadcast_EmptyURL(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	Broadcast(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 BadRequest for invalid URL, got %d", rr.Code)
	}
}

// Test Broadcast handler error on invalid URL.
func TestBroadcast_InvalidURL(t *testing.T) {
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?url=http://%41:80:invalid-url",
		http.NoBody,
	)
	rr := httptest.NewRecorder()
	Broadcast(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 BadRequest for invalid URL, got %d", rr.Code)
	}
}

// Test Broadcast handler error on unresolvable host.
func TestBroadcast_UnresolvableHost(t *testing.T) {
	// .invalid is guaranteed non-resolvable.
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?url=http://nonexistent.invalid/",
		http.NoBody,
	)
	rr := httptest.NewRecorder()
	Broadcast(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 InternalServerError for unresolvable host, got %d", rr.Code)
	}
}

// Test Broadcast handler.
func TestBroadcast(t *testing.T) {
	// Create a mock HTTP server.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("mock response"))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)

			return
		}
	}))
	defer mockServer.Close()

	// Use the mock server's URL.
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?url="+mockServer.URL,
		http.NoBody,
	)
	req.Header.Add("Authorization", "Bearer test-token")

	rr := httptest.NewRecorder()
	Broadcast(rr, req)

	var result Result

	err := json.Unmarshal(rr.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rr.Code)
	}

	expectedURL, _ := url.Parse(mockServer.URL)
	if result.Host != expectedURL.Hostname() {
		t.Errorf("unexpected host: got %s, want %s", result.Host, expectedURL.Hostname())
	}

	if result.Counts.Total != result.Counts.Success+result.Counts.Failed {
		t.Errorf(
			"total count mismatch: total=%d, success=%d, failed=%d",
			result.Counts.Total,
			result.Counts.Success,
			result.Counts.Failed,
		)
	}

	if result.Counts.Failed != 0 {
		t.Errorf("expected all requests to succeed, but failed=%d", result.Counts.Failed)
	}
}
