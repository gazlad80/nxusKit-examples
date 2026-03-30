//! Scale definitions and pitch-class membership
//!
//! Provides scale interval patterns and membership checking.

use crate::types::{KeySignature, Mode, PitchClass, ScaleType};

/// Information about a scale
pub struct ScaleInfo {
    /// Scale type
    pub scale_type: ScaleType,
    /// Pitch classes in the scale
    pub pitch_classes: Vec<PitchClass>,
    /// Semitone intervals from root
    pub intervals: Vec<u8>,
}

/// Get the semitone intervals for a scale type
pub fn get_scale_intervals(scale_type: ScaleType) -> &'static [u8] {
    match scale_type {
        ScaleType::Major => &[0, 2, 4, 5, 7, 9, 11],
        ScaleType::NaturalMinor | ScaleType::Aeolian => &[0, 2, 3, 5, 7, 8, 10],
        ScaleType::HarmonicMinor => &[0, 2, 3, 5, 7, 8, 11],
        ScaleType::MelodicMinor => &[0, 2, 3, 5, 7, 9, 11],
        ScaleType::Pentatonic => &[0, 2, 4, 7, 9],
        ScaleType::Blues => &[0, 3, 5, 6, 7, 10],
        ScaleType::Dorian => &[0, 2, 3, 5, 7, 9, 10],
        ScaleType::Phrygian => &[0, 1, 3, 5, 7, 8, 10],
        ScaleType::Lydian => &[0, 2, 4, 6, 7, 9, 11],
        ScaleType::Mixolydian => &[0, 2, 4, 5, 7, 9, 10],
        ScaleType::Locrian => &[0, 1, 3, 5, 6, 8, 10],
    }
}

/// Get the scale type from a mode
pub fn mode_to_scale_type(mode: Mode) -> ScaleType {
    match mode {
        Mode::Major => ScaleType::Major,
        Mode::Minor | Mode::Aeolian => ScaleType::NaturalMinor,
        Mode::Dorian => ScaleType::Dorian,
        Mode::Phrygian => ScaleType::Phrygian,
        Mode::Lydian => ScaleType::Lydian,
        Mode::Mixolydian => ScaleType::Mixolydian,
        Mode::Locrian => ScaleType::Locrian,
    }
}

/// Check if a pitch class is in a scale
pub fn is_in_scale(pitch_class: PitchClass, key: &KeySignature) -> bool {
    let root_semitone = key.root.to_semitone();
    let scale_intervals = get_scale_intervals(mode_to_scale_type(key.mode));
    let pitch_semitone = pitch_class.to_semitone();

    // Calculate interval from root
    let interval = (pitch_semitone + 12 - root_semitone) % 12;

    scale_intervals.contains(&interval)
}

/// Get all pitch classes in a scale
pub fn get_scale_pitch_classes(key: &KeySignature) -> Vec<PitchClass> {
    let root_semitone = key.root.to_semitone();
    let intervals = get_scale_intervals(mode_to_scale_type(key.mode));

    intervals
        .iter()
        .map(|&interval| {
            let semitone = (root_semitone + interval) % 12;
            PitchClass::ALL[semitone as usize]
        })
        .collect()
}

/// Build scale info for a given key
pub fn build_scale_info(key: &KeySignature) -> ScaleInfo {
    let scale_type = mode_to_scale_type(key.mode);
    let intervals = get_scale_intervals(scale_type).to_vec();
    let pitch_classes = get_scale_pitch_classes(key);

    ScaleInfo {
        scale_type,
        pitch_classes,
        intervals,
    }
}

/// Check how many notes in a sequence are in the scale
pub fn count_in_scale(notes: &[crate::types::Note], key: &KeySignature) -> (usize, usize) {
    let in_scale = notes
        .iter()
        .filter(|n| is_in_scale(n.pitch_class(), key))
        .count();

    (in_scale, notes.len())
}

/// Calculate harmonic coherence as percentage
pub fn harmonic_coherence(notes: &[crate::types::Note], key: &KeySignature) -> f32 {
    let (in_scale, total) = count_in_scale(notes, key);
    if total == 0 {
        return 100.0;
    }
    (in_scale as f32 / total as f32) * 100.0
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_in_scale_c_major() {
        let c_major = KeySignature::new(PitchClass::C, Mode::Major);

        // C, D, E, F, G, A, B should be in scale
        assert!(is_in_scale(PitchClass::C, &c_major));
        assert!(is_in_scale(PitchClass::D, &c_major));
        assert!(is_in_scale(PitchClass::E, &c_major));
        assert!(is_in_scale(PitchClass::G, &c_major));

        // C#, F#, etc. should not be
        assert!(!is_in_scale(PitchClass::Cs, &c_major));
        assert!(!is_in_scale(PitchClass::Fs, &c_major));
    }

    #[test]
    fn test_is_in_scale_e_minor() {
        let e_minor = KeySignature::new(PitchClass::E, Mode::Minor);

        // E, F#, G, A, B, C, D should be in scale
        assert!(is_in_scale(PitchClass::E, &e_minor));
        assert!(is_in_scale(PitchClass::Fs, &e_minor));
        assert!(is_in_scale(PitchClass::G, &e_minor));

        // D# should not be
        assert!(!is_in_scale(PitchClass::Ds, &e_minor));
    }

    #[test]
    fn test_get_scale_pitch_classes() {
        let c_major = KeySignature::new(PitchClass::C, Mode::Major);
        let pcs = get_scale_pitch_classes(&c_major);

        assert_eq!(pcs.len(), 7);
        assert_eq!(pcs[0], PitchClass::C);
        assert_eq!(pcs[1], PitchClass::D);
        assert_eq!(pcs[2], PitchClass::E);
    }
}
