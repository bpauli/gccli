package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bpauli/gccli/internal/garminapi"
)

// Process exit codes. README.md documents them for scripts.
const (
	exitError     = 1
	exitUsage     = 2
	exitAuth      = 3
	exitNotFound  = 4
	exitRateLimit = 5
)

// authRequiredError means the account has no usable credentials.
type authRequiredError struct {
	msg string
}

func (e *authRequiredError) Error() string { return e.msg }

func errNoAccount() error {
	return &authRequiredError{msg: "no account specified; use --account, set GCCLI_ACCOUNT, or run gccli auth login"}
}

func errNoCredentials(email string) error {
	return &authRequiredError{msg: fmt.Sprintf("no credentials stored for %s; run: gccli auth login %s", email, email)}
}

// exitCode returns the process exit code for the given error.
func exitCode(err error) int {
	var authErr *authRequiredError
	var tokenErr *garminapi.TokenExpiredError
	var rateErr *garminapi.RateLimitError
	var apiErr *garminapi.GarminAPIError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &authErr), errors.As(err, &tokenErr):
		return exitAuth
	case errors.As(err, &rateErr):
		return exitRateLimit
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
		return exitNotFound
	}
	return exitError
}

// exitSignal is used internally to intercept Kong's help/version exit calls
// without calling os.Exit.
type exitSignal struct {
	code int
}
