//go:build e2e

package e2e

import "testing"

// RegisterCleanup registers a cleanup function that will run when the test
// finishes, even on failure or panic. This is a thin wrapper around
// t.Cleanup() to provide a consistent naming convention and make the
// intent explicit in E2E tests.
//
// Usage:
//
//	e2e.RegisterCleanup(t, func() {
//	    _ = client.DeleteActivity(ctx, activityID)
//	})
func RegisterCleanup(t *testing.T, fn func()) {
	t.Helper()
	t.Cleanup(fn)
}
