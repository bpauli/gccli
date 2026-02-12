package garminapi

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// roundTripFunc is an adapter to use a function as an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// newResponse creates a minimal *http.Response with the given status and body.
func newResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// --- RetryTransport tests ---

func TestRetryTransport_NoRetryOnSuccess(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return newResponse(200, `{"ok":true}`), nil
	})

	rt := NewRetryTransport(base)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want 1", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_Retry429(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n <= 2 {
			resp := newResponse(429, "")
			resp.Header.Set("Retry-After", "1")
			return resp, nil
		}
		return newResponse(200, "ok"), nil
	})

	rt := NewRetryTransport(base)
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_429ExhaustsRetries(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		resp := newResponse(429, "")
		resp.Header.Set("Retry-After", "1")
		return resp, nil
	})

	rt := NewRetryTransport(base)
	rt.MaxRetries429 = 2
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	// After exhausting retries, returns the 429 response.
	if resp.StatusCode != 429 {
		t.Errorf("StatusCode = %d, want 429", resp.StatusCode)
	}
	// 1 initial + 2 retries = 3
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_Retry5xx(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			return newResponse(503, "unavailable"), nil
		}
		return newResponse(200, "ok"), nil
	})

	rt := NewRetryTransport(base)
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_5xxExhaustsRetries(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return newResponse(500, "error"), nil
	})

	rt := NewRetryTransport(base)
	rt.MaxRetries5xx = 2
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", resp.StatusCode)
	}
	// 1 initial + 2 retries = 3
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_NoRetryOn4xx(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		atomic.AddInt32(&calls, 1)
		return newResponse(404, "not found"), nil
	})

	rt := NewRetryTransport(base)
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (no retry for 4xx)", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_PreservesBody(t *testing.T) {
	var bodies []string
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		bodies = append(bodies, string(b))
		if len(bodies) == 1 {
			return newResponse(500, "error"), nil
		}
		return newResponse(200, "ok"), nil
	})

	rt := NewRetryTransport(base)
	rt.MaxRetries5xx = 1
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	body := strings.NewReader(`{"data":"test"}`)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://example.com/api", body)
	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	if len(bodies) != 2 {
		t.Fatalf("got %d requests, want 2", len(bodies))
	}
	for i, b := range bodies {
		if b != `{"data":"test"}` {
			t.Errorf("request %d body = %q, want {\"data\":\"test\"}", i, b)
		}
	}
}

func TestRetryTransport_RetryAfterHeader(t *testing.T) {
	var sleepDurations []time.Duration
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		if len(sleepDurations) == 0 {
			resp := newResponse(429, "")
			resp.Header.Set("Retry-After", "5")
			return resp, nil
		}
		return newResponse(200, "ok"), nil
	})

	rt := NewRetryTransport(base)
	rt.sleepFn = func(_ context.Context, d time.Duration) error {
		sleepDurations = append(sleepDurations, d)
		return nil
	}

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}

	if len(sleepDurations) != 1 {
		t.Fatalf("sleeps = %d, want 1", len(sleepDurations))
	}
	if sleepDurations[0] != 5*time.Second {
		t.Errorf("sleep duration = %v, want 5s", sleepDurations[0])
	}
}

func TestRetryTransport_ContextCancelled(t *testing.T) {
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return newResponse(500, "error"), nil
	})

	rt := NewRetryTransport(base)
	// Real sleep that respects context cancellation.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com/api", nil)
	_, err := rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestRetryTransport_Mixed429And5xx(t *testing.T) {
	var calls int32
	base := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		n := atomic.AddInt32(&calls, 1)
		switch n {
		case 1:
			resp := newResponse(429, "")
			resp.Header.Set("Retry-After", "1")
			return resp, nil
		case 2:
			return newResponse(503, "unavailable"), nil
		default:
			return newResponse(200, "ok"), nil
		}
	})

	rt := NewRetryTransport(base)
	rt.sleepFn = func(_ context.Context, _ time.Duration) error { return nil }

	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com/api", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", atomic.LoadInt32(&calls))
	}
}

func TestRetryTransport_NilBase(t *testing.T) {
	rt := NewRetryTransport(nil)
	if rt.Base == nil {
		t.Error("expected non-nil Base after NewRetryTransport(nil)")
	}
}

// --- parseRetryAfter tests ---

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		val  string
		want time.Duration
	}{
		{"", time.Second},
		{"0", time.Second},
		{"-1", time.Second},
		{"abc", time.Second},
		{"1", time.Second},
		{"5", 5 * time.Second},
		{"60", 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			got := parseRetryAfter(tt.val)
			if got != tt.want {
				t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

// --- exponentialBackoff tests ---

func TestExponentialBackoff(t *testing.T) {
	// Attempt 1: base=500ms, jitter up to 125ms → [500ms, 625ms]
	d1 := exponentialBackoff(1)
	if d1 < 500*time.Millisecond || d1 > 625*time.Millisecond {
		t.Errorf("attempt 1: %v, want [500ms, 625ms]", d1)
	}

	// Attempt 2: base=1000ms, jitter up to 250ms → [1000ms, 1250ms]
	d2 := exponentialBackoff(2)
	if d2 < 1000*time.Millisecond || d2 > 1250*time.Millisecond {
		t.Errorf("attempt 2: %v, want [1000ms, 1250ms]", d2)
	}

	// Attempt 3: base=2000ms, jitter up to 500ms → [2000ms, 2500ms]
	d3 := exponentialBackoff(3)
	if d3 < 2000*time.Millisecond || d3 > 2500*time.Millisecond {
		t.Errorf("attempt 3: %v, want [2000ms, 2500ms]", d3)
	}
}

// --- contextSleep tests ---

func TestContextSleep_Completes(t *testing.T) {
	ctx := context.Background()
	start := time.Now()
	err := contextSleep(ctx, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("contextSleep: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 10*time.Millisecond {
		t.Errorf("elapsed = %v, want >= 10ms", elapsed)
	}
}

func TestContextSleep_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := contextSleep(ctx, time.Hour)
	if err != context.Canceled {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}
