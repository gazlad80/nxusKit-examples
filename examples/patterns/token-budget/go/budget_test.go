// Package main tests for the token budget pattern.
package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestStreamWithBudget_BudgetNotReached(t *testing.T) {
	ctx := context.Background()

	// Create mock provider that streams a short response
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockStreamResponse([]nxuskit.StreamChunk{
			{Delta: "Hello"},
			{Delta: " world"},
			{Delta: "!"},
		}),
	)

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set a high budget that won't be reached
	result, err := StreamWithBudget(ctx, provider, req, 100)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if result.BudgetReached {
		t.Error("Budget should not have been reached")
	}

	expectedContent := "Hello world!"
	if result.Content != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, result.Content)
	}
}

func TestStreamWithBudget_BudgetReached(t *testing.T) {
	ctx := context.Background()

	// Create mock provider that streams a longer response
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockStreamResponse([]nxuskit.StreamChunk{
			{Delta: "This is a long response that "},
			{Delta: "will exceed the token budget "},
			{Delta: "and should be truncated before "},
			{Delta: "reaching this point."},
		}),
	)

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set a low budget that will be reached (~4 chars per token)
	result, err := StreamWithBudget(ctx, provider, req, 10)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if !result.BudgetReached {
		t.Error("Budget should have been reached")
	}

	// Content should be truncated
	if len(result.Content) > 60 { // 10 tokens * 4 chars + some margin
		t.Errorf("Content longer than expected: %d characters", len(result.Content))
	}
}

func TestStreamWithBudget_EstimatedTokens(t *testing.T) {
	ctx := context.Background()

	// Create mock provider that streams 80 characters (~20 tokens)
	provider := nxuskit.NewMockProvider(
		nxuskit.WithMockStreamResponse([]nxuskit.StreamChunk{
			{Delta: "12345678901234567890"}, // 20 chars
			{Delta: "12345678901234567890"}, // 20 chars
			{Delta: "12345678901234567890"}, // 20 chars
			{Delta: "12345678901234567890"}, // 20 chars
		}),
	)

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	result, err := StreamWithBudget(ctx, provider, req, 100)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	// 80 characters / 4 = 20 estimated tokens
	if result.EstimatedTokens != 20 {
		t.Errorf("Expected 20 estimated tokens, got %d", result.EstimatedTokens)
	}
}
