//! Riffer - Music Sequence Analysis and Transformation
//!
//! This library provides functionality to analyze and transform music sequences:
//!
//! - Read/write MIDI and MusicXML formats
//! - Detect keys using Krumhansl-Schmuckler algorithm
//! - Classify intervals and analyze harmony
//! - Score musicality using rule-based CLIPS engine
//! - Transform sequences (transpose, tempo changes, etc.)
//!
//! # Example
//!
//! ```no_run
//! use riffer::{formats, theory, KeySignature, Mode, PitchClass};
//!
//! // Read a MIDI file
//! // let sequence = formats::read_sequence(Path::new("melody.mid"))?;
//!
//! // Detect the key
//! // let key_detection = theory::detect_key(&sequence.notes);
//! // println!("Detected key: {}", key_detection.key);
//! ```

// Clippy allowances for example code - prioritize readability and pedagogy
#![allow(clippy::collapsible_if)]
#![allow(clippy::clone_on_copy)]
#![allow(clippy::should_implement_trait)]
#![allow(clippy::field_reassign_with_default)]
#![allow(clippy::io_other_error)]
#![allow(clippy::manual_clamp)]
#![allow(clippy::unnecessary_cast)]
#![allow(clippy::manual_range_contains)]

pub mod engine;
pub mod errors;
pub mod formats;
pub mod theory;
pub mod types;

// Re-export main types
pub use engine::{
    AnalysisResult, ClipsResult, ClipsRuleEngine, ClipsSuggestion, MusicScore, ScoreDimension,
    ScoreSummary, ScoringAdjustment, analyze_sequence, score_sequence,
};
pub use errors::{ExitCode, Result, RifferError};
pub use theory::{KeyDetection, detect_key};
pub use types::{
    Context, ContourType, Direction, IntervalQuality, KeySignature, Metadata, MidiNote, Mode, Note,
    OutputFormat, PitchClass, ScaleType, Sequence, Severity, SuggestionCategory, Ticks,
    TimeSignature, Velocity,
};

/// Riffer version
pub const VERSION: &str = "0.1.0";
