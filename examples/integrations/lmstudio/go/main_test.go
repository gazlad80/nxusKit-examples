//go:build nxuskit

package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// TestLMStudioExampleCompiles verifies the example code compiles and uses valid APIs.
func TestLMStudioExampleCompiles(t *testing.T) {
	// Verify FFI provider creation compiles
	_, err := nxuskit.NewLmStudioFFIProvider()
	if err != nil {
		t.Logf("Provider creation: %v", err)
	}

	// Verify FFI provider with options compiles
	_, err = nxuskit.NewLmStudioFFIProvider(
		nxuskit.WithLmStudioBaseURL("http://localhost:1234"),
	)
	if err != nil {
		t.Logf("Provider with options: %v", err)
	}
}

func TestLMStudioAPIUsage(t *testing.T) {
	// Test that the APIs used in the example are valid

	// NewChatRequest with options
	req, err := nxuskit.NewChatRequest("local-model",
		nxuskit.WithMessages(
			nxuskit.SystemMessage("You are a helpful assistant."),
			nxuskit.UserMessage("Hello"),
		),
	)
	if err != nil {
		t.Fatalf("NewChatRequest failed: %v", err)
	}
	if req.Model != "local-model" {
		t.Errorf("expected model local-model, got %s", req.Model)
	}
	if len(req.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(req.Messages))
	}
}

func TestLMStudioProviderInterface(t *testing.T) {
	// Verify LmStudioProvider implements LLMProvider interface
	var _ nxuskit.LLMProvider = (*nxuskit.LmStudioProvider)(nil)
}

func TestLMStudioCapabilities(t *testing.T) {
	provider, err := nxuskit.NewLmStudioFFIProvider()
	if err != nil {
		t.Skipf("FFI provider not available: %v", err)
	}

	caps := provider.GetCapabilities()

	// LM Studio uses OpenAI-compatible API
	if !caps.SupportsStreaming {
		t.Error("LM Studio should support streaming")
	}
	if !caps.SupportsSystemMessages {
		t.Error("LM Studio should support system messages")
	}
}

func TestMessageCreation(t *testing.T) {
	// Test message creation functions used in examples
	sysMsg := nxuskit.SystemMessage("You are helpful")
	if sysMsg.Role != nxuskit.RoleSystem {
		t.Errorf("expected system role, got %s", sysMsg.Role)
	}

	userMsg := nxuskit.UserMessage("Hello")
	if userMsg.Role != nxuskit.RoleUser {
		t.Errorf("expected user role, got %s", userMsg.Role)
	}

	assistantMsg := nxuskit.AssistantMessage("Hi there")
	if assistantMsg.Role != nxuskit.RoleAssistant {
		t.Errorf("expected assistant role, got %s", assistantMsg.Role)
	}
}

func TestStreamingAPI(t *testing.T) {
	// Verify streaming types compile correctly
	var chunks <-chan nxuskit.StreamChunk
	var errs <-chan error

	// These would be returned by ChatStream
	_ = chunks
	_ = errs

	// Verify StreamWithUsage return types
	var usage <-chan nxuskit.TokenUsage
	_ = usage
}

func TestTokenUsageAccess(t *testing.T) {
	usage := nxuskit.TokenUsage{
		Estimated: nxuskit.TokenCount{
			PromptTokens:     100,
			CompletionTokens: 50,
		},
		IsComplete: true,
	}

	if usage.TotalTokens() != 150 {
		t.Errorf("expected 150 total tokens, got %d", usage.TotalTokens())
	}

	if !usage.IsComplete {
		t.Error("IsComplete should be true")
	}
}

// TestLMStudioIntegration runs actual API calls if LM Studio is running
func TestLMStudioIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	provider, err := nxuskit.NewLmStudioFFIProvider()
	if err != nil {
		t.Skipf("FFI provider not available: %v", err)
	}

	// Try to ping - skip if LM Studio not running
	if err := provider.Ping(context.Background()); err != nil {
		t.Skipf("LM Studio not available: %v", err)
	}

	// List models
	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	t.Logf("Found %d models", len(models))
	for _, m := range models {
		t.Logf("  - %s", m.Name)
	}
}
