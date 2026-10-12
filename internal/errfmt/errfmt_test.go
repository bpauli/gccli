package errfmt

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

func TestAuthRequiredError(t *testing.T) {
	t.Run("with email", func(t *testing.T) {
		err := &AuthRequiredError{Email: "user@example.com"}
		if got := err.Error(); got != "no auth found for user@example.com" {
			t.Errorf("Error() = %q, want %q", got, "no auth found for user@example.com")
		}
	})

	t.Run("without email", func(t *testing.T) {
		err := &AuthRequiredError{}
		if got := err.Error(); got != "no auth found" {
			t.Errorf("Error() = %q, want %q", got, "no auth found")
		}
	})
}

func TestTokenExpiredError(t *testing.T) {
	t.Run("with email", func(t *testing.T) {
		err := &TokenExpiredError{Email: "user@example.com"}
		if got := err.Error(); got != "token expired for user@example.com" {
			t.Errorf("Error() = %q, want %q", got, "token expired for user@example.com")
		}
	})

	t.Run("without email", func(t *testing.T) {
		err := &TokenExpiredError{}
		if got := err.Error(); got != "token expired" {
			t.Errorf("Error() = %q, want %q", got, "token expired")
		}
	})
}

func TestRateLimitError(t *testing.T) {
	t.Run("with retry-after", func(t *testing.T) {
		err := &RateLimitError{RetryAfter: "30s"}
		if got := err.Error(); got != "rate limited; retry after 30s" {
			t.Errorf("Error() = %q, want %q", got, "rate limited; retry after 30s")
		}
	})

	t.Run("without retry-after", func(t *testing.T) {
		err := &RateLimitError{}
		if got := err.Error(); got != "rate limited" {
			t.Errorf("Error() = %q, want %q", got, "rate limited")
		}
	})
}

func TestGarminAPIError(t *testing.T) {
	t.Run("known status with message", func(t *testing.T) {
		err := &GarminAPIError{StatusCode: 403, Message: "access denied"}
		want := "garmin api: 403 Forbidden: access denied"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("known status without message", func(t *testing.T) {
		err := &GarminAPIError{StatusCode: 500}
		want := "garmin api: 500 Internal Server Error"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("unknown status code", func(t *testing.T) {
		err := &GarminAPIError{StatusCode: 999, Message: "unknown"}
		want := "garmin api: 999: unknown"
		if got := err.Error(); got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "token expired with email",
			err:  fmt.Errorf("get goals: %w", &garminapi.TokenExpiredError{Email: "user@example.com"}),
			want: "Token expired. Run: gccli auth login user@example.com",
		},
		{
			name: "token expired without email",
			err:  &garminapi.TokenExpiredError{},
			want: "Token expired. Run: gccli auth login <email>",
		},
		{
			name: "rate limited with retry-after",
			err:  fmt.Errorf("get goals: %w", &garminapi.RateLimitError{RetryAfter: "60"}),
			want: "Rate limited. Wait 60 and retry.",
		},
		{
			name: "rate limited without retry-after",
			err:  &garminapi.RateLimitError{},
			want: "Rate limited. Wait and retry.",
		},
		{
			name: "api error keeps the operation context",
			err:  fmt.Errorf("get goals: %w", &garminapi.GarminAPIError{StatusCode: 404, Message: "not found"}),
			want: "get goals: garmin api: 404 Not Found: not found",
		},
		{
			name: "unknown error",
			err:  errors.New("something went wrong"),
			want: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Format(tt.err); got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestErrorInterfaces(t *testing.T) {
	// Verify all error types implement the error interface.
	var _ error = (*AuthRequiredError)(nil)
	var _ error = (*TokenExpiredError)(nil)
	var _ error = (*RateLimitError)(nil)
	var _ error = (*GarminAPIError)(nil)
}
