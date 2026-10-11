package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetHealth(t *testing.T) {
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", http.NoBody)
	rr := httptest.NewRecorder()

	GetHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("GetHealth returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	expected := "OK\n"
	if rr.Body.String() != expected {
		t.Errorf("GetHealth returned unexpected body: got %q want %q", rr.Body.String(), expected)
	}
}
