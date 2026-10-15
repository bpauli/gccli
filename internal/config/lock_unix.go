//go:build unix

package config

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// LockFile blocks until it holds an exclusive advisory lock on path. When
// the wait passes one second, LockFile calls waiting once, unless waiting is
// nil. Call unlock to release the lock.
func LockFile(path string, waiting func()) (unlock func(), err error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if waiting != nil {
		done := make(chan struct{})
		t := time.AfterFunc(time.Second, func() {
			defer close(done)
			waiting()
		})
		defer func() {
			if !t.Stop() {
				<-done
			}
		}()
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("lock %s: %w", path, err)
	}
	return func() { _ = f.Close() }, nil
}
