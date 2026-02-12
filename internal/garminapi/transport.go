package garminapi

import (
	"bytes"
	"context"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// RetryTransport wraps an http.RoundTripper and retries requests on
// 429 (Too Many Requests) and 5xx responses with backoff.
type RetryTransport struct {
	// Base is the underlying RoundTripper. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// MaxRetries429 is the maximum number of retries for 429 responses. Default: 3.
	MaxRetries429 int

	// MaxRetries5xx is the maximum number of retries for 5xx responses. Default: 2.
	MaxRetries5xx int

	// sleepFn is used for testing to override time.Sleep behavior.
	sleepFn func(ctx context.Context, d time.Duration) error
}

// NewRetryTransport creates a RetryTransport with sensible defaults.
func NewRetryTransport(base http.RoundTripper) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &RetryTransport{
		Base:          base,
		MaxRetries429: 3,
		MaxRetries5xx: 2,
	}
}

// RoundTrip implements http.RoundTripper with retry logic.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	var (
		resp       *http.Response
		err        error
		retries429 int
		retries5xx int
	)

	for {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err = t.base().RoundTrip(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusTooManyRequests && retries429 < t.maxRetries429() {
			retries429++
			delay := parseRetryAfter(resp.Header.Get("Retry-After"))
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if err := t.sleep(req.Context(), delay); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode >= 500 && retries5xx < t.maxRetries5xx() {
			retries5xx++
			delay := exponentialBackoff(retries5xx)
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if err := t.sleep(req.Context(), delay); err != nil {
				return nil, err
			}
			continue
		}

		return resp, nil
	}
}

func (t *RetryTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *RetryTransport) maxRetries429() int {
	if t.MaxRetries429 > 0 {
		return t.MaxRetries429
	}
	return 3
}

func (t *RetryTransport) maxRetries5xx() int {
	if t.MaxRetries5xx > 0 {
		return t.MaxRetries5xx
	}
	return 2
}

func (t *RetryTransport) sleep(ctx context.Context, d time.Duration) error {
	if t.sleepFn != nil {
		return t.sleepFn(ctx, d)
	}
	return contextSleep(ctx, d)
}

// contextSleep sleeps for d, but returns early if ctx is cancelled.
func contextSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// parseRetryAfter parses the Retry-After header value as seconds.
// Returns a minimum of 1 second if the header is missing or unparseable.
func parseRetryAfter(val string) time.Duration {
	if val == "" {
		return time.Second
	}
	secs, err := strconv.Atoi(val)
	if err != nil || secs <= 0 {
		return time.Second
	}
	return time.Duration(secs) * time.Second
}

// exponentialBackoff returns a backoff duration with jitter for retry attempt n (1-based).
// Base delay is 500ms, doubling each attempt, with up to 25% jitter.
func exponentialBackoff(attempt int) time.Duration {
	base := 500.0 * math.Pow(2, float64(attempt-1))
	jitter := base * 0.25 * rand.Float64()
	return time.Duration(base+jitter) * time.Millisecond
}
