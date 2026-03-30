//! Engine module for Riffer
//!
//! Contains the analysis, scoring, and transformation engines for music sequences.

pub mod analyzer;
pub mod clips_bridge;
pub mod scorer;
pub mod transformer;

pub use analyzer::{AnalysisResult, analyze_sequence};
#[allow(unused_imports)]
pub use clips_bridge::{ClipsResult, ClipsRuleEngine, ClipsSuggestion, ScoringAdjustment};
pub use scorer::{MusicScore, score_sequence, score_sequence_async};
#[allow(unused_imports)]
pub use scorer::{ScoreDimension, ScoreSummary};
pub use transformer::{augment, change_tempo, diminish, invert, key_change, retrograde, transpose};
