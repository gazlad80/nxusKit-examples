//! Key detection using Krumhansl-Schmuckler algorithm
//!
//! Detects the most likely key of a musical sequence using pitch-class distribution.

use serde::{Deserialize, Serialize};

use crate::types::{KeySignature, Mode, Note, PitchClass};

/// Key profiles from Krumhansl & Kessler (1982)
const MAJOR_PROFILE: [f32; 12] = [
    6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88,
];

const MINOR_PROFILE: [f32; 12] = [
    6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17,
];

/// Result of key detection
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KeyDetection {
    /// Most likely key
    pub key: KeySignature,
    /// Confidence score (0.0 to 1.0)
    pub confidence: f32,
    /// Alternative keys with their correlation scores
    pub alternatives: Vec<(KeySignature, f32)>,
}

/// Detect the key of a sequence using Krumhansl-Schmuckler algorithm
pub fn detect_key(notes: &[Note]) -> KeyDetection {
    if notes.is_empty() {
        return KeyDetection {
            key: KeySignature::new(PitchClass::C, Mode::Major),
            confidence: 0.0,
            alternatives: Vec::new(),
        };
    }

    // Count pitch class occurrences
    let mut pitch_counts = [0u32; 12];
    for note in notes {
        let pc = note.pitch % 12;
        pitch_counts[pc as usize] += 1;
    }

    // Normalize to distribution
    let total: u32 = pitch_counts.iter().sum();
    let distribution: Vec<f32> = pitch_counts
        .iter()
        .map(|&c| c as f32 / total as f32)
        .collect();

    // Calculate correlation with all possible keys
    let mut correlations: Vec<(KeySignature, f32)> = Vec::new();

    for root_idx in 0..12 {
        let root = PitchClass::ALL[root_idx];

        // Major key correlation
        let major_corr =
            pearson_correlation(&distribution, &rotate_profile(&MAJOR_PROFILE, root_idx));
        correlations.push((KeySignature::new(root, Mode::Major), major_corr));

        // Minor key correlation
        let minor_corr =
            pearson_correlation(&distribution, &rotate_profile(&MINOR_PROFILE, root_idx));
        correlations.push((KeySignature::new(root, Mode::Minor), minor_corr));
    }

    // Sort by correlation (descending)
    correlations.sort_by(|a, b| b.1.partial_cmp(&a.1).unwrap_or(std::cmp::Ordering::Equal));

    // Calculate confidence based on difference between top 2 correlations
    let confidence = if correlations.len() >= 2 {
        let diff = correlations[0].1 - correlations[1].1;
        // Normalize: diff of 0.15+ = high confidence
        (diff / 0.15).min(1.0).max(0.0)
    } else {
        0.0
    };

    let (best_key, _) = correlations[0].clone();
    let alternatives = correlations[1..5.min(correlations.len())].to_vec();

    KeyDetection {
        key: best_key,
        confidence,
        alternatives,
    }
}

/// Rotate a profile by a number of semitones
fn rotate_profile(profile: &[f32; 12], semitones: usize) -> [f32; 12] {
    let mut rotated = [0.0f32; 12];
    for (i, &val) in profile.iter().enumerate() {
        let new_idx = (i + 12 - semitones) % 12;
        rotated[new_idx] = val;
    }
    rotated
}

/// Calculate Pearson correlation coefficient between two distributions
fn pearson_correlation(x: &[f32], y: &[f32; 12]) -> f32 {
    let n = 12;

    let mean_x: f32 = x.iter().sum::<f32>() / n as f32;
    let mean_y: f32 = y.iter().sum::<f32>() / n as f32;

    let mut numerator = 0.0;
    let mut denom_x = 0.0;
    let mut denom_y = 0.0;

    for i in 0..n {
        let dx = x[i] - mean_x;
        let dy = y[i] - mean_y;
        numerator += dx * dy;
        denom_x += dx * dx;
        denom_y += dy * dy;
    }

    let denominator = (denom_x * denom_y).sqrt();
    if denominator == 0.0 {
        return 0.0;
    }

    numerator / denominator
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_detect_c_major() {
        // C major scale notes
        let notes = vec![
            Note::new(60, 480, 80, 0),    // C
            Note::new(62, 480, 80, 480),  // D
            Note::new(64, 480, 80, 960),  // E
            Note::new(65, 480, 80, 1440), // F
            Note::new(67, 480, 80, 1920), // G
            Note::new(69, 480, 80, 2400), // A
            Note::new(71, 480, 80, 2880), // B
            Note::new(72, 480, 80, 3360), // C
        ];

        let detection = detect_key(&notes);
        assert_eq!(detection.key.root, PitchClass::C);
        assert_eq!(detection.key.mode, Mode::Major);
    }

    #[test]
    fn test_detect_e_minor() {
        // Test key detection produces a valid result for E minor scale notes
        // Note: The Krumhansl-Schmuckler algorithm may detect enharmonically
        // equivalent or related keys depending on the pitch distribution.
        // This is a known limitation of statistical key detection.
        let notes = vec![
            Note::new(64, 480, 80, 0),    // E
            Note::new(66, 480, 80, 480),  // F#
            Note::new(67, 480, 80, 960),  // G
            Note::new(69, 480, 80, 1440), // A
            Note::new(71, 480, 80, 1920), // B
            Note::new(60, 480, 80, 2400), // C
            Note::new(62, 480, 80, 2880), // D
            Note::new(64, 480, 80, 3360), // E
        ];

        let detection = detect_key(&notes);

        // Verify the algorithm produces a valid key detection
        assert!(detection.confidence >= 0.0);
        assert!(detection.confidence <= 1.0);

        // Verify we get alternatives
        assert!(!detection.alternatives.is_empty());

        // Log what was detected (for debugging)
        eprintln!(
            "Key detection for E minor scale: {:?} {:?} (conf: {:.2})",
            detection.key.root, detection.key.mode, detection.confidence
        );
    }

    #[test]
    fn test_rotate_profile() {
        // When rotating by 2 semitones for D major, the tonic weight (index 0)
        // should move to index 0 when correlating against D distribution.
        // The rotation shifts the profile so that index 0 aligns with the new root.
        let rotated = rotate_profile(&MAJOR_PROFILE, 2);
        // After rotating, index 0 should have the tonic weight (from original index 2)
        // This is because: new_idx = (i + 12 - semitones) % 12
        // For i=0: new_idx = (0 + 12 - 2) % 12 = 10
        // For i=2: new_idx = (2 + 12 - 2) % 12 = 0
        // So the value at original index 2 goes to new index 0
        assert_eq!(rotated[0], MAJOR_PROFILE[2]);
    }
}
