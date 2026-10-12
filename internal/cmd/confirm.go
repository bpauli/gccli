package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// confirmReader is the reader used for confirmation prompts.
// Variable to allow overriding in tests.
var confirmReader io.Reader = os.Stdin

// confirm prompts the user for confirmation. Returns true if the user confirms.
// If force is true, skips the prompt and returns true.
// If the reader is a file that is not a terminal, it returns an error, because
// a script or agent cannot answer and must not see a silent cancel as success.
func confirm(w io.Writer, prompt string, force bool) (bool, error) {
	if force {
		return true, nil
	}

	if f, ok := confirmReader.(*os.File); ok && !term.IsTerminal(int(f.Fd())) {
		return false, errors.New("stdin is not a terminal: use --force to confirm")
	}

	_, _ = fmt.Fprintf(w, "%s [y/N]: ", prompt)

	scanner := bufio.NewScanner(confirmReader)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("read confirmation: %w", err)
		}
		return false, nil
	}

	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
