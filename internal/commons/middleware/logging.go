package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/felixge/httpsnoop"

	"github.com/zekihan/headless-broadcaster/internal/commons/utils"
)

var defaultSkippedPaths = []string{
	"/ping",
	"/health",
	"/ready",
	"/api/ping",
	"/api/health",
	"/api/ready",
}

var defaultSkippedContains = []string{
	"favicon.ico",
}

func Logging(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.Contains(defaultSkippedPaths, r.URL.Path) {
			handler.ServeHTTP(w, r)

			return
		}

		if utils.Contains(defaultSkippedContains, r.URL.Path) {
			handler.ServeHTTP(w, r)

			return
		}

		ip, err := utils.GetIP(r)
		if err != nil {
			slog.ErrorContext(r.Context(), "Error getting IP: ", slog.Any("error", err))

			ip = "unknown"
		}

		metrics := httpsnoop.CaptureMetrics(handler, w, r)
		slog.InfoContext(r.Context(), "HTTP request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.String("ip", ip),
			slog.Int("status", metrics.Code),
			slog.Int64("size", metrics.Written),
			slog.Duration("duration", metrics.Duration.Round(time.Millisecond)),
		)
	})
}
