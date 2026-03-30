package arbiter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFailureTypeLowConfidence(t *testing.T) {
	evalOutput := `{"status":"retry","failure_type":"low_confidence","confidence":0.5}`
	ft := DetectFailureType(evalOutput)
	if ft == nil {
		t.Fatal("Expected failure type, got nil")
	}
	if *ft != LowConfidence {
		t.Errorf("Expected LowConfidence, got %s", *ft)
	}
}

func TestDetectFailureTypeInvalidCategory(t *testing.T) {
	evalOutput := `{"status":"retry","failure_type":"invalid_category","confidence":0.9}`
	ft := DetectFailureType(evalOutput)
	if ft == nil {
		t.Fatal("Expected failure type, got nil")
	}
	if *ft != InvalidCategory {
		t.Errorf("Expected InvalidCategory, got %s", *ft)
	}
}

func TestDetectFailureTypeValid(t *testing.T) {
	evalOutput := `{"status":"valid","failure_type":"none","confidence":0.85}`
	ft := DetectFailureType(evalOutput)
	if ft != nil {
		t.Errorf("Expected nil for valid status, got %v", ft)
	}
}

func TestApplyAdjustmentsDelta(t *testing.T) {
	delta := 0.2
	strategy := &FailureStrategy{
		FailureType: LowConfidence,
		Adjustments: []KnobAdjustment{
			{Knob: "temperature", Action: Delta, Value: &delta, Min: 0.0, Max: 2.0},
		},
	}

	params := map[string]any{"temperature": 0.7}
	ApplyAdjustments(params, strategy)

	newTemp, ok := params["temperature"].(float64)
	if !ok {
		t.Fatal("temperature should be float64")
	}
	expected := 0.9
	if newTemp < expected-0.001 || newTemp > expected+0.001 {
		t.Errorf("Expected %.1f, got %.1f", expected, newTemp)
	}
}

func TestApplyAdjustmentsSet(t *testing.T) {
	setVal := 0.0
	strategy := &FailureStrategy{
		FailureType: ParseError,
		Adjustments: []KnobAdjustment{
			{Knob: "temperature", Action: Set, Value: &setVal, Min: 0.0, Max: 2.0},
		},
	}

	params := map[string]any{"temperature": 0.7}
	ApplyAdjustments(params, strategy)

	newTemp, ok := params["temperature"].(float64)
	if !ok {
		t.Fatal("temperature should be float64")
	}
	if newTemp != 0.0 {
		t.Errorf("Expected 0.0, got %.1f", newTemp)
	}
}

func TestApplyAdjustmentsEnable(t *testing.T) {
	strategy := &FailureStrategy{
		FailureType: MissingReasoning,
		Adjustments: []KnobAdjustment{
			{Knob: "thinking_enabled", Action: Enable, Min: 0.0, Max: 1.0},
		},
	}

	params := map[string]any{"thinking_enabled": 0.0}
	ApplyAdjustments(params, strategy)

	thinking, ok := params["thinking_enabled"].(float64)
	if !ok {
		t.Fatal("thinking_enabled should be float64")
	}
	if thinking != 1.0 {
		t.Errorf("Expected 1.0, got %.1f", thinking)
	}
}

func TestApplyAdjustmentsClamps(t *testing.T) {
	delta := 1.0
	strategy := &FailureStrategy{
		FailureType: LowConfidence,
		Adjustments: []KnobAdjustment{
			{Knob: "temperature", Action: Delta, Value: &delta, Min: 0.0, Max: 1.5},
		},
	}

	params := map[string]any{"temperature": 1.0}
	ApplyAdjustments(params, strategy)

	newTemp, ok := params["temperature"].(float64)
	if !ok {
		t.Fatal("temperature should be float64")
	}
	// Should be clamped to max of 1.5
	if newTemp != 1.5 {
		t.Errorf("Expected 1.5 (clamped), got %.1f", newTemp)
	}
}

func TestScoreAttemptValid(t *testing.T) {
	conf := 0.9
	attempt := RetryAttempt{
		AttemptNumber: 1,
		Parameters:    map[string]any{},
		LLMResponse:   "test",
		Evaluation: EvaluationResult{
			Status:     Valid,
			Confidence: &conf,
		},
		DurationMS: 1000,
		TokensUsed: 100,
	}

	score := ScoreAttempt(&attempt)
	// 0.9 * 100 + 50 (valid bonus) = 140
	expected := 140.0
	if score < expected-0.001 || score > expected+0.001 {
		t.Errorf("Expected %.1f, got %.1f", expected, score)
	}
}

func TestScoreAttemptParseError(t *testing.T) {
	ft := ParseError
	attempt := RetryAttempt{
		AttemptNumber: 1,
		Parameters:    map[string]any{},
		LLMResponse:   "invalid",
		Evaluation: EvaluationResult{
			Status:      Invalid,
			FailureType: &ft,
		},
		DurationMS: 500,
		TokensUsed: 50,
	}

	score := ScoreAttempt(&attempt)
	// 0 (no confidence) - 100 (parse error penalty) = -100
	expected := -100.0
	if score < expected-0.001 || score > expected+0.001 {
		t.Errorf("Expected %.1f, got %.1f", expected, score)
	}
}

func TestParseEvaluationResultValid(t *testing.T) {
	evalOutput := `{"status":"valid","failure_type":"none","suggested_adjustment":"","confidence":0.85}`
	result, err := ParseEvaluationResult(evalOutput)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Status != Valid {
		t.Errorf("Expected Valid status, got %s", result.Status)
	}
	if result.Confidence == nil || *result.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %v", result.Confidence)
	}
}

func TestParseEvaluationResultRetry(t *testing.T) {
	evalOutput := `{"status":"retry","failure_type":"low_confidence","suggested_adjustment":"increase_temperature","confidence":0.5}`
	result, err := ParseEvaluationResult(evalOutput)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Status != Retry {
		t.Errorf("Expected Retry status, got %s", result.Status)
	}
	if result.FailureType == nil || *result.FailureType != LowConfidence {
		t.Errorf("Expected LowConfidence, got %v", result.FailureType)
	}
	if result.SuggestedAdjustment != "increase_temperature" {
		t.Errorf("Expected increase_temperature, got %s", result.SuggestedAdjustment)
	}
}

func TestSolverConfigValidate(t *testing.T) {
	config := SolverConfig{
		MaxRetries:          3,
		Strategies:          DefaultStrategies(),
		EvaluationRules:     "rules/test.clp",
		ConclusionType:      Classification,
		ConfidenceThreshold: 0.7,
		TimeoutMS:           30000,
		ValidCategories:     []string{"high", "low"},
	}

	if err := config.Validate(); err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestSolverConfigValidateInvalidMaxRetries(t *testing.T) {
	config := SolverConfig{
		MaxRetries:      0,
		EvaluationRules: "rules/test.clp",
		ConclusionType:  Classification,
	}

	if err := config.Validate(); err == nil {
		t.Error("Expected error for invalid max_retries")
	}
}

func TestSolverConfigValidateDuplicateStrategies(t *testing.T) {
	config := SolverConfig{
		MaxRetries: 3,
		Strategies: []FailureStrategy{
			{FailureType: LowConfidence, Adjustments: []KnobAdjustment{}},
			{FailureType: LowConfidence, Adjustments: []KnobAdjustment{}}, // Duplicate
		},
		EvaluationRules:     "rules/test.clp",
		ConclusionType:      Classification,
		ConfidenceThreshold: 0.7,
		TimeoutMS:           30000,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for duplicate failure types")
	}
}

func TestDefaultStrategies(t *testing.T) {
	strategies := DefaultStrategies()
	if len(strategies) != 6 {
		t.Errorf("Expected 6 strategies, got %d", len(strategies))
	}

	// Verify all failure types are covered
	covered := make(map[FailureType]bool)
	for _, s := range strategies {
		covered[s.FailureType] = true
	}

	expectedTypes := []FailureType{
		LowConfidence, InvalidCategory, MissingReasoning,
		IncompleteExtraction, InconsistentData, ParseError,
	}
	for _, ft := range expectedTypes {
		if !covered[ft] {
			t.Errorf("Missing strategy for failure type: %s", ft)
		}
	}
}

func TestFailureTypeDescription(t *testing.T) {
	tests := []struct {
		ft       FailureType
		expected string
	}{
		{LowConfidence, "Confidence below threshold"},
		{InvalidCategory, "Category not in allowed set"},
		{MissingReasoning, "Empty or missing reasoning"},
	}

	for _, tc := range tests {
		if tc.ft.Description() != tc.expected {
			t.Errorf("Expected %q, got %q", tc.expected, tc.ft.Description())
		}
	}
}

func TestIsValidKnob(t *testing.T) {
	validKnobs := []string{
		"temperature", "top_p", "top_k",
		"presence_penalty", "frequency_penalty",
		"max_tokens", "thinking_enabled",
	}

	for _, knob := range validKnobs {
		if !IsValidKnob(knob) {
			t.Errorf("Expected %q to be valid", knob)
		}
	}

	invalidKnobs := []string{"invalid_knob", "Temperature", "TOP_P", "bad"}
	for _, knob := range invalidKnobs {
		if IsValidKnob(knob) {
			t.Errorf("Expected %q to be invalid", knob)
		}
	}
}

func TestSolverConfigValidateInvalidKnob(t *testing.T) {
	delta := 0.1
	config := SolverConfig{
		MaxRetries: 3,
		Strategies: []FailureStrategy{
			{
				FailureType: LowConfidence,
				Adjustments: []KnobAdjustment{
					{Knob: "invalid_knob", Action: Delta, Value: &delta, Min: 0.0, Max: 1.0},
				},
			},
		},
		EvaluationRules:     "rules/test.clp",
		ConclusionType:      Classification,
		ConfidenceThreshold: 0.7,
		TimeoutMS:           30000,
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid knob name")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	configJSON := `{
		"max_retries": 5,
		"evaluation_rules": "test.clp",
		"conclusion_type": "classification",
		"confidence_threshold": 0.8,
		"timeout_ms": 60000,
		"valid_categories": ["a", "b", "c"]
	}`

	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfigFromFile(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.MaxRetries != 5 {
		t.Errorf("Expected MaxRetries 5, got %d", config.MaxRetries)
	}
	if config.ConfidenceThreshold != 0.8 {
		t.Errorf("Expected ConfidenceThreshold 0.8, got %f", config.ConfidenceThreshold)
	}
	if config.ConclusionType != Classification {
		t.Errorf("Expected Classification, got %s", config.ConclusionType)
	}
	if len(config.ValidCategories) != 3 {
		t.Errorf("Expected 3 categories, got %d", len(config.ValidCategories))
	}
}

func TestLoadConfigFromFileNotFound(t *testing.T) {
	_, err := LoadConfigFromFile("/nonexistent/path/config.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestLoadConfigFromFileInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(configPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadConfigFromFile(configPath)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}
