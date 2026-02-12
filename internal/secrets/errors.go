package secrets

import "errors"

// ErrNotFound is returned when no token data exists for the given email.
var ErrNotFound = errors.New("token not found")
