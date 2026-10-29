package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zekihan/headless-broadcaster/config"
	"github.com/zekihan/headless-broadcaster/internal/commons/middleware"
)

func securityConfig(t *testing.T) {
	t.Helper()
	t.Setenv("CONFIG_PATH", t.TempDir()+"/absent.yaml")
	config.Reset()
	t.Cleanup(config.Reset)
}

func TestBroadcastCredentialIsolation(t *testing.T) {
	for _, carrier := range []string{
		"header", "query", "cookie", "basic", "redundant",
		"downstream basic", "disabled", "cookie whitespace", "repeated auth",
	} {
		t.Run(carrier, func(t *testing.T) {
			securityConfig(t)

			key := "fixture-broadcaster-key"
			if carrier == "disabled" {
				key = ""
			}

			t.Setenv("HB__WEB_API_KEY", key)

			var received http.Header

			backend := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					received = r.Header.Clone()
					if r.URL.Query().Get("apiKey") != "downstream-query" {
						t.Error("target query changed")
					}
				}),
			)
			defer backend.Close()

			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				"/broadcast?url="+url.QueryEscape(
					backend.URL+"?apiKey=downstream-query",
				),
				http.NoBody,
			)
			req.Header["X-Tag"] = []string{"one", "two"}
			req.Header["Cookie"] = []string{"session=one; repeated=a", "repeated=b"}
			req.Header.Set("Authorization", "Bearer downstream-token")

			wantAuthorization := configureCredential(req, key, carrier)

			original := req.Header.Clone()
			handler := middleware.GetAuthMiddlewareForSingleAPIKey(
				key,
			)(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					broadcast(
						w,
						r,
						func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil },
					)
				}),
			)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("%d: %s", recorder.Code, recorder.Body.String())
			}

			if received.Get("X-Api-Key") != "" ||
				strings.Contains(strings.Join(received.Values("Cookie"), ";"), "apiKey") ||
				len(received.Values("Authorization")) > 1 {
				t.Fatalf("credential leaked: %v", received)
			}

			if received.Get("Authorization") != wantAuthorization {
				t.Fatalf("Authorization = %q", received.Get("Authorization"))
			}

			if !reflect.DeepEqual(received.Values("X-Tag"), []string{"one", "two"}) {
				t.Fatal("repeated headers lost")
			}

			checkApplicationCookies(t, received)

			if !reflect.DeepEqual(req.Header, original) {
				t.Fatal("input headers mutated")
			}
		})
	}
}

func TestBroadcastRequestBudget(t *testing.T) {
	securityConfig(t)
	t.Setenv("HB__PROXY_MAX_REQUEST_BODY_BYTES", "8")

	var calls atomic.Int64

	backend := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }),
	)
	defer backend.Close()

	for _, size := range []int{0, 7, 8, 9} {
		for _, unknown := range []bool{false, true} {
			req := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodPost,
				"/?url="+url.QueryEscape(backend.URL),
				strings.NewReader(strings.Repeat("x", size)),
			)
			if unknown {
				req.ContentLength = -1
				req.TransferEncoding = []string{"chunked"}
			}

			before := calls.Load()
			recorder := httptest.NewRecorder()
			broadcast(
				recorder,
				req,
				func(context.Context, string) ([]string, error) { return []string{"127.0.0.1"}, nil },
			)

			want := http.StatusOK
			if size > 8 {
				want = http.StatusRequestEntityTooLarge
			}

			if recorder.Code != want {
				t.Fatalf(
					"size=%d unknown=%v: %d %s",
					size,
					unknown,
					recorder.Code,
					recorder.Body.String(),
				)
			}

			if size > 8 && calls.Load() != before {
				t.Fatal("oversized request sent upstream")
			}
		}
	}
}

func TestBroadcastResponseBudgetKeepsPartialSuccess(t *testing.T) {
	securityConfig(t)
	t.Setenv("HB__PROXY_MAX_RESPONSE_BODY_BYTES", "8")

	var calls atomic.Int64

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 2 {
			_ = http.NewResponseController(w).Flush()
			_, _ = w.Write([]byte("too large"))

			return
		}

		_, _ = w.Write([]byte("success"))
	}))
	defer backend.Close()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/?url="+url.QueryEscape(backend.URL),
		http.NoBody,
	)
	broadcast(recorder, req, func(context.Context, string) ([]string, error) {
		return []string{"127.0.0.1", "127.0.0.1", "127.0.0.1"}, nil
	})

	var result Result

	err := json.Unmarshal(recorder.Body.Bytes(), &result)
	if err != nil {
		t.Fatal(err)
	}

	if recorder.Code != http.StatusInternalServerError || result.Counts != (ResultCounts{3, 2, 1}) {
		t.Fatalf("%d %+v", recorder.Code, result)
	}

	if result.Results[0].Result != "success" || result.Results[2].Result != "success" ||
		result.Results[1].Err == "" ||
		result.Results[1].Result != nil {
		t.Fatalf("results: %+v", result.Results)
	}
}

func checkApplicationCookies(t *testing.T, received http.Header) {
	t.Helper()

	cookies := (&http.Request{Header: received}).Cookies()
	if len(cookies) < 3 || cookies[0].Name != "session" || cookies[1].Value != "a" ||
		cookies[2].Value != "b" {
		t.Fatalf("cookies lost: %v", cookies)
	}
}

func configureCredential(req *http.Request, key, carrier string) string {
	wantAuthorization := "Bearer downstream-token"

	switch carrier {
	case "cookie whitespace":
		req.Header.Add("Cookie", "apiKey ="+key)
	case "repeated auth":
		req.Header.Set("X-Api-Key", key)
		basic := &http.Request{Header: make(http.Header)}
		basic.SetBasicAuth("broadcaster", key)
		req.Header.Add("Authorization", basic.Header.Get("Authorization"))
	case "header":
		req.Header.Set("X-Api-Key", key)
	case "query":
		req.URL.RawQuery += "&apiKey=" + key
	case "cookie":
		req.Header.Add("Cookie", "apiKey="+key+"; app=ok; apiKey="+key)
	case "basic":
		req.SetBasicAuth("broadcaster", key)

		wantAuthorization = ""
	case "redundant":
		req.Header.Set("X-Api-Key", key)
		req.Header.Add("Cookie", "apiKey="+key)
		req.SetBasicAuth("broadcaster", key)

		wantAuthorization = ""
	case "downstream basic":
		req.Header.Set("X-Api-Key", key)
		req.SetBasicAuth("application", "downstream-password")
		wantAuthorization = req.Header.Get("Authorization")
	case "disabled":
		req.SetBasicAuth("application", "downstream-password")
		wantAuthorization = req.Header.Get("Authorization")
	}

	return wantAuthorization
}
