//! Solver pattern types for auto-retry LLM with CLIPS validation.
//!
//! The Solver pattern uses CLIPS rules to evaluate LLM output quality and
//! automatically retry with adjusted parameters when validation fails.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Configuration for a solver instance.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SolverConfig {
    /// Maximum retry attempts (default: 3)
    #[serde(default = "default_max_retries")]
    pub max_retries: u32,

    /// Failure-to-adjustment mappings
    #[serde(default)]
    pub strategies: Vec<FailureStrategy>,

    /// Path to CLIPS rules file or inline rules
    pub evaluation_rules: String,

    /// Type of LLM output expected
    pub conclusion_type: ConclusionType,

    /// Minimum confidence for valid result (default: 0.7)
    #[serde(default = "default_confidence_threshold")]
    pub confidence_threshold: f64,

    /// Total timeout for all retries in milliseconds (default: 30000)
    #[serde(default = "default_timeout_ms")]
    pub timeout_ms: u64,

    /// Valid categories for classification type
    #[serde(default)]
    pub valid_categories: Vec<String>,
}

fn default_max_retries() -> u32 {
    3
}

fn default_confidence_threshold() -> f64 {
    0.7
}

fn default_timeout_ms() -> u64 {
    30000
}

impl Default for SolverConfig {
    fn default() -> Self {
        Self {
            max_retries: default_max_retries(),
            strategies: Vec::new(),
            evaluation_rules: String::new(),
            conclusion_type: ConclusionType::Classification,
            confidence_threshold: default_confidence_threshold(),
            timeout_ms: default_timeout_ms(),
            valid_categories: Vec::new(),
        }
    }
}

/// Type of LLM conclusion expected.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ConclusionType {
    /// LLM categorizes input into predefined categories
    Classification,
    /// LLM extracts structured fields from input
    Extraction,
    /// LLM performs logical reasoning with chain-of-thought
    Reasoning,
}

/// Type of validation failure detected by CLIPS.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum FailureType {
    /// Confidence below threshold
    LowConfidence,
    /// Category not in allowed set
    InvalidCategory,
    /// Empty or null reasoning
    MissingReasoning,
    /// Required fields missing (extraction)
    IncompleteExtraction,
    /// Cross-field validation failed
    InconsistentData,
    /// Cannot parse LLM output
    ParseError,
}

impl FailureType {
    /// Returns a human-readable description of this failure type.
    pub fn description(&self) -> &'static str {
        match self {
            FailureType::LowConfidence => "Confidence below threshold",
            FailureType::InvalidCategory => "Category not in allowed set",
            FailureType::MissingReasoning => "Empty or missing reasoning",
            FailureType::IncompleteExtraction => "Required fields missing",
            FailureType::InconsistentData => "Cross-field validation failed",
            FailureType::ParseError => "Cannot parse LLM output",
        }
    }
}

/// Mapping from failure type to parameter adjustments.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FailureStrategy {
    /// The failure condition this strategy handles
    pub failure_type: FailureType,

    /// Parameter changes to apply on this failure
    pub adjustments: Vec<KnobAdjustment>,
}

/// Single parameter adjustment specification.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct KnobAdjustment {
    /// Parameter name (temperature, top_p, etc.)
    pub knob: String,

    /// How to modify the value
    pub action: AdjustAction,

    /// Value for set/delta actions
    #[serde(default)]
    pub value: Option<f64>,

    /// Minimum allowed value (default: 0.0)
    #[serde(default)]
    pub min: f64,

    /// Maximum allowed value (default: 2.0)
    #[serde(default = "default_max_knob")]
    pub max: f64,
}

fn default_max_knob() -> f64 {
    2.0
}

/// Action to take when adjusting a knob.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum AdjustAction {
    /// Set to specific value
    Set,
    /// Add/subtract from current value
    Delta,
    /// Set boolean to true
    Enable,
    /// Set boolean to false
    Disable,
}

/// Result from CLIPS validation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EvaluationResult {
    /// Evaluation status
    pub status: EvalStatus,

    /// Type of failure (if status != Valid)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub failure_type: Option<FailureType>,

    /// CLIPS-suggested adjustment
    #[serde(skip_serializing_if = "Option::is_none")]
    pub suggested_adjustment: Option<String>,

    /// Extracted confidence value
    #[serde(skip_serializing_if = "Option::is_none")]
    pub confidence: Option<f64>,

    /// Additional evaluation metadata
    #[serde(default, skip_serializing_if = "HashMap::is_empty")]
    pub details: HashMap<String, serde_json::Value>,
}

/// Evaluation status from CLIPS validation.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum EvalStatus {
    /// Output passed all validation rules
    Valid,
    /// Output failed validation, no retry suggested
    Invalid,
    /// Output failed validation, retry recommended
    Retry,
}

/// Record of a single retry attempt.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RetryAttempt {
    /// 1-indexed attempt number
    pub attempt_number: u32,

    /// LLM parameters used for this attempt
    pub parameters: HashMap<String, serde_json::Value>,

    /// Raw LLM output
    pub llm_response: String,

    /// CLIPS evaluation result
    pub evaluation: EvaluationResult,

    /// Time taken for this attempt in milliseconds
    pub duration_ms: u64,

    /// Tokens used for this attempt
    #[serde(default)]
    pub tokens_used: u64,
}

/// Final result from solver execution.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SolverResult {
    /// Whether validation ultimately passed
    pub success: bool,

    /// Parsed output from best attempt
    pub final_output: serde_json::Value,

    /// The best attempt (by score)
    pub best_attempt: RetryAttempt,

    /// All attempts in order
    pub retry_history: Vec<RetryAttempt>,

    /// Total execution time in milliseconds
    pub total_duration_ms: u64,

    /// Total tokens consumed across all attempts
    pub total_tokens: u64,
}

/// Default failure strategies based on research.md design decisions.
pub fn default_strategies() -> Vec<FailureStrategy> {
    vec![
        FailureStrategy {
            failure_type: FailureType::LowConfidence,
            adjustments: vec![KnobAdjustment {
                knob: "temperature".to_string(),
                action: AdjustAction::Delta,
                value: Some(0.2),
                min: 0.0,
                max: 2.0,
            }],
        },
        FailureStrategy {
            failure_type: FailureType::InvalidCategory,
            adjustments: vec![KnobAdjustment {
                knob: "temperature".to_string(),
                action: AdjustAction::Delta,
                value: Some(-0.2),
                min: 0.0,
                max: 2.0,
            }],
        },
        FailureStrategy {
            failure_type: FailureType::MissingReasoning,
            adjustments: vec![KnobAdjustment {
                knob: "thinking_enabled".to_string(),
                action: AdjustAction::Enable,
                value: None,
                min: 0.0,
                max: 1.0,
            }],
        },
        FailureStrategy {
            failure_type: FailureType::IncompleteExtraction,
            adjustments: vec![
                KnobAdjustment {
                    knob: "temperature".to_string(),
                    action: AdjustAction::Delta,
                    value: Some(-0.1),
                    min: 0.0,
                    max: 2.0,
                },
                KnobAdjustment {
                    knob: "max_tokens".to_string(),
                    action: AdjustAction::Delta,
                    value: Some(500.0),
                    min: 100.0,
                    max: 8000.0,
                },
            ],
        },
        FailureStrategy {
            failure_type: FailureType::InconsistentData,
            adjustments: vec![KnobAdjustment {
                knob: "thinking_enabled".to_string(),
                action: AdjustAction::Enable,
                value: None,
                min: 0.0,
                max: 1.0,
            }],
        },
        FailureStrategy {
            failure_type: FailureType::ParseError,
            adjustments: vec![KnobAdjustment {
                knob: "temperature".to_string(),
                action: AdjustAction::Set,
                value: Some(0.0),
                min: 0.0,
                max: 2.0,
            }],
        },
    ]
}

// ============================================================================
// Solver Implementation Functions
// ============================================================================

/// Detects the failure type from CLIPS evaluation output.
///
/// Parses the CLIPS evaluation JSON and extracts the failure type.
pub fn detect_failure_type(eval_output: &str) -> Option<FailureType> {
    let parsed: serde_json::Value = serde_json::from_str(eval_output).ok()?;

    let status = parsed.get("status")?.as_str()?;
    if status == "valid" {
        return None;
    }

    let failure_str = parsed.get("failure_type")?.as_str()?;
    match failure_str {
        "low_confidence" => Some(FailureType::LowConfidence),
        "invalid_category" => Some(FailureType::InvalidCategory),
        "missing_reasoning" => Some(FailureType::MissingReasoning),
        "incomplete_extraction" => Some(FailureType::IncompleteExtraction),
        "inconsistent_data" => Some(FailureType::InconsistentData),
        "parse_error" => Some(FailureType::ParseError),
        _ => None,
    }
}

/// Parses CLIPS evaluation output into an EvaluationResult.
pub fn parse_evaluation_result(eval_output: &str) -> Result<EvaluationResult, String> {
    let parsed: serde_json::Value =
        serde_json::from_str(eval_output).map_err(|e| format!("JSON parse error: {}", e))?;

    let status_str = parsed
        .get("status")
        .and_then(|s| s.as_str())
        .ok_or("Missing status field")?;

    let status = match status_str {
        "valid" => EvalStatus::Valid,
        "invalid" => EvalStatus::Invalid,
        "retry" => EvalStatus::Retry,
        _ => return Err(format!("Unknown status: {}", status_str)),
    };

    let failure_type = if status != EvalStatus::Valid {
        detect_failure_type(eval_output)
    } else {
        None
    };

    let suggested_adjustment = parsed
        .get("suggested_adjustment")
        .and_then(|s| s.as_str())
        .map(String::from);

    let confidence = parsed.get("confidence").and_then(|c| c.as_f64());

    Ok(EvaluationResult {
        status,
        failure_type,
        suggested_adjustment,
        confidence,
        details: HashMap::new(),
    })
}

/// Applies knob adjustments to current parameters.
///
/// Returns the new parameter values after applying the strategy adjustments.
pub fn apply_adjustments(
    current_params: &mut HashMap<String, serde_json::Value>,
    strategy: &FailureStrategy,
) {
    for adj in &strategy.adjustments {
        let current_val = current_params
            .get(&adj.knob)
            .and_then(|v| v.as_f64())
            .unwrap_or(0.5); // Default for most knobs

        let new_val = match adj.action {
            AdjustAction::Set => adj.value.unwrap_or(current_val),
            AdjustAction::Delta => {
                let delta = adj.value.unwrap_or(0.0);
                (current_val + delta).clamp(adj.min, adj.max)
            }
            AdjustAction::Enable => 1.0,
            AdjustAction::Disable => 0.0,
        };

        current_params.insert(adj.knob.clone(), serde_json::json!(new_val));
    }
}

/// Scores an attempt for best-attempt selection.
///
/// Higher scores are better. Factors:
/// - Confidence value (if available)
/// - Whether it was a valid result
/// - Whether it had reasoning
pub fn score_attempt(attempt: &RetryAttempt) -> f64 {
    let mut score = 0.0;

    // Base score from confidence
    if let Some(conf) = attempt.evaluation.confidence {
        score += conf * 100.0;
    }

    // Bonus for valid status
    if attempt.evaluation.status == EvalStatus::Valid {
        score += 50.0;
    }

    // Penalty for parse errors
    if attempt.evaluation.failure_type == Some(FailureType::ParseError) {
        score -= 100.0;
    }

    score
}

/// Finds the strategy for a given failure type.
pub fn find_strategy_for_failure(
    strategies: &[FailureStrategy],
    failure_type: FailureType,
) -> Option<&FailureStrategy> {
    strategies.iter().find(|s| s.failure_type == failure_type)
}

/// Loads solver config from a JSON file.
pub fn load_config_from_file(path: &str) -> Result<SolverConfig, String> {
    let content =
        std::fs::read_to_string(path).map_err(|e| format!("Failed to read file: {}", e))?;
    serde_json::from_str(&content).map_err(|e| format!("Failed to parse config: {}", e))
}

/// Valid knob names that can be adjusted.
pub const VALID_KNOBS: &[&str] = &[
    "temperature",
    "top_p",
    "top_k",
    "presence_penalty",
    "frequency_penalty",
    "max_tokens",
    "thinking_enabled",
];

/// Checks if a knob name is valid.
pub fn is_valid_knob(knob: &str) -> bool {
    VALID_KNOBS.contains(&knob)
}

/// Validates a solver configuration.
pub fn validate_config(config: &SolverConfig) -> Result<(), String> {
    if config.max_retries < 1 || config.max_retries > 10 {
        return Err("max_retries must be between 1 and 10".to_string());
    }

    if config.confidence_threshold < 0.0 || config.confidence_threshold > 1.0 {
        return Err("confidence_threshold must be between 0.0 and 1.0".to_string());
    }

    if config.evaluation_rules.is_empty() {
        return Err("evaluation_rules must be specified".to_string());
    }

    // Check for duplicate failure types in strategies
    let mut seen = std::collections::HashSet::new();
    for strategy in &config.strategies {
        if !seen.insert(strategy.failure_type) {
            return Err(format!(
                "Duplicate failure type in strategies: {:?}",
                strategy.failure_type
            ));
        }

        // Validate knob names
        for adjustment in &strategy.adjustments {
            if !is_valid_knob(&adjustment.knob) {
                return Err(format!(
                    "Invalid knob name '{}' in strategy for {:?}. Valid knobs: {:?}",
                    adjustment.knob, strategy.failure_type, VALID_KNOBS
                ));
            }
        }
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_solver_config_default() {
        let config = SolverConfig::default();
        assert_eq!(config.max_retries, 3);
        assert_eq!(config.confidence_threshold, 0.7);
        assert_eq!(config.timeout_ms, 30000);
    }

    #[test]
    fn test_failure_type_description() {
        assert_eq!(
            FailureType::LowConfidence.description(),
            "Confidence below threshold"
        );
        assert_eq!(
            FailureType::InvalidCategory.description(),
            "Category not in allowed set"
        );
    }

    #[test]
    fn test_default_strategies_coverage() {
        let strategies = default_strategies();
        assert_eq!(strategies.len(), 6, "Should have 6 default strategies");

        // Verify all failure types are covered
        let covered: std::collections::HashSet<_> =
            strategies.iter().map(|s| s.failure_type).collect();
        assert!(covered.contains(&FailureType::LowConfidence));
        assert!(covered.contains(&FailureType::InvalidCategory));
        assert!(covered.contains(&FailureType::MissingReasoning));
        assert!(covered.contains(&FailureType::IncompleteExtraction));
        assert!(covered.contains(&FailureType::InconsistentData));
        assert!(covered.contains(&FailureType::ParseError));
    }

    #[test]
    fn test_solver_config_json_roundtrip() {
        let config = SolverConfig {
            max_retries: 5,
            strategies: vec![FailureStrategy {
                failure_type: FailureType::LowConfidence,
                adjustments: vec![KnobAdjustment {
                    knob: "temperature".to_string(),
                    action: AdjustAction::Delta,
                    value: Some(0.1),
                    min: 0.0,
                    max: 1.0,
                }],
            }],
            evaluation_rules: "rules/solver/test.clp".to_string(),
            conclusion_type: ConclusionType::Classification,
            confidence_threshold: 0.8,
            timeout_ms: 60000,
            valid_categories: vec!["high".to_string(), "medium".to_string(), "low".to_string()],
        };

        let json = serde_json::to_string(&config).unwrap();
        let parsed: SolverConfig = serde_json::from_str(&json).unwrap();

        assert_eq!(parsed.max_retries, config.max_retries);
        assert_eq!(parsed.confidence_threshold, config.confidence_threshold);
        assert_eq!(parsed.valid_categories.len(), 3);
    }

    #[test]
    fn test_detect_failure_type_low_confidence() {
        let eval_output = r#"{"status":"retry","failure_type":"low_confidence","confidence":0.5}"#;
        assert_eq!(
            detect_failure_type(eval_output),
            Some(FailureType::LowConfidence)
        );
    }

    #[test]
    fn test_detect_failure_type_invalid_category() {
        let eval_output =
            r#"{"status":"retry","failure_type":"invalid_category","confidence":0.9}"#;
        assert_eq!(
            detect_failure_type(eval_output),
            Some(FailureType::InvalidCategory)
        );
    }

    #[test]
    fn test_detect_failure_type_valid() {
        let eval_output = r#"{"status":"valid","failure_type":"none","confidence":0.85}"#;
        assert_eq!(detect_failure_type(eval_output), None);
    }

    #[test]
    fn test_apply_adjustments_delta() {
        let strategy = FailureStrategy {
            failure_type: FailureType::LowConfidence,
            adjustments: vec![KnobAdjustment {
                knob: "temperature".to_string(),
                action: AdjustAction::Delta,
                value: Some(0.2),
                min: 0.0,
                max: 2.0,
            }],
        };

        let mut params = HashMap::new();
        params.insert("temperature".to_string(), serde_json::json!(0.7));

        apply_adjustments(&mut params, &strategy);

        let new_temp = params.get("temperature").unwrap().as_f64().unwrap();
        assert!((new_temp - 0.9).abs() < 0.001);
    }

    #[test]
    fn test_apply_adjustments_set() {
        let strategy = FailureStrategy {
            failure_type: FailureType::ParseError,
            adjustments: vec![KnobAdjustment {
                knob: "temperature".to_string(),
                action: AdjustAction::Set,
                value: Some(0.0),
                min: 0.0,
                max: 2.0,
            }],
        };

        let mut params = HashMap::new();
        params.insert("temperature".to_string(), serde_json::json!(0.7));

        apply_adjustments(&mut params, &strategy);

        let new_temp = params.get("temperature").unwrap().as_f64().unwrap();
        assert!((new_temp - 0.0).abs() < 0.001);
    }

    #[test]
    fn test_apply_adjustments_enable() {
        let strategy = FailureStrategy {
            failure_type: FailureType::MissingReasoning,
            adjustments: vec![KnobAdjustment {
                knob: "thinking_enabled".to_string(),
                action: AdjustAction::Enable,
                value: None,
                min: 0.0,
                max: 1.0,
            }],
        };

        let mut params = HashMap::new();
        params.insert("thinking_enabled".to_string(), serde_json::json!(0.0));

        apply_adjustments(&mut params, &strategy);

        let thinking = params.get("thinking_enabled").unwrap().as_f64().unwrap();
        assert!((thinking - 1.0).abs() < 0.001);
    }

    #[test]
    fn test_apply_adjustments_clamps() {
        let strategy = FailureStrategy {
            failure_type: FailureType::LowConfidence,
            adjustments: vec![KnobAdjustment {
                knob: "temperature".to_string(),
                action: AdjustAction::Delta,
                value: Some(1.0),
                min: 0.0,
                max: 1.5,
            }],
        };

        let mut params = HashMap::new();
        params.insert("temperature".to_string(), serde_json::json!(1.0));

        apply_adjustments(&mut params, &strategy);

        let new_temp = params.get("temperature").unwrap().as_f64().unwrap();
        // Should be clamped to max of 1.5, not 2.0
        assert!((new_temp - 1.5).abs() < 0.001);
    }

    #[test]
    fn test_score_attempt_valid() {
        let attempt = RetryAttempt {
            attempt_number: 1,
            parameters: HashMap::new(),
            llm_response: "test".to_string(),
            evaluation: EvaluationResult {
                status: EvalStatus::Valid,
                failure_type: None,
                suggested_adjustment: None,
                confidence: Some(0.9),
                details: HashMap::new(),
            },
            duration_ms: 1000,
            tokens_used: 100,
        };

        let score = score_attempt(&attempt);
        // 0.9 * 100 + 50 (valid bonus) = 140
        assert!((score - 140.0).abs() < 0.001);
    }

    #[test]
    fn test_score_attempt_parse_error() {
        let attempt = RetryAttempt {
            attempt_number: 1,
            parameters: HashMap::new(),
            llm_response: "invalid".to_string(),
            evaluation: EvaluationResult {
                status: EvalStatus::Invalid,
                failure_type: Some(FailureType::ParseError),
                suggested_adjustment: None,
                confidence: None,
                details: HashMap::new(),
            },
            duration_ms: 500,
            tokens_used: 50,
        };

        let score = score_attempt(&attempt);
        // 0 (no confidence) - 100 (parse error penalty) = -100
        assert!((score - (-100.0)).abs() < 0.001);
    }

    #[test]
    fn test_find_strategy_for_failure() {
        let strategies = default_strategies();

        let found = find_strategy_for_failure(&strategies, FailureType::LowConfidence);
        assert!(found.is_some());
        assert_eq!(found.unwrap().failure_type, FailureType::LowConfidence);

        let found = find_strategy_for_failure(&strategies, FailureType::InvalidCategory);
        assert!(found.is_some());
        assert_eq!(found.unwrap().failure_type, FailureType::InvalidCategory);
    }

    #[test]
    fn test_validate_config_valid() {
        let config = SolverConfig {
            max_retries: 3,
            strategies: default_strategies(),
            evaluation_rules: "rules/test.clp".to_string(),
            conclusion_type: ConclusionType::Classification,
            confidence_threshold: 0.7,
            timeout_ms: 30000,
            valid_categories: vec!["high".to_string(), "low".to_string()],
        };

        assert!(validate_config(&config).is_ok());
    }

    #[test]
    fn test_validate_config_invalid_max_retries() {
        let config = SolverConfig {
            max_retries: 0,
            evaluation_rules: "rules/test.clp".to_string(),
            ..Default::default()
        };

        assert!(validate_config(&config).is_err());
    }

    #[test]
    fn test_validate_config_duplicate_strategies() {
        let config = SolverConfig {
            max_retries: 3,
            strategies: vec![
                FailureStrategy {
                    failure_type: FailureType::LowConfidence,
                    adjustments: vec![],
                },
                FailureStrategy {
                    failure_type: FailureType::LowConfidence, // Duplicate!
                    adjustments: vec![],
                },
            ],
            evaluation_rules: "rules/test.clp".to_string(),
            conclusion_type: ConclusionType::Classification,
            confidence_threshold: 0.7,
            timeout_ms: 30000,
            valid_categories: vec![],
        };

        let result = validate_config(&config);
        assert!(result.is_err());
        assert!(result.unwrap_err().contains("Duplicate"));
    }

    #[test]
    fn test_parse_evaluation_result_valid() {
        let eval_output = r#"{"status":"valid","failure_type":"none","suggested_adjustment":"","confidence":0.85}"#;
        let result = parse_evaluation_result(eval_output).unwrap();
        assert_eq!(result.status, EvalStatus::Valid);
        assert_eq!(result.confidence, Some(0.85));
    }

    #[test]
    fn test_parse_evaluation_result_retry() {
        let eval_output = r#"{"status":"retry","failure_type":"low_confidence","suggested_adjustment":"increase_temperature","confidence":0.5}"#;
        let result = parse_evaluation_result(eval_output).unwrap();
        assert_eq!(result.status, EvalStatus::Retry);
        assert_eq!(result.failure_type, Some(FailureType::LowConfidence));
        assert_eq!(
            result.suggested_adjustment,
            Some("increase_temperature".to_string())
        );
    }

    #[test]
    fn test_validate_config_invalid_knob() {
        let config = SolverConfig {
            max_retries: 3,
            strategies: vec![FailureStrategy {
                failure_type: FailureType::LowConfidence,
                adjustments: vec![KnobAdjustment {
                    knob: "invalid_knob".to_string(),
                    action: AdjustAction::Delta,
                    value: Some(0.1),
                    min: 0.0,
                    max: 1.0,
                }],
            }],
            evaluation_rules: "rules/test.clp".to_string(),
            conclusion_type: ConclusionType::Classification,
            confidence_threshold: 0.7,
            timeout_ms: 30000,
            valid_categories: vec![],
        };

        let result = validate_config(&config);
        assert!(result.is_err());
        assert!(result.unwrap_err().contains("Invalid knob name"));
    }

    #[test]
    fn test_is_valid_knob() {
        assert!(is_valid_knob("temperature"));
        assert!(is_valid_knob("top_p"));
        assert!(is_valid_knob("thinking_enabled"));
        assert!(!is_valid_knob("invalid_knob"));
        assert!(!is_valid_knob("Temperature")); // case sensitive
    }
}
