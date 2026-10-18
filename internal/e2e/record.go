//go:build e2e

package e2e

import (
	"os"
	"sync"
	"testing"

	"github.com/bpauli/gccli/internal/e2e/harness"
	"github.com/bpauli/gccli/internal/garminapi"
)

var (
	recordOnce sync.Once
	recorder   *harness.Recorder
	recordErr  error
)

func recordOptions(t *testing.T) []garminapi.ClientOption {
	t.Helper()
	dir := os.Getenv("GCCLI_RECORD_DIR")
	if dir == "" {
		return nil
	}
	recordOnce.Do(func() { recorder, recordErr = harness.NewRecorder(dir) })
	if recordErr != nil {
		t.Fatalf("GCCLI_RECORD_DIR: %v", recordErr)
	}
	return []garminapi.ClientOption{garminapi.WithAttemptTransport(recorder.Wrap)}
}
