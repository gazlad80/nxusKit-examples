//! Interactive mode utilities for nxusKit examples.
//!
//! This crate provides two debugging modes for nxusKit examples:
//!
//! - **Verbose mode** (`--verbose` or `-v`): Shows raw HTTP request/response data
//! - **Step mode** (`--step` or `-s`): Pauses at each API call with explanations
//!
//! # Usage
//!
//! ```rust,ignore
//! use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
//!
//! fn main() {
//!     let mut config = InteractiveConfig::from_args();
//!
//!     // Step mode: pause with explanation
//!     if config.step_pause("Creating provider...", &[
//!         "This initializes the HTTP client",
//!         "API key is validated on first request",
//!     ]) == StepAction::Quit {
//!         return;
//!     }
//!
//!     // Verbose mode: show request (pass any serializable type)
//!     let request = serde_json::json!({"model": "test"});
//!     config.print_request("POST", "https://api.example.com/chat", &request);
//!
//!     // ... make request ...
//!
//!     // Verbose mode: show response
//!     let response = serde_json::json!({"result": "ok"});
//!     config.print_response(200, 100, &response);
//! }
//! ```
//!
//! # Environment Variables
//!
//! - `NXUSKIT_VERBOSE=1`: Enable verbose mode (alternative to `--verbose`)
//! - `NXUSKIT_STEP=1`: Enable step mode (alternative to `--step`)
//! - `NXUSKIT_VERBOSE_LIMIT=N`: Max characters before truncation (default: 2000)

mod config;
mod step;
mod tty;
mod verbose;

pub use config::InteractiveConfig;
pub use step::StepAction;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_config_default() {
        let config = InteractiveConfig::default();
        assert!(!config.is_verbose());
        assert!(!config.is_step());
        assert_eq!(config.get_verbose_limit(), 2000);
    }

    #[test]
    fn test_config_new() {
        let config = InteractiveConfig::new(true, false);
        assert!(config.is_verbose());
        // Step mode depends on TTY, may be false even if requested
    }

    #[test]
    fn test_step_action_equality() {
        assert_eq!(StepAction::Continue, StepAction::Continue);
        assert_eq!(StepAction::Quit, StepAction::Quit);
        assert_eq!(StepAction::Skip, StepAction::Skip);
        assert_ne!(StepAction::Continue, StepAction::Quit);
    }

    #[test]
    fn test_verbose_does_nothing_when_disabled() {
        let config = InteractiveConfig::new(false, false);
        // These should not panic or produce output when verbose is disabled
        config.print_request("GET", "http://test", &"test");
        config.print_response(200, 100, &"test");
        config.print_stream_chunk(1, "data");
        config.print_stream_done(100, 5);
    }
}
