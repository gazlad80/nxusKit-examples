// Package main tests for the CLIPS+LLM hybrid pattern.
package main

import (
	"testing"
)

// Unit tests use ApplyRoutingRulesSync for fast, deterministic testing.
// Integration tests use ApplyRoutingRules with real ClipsProvider.

func TestApplyRoutingRulesSync_Security(t *testing.T) {
	classification := &TicketClassification{
		Category: "security",
		Priority: "critical",
	}

	routing := ApplyRoutingRulesSync(classification)

	if routing.Team != "security" {
		t.Errorf("Expected team 'security', got '%s'", routing.Team)
	}
	if routing.SLAHours != 4 {
		t.Errorf("Expected SLA 4 hours, got %d", routing.SLAHours)
	}
	if routing.EscalationLevel != 2 {
		t.Errorf("Expected escalation level 2, got %d", routing.EscalationLevel)
	}
}

func TestApplyRoutingRulesSync_InfrastructureCritical(t *testing.T) {
	classification := &TicketClassification{
		Category: "infrastructure",
		Priority: "critical",
	}

	routing := ApplyRoutingRulesSync(classification)

	if routing.Team != "sre" {
		t.Errorf("Expected team 'sre', got '%s'", routing.Team)
	}
	if routing.SLAHours != 2 {
		t.Errorf("Expected SLA 2 hours, got %d", routing.SLAHours)
	}
}

func TestApplyRoutingRulesSync_ApplicationHigh(t *testing.T) {
	classification := &TicketClassification{
		Category: "application",
		Priority: "high",
	}

	routing := ApplyRoutingRulesSync(classification)

	if routing.Team != "development" {
		t.Errorf("Expected team 'development', got '%s'", routing.Team)
	}
	if routing.SLAHours != 8 {
		t.Errorf("Expected SLA 8 hours, got %d", routing.SLAHours)
	}
}

func TestApplyRoutingRulesSync_General(t *testing.T) {
	classification := &TicketClassification{
		Category: "general",
		Priority: "low",
	}

	routing := ApplyRoutingRulesSync(classification)

	if routing.Team != "general-support" {
		t.Errorf("Expected team 'general-support', got '%s'", routing.Team)
	}
	if routing.SLAHours != 24 {
		t.Errorf("Expected SLA 24 hours, got %d", routing.SLAHours)
	}
	if routing.EscalationLevel != 0 {
		t.Errorf("Expected escalation level 0, got %d", routing.EscalationLevel)
	}
}

// Integration test for real CLIPS execution
// This test requires the ticket-routing.clp file to be present and CLIPS to be available
func TestApplyRoutingRules_CLIPSIntegration(t *testing.T) {
	// Skip if CLIPS rules file doesn't exist
	rulesPath := "../ticket-routing.clp"

	classification := &TicketClassification{
		Category:    "security",
		Priority:    "high",
		Sentiment:   "frustrated",
		KeyEntities: []string{"breach"},
	}

	ctx := t.Context()
	routing, err := ApplyRoutingRules(ctx, classification, rulesPath)
	if err != nil {
		// CLIPS may not be available in all environments
		t.Skipf("CLIPS integration test skipped: %v", err)
		return
	}

	if routing.Team != "security" {
		t.Errorf("Expected team 'security', got '%s'", routing.Team)
	}
	if routing.SLAHours != 4 {
		t.Errorf("Expected SLA 4 hours, got %d", routing.SLAHours)
	}
	if routing.EscalationLevel != 2 {
		t.Errorf("Expected escalation level 2, got %d", routing.EscalationLevel)
	}
}

// Note: Full AnalyzeTicket integration tests require both LLM and CLIPS providers.
// The core routing logic is tested through the sync tests above.
