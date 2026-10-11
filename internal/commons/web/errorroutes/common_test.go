package errorroutes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendErrorMessage(t *testing.T) {
	tests := []struct {
		name           string
		acceptType     string
		expectedBody   string
		expectedStatus int
	}{
		{
			name:           "HTML response",
			acceptType:     "text/html",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "<html><body><h1>401 Unauthorized</h1></body></html>",
		},
		{
			name:           "JSON response",
			acceptType:     "application/json",
			expectedStatus: http.StatusBadGateway,
			expectedBody:   `{"error": "Bad Gateway"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/nonexistent",
				http.NoBody,
			)
			req.Header.Set("Accept", tt.acceptType)

			rec := httptest.NewRecorder()

			SendErrorMessage(rec, req, tt.expectedStatus)

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
