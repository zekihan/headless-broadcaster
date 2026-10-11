package middleware

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuth_NoConfAPIKey(t *testing.T) {
	Auth := GetAuthMiddlewareForSingleAPIKey("")
	handler := Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("OK"))
		if err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)

			return
		}
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("expected body OK, got %q", rr.Body.String())
	}
}

func TestAuth_CorrectHeaderAPIKey(t *testing.T) {
	Auth := GetAuthMiddlewareForSingleAPIKey("secret")
	handler := Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.Header.Set("X-Api-Key", "secret")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestAuth_CorrectQueryAPIKey(t *testing.T) {
	Auth := GetAuthMiddlewareForSingleAPIKey("secret")
	handler := Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?apiKey=secret",
		http.NoBody,
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestAuth_WrongAPIKey(t *testing.T) {
	Auth := GetAuthMiddlewareForSingleAPIKey("secret")
	handler := Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?apiKey=wrong",
		http.NoBody,
	)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rr.Code)
	}
}

func TestAuth_BasicAuthPasswordAPIKey(t *testing.T) {
	Auth := GetAuthMiddlewareForSingleAPIKey("secret")
	handler := Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Any username, password is the API key
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("anyuser:secret"))
	req.Header.Set("Authorization", auth)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestAuth_CorrectCookieAPIKey(t *testing.T) {
	Auth := GetAuthMiddlewareForSingleAPIKey("secret")
	handler := Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.AddCookie(&http.Cookie{Name: "apiKey", Value: "secret"}) //nolint:gosec // generic method

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}
