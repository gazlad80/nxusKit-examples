// Package interactive provides interactive debugging modes for nxusKit examples.
//
// It supports two modes:
//   - Verbose mode (--verbose or -v): Shows raw HTTP request/response data
//   - Step mode (--step or -s): Pauses at each API call with explanations
package interactive

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds interactive mode configuration.
type Config struct {
	// Verbose enables raw request/response output
	Verbose bool
	// Step enables step-through mode with pauses
	Step bool
	// VerboseLimit is the max characters before truncation
	VerboseLimit int
	// isTTY indicates if stdin is a terminal
	isTTY bool
	// stepSkipped indicates user pressed 's' to skip
	stepSkipped bool
}

// FromArgs creates Config from CLI flags and environment variables.
// CLI flags take precedence over environment variables.
//
// Scans os.Args directly instead of using flag.Parse() so apps that
// define their own flags or positional args are not rejected. Only
// --verbose/-v and --step/-s are recognised; everything else is
// left untouched for the app's own argument handling.
func FromArgs() *Config {
	verboseFound, stepFound := false, false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--verbose", "-v":
			verboseFound = true
		case "--step", "-s":
			stepFound = true
		}
	}

	// Check environment variables as fallback
	verboseEnabled := verboseFound || os.Getenv("NXUSKIT_VERBOSE") == "1"
	stepEnabled := stepFound || os.Getenv("NXUSKIT_STEP") == "1"

	// Parse verbose limit from environment
	verboseLimit := 2000
	if limitStr := os.Getenv("NXUSKIT_VERBOSE_LIMIT"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil {
			verboseLimit = clamp(parsed, 100, 100000)
		}
	}

	isTTY := IsTTY()

	// Warn if step mode requested in non-TTY environment
	if stepEnabled && !isTTY {
		fmt.Fprintln(os.Stderr, "[nxusKit] Warning: Step mode disabled (not a TTY). Use --verbose for debugging.")
	}

	return &Config{
		Verbose:      verboseEnabled,
		Step:         stepEnabled && isTTY, // Auto-disable step mode in non-TTY
		VerboseLimit: verboseLimit,
		isTTY:        isTTY,
		stepSkipped:  false,
	}
}

// New creates a Config with specified values (for testing or programmatic use).
func New(verbose, step bool) *Config {
	isTTY := IsTTY()
	return &Config{
		Verbose:      verbose,
		Step:         step && isTTY,
		VerboseLimit: 2000,
		isTTY:        isTTY,
		stepSkipped:  false,
	}
}

// IsVerbose returns true if verbose mode is enabled.
func (c *Config) IsVerbose() bool {
	return c.Verbose
}

// IsStep returns true if step mode is enabled and not skipped.
func (c *Config) IsStep() bool {
	return c.Step && !c.stepSkipped
}

// IsTTYMode returns true if running in a TTY.
func (c *Config) IsTTYMode() bool {
	return c.isTTY
}

// GetVerboseLimit returns the verbose output truncation limit.
func (c *Config) GetVerboseLimit() int {
	return c.VerboseLimit
}

// SkipSteps marks step mode as skipped (user pressed 's').
func (c *Config) SkipSteps() {
	c.stepSkipped = true
}

// clamp restricts a value to the given range.
func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
