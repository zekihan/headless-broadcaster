package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestBroadcastRejectsInvalidDestinationBeforeLookup(t *testing.T) {
	var requests atomic.Int64

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	target, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	for _, destination := range []string{
		"/relative/path",
		"//" + target.Host + "/path",
		"http:opaque",
		"http:///missing-host",
		"https:///missing-host",
		"http://:" + target.Port() + "/path",
		"ftp://" + target.Host + "/path",
		"file://" + target.Host + "/path",
	} {
		t.Run(destination, func(t *testing.T) {
			lookups := 0
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet,
				"/?url="+url.QueryEscape(destination), http.NoBody)
			broadcast(recorder, request, func(context.Context, string) ([]string, error) {
				lookups++

				return []string{target.Hostname()}, nil
			})

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status = %d; want 400: %s", recorder.Code, recorder.Body.String())
			}

			if lookups != 0 {
				t.Errorf("invalid destination triggered %d DNS lookups", lookups)
			}
		})
	}

	if requests.Load() != 0 {
		t.Errorf("invalid destinations reached backend %d times", requests.Load())
	}
}
