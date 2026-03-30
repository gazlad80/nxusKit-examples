//! MIDI file format support using midly
//!
//! Provides reading and writing of Standard MIDI Files (SMF).

use midly::{
    Format, Header, MetaMessage, MidiMessage, Smf, Timing, Track, TrackEvent, TrackEventKind,
};
use std::fs;
use std::path::Path;

use crate::errors::{Result, RifferError};
use crate::types::{Context, Note, Sequence, TimeSignature};

/// Read a MIDI file and convert to Sequence
pub fn read_midi(path: &Path) -> Result<Sequence> {
    let data = fs::read(path).map_err(|e| {
        if e.kind() == std::io::ErrorKind::NotFound {
            RifferError::FileNotFound(path.display().to_string())
        } else {
            RifferError::IoError(e)
        }
    })?;

    let smf = Smf::parse(&data).map_err(|e| RifferError::MidiParseError(e.to_string()))?;

    // Extract timing information
    let ticks_per_quarter = match smf.header.timing {
        Timing::Metrical(tpq) => tpq.as_int(),
        Timing::Timecode(_, _) => 480, // Default for SMPTE
    };

    // Check for multi-track and warn
    if smf.tracks.len() > 1 {
        eprintln!(
            "Warning: Multi-track MIDI file detected. Using first track only. {} tracks ignored.",
            smf.tracks.len() - 1
        );
    }

    // Get the first track with note data
    let track = find_note_track(&smf.tracks).ok_or_else(|| {
        RifferError::MidiParseError("No note data found in MIDI file".to_string())
    })?;

    // Parse notes and context from track
    let (notes, context) = parse_track(track, ticks_per_quarter)?;

    if notes.is_empty() {
        return Err(RifferError::EmptySequence);
    }

    let filename = path
        .file_stem()
        .and_then(|s| s.to_str())
        .unwrap_or("sequence")
        .to_string();

    let mut seq = Sequence::new(uuid::Uuid::new_v4().to_string(), notes);
    seq.context = context;
    seq.name = Some(filename);
    if let Some(source) = path.to_str() {
        let mut metadata = crate::types::Metadata::default();
        metadata.source_file = Some(source.to_string());
        seq.metadata = Some(metadata);
    }

    Ok(seq)
}

/// Find the first track that contains note events
fn find_note_track<'a>(tracks: &'a [Track<'a>]) -> Option<&'a Track<'a>> {
    for track in tracks {
        for event in track.iter() {
            if let TrackEventKind::Midi { message, .. } = event.kind {
                if matches!(message, MidiMessage::NoteOn { .. }) {
                    return Some(track);
                }
            }
        }
    }
    None
}

/// Parse a MIDI track into notes and context
fn parse_track(track: &Track, ticks_per_quarter: u16) -> Result<(Vec<Note>, Context)> {
    let mut notes = Vec::new();
    let mut active_notes: std::collections::HashMap<u8, (u64, u8)> =
        std::collections::HashMap::new();
    let mut current_tick: u64 = 0;
    let mut tempo: u16 = 120;
    let mut time_sig = TimeSignature::default();

    for event in track.iter() {
        current_tick += event.delta.as_int() as u64;

        match event.kind {
            TrackEventKind::Midi { message, .. } => match message {
                MidiMessage::NoteOn { key, vel } => {
                    if vel > 0 {
                        // Note on
                        active_notes.insert(key.as_int(), (current_tick, vel.as_int()));
                    } else {
                        // Note off (velocity 0)
                        if let Some((start_tick, velocity)) = active_notes.remove(&key.as_int()) {
                            let duration = (current_tick - start_tick) as u32;
                            notes.push(Note::new(key.as_int(), duration, velocity, start_tick));
                        }
                    }
                }
                MidiMessage::NoteOff { key, .. } => {
                    if let Some((start_tick, velocity)) = active_notes.remove(&key.as_int()) {
                        let duration = (current_tick - start_tick) as u32;
                        notes.push(Note::new(key.as_int(), duration, velocity, start_tick));
                    }
                }
                _ => {}
            },
            TrackEventKind::Meta(meta) => match meta {
                MetaMessage::Tempo(t) => {
                    // Convert microseconds per beat to BPM
                    let us_per_beat = t.as_int();
                    tempo = (60_000_000 / us_per_beat) as u16;
                }
                MetaMessage::TimeSignature(num, denom, _, _) => {
                    time_sig = TimeSignature {
                        numerator: num,
                        denominator: 1 << denom,
                    };
                }
                _ => {}
            },
            _ => {}
        }
    }

    // Sort notes by start time
    notes.sort_by_key(|n| n.start_tick);

    let context = Context {
        key_signature: None,
        time_signature: time_sig,
        tempo,
        ticks_per_quarter,
    };

    Ok((notes, context))
}

/// Write a Sequence to a MIDI file
pub fn write_midi(sequence: &Sequence, path: &Path) -> Result<()> {
    let tpq = sequence.context.ticks_per_quarter;

    let mut track_events: Vec<TrackEvent<'static>> = Vec::new();

    // Add tempo meta event
    let us_per_beat = 60_000_000 / sequence.context.tempo as u32;
    track_events.push(TrackEvent {
        delta: 0.into(),
        kind: TrackEventKind::Meta(MetaMessage::Tempo(us_per_beat.into())),
    });

    // Add time signature
    let ts = &sequence.context.time_signature;
    let denom_power = (ts.denominator as f32).log2() as u8;
    track_events.push(TrackEvent {
        delta: 0.into(),
        kind: TrackEventKind::Meta(MetaMessage::TimeSignature(ts.numerator, denom_power, 24, 8)),
    });

    // Collect all note events with absolute times
    let mut events: Vec<(u64, TrackEventKind<'static>)> = Vec::new();

    for note in &sequence.notes {
        // Note on
        events.push((
            note.start_tick,
            TrackEventKind::Midi {
                channel: 0.into(),
                message: MidiMessage::NoteOn {
                    key: note.pitch.into(),
                    vel: note.velocity.into(),
                },
            },
        ));

        // Note off
        events.push((
            note.end_tick(),
            TrackEventKind::Midi {
                channel: 0.into(),
                message: MidiMessage::NoteOff {
                    key: note.pitch.into(),
                    vel: 0.into(),
                },
            },
        ));
    }

    // Sort by time
    events.sort_by_key(|(t, _)| *t);

    // Convert to delta times
    let mut last_tick: u64 = 0;
    for (tick, kind) in events {
        let delta = tick - last_tick;
        track_events.push(TrackEvent {
            delta: (delta as u32).into(),
            kind,
        });
        last_tick = tick;
    }

    // End of track
    track_events.push(TrackEvent {
        delta: 0.into(),
        kind: TrackEventKind::Meta(MetaMessage::EndOfTrack),
    });

    // Build SMF
    let header = Header::new(Format::SingleTrack, Timing::Metrical(tpq.into()));
    let track: Track = track_events.into_iter().collect();
    let smf = Smf {
        header,
        tracks: vec![track],
    };

    // Write to file
    smf.save(path)
        .map_err(|e| RifferError::IoError(std::io::Error::new(std::io::ErrorKind::Other, e)))?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::NamedTempFile;

    #[test]
    fn test_round_trip() {
        let notes = vec![
            Note::new(60, 480, 80, 0),   // C4
            Note::new(64, 480, 85, 480), // E4
            Note::new(67, 480, 90, 960), // G4
        ];
        let sequence = Sequence::new("test", notes);

        let temp = NamedTempFile::new().unwrap();
        let path = temp.path();

        // Write
        write_midi(&sequence, path).unwrap();

        // Read back
        let loaded = read_midi(path).unwrap();

        assert_eq!(loaded.notes.len(), 3);
        assert_eq!(loaded.notes[0].pitch, 60);
        assert_eq!(loaded.notes[1].pitch, 64);
        assert_eq!(loaded.notes[2].pitch, 67);
    }
}
