package harness

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type statusTransport int

func (s statusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: int(s),
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     http.Header{},
		Request:    req,
	}, nil
}

func TestLoginMarksRateLimit(t *testing.T) {
	_, err := Login(context.Background(), "e@example.com", "x", statusTransport(http.StatusTooManyRequests))
	var rateLimited *SSORateLimitError
	if !errors.As(err, &rateLimited) {
		t.Fatalf("err = %v, want a *SSORateLimitError", err)
	}
	if !strings.HasPrefix(err.Error(), "Garmin SSO rate limit: ") {
		t.Errorf("err = %q, want the Garmin SSO rate limit prefix", err)
	}
}

func TestLoginKeepsOtherFailures(t *testing.T) {
	_, err := Login(context.Background(), "e@example.com", "x", statusTransport(http.StatusForbidden))
	if err == nil {
		t.Fatal("expected a login error")
	}
	var rateLimited *SSORateLimitError
	if errors.As(err, &rateLimited) {
		t.Errorf("err = %v, want no rate limit mark without a 429", err)
	}
}
