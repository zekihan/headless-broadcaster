package api

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestBroadcast_CompressedResponses(t *testing.T) {
	for _, tc := range []struct {
		name       string
		accept     string
		body       string
		want       string
		status     int
		corrupt    bool
		wantFailed bool
	}{
		{"automatic gzip", "", `{"ok":true}`, `{"ok":true}`, http.StatusOK, false, false},
		{"explicit gzip", "gzip", `{"ok":true}`, `{"ok":true}`, http.StatusOK, false, false},
		{"browser encodings", "gzip, deflate, br", "hello", `"hello"`, http.StatusOK, false, false},
		{"upstream error", "gzip", `"unavailable"`, `"unavailable"`, http.StatusServiceUnavailable, false, true},
		{"corrupt gzip", "gzip", "", "", http.StatusOK, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			backend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if got := r.Header.Get("Accept-Encoding"); got != "gzip" {
						t.Errorf("upstream Accept-Encoding = %q; want transport-managed gzip", got)
					}

					w.Header().Set("Content-Encoding", "gzip")
					w.WriteHeader(tc.status)

					if tc.corrupt {
						_, _ = w.Write([]byte("invalid gzip"))

						return
					}

					writer := gzip.NewWriter(w)
					_, _ = writer.Write([]byte(tc.body))
					_ = writer.Close()
				}),
			)
			defer backend.Close()

			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
				"/broadcast?url="+url.QueryEscape(backend.URL), http.NoBody)
			if tc.accept != "" {
				request.Header.Set("Accept-Encoding", tc.accept)
			}

			recorder := httptest.NewRecorder()
			broadcast(recorder, request, func(context.Context, string) ([]string, error) {
				return []string{"127.0.0.1"}, nil
			})

			result := decodeCompressedAggregate(t, recorder)

			upstream := result.Results[0]
			if upstream.StatusCode != tc.status || (upstream.Err != "") != tc.corrupt ||
				string(upstream.Result) != tc.want {
				t.Errorf("upstream result = %+v; want status=%d payload=%s corrupt=%v",
					upstream, tc.status, tc.want, tc.corrupt)
			}

			wantStatus := http.StatusOK
			wantCounts := ResultCounts{1, 1, 0}

			if tc.wantFailed {
				wantStatus = http.StatusInternalServerError
				wantCounts = ResultCounts{1, 0, 1}
			}

			if recorder.Code != wantStatus || result.Counts != wantCounts {
				t.Errorf("status=%d counts=%+v; want %d %+v",
					recorder.Code, result.Counts, wantStatus, wantCounts)
			}

			if request.Header.Get("Accept-Encoding") != tc.accept {
				t.Error("caller headers were mutated")
			}
		})
	}
}

type compressedAggregate struct {
	Results []struct {
		Err        string          `json:"err"`
		Result     json.RawMessage `json:"result"`
		StatusCode int             `json:"statusCode"`
	} `json:"results"`
	Counts ResultCounts `json:"counts"`
}

func decodeCompressedAggregate(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
) compressedAggregate {
	t.Helper()

	var result compressedAggregate

	err := json.Unmarshal(recorder.Body.Bytes(), &result)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Results) != 1 {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}

	return result
}
