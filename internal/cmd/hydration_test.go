package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bpauli/gccli/internal/outfmt"
)

func TestHydrationAdd_ServerError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/usersummary-service/usersummary/hydration/log", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("error"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	store := newTestSecretsStore(t)
	overrideLoadSecrets(t, store)
	overrideNewClient(t, server)
	storeTestTokens(t, store, "test@example.com", testTokens())

	overrideNowFn(t, time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC))

	var buf bytes.Buffer
	g := testGlobals(t, &buf, outfmt.JSON, "test@example.com")
	cmd := &HydrationAddCmd{Amount: 500}
	err := cmd.Run(g)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
