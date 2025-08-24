package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	log.Debug("Broadcasting to ", ip, " ", parsedURL.String())

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
		log.Error("Error broadcasting to ", parsedURL.Hostname(), err)
		result.Err = err.Error()
	}

	if response != nil {
		parseResponse(response, &result)
	}

	return &result, nil
}

func parseResponse(response *http.Response, result *Result) {
	result.StatusCode = response.StatusCode
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Error reading response body ", err)
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
		log.Error("Error unmarshalling response body ", err)
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
