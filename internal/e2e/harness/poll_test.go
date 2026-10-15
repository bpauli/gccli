package harness

import (
	"testing"
	"time"
)

func TestPollRetriesUntilFound(t *testing.T) {
	calls := 0
	found := Poll(time.Second, time.Millisecond, func() bool {
		calls++
		return calls == 3
	})
	if !found || calls != 3 {
		t.Errorf("Poll = %v after %d calls, want true after 3", found, calls)
	}
}

func TestPollGivesUpAfterTimeout(t *testing.T) {
	start := time.Now()
	found := Poll(50*time.Millisecond, 10*time.Millisecond, func() bool { return false })
	if found {
		t.Error("Poll = true, want false")
	}
	if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
		t.Errorf("Poll gave up after %v, want at least 50ms", elapsed)
	}
}
