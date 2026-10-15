package harness

import "time"

func Poll(timeout, interval time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for !check() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(interval)
	}
	return true
}
