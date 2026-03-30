// Package arbiter provides the auto-retry LLM pattern with CLIPS validation.
//
// ## nxusKit Features Demonstrated
// - ClipsProvider for deterministic output validation
// - Auto-retry loop with CLIPS-driven parameter adjustment
// - JSON-based fact assertion for LLM output evaluation
// - Provider-agnostic solver pattern (works with any LLMProvider)
//
// ## Why This Pattern Matters
// LLM outputs can be inconsistent. Using CLIPS rules to validate outputs
// enables automatic retries with intelligent parameter adjustments.
// This ensures consistent, policy-compliant results.
package arbiter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Solver orchestrates the auto-retry loop with CLIPS validation.
type DemoSolver struct {
	Config     SolverConfig
	Strategies []FailureStrategy
}

// NewSolver creates a new Solver with the given configuration.
func NewDemoSolver(config SolverConfig) *DemoSolver {
	strategies := config.Strategies
	if len(strategies) == 0 {
		strategies = DefaultStrategies()
	}

	return &DemoSolver{
		Config:     config,
		Strategies: strategies,
	}
}

// Run executes the solver loop with auto-retry.
//
// nxusKit Features: Uses CLIPS rules to evaluate LLM output quality.
// The DemoSolver uses hardcoded responses for offline testing.
// The CLIPS validation logic is demonstrated in clips_validator.go (build tag: clips).
func (s *DemoSolver) Run(input string, verbose bool) (*SolverResult, error) {
	if err := s.Config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	startTime := time.Now()
	var retryHistory []RetryAttempt
	currentParams := s.initializeParams()

	for attemptNum := 1; attemptNum <= s.Config.MaxRetries; attemptNum++ {
		attemptStart := time.Now()

		if verbose {
			temp := 0.7
			if v, ok := currentParams["temperature"].(float64); ok {
				temp = v
			}
			fmt.Printf("Attempt %d:\n", attemptNum)
			fmt.Printf("  Parameters: temperature=%.1f\n", temp)
		}

		// nxusKit: Generate LLM response (build with -tags nxuskit for real SDK)
		llmResponse, evalJSON := s.generateLLMResponse(attemptNum, input)

		if verbose {
			fmt.Printf("  LLM Response: %s\n", truncate(llmResponse, 80))
		}

		// Parse evaluation result
		evaluation, err := ParseEvaluationResult(evalJSON)
		if err != nil {
			if verbose {
				fmt.Printf("  Evaluation parse error: %v\n", err)
			}
			continue
		}

		if verbose {
			fmt.Printf("  Evaluation: %s\n", evaluation.Status)
			if evaluation.FailureType != nil {
				fmt.Printf("  Failure: %s\n", *evaluation.FailureType)
			}
		}

		attempt := RetryAttempt{
			AttemptNumber: attemptNum,
			Parameters:    copyParams(currentParams),
			LLMResponse:   llmResponse,
			Evaluation:    *evaluation,
			DurationMS:    time.Since(attemptStart).Milliseconds(),
			TokensUsed:    150 + int64(attemptNum)*50, // Example (real token counts available with -tags nxuskit)
		}

		retryHistory = append(retryHistory, attempt)

		// Check if valid
		if evaluation.Status == Valid {
			totalTokens := sumTokens(retryHistory)
			var finalOutput json.RawMessage
			finalOutput = []byte(llmResponse)

			return &SolverResult{
				Success:         true,
				FinalOutput:     finalOutput,
				BestAttempt:     attempt,
				RetryHistory:    retryHistory,
				TotalDurationMS: time.Since(startTime).Milliseconds(),
				TotalTokens:     totalTokens,
			}, nil
		}

		// Apply adjustments for retry
		if evaluation.FailureType != nil {
			if strategy := s.findStrategy(*evaluation.FailureType); strategy != nil {
				ApplyAdjustments(currentParams, strategy)
				if verbose {
					knobs := make([]string, 0, len(strategy.Adjustments))
					for _, adj := range strategy.Adjustments {
						knobs = append(knobs, adj.Knob)
					}
					fmt.Printf("  Adjustment: %v\n", knobs)
				}
			}
		}

		if verbose {
			fmt.Println()
		}
	}

	// Max retries exceeded - return best attempt
	best := findBestAttempt(retryHistory)
	totalTokens := sumTokens(retryHistory)
	var finalOutput json.RawMessage
	finalOutput = []byte(best.LLMResponse)

	return &SolverResult{
		Success:         false,
		FinalOutput:     finalOutput,
		BestAttempt:     best,
		RetryHistory:    retryHistory,
		TotalDurationMS: time.Since(startTime).Milliseconds(),
		TotalTokens:     totalTokens,
	}, nil
}

func (s *DemoSolver) initializeParams() map[string]any {
	return map[string]any{
		"temperature":      0.7,
		"max_tokens":       1000.0,
		"thinking_enabled": 0.0,
	}
}

func (s *DemoSolver) findStrategy(failureType FailureType) *FailureStrategy {
	for i := range s.Strategies {
		if s.Strategies[i].FailureType == failureType {
			return &s.Strategies[i]
		}
	}
	return nil
}

// generateLLMResponse generates LLM response for demonstration.
// Build with -tags nxuskit to use real LLM + CLIPS providers.
// See clips_validator.go for real CLIPS integration (build tag: clips).
func (s *DemoSolver) generateLLMResponse(attempt int, _ string) (string, string) {
	switch attempt {
	case 1:
		response := `{"category": "high", "confidence": 0.5, "reasoning": ""}`
		eval := `{"status":"retry","failure_type":"low_confidence","suggested_adjustment":"increase_temperature","confidence":0.5}`
		return response, eval
	case 2:
		response := `{"category": "critical", "confidence": 0.85, "reasoning": "Account security issue"}`
		eval := `{"status":"retry","failure_type":"invalid_category","suggested_adjustment":"decrease_temperature","confidence":0.85}`
		return response, eval
	default:
		category := "high"
		if len(s.Config.ValidCategories) > 0 {
			category = s.Config.ValidCategories[0]
		}
		response := fmt.Sprintf(`{"category": "%s", "confidence": 0.87, "reasoning": "Based on keywords indicating urgency"}`, category)
		eval := `{"status":"valid","failure_type":"none","suggested_adjustment":"","confidence":0.87}`
		return response, eval
	}
}

// ParseEvaluationResult parses CLIPS evaluation JSON into an EvaluationResult.
func ParseEvaluationResult(evalJSON string) (*EvaluationResult, error) {
	var data struct {
		Status              string   `json:"status"`
		FailureType         string   `json:"failure_type"`
		SuggestedAdjustment string   `json:"suggested_adjustment"`
		Confidence          *float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(evalJSON), &data); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	var status EvalStatus
	switch data.Status {
	case "valid":
		status = Valid
	case "invalid":
		status = Invalid
	case "retry":
		status = Retry
	default:
		return nil, fmt.Errorf("unknown status: %s", data.Status)
	}

	result := &EvaluationResult{
		Status:              status,
		SuggestedAdjustment: data.SuggestedAdjustment,
		Confidence:          data.Confidence,
	}

	if status != Valid && data.FailureType != "none" && data.FailureType != "" {
		ft := FailureType(data.FailureType)
		result.FailureType = &ft
	}

	return result, nil
}

// DetectFailureType extracts the failure type from CLIPS evaluation output.
func DetectFailureType(evalJSON string) *FailureType {
	result, err := ParseEvaluationResult(evalJSON)
	if err != nil {
		return nil
	}
	return result.FailureType
}

// ApplyAdjustments applies knob adjustments to current parameters.
func ApplyAdjustments(currentParams map[string]any, strategy *FailureStrategy) {
	for _, adj := range strategy.Adjustments {
		currentVal := 0.5 // Default
		if v, ok := currentParams[adj.Knob].(float64); ok {
			currentVal = v
		}

		var newVal float64
		switch adj.Action {
		case Set:
			if adj.Value != nil {
				newVal = *adj.Value
			} else {
				newVal = currentVal
			}
		case Delta:
			delta := 0.0
			if adj.Value != nil {
				delta = *adj.Value
			}
			newVal = currentVal + delta
			// Clamp
			if newVal < adj.Min {
				newVal = adj.Min
			}
			if newVal > adj.Max {
				newVal = adj.Max
			}
		case Enable:
			newVal = 1.0
		case Disable:
			newVal = 0.0
		}

		currentParams[adj.Knob] = newVal
	}
}

// ScoreAttempt scores an attempt for best-attempt selection.
func ScoreAttempt(attempt *RetryAttempt) float64 {
	score := 0.0

	if attempt.Evaluation.Confidence != nil {
		score += *attempt.Evaluation.Confidence * 100.0
	}

	if attempt.Evaluation.Status == Valid {
		score += 50.0
	}

	if attempt.Evaluation.FailureType != nil && *attempt.Evaluation.FailureType == ParseError {
		score -= 100.0
	}

	return score
}

func findBestAttempt(history []RetryAttempt) RetryAttempt {
	if len(history) == 0 {
		return RetryAttempt{}
	}

	best := history[0]
	bestScore := ScoreAttempt(&best)

	for i := 1; i < len(history); i++ {
		score := ScoreAttempt(&history[i])
		if score > bestScore {
			best = history[i]
			bestScore = score
		}
	}

	return best
}

func sumTokens(history []RetryAttempt) int64 {
	var total int64
	for _, attempt := range history {
		total += attempt.TokensUsed
	}
	return total
}

func copyParams(params map[string]any) map[string]any {
	copy := make(map[string]any)
	for k, v := range params {
		copy[k] = v
	}
	return copy
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// stripMarkdownFences removes markdown code fences (```json...```) from LLM output.
func stripMarkdownFences(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	// Skip the opening ``` and optional language tag
	rest := trimmed[3:]
	if idx := strings.Index(rest, "\n"); idx != -1 {
		rest = rest[idx+1:]
	}
	// Remove closing ```
	if idx := strings.LastIndex(rest, "```"); idx != -1 {
		rest = rest[:idx]
	}
	return strings.TrimSpace(rest)
}
