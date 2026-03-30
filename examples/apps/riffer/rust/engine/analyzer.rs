//! Music Sequence Analyzer
//!
//! Provides comprehensive analysis of musical sequences including:
//! - Key detection
//! - Scale membership analysis
//! - Interval analysis
//! - Melodic contour detection
//! - Statistical summaries

use serde::{Deserialize, Serialize};

use crate::errors::Result;
use crate::theory::intervals::{IntervalInfo, analyze_intervals};
use crate::theory::keys::{KeyDetection, detect_key};
use crate::theory::scales::{count_in_scale, get_scale_pitch_classes, harmonic_coherence};
use crate::types::{
    ContourType, Direction, IntervalQuality, KeySignature, Note, PitchClass, Sequence,
};

/// Complete analysis result for a sequence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AnalysisResult {
    /// Sequence name
    pub name: Option<String>,
    /// Number of notes
    pub note_count: usize,
    /// Total duration in ticks
    pub duration_ticks: u64,
    /// Key detection result
    pub key_detection: KeyDetection,
    /// Scale analysis
    pub scale_analysis: ScaleAnalysis,
    /// Interval analysis
    pub interval_analysis: IntervalAnalysis,
    /// Melodic contour analysis
    pub contour_analysis: ContourAnalysis,
    /// Rhythm analysis
    pub rhythm_analysis: RhythmAnalysis,
    /// Dynamics analysis
    pub dynamics_analysis: DynamicsAnalysis,
}

/// Scale membership analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScaleAnalysis {
    /// Detected or specified key
    pub key: KeySignature,
    /// Notes in the scale
    pub scale_notes: Vec<PitchClass>,
    /// Count of notes in scale
    pub in_scale_count: usize,
    /// Count of notes out of scale
    pub out_of_scale_count: usize,
    /// Percentage of notes in scale
    pub coherence_percentage: f32,
    /// Pitch classes used (unique)
    pub pitch_classes_used: Vec<PitchClass>,
    /// Chromatic alterations (out-of-scale notes)
    pub chromatic_alterations: Vec<ChromaticAlteration>,
}

/// A chromatic alteration (out-of-scale note)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ChromaticAlteration {
    /// Note index in sequence
    pub note_index: usize,
    /// The pitch class
    pub pitch_class: PitchClass,
    /// MIDI pitch
    pub pitch: u8,
}

/// Interval analysis summary
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct IntervalAnalysis {
    /// Total number of intervals
    pub count: usize,
    /// All intervals (if requested)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub intervals: Option<Vec<IntervalInfo>>,
    /// Count by quality
    pub by_quality: IntervalQualityCounts,
    /// Count by direction
    pub by_direction: DirectionCounts,
    /// Largest interval (absolute semitones)
    pub largest_interval: i8,
    /// Smallest interval (absolute semitones, excluding unison)
    pub smallest_interval: i8,
    /// Average interval size
    pub average_interval: f32,
    /// Interval variety (unique interval sizes)
    pub interval_variety: usize,
}

/// Counts by interval quality
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct IntervalQualityCounts {
    pub perfect_consonance: usize,
    pub imperfect_consonance: usize,
    pub mild_dissonance: usize,
    pub strong_dissonance: usize,
}

/// Counts by direction
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct DirectionCounts {
    pub ascending: usize,
    pub descending: usize,
    pub unison: usize,
}

/// Melodic contour analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContourAnalysis {
    /// Overall contour type
    pub contour_type: ContourType,
    /// Number of direction changes
    pub direction_changes: usize,
    /// Highest pitch (MIDI note number)
    pub highest_pitch: u8,
    /// Lowest pitch (MIDI note number)
    pub lowest_pitch: u8,
    /// Pitch range in semitones
    pub pitch_range: u8,
    /// Position of highest note (0.0 to 1.0)
    pub climax_position: f32,
    /// Position of lowest note (0.0 to 1.0)
    pub nadir_position: f32,
}

/// Rhythm analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RhythmAnalysis {
    /// Unique duration values
    pub unique_durations: usize,
    /// Most common duration
    pub most_common_duration: u32,
    /// Duration variety (entropy-like measure, 0-1)
    pub duration_variety: f32,
    /// Average duration in ticks
    pub average_duration: f32,
    /// Shortest note duration
    pub shortest_duration: u32,
    /// Longest note duration
    pub longest_duration: u32,
}

/// Dynamics analysis
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DynamicsAnalysis {
    /// Minimum velocity
    pub min_velocity: u8,
    /// Maximum velocity
    pub max_velocity: u8,
    /// Velocity range
    pub velocity_range: u8,
    /// Average velocity
    pub average_velocity: f32,
    /// Velocity variance
    pub velocity_variance: f32,
    /// Has dynamic variation (range > 20)
    pub has_dynamics: bool,
}

/// Analyze a sequence and return comprehensive analysis
pub fn analyze_sequence(sequence: &Sequence, include_intervals: bool) -> Result<AnalysisResult> {
    let notes = &sequence.notes;

    // Key detection
    let key_detection = detect_key(notes);

    // Use detected key or specified key
    let key = sequence
        .context
        .key_signature
        .clone()
        .unwrap_or_else(|| key_detection.key.clone());

    // Scale analysis
    let scale_analysis = analyze_scale(notes, &key);

    // Interval analysis
    let interval_analysis = analyze_interval_summary(notes, include_intervals);

    // Contour analysis
    let contour_analysis = analyze_contour(notes);

    // Rhythm analysis
    let rhythm_analysis = analyze_rhythm(notes);

    // Dynamics analysis
    let dynamics_analysis = analyze_dynamics(notes);

    Ok(AnalysisResult {
        name: sequence.name.clone(),
        note_count: notes.len(),
        duration_ticks: sequence.total_duration(),
        key_detection,
        scale_analysis,
        interval_analysis,
        contour_analysis,
        rhythm_analysis,
        dynamics_analysis,
    })
}

/// Analyze scale membership
fn analyze_scale(notes: &[Note], key: &KeySignature) -> ScaleAnalysis {
    let (in_scale, total) = count_in_scale(notes, key);
    let out_of_scale = total - in_scale;
    let coherence = harmonic_coherence(notes, key);

    // Get scale notes
    let scale_notes = get_scale_pitch_classes(key);

    // Find unique pitch classes used
    let mut pitch_classes_used: Vec<PitchClass> = notes
        .iter()
        .map(|n| n.pitch_class())
        .collect::<std::collections::HashSet<_>>()
        .into_iter()
        .collect();
    pitch_classes_used.sort_by_key(|pc| pc.to_semitone());

    // Find chromatic alterations
    let chromatic_alterations: Vec<ChromaticAlteration> = notes
        .iter()
        .enumerate()
        .filter(|(_, n)| !crate::theory::scales::is_in_scale(n.pitch_class(), key))
        .map(|(i, n)| ChromaticAlteration {
            note_index: i,
            pitch_class: n.pitch_class(),
            pitch: n.pitch,
        })
        .collect();

    ScaleAnalysis {
        key: key.clone(),
        scale_notes,
        in_scale_count: in_scale,
        out_of_scale_count: out_of_scale,
        coherence_percentage: coherence,
        pitch_classes_used,
        chromatic_alterations,
    }
}

/// Analyze intervals
fn analyze_interval_summary(notes: &[Note], include_all: bool) -> IntervalAnalysis {
    let intervals = analyze_intervals(notes);

    if intervals.is_empty() {
        return IntervalAnalysis {
            count: 0,
            intervals: if include_all { Some(Vec::new()) } else { None },
            by_quality: IntervalQualityCounts::default(),
            by_direction: DirectionCounts::default(),
            largest_interval: 0,
            smallest_interval: 0,
            average_interval: 0.0,
            interval_variety: 0,
        };
    }

    // Count by quality
    let mut by_quality = IntervalQualityCounts::default();
    for interval in &intervals {
        match interval.quality {
            IntervalQuality::PerfectConsonance => by_quality.perfect_consonance += 1,
            IntervalQuality::ImperfectConsonance => by_quality.imperfect_consonance += 1,
            IntervalQuality::MildDissonance => by_quality.mild_dissonance += 1,
            IntervalQuality::StrongDissonance => by_quality.strong_dissonance += 1,
        }
    }

    // Count by direction
    let mut by_direction = DirectionCounts::default();
    for interval in &intervals {
        match interval.direction {
            Direction::Ascending => by_direction.ascending += 1,
            Direction::Descending => by_direction.descending += 1,
            Direction::Unison => by_direction.unison += 1,
        }
    }

    // Statistics
    let abs_intervals: Vec<i8> = intervals.iter().map(|i| i.semitones.abs()).collect();
    let largest = *abs_intervals.iter().max().unwrap_or(&0);
    let smallest = *abs_intervals.iter().filter(|&&x| x > 0).min().unwrap_or(&0);
    let sum: i32 = abs_intervals.iter().map(|&x| x as i32).sum();
    let average = sum as f32 / intervals.len() as f32;

    // Interval variety (unique absolute intervals)
    let variety: std::collections::HashSet<i8> = abs_intervals.into_iter().collect();

    IntervalAnalysis {
        count: intervals.len(),
        intervals: if include_all { Some(intervals) } else { None },
        by_quality,
        by_direction,
        largest_interval: largest,
        smallest_interval: smallest,
        average_interval: average,
        interval_variety: variety.len(),
    }
}

/// Detect melodic contour
pub fn analyze_contour(notes: &[Note]) -> ContourAnalysis {
    if notes.is_empty() {
        return ContourAnalysis {
            contour_type: ContourType::Static,
            direction_changes: 0,
            highest_pitch: 0,
            lowest_pitch: 0,
            pitch_range: 0,
            climax_position: 0.0,
            nadir_position: 0.0,
        };
    }

    let pitches: Vec<u8> = notes.iter().map(|n| n.pitch).collect();

    // Find extremes
    let highest = *pitches.iter().max().unwrap();
    let lowest = *pitches.iter().min().unwrap();
    let range = highest - lowest;

    // Find positions of extremes
    let climax_idx = pitches.iter().position(|&p| p == highest).unwrap();
    let nadir_idx = pitches.iter().position(|&p| p == lowest).unwrap();
    let climax_position = climax_idx as f32 / (pitches.len() - 1).max(1) as f32;
    let nadir_position = nadir_idx as f32 / (pitches.len() - 1).max(1) as f32;

    // Count direction changes
    let mut direction_changes = 0;
    let mut prev_direction: Option<i8> = None;

    for i in 1..pitches.len() {
        let diff = pitches[i] as i16 - pitches[i - 1] as i16;
        let direction = if diff > 0 {
            1
        } else if diff < 0 {
            -1
        } else {
            0
        };

        if direction != 0 {
            if let Some(prev) = prev_direction {
                if prev != direction {
                    direction_changes += 1;
                }
            }
            prev_direction = Some(direction);
        }
    }

    // Determine contour type
    let contour_type =
        detect_contour_type(&pitches, climax_position, nadir_position, direction_changes);

    ContourAnalysis {
        contour_type,
        direction_changes,
        highest_pitch: highest,
        lowest_pitch: lowest,
        pitch_range: range,
        climax_position,
        nadir_position,
    }
}

/// Detect the overall contour type
fn detect_contour_type(
    pitches: &[u8],
    climax_pos: f32,
    nadir_pos: f32,
    direction_changes: usize,
) -> ContourType {
    if pitches.len() < 2 {
        return ContourType::Static;
    }

    let first = pitches[0];
    let last = *pitches.last().unwrap();

    // Check for predominantly ascending or descending
    if direction_changes == 0 {
        if last > first {
            return ContourType::Ascending;
        } else if last < first {
            return ContourType::Descending;
        } else {
            return ContourType::Static;
        }
    }

    // Check for arch (climax in middle)
    if climax_pos > 0.2 && climax_pos < 0.8 && direction_changes <= 2 {
        return ContourType::Arch;
    }

    // Check for inverse arch (nadir in middle)
    if nadir_pos > 0.2 && nadir_pos < 0.8 && direction_changes <= 2 {
        return ContourType::InverseArch;
    }

    // Multiple direction changes = wave
    if direction_changes >= 2 {
        return ContourType::Wave;
    }

    // Default based on overall direction
    if last > first {
        ContourType::Ascending
    } else if last < first {
        ContourType::Descending
    } else {
        ContourType::Static
    }
}

/// Analyze rhythm characteristics
fn analyze_rhythm(notes: &[Note]) -> RhythmAnalysis {
    if notes.is_empty() {
        return RhythmAnalysis {
            unique_durations: 0,
            most_common_duration: 0,
            duration_variety: 0.0,
            average_duration: 0.0,
            shortest_duration: 0,
            longest_duration: 0,
        };
    }

    let durations: Vec<u32> = notes.iter().map(|n| n.duration).collect();

    // Find unique durations
    let unique: std::collections::HashSet<u32> = durations.iter().copied().collect();
    let unique_count = unique.len();

    // Find most common
    let mut counts: std::collections::HashMap<u32, usize> = std::collections::HashMap::new();
    for &d in &durations {
        *counts.entry(d).or_insert(0) += 1;
    }
    let most_common = counts
        .into_iter()
        .max_by_key(|(_, count)| *count)
        .map(|(d, _)| d)
        .unwrap_or(0);

    // Statistics
    let sum: u64 = durations.iter().map(|&d| d as u64).sum();
    let average = sum as f32 / durations.len() as f32;
    let shortest = *durations.iter().min().unwrap();
    let longest = *durations.iter().max().unwrap();

    // Duration variety (normalized entropy-like measure)
    let variety = if unique_count <= 1 {
        0.0
    } else {
        // Simple measure: unique / total, capped at 1
        (unique_count as f32 / durations.len() as f32).min(1.0)
    };

    RhythmAnalysis {
        unique_durations: unique_count,
        most_common_duration: most_common,
        duration_variety: variety,
        average_duration: average,
        shortest_duration: shortest,
        longest_duration: longest,
    }
}

/// Analyze dynamics (velocity)
fn analyze_dynamics(notes: &[Note]) -> DynamicsAnalysis {
    if notes.is_empty() {
        return DynamicsAnalysis {
            min_velocity: 0,
            max_velocity: 0,
            velocity_range: 0,
            average_velocity: 0.0,
            velocity_variance: 0.0,
            has_dynamics: false,
        };
    }

    let velocities: Vec<u8> = notes.iter().map(|n| n.velocity).collect();

    let min = *velocities.iter().min().unwrap();
    let max = *velocities.iter().max().unwrap();
    let range = max - min;

    let sum: u32 = velocities.iter().map(|&v| v as u32).sum();
    let average = sum as f32 / velocities.len() as f32;

    // Calculate variance
    let variance: f32 = velocities
        .iter()
        .map(|&v| {
            let diff = v as f32 - average;
            diff * diff
        })
        .sum::<f32>()
        / velocities.len() as f32;

    DynamicsAnalysis {
        min_velocity: min,
        max_velocity: max,
        velocity_range: range,
        average_velocity: average,
        velocity_variance: variance,
        has_dynamics: range > 20,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{Context, Mode, TimeSignature};

    fn create_c_major_scale() -> Sequence {
        let notes = vec![
            Note::new(60, 480, 80, 0),
            Note::new(62, 480, 80, 480),
            Note::new(64, 480, 80, 960),
            Note::new(65, 480, 80, 1440),
            Note::new(67, 480, 80, 1920),
            Note::new(69, 480, 80, 2400),
            Note::new(71, 480, 80, 2880),
            Note::new(72, 480, 80, 3360),
        ];

        let mut seq = Sequence::new("test".to_string(), notes);
        seq.context = Context {
            key_signature: Some(KeySignature::new(PitchClass::C, Mode::Major)),
            time_signature: TimeSignature::default(),
            tempo: 120,
            ticks_per_quarter: 480,
        };
        seq.name = Some("C Major Scale".to_string());
        seq
    }

    #[test]
    fn test_analyze_c_major_scale() {
        let seq = create_c_major_scale();
        let result = analyze_sequence(&seq, false).unwrap();

        assert_eq!(result.note_count, 8);
        assert_eq!(result.scale_analysis.in_scale_count, 8);
        assert_eq!(result.scale_analysis.out_of_scale_count, 0);
        assert!(result.scale_analysis.coherence_percentage > 99.0);
    }

    #[test]
    fn test_contour_ascending() {
        let seq = create_c_major_scale();
        let result = analyze_sequence(&seq, false).unwrap();

        assert_eq!(result.contour_analysis.contour_type, ContourType::Ascending);
        assert_eq!(result.contour_analysis.direction_changes, 0);
    }

    #[test]
    fn test_interval_analysis() {
        let seq = create_c_major_scale();
        let result = analyze_sequence(&seq, true).unwrap();

        assert_eq!(result.interval_analysis.count, 7);
        // All seconds (some major, some minor)
        assert!(
            result.interval_analysis.by_quality.mild_dissonance > 0
                || result.interval_analysis.by_quality.strong_dissonance > 0
        );
    }

    #[test]
    fn test_rhythm_analysis() {
        let seq = create_c_major_scale();
        let result = analyze_sequence(&seq, false).unwrap();

        // All same duration
        assert_eq!(result.rhythm_analysis.unique_durations, 1);
        assert_eq!(result.rhythm_analysis.most_common_duration, 480);
    }

    #[test]
    fn test_dynamics_uniform() {
        let seq = create_c_major_scale();
        let result = analyze_sequence(&seq, false).unwrap();

        // All same velocity
        assert_eq!(result.dynamics_analysis.velocity_range, 0);
        assert!(!result.dynamics_analysis.has_dynamics);
    }
}
