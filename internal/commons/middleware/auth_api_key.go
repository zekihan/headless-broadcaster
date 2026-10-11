package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func GetAuthMiddlewareWithFunction(
	f func(r *http.Request, apiKey string) bool,
) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAPIKey := getAPIKeyFromRequest(r)

			if !f(r, userAPIKey) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)

				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

func GetAuthMiddlewareForSingleAPIKey(apiKey string) func(http.Handler) http.Handler {
	return GetAuthMiddlewareWithFunction(func(r *http.Request, userAPIKey string) bool {
		if apiKey == "" {
			return true
		}

		return subtle.ConstantTimeCompare([]byte(userAPIKey), []byte(apiKey)) == 1
	})
}

func getAPIKeyFromRequest(r *http.Request) string {
	userAPIKey := r.Header.Get("X-Api-Key")

	userAPIKey = strings.TrimSpace(userAPIKey)
	if userAPIKey != "" {
		return userAPIKey
	}

	userAPIKey = r.URL.Query().Get("apiKey")

	userAPIKey = strings.TrimSpace(userAPIKey)
	if userAPIKey != "" {
		return userAPIKey
	}

	cookie, err := r.Cookie("apiKey")
	if err == nil {
		cookieVal := strings.TrimSpace(cookie.Value)
		if cookieVal != "" {
			return cookieVal
		}
	}

	_, password, ok := r.BasicAuth()
	if ok {
		password = strings.TrimSpace(password)
		if password != "" {
			return password
		}
	}

	return userAPIKey
}
