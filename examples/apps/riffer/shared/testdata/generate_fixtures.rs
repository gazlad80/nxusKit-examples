//! Test fixture generator for Riffer
//!
//! Run with: cargo run --example generate_fixtures --features clips
//!
//! This generates the MIDI test files used by the test suite.

use std::path::Path;

// Include the riffer modules
#[path = "../types.rs"]
mod types;
#[path = "../errors.rs"]
mod errors;
#[path = "../formats/mod.rs"]
mod formats;

use types::{Note, Sequence, Context, TimeSignature, KeySignature, PitchClass, Mode};

fn main() {
    let testdata_dir = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("examples/riffer/testdata");

    // Generate C major scale
    generate_c_major_scale(&testdata_dir);

    // Generate E minor riff
    generate_e_minor_riff(&testdata_dir);

    // Generate chromatic test (for dissonance testing)
    generate_chromatic_test(&testdata_dir);

    println!("Test fixtures generated successfully!");
}

fn generate_c_major_scale(dir: &Path) {
    // C major scale: C D E F G A B C
    let notes = vec![
        Note::new(60, 480, 80, 0),      // C4
        Note::new(62, 480, 80, 480),    // D4
        Note::new(64, 480, 80, 960),    // E4
        Note::new(65, 480, 80, 1440),   // F4
        Note::new(67, 480, 80, 1920),   // G4
        Note::new(69, 480, 80, 2400),   // A4
        Note::new(71, 480, 80, 2880),   // B4
        Note::new(72, 480, 80, 3360),   // C5
    ];

    let mut seq = Sequence::new("c_major_scale".to_string(), notes);
    seq.context = Context {
        key_signature: Some(KeySignature::new(PitchClass::C, Mode::Major)),
        time_signature: TimeSignature { numerator: 4, denominator: 4 },
        tempo: 120,
        ticks_per_quarter: 480,
    };
    seq.name = Some("C Major Scale".to_string());

    let path = dir.join("c_major_scale.mid");
    formats::write_midi(&seq, &path).expect("Failed to write c_major_scale.mid");
    println!("Generated: {}", path.display());
}

fn generate_e_minor_riff(dir: &Path) {
    // E minor pentatonic riff with varying rhythms and dynamics
    // E G A B D E pattern with some variation
    let notes = vec![
        // Measure 1: E-G-A-B
        Note::new(64, 240, 90, 0),      // E4 (eighth)
        Note::new(67, 240, 85, 240),    // G4 (eighth)
        Note::new(69, 480, 80, 480),    // A4 (quarter)
        Note::new(71, 480, 95, 960),    // B4 (quarter, accent)
        // Measure 2: D-E-G-E
        Note::new(74, 240, 75, 1440),   // D5 (eighth)
        Note::new(76, 240, 70, 1680),   // E5 (eighth)
        Note::new(79, 480, 85, 1920),   // G5 (quarter)
        Note::new(76, 960, 90, 2400),   // E5 (half, resolution)
        // Measure 3: descending pattern
        Note::new(74, 240, 80, 3360),   // D5
        Note::new(71, 240, 80, 3600),   // B4
        Note::new(69, 240, 80, 3840),   // A4
        Note::new(67, 240, 80, 4080),   // G4
        Note::new(64, 960, 85, 4320),   // E4 (half, tonic resolution)
    ];

    let mut seq = Sequence::new("e_minor_riff".to_string(), notes);
    seq.context = Context {
        key_signature: Some(KeySignature::new(PitchClass::E, Mode::Minor)),
        time_signature: TimeSignature { numerator: 4, denominator: 4 },
        tempo: 100,
        ticks_per_quarter: 480,
    };
    seq.name = Some("E Minor Riff".to_string());

    let path = dir.join("e_minor_riff.mid");
    formats::write_midi(&seq, &path).expect("Failed to write e_minor_riff.mid");
    println!("Generated: {}", path.display());
}

fn generate_chromatic_test(dir: &Path) {
    // Chromatic passage with tritones and dissonances for testing
    let notes = vec![
        Note::new(60, 480, 80, 0),      // C4
        Note::new(61, 480, 80, 480),    // C#4 (chromatic)
        Note::new(62, 480, 80, 960),    // D4
        Note::new(66, 480, 80, 1440),   // F#4 (tritone from C)
        Note::new(67, 480, 80, 1920),   // G4 (resolution)
        Note::new(71, 480, 80, 2400),   // B4 (leading tone)
        Note::new(72, 480, 80, 2880),   // C5 (resolution)
    ];

    let mut seq = Sequence::new("chromatic_test".to_string(), notes);
    seq.context = Context {
        key_signature: Some(KeySignature::new(PitchClass::C, Mode::Major)),
        time_signature: TimeSignature { numerator: 4, denominator: 4 },
        tempo: 120,
        ticks_per_quarter: 480,
    };
    seq.name = Some("Chromatic Test".to_string());

    let path = dir.join("chromatic_test.mid");
    formats::write_midi(&seq, &path).expect("Failed to write chromatic_test.mid");
    println!("Generated: {}", path.display());
}
