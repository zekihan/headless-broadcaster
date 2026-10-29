package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zekihan/headless-broadcaster/config"
)

const (
	defaultTimeout = 30 * time.Second
)

func RequestToIP(
	ctx context.Context,
	ip string,
	parsedURL *url.URL,
	method string,
	body []byte,
	headers http.Header,
) (*Result, error) {
	result := Result{IP: ip}
	slog.DebugContext(ctx, "Proxying request",
		slog.String("method", method),
		slog.String("url", parsedURL.String()),
		slog.String("ip", ip),
	)

	bodyReader := bytes.NewReader([]byte{})
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	request, err := http.NewRequestWithContext(ctx, method, parsedURL.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	request.Header = headers.Clone()

	client := getHTTPClient(ip) //nolint:contextcheck // Reads configuration initialized at startup.
	// Each client owns a one-shot transport; release its pool after the body closes.
	defer client.CloseIdleConnections()

	response, err := client.Do(request)
	if err != nil {
		slog.ErrorContext(ctx, "Error making proxied request",
			slog.String("method", method),
			slog.String("url", parsedURL.String()),
			slog.String("ip", ip),
			slog.Any("error", err),
		)
		result.Err = err.Error()
	}

	if response != nil {
		parseResponse(ctx, response, &result)
	}

	return &result, nil
}

func parseResponse(ctx context.Context, response *http.Response, result *Result) {
	result.StatusCode = response.StatusCode
	defer response.Body.Close()

	limit := config.GetConfig().ProxyConfig.MaxResponseBodyBytes //nolint:contextcheck // Startup configuration.
	if response.ContentLength > limit {
		result.Err = "upstream response body exceeds configured limit"

		return
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if int64(len(body)) > limit {
		result.Err = "upstream response body exceeds configured limit"

		return
	}

	if err != nil {
		slog.ErrorContext(ctx, "Error reading response body", slog.Any("error", err))
		result.Err = err.Error()

		return
	}

	if !json.Valid(body) {
		result.Result = string(body)

		return
	}

	var jsonBody any

	err = json.Unmarshal(body, &jsonBody)
	if err != nil {
		slog.ErrorContext(ctx, "Error unmarshaling JSON response", slog.Any("error", err))
		result.Err = err.Error()
	} else {
		result.Result = jsonBody
	}
}

func getHTTPClient(ip string) *http.Client {
	dialer := &net.Dialer{
		Timeout:   defaultTimeout,
		KeepAlive: defaultTimeout,
	}

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				RootCAs:            config.GetConfig().ProxyConfig.TLSRootCAs(),
				InsecureSkipVerify: config.GetConfig().ProxyConfig.InsecureSkipVerify, //nolint:gosec // Explicit opt-out.
			},
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				if strings.Contains(ip, ":") {
					addr = "[" + ip + "]" + addr[strings.LastIndex(addr, ":"):]
				} else {
					addr = ip + addr[strings.LastIndex(addr, ":"):]
				}

				return dialer.DialContext(ctx, network, addr)
			},
		},
		Timeout: defaultTimeout,
	}

	return &client
}

type Result struct {
	Result     any    `json:"result,omitempty"`
	IP         string `json:"ip"`
	Err        string `json:"err,omitempty"`
	StatusCode int    `json:"statusCode"`
}
