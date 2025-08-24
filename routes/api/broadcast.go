package api

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/zekihan/headless-broadcaster/internal/commons/web"

	"github.com/zekihan/headless-broadcaster/proxy"
)

var errMissingURLParam = errors.New("missing 'url' query parameter")

func Broadcast(w http.ResponseWriter, r *http.Request) {
	urlFromQuery := r.URL.Query().Get("url")
	if urlFromQuery == "" {
		web.CheckError(
			w,
			errMissingURLParam,
			http.StatusBadRequest,
			"Missing 'url' query parameter",
		)

		return
	}

	parsedURL, err := url.Parse(urlFromQuery)
	if web.CheckError(w, err, http.StatusBadRequest, "Invalid URL") {
		return
	}

	host := parsedURL.Hostname()

	resolver := &net.Resolver{}

	hostIPs, err := resolver.LookupHost(r.Context(), host)
	if web.CheckError(w, err, http.StatusInternalServerError, "Failed to resolve host") {
		return
	}

	var body []byte

	if r.Body != nil {
		defer r.Body.Close()

		body, err = io.ReadAll(r.Body)
		if web.CheckError(
			w,
			err,
			http.StatusInternalServerError,
			"Failed to read request body",
		) {
			return
		}
	}

	headers := make(map[string]string)
	for key, values := range r.Header {
		headers[key] = values[0]
	}

	results := make([]*proxy.Result, 0)
	for _, ip := range hostIPs {
		result, proxyErr := proxy.RequestToIP(r.Context(), ip, parsedURL, r.Method, body, headers)
		if web.CheckError(
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
	if web.CheckError(w, err, http.StatusInternalServerError, "Failed to marshal result") {
		return
	}

	web.Write(w, resultBytes)
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
		if res.StatusCode == http.StatusOK {
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
