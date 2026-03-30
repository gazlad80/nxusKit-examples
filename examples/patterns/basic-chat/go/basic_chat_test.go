package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestBasicChat_WithMockProvider(t *testing.T) {
	ctx := context.Background()

	// Create mock provider with a pre-configured response
	mockResp := &nxuskit.ChatResponse{
		Content: "Go is a statically typed, compiled programming language designed at Google.",
		Model:   "mock-model",
		Usage: nxuskit.TokenUsage{
			Actual: &nxuskit.TokenCount{
				PromptTokens:     25,
				CompletionTokens: 50,
			},
			IsComplete: true,
		},
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockResponse(mockResp))

	// Create request similar to main.go
	req, err := nxuskit.NewChatRequest("mock-model",
		nxuskit.WithMessages(
			nxuskit.SystemMessage("You are a helpful programming assistant."),
			nxuskit.UserMessage("What is Go and why should I use it?"),
		),
		nxuskit.WithTemperature(0.7),
		nxuskit.WithMaxTokens(500),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Send request
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Verify response
	if resp.Content == "" {
		t.Error("Expected non-empty response content")
	}
	if resp.Model != "mock-model" {
		t.Errorf("Expected model 'mock-model', got '%s'", resp.Model)
	}

	// Verify token usage
	if resp.Usage.TotalTokens() == 0 {
		t.Error("Expected non-zero token usage")
	}

	// Verify request was recorded
	recorded := provider.GetRecordedRequests()
	if len(recorded) != 1 {
		t.Errorf("Expected 1 recorded request, got %d", len(recorded))
	}
	if len(recorded[0].Messages) != 2 {
		t.Errorf("Expected 2 messages (system + user), got %d", len(recorded[0].Messages))
	}
}

func TestBasicChat_RequestParameters(t *testing.T) {
	ctx := context.Background()
	provider := nxuskit.NewMockProvider()

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(
			nxuskit.UserMessage("Hello"),
		),
		nxuskit.WithTemperature(0.5),
		nxuskit.WithMaxTokens(100),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Verify request parameters were captured
	recorded := provider.GetRecordedRequests()
	if len(recorded) != 1 {
		t.Fatalf("Expected 1 recorded request, got %d", len(recorded))
	}

	r := recorded[0]
	if r.Model != "test-model" {
		t.Errorf("Expected model 'test-model', got '%s'", r.Model)
	}
	if r.Temperature == nil || *r.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %v", r.Temperature)
	}
	if r.MaxTokens == nil || *r.MaxTokens != 100 {
		t.Errorf("Expected max_tokens 100, got %v", r.MaxTokens)
	}
}
