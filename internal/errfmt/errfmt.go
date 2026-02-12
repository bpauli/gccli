package errfmt

import (
	"errors"
	"fmt"
	"net/http"
)

// AuthRequiredError indicates no authentication credentials were found.
type AuthRequiredError struct {
	Email string
}

func (e *AuthRequiredError) Error() string {
	if e.Email != "" {
		return fmt.Sprintf("no auth found for %s", e.Email)
	}
	return "no auth found"
}

// TokenExpiredError indicates the stored token has expired.
type TokenExpiredError struct {
	Email string
}

func (e *TokenExpiredError) Error() string {
	if e.Email != "" {
		return fmt.Sprintf("token expired for %s", e.Email)
	}
	return "token expired"
}

// RateLimitError indicates the API returned a rate limit response.
type RateLimitError struct {
	RetryAfter string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("rate limited; retry after %s", e.RetryAfter)
	}
	return "rate limited"
}

// GarminAPIError represents an error response from the Garmin Connect API.
type GarminAPIError struct {
	StatusCode int
	Message    string
}

func (e *GarminAPIError) Error() string {
	status := http.StatusText(e.StatusCode)
	if status == "" {
		status = fmt.Sprintf("%d", e.StatusCode)
	} else {
		status = fmt.Sprintf("%d %s", e.StatusCode, status)
	}
	if e.Message != "" {
		return fmt.Sprintf("garmin api: %s: %s", status, e.Message)
	}
	return fmt.Sprintf("garmin api: %s", status)
}

// Format converts an error into an actionable user-facing message.
// Known error types produce specific guidance; unknown errors are
// returned as-is via err.Error().
func Format(err error) string {
	var authErr *AuthRequiredError
	if errors.As(err, &authErr) {
		if authErr.Email != "" {
			return fmt.Sprintf("No auth found. Run: gccli auth login %s", authErr.Email)
		}
		return "No auth found. Run: gccli auth login <email>"
	}

	var tokenErr *TokenExpiredError
	if errors.As(err, &tokenErr) {
		if tokenErr.Email != "" {
			return fmt.Sprintf("Token expired. Run: gccli auth login %s", tokenErr.Email)
		}
		return "Token expired. Run: gccli auth login <email>"
	}

	var rateErr *RateLimitError
	if errors.As(err, &rateErr) {
		if rateErr.RetryAfter != "" {
			return fmt.Sprintf("Rate limited. Wait %s and retry.", rateErr.RetryAfter)
		}
		return "Rate limited. Wait and retry."
	}

	var apiErr *GarminAPIError
	if errors.As(err, &apiErr) {
		status := http.StatusText(apiErr.StatusCode)
		if status == "" {
			status = fmt.Sprintf("%d", apiErr.StatusCode)
		} else {
			status = fmt.Sprintf("%d %s", apiErr.StatusCode, status)
		}
		if apiErr.Message != "" {
			return fmt.Sprintf("Garmin API error (%s): %s", status, apiErr.Message)
		}
		return fmt.Sprintf("Garmin API error (%s)", status)
	}

	return err.Error()
}
