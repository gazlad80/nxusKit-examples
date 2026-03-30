//! File format support for Riffer
//!
//! This module provides reading and writing support for MIDI and MusicXML formats.

pub mod midi;
pub mod musicxml;

pub use midi::{read_midi, write_midi};
pub use musicxml::{read_musicxml, write_musicxml};

use crate::errors::{Result, RifferError};
use crate::types::Sequence;
use std::path::Path;

/// Auto-detect format and read a sequence from file
pub fn read_sequence(path: &Path) -> Result<Sequence> {
    let ext = path
        .extension()
        .and_then(|e| e.to_str())
        .map(|s| s.to_lowercase())
        .unwrap_or_default();

    match ext.as_str() {
        "mid" | "midi" => read_midi(path),
        "xml" | "musicxml" | "mxl" => read_musicxml(path),
        _ => Err(RifferError::UnsupportedFormat(ext)),
    }
}

/// Auto-detect format and write a sequence to file
pub fn write_sequence(sequence: &Sequence, path: &Path) -> Result<()> {
    let ext = path
        .extension()
        .and_then(|e| e.to_str())
        .map(|s| s.to_lowercase())
        .unwrap_or_default();

    match ext.as_str() {
        "mid" | "midi" => write_midi(sequence, path),
        "xml" | "musicxml" => write_musicxml(sequence, path),
        _ => Err(RifferError::UnsupportedFormat(ext)),
    }
}
