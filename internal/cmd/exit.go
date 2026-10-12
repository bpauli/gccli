package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bpauli/gccli/internal/errfmt"
	"github.com/bpauli/gccli/internal/garminapi"
)

// errorKind is how scripts see an error: the process exit code and the
// "code" field of the --json error object. README.md documents both.
type errorKind struct {
	exit int
	name string
}

var (
	kindError     = errorKind{exit: 1, name: "error"}
	kindUsage     = errorKind{exit: 2, name: "usage"}
	kindAuth      = errorKind{exit: 3, name: "auth_required"}
	kindNotFound  = errorKind{exit: 4, name: "not_found"}
	kindRateLimit = errorKind{exit: 5, name: "rate_limited"}
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

// classify returns the kind of err and the HTTP status behind it, or 0.
func classify(err error) (errorKind, int) {
	var authErr *authRequiredError
	var tokenErr *garminapi.TokenExpiredError
	var rateErr *garminapi.RateLimitError
	var apiErr *garminapi.GarminAPIError
	switch {
	case errors.As(err, &authErr):
		return kindAuth, 0
	case errors.As(err, &tokenErr):
		return kindAuth, http.StatusUnauthorized
	case errors.As(err, &rateErr):
		return kindRateLimit, http.StatusTooManyRequests
	case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
		return kindNotFound, apiErr.StatusCode
	case errors.As(err, &apiErr):
		return kindError, apiErr.StatusCode
	}
	return kindError, 0
}

// exitCode returns the process exit code for the given error.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	kind, _ := classify(err)
	return kind.exit
}

type jsonError struct {
	Code    string `json:"code"`
	Status  int    `json:"status,omitempty"`
	Message string `json:"message"`
}

// writeJSONError writes err as {"error":{...}} for --json mode.
func writeJSONError(w io.Writer, err error) {
	kind, status := classify(err)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(struct {
		Error jsonError `json:"error"`
	}{jsonError{Code: kind.name, Status: status, Message: errfmt.Format(err)}})
}

// exitSignal is used internally to intercept Kong's help/version exit calls
// without calling os.Exit.
type exitSignal struct {
	code int
}
