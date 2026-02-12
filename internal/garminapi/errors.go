package garminapi

import (
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

// TokenExpiredError indicates the stored token has expired and could not be refreshed.
type TokenExpiredError struct {
	Email string
}

func (e *TokenExpiredError) Error() string {
	if e.Email != "" {
		return fmt.Sprintf("token expired for %s", e.Email)
	}
	return "token expired"
}

// RateLimitError indicates the API returned a 429 Too Many Requests response.
type RateLimitError struct {
	RetryAfter string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("rate limited; retry after %s", e.RetryAfter)
	}
	return "rate limited"
}

// GarminAPIError represents a non-2xx error response from the Garmin Connect API.
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

// InvalidFileFormatError indicates an unsupported file format was requested.
type InvalidFileFormatError struct {
	Format string
}

func (e *InvalidFileFormatError) Error() string {
	return fmt.Sprintf("invalid file format: %s", e.Format)
}
