package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequestToIP_ReclaimsConnections(t *testing.T) {
	for _, mode := range []string{"success", "http error", "truncated", "canceled"} {
		t.Run(mode, func(t *testing.T) {
			testReclaimedConnections(t, mode)
		})
	}
}

func testReclaimedConnections(t *testing.T, mode string) {
	t.Helper()

	var opened, closed atomic.Int64

	reclaimed := make(chan struct{}, 30)

	started := make(chan struct{}, 1)
	server := httptest.NewUnstartedServer(transportFixture(t, mode, started))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			opened.Add(1)
		}

		if state == http.StateClosed {
			closed.Add(1)

			reclaimed <- struct{}{}
		}
	}

	server.Start()
	defer server.Close()

	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	ip := target.Hostname()
	target.Host = "target.invalid:" + target.Port() // Dial must remain bound to the supplied IP.

	for range 30 {
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)

		if mode == "canceled" {
			go func() {
				select {
				case <-started:
					cancel()
				case <-ctx.Done():
				}
			}()
		}

		result, err := RequestToIP(ctx, ip, target, http.MethodGet, nil, nil)

		cancel()

		if err != nil {
			t.Fatal(err)
		}

		checkTransportResult(t, mode, result)

		select {
		case <-reclaimed:
		case <-time.After(time.Second):
			t.Fatal("connection was not reclaimed after request")
		}

		if opened.Load() != closed.Load() {
			t.Fatalf(
				"connections not reclaimed: opened=%d closed=%d",
				opened.Load(),
				closed.Load(),
			)
		}
	}

	if opened.Load() != 30 {
		t.Fatalf("expected 30 one-shot connections, got %d", opened.Load())
	}
}

func checkTransportResult(t *testing.T, mode string, result *Result) {
	t.Helper()

	wantError := mode == "truncated" || mode == "canceled"
	if (result.Err != "") != wantError {
		t.Fatalf("unexpected result: %+v", result)
	}

	wantStatus := http.StatusOK
	if mode == "http error" {
		wantStatus = http.StatusServiceUnavailable
	}

	if result.StatusCode != wantStatus &&
		(mode != "canceled" || result.StatusCode != 0) {
		t.Fatalf("status=%d; want %d", result.StatusCode, wantStatus)
	}
}

func transportFixture(t *testing.T, mode string, started chan<- struct{}) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch mode {
		case "http error":
			w.WriteHeader(http.StatusServiceUnavailable)
		case "truncated":
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte("short"))
		case "canceled":
			w.WriteHeader(http.StatusOK)

			err := http.NewResponseController(w).Flush()
			if err != nil {
				t.Error(err)
			}

			started <- struct{}{}

			<-r.Context().Done()
		default:
			_, _ = w.Write([]byte("ok"))
		}
	})
}
