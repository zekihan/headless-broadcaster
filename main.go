package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/justinas/alice"
	"github.com/zekihan/headless-broadcaster/internal/commons/environment"
	"github.com/zekihan/headless-broadcaster/internal/commons/middleware"
	"github.com/zekihan/headless-broadcaster/internal/commons/utils"
	"github.com/zekihan/headless-broadcaster/internal/commons/web/errorroutes"
	"github.com/zekihan/headless-broadcaster/internal/commons/web/health"
	"github.com/zekihan/headless-broadcaster/internal/commons/web/ping"

	"github.com/zekihan/headless-broadcaster/config"
	"github.com/zekihan/headless-broadcaster/routes/api"
)

const (
	defaultTimeout = 10 * time.Second
)

var (
	stdChain  alice.Chain
	authChain alice.Chain
	cfg       *config.Config
)

func init() {
	err := environment.LoadEnvironmentFile()
	if err != nil {
		slog.ErrorContext(
			nil,
			"Failed to load environment file",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	// Init config.
	cfg = config.GetConfig()

	// Init logger.
	utils.InitLogger(&utils.LoggerConfig{
		Level:     cfg.LogConfig.Level,
		AddSource: cfg.LogConfig.AddSource,
	})

	// Init middleware.
	stdChain = alice.New(middleware.RecoverHandler, middleware.Logging)
	authChain = alice.New(middleware.GetAuthMiddlewareForSingleAPIKey(cfg.WebConfig.APIKey))
}

func main() {
	slog.InfoContext(context.Background(), "Starting headless broadcaster server")
	utils.LogVersionInfo(context.Background())

	// API routes.
	http.HandleFunc("/api/ping", ping.GetPing)
	http.HandleFunc("/api/health", health.GetHealth)

	// Broadcast routes.
	http.Handle("/broadcast", authChain.ThenFunc(api.Broadcast))

	// Error routes.
	http.HandleFunc("/404", errorroutes.GetNotFound)
	http.HandleFunc("/5xx", errorroutes.Get5xx)

	// Redirect all other paths.
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Redirect all other paths to 404.
		http.Redirect(w, r, "/404", http.StatusMovedPermanently)
	})

	port := cfg.WebConfig.Port
	host := cfg.WebConfig.Host
	slog.InfoContext(context.Background(), fmt.Sprintf("Starting server on %s:%d", host, port))

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      stdChain.Then(http.DefaultServeMux),
		ReadTimeout:  defaultTimeout,
		WriteTimeout: defaultTimeout,
	}

	err := srv.ListenAndServe()
	if err != nil {
		slog.ErrorContext(context.Background(), "Failed to start server", slog.Any("error", err))
		os.Exit(1)
	}
}
