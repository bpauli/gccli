package garminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchExerciseCatalog_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"categories":{"BENCH_PRESS":{"exercises":{}}}}`))
	}))
	defer server.Close()

	orig := fetchExercisesURL
	fetchExercisesURL = server.URL
	defer func() { fetchExercisesURL = orig }()

	data, err := FetchExerciseCatalog(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty response")
	}
}

func TestFetchExerciseCatalog_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	orig := fetchExercisesURL
	fetchExercisesURL = server.URL
	defer func() { fetchExercisesURL = orig }()

	data, err := FetchExerciseCatalog(context.Background())
	if err != nil {
		t.Fatalf("a 204 is an empty catalog, got error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("a 204 is an empty catalog, got %q", data)
	}
}

func TestFetchExerciseCatalog_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	orig := fetchExercisesURL
	fetchExercisesURL = server.URL
	defer func() { fetchExercisesURL = orig }()

	_, err := FetchExerciseCatalog(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchExerciseCatalog_TimesOutWaitingForHeaders(t *testing.T) {
	origTimeout := stallTimeout
	t.Cleanup(func() { stallTimeout = origTimeout })
	stallTimeout = 200 * time.Millisecond

	stop := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-stop:
		}
	}))
	t.Cleanup(server.Close)
	t.Cleanup(func() { close(stop) })

	origURL := fetchExercisesURL
	fetchExercisesURL = server.URL
	t.Cleanup(func() { fetchExercisesURL = origURL })

	done := make(chan error, 1)
	go func() {
		_, err := FetchExerciseCatalog(context.Background())
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FetchExerciseCatalog returned nil error, want a timeout error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("FetchExerciseCatalog still waiting after 3s, want a timeout error")
	}
}
