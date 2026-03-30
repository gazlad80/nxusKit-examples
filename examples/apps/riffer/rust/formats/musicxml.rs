//! MusicXML file format support using quick-xml
//!
//! Provides reading and writing of MusicXML partwise format.

use quick_xml::de::from_reader;
use quick_xml::se::to_string;
use serde::{Deserialize, Serialize};
use std::fs::File;
use std::io::BufReader;
use std::path::Path;

use crate::errors::{Result, RifferError};
use crate::types::{Context, KeySignature, Mode, Note, PitchClass, Sequence, TimeSignature};

/// MusicXML score-partwise root element
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename = "score-partwise")]
struct ScorePartwise {
    #[serde(rename = "part-list", default)]
    part_list: PartList,
    #[serde(rename = "part", default)]
    parts: Vec<Part>,
}

#[derive(Debug, Default, Serialize, Deserialize)]
struct PartList {
    #[serde(rename = "score-part", default)]
    score_parts: Vec<ScorePart>,
}

#[derive(Debug, Serialize, Deserialize)]
struct ScorePart {
    #[serde(rename = "@id")]
    id: String,
    #[serde(rename = "part-name", default, skip_serializing_if = "Option::is_none")]
    part_name: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Part {
    #[serde(rename = "@id")]
    id: String,
    #[serde(rename = "measure", default)]
    measures: Vec<Measure>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Measure {
    #[serde(rename = "@number")]
    number: String,
    #[serde(rename = "attributes", default)]
    attributes: Option<Attributes>,
    #[serde(rename = "direction", default)]
    directions: Vec<Direction>,
    #[serde(rename = "note", default)]
    notes: Vec<MusicXmlNote>,
}

#[derive(Debug, Default, Serialize, Deserialize)]
struct Attributes {
    #[serde(rename = "divisions", default, skip_serializing_if = "Option::is_none")]
    divisions: Option<i32>,
    #[serde(rename = "key", default, skip_serializing_if = "Option::is_none")]
    key: Option<Key>,
    #[serde(rename = "time", default, skip_serializing_if = "Option::is_none")]
    time: Option<Time>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Key {
    #[serde(rename = "fifths")]
    fifths: i32,
    #[serde(rename = "mode", default, skip_serializing_if = "Option::is_none")]
    mode: Option<String>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Time {
    #[serde(rename = "beats")]
    beats: String,
    #[serde(rename = "beat-type")]
    beat_type: String,
}

#[derive(Debug, Serialize, Deserialize)]
struct Direction {
    #[serde(rename = "sound", default, skip_serializing_if = "Option::is_none")]
    sound: Option<Sound>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Sound {
    #[serde(rename = "@tempo", default, skip_serializing_if = "Option::is_none")]
    tempo: Option<f32>,
}

#[derive(Debug, Serialize, Deserialize)]
struct MusicXmlNote {
    #[serde(rename = "pitch", default, skip_serializing_if = "Option::is_none")]
    pitch: Option<Pitch>,
    #[serde(rename = "rest", default, skip_serializing_if = "Option::is_none")]
    rest: Option<Rest>,
    #[serde(rename = "duration")]
    duration: i32,
    #[serde(rename = "voice", default, skip_serializing_if = "Option::is_none")]
    voice: Option<String>,
    #[serde(rename = "type", default, skip_serializing_if = "Option::is_none")]
    note_type: Option<String>,
    #[serde(rename = "dynamics", default, skip_serializing_if = "Option::is_none")]
    dynamics: Option<Dynamics>,
}

#[derive(Debug, Serialize, Deserialize)]
struct Pitch {
    #[serde(rename = "step")]
    step: String,
    #[serde(rename = "alter", default, skip_serializing_if = "Option::is_none")]
    alter: Option<i32>,
    #[serde(rename = "octave")]
    octave: i32,
}

#[derive(Debug, Serialize, Deserialize)]
struct Rest {}

#[derive(Debug, Serialize, Deserialize)]
struct Dynamics {
    #[serde(rename = "$text", default)]
    value: Option<String>,
}

/// Read a MusicXML file and convert to Sequence
pub fn read_musicxml(path: &Path) -> Result<Sequence> {
    let file = File::open(path).map_err(|e| {
        if e.kind() == std::io::ErrorKind::NotFound {
            RifferError::FileNotFound(path.display().to_string())
        } else {
            RifferError::IoError(e)
        }
    })?;

    let reader = BufReader::new(file);
    let score: ScorePartwise =
        from_reader(reader).map_err(|e| RifferError::MusicXmlParseError(e.to_string()))?;

    // Get first part
    let part = score
        .parts
        .first()
        .ok_or_else(|| RifferError::MusicXmlParseError("No parts found in MusicXML".to_string()))?;

    // Parse notes and context
    let (notes, context) = parse_part(part)?;

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

    Ok(seq)
}

/// Parse a MusicXML part into notes and context
fn parse_part(part: &Part) -> Result<(Vec<Note>, Context)> {
    let mut notes = Vec::new();
    let mut current_tick: u64 = 0;
    let mut divisions = 1; // Divisions per quarter note
    let mut tempo: u16 = 120;
    let mut time_sig = TimeSignature::default();
    let mut key_sig: Option<KeySignature> = None;

    for measure in &part.measures {
        // Check attributes
        if let Some(attrs) = &measure.attributes {
            if let Some(div) = attrs.divisions {
                divisions = div;
            }
            if let Some(key) = &attrs.key {
                key_sig = Some(fifths_to_key(key.fifths, key.mode.as_deref()));
            }
            if let Some(time) = &attrs.time {
                time_sig = TimeSignature {
                    numerator: time.beats.parse().unwrap_or(4),
                    denominator: time.beat_type.parse().unwrap_or(4),
                };
            }
        }

        // Check directions for tempo
        for dir in &measure.directions {
            if let Some(sound) = &dir.sound {
                if let Some(t) = sound.tempo {
                    tempo = t as u16;
                }
            }
        }

        // Parse notes
        for note in &measure.notes {
            if note.rest.is_some() {
                // Rest - advance time but don't create note
                let ticks = (note.duration * 480 / divisions) as u64;
                current_tick += ticks;
                continue;
            }

            if let Some(pitch) = &note.pitch {
                let midi_pitch = pitch_to_midi(pitch);
                let duration_ticks = (note.duration * 480 / divisions) as u32;
                let velocity = 80u8; // Default velocity

                notes.push(Note::new(
                    midi_pitch,
                    duration_ticks,
                    velocity,
                    current_tick,
                ));
                current_tick += duration_ticks as u64;
            }
        }
    }

    let context = Context {
        key_signature: key_sig,
        time_signature: time_sig,
        tempo,
        ticks_per_quarter: 480,
    };

    Ok((notes, context))
}

/// Convert pitch element to MIDI note number
fn pitch_to_midi(pitch: &Pitch) -> u8 {
    let step_semitone = match pitch.step.as_str() {
        "C" => 0,
        "D" => 2,
        "E" => 4,
        "F" => 5,
        "G" => 7,
        "A" => 9,
        "B" => 11,
        _ => 0,
    };

    let alter = pitch.alter.unwrap_or(0);
    let octave = pitch.octave;

    // MIDI note: (octave + 1) * 12 + step + alter
    ((octave + 1) * 12 + step_semitone + alter) as u8
}

/// Convert fifths to KeySignature
fn fifths_to_key(fifths: i32, mode: Option<&str>) -> KeySignature {
    let mode = match mode {
        Some("minor") => Mode::Minor,
        _ => Mode::Major,
    };

    // Circle of fifths: F C G D A E B / Gb Db Ab Eb Bb
    let root = match fifths {
        -7 => PitchClass::B,  // Cb -> B
        -6 => PitchClass::Fs, // Gb
        -5 => PitchClass::Cs, // Db
        -4 => PitchClass::Gs, // Ab
        -3 => PitchClass::Ds, // Eb
        -2 => PitchClass::As, // Bb
        -1 => PitchClass::F,
        0 => PitchClass::C,
        1 => PitchClass::G,
        2 => PitchClass::D,
        3 => PitchClass::A,
        4 => PitchClass::E,
        5 => PitchClass::B,
        6 => PitchClass::Fs,
        7 => PitchClass::Cs,
        _ => PitchClass::C,
    };

    KeySignature::new(root, mode)
}

/// Write a Sequence to MusicXML file
pub fn write_musicxml(sequence: &Sequence, path: &Path) -> Result<()> {
    let divisions = 1; // 1 division = 1 quarter note = 480 ticks

    // Convert notes to MusicXML notes
    let mut mxl_notes: Vec<MusicXmlNote> = Vec::new();
    for note in &sequence.notes {
        let (step, alter, octave) = midi_to_pitch(note.pitch);
        let duration = (note.duration as i32) / 480; // Convert to divisions

        mxl_notes.push(MusicXmlNote {
            pitch: Some(Pitch {
                step,
                alter: if alter != 0 { Some(alter) } else { None },
                octave,
            }),
            rest: None,
            duration: duration.max(1),
            voice: Some("1".to_string()),
            note_type: Some(duration_to_type(note.duration)),
            dynamics: None,
        });
    }

    // Build attributes
    let attrs = Attributes {
        divisions: Some(divisions),
        key: sequence.context.key_signature.as_ref().map(|k| Key {
            fifths: key_to_fifths(k),
            mode: Some(if k.mode == Mode::Minor {
                "minor".to_string()
            } else {
                "major".to_string()
            }),
        }),
        time: Some(Time {
            beats: sequence.context.time_signature.numerator.to_string(),
            beat_type: sequence.context.time_signature.denominator.to_string(),
        }),
    };

    // Build measure
    let measure = Measure {
        number: "1".to_string(),
        attributes: Some(attrs),
        directions: vec![Direction {
            sound: Some(Sound {
                tempo: Some(sequence.context.tempo as f32),
            }),
        }],
        notes: mxl_notes,
    };

    // Build part
    let part = Part {
        id: "P1".to_string(),
        measures: vec![measure],
    };

    // Build score
    let score = ScorePartwise {
        part_list: PartList {
            score_parts: vec![ScorePart {
                id: "P1".to_string(),
                part_name: sequence.name.clone(),
            }],
        },
        parts: vec![part],
    };

    // Serialize to XML
    let xml = to_string(&score).map_err(|e| RifferError::MusicXmlParseError(e.to_string()))?;

    // Write with XML declaration
    let full_xml = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE score-partwise PUBLIC "-//Recordare//DTD MusicXML 4.0 Partwise//EN" "http://www.musicxml.org/dtds/partwise.dtd">
{}"#,
        xml
    );

    std::fs::write(path, full_xml)?;

    Ok(())
}

/// Convert MIDI note to pitch components
fn midi_to_pitch(midi: u8) -> (String, i32, i32) {
    let octave = (midi / 12) as i32 - 1;
    let note = midi % 12;

    let (step, alter) = match note {
        0 => ("C", 0),
        1 => ("C", 1),
        2 => ("D", 0),
        3 => ("D", 1),
        4 => ("E", 0),
        5 => ("F", 0),
        6 => ("F", 1),
        7 => ("G", 0),
        8 => ("G", 1),
        9 => ("A", 0),
        10 => ("A", 1),
        11 => ("B", 0),
        _ => ("C", 0),
    };

    (step.to_string(), alter, octave)
}

/// Convert key signature to fifths
fn key_to_fifths(key: &KeySignature) -> i32 {
    let base = match key.root {
        PitchClass::C => 0,
        PitchClass::G => 1,
        PitchClass::D => 2,
        PitchClass::A => 3,
        PitchClass::E => 4,
        PitchClass::B => 5,
        PitchClass::Fs => 6,
        PitchClass::Cs => 7,
        PitchClass::F => -1,
        PitchClass::As => -2,
        PitchClass::Ds => -3,
        PitchClass::Gs => -4,
    };

    if key.mode == Mode::Minor {
        base - 3 // Relative minor is 3 fifths back
    } else {
        base
    }
}

/// Convert duration in ticks to note type
fn duration_to_type(ticks: u32) -> String {
    match ticks {
        t if t >= 1920 => "whole",
        t if t >= 960 => "half",
        t if t >= 480 => "quarter",
        t if t >= 240 => "eighth",
        t if t >= 120 => "16th",
        _ => "32nd",
    }
    .to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pitch_to_midi() {
        let pitch = Pitch {
            step: "C".to_string(),
            alter: None,
            octave: 4,
        };
        assert_eq!(pitch_to_midi(&pitch), 60); // Middle C

        let pitch_sharp = Pitch {
            step: "F".to_string(),
            alter: Some(1),
            octave: 4,
        };
        assert_eq!(pitch_to_midi(&pitch_sharp), 66); // F#4
    }

    #[test]
    fn test_midi_to_pitch() {
        let (step, alter, octave) = midi_to_pitch(60);
        assert_eq!(step, "C");
        assert_eq!(alter, 0);
        assert_eq!(octave, 4);
    }
}
