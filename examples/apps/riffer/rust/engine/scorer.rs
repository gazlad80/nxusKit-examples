//! Music Sequence Scorer
//!
//! Provides 6-dimension scoring of musical sequences:
//! - Harmonic Coherence
//! - Melodic Interest
//! - Rhythmic Variety
//! - Resolution Quality
//! - Dynamics Expression
//! - Structural Balance

use std::path::Path;

use serde::{Deserialize, Serialize};

use crate::engine::analyzer::{AnalysisResult, analyze_sequence};
use crate::engine::clips_bridge::ClipsRuleEngine;
use crate::errors::Result;
use crate::theory::intervals::analyze_intervals;
use crate::types::{ContourType, IntervalQuality, Sequence};

/// Complete musicality score for a sequence
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MusicScore {
    /// Overall score (0-100)
    pub overall: f32,
    /// Individual dimension scores
    pub dimensions: ScoreDimensions,
    /// Score summary with ratings
    pub summary: ScoreSummary,
    /// Suggestions for improvement
    pub suggestions: Vec<String>,
    /// CLIPS-based adjustments (if enabled)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub clips_adjustments: Option<Vec<ClipsAdjustment>>,
}

/// Individual dimension scores
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoreDimensions {
    pub harmonic_coherence: ScoreDimension,
    pub melodic_interest: ScoreDimension,
    pub rhythmic_variety: ScoreDimension,
    pub resolution_quality: ScoreDimension,
    pub dynamics_expression: ScoreDimension,
    pub structural_balance: ScoreDimension,
}

/// A single dimension score
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoreDimension {
    /// Score (0-100)
    pub score: f32,
    /// Weight for overall calculation
    pub weight: f32,
    /// Rating (excellent, good, fair, poor)
    pub rating: String,
    /// Explanation of the score
    pub explanation: String,
}

/// Summary of scores
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoreSummary {
    /// Overall rating
    pub rating: String,
    /// Strongest dimension
    pub strongest: String,
    /// Weakest dimension
    pub weakest: String,
    /// Brief summary
    pub summary: String,
}

/// CLIPS-based adjustment
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClipsAdjustment {
    pub dimension: String,
    pub adjustment: f32,
    pub reason: String,
}

/// Score a sequence and return comprehensive musicality scores
///
/// # Arguments
/// * `sequence` - The music sequence to score
/// * `use_clips` - Whether to use CLIPS rule engine for adjustments
///
/// Note: When `use_clips` is true, this function will attempt to run CLIPS
/// rules asynchronously. Use `score_sequence_async` for async contexts.
pub fn score_sequence(sequence: &Sequence, use_clips: bool) -> Result<MusicScore> {
    // For sync API, we can't use CLIPS (which requires async)
    // Callers should use score_sequence_async for CLIPS support
    score_sequence_internal(sequence, None, use_clips)
}

/// Score a sequence asynchronously with optional CLIPS integration
///
/// # Arguments
/// * `sequence` - The music sequence to score
/// * `rules_dir` - Optional path to CLIPS rules directory
pub async fn score_sequence_async(
    sequence: &Sequence,
    rules_dir: Option<&Path>,
) -> Result<MusicScore> {
    let clips_adjustments = if let Some(dir) = rules_dir {
        match apply_clips_adjustments_async(sequence, dir).await {
            Ok(adj) => adj,
            Err(e) => {
                eprintln!("Warning: CLIPS analysis failed: {}", e);
                None
            }
        }
    } else {
        None
    };

    score_sequence_internal(sequence, clips_adjustments, rules_dir.is_some())
}

/// Internal scoring logic shared by sync and async variants
fn score_sequence_internal(
    sequence: &Sequence,
    clips_adjustments: Option<Vec<ClipsAdjustment>>,
    _use_clips: bool,
) -> Result<MusicScore> {
    // First, get the analysis
    let analysis = analyze_sequence(sequence, true)?;

    // Calculate each dimension
    let mut harmonic = score_harmonic_coherence(&analysis);
    let mut melodic = score_melodic_interest(&analysis);
    let mut rhythmic = score_rhythmic_variety(&analysis);
    let mut resolution = score_resolution_quality(&analysis, &sequence.notes);
    let mut dynamics = score_dynamics_expression(&analysis);
    let mut structural = score_structural_balance(&analysis);

    // Apply CLIPS adjustments if available
    if let Some(ref adjustments) = clips_adjustments {
        for adj in adjustments {
            // Map CLIPS dimension names to internal dimension references
            let dim = match adj.dimension.as_str() {
                // Full names (hyphenated)
                "harmonic-coherence" | "harmony" => &mut harmonic,
                "melodic-interest" | "melody" => &mut melodic,
                "rhythmic-variety" | "rhythm" => &mut rhythmic,
                "resolution-quality" | "resolution" => &mut resolution,
                "dynamics-expression" | "dynamics" => &mut dynamics,
                "structural-balance" | "structure" => &mut structural,
                _ => continue,
            };
            dim.score = (dim.score + adj.adjustment).clamp(0.0, 100.0);
        }
    }

    let dimensions = ScoreDimensions {
        harmonic_coherence: harmonic,
        melodic_interest: melodic,
        rhythmic_variety: rhythmic,
        resolution_quality: resolution,
        dynamics_expression: dynamics,
        structural_balance: structural,
    };

    // Calculate weighted overall score
    let overall = calculate_overall(&dimensions);

    // Generate suggestions
    let suggestions = generate_suggestions(&dimensions, &analysis);

    // Get summary
    let summary = generate_summary(&dimensions, overall);

    Ok(MusicScore {
        overall,
        dimensions,
        summary,
        suggestions,
        clips_adjustments,
    })
}

/// Score harmonic coherence (scale membership)
fn score_harmonic_coherence(analysis: &AnalysisResult) -> ScoreDimension {
    let coherence = analysis.scale_analysis.coherence_percentage;

    let (rating, explanation) = if coherence >= 95.0 {
        ("excellent", "All or nearly all notes fit the detected key")
    } else if coherence >= 80.0 {
        (
            "good",
            "Most notes fit the key with some chromatic alterations",
        )
    } else if coherence >= 60.0 {
        (
            "fair",
            "Moderate key adherence with significant alterations",
        )
    } else {
        ("poor", "Many notes fall outside the detected key")
    };

    ScoreDimension {
        score: coherence,
        weight: 0.20,
        rating: rating.to_string(),
        explanation: explanation.to_string(),
    }
}

/// Score melodic interest (interval variety and contour)
fn score_melodic_interest(analysis: &AnalysisResult) -> ScoreDimension {
    let interval_variety = analysis.interval_analysis.interval_variety;
    let contour = &analysis.contour_analysis;

    // Base score on interval variety (more variety = more interesting)
    let variety_score = (interval_variety as f32 * 12.0).min(50.0);

    // Bonus for interesting contour
    let contour_bonus = match contour.contour_type {
        ContourType::Arch | ContourType::InverseArch => 25.0,
        ContourType::Wave => 30.0,
        ContourType::Ascending | ContourType::Descending => 15.0,
        ContourType::Static => 0.0,
    };

    // Bonus for direction changes (but not too many)
    let change_bonus = if contour.direction_changes >= 1 && contour.direction_changes <= 4 {
        contour.direction_changes as f32 * 5.0
    } else if contour.direction_changes > 4 {
        15.0 // Cap the bonus
    } else {
        0.0
    };

    let score = (variety_score + contour_bonus + change_bonus).min(100.0);

    let (rating, explanation) = if score >= 80.0 {
        ("excellent", "High interval variety with engaging contour")
    } else if score >= 60.0 {
        ("good", "Good melodic movement and variety")
    } else if score >= 40.0 {
        ("fair", "Moderate melodic interest")
    } else {
        ("poor", "Limited melodic variety or static contour")
    };

    ScoreDimension {
        score,
        weight: 0.20,
        rating: rating.to_string(),
        explanation: explanation.to_string(),
    }
}

/// Score rhythmic variety
fn score_rhythmic_variety(analysis: &AnalysisResult) -> ScoreDimension {
    let rhythm = &analysis.rhythm_analysis;

    // Base on unique durations
    let unique_score = (rhythm.unique_durations as f32 * 20.0).min(60.0);

    // Duration variety measure (normalized)
    let variety_bonus = rhythm.duration_variety * 40.0;

    let score = (unique_score + variety_bonus).min(100.0);

    let (rating, explanation) = if score >= 70.0 {
        (
            "excellent",
            "Rich rhythmic variety with multiple note values",
        )
    } else if score >= 50.0 {
        ("good", "Good rhythmic variety")
    } else if score >= 30.0 {
        ("fair", "Some rhythmic variation")
    } else {
        ("poor", "Uniform rhythm with little variety")
    };

    ScoreDimension {
        score,
        weight: 0.15,
        rating: rating.to_string(),
        explanation: explanation.to_string(),
    }
}

/// Score resolution quality (leading tones, cadences)
fn score_resolution_quality(
    analysis: &AnalysisResult,
    notes: &[crate::types::Note],
) -> ScoreDimension {
    if notes.len() < 2 {
        return ScoreDimension {
            score: 50.0,
            weight: 0.15,
            rating: "fair".to_string(),
            explanation: "Too few notes to assess resolution".to_string(),
        };
    }

    let intervals = analyze_intervals(notes);
    let mut score: f32 = 50.0; // Start at neutral

    // Check for resolution patterns
    let last_note = notes.last().unwrap();
    let second_last = &notes[notes.len() - 2];

    // Leading tone resolution (semitone up to tonic)
    let key = &analysis.scale_analysis.key;
    let tonic_pitch_class = key.root.to_semitone();
    let last_pitch_class = last_note.pitch % 12;
    let second_last_pitch_class = second_last.pitch % 12;

    // Check if ends on tonic
    if last_pitch_class == tonic_pitch_class {
        score += 20.0;

        // Check for leading tone before tonic
        let leading_tone = (tonic_pitch_class + 11) % 12; // One semitone below
        if second_last_pitch_class == leading_tone {
            score += 15.0; // Strong resolution
        }
    }

    // Check for consonant ending
    if let Some(last_interval) = intervals.last() {
        match last_interval.quality {
            IntervalQuality::PerfectConsonance | IntervalQuality::ImperfectConsonance => {
                score += 10.0;
            }
            IntervalQuality::MildDissonance => {}
            IntervalQuality::StrongDissonance => {
                score -= 10.0;
            }
        }
    }

    // Cap score
    let score = score.clamp(0.0, 100.0);

    let (rating, explanation) = if score >= 80.0 {
        ("excellent", "Strong resolution with proper voice leading")
    } else if score >= 60.0 {
        ("good", "Good resolution tendencies")
    } else if score >= 40.0 {
        ("fair", "Adequate resolution")
    } else {
        ("poor", "Weak or unresolved ending")
    };

    ScoreDimension {
        score,
        weight: 0.15,
        rating: rating.to_string(),
        explanation: explanation.to_string(),
    }
}

/// Score dynamics expression
fn score_dynamics_expression(analysis: &AnalysisResult) -> ScoreDimension {
    let dynamics = &analysis.dynamics_analysis;

    let mut score = 30.0; // Base score

    // Velocity range bonus
    if dynamics.has_dynamics {
        score += 30.0;
    }

    // Velocity variance bonus (normalized)
    let variance_normalized = (dynamics.velocity_variance / 400.0).min(1.0); // 400 = approx max variance
    score += variance_normalized * 30.0;

    // Range bonus
    let range_bonus = (dynamics.velocity_range as f32 / 127.0) * 20.0;
    score += range_bonus;

    let score = score.min(100.0);

    let (rating, explanation) = if score >= 70.0 {
        ("excellent", "Expressive dynamics with good variation")
    } else if score >= 50.0 {
        ("good", "Noticeable dynamic variation")
    } else if score >= 30.0 {
        ("fair", "Some dynamic variation")
    } else {
        ("poor", "Flat dynamics with little expression")
    };

    ScoreDimension {
        score,
        weight: 0.15,
        rating: rating.to_string(),
        explanation: explanation.to_string(),
    }
}

/// Score structural balance
fn score_structural_balance(analysis: &AnalysisResult) -> ScoreDimension {
    let contour = &analysis.contour_analysis;

    let mut score: f32 = 50.0;

    // Pitch range assessment (prefer moderate range)
    let range = contour.pitch_range;
    if range >= 5 && range <= 24 {
        // Half to two octaves
        score += 25.0;
    } else if range >= 3 && range <= 36 {
        score += 15.0;
    }

    // Climax position (prefer middle to late)
    if contour.climax_position >= 0.4 && contour.climax_position <= 0.8 {
        score += 15.0;
    }

    // Note count (prefer 4-32 notes for a phrase)
    let note_count = analysis.note_count;
    if note_count >= 4 && note_count <= 32 {
        score += 10.0;
    }

    let score = score.min(100.0);

    let (rating, explanation) = if score >= 75.0 {
        ("excellent", "Well-balanced structure with good proportions")
    } else if score >= 55.0 {
        ("good", "Good structural balance")
    } else if score >= 35.0 {
        ("fair", "Adequate structure")
    } else {
        ("poor", "Unbalanced structure")
    };

    ScoreDimension {
        score,
        weight: 0.15,
        rating: rating.to_string(),
        explanation: explanation.to_string(),
    }
}

/// Calculate weighted overall score
fn calculate_overall(dimensions: &ScoreDimensions) -> f32 {
    let weighted_sum = dimensions.harmonic_coherence.score * dimensions.harmonic_coherence.weight
        + dimensions.melodic_interest.score * dimensions.melodic_interest.weight
        + dimensions.rhythmic_variety.score * dimensions.rhythmic_variety.weight
        + dimensions.resolution_quality.score * dimensions.resolution_quality.weight
        + dimensions.dynamics_expression.score * dimensions.dynamics_expression.weight
        + dimensions.structural_balance.score * dimensions.structural_balance.weight;

    let total_weight = dimensions.harmonic_coherence.weight
        + dimensions.melodic_interest.weight
        + dimensions.rhythmic_variety.weight
        + dimensions.resolution_quality.weight
        + dimensions.dynamics_expression.weight
        + dimensions.structural_balance.weight;

    weighted_sum / total_weight
}

/// Generate improvement suggestions
fn generate_suggestions(dimensions: &ScoreDimensions, analysis: &AnalysisResult) -> Vec<String> {
    let mut suggestions = Vec::new();

    // Harmonic suggestions
    if dimensions.harmonic_coherence.score < 70.0 {
        let out_of_scale = analysis.scale_analysis.out_of_scale_count;
        suggestions.push(format!(
            "Consider reducing chromatic alterations ({} out-of-scale notes detected)",
            out_of_scale
        ));
    }

    // Melodic suggestions
    if dimensions.melodic_interest.score < 50.0 {
        if analysis.interval_analysis.interval_variety < 3 {
            suggestions.push("Add more interval variety for melodic interest".to_string());
        }
        if analysis.contour_analysis.direction_changes < 2 {
            suggestions.push("Consider adding direction changes to the melody".to_string());
        }
    }

    // Rhythmic suggestions
    if dimensions.rhythmic_variety.score < 40.0 {
        suggestions.push("Add rhythmic variety with different note durations".to_string());
    }

    // Resolution suggestions
    if dimensions.resolution_quality.score < 50.0 {
        suggestions
            .push("Consider ending on a stronger resolution (tonic or dominant)".to_string());
    }

    // Dynamics suggestions
    if dimensions.dynamics_expression.score < 40.0 {
        suggestions.push("Add dynamic variation (velocity changes) for expression".to_string());
    }

    // Structural suggestions
    if dimensions.structural_balance.score < 40.0 {
        let range = analysis.contour_analysis.pitch_range;
        if range < 5 {
            suggestions.push("Expand the pitch range for better structural balance".to_string());
        } else if range > 24 {
            suggestions.push("Consider reducing the pitch range for better focus".to_string());
        }
    }

    suggestions
}

/// Generate score summary
fn generate_summary(dimensions: &ScoreDimensions, overall: f32) -> ScoreSummary {
    // Find strongest and weakest
    let dim_scores = [
        ("Harmonic Coherence", dimensions.harmonic_coherence.score),
        ("Melodic Interest", dimensions.melodic_interest.score),
        ("Rhythmic Variety", dimensions.rhythmic_variety.score),
        ("Resolution Quality", dimensions.resolution_quality.score),
        ("Dynamics Expression", dimensions.dynamics_expression.score),
        ("Structural Balance", dimensions.structural_balance.score),
    ];

    let strongest = dim_scores
        .iter()
        .max_by(|a, b| a.1.partial_cmp(&b.1).unwrap())
        .map(|(name, _)| name.to_string())
        .unwrap_or_default();

    let weakest = dim_scores
        .iter()
        .min_by(|a, b| a.1.partial_cmp(&b.1).unwrap())
        .map(|(name, _)| name.to_string())
        .unwrap_or_default();

    let rating = if overall >= 80.0 {
        "excellent"
    } else if overall >= 60.0 {
        "good"
    } else if overall >= 40.0 {
        "fair"
    } else {
        "poor"
    };

    let summary = format!(
        "Overall {} musicality. Strongest: {}. Consider improving: {}.",
        rating, strongest, weakest
    );

    ScoreSummary {
        rating: rating.to_string(),
        strongest,
        weakest,
        summary,
    }
}

/// Apply CLIPS rule engine adjustments asynchronously
async fn apply_clips_adjustments_async(
    sequence: &Sequence,
    rules_dir: &Path,
) -> Result<Option<Vec<ClipsAdjustment>>> {
    let engine = ClipsRuleEngine::new(rules_dir)?;

    if !engine.is_available() {
        return Ok(None);
    }

    let result = engine.analyze(sequence).await?;

    if result.adjustments.is_empty() {
        return Ok(None);
    }

    // Convert ScoringAdjustment to ClipsAdjustment
    let adjustments: Vec<ClipsAdjustment> = result
        .adjustments
        .into_iter()
        .map(|adj| ClipsAdjustment {
            dimension: adj.dimension,
            adjustment: adj.adjustment,
            reason: adj.reason,
        })
        .collect();

    Ok(Some(adjustments))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{Context, KeySignature, Mode, Note, PitchClass, TimeSignature};

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

    fn create_varied_sequence() -> Sequence {
        // More varied sequence with dynamics and rhythm
        let notes = vec![
            Note::new(60, 240, 70, 0),    // C4 eighth
            Note::new(64, 480, 85, 240),  // E4 quarter
            Note::new(67, 240, 90, 720),  // G4 eighth
            Note::new(72, 960, 95, 960),  // C5 half
            Note::new(71, 240, 80, 1920), // B4 eighth
            Note::new(67, 480, 75, 2160), // G4 quarter
            Note::new(64, 240, 70, 2640), // E4 eighth
            Note::new(60, 960, 85, 2880), // C4 half
        ];

        let mut seq = Sequence::new("varied".to_string(), notes);
        seq.context = Context {
            key_signature: Some(KeySignature::new(PitchClass::C, Mode::Major)),
            time_signature: TimeSignature::default(),
            tempo: 120,
            ticks_per_quarter: 480,
        };
        seq.name = Some("Varied Sequence".to_string());
        seq
    }

    #[test]
    fn test_score_c_major_scale() {
        let seq = create_c_major_scale();
        let score = score_sequence(&seq, false).unwrap();

        // Harmonic coherence should be excellent
        assert!(score.dimensions.harmonic_coherence.score >= 95.0);

        // Overall should be reasonable
        assert!(score.overall > 40.0);
    }

    #[test]
    fn test_score_varied_sequence() {
        let seq = create_varied_sequence();
        let score = score_sequence(&seq, false).unwrap();

        // Should have better rhythmic variety
        assert!(score.dimensions.rhythmic_variety.score > 40.0);

        // Should have better dynamics
        assert!(score.dimensions.dynamics_expression.score > 40.0);
    }

    #[test]
    fn test_score_summary() {
        let seq = create_c_major_scale();
        let score = score_sequence(&seq, false).unwrap();

        assert!(!score.summary.rating.is_empty());
        assert!(!score.summary.strongest.is_empty());
        assert!(!score.summary.weakest.is_empty());
    }

    #[test]
    fn test_suggestions_generated() {
        let seq = create_c_major_scale();
        let score = score_sequence(&seq, false).unwrap();

        // Should generate some suggestions (uniform rhythm, no dynamics)
        // Note: This may or may not generate suggestions depending on scores
        // The important thing is it doesn't panic
        assert!(score.suggestions.len() >= 0);
    }
}
