package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/zekihan/headless-broadcaster/config"
	"github.com/zekihan/headless-broadcaster/internal/commons/web"
	"github.com/zekihan/headless-broadcaster/proxy"
)

var errMissingURLParam = errors.New("missing 'url' query parameter")

func Broadcast(w http.ResponseWriter, r *http.Request) {
	resolver := &net.Resolver{}
	broadcast(w, r, resolver.LookupHost)
}

func broadcast(
	w http.ResponseWriter,
	r *http.Request,
	lookupHost func(context.Context, string) ([]string, error),
) {
	urlFromQuery := r.URL.Query().Get("url")
	if urlFromQuery == "" {
		web.CheckError(
			r.Context(),
			w,
			errMissingURLParam,
			http.StatusBadRequest,
			"Missing 'url' query parameter",
		)

		return
	}

	parsedURL, err := url.Parse(urlFromQuery)
	if web.CheckError(r.Context(), w, err, http.StatusBadRequest, "Invalid URL") {
		return
	}

	host := parsedURL.Hostname()

	hostIPs, err := lookupHost(r.Context(), host)
	if web.CheckError(
		r.Context(),
		w,
		err,
		http.StatusInternalServerError,
		"Failed to resolve host",
	) {
		return
	}

	var body []byte

	if r.Body != nil {
		defer r.Body.Close()

		limit := config.GetConfig().ProxyConfig.MaxRequestBodyBytes //nolint:contextcheck // Startup configuration.
		if r.ContentLength > limit {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)

			return
		}

		body, err = io.ReadAll(http.MaxBytesReader(w, r.Body, limit))

		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)

			return
		}

		if web.CheckError(
			r.Context(),
			w,
			err,
			http.StatusInternalServerError,
			"Failed to read request body",
		) {
			return
		}
	}

	headers := upstreamHeaders(
		r,
		config.GetConfig().WebConfig.APIKey, //nolint:contextcheck // Startup configuration.
	)

	results := make([]*proxy.Result, 0)

	for _, ip := range hostIPs {
		result, proxyErr := proxy.RequestToIP(r.Context(), ip, parsedURL, r.Method, body, headers)
		if web.CheckError(
			r.Context(),
			w,
			proxyErr,
			http.StatusInternalServerError,
			"Failed to make request",
		) {
			return
		}

		results = append(results, result)
	}

	result := NewResult(host, results)

	if result.Failed() {
		w.WriteHeader(http.StatusInternalServerError)
	}

	resultBytes, err := json.Marshal(result)
	if web.CheckError(
		r.Context(),
		w,
		err,
		http.StatusInternalServerError,
		"Failed to marshal result",
	) {
		return
	}

	web.Write(r.Context(), w, resultBytes)
}

// upstreamHeaders removes reserved broadcaster credentials, preserving downstream auth.
func upstreamHeaders(r *http.Request, apiKey string) http.Header {
	headers := r.Header.Clone()
	headers.Del("X-Api-Key")
	// Filter raw cookie pairs rather than reserializing all cookies: preserve repeated
	// application cookies and their header lines whenever no reserved cookie is present.
	for i, line := range headers.Values("Cookie") {
		parts := strings.Split(line, ";")
		kept := make([]string, 0, len(parts))
		removed := false

		for _, part := range parts {
			name, _, _ := strings.Cut(strings.TrimSpace(part), "=")
			if strings.TrimSpace(name) == "apiKey" {
				removed = true

				continue
			}

			kept = append(kept, part)
		}

		if removed {
			headers["Cookie"][i] = strings.TrimSpace(strings.Join(kept, ";"))
		}
	}

	cookies := headers.Values("Cookie")
	if cookies != nil {
		kept := cookies[:0]
		for _, line := range cookies {
			if line != "" {
				kept = append(kept, line)
			}
		}

		if len(kept) == 0 {
			headers.Del("Cookie")
		} else {
			headers["Cookie"] = kept
		}
	}
	// Basic is also a broadcaster carrier. Strip a matching password even when
	// a higher-priority carrier authenticated the request; preserve distinct auth.
	if apiKey != "" {
		authValues := headers.Values("Authorization")

		kept := authValues[:0]
		for _, value := range authValues {
			authRequest := &http.Request{Header: http.Header{"Authorization": {value}}}

			_, password, ok := authRequest.BasicAuth()
			if !ok || strings.TrimSpace(password) != apiKey {
				kept = append(kept, value)
			}
		}

		if len(kept) == 0 {
			headers.Del("Authorization")
		} else {
			headers["Authorization"] = kept
		}
	}

	return headers
}

type Result struct {
	Host    string          `json:"host"`
	Results []*proxy.Result `json:"results"`
	Counts  ResultCounts    `json:"counts"`
}

type ResultCounts struct {
	Total   int `json:"total"`
	Success int `json:"success"`
	Failed  int `json:"failed"`
}

func NewResult(host string, results []*proxy.Result) *Result {
	successCount := 0
	failedCount := 0

	for _, res := range results {
		if res.Err == "" && res.StatusCode >= http.StatusOK &&
			res.StatusCode < http.StatusMultipleChoices {
			successCount++
		} else {
			failedCount++
		}
	}

	return &Result{
		Host:    host,
		Results: results,
		Counts: ResultCounts{
			Total:   len(results),
			Success: successCount,
			Failed:  failedCount,
		},
	}
}

func (r *Result) Success() bool {
	return r.Counts.Success == r.Counts.Total
}

func (r *Result) Failed() bool {
	return r.Counts.Failed > 0
}
