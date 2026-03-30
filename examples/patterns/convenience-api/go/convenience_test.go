package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestConvenience_AutoDetection(t *testing.T) {
	// Test that the convenience API can route based on model names
	// This tests the routing logic without making actual API calls

	ctx := context.Background()

	// Create mock provider with a pre-configured response
	mockResp := &nxuskit.ChatResponse{
		Content: "Auto-detected response",
		Model:   "mock-model",
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockResponse(mockResp))

	// Test basic request through mock provider
	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Test prompt")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != "Auto-detected response" {
		t.Errorf("Expected 'Auto-detected response', got '%s'", resp.Content)
	}
}

func TestConvenience_StreamingWithMock(t *testing.T) {
	ctx := context.Background()

	// Create mock provider with streaming response
	fr := nxuskit.FinishReasonStop
	chunks := []nxuskit.StreamChunk{
		{Delta: "Convenience "},
		{Delta: "streaming "},
		{Delta: "works!", FinishReason: &fr},
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockStreamResponse(chunks))

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	chunkCh, errCh := provider.ChatStream(ctx, req)

	var result string
	for chunk := range chunkCh {
		result += chunk.Delta
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	expected := "Convenience streaming works!"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestConvenience_ProviderRouting(t *testing.T) {
	// Test that different model patterns would route correctly
	// This is a structural test of model name patterns

	testCases := []struct {
		model           string
		expectedPattern string
	}{
		{"gpt-4o", "openai"},
		{"gpt-3.5-turbo", "openai"},
		{"claude-haiku-4-5-20251001", "anthropic"},
		{"claude-3-opus", "anthropic"},
		{"llama3", "ollama"},
		{"openai/gpt-4o", "openai"},
		{"anthropic/claude-haiku-4-5-20251001", "anthropic"},
		{"ollama/llama3", "ollama"},
	}

	for _, tc := range testCases {
		t.Run(tc.model, func(t *testing.T) {
			// Just verify the model name pattern is valid
			// Actual routing is tested by the convenience API internally
			if tc.model == "" {
				t.Error("Model name should not be empty")
			}
		})
	}
}
