// Package main demonstrates the alert triage pattern with LLM integration.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// Alert represents an alert from a monitoring system (Alertmanager format).
type Alert struct {
	// AlertName is the name of the alert rule
	AlertName string `json:"alertname"`
	// Severity level
	Severity string `json:"severity"`
	// Instance that triggered the alert
	Instance string `json:"instance"`
	// Description is human-readable description
	Description string `json:"description"`
}

// TriageResult represents the result of triaging an alert.
type TriageResult struct {
	// AlertName is copied from input
	AlertName string `json:"alertname"`
	// Priority 1-5 (1 = highest)
	Priority int `json:"priority"`
	// Summary is a one-line assessment
	Summary string `json:"summary"`
	// LikelyCause is best guess at root cause
	LikelyCause string `json:"likely_cause"`
	// SuggestedActions are recommended actions to take
	SuggestedActions []string `json:"suggested_actions"`
}

const triageSystemPrompt = `You are an SRE assistant. Triage the provided alerts and suggest actions.
Return a JSON array with one object per alert. Each object must have:
- alertname: string (copy from input)
- priority: number 1-5 (1 = highest/critical, 5 = lowest/informational)
- summary: string (one-line assessment)
- likely_cause: string (best guess at root cause)
- suggested_actions: array of strings (recommended actions)

Critical and high severity alerts should have priority 1-2.`

// TriageAlerts triages a batch of alerts using an LLM.
func TriageAlerts(ctx context.Context, provider nxuskit.LLMProvider, model string, alerts []Alert) ([]TriageResult, error) {
	alertsJSON, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal alerts: %w", err)
	}

	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.SystemMessage(triageSystemPrompt),
			nxuskit.UserMessage(fmt.Sprintf("Triage these alerts:\n%s", alertsJSON)),
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

	payload := triageJSONPayload(resp.Content)
	var results []TriageResult
	if err := json.Unmarshal([]byte(payload), &results); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return results, nil
}

// triageJSONPayload extracts a JSON array from model output (markdown fences, preamble).
func triageJSONPayload(content string) string {
	s := strings.TrimSpace(content)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSpace(s)
		if i := strings.Index(s, "```"); i >= 0 {
			s = strings.TrimSpace(s[:i])
		}
	}
	if i := strings.Index(s, "["); i >= 0 {
		if j := strings.LastIndex(s, "]"); j > i {
			return s[i : j+1]
		}
	}
	return s
}

// ParseTriageResults parses triage results from JSON (useful for testing).
func ParseTriageResults(jsonStr string) ([]TriageResult, error) {
	var results []TriageResult
	if err := json.Unmarshal([]byte(jsonStr), &results); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return results, nil
}
