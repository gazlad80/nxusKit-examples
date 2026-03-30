package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestVision_ImageAttachment(t *testing.T) {
	ctx := context.Background()

	// Create mock provider
	mockResp := &nxuskit.ChatResponse{
		Content: "I see a programming logo in the image.",
		Model:   "vision-model",
		Usage: nxuskit.TokenUsage{
			Actual: &nxuskit.TokenCount{
				PromptTokens:     100,
				CompletionTokens: 20,
			},
		},
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockResponse(mockResp))

	// Create message with image attachment
	msg := nxuskit.UserMessage("What's in this image?").
		WithImageURL("https://example.com/test-image.png")

	req, err := nxuskit.NewChatRequest("vision-model",
		nxuskit.WithMessages(msg),
		nxuskit.WithMaxTokens(300),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content == "" {
		t.Error("Expected non-empty response for image analysis")
	}

	// Verify request was recorded with image
	recorded := provider.GetRecordedRequests()
	if len(recorded) != 1 {
		t.Fatalf("Expected 1 recorded request, got %d", len(recorded))
	}
}

func TestVision_MultipleImages(t *testing.T) {
	ctx := context.Background()

	mockResp := &nxuskit.ChatResponse{
		Content: "I see two different logos.",
		Model:   "vision-model",
	}
	provider := nxuskit.NewMockProvider(nxuskit.WithMockResponse(mockResp))

	// Create message with multiple images
	msg := nxuskit.UserMessage("Compare these two images.").
		WithImageURL("https://example.com/image1.png").
		WithImageURL("https://example.com/image2.png")

	req, err := nxuskit.NewChatRequest("vision-model",
		nxuskit.WithMessages(msg),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content == "" {
		t.Error("Expected non-empty response")
	}
}

func TestVision_DetailLevels(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		detail string
	}{
		{"low"},
		{"high"},
		{"auto"},
	}

	for _, tc := range testCases {
		t.Run(tc.detail, func(t *testing.T) {
			provider := nxuskit.NewMockProvider()

			msg := nxuskit.UserMessage("Analyze this image.").
				WithImageURL("https://example.com/test.png").
				WithDetail(tc.detail)

			req, err := nxuskit.NewChatRequest("vision-model",
				nxuskit.WithMessages(msg),
			)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			_, err = provider.Chat(ctx, req)
			if err != nil {
				t.Fatalf("Chat failed with detail level '%s': %v", tc.detail, err)
			}
		})
	}
}

func TestVision_ModelCapabilityCheck(t *testing.T) {
	ctx := context.Background()

	// Create mock provider that returns vision-capable models
	provider := nxuskit.NewMockProvider()

	models, err := provider.ListModels(ctx)
	if err != nil {
		t.Fatalf("Failed to list models: %v", err)
	}

	// Mock provider should return at least one model
	if len(models) == 0 {
		t.Skip("Mock provider returned no models")
	}

	// Check for vision capability method
	for _, model := range models {
		// SupportsVision should be callable
		_ = model.SupportsVision()
	}
}
