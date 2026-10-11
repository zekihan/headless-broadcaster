package utils

import (
	"bytes"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLogVersionInfoWithTextHandlerValidBuildDate(t *testing.T) {
	originalBuildDate := BuildDate

	defer func() { BuildDate = originalBuildDate }()

	timestamp := time.Now().Unix()
	BuildDate = strconv.FormatInt(timestamp, 10)
	expectedDate := time.Unix(timestamp, 0).UTC().Format(time.RFC3339)

	origLogger := slog.Default()
	defer slog.SetDefault(origLogger)

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	LogVersionInfo(t.Context())

	out := buf.String()
	if !strings.Contains(out, Version) {
		t.Fatalf("expected version in log, got: %q", out)
	}

	if !strings.Contains(out, CommitHash) {
		t.Fatalf("expected commit hash in log, got: %q", out)
	}

	if !strings.Contains(out, GitTreeState) {
		t.Fatalf("expected git tree state in log, got: %q", out)
	}

	if !strings.Contains(out, expectedDate) {
		t.Fatalf("expected build date %q in log, got: %q", expectedDate, out)
	}
}

func TestLogVersionInfoWithTextHandlerInvalidBuildDateProducesZeroTime(t *testing.T) {
	originalBuildDate := BuildDate

	defer func() { BuildDate = originalBuildDate }()

	BuildDate = "not-a-timestamp"

	origLogger := slog.Default()
	defer slog.SetDefault(origLogger)

	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	LogVersionInfo(t.Context())

	out := buf.String()
	if !strings.Contains(out, Version) {
		t.Fatalf("expected version in log, got: %q", out)
	}
	// zero time formatted appears as 0001-01-01
	if !strings.Contains(out, "0001-01-01") {
		t.Fatalf("expected zero buildDate in log when BuildDate invalid, got: %q", out)
	}
}
