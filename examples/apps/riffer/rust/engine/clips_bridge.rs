//! CLIPS Rule Engine Bridge for Riffer
//!
//! Provides integration with the CLIPS expert system for:
//! - Scoring adjustments (bonuses/penalties based on music theory rules)
//! - Context-aware suggestions for improvement
//!
//! Uses nxuskit's NxuskitProvider with provider_type "clips" for rule execution.
//!
//! Note: This module uses NxuskitProvider for CLIPS integration.
//! Currently unused but available for future activation.

use nxuskit_examples_clips_wire::{ClipsInputWire, ClipsOutputWire};
use serde::{Deserialize, Serialize};
use std::path::Path;

use crate::errors::Result;
#[allow(unused_imports)]
use crate::errors::RifferError;
use crate::theory::intervals::{IntervalInfo, analyze_intervals};
use crate::theory::scales::is_in_scale;
use crate::types::{Context, IntervalQuality, Mode, Note, Sequence};

/// Scoring adjustment from CLIPS rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoringAdjustment {
    /// Which dimension this adjustment applies to
    pub dimension: String,
    /// Adjustment value (positive = bonus, negative = penalty)
    pub adjustment: f32,
    /// Reason for the adjustment
    pub reason: String,
    /// Rule that triggered this adjustment
    pub rule_name: String,
}

/// Suggestion from CLIPS rules
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ClipsSuggestion {
    /// Suggestion category (harmony, melody, rhythm, dynamics, resolution, structure)
    pub category: String,
    /// Severity level (info, suggestion, warning)
    pub severity: String,
    /// The suggestion text
    pub message: String,
    /// Rule that generated this suggestion
    pub rule_name: String,
    /// Optional note indices this applies to
    pub note_indices: Option<Vec<usize>>,
}

/// Result from CLIPS rule execution
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ClipsResult {
    /// Scoring adjustments (bonuses/penalties)
    pub adjustments: Vec<ScoringAdjustment>,
    /// Improvement suggestions
    pub suggestions: Vec<ClipsSuggestion>,
    /// Number of rules that fired
    pub rules_fired: u32,
    /// Execution time in milliseconds
    pub execution_time_ms: u64,
}

/// CLIPS Rule Engine for music analysis
pub struct ClipsRuleEngine {
    /// Path to the rules directory
    #[allow(dead_code)]
    rules_dir: std::path::PathBuf,
    /// Whether CLIPS is available
    available: bool,
}

impl ClipsRuleEngine {
    /// Create a new CLIPS rule engine
    ///
    /// # Arguments
    /// * `rules_dir` - Path to directory containing .clp rule files
    pub fn new(rules_dir: &Path) -> Result<Self> {
        let rules_dir = rules_dir.to_path_buf();

        // Check if rules directory exists
        let available = rules_dir.exists() && rules_dir.is_dir();

        if !available {
            eprintln!(
                "Warning: CLIPS rules directory not found at {}. Using deterministic scoring only.",
                rules_dir.display()
            );
        }

        Ok(Self {
            rules_dir,
            available,
        })
    }

    /// Check if CLIPS is available
    pub fn is_available(&self) -> bool {
        self.available
    }

    /// Convert a sequence to CLIPS provider-chat input (`ClipsInput`-shaped JSON).
    pub fn sequence_to_facts(
        &self,
        sequence: &Sequence,
    ) -> std::result::Result<ClipsInputWire, serde_json::Error> {
        let mut facts: Vec<serde_json::Value> = Vec::new();

        // Add context fact
        let context_fact = self.context_to_fact(&sequence.context);
        facts.push(context_fact);

        // Add note facts
        for (idx, note) in sequence.notes.iter().enumerate() {
            let note_fact = self.note_to_fact(note, idx);
            facts.push(note_fact);
        }

        // Add interval facts
        let intervals = analyze_intervals(&sequence.notes);
        for interval in &intervals {
            let interval_fact = self.interval_to_fact(interval);
            facts.push(interval_fact);
        }

        // Add scale membership facts
        if let Some(key) = &sequence.context.key_signature {
            for (idx, note) in sequence.notes.iter().enumerate() {
                let pc = note.pitch_class();
                let in_scale = is_in_scale(pc, key);
                let membership_fact = serde_json::json!({
                    "template": "scale-membership",
                    "values": {
                        "note-index": idx,
                        "pitch-class": {"symbol": pc.to_string()},
                        "in-scale": {"symbol": if in_scale { "yes" } else { "no" }}
                    }
                });
                facts.push(membership_fact);
            }
        }

        // Add rhythmic-entropy fact
        let unique_durations: std::collections::HashSet<_> =
            sequence.notes.iter().map(|n| n.duration).collect();
        let rhythmic_variety = if sequence.notes.is_empty() {
            0.0
        } else {
            (unique_durations.len() as f32 / sequence.notes.len() as f32).min(1.0)
        };
        let rhythmic_entropy_fact = serde_json::json!({
            "template": "rhythmic-entropy",
            "values": {
                "value": rhythmic_variety,
                "unique-durations": unique_durations.len(),
                "total-notes": sequence.notes.len()
            }
        });
        facts.push(rhythmic_entropy_fact);

        // Add dynamics-summary fact
        if !sequence.notes.is_empty() {
            let velocities: Vec<u8> = sequence.notes.iter().map(|n| n.velocity).collect();
            let min_vel = *velocities.iter().min().unwrap_or(&0);
            let max_vel = *velocities.iter().max().unwrap_or(&127);
            let velocity_range = max_vel.saturating_sub(min_vel);

            // Calculate standard deviation
            let mean = velocities.iter().map(|&v| v as f32).sum::<f32>() / velocities.len() as f32;
            let variance = velocities
                .iter()
                .map(|&v| (v as f32 - mean).powi(2))
                .sum::<f32>()
                / velocities.len() as f32;
            let velocity_std = variance.sqrt();

            let dynamics_fact = serde_json::json!({
                "template": "dynamics-summary",
                "values": {
                    "min-velocity": min_vel,
                    "max-velocity": max_vel,
                    "velocity-range": velocity_range,
                    "velocity-std": velocity_std
                }
            });
            facts.push(dynamics_fact);
        }

        // Add contour fact
        if !intervals.is_empty() {
            let mut direction_changes = 0;
            let mut ascending_count = 0;
            let mut descending_count = 0;
            let mut last_direction: Option<bool> = None;

            for interval in &intervals {
                let is_ascending = interval.semitones > 0;
                let is_descending = interval.semitones < 0;

                if is_ascending {
                    ascending_count += 1;
                } else if is_descending {
                    descending_count += 1;
                }

                if let Some(was_ascending) = last_direction {
                    if (was_ascending && is_descending) || (!was_ascending && is_ascending) {
                        direction_changes += 1;
                    }
                }
                if is_ascending || is_descending {
                    last_direction = Some(is_ascending);
                }
            }

            let contour_type = if direction_changes == 0 && ascending_count > descending_count {
                "ascending"
            } else if direction_changes == 0 && descending_count > ascending_count {
                "descending"
            } else if ascending_count == descending_count {
                "static"
            } else if direction_changes >= 2 {
                "wave"
            } else if ascending_count > descending_count {
                "arch"
            } else {
                "inverse-arch"
            };

            let contour_fact = serde_json::json!({
                "template": "contour",
                "values": {
                    "type": {"symbol": contour_type},
                    "direction-changes": direction_changes
                }
            });
            facts.push(contour_fact);
        }

        serde_json::from_value(serde_json::json!({
            "facts": facts,
            "config": {
                "include_trace": true,
                "max_rules": 100
            }
        }))
    }

    /// Convert context to a CLIPS fact
    fn context_to_fact(&self, context: &Context) -> serde_json::Value {
        let detected_key = context
            .key_signature
            .as_ref()
            .map(|k| k.root.to_string())
            .unwrap_or_else(|| "C".to_string());

        let detected_mode = context
            .key_signature
            .as_ref()
            .map(|k| match k.mode {
                Mode::Major => "major",
                Mode::Minor => "minor",
                Mode::Dorian => "dorian",
                Mode::Phrygian => "phrygian",
                Mode::Lydian => "lydian",
                Mode::Mixolydian => "mixolydian",
                Mode::Aeolian => "aeolian",
                Mode::Locrian => "locrian",
            })
            .unwrap_or("major");

        serde_json::json!({
            "template": "context",
            "values": {
                "detected-key": detected_key,
                "detected-mode": {"symbol": detected_mode},
                "confidence": 0.8,
                "tempo": context.tempo,
                "ticks-per-quarter": context.ticks_per_quarter
            }
        })
    }

    /// Convert a note to a CLIPS fact
    fn note_to_fact(&self, note: &Note, index: usize) -> serde_json::Value {
        let pc = note.pitch_class();
        serde_json::json!({
            "template": "note",
            "values": {
                "index": index,
                "pitch": note.pitch,
                "pitch-class": {"symbol": pc.to_string()},
                "octave": note.octave(),
                "duration": note.duration,
                "velocity": note.velocity,
                "start-tick": note.start_tick
            }
        })
    }

    /// Convert an interval to a CLIPS fact
    fn interval_to_fact(&self, interval: &IntervalInfo) -> serde_json::Value {
        let quality = match interval.quality {
            IntervalQuality::PerfectConsonance => "perfect-consonance",
            IntervalQuality::ImperfectConsonance => "imperfect-consonance",
            IntervalQuality::MildDissonance => "mild-dissonance",
            IntervalQuality::StrongDissonance => "strong-dissonance",
        };

        // Generate a human-readable interval name
        let name = crate::theory::intervals::interval_name(interval.semitones as i8);

        serde_json::json!({
            "template": "interval",
            "values": {
                "from-index": interval.from_index,
                "to-index": interval.to_index,
                "semitones": interval.semitones,
                "name": name,
                "quality": {"symbol": quality}
            }
        })
    }

    /// Extract scoring adjustments from CLIPS output
    pub fn extract_adjustments(&self, output: &serde_json::Value) -> Vec<ScoringAdjustment> {
        let mut adjustments = Vec::new();

        if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
            for conclusion in conclusions {
                if conclusion.get("template").and_then(|t| t.as_str()) == Some("scoring-adjustment")
                {
                    if let Some(values) = conclusion.get("values") {
                        // Handle dimension which might be a symbol (object with "symbol" key) or a string
                        let dimension = values
                            .get("dimension")
                            .map(|d| {
                                // Try as object with symbol key first
                                if let Some(sym) = d.get("symbol").and_then(|s| s.as_str()) {
                                    sym.to_string()
                                } else if let Some(s) = d.as_str() {
                                    s.to_string()
                                } else {
                                    "unknown".to_string()
                                }
                            })
                            .unwrap_or_else(|| "unknown".to_string());

                        // Handle amount (the template uses 'amount', not 'adjustment')
                        let amount = values
                            .get("amount")
                            .and_then(|a| a.as_f64())
                            .or_else(|| values.get("adjustment").and_then(|a| a.as_f64()))
                            .unwrap_or(0.0) as f32;

                        let adjustment = ScoringAdjustment {
                            dimension,
                            adjustment: amount,
                            reason: values
                                .get("reason")
                                .and_then(|r| r.as_str())
                                .unwrap_or("")
                                .to_string(),
                            rule_name: values
                                .get("rule-name")
                                .and_then(|r| r.as_str())
                                .unwrap_or("unknown")
                                .to_string(),
                        };
                        adjustments.push(adjustment);
                    }
                }
            }
        }

        adjustments
    }

    /// Extract suggestions from CLIPS output
    pub fn extract_suggestions(&self, output: &serde_json::Value) -> Vec<ClipsSuggestion> {
        let mut suggestions = Vec::new();

        if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
            for conclusion in conclusions {
                if conclusion.get("template").and_then(|t| t.as_str()) == Some("suggestion") {
                    if let Some(values) = conclusion.get("values") {
                        let suggestion = ClipsSuggestion {
                            category: values
                                .get("category")
                                .and_then(|c| c.as_str())
                                .unwrap_or("general")
                                .to_string(),
                            severity: values
                                .get("severity")
                                .and_then(|s| s.as_str())
                                .unwrap_or("info")
                                .to_string(),
                            message: values
                                .get("message")
                                .and_then(|m| m.as_str())
                                .unwrap_or("")
                                .to_string(),
                            rule_name: values
                                .get("rule-name")
                                .and_then(|r| r.as_str())
                                .unwrap_or("unknown")
                                .to_string(),
                            note_indices: values
                                .get("note-indices")
                                .and_then(|n| n.as_array())
                                .map(|arr| {
                                    arr.iter()
                                        .filter_map(|v| v.as_u64().map(|n| n as usize))
                                        .collect()
                                }),
                        };
                        suggestions.push(suggestion);
                    }
                }
            }
        }

        suggestions
    }

    /// Run CLIPS rules on a sequence and return results
    ///
    /// This is the main entry point for CLIPS integration.
    /// If CLIPS is not available, returns an empty result.
    pub async fn analyze(&self, sequence: &Sequence) -> Result<ClipsResult> {
        use nxuskit::{ChatRequest, Message, NxuskitProvider, ProviderConfig};

        if !self.available {
            return Ok(ClipsResult::default());
        }

        // Build provider via NxuskitProvider with clips provider type
        let provider = NxuskitProvider::new(ProviderConfig {
            provider_type: "clips".to_string(),
            model: Some(self.rules_dir.to_str().unwrap_or(".").to_string()),
            ..Default::default()
        })
        .map_err(|e| RifferError::ClipsError(e.to_string()))?;

        // Convert sequence to ClipsInput-shaped wire (shared crate mirrors engine JSON).
        let clips_input = self
            .sequence_to_facts(sequence)
            .map_err(|e| RifferError::ClipsError(format!("CLIPS input wire: {e}")))?;

        let input_json = serde_json::to_string(&clips_input)
            .map_err(|e| RifferError::ClipsError(format!("CLIPS input JSON: {e}")))?;

        // Create request with all rule files
        let rule_files = "templates.clp,music-theory.clp,scoring-adjustments.clp,suggestions.clp";
        let request = ChatRequest::new(rule_files).with_message(Message::user(input_json));

        // Execute rules
        let start = std::time::Instant::now();
        let response = provider
            .chat(request)
            .map_err(|e| RifferError::ClipsError(e.to_string()))?;
        let execution_time_ms = start.elapsed().as_millis() as u64;

        // Parse output
        let clips_output: ClipsOutputWire = serde_json::from_str(&response.content)
            .map_err(|e| RifferError::ClipsError(format!("Failed to parse CLIPS output: {}", e)))?;

        let output_val = serde_json::to_value(&clips_output)
            .map_err(|e| RifferError::ClipsError(format!("CLIPS output re-encode: {e}")))?;

        // Extract results (reuse Value-based helpers)
        let adjustments = self.extract_adjustments(&output_val);
        let suggestions = self.extract_suggestions(&output_val);
        let rules_fired = clips_output.stats.total_rules_fired as u32;

        Ok(ClipsResult {
            adjustments,
            suggestions,
            rules_fired,
            execution_time_ms,
        })
    }
}

/// Apply scoring adjustments to dimension scores
pub fn apply_adjustments(
    scores: &mut std::collections::HashMap<String, f32>,
    adjustments: &[ScoringAdjustment],
) {
    for adj in adjustments {
        if let Some(score) = scores.get_mut(&adj.dimension) {
            *score = (*score + adj.adjustment).clamp(0.0, 100.0);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_note_to_fact() {
        let engine = ClipsRuleEngine {
            rules_dir: std::path::PathBuf::from("."),
            available: true,
        };

        let note = Note::new(60, 480, 80, 0); // Middle C
        let fact = engine.note_to_fact(&note, 0);

        assert_eq!(fact["template"], "note");
        assert_eq!(fact["values"]["pitch"], 60);
        assert_eq!(fact["values"]["index"], 0);
    }

    #[test]
    fn test_extract_adjustments() {
        let engine = ClipsRuleEngine {
            rules_dir: std::path::PathBuf::from("."),
            available: true,
        };

        let output = serde_json::json!({
            "conclusions": [
                {
                    "template": "scoring-adjustment",
                    "values": {
                        "dimension": "harmonic-coherence",
                        "adjustment": 5.0,
                        "reason": "Good use of leading tone",
                        "rule-name": "leading-tone-bonus"
                    }
                }
            ]
        });

        let adjustments = engine.extract_adjustments(&output);
        assert_eq!(adjustments.len(), 1);
        assert_eq!(adjustments[0].dimension, "harmonic-coherence");
        assert_eq!(adjustments[0].adjustment, 5.0);
    }

    #[test]
    fn test_extract_suggestions() {
        let engine = ClipsRuleEngine {
            rules_dir: std::path::PathBuf::from("."),
            available: true,
        };

        let output = serde_json::json!({
            "conclusions": [
                {
                    "template": "suggestion",
                    "values": {
                        "category": "resolution",
                        "severity": "warning",
                        "message": "Unresolved tritone at measure 4",
                        "rule-name": "unresolved-tritone"
                    }
                }
            ]
        });

        let suggestions = engine.extract_suggestions(&output);
        assert_eq!(suggestions.len(), 1);
        assert_eq!(suggestions[0].category, "resolution");
        assert_eq!(suggestions[0].severity, "warning");
    }
}
