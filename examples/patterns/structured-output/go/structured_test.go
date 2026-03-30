// Package main tests for the structured output pattern.
package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestLogClassification_Validate(t *testing.T) {
	tests := []struct {
		name    string
		lc      LogClassification
		wantErr bool
	}{
		{
			name: "valid classification",
			lc: LogClassification{
				Severity:   "error",
				Category:   "auth",
				Summary:    "Login failed",
				Actionable: true,
			},
			wantErr: false,
		},
		{
			name: "invalid severity",
			lc: LogClassification{
				Severity: "invalid",
				Category: "auth",
			},
			wantErr: true,
		},
		{
			name: "invalid category",
			lc: LogClassification{
				Severity: "error",
				Category: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.lc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClassifyLog_Success(t *testing.T) {
	ctx := context.Background()

	// Mock response with valid JSON
	mockJSON := `{"severity": "error", "category": "auth", "summary": "Failed login attempt", "actionable": true}`
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: mockJSON,
			Model:   "test-model",
		}),
	)

	logEntry := "2024-01-15 10:23:45 ERROR Failed login attempt for user admin"

	result, err := ClassifyLog(ctx, provider, "test-model", logEntry)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result.Severity != "error" {
		t.Errorf("Expected severity 'error', got '%s'", result.Severity)
	}
	if result.Category != "auth" {
		t.Errorf("Expected category 'auth', got '%s'", result.Category)
	}
	if !result.Actionable {
		t.Error("Expected actionable to be true")
	}
}

func TestClassifyLog_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "not valid json",
			Model:   "test-model",
		}),
	)

	_, err := ClassifyLog(ctx, provider, "test-model", "test log")
	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}

func TestClassifyLog_InvalidClassification(t *testing.T) {
	ctx := context.Background()

	// Mock response with invalid severity
	mockJSON := `{"severity": "invalid", "category": "auth", "summary": "test", "actionable": false}`
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: mockJSON,
			Model:   "test-model",
		}),
	)

	_, err := ClassifyLog(ctx, provider, "test-model", "test log")
	if err == nil {
		t.Error("Expected validation error for invalid severity")
	}
}

func TestParseClassification_Success(t *testing.T) {
	jsonStr := `{"severity": "critical", "category": "system", "summary": "Disk full", "actionable": true}`

	result, err := ParseClassification(jsonStr)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result.Severity != "critical" {
		t.Errorf("Expected severity 'critical', got '%s'", result.Severity)
	}
	if result.Category != "system" {
		t.Errorf("Expected category 'system', got '%s'", result.Category)
	}
}
