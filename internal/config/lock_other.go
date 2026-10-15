//go:build !unix

package config

// Windows is not a release target, so config writes there are atomic but not
// locked, and token refreshes are not locked.
func LockFile(string) (unlock func(), err error) {
	return func() {}, nil
}
