//! Configuration for interactive modes in nxusKit examples.

use std::env;

use crate::tty::is_tty;

/// Configuration for interactive debugging modes.
///
/// Created once at program start from CLI args and environment variables.
#[derive(Debug, Clone)]
pub struct InteractiveConfig {
    /// Enable verbose output showing raw request/response
    pub verbose: bool,
    /// Enable step-through mode with pauses
    pub step: bool,
    /// Maximum characters before truncation in verbose output
    pub verbose_limit: usize,
    /// Whether stdin is a terminal (for step mode)
    is_tty: bool,
    /// Whether step mode has been skipped by the user
    step_skipped: bool,
}

impl InteractiveConfig {
    /// Parse configuration from CLI args and environment variables.
    ///
    /// Scans `std::env::args()` directly instead of using clap so apps
    /// with their own arg parsing are not rejected. Only `--verbose`/`-v`
    /// and `--step`/`-s` are recognised; everything else is left
    /// untouched for the app's own argument handling.
    ///
    /// CLI flags take precedence over environment variables.
    pub fn from_args() -> Self {
        let mut verbose_found = false;
        let mut step_found = false;
        for arg in env::args().skip(1) {
            match arg.as_str() {
                "--verbose" | "-v" => verbose_found = true,
                "--step" | "-s" => step_found = true,
                _ => {}
            }
        }

        // Check environment variables as fallback
        let verbose = verbose_found || env::var("NXUSKIT_VERBOSE").is_ok_and(|v| v == "1");
        let step = step_found || env::var("NXUSKIT_STEP").is_ok_and(|v| v == "1");

        // Parse verbose limit from environment
        let verbose_limit = env::var("NXUSKIT_VERBOSE_LIMIT")
            .ok()
            .and_then(|v| v.parse().ok())
            .unwrap_or(2000)
            .clamp(100, 100000);

        let is_tty = is_tty();

        // Warn if step mode requested in non-TTY environment
        if step && !is_tty {
            eprintln!(
                "[nxusKit] Warning: Step mode disabled (not a TTY). Use --verbose for debugging."
            );
        }

        Self {
            verbose,
            step: step && is_tty, // Auto-disable step mode in non-TTY
            verbose_limit,
            is_tty,
            step_skipped: false,
        }
    }

    /// Create a new config with specified values (for testing or programmatic use).
    pub fn new(verbose: bool, step: bool) -> Self {
        let is_tty = is_tty();
        Self {
            verbose,
            step: step && is_tty,
            verbose_limit: 2000,
            is_tty,
            step_skipped: false,
        }
    }

    /// Check if verbose mode is enabled.
    pub fn is_verbose(&self) -> bool {
        self.verbose
    }

    /// Check if step mode is enabled and not skipped.
    pub fn is_step(&self) -> bool {
        self.step && !self.step_skipped
    }

    /// Check if running in a TTY.
    pub fn is_tty(&self) -> bool {
        self.is_tty
    }

    /// Get the verbose output truncation limit.
    pub fn get_verbose_limit(&self) -> usize {
        self.verbose_limit
    }

    /// Mark step mode as skipped (user pressed 's').
    pub fn skip_steps(&mut self) {
        self.step_skipped = true;
    }
}

impl Default for InteractiveConfig {
    fn default() -> Self {
        Self {
            verbose: false,
            step: false,
            verbose_limit: 2000,
            is_tty: is_tty(),
            step_skipped: false,
        }
    }
}
