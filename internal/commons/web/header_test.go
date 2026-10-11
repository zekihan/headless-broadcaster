package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetAcceptType(t *testing.T) {
	tests := []struct {
		name           string
		acceptHeader   string
		expectedResult AcceptType
	}{
		{"JSON Accept Header", "application/json", AcceptTypeJSON},
		{"HTML Accept Header", "text/html", AcceptTypeHTML},
		{"Plain Accept Header", "text/plain", AcceptTypePlain},
		{"No Accept Header", "", AcceptTypeNone},
		{"Unknown Accept Header", "application/xml", AcceptTypeNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
			if tt.acceptHeader != "" {
				req.Header.Set("Accept", tt.acceptHeader)
			}

			result := GetAcceptType(req)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
		})
	}
}
