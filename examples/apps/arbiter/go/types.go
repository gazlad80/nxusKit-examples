// Package arbiter provides types and implementation for the auto-retry LLM pattern with CLIPS validation.
package arbiter

import (
	"encoding/json"
	"time"
)

// ConclusionType defines the type of LLM conclusion expected.
type ConclusionType string

const (
	// Classification indicates LLM categorizes input into predefined categories.
	Classification ConclusionType = "classification"
	// Extraction indicates LLM extracts structured fields from input.
	Extraction ConclusionType = "extraction"
	// Reasoning indicates LLM performs logical reasoning with chain-of-thought.
	Reasoning ConclusionType = "reasoning"
)

// FailureType defines the type of validation failure detected by CLIPS.
type FailureType string

const (
	// LowConfidence indicates confidence below threshold.
	LowConfidence FailureType = "low_confidence"
	// InvalidCategory indicates category not in allowed set.
	InvalidCategory FailureType = "invalid_category"
	// MissingReasoning indicates empty or null reasoning.
	MissingReasoning FailureType = "missing_reasoning"
	// IncompleteExtraction indicates required fields missing.
	IncompleteExtraction FailureType = "incomplete_extraction"
	// InconsistentData indicates cross-field validation failed.
	InconsistentData FailureType = "inconsistent_data"
	// ParseError indicates cannot parse LLM output.
	ParseError FailureType = "parse_error"
)

// Description returns a human-readable description of this failure type.
func (f FailureType) Description() string {
	switch f {
	case LowConfidence:
		return "Confidence below threshold"
	case InvalidCategory:
		return "Category not in allowed set"
	case MissingReasoning:
		return "Empty or missing reasoning"
	case IncompleteExtraction:
		return "Required fields missing"
	case InconsistentData:
		return "Cross-field validation failed"
	case ParseError:
		return "Cannot parse LLM output"
	default:
		return "Unknown failure type"
	}
}

// AdjustAction defines how to modify a knob value.
type AdjustAction string

const (
	// Set sets the knob to a specific value.
	Set AdjustAction = "set"
	// Delta adds/subtracts from current value.
	Delta AdjustAction = "delta"
	// Enable sets boolean to true.
	Enable AdjustAction = "enable"
	// Disable sets boolean to false.
	Disable AdjustAction = "disable"
)

// KnobAdjustment specifies how to adjust a single parameter.
type KnobAdjustment struct {
	// Knob is the parameter name (temperature, top_p, etc.)
	Knob string `json:"knob"`
	// Action specifies how to modify the value.
	Action AdjustAction `json:"action"`
	// Value for set/delta actions.
	Value *float64 `json:"value,omitempty"`
	// Min is the minimum allowed value (default: 0.0).
	Min float64 `json:"min,omitempty"`
	// Max is the maximum allowed value (default: 2.0).
	Max float64 `json:"max,omitempty"`
}

// FailureStrategy maps a failure type to parameter adjustments.
type FailureStrategy struct {
	// FailureType is the failure condition this strategy handles.
	FailureType FailureType `json:"failure_type"`
	// Adjustments are the parameter changes to apply.
	Adjustments []KnobAdjustment `json:"adjustments"`
}

// SolverConfig configures a solver instance.
type SolverConfig struct {
	// MaxRetries is the maximum retry attempts (default: 3).
	MaxRetries int `json:"max_retries,omitempty"`
	// Strategies are failure-to-adjustment mappings.
	Strategies []FailureStrategy `json:"strategies,omitempty"`
	// EvaluationRules is the path to CLIPS rules file or inline rules.
	EvaluationRules string `json:"evaluation_rules"`
	// ConclusionType is the type of LLM output expected.
	ConclusionType ConclusionType `json:"conclusion_type"`
	// ConfidenceThreshold is the minimum confidence for valid result (default: 0.7).
	ConfidenceThreshold float64 `json:"confidence_threshold,omitempty"`
	// TimeoutMS is the total timeout for all retries in milliseconds (default: 30000).
	TimeoutMS int64 `json:"timeout_ms,omitempty"`
	// ValidCategories for classification type.
	ValidCategories []string `json:"valid_categories,omitempty"`
}

// DefaultConfig returns a SolverConfig with default values.
func DefaultConfig() SolverConfig {
	return SolverConfig{
		MaxRetries:          3,
		ConfidenceThreshold: 0.7,
		TimeoutMS:           30000,
	}
}

// EvalStatus is the result status from CLIPS validation.
type EvalStatus string

const (
	// Valid indicates output passed all validation rules.
	Valid EvalStatus = "valid"
	// Invalid indicates output failed validation, no retry suggested.
	Invalid EvalStatus = "invalid"
	// Retry indicates output failed validation, retry recommended.
	Retry EvalStatus = "retry"
)

// EvaluationResult holds the result from CLIPS validation.
type EvaluationResult struct {
	// Status is the evaluation status.
	Status EvalStatus `json:"status"`
	// FailureType is the type of failure (if status != valid).
	FailureType *FailureType `json:"failure_type,omitempty"`
	// SuggestedAdjustment from CLIPS.
	SuggestedAdjustment string `json:"suggested_adjustment,omitempty"`
	// Confidence is the extracted confidence value.
	Confidence *float64 `json:"confidence,omitempty"`
	// Details contains additional evaluation metadata.
	Details map[string]any `json:"details,omitempty"`
}

// RetryAttempt records a single retry attempt.
type RetryAttempt struct {
	// AttemptNumber is 1-indexed.
	AttemptNumber int `json:"attempt_number"`
	// Parameters are the LLM parameters used.
	Parameters map[string]any `json:"parameters"`
	// LLMResponse is the raw LLM output.
	LLMResponse string `json:"llm_response"`
	// Evaluation is the CLIPS evaluation result.
	Evaluation EvaluationResult `json:"evaluation"`
	// DurationMS is the time taken for this attempt.
	DurationMS int64 `json:"duration_ms"`
	// TokensUsed for this attempt.
	TokensUsed int64 `json:"tokens_used,omitempty"`
}

// SolverResult is the final result from solver execution.
type SolverResult struct {
	// Success indicates whether validation ultimately passed.
	Success bool `json:"success"`
	// FinalOutput is the parsed output from best attempt.
	FinalOutput json.RawMessage `json:"final_output"`
	// BestAttempt is the best attempt by score.
	BestAttempt RetryAttempt `json:"best_attempt"`
	// RetryHistory contains all attempts in order.
	RetryHistory []RetryAttempt `json:"retry_history"`
	// TotalDurationMS is the total execution time.
	TotalDurationMS int64 `json:"total_duration_ms"`
	// TotalTokens is the total tokens consumed.
	TotalTokens int64 `json:"total_tokens"`
}

// DefaultStrategies returns the default failure-to-adjustment mappings.
func DefaultStrategies() []FailureStrategy {
	tempDelta02 := 0.2
	tempDeltaNeg02 := -0.2
	tempDeltaNeg01 := -0.1
	tempSet0 := 0.0
	maxTokensDelta := 500.0

	return []FailureStrategy{
		{
			FailureType: LowConfidence,
			Adjustments: []KnobAdjustment{
				{Knob: "temperature", Action: Delta, Value: &tempDelta02, Min: 0.0, Max: 2.0},
			},
		},
		{
			FailureType: InvalidCategory,
			Adjustments: []KnobAdjustment{
				{Knob: "temperature", Action: Delta, Value: &tempDeltaNeg02, Min: 0.0, Max: 2.0},
			},
		},
		{
			FailureType: MissingReasoning,
			Adjustments: []KnobAdjustment{
				{Knob: "thinking_enabled", Action: Enable, Min: 0.0, Max: 1.0},
			},
		},
		{
			FailureType: IncompleteExtraction,
			Adjustments: []KnobAdjustment{
				{Knob: "temperature", Action: Delta, Value: &tempDeltaNeg01, Min: 0.0, Max: 2.0},
				{Knob: "max_tokens", Action: Delta, Value: &maxTokensDelta, Min: 100.0, Max: 8000.0},
			},
		},
		{
			FailureType: InconsistentData,
			Adjustments: []KnobAdjustment{
				{Knob: "thinking_enabled", Action: Enable, Min: 0.0, Max: 1.0},
			},
		},
		{
			FailureType: ParseError,
			Adjustments: []KnobAdjustment{
				{Knob: "temperature", Action: Set, Value: &tempSet0, Min: 0.0, Max: 2.0},
			},
		},
	}
}

// Timeout returns the configured timeout as a time.Duration.
func (c *SolverConfig) Timeout() time.Duration {
	if c.TimeoutMS <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.TimeoutMS) * time.Millisecond
}

// Validate validates the solver configuration.
func (c *SolverConfig) Validate() error {
	if c.MaxRetries < 1 || c.MaxRetries > 10 {
		return &ConfigError{Field: "max_retries", Message: "must be between 1 and 10"}
	}
	if c.ConfidenceThreshold < 0.0 || c.ConfidenceThreshold > 1.0 {
		return &ConfigError{Field: "confidence_threshold", Message: "must be between 0.0 and 1.0"}
	}
	if c.EvaluationRules == "" {
		return &ConfigError{Field: "evaluation_rules", Message: "must be specified"}
	}
	if c.ConclusionType == "" {
		return &ConfigError{Field: "conclusion_type", Message: "must be specified"}
	}

	// Check for duplicate failure types in strategies
	seen := make(map[FailureType]bool)
	for _, s := range c.Strategies {
		if seen[s.FailureType] {
			return &ConfigError{
				Field:   "strategies",
				Message: "duplicate failure type: " + string(s.FailureType),
			}
		}
		seen[s.FailureType] = true

		// Validate knob names
		for _, adj := range s.Adjustments {
			if !IsValidKnob(adj.Knob) {
				return &ConfigError{
					Field:   "strategies",
					Message: "invalid knob name '" + adj.Knob + "' in strategy for " + string(s.FailureType),
				}
			}
		}
	}

	return nil
}

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "invalid config: " + e.Field + " " + e.Message
}
