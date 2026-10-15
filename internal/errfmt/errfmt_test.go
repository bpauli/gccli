package errfmt

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bpauli/gccli/internal/garminapi"
)

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
			name: "token expired after a failed refresh",
			err:  fmt.Errorf("get goals: %w", &garminapi.TokenExpiredError{Email: "user@example.com", Err: errors.New("HTTP 400: invalid_grant")}),
			want: "Token expired: HTTP 400: invalid_grant. Run: gccli auth login user@example.com",
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
