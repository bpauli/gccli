package cmd

import (
	"errors"
	"fmt"
)

// ExitError wraps an error with a specific process exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// exitCode returns the appropriate exit code for the given error.
// Returns 0 for nil, the ExitError code if present, or 1 for all other errors.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *ExitError
	if errors.As(err, &exitErr) {
		return exitErr.Code
	}
	return 1
}

// exitSignal is used internally to intercept Kong's help/version exit calls
// without calling os.Exit.
type exitSignal struct {
	code int
}
