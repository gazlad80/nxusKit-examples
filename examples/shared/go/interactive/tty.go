package interactive

import (
	"bufio"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// IsTTY checks if stdin is a terminal (TTY).
//
// Returns false if:
//   - Running in a pipe
//   - Running with redirected input
//   - Running in a CI environment without a terminal
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// ReadLine reads a single line from stdin, trimmed and lowercased.
// Returns empty string if reading fails.
func ReadLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ToLower(line))
}
