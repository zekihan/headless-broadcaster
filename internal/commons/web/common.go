package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/zekihan/headless-broadcaster/internal/commons/utils"
)

func Write(ctx context.Context, w http.ResponseWriter, b []byte) bool {
	_, err := w.Write(b)

	return CheckError(ctx, w, err, http.StatusInternalServerError, "failed to write string")
}

func WriteString(ctx context.Context, w http.ResponseWriter, s string) bool {
	_, err := io.WriteString(w, s)

	return CheckError(ctx, w, err, http.StatusInternalServerError, "failed to write string")
}

// CheckError checks the error, and writes to w if there is an error.  Returns
// true if err is non-nil.
func CheckError(
	ctx context.Context,
	w http.ResponseWriter,
	err error,
	statusCode int,
	messages ...string,
) bool {
	if err != nil {
		err = utils.WrapErrors(err, messages...)
		if err == nil {
			return false
		}

		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}

		if statusCode >= http.StatusInternalServerError {
			slog.ErrorContext(
				ctx,
				"Returned non 2xx http code",
				slog.Int("httpCode", statusCode),
				slog.Any("error", err),
			)
		} else {
			slog.WarnContext(
				ctx,
				"Returned non 2xx http code",
				slog.Int("httpCode", statusCode),
				slog.Any("error", err),
			)
		}

		http.Error(w, err.Error(), statusCode)

		return true
	}

	return false
}
