//! Core data types for Riffer music analysis
//!
//! This module defines the fundamental types for representing musical sequences,
//! notes, and analysis results.

use serde::{Deserialize, Serialize};
use std::fmt;

/// MIDI note number (0-127)
pub type MidiNote = u8;

/// Duration in MIDI ticks
pub type Ticks = u32;

/// Velocity (0-127)
pub type Velocity = u8;

/// Pitch class representing the 12 chromatic notes
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
pub enum PitchClass {
    C,
    Cs, // C#/Db
    D,
    Ds, // D#/Eb
    E,
    F,
    Fs, // F#/Gb
    G,
    Gs, // G#/Ab
    A,
    As, // A#/Bb
    B,
}

impl PitchClass {
    /// All pitch classes in chromatic order
    pub const ALL: [PitchClass; 12] = [
        PitchClass::C,
        PitchClass::Cs,
        PitchClass::D,
        PitchClass::Ds,
        PitchClass::E,
        PitchClass::F,
        PitchClass::Fs,
        PitchClass::G,
        PitchClass::Gs,
        PitchClass::A,
        PitchClass::As,
        PitchClass::B,
    ];

    /// Convert MIDI note number to pitch class
    pub fn from_midi(note: MidiNote) -> Self {
        Self::ALL[(note % 12) as usize]
    }

    /// Convert pitch class to semitone offset from C
    pub fn to_semitone(&self) -> u8 {
        match self {
            PitchClass::C => 0,
            PitchClass::Cs => 1,
            PitchClass::D => 2,
            PitchClass::Ds => 3,
            PitchClass::E => 4,
            PitchClass::F => 5,
            PitchClass::Fs => 6,
            PitchClass::G => 7,
            PitchClass::Gs => 8,
            PitchClass::A => 9,
            PitchClass::As => 10,
            PitchClass::B => 11,
        }
    }

    /// Parse pitch class from string (e.g., "C", "Cs", "Db")
    pub fn from_str(s: &str) -> Option<Self> {
        match s.to_lowercase().as_str() {
            "c" => Some(PitchClass::C),
            "cs" | "c#" | "db" => Some(PitchClass::Cs),
            "d" => Some(PitchClass::D),
            "ds" | "d#" | "eb" => Some(PitchClass::Ds),
            "e" => Some(PitchClass::E),
            "f" => Some(PitchClass::F),
            "fs" | "f#" | "gb" => Some(PitchClass::Fs),
            "g" => Some(PitchClass::G),
            "gs" | "g#" | "ab" => Some(PitchClass::Gs),
            "a" => Some(PitchClass::A),
            "as" | "a#" | "bb" => Some(PitchClass::As),
            "b" => Some(PitchClass::B),
            _ => None,
        }
    }
}

impl fmt::Display for PitchClass {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            PitchClass::C => write!(f, "C"),
            PitchClass::Cs => write!(f, "C#"),
            PitchClass::D => write!(f, "D"),
            PitchClass::Ds => write!(f, "D#"),
            PitchClass::E => write!(f, "E"),
            PitchClass::F => write!(f, "F"),
            PitchClass::Fs => write!(f, "F#"),
            PitchClass::G => write!(f, "G"),
            PitchClass::Gs => write!(f, "G#"),
            PitchClass::A => write!(f, "A"),
            PitchClass::As => write!(f, "A#"),
            PitchClass::B => write!(f, "B"),
        }
    }
}

/// Musical mode (scale type)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Mode {
    Major,
    Minor, // Natural minor (Aeolian)
    Dorian,
    Phrygian,
    Lydian,
    Mixolydian,
    Aeolian,
    Locrian,
}

impl fmt::Display for Mode {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Mode::Major => write!(f, "Major"),
            Mode::Minor => write!(f, "Minor"),
            Mode::Dorian => write!(f, "Dorian"),
            Mode::Phrygian => write!(f, "Phrygian"),
            Mode::Lydian => write!(f, "Lydian"),
            Mode::Mixolydian => write!(f, "Mixolydian"),
            Mode::Aeolian => write!(f, "Aeolian"),
            Mode::Locrian => write!(f, "Locrian"),
        }
    }
}

/// Key signature (root + mode)
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct KeySignature {
    pub root: PitchClass,
    pub mode: Mode,
}

impl KeySignature {
    pub fn new(root: PitchClass, mode: Mode) -> Self {
        Self { root, mode }
    }
}

impl fmt::Display for KeySignature {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{} {}", self.root, self.mode)
    }
}

/// Time signature
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct TimeSignature {
    pub numerator: u8,
    pub denominator: u8,
}

impl Default for TimeSignature {
    fn default() -> Self {
        Self {
            numerator: 4,
            denominator: 4,
        }
    }
}

impl fmt::Display for TimeSignature {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}/{}", self.numerator, self.denominator)
    }
}

/// A single musical note
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Note {
    /// MIDI pitch (0-127)
    pub pitch: MidiNote,
    /// Duration in ticks
    pub duration: Ticks,
    /// Attack velocity (0-127)
    pub velocity: Velocity,
    /// Start position in ticks
    pub start_tick: u64,
}

impl Note {
    pub fn new(pitch: MidiNote, duration: Ticks, velocity: Velocity, start_tick: u64) -> Self {
        Self {
            pitch,
            duration,
            velocity,
            start_tick,
        }
    }

    /// Get the pitch class of this note
    pub fn pitch_class(&self) -> PitchClass {
        PitchClass::from_midi(self.pitch)
    }

    /// Get the octave of this note (-1 to 9)
    pub fn octave(&self) -> i8 {
        (self.pitch as i8 / 12) - 1
    }

    /// Get end tick (start + duration)
    pub fn end_tick(&self) -> u64 {
        self.start_tick + self.duration as u64
    }
}

/// Musical context for a sequence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Context {
    /// Detected or specified key signature
    pub key_signature: Option<KeySignature>,
    /// Time signature
    pub time_signature: TimeSignature,
    /// Tempo in BPM
    pub tempo: u16,
    /// MIDI ticks per quarter note
    pub ticks_per_quarter: u16,
}

impl Default for Context {
    fn default() -> Self {
        Self {
            key_signature: None,
            time_signature: TimeSignature::default(),
            tempo: 120,
            ticks_per_quarter: 480,
        }
    }
}

/// Metadata about a sequence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Metadata {
    pub created: Option<String>,
    pub author: Option<String>,
    pub source_file: Option<String>,
    pub version: String,
}

impl Default for Metadata {
    fn default() -> Self {
        Self {
            created: None,
            author: None,
            source_file: None,
            version: "1.0".to_string(),
        }
    }
}

/// A musical sequence (collection of notes with context)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Sequence {
    /// Unique identifier
    pub id: String,
    /// Human-readable name
    pub name: Option<String>,
    /// Ordered list of notes
    pub notes: Vec<Note>,
    /// Musical context
    pub context: Context,
    /// Optional metadata
    pub metadata: Option<Metadata>,
}

impl Sequence {
    pub fn new(id: impl Into<String>, notes: Vec<Note>) -> Self {
        Self {
            id: id.into(),
            name: None,
            notes,
            context: Context::default(),
            metadata: None,
        }
    }

    pub fn with_context(mut self, context: Context) -> Self {
        self.context = context;
        self
    }

    pub fn with_name(mut self, name: impl Into<String>) -> Self {
        self.name = Some(name.into());
        self
    }

    /// Check if sequence is empty
    pub fn is_empty(&self) -> bool {
        self.notes.is_empty()
    }

    /// Get number of notes
    pub fn len(&self) -> usize {
        self.notes.len()
    }

    /// Get total duration in ticks
    pub fn total_duration(&self) -> u64 {
        self.notes.iter().map(|n| n.end_tick()).max().unwrap_or(0)
    }
}

/// Interval quality classification
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum IntervalQuality {
    PerfectConsonance,
    ImperfectConsonance,
    MildDissonance,
    StrongDissonance,
}

impl fmt::Display for IntervalQuality {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            IntervalQuality::PerfectConsonance => write!(f, "Perfect Consonance"),
            IntervalQuality::ImperfectConsonance => write!(f, "Imperfect Consonance"),
            IntervalQuality::MildDissonance => write!(f, "Mild Dissonance"),
            IntervalQuality::StrongDissonance => write!(f, "Strong Dissonance"),
        }
    }
}

/// Direction of an interval
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Direction {
    Ascending,
    Descending,
    Unison,
}

/// Melodic contour type
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum ContourType {
    Ascending,
    Descending,
    Arch,
    InverseArch,
    Wave,
    Static,
}

impl fmt::Display for ContourType {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ContourType::Ascending => write!(f, "Ascending"),
            ContourType::Descending => write!(f, "Descending"),
            ContourType::Arch => write!(f, "Arch"),
            ContourType::InverseArch => write!(f, "Inverse Arch"),
            ContourType::Wave => write!(f, "Wave"),
            ContourType::Static => write!(f, "Static"),
        }
    }
}

/// Scale type enumeration
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum ScaleType {
    Major,
    NaturalMinor,
    HarmonicMinor,
    MelodicMinor,
    Pentatonic,
    Blues,
    Dorian,
    Phrygian,
    Lydian,
    Mixolydian,
    Aeolian,
    Locrian,
}

/// Suggestion category
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum SuggestionCategory {
    Harmony,
    Melody,
    Rhythm,
    Dynamics,
    Resolution,
    Structure,
}

/// Suggestion severity
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum Severity {
    Info,
    Suggestion,
    Warning,
}

/// Output format options
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OutputFormat {
    Json,
    Markdown,
    Midi,
    MusicXml,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pitch_class_from_midi() {
        assert_eq!(PitchClass::from_midi(60), PitchClass::C); // Middle C
        assert_eq!(PitchClass::from_midi(64), PitchClass::E);
        assert_eq!(PitchClass::from_midi(67), PitchClass::G);
        assert_eq!(PitchClass::from_midi(72), PitchClass::C); // C5
    }

    #[test]
    fn test_note_octave() {
        let note = Note::new(60, 480, 80, 0);
        assert_eq!(note.octave(), 4); // Middle C is C4

        let note_low = Note::new(36, 480, 80, 0);
        assert_eq!(note_low.octave(), 2); // C2
    }

    #[test]
    fn test_key_signature_display() {
        let key = KeySignature::new(PitchClass::E, Mode::Minor);
        assert_eq!(format!("{}", key), "E Minor");
    }
}
