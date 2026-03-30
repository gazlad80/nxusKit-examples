// Package main demonstrates structured output (JSON mode) pattern.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// LogClassification represents a classified log entry.
type LogClassification struct {
	// Severity level: "info", "warning", "error", "critical"
	Severity string `json:"severity"`
	// Category: "auth", "network", "system", "application"
	Category string `json:"category"`
	// One-line summary of the log entry
	Summary string `json:"summary"`
	// Whether immediate action is required
	Actionable bool `json:"actionable"`
}

// Validate checks if the classification fields are valid.
func (lc *LogClassification) Validate() error {
	validSeverities := []string{"info", "warning", "error", "critical"}
	validSeverity := false
	for _, s := range validSeverities {
		if lc.Severity == s {
			validSeverity = true
			break
		}
	}
	if !validSeverity {
		return fmt.Errorf("invalid severity: %s", lc.Severity)
	}

	validCategories := []string{"auth", "network", "system", "application"}
	validCategory := false
	for _, c := range validCategories {
		if lc.Category == c {
			validCategory = true
			break
		}
	}
	if !validCategory {
		return fmt.Errorf("invalid category: %s", lc.Category)
	}

	return nil
}

const systemPrompt = `You are a log classifier. Analyze the log entry and respond with JSON only.
Format: {"severity": "info|warning|error|critical", "category": "auth|network|system|application", "summary": "one-line summary", "actionable": true|false}`

// ClassifyLog classifies a log entry using an LLM with JSON mode.
func ClassifyLog(ctx context.Context, provider nxuskit.LLMProvider, model, logEntry string) (*LogClassification, error) {
	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.SystemMessage(systemPrompt),
			nxuskit.UserMessage(logEntry),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Enable JSON mode for structured output
	req.JSONMode = true

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	var classification LogClassification
	if err := json.Unmarshal([]byte(resp.Content), &classification); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if err := classification.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &classification, nil
}

// ParseClassification parses a JSON string into a LogClassification.
// Useful for testing with mock responses.
func ParseClassification(jsonStr string) (*LogClassification, error) {
	var classification LogClassification
	if err := json.Unmarshal([]byte(jsonStr), &classification); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if err := classification.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &classification, nil
}
