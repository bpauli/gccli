package harness

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
)

type roundTripFunc func(*http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req), nil }

func page(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{},
		Request:    req,
	}
}

type statusTransport int

func (s statusTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return page(req, int(s), ""), nil
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

func TestLoginKeepsFailureAfterRecoveredRateLimit(t *testing.T) {
	posts := 0
	sso := roundTripFunc(func(req *http.Request) *http.Response {
		if req.Method == http.MethodGet {
			return page(req, http.StatusOK, `<html><input name="_csrf" value="c"></html>`)
		}
		posts++
		if posts == 1 {
			return page(req, http.StatusTooManyRequests, "")
		}
		return page(req, http.StatusOK, "<html><title>Wrong password</title></html>")
	})
	_, err := Login(context.Background(), "e@example.com", "x", sso)
	if err == nil || !strings.Contains(err.Error(), "Wrong password") {
		t.Fatalf("err = %v, want the Wrong password failure", err)
	}
	var rateLimited *SSORateLimitError
	if errors.As(err, &rateLimited) {
		t.Errorf("err = %v, want no rate limit mark after the retry recovered from 429", err)
	}
	if posts != 2 {
		t.Errorf("posts = %d, want 2 (one 429, one retry)", posts)
	}
}

type resetAfter429 struct{ posts int }

func (s *resetAfter429) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == http.MethodGet {
		return page(req, http.StatusOK, `<html><input name="_csrf" value="c"></html>`), nil
	}
	s.posts++
	if s.posts == 1 {
		return page(req, http.StatusTooManyRequests, ""), nil
	}
	return nil, syscall.ECONNRESET
}

func TestLoginKeepsTransportErrorAfterRateLimit(t *testing.T) {
	sso := &resetAfter429{}
	_, err := Login(context.Background(), "e@example.com", "x", sso)
	if !errors.Is(err, syscall.ECONNRESET) {
		t.Fatalf("err = %v, want the connection reset", err)
	}
	var rateLimited *SSORateLimitError
	if errors.As(err, &rateLimited) {
		t.Errorf("err = %v, want no rate limit mark when the request after the 429 got no response", err)
	}
	if sso.posts != 2 {
		t.Errorf("posts = %d, want 2 (one 429, one reset)", sso.posts)
	}
}
