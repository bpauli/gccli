package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// confirmReader is the reader used for confirmation prompts.
// Variable to allow overriding in tests.
var confirmReader io.Reader = os.Stdin

// confirm prompts the user for confirmation. Returns true if the user confirms.
// If force is true, skips the prompt and returns true.
func confirm(w io.Writer, prompt string, force bool) (bool, error) {
	if force {
		return true, nil
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
