package errorroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGet5xx(t *testing.T) {
	tests := []struct {
		name           string
		acceptType     string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "HTML response",
			acceptType:     "text/html",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "<html><body><h1>500 Internal Server Error</h1></body></html>",
		},
		{
			name:           "JSON response",
			acceptType:     "application/json",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `{"error": "Internal Server Error"}`,
		},
		{
			name:           "Plain text response",
			acceptType:     "text/plain",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
		},
		{
			name:           "No Accept header",
			acceptType:     "",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
		},
		{
			name:           "Default response",
			acceptType:     "application/xml",
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/error",
				http.NoBody,
			)
			req.Header.Set("Accept", tt.acceptType)

			rec := httptest.NewRecorder()

			Get5xx(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, res.StatusCode)
			}

			body := rec.Body.String()
			if tt.expectedBody != "" && body[:len(tt.expectedBody)] != tt.expectedBody {
				t.Errorf("expected body to start with %q, got %q", tt.expectedBody, body)
			}
		})
	}
}
