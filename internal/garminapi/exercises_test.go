package garminapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
