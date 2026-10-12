package garminapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

// These tests build the client with NewClient and WithBaseURL only, so the
// default HTTP client is what runs the request.

// scriptedServer answers the n-th request with statuses[n-1], and with the
// last status once the script is used up. It returns the request counter.
func scriptedServer(t *testing.T, retryAfter string, statuses ...int) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := int(atomic.AddInt32(&calls, 1))
		status := statuses[min(n, len(statuses))-1]
		if status == http.StatusTooManyRequests && retryAfter != "" {
			w.Header().Set("Retry-After", retryAfter)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"n":` + strconv.Itoa(n) + `}`))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

// hookSleep replaces the retry sleep of the client's default transport so the
// tests run fast. Without a RetryTransport there is nothing to hook, and the
// request counts below report the missing retry.
func hookSleep(c *Client, fn func(context.Context, time.Duration) error) {
	if rt, ok := c.httpClient.Transport.(*RetryTransport); ok {
		rt.sleepFn = fn
	}
}

func TestNewClient_DefaultClientRetries429AfterRetryAfter(t *testing.T) {
	srv, calls := scriptedServer(t, "1", http.StatusTooManyRequests, http.StatusOK)
	client := NewClient(testTokens(), WithBaseURL(srv.URL))

	var slept []time.Duration
	hookSleep(client, func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	})

	data, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}
	if string(data) != `{"n":2}` {
		t.Errorf("data = %s, want the second response", data)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
	if len(slept) != 1 || slept[0] != time.Second {
		t.Errorf("sleeps = %v, want [1s] from Retry-After", slept)
	}
}

func TestNewClient_LongRetryAfterIsNotRetried(t *testing.T) {
	srv, calls := scriptedServer(t, "120", http.StatusTooManyRequests, http.StatusOK)
	client := NewClient(testTokens(), WithBaseURL(srv.URL))
	hookSleep(client, func(_ context.Context, d time.Duration) error {
		t.Errorf("slept %v, want no sleep", d)
		return nil
	})

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("error = %T (%v), want *RateLimitError", err, err)
	}
	if rlErr.RetryAfter != "120" {
		t.Errorf("RetryAfter = %q, want 120", rlErr.RetryAfter)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

func TestNewClient_DefaultClientRetries5xxForGET(t *testing.T) {
	srv, calls := scriptedServer(t, "", http.StatusServiceUnavailable, http.StatusOK)
	client := NewClient(testTokens(), WithBaseURL(srv.URL))
	hookSleep(client, func(context.Context, time.Duration) error { return nil })

	data, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	if err != nil {
		t.Fatalf("ConnectAPI: %v", err)
	}
	if string(data) != `{"n":2}` {
		t.Errorf("data = %s, want the second response", data)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Errorf("requests = %d, want 2", got)
	}
}

func TestNewClient_DefaultClientDoesNotRetryWritesOn5xx(t *testing.T) {
	// AddHydration is a PUT that adds to the day's total, and a DELETE that
	// worked behind a 502 answers the retry with 404. Only reads are resent.
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			srv, calls := scriptedServer(t, "", http.StatusBadGateway, http.StatusOK)
			client := NewClient(testTokens(), WithBaseURL(srv.URL))
			hookSleep(client, func(_ context.Context, d time.Duration) error {
				t.Errorf("slept %v, want no sleep", d)
				return nil
			})

			_, err := client.ConnectAPI(context.Background(), method, "/api/path", nil)
			var apiErr *GarminAPIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %T (%v), want *GarminAPIError", err, err)
			}
			if apiErr.StatusCode != http.StatusBadGateway {
				t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
			}
			if got := atomic.LoadInt32(calls); got != 1 {
				t.Errorf("requests = %d, want 1", got)
			}
		})
	}
}

func TestNewClient_LongRetryAfterDateIsNotRetried(t *testing.T) {
	date := time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)
	srv, calls := scriptedServer(t, date, http.StatusTooManyRequests, http.StatusOK)
	client := NewClient(testTokens(), WithBaseURL(srv.URL))
	hookSleep(client, func(_ context.Context, d time.Duration) error {
		t.Errorf("slept %v, want no sleep", d)
		return nil
	})

	_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
	var rlErr *RateLimitError
	if !errors.As(err, &rlErr) {
		t.Fatalf("error = %T (%v), want *RateLimitError", err, err)
	}
	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

func TestNewClient_DefaultClientTimesOutWaitingForHeaders(t *testing.T) {
	orig := responseHeaderTimeout
	t.Cleanup(func() { responseHeaderTimeout = orig })
	responseHeaderTimeout = 200 * time.Millisecond

	stop := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-stop:
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(stop) })
	client := NewClient(testTokens(), WithBaseURL(srv.URL))

	done := make(chan error, 1)
	start := time.Now()
	go func() {
		_, err := client.ConnectAPI(context.Background(), http.MethodGet, "/api/path", nil)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ConnectAPI returned nil error, want a timeout error")
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Errorf("ConnectAPI failed after %v, want under 2s", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("ConnectAPI still waiting after 3s, want a timeout error")
	}
}
