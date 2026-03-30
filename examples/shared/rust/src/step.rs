//! Step-through mode for learning and debugging.

use crate::InteractiveConfig;
use crate::tty::read_line;

/// Result of a step pause, indicating user's choice.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum StepAction {
    /// User pressed Enter, proceed to next step
    Continue,
    /// User typed 'q', exit program gracefully
    Quit,
    /// User typed 's', disable step mode and continue
    Skip,
}

impl InteractiveConfig {
    /// Pause for step-through mode with explanation.
    ///
    /// Displays a title and explanation bullet points, then waits for user input.
    /// Returns the action the user chose.
    ///
    /// # Arguments
    /// * `title` - Brief description of the step (e.g., "Creating Claude provider...")
    /// * `explanation` - Bullet points explaining what will happen
    ///
    /// # Returns
    /// * `StepAction::Continue` - User pressed Enter
    /// * `StepAction::Quit` - User typed 'q'
    /// * `StepAction::Skip` - User typed 's'
    pub fn step_pause(&mut self, title: &str, explanation: &[&str]) -> StepAction {
        if !self.is_step() {
            return StepAction::Continue;
        }

        // Print step header
        eprintln!("\n[nxusKit STEP] {}", title);

        // Print explanation bullets
        for line in explanation {
            eprintln!("  - {}", line);
        }

        // Print prompt
        eprintln!("[Press Enter to continue, 'q' to quit, 's' to skip steps]");

        // Read user input
        match read_line() {
            Some(input) => {
                if input == "q" || input == "quit" {
                    StepAction::Quit
                } else if input == "s" || input == "skip" {
                    self.skip_steps();
                    eprintln!("[nxusKit] Step mode disabled, running to completion...");
                    StepAction::Skip
                } else {
                    StepAction::Continue
                }
            }
            None => {
                // If we can't read input, continue automatically
                StepAction::Continue
            }
        }
    }
}
