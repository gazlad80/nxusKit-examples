package interactive

import (
	"fmt"
	"os"
)

// StepAction represents the result of a step pause.
type StepAction int

const (
	// ActionContinue means user pressed Enter, proceed to next step
	ActionContinue StepAction = iota
	// ActionQuit means user typed 'q', exit program gracefully
	ActionQuit
	// ActionSkip means user typed 's', disable step mode and continue
	ActionSkip
)

// StepPause pauses for step-through mode with explanation.
//
// Displays a title and explanation bullet points, then waits for user input.
// Returns the action the user chose.
//
// Parameters:
//   - title: Brief description of the step (e.g., "Creating Claude provider...")
//   - explanation: Bullet points explaining what will happen
//
// Returns:
//   - ActionContinue: User pressed Enter
//   - ActionQuit: User typed 'q'
//   - ActionSkip: User typed 's'
func (c *Config) StepPause(title string, explanation []string) StepAction {
	if !c.IsStep() {
		return ActionContinue
	}

	// Print step header
	fmt.Fprintf(os.Stderr, "\n[nxusKit STEP] %s\n", title)

	// Print explanation bullets
	for _, line := range explanation {
		fmt.Fprintf(os.Stderr, "  - %s\n", line)
	}

	// Print prompt
	fmt.Fprintln(os.Stderr, "[Press Enter to continue, 'q' to quit, 's' to skip steps]")

	// Read user input
	input := ReadLine()

	switch input {
	case "q", "quit":
		return ActionQuit
	case "s", "skip":
		c.SkipSteps()
		fmt.Fprintln(os.Stderr, "[nxusKit] Step mode disabled, running to completion...")
		return ActionSkip
	default:
		return ActionContinue
	}
}
