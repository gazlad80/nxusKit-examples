// Package riffer provides music sequence analysis and transformation.
package riffer

import (
	"fmt"
)

// ExitCode represents CLI exit codes
type ExitCode int

const (
	ExitSuccess             ExitCode = 0
	ExitInvalidArguments    ExitCode = 1
	ExitFileNotFound        ExitCode = 2
	ExitParseError          ExitCode = 3
	ExitTransformationError ExitCode = 4
	ExitClipsError          ExitCode = 5
	ExitLLMError            ExitCode = 6
)

// RifferError represents an error in Riffer operations
type RifferError struct {
	Kind    ErrorKind
	Message string
	Cause   error
}

// ErrorKind represents the type of error
type ErrorKind int

const (
	ErrInvalidArguments ErrorKind = iota
	ErrFileNotFound
	ErrMidiParse
	ErrMusicXMLParse
	ErrTransformation
	ErrInvalidTransformation
	ErrIO
	ErrJSON
	ErrClips
	ErrClipsLoad
	ErrLLMUnavailable
	ErrLLMParse
	ErrValidation
	ErrEmptySequence
	ErrUnsupportedFormat
)

// NewRifferError creates a new RifferError
func NewRifferError(kind ErrorKind, message string) *RifferError {
	return &RifferError{Kind: kind, Message: message}
}

// NewRifferErrorWithCause creates a new RifferError with a cause
func NewRifferErrorWithCause(kind ErrorKind, message string, cause error) *RifferError {
	return &RifferError{Kind: kind, Message: message, Cause: cause}
}

// Error returns the error message
func (e *RifferError) Error() string {
	return e.UserMessage()
}

// Unwrap returns the underlying cause
func (e *RifferError) Unwrap() error {
	return e.Cause
}

// ExitCode returns the appropriate exit code for this error
func (e *RifferError) ExitCode() ExitCode {
	switch e.Kind {
	case ErrInvalidArguments:
		return ExitInvalidArguments
	case ErrFileNotFound:
		return ExitFileNotFound
	case ErrMidiParse, ErrMusicXMLParse, ErrJSON:
		return ExitParseError
	case ErrTransformation, ErrInvalidTransformation:
		return ExitTransformationError
	case ErrClips, ErrClipsLoad:
		return ExitClipsError
	case ErrLLMUnavailable, ErrLLMParse:
		return ExitLLMError
	case ErrIO:
		return ExitFileNotFound
	case ErrValidation, ErrEmptySequence, ErrUnsupportedFormat:
		return ExitInvalidArguments
	default:
		return ExitInvalidArguments
	}
}

// UserMessage returns a user-friendly error message
func (e *RifferError) UserMessage() string {
	switch e.Kind {
	case ErrInvalidArguments:
		return fmt.Sprintf("Error: Invalid arguments - %s", e.Message)
	case ErrFileNotFound:
		return fmt.Sprintf("Error: File not found: %s", e.Message)
	case ErrMidiParse:
		return fmt.Sprintf("Error: Invalid MIDI file - %s", e.Message)
	case ErrMusicXMLParse:
		return fmt.Sprintf("Error: Invalid MusicXML file - %s", e.Message)
	case ErrTransformation:
		return fmt.Sprintf("Error: Transformation failed - %s", e.Message)
	case ErrInvalidTransformation:
		return fmt.Sprintf("Error: Invalid transformation - %s", e.Message)
	case ErrIO:
		return fmt.Sprintf("Error: I/O error - %s", e.Message)
	case ErrJSON:
		return fmt.Sprintf("Error: JSON error - %s", e.Message)
	case ErrClips:
		return fmt.Sprintf("Error: CLIPS rule engine error - %s", e.Message)
	case ErrClipsLoad:
		return fmt.Sprintf("Warning: Could not load CLIPS rules - %s. Using deterministic scoring.", e.Message)
	case ErrLLMUnavailable:
		return fmt.Sprintf("Error: LLM provider not configured - %s.\nSet ANTHROPIC_API_KEY, OPENAI_API_KEY, or configure Ollama.\nUse explicit flags instead: --transpose, --tempo, --key, etc.", e.Message)
	case ErrLLMParse:
		return fmt.Sprintf("Error: Could not parse LLM response - %s", e.Message)
	case ErrValidation:
		return fmt.Sprintf("Error: Validation failed - %s", e.Message)
	case ErrEmptySequence:
		return "Error: Sequence contains no notes"
	case ErrUnsupportedFormat:
		return fmt.Sprintf("Error: Unsupported file format: %s (expected .mid or .musicxml)", e.Message)
	default:
		return fmt.Sprintf("Error: %s", e.Message)
	}
}

// Common error constructors

// ErrFileNotFoundError creates a file not found error
func ErrFileNotFoundError(path string) *RifferError {
	return NewRifferError(ErrFileNotFound, path)
}

// ErrMidiParseError creates a MIDI parse error
func ErrMidiParseError(msg string) *RifferError {
	return NewRifferError(ErrMidiParse, msg)
}

// ErrMusicXMLParseError creates a MusicXML parse error
func ErrMusicXMLParseError(msg string) *RifferError {
	return NewRifferError(ErrMusicXMLParse, msg)
}

// ErrTransformationError creates a transformation error
func ErrTransformationError(msg string) *RifferError {
	return NewRifferError(ErrTransformation, msg)
}

// ErrEmptySequenceError creates an empty sequence error
func ErrEmptySequenceError() *RifferError {
	return NewRifferError(ErrEmptySequence, "")
}

// ErrUnsupportedFormatError creates an unsupported format error
func ErrUnsupportedFormatError(format string) *RifferError {
	return NewRifferError(ErrUnsupportedFormat, format)
}

// ErrClipsLoadError creates a CLIPS load error
func ErrClipsLoadError(msg string) *RifferError {
	return NewRifferError(ErrClipsLoad, msg)
}

// ErrLLMUnavailableError creates an LLM unavailable error
func ErrLLMUnavailableError(msg string) *RifferError {
	return NewRifferError(ErrLLMUnavailable, msg)
}
