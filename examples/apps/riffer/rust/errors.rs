//! Error types for Riffer
//!
//! Defines error types with corresponding exit codes for CLI usage.

use std::fmt;
use std::io;

/// Exit codes for CLI operations
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExitCode {
    Success = 0,
    InvalidArguments = 1,
    FileNotFound = 2,
    ParseError = 3,
    TransformationError = 4,
    ClipsError = 5,
    LlmError = 6,
}

/// Main error type for Riffer operations
#[derive(Debug)]
pub enum RifferError {
    /// Invalid command-line arguments
    InvalidArguments(String),

    /// File not found
    FileNotFound(String),

    /// MIDI parse error
    MidiParseError(String),

    /// MusicXML parse error
    MusicXmlParseError(String),

    /// Transformation error (e.g., notes out of range)
    TransformationError(String),

    /// Invalid transformation parameters
    InvalidTransformation(String),

    /// I/O error
    IoError(io::Error),

    /// JSON serialization/deserialization error
    JsonError(serde_json::Error),

    /// CLIPS rule engine error
    ClipsError(String),

    /// CLIPS rules failed to load
    ClipsLoadError(String),

    /// LLM provider not available
    LlmUnavailable(String),

    /// LLM response parse error
    LlmParseError(String),

    /// Sequence validation error
    ValidationError(String),

    /// Empty sequence
    EmptySequence,

    /// Unsupported file format
    UnsupportedFormat(String),
}

impl RifferError {
    /// Get the exit code for this error
    pub fn exit_code(&self) -> ExitCode {
        match self {
            RifferError::InvalidArguments(_) => ExitCode::InvalidArguments,
            RifferError::FileNotFound(_) => ExitCode::FileNotFound,
            RifferError::MidiParseError(_) => ExitCode::ParseError,
            RifferError::MusicXmlParseError(_) => ExitCode::ParseError,
            RifferError::TransformationError(_) => ExitCode::TransformationError,
            RifferError::InvalidTransformation(_) => ExitCode::TransformationError,
            RifferError::IoError(_) => ExitCode::FileNotFound,
            RifferError::JsonError(_) => ExitCode::ParseError,
            RifferError::ClipsError(_) => ExitCode::ClipsError,
            RifferError::ClipsLoadError(_) => ExitCode::ClipsError,
            RifferError::LlmUnavailable(_) => ExitCode::LlmError,
            RifferError::LlmParseError(_) => ExitCode::LlmError,
            RifferError::ValidationError(_) => ExitCode::InvalidArguments,
            RifferError::EmptySequence => ExitCode::InvalidArguments,
            RifferError::UnsupportedFormat(_) => ExitCode::InvalidArguments,
        }
    }

    /// Get a user-friendly error message
    pub fn user_message(&self) -> String {
        match self {
            RifferError::InvalidArguments(msg) => format!("Error: Invalid arguments - {}", msg),
            RifferError::FileNotFound(path) => format!("Error: File not found: {}", path),
            RifferError::MidiParseError(msg) => format!("Error: Invalid MIDI file - {}", msg),
            RifferError::MusicXmlParseError(msg) => {
                format!("Error: Invalid MusicXML file - {}", msg)
            }
            RifferError::TransformationError(msg) => {
                format!("Error: Transformation failed - {}", msg)
            }
            RifferError::InvalidTransformation(msg) => {
                format!("Error: Invalid transformation - {}", msg)
            }
            RifferError::IoError(e) => format!("Error: I/O error - {}", e),
            RifferError::JsonError(e) => format!("Error: JSON error - {}", e),
            RifferError::ClipsError(msg) => format!("Error: CLIPS rule engine error - {}", msg),
            RifferError::ClipsLoadError(msg) => format!(
                "Warning: Could not load CLIPS rules - {}. Using deterministic scoring.",
                msg
            ),
            RifferError::LlmUnavailable(msg) => format!(
                "Error: LLM provider not configured - {}.\nSet ANTHROPIC_API_KEY, OPENAI_API_KEY, or configure Ollama.\nUse explicit flags instead: --transpose, --tempo, --key, etc.",
                msg
            ),
            RifferError::LlmParseError(msg) => {
                format!("Error: Could not parse LLM response - {}", msg)
            }
            RifferError::ValidationError(msg) => format!("Error: Validation failed - {}", msg),
            RifferError::EmptySequence => "Error: Sequence contains no notes".to_string(),
            RifferError::UnsupportedFormat(fmt) => {
                format!(
                    "Error: Unsupported file format: {} (expected .mid or .musicxml)",
                    fmt
                )
            }
        }
    }
}

impl fmt::Display for RifferError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.user_message())
    }
}

impl std::error::Error for RifferError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            RifferError::IoError(e) => Some(e),
            RifferError::JsonError(e) => Some(e),
            _ => None,
        }
    }
}

impl From<io::Error> for RifferError {
    fn from(err: io::Error) -> Self {
        if err.kind() == io::ErrorKind::NotFound {
            RifferError::FileNotFound(err.to_string())
        } else {
            RifferError::IoError(err)
        }
    }
}

impl From<serde_json::Error> for RifferError {
    fn from(err: serde_json::Error) -> Self {
        RifferError::JsonError(err)
    }
}

/// Result type for Riffer operations
pub type Result<T> = std::result::Result<T, RifferError>;

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_exit_codes() {
        assert_eq!(
            RifferError::InvalidArguments("test".into()).exit_code(),
            ExitCode::InvalidArguments
        );
        assert_eq!(
            RifferError::FileNotFound("test.mid".into()).exit_code(),
            ExitCode::FileNotFound
        );
        assert_eq!(
            RifferError::MidiParseError("corrupt".into()).exit_code(),
            ExitCode::ParseError
        );
    }

    #[test]
    fn test_user_message() {
        let err = RifferError::FileNotFound("/path/to/file.mid".into());
        assert!(err.user_message().contains("/path/to/file.mid"));
    }
}
