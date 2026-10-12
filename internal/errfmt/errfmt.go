package errfmt

import (
	"errors"
	"fmt"

	"github.com/bpauli/gccli/internal/garminapi"
)

// Format converts an error into an actionable user-facing message.
// Errors the user can act on get a hint; all other errors are
// returned as-is via err.Error(), which keeps the operation context.
func Format(err error) string {
	var tokenErr *garminapi.TokenExpiredError
	if errors.As(err, &tokenErr) {
		if tokenErr.Email != "" {
			return fmt.Sprintf("Token expired. Run: gccli auth login %s", tokenErr.Email)
		}
		return "Token expired. Run: gccli auth login <email>"
	}

	var rateErr *garminapi.RateLimitError
	if errors.As(err, &rateErr) {
		if rateErr.RetryAfter != "" {
			return fmt.Sprintf("Rate limited. Wait %s and retry.", rateErr.RetryAfter)
		}
		return "Rate limited. Wait and retry."
	}

	return err.Error()
}
