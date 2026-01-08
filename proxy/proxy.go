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
	headers map[string]string,
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

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	client := getHTTPClient(ip)

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

	body, err := io.ReadAll(response.Body)
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
				InsecureSkipVerify: true, //nolint:gosec // Ideally, this should be configurable by the user.
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
