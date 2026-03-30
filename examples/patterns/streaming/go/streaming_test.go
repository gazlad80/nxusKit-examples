package main

import (
	"context"
	"strings"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestStreaming_WithMockProvider(t *testing.T) {
	ctx := context.Background()

	// Create mock provider with stream chunks
	fr := nxuskit.FinishReasonStop
	chunks := []nxuskit.StreamChunk{
		{Delta: "Hello "},
		{Delta: "from "},
		{Delta: "streaming!", FinishReason: &fr},
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockStreamResponse(chunks))

	// Create request
	req, err := nxuskit.NewChatRequest("mock-model",
		nxuskit.WithMessages(
			nxuskit.UserMessage("Say hello"),
		),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Stream response
	chunkCh, errCh := provider.ChatStream(ctx, req)

	var fullContent strings.Builder
	chunkCount := 0

	for chunk := range chunkCh {
		fullContent.WriteString(chunk.Delta)
		chunkCount++
	}

	// Check for errors
	if err := <-errCh; err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	// Verify chunks were received
	if chunkCount != 3 {
		t.Errorf("Expected 3 chunks, got %d", chunkCount)
	}

	// Verify full content
	expected := "Hello from streaming!"
	if fullContent.String() != expected {
		t.Errorf("Expected content '%s', got '%s'", expected, fullContent.String())
	}
}

func TestStreaming_FinishReason(t *testing.T) {
	ctx := context.Background()

	fr := nxuskit.FinishReasonStop
	chunks := []nxuskit.StreamChunk{
		{Delta: "Test"},
		{Delta: " response", FinishReason: &fr},
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockStreamResponse(chunks))

	req, _ := nxuskit.NewChatRequest("mock-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)

	chunkCh, errCh := provider.ChatStream(ctx, req)

	var lastFinishReason *nxuskit.FinishReason
	for chunk := range chunkCh {
		if chunk.FinishReason != nil {
			lastFinishReason = chunk.FinishReason
		}
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	if lastFinishReason == nil {
		t.Error("Expected finish reason in last chunk")
	} else if *lastFinishReason != nxuskit.FinishReasonStop {
		t.Errorf("Expected FinishReasonStop, got %v", *lastFinishReason)
	}
}

func TestStreaming_EmptyStream(t *testing.T) {
	ctx := context.Background()

	// Empty stream chunks
	chunks := []nxuskit.StreamChunk{}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockStreamResponse(chunks))

	req, _ := nxuskit.NewChatRequest("mock-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)

	chunkCh, errCh := provider.ChatStream(ctx, req)

	chunkCount := 0
	for range chunkCh {
		chunkCount++
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Stream error: %v", err)
	}

	if chunkCount != 0 {
		t.Errorf("Expected 0 chunks for empty stream, got %d", chunkCount)
	}
}
