package harness

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"sync/atomic"

	"github.com/bpauli/gccli/internal/garminauth"
)

type SSORateLimitError struct{ err error }

func (e *SSORateLimitError) Error() string { return "Garmin SSO rate limit: " + e.err.Error() }

func (e *SSORateLimitError) Unwrap() error { return e.err }

func Login(ctx context.Context, email, password string, base http.RoundTripper) (*garminauth.Tokens, error) {
	jar, _ := cookiejar.New(nil)
	watch := &rateLimitWatch{base: base}
	tokens, err := garminauth.LoginHeadless(ctx, email, password, garminauth.LoginOptions{
		HTTPClient: &http.Client{Jar: jar, Transport: watch},
	})
	if err != nil && watch.hit.Load() {
		return nil, &SSORateLimitError{err: err}
	}
	return tokens, err
}

type rateLimitWatch struct {
	base http.RoundTripper
	hit  atomic.Bool
}

func (w *rateLimitWatch) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := w.base.RoundTrip(req)
	if err == nil && resp.StatusCode == http.StatusTooManyRequests {
		w.hit.Store(true)
	}
	return resp, err
}
