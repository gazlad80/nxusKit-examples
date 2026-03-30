// Package main tests for the alert triage pattern.
package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestTriageAlerts_Success(t *testing.T) {
	ctx := context.Background()

	// Mock response with valid JSON array
	mockJSON := `[{"alertname": "HighMemoryUsage", "priority": 3, "summary": "Memory usage is high", "likely_cause": "Memory leak", "suggested_actions": ["Restart service", "Check logs"]}]`
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: mockJSON,
			Model:   "test-model",
		}),
	)

	alerts := []Alert{
		{
			AlertName:   "HighMemoryUsage",
			Severity:    "warning",
			Instance:    "web-server-01",
			Description: "Memory usage above 85%",
		},
	}

	results, err := TriageAlerts(ctx, provider, "test-model", alerts)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].AlertName != "HighMemoryUsage" {
		t.Errorf("Expected alertname 'HighMemoryUsage', got '%s'", results[0].AlertName)
	}

	if results[0].Priority != 3 {
		t.Errorf("Expected priority 3, got %d", results[0].Priority)
	}

	if len(results[0].SuggestedActions) != 2 {
		t.Errorf("Expected 2 suggested actions, got %d", len(results[0].SuggestedActions))
	}
}

func TestTriageAlerts_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "not valid json",
			Model:   "test-model",
		}),
	)

	alerts := []Alert{
		{
			AlertName: "TestAlert",
			Severity:  "warning",
		},
	}

	_, err := TriageAlerts(ctx, provider, "test-model", alerts)
	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}

func TestTriageAlerts_MultipleAlerts(t *testing.T) {
	ctx := context.Background()

	mockJSON := `[
		{"alertname": "Alert1", "priority": 1, "summary": "Critical", "likely_cause": "Cause 1", "suggested_actions": ["Action 1"]},
		{"alertname": "Alert2", "priority": 4, "summary": "Low priority", "likely_cause": "Cause 2", "suggested_actions": ["Action 2"]}
	]`
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: mockJSON,
			Model:   "test-model",
		}),
	)

	alerts := []Alert{
		{AlertName: "Alert1", Severity: "critical"},
		{AlertName: "Alert2", Severity: "warning"},
	}

	results, err := TriageAlerts(ctx, provider, "test-model", alerts)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].Priority != 1 {
		t.Errorf("Expected first alert priority 1, got %d", results[0].Priority)
	}

	if results[1].Priority != 4 {
		t.Errorf("Expected second alert priority 4, got %d", results[1].Priority)
	}
}

func TestTriageJSONPayload_MarkdownFence(t *testing.T) {
	raw := "Here is the result:\n```json\n[{\"alertname\":\"X\",\"priority\":1,\"summary\":\"s\",\"likely_cause\":\"c\",\"suggested_actions\":[\"a\"]}]\n```"
	payload := triageJSONPayload(raw)
	var decoded []TriageResult
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(decoded) != 1 || decoded[0].AlertName != "X" {
		t.Fatalf("got %+v", decoded)
	}
}

func TestParseTriageResults_Success(t *testing.T) {
	jsonStr := `[{"alertname": "Test", "priority": 2, "summary": "Test summary", "likely_cause": "Test cause", "suggested_actions": ["action"]}]`

	results, err := ParseTriageResults(jsonStr)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].AlertName != "Test" {
		t.Errorf("Expected alertname 'Test', got '%s'", results[0].AlertName)
	}
}
