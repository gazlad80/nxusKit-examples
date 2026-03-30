//go:build nxuskit

package arbiter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// ClipsRulesValidator validates LLM responses using actual CLIPS rules.
type ClipsRulesValidator struct {
	provider nxuskit.LLMProvider
	rulesDir string
}

// NewClipsRulesValidator creates a new CLIPS validator with the given rules directory.
func NewClipsRulesValidator(rulesDir string) (*ClipsRulesValidator, error) {
	provider, err := nxuskit.NewClipsFFIProvider(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIPS provider: %w", err)
	}

	return &ClipsRulesValidator{
		provider: provider,
		rulesDir: rulesDir,
	}, nil
}

// Validate validates an LLM response using CLIPS rules.
func (v *ClipsRulesValidator) Validate(ctx context.Context, llmResponse string, config *SolverConfig) (*EvaluationResult, error) {
	// Strip markdown code fences if present (LLMs often wrap JSON in ```json...```)
	cleaned := stripMarkdownFences(llmResponse)

	// Parse the LLM response to extract fields
	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &responseData); err != nil {
		ft := ParseError
		return &EvaluationResult{
			Status:      Retry,
			FailureType: &ft,
		}, nil
	}

	// Build CLIPS facts from the response and config
	input := buildValidationInput(responseData, config)

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal validation input: %w", err)
	}

	req := &nxuskit.ChatRequest{
		Model: "classification-eval.clp",
		Messages: []nxuskit.Message{
			nxuskit.UserMessage(string(inputJSON)),
		},
	}

	resp, err := v.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("CLIPS validation failed: %w", err)
	}

	// Parse CLIPS output
	var output clipsOutputWire
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		return nil, fmt.Errorf("failed to parse CLIPS output: %w", err)
	}

	// Extract evaluation result from conclusions
	return extractEvaluationResult(output.Conclusions, config)
}

func buildValidationInput(responseData map[string]interface{}, config *SolverConfig) clipsInputWire {
	var facts []clipsFactWire

	// Map LLM response fields to classification-output template
	// (matching classification-eval.clp deftemplates)
	category, _ := responseData["category"].(string)
	confidence, _ := responseData["confidence"].(float64)
	reasoning, _ := responseData["reasoning"].(string)

	facts = append(facts, clipsFactWire{
		Template: "classification-output",
		Values: map[string]interface{}{
			"category":     category,
			"confidence":   confidence,
			"reasoning":    reasoning,
			"raw-response": fmt.Sprintf("%v", responseData),
		},
	})

	// Build valid-categories as individual values for CLIPS multislot
	facts = append(facts, clipsFactWire{
		Template: "eval-config",
		Values: map[string]interface{}{
			"confidence-threshold": config.ConfidenceThreshold,
			"valid-categories":     strings.Join(config.ValidCategories, " "),
			"require-reasoning":    1,
		},
	})

	inc, der := true, true
	maxR := int64(1000)
	return clipsInputWire{
		Facts: facts,
		Config: &clipsRequestConfigWire{
			IncludeTrace:   &inc,
			MaxRules:       &maxR,
			DerivedOnlyNew: &der,
		},
	}
}

func extractEvaluationResult(conclusions []clipsConclusionWire, config *SolverConfig) (*EvaluationResult, error) {
	result := &EvaluationResult{
		Status: Valid,
	}

	// Look for validation conclusions
	for _, c := range conclusions {
		switch c.Template {
		case "evaluation-result", "validation-result":
			if status, ok := c.Values["status"].(string); ok {
				switch status {
				case "valid":
					result.Status = Valid
				case "invalid":
					result.Status = Invalid
				case "retry":
					result.Status = Retry
				}
			}
			ft, _ := c.Values["failure_type"].(string)
			if ft == "" {
				ft, _ = c.Values["failure-type"].(string)
			}
			if ft != "" && ft != "none" {
				failureType := FailureType(ft)
				result.FailureType = &failureType
			}
			conf, ok := c.Values["confidence"].(float64)
			if !ok {
				conf, ok = c.Values["extracted-confidence"].(float64)
			}
			if ok {
				result.Confidence = &conf
			}
			adj, _ := c.Values["suggested_adjustment"].(string)
			if adj == "" {
				adj, _ = c.Values["suggested-adjustment"].(string)
			}
			if adj != "" {
				result.SuggestedAdjustment = adj
			}

		case "low-confidence":
			result.Status = Retry
			ft := LowConfidence
			result.FailureType = &ft
			if conf, ok := c.Values["confidence"].(float64); ok {
				result.Confidence = &conf
			}

		case "invalid-category":
			result.Status = Retry
			ft := InvalidCategory
			result.FailureType = &ft

		case "missing-reasoning":
			result.Status = Retry
			ft := MissingReasoning
			result.FailureType = &ft

		case "valid-response":
			result.Status = Valid
			if conf, ok := c.Values["confidence"].(float64); ok {
				result.Confidence = &conf
			}
		}
	}

	return result, nil
}
