// Package main tests for the cost routing pattern.
package main

import (
	"testing"
)

func TestClassifyTask_Economy(t *testing.T) {
	tests := []string{
		"What is 2+2?",
		"Hello",
		"What time is it?",
	}

	for _, prompt := range tests {
		tier := ClassifyTask(prompt)
		if tier != TierEconomy {
			t.Errorf("Expected Economy tier for '%s', got %s", prompt, tier.Name())
		}
	}
}

func TestClassifyTask_Standard(t *testing.T) {
	// Medium-length prompt (>200 chars but no premium keywords)
	prompt := "I need help understanding how to properly structure my Go project. " +
		"I have several packages and want to organize them in a way that makes sense. " +
		"Can you give me some best practices for package organization in Go?"

	tier := ClassifyTask(prompt)
	if tier != TierStandard {
		t.Errorf("Expected Standard tier, got %s", tier.Name())
	}
}

func TestClassifyTask_Premium_Keyword(t *testing.T) {
	tests := []string{
		"Analyze the performance characteristics of this algorithm",
		"Compare and contrast microservices vs monolith",
		"Evaluate the trade-offs between these approaches",
		"Synthesize information from multiple sources",
		"Critique this implementation",
	}

	for _, prompt := range tests {
		tier := ClassifyTask(prompt)
		if tier != TierPremium {
			t.Errorf("Expected Premium tier for '%s' (contains keyword), got %s", prompt, tier.Name())
		}
	}
}

func TestClassifyTask_Premium_Length(t *testing.T) {
	// Very long prompt (>1000 chars)
	prompt := ""
	for i := 0; i < 30; i++ {
		prompt += "This is a very long prompt that exceeds the character limit. "
	}

	tier := ClassifyTask(prompt)
	if tier != TierPremium {
		t.Errorf("Expected Premium tier for long prompt, got %s", tier.Name())
	}
}

func TestCostTier_ModelName(t *testing.T) {
	tests := []struct {
		tier     CostTier
		expected string
	}{
		{TierEconomy, "gpt-4o-mini"},
		{TierStandard, "gpt-4o"},
		{TierPremium, "gpt-4-turbo"},
	}

	for _, tt := range tests {
		model := tt.tier.ModelName()
		if model != tt.expected {
			t.Errorf("Expected model '%s' for tier %s, got '%s'", tt.expected, tt.tier.Name(), model)
		}
	}
}

func TestCostTier_Name(t *testing.T) {
	tests := []struct {
		tier     CostTier
		expected string
	}{
		{TierEconomy, "Economy"},
		{TierStandard, "Standard"},
		{TierPremium, "Premium"},
	}

	for _, tt := range tests {
		name := tt.tier.Name()
		if name != tt.expected {
			t.Errorf("Expected name '%s', got '%s'", tt.expected, name)
		}
	}
}
