package api

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/zekihan/headless-broadcaster/proxy"
)

func TestNewResult_StatusAndError(t *testing.T) {
	for _, tc := range []struct {
		err     string
		status  int
		success bool
	}{
		{status: 0, err: "", success: false},
		{status: 199, err: "", success: false},
		{status: 200, err: "", success: true},
		{status: 201, err: "", success: true},
		{status: 202, err: "", success: true},
		{status: 204, err: "", success: true},
		{status: 299, err: "", success: true},
		{status: 300, err: "", success: false},
		{status: 302, err: "", success: false},
		{status: 400, err: "", success: false},
		{status: 404, err: "", success: false},
		{status: 500, err: "", success: false},
		{status: 503, err: "", success: false},
		{status: 200, err: "unexpected EOF", success: false},
		{status: 204, err: "canceled", success: false},
		{status: 0, err: "dial failed", success: false},
	} {
		t.Run(strconv.Itoa(tc.status)+tc.err, func(t *testing.T) {
			result := NewResult("fixture", []*proxy.Result{{StatusCode: tc.status, Err: tc.err}})

			success := 0
			if tc.success {
				success = 1
			}

			if result.Counts != (ResultCounts{Total: 1, Success: success, Failed: 1 - success}) ||
				result.Success() != tc.success ||
				result.Failed() == tc.success {
				t.Fatalf("unexpected classification: %+v", result)
			}
		})
	}
}

func TestBroadcast_LocalResults(t *testing.T) {
	for _, tc := range []struct {
		name      string
		statuses  []int
		truncated bool
		want      ResultCounts
		status    int
	}{
		{"all 2xx", []int{200, 201, 202, 204, 299}, false, ResultCounts{5, 5, 0}, 200},
		{"mixed", []int{201, 503, 204}, false, ResultCounts{3, 2, 1}, 500},
		{"truncated 200", []int{200}, true, ResultCounts{1, 0, 1}, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			index := 0

			backend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if tc.truncated {
						w.Header().Set("Content-Length", "100")
					}

					w.WriteHeader(tc.statuses[index])
					index++

					if tc.truncated {
						_, _ = w.Write([]byte("short"))
					}
				}),
			)
			defer backend.Close()

			target, err := url.Parse(backend.URL)
			if err != nil {
				t.Fatal(err)
			}

			ip := target.Hostname()
			target.Host = "fixture.invalid:" + target.Port()
			// Deterministic local destinations exercise the full request and aggregation path.
			lookup := func(context.Context, string) ([]string, error) {
				ips := make([]string, len(tc.statuses))
				for i := range ips {
					ips[i] = ip
				}

				return ips, nil
			}
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/?url="+url.QueryEscape(target.String()),
				http.NoBody,
			)
			broadcast(recorder, request, lookup)

			assertJSONContentType(t, recorder)

			var result Result

			err = json.Unmarshal(recorder.Body.Bytes(), &result)
			if err != nil {
				t.Fatal(err)
			}

			if recorder.Code != tc.status || result.Counts != tc.want {
				t.Fatalf(
					"status=%d counts=%+v; want %d %+v",
					recorder.Code,
					result.Counts,
					tc.status,
					tc.want,
				)
			}

			if tc.truncated &&
				(result.Results[0].StatusCode != http.StatusOK || result.Results[0].Err == "") {
				t.Fatalf("missing body read error: %+v", result.Results[0])
			}
		})
	}
}

func TestBroadcast_PreservesJSONNumbers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		status int
	}{
		{"large integer", `{"id":9007199254740993,"nested":[-9007199254740993]}`, http.StatusOK},
		{"large exponent", `{"value":1e400}`, http.StatusOK},
		{"error response", `{"id":9007199254740993}`, http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
				}),
			)
			defer backend.Close()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
				"/?url="+url.QueryEscape(backend.URL), http.NoBody)
			recorder := httptest.NewRecorder()
			broadcast(recorder, request, func(context.Context, string) ([]string, error) {
				return []string{"127.0.0.1"}, nil
			})

			var result struct {
				Results []struct {
					Err    string          `json:"err"`
					Result json.RawMessage `json:"result"`
				} `json:"results"`
				Counts ResultCounts `json:"counts"`
			}

			err := json.Unmarshal(recorder.Body.Bytes(), &result)
			if err != nil {
				t.Fatal(err)
			}

			if len(result.Results) != 1 {
				t.Fatalf("results = %s", recorder.Body.String())
			}

			if string(result.Results[0].Result) != tc.body || result.Results[0].Err != "" {
				t.Errorf(
					"upstream JSON changed: result=%s err=%q; want %s",
					result.Results[0].Result,
					result.Results[0].Err,
					tc.body,
				)
			}

			wantStatus := http.StatusOK
			wantCounts := ResultCounts{1, 1, 0}

			if tc.status >= http.StatusMultipleChoices {
				wantStatus = http.StatusInternalServerError
				wantCounts = ResultCounts{1, 0, 1}
			}

			if recorder.Code != wantStatus || result.Counts != wantCounts {
				t.Errorf(
					"status=%d counts=%+v; want %d %+v",
					recorder.Code,
					result.Counts,
					wantStatus,
					wantCounts,
				)
			}
		})
	}
}

func assertJSONContentType(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()

	response := recorder.Result()
	defer response.Body.Close()

	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		t.Errorf("Content-Type = %q; want application/json", response.Header.Get("Content-Type"))
	}
}
