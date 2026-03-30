//! Interval classification and analysis
//!
//! Classifies intervals by semitone distance, name, and consonance/dissonance quality.

use serde::{Deserialize, Serialize};

use crate::types::{Direction, IntervalQuality, Note};

/// Information about an interval between two notes
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntervalInfo {
    /// Index of the source note
    pub from_index: usize,
    /// Index of the target note
    pub to_index: usize,
    /// Signed interval in semitones (negative = descending)
    pub semitones: i8,
    /// Interval name (e.g., "minor 3rd")
    pub name: String,
    /// Consonance/dissonance classification
    pub quality: IntervalQuality,
    /// Direction of the interval
    pub direction: Direction,
}

impl IntervalInfo {
    /// Create a new interval info from two consecutive notes
    pub fn from_notes(from_index: usize, from: &Note, to: &Note) -> Self {
        let semitones = to.pitch as i8 - from.pitch as i8;
        let direction = match semitones.cmp(&0) {
            std::cmp::Ordering::Greater => Direction::Ascending,
            std::cmp::Ordering::Less => Direction::Descending,
            std::cmp::Ordering::Equal => Direction::Unison,
        };

        Self {
            from_index,
            to_index: from_index + 1,
            semitones,
            name: interval_name(semitones),
            quality: classify_interval(semitones),
            direction,
        }
    }
}

/// Classify an interval by its consonance/dissonance quality
pub fn classify_interval(semitones: i8) -> IntervalQuality {
    let abs_semitones = (semitones.abs() % 12) as u8;

    match abs_semitones {
        0 | 7 | 12 => IntervalQuality::PerfectConsonance, // Unison, P5, Octave
        3 | 4 | 8 | 9 => IntervalQuality::ImperfectConsonance, // m3, M3, m6, M6
        2 | 10 => IntervalQuality::MildDissonance,        // M2, m7
        1 | 6 | 11 => IntervalQuality::StrongDissonance,  // m2, tritone, M7
        5 => IntervalQuality::ImperfectConsonance, // P4 (context-dependent, treat as consonant)
        _ => IntervalQuality::MildDissonance,
    }
}

/// Get the name of an interval by semitone distance
pub fn interval_name(semitones: i8) -> String {
    let abs_semitones = semitones.abs() % 12;
    let direction = if semitones < 0 { "descending " } else { "" };

    let name = match abs_semitones {
        0 => "unison",
        1 => "minor 2nd",
        2 => "major 2nd",
        3 => "minor 3rd",
        4 => "major 3rd",
        5 => "perfect 4th",
        6 => "tritone",
        7 => "perfect 5th",
        8 => "minor 6th",
        9 => "major 6th",
        10 => "minor 7th",
        11 => "major 7th",
        _ => "octave",
    };

    // Handle octave+ intervals
    let octaves = semitones.abs() / 12;
    if octaves > 0 && abs_semitones == 0 {
        format!("{}octave", direction)
    } else if octaves > 0 {
        format!(
            "{}{} + {} octave{}",
            direction,
            name,
            octaves,
            if octaves > 1 { "s" } else { "" }
        )
    } else {
        format!("{}{}", direction, name)
    }
}

/// Check if an interval is dissonant
pub fn is_dissonant(semitones: i8) -> bool {
    matches!(
        classify_interval(semitones),
        IntervalQuality::MildDissonance | IntervalQuality::StrongDissonance
    )
}

/// Check if an interval is strongly dissonant (needs resolution)
pub fn is_strongly_dissonant(semitones: i8) -> bool {
    matches!(
        classify_interval(semitones),
        IntervalQuality::StrongDissonance
    )
}

/// Analyze all consecutive intervals in a note sequence
pub fn analyze_intervals(notes: &[Note]) -> Vec<IntervalInfo> {
    if notes.len() < 2 {
        return Vec::new();
    }

    notes
        .windows(2)
        .enumerate()
        .map(|(i, pair)| IntervalInfo::from_notes(i, &pair[0], &pair[1]))
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_classify_interval() {
        assert_eq!(classify_interval(0), IntervalQuality::PerfectConsonance); // Unison
        assert_eq!(classify_interval(7), IntervalQuality::PerfectConsonance); // P5
        assert_eq!(classify_interval(3), IntervalQuality::ImperfectConsonance); // m3
        assert_eq!(classify_interval(4), IntervalQuality::ImperfectConsonance); // M3
        assert_eq!(classify_interval(1), IntervalQuality::StrongDissonance); // m2
        assert_eq!(classify_interval(6), IntervalQuality::StrongDissonance); // tritone
        assert_eq!(classify_interval(2), IntervalQuality::MildDissonance); // M2
    }

    #[test]
    fn test_interval_name() {
        assert_eq!(interval_name(0), "unison");
        assert_eq!(interval_name(3), "minor 3rd");
        assert_eq!(interval_name(7), "perfect 5th");
        assert_eq!(interval_name(-3), "descending minor 3rd");
        assert_eq!(interval_name(12), "octave");
    }

    #[test]
    fn test_analyze_intervals() {
        let notes = vec![
            Note::new(60, 480, 80, 0),   // C4
            Note::new(64, 480, 80, 480), // E4 (M3 up)
            Note::new(67, 480, 80, 960), // G4 (m3 up)
        ];

        let intervals = analyze_intervals(&notes);
        assert_eq!(intervals.len(), 2);
        assert_eq!(intervals[0].semitones, 4); // C to E
        assert_eq!(intervals[1].semitones, 3); // E to G
    }
}
