package testutil

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

// NewServer creates a test HTTP server and registers cleanup.
func NewServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// NewClientWithServer creates a test HTTP server and a configured API client
// pointing at it. Returns both for test use.
func NewClientWithServer(t *testing.T, handler http.Handler) (*httptest.Server, *garminapi.Client) {
	t.Helper()
	srv := NewServer(t, handler)
	tokens := TestTokens()
	client := garminapi.NewClient(tokens,
		garminapi.WithHTTPClient(srv.Client()),
		garminapi.WithBaseURL(srv.URL),
	)
	return srv, client
}

// JSONHandler returns an http.HandlerFunc that responds with the given JSON
// string and status code.
func JSONHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}
}
