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

// maxRetryAfter is the longest Retry-After wait the transport sits through.
// A 429 with a longer wait goes back to the caller, which tells the user
// how long to wait instead of blocking the command silently.
const maxRetryAfter = 30 * time.Second

// readMethods lists the methods that are resent after a 5xx. Writes are not:
// a 5xx can come after the backend saved the change, and Garmin's PUT is not
// always idempotent (AddHydration adds to the day's total).
var readMethods = map[string]bool{
	http.MethodGet:  true,
	http.MethodHead: true,
}

// RetryTransport wraps an http.RoundTripper and retries requests on
// 429 (Too Many Requests) and 5xx responses with backoff.
//
// A 429 is retried for every method, waiting for the Retry-After header,
// unless that wait is longer than maxRetryAfter. A 5xx is retried only for
// GET and HEAD unless Retry5xxAnyMethod is set.
type RetryTransport struct {
	// Base is the underlying RoundTripper. If nil, http.DefaultTransport is used.
	Base http.RoundTripper

	// MaxRetries429 is the maximum number of retries for 429 responses. Default: 3.
	MaxRetries429 int

	// MaxRetries5xx is the maximum number of retries for 5xx responses. Default: 2.
	MaxRetries5xx int

	// Retry5xxAnyMethod also retries writes on 5xx. Set it only for
	// requests that are safe to send twice.
	Retry5xxAnyMethod bool

	// sleepFn is used for testing to override time.Sleep behavior.
	sleepFn func(ctx context.Context, d time.Duration) error
}

// retryCount tracks the retries already spent on one request.
type retryCount struct {
	on429 int
	on5xx int
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

	var spent retryCount
	for {
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := t.base().RoundTrip(req)
		if err != nil {
			return nil, err
		}

		delay, retry := t.retryDelay(req.Method, resp, &spent)
		if !retry {
			return resp, nil
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		if err := t.sleep(req.Context(), delay); err != nil {
			return nil, err
		}
	}
}

// retryDelay decides whether to send the request again after resp and how
// long to wait first. It counts the retry in spent when it says yes.
func (t *RetryTransport) retryDelay(method string, resp *http.Response, spent *retryCount) (time.Duration, bool) {
	switch {
	case resp.StatusCode == http.StatusTooManyRequests:
		delay := parseRetryAfter(resp.Header.Get("Retry-After"))
		if spent.on429 >= t.maxRetries429() || delay > maxRetryAfter {
			return 0, false
		}
		spent.on429++
		return delay, true
	case resp.StatusCode >= 500:
		safeToResend := t.Retry5xxAnyMethod || readMethods[method]
		if spent.on5xx >= t.maxRetries5xx() || !safeToResend {
			return 0, false
		}
		spent.on5xx++
		return exponentialBackoff(spent.on5xx), true
	}
	return 0, false
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

// parseRetryAfter parses the Retry-After header value as seconds or as an
// HTTP date. Returns a minimum of 1 second if the header is missing,
// unparseable or in the past.
func parseRetryAfter(val string) time.Duration {
	if secs, err := strconv.Atoi(val); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if at, err := http.ParseTime(val); err == nil {
		if d := time.Until(at); d > time.Second {
			return d
		}
	}
	return time.Second
}

// exponentialBackoff returns a backoff duration with jitter for retry attempt n (1-based).
// Base delay is 500ms, doubling each attempt, with up to 25% jitter.
func exponentialBackoff(attempt int) time.Duration {
	base := 500.0 * math.Pow(2, float64(attempt-1))
	jitter := base * 0.25 * rand.Float64()
	return time.Duration(base+jitter) * time.Millisecond
}
