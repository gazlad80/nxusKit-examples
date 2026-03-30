//! TTY detection and terminal utilities.

use crossterm::tty::IsTty;
use std::io::stdin;

/// Check if stdin is a terminal (TTY).
///
/// Returns false if:
/// - Running in a pipe
/// - Running with redirected input
/// - Running in a CI environment without a terminal
pub fn is_tty() -> bool {
    stdin().is_tty()
}

/// Read a single line from stdin, trimmed.
///
/// Returns None if reading fails or stdin is not available.
pub fn read_line() -> Option<String> {
    let mut input = String::new();
    match std::io::stdin().read_line(&mut input) {
        Ok(_) => Some(input.trim().to_lowercase()),
        Err(_) => None,
    }
}
