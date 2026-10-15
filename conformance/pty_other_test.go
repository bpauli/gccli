//go:build !linux

package conformance

import (
	"errors"
	"os"
)

func openTerminal() (ptmx, pts *os.File, err error) {
	return nil, nil, errors.New("no pseudo-terminal outside linux")
}
