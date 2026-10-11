package utils

import (
	"context"
	"log/slog"
	"strconv"
	"time"
)

var (
	Version      = "dev"
	CommitHash   = "none"
	BuildDate    = "I don't remember exactly"
	GitTreeState = "not a git repo"
)

func LogVersionInfo(ctx context.Context) {
	date := time.Time{}

	timestamp, err := strconv.ParseInt(BuildDate, 10, 64)
	if err == nil {
		date = time.Unix(timestamp, 0).UTC()
	}

	slog.InfoContext(
		ctx,
		"Application version info",
		slog.String("version", Version),
		slog.String("buildDate", date.Format(time.RFC3339)),
		slog.String("commitHash", CommitHash),
		slog.String("gitTreeState", GitTreeState),
	)
}
