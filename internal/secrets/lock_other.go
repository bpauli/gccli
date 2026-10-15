//go:build !unix

package secrets

import "os"

// Without flock, gccli cannot tell a temp dir of a killed write from one of
// a write in flight, so it never removes temp dirs here.
func lockShared(*os.File) error { return nil }

func tryLockExclusive(*os.File) bool { return false }
