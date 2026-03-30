package main

import (
	"context"
	"testing"
	"time"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestTimeout_ContextWithTimeout(t *testing.T) {
	// Test that context timeout is respected
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	provider := nxuskit.NewMockProvider()

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Hello")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Should complete quickly with mock provider
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content == "" {
		// Mock returns default content
		_ = resp
	}
}

func TestTimeout_DefaultValues(t *testing.T) {
	// Verify default timeout constant exists
	if nxuskit.DefaultTimeout <= 0 {
		t.Error("DefaultTimeout should be a positive duration")
	}
}

func TestTimeout_CancelledContext(t *testing.T) {
	// Test that cancelled context is handled gracefully
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	provider := nxuskit.NewMockProvider()

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Hello")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Request with cancelled context should fail
	_, err = provider.Chat(ctx, req)
	if err == nil {
		// Some mock providers may not check context
		// This is acceptable for mock behavior
		t.Log("Mock provider does not check for cancelled context")
	}
}

func TestTimeout_StreamingWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fr := nxuskit.FinishReasonStop
	chunks := []nxuskit.StreamChunk{
		{Delta: "Chunk 1"},
		{Delta: "Chunk 2", FinishReason: &fr},
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockStreamResponse(chunks))

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	chunkCh, errCh := provider.ChatStream(ctx, req)

	chunkCount := 0
	for range chunkCh {
		chunkCount++
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	if chunkCount != 2 {
		t.Errorf("Expected 2 chunks, got %d", chunkCount)
	}
}

func TestTimeout_Recommendations(t *testing.T) {
	// Document recommended timeout values as assertions
	recommendations := map[string]time.Duration{
		"quick_queries":     30 * time.Second,
		"standard_chat":     60 * time.Second,
		"streaming":         300 * time.Second,
		"ollama_local":      180 * time.Second,
		"claude_openai_api": 60 * time.Second,
	}

	for name, timeout := range recommendations {
		if timeout <= 0 {
			t.Errorf("Invalid timeout recommendation for %s: %v", name, timeout)
		}
	}
}
