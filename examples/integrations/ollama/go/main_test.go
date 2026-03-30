//go:build nxuskit

package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// TestOllamaExampleCompiles verifies the example code compiles and uses valid APIs.
// This doesn't require a running Ollama instance - it just validates the API usage.
func TestOllamaExampleCompiles(t *testing.T) {
	// Verify FFI provider creation compiles
	_, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		// This is expected if Ollama isn't running - we're just testing compilation
		t.Logf("Provider creation (expected to work): %v", err)
	}

	// Verify FFI provider with options compiles
	_, err = nxuskit.NewOllamaFFIProvider(
		nxuskit.WithOllamaBaseURL("http://localhost:11434"),
	)
	if err != nil {
		t.Logf("Provider with options: %v", err)
	}
}

func TestOllamaAPIUsage(t *testing.T) {
	// Test that the APIs used in main.go are valid

	// NewChatRequest with options
	req, err := nxuskit.NewChatRequest("llama3.2",
		nxuskit.WithMessages(
			nxuskit.UserMessage("Hello"),
		),
	)
	if err != nil {
		t.Fatalf("NewChatRequest failed: %v", err)
	}
	if req.Model != "llama3.2" {
		t.Errorf("expected model llama3.2, got %s", req.Model)
	}

	// WithThinkingMode option
	req, err = nxuskit.NewChatRequest("qwen3:latest",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
		nxuskit.WithThinkingMode(nxuskit.ThinkingModeEnabled),
	)
	if err != nil {
		t.Fatalf("NewChatRequest with thinking mode failed: %v", err)
	}
	if req.ThinkingMode != nxuskit.ThinkingModeEnabled {
		t.Errorf("expected ThinkingModeEnabled, got %v", req.ThinkingMode)
	}
}

func TestOllamaProviderInterface(t *testing.T) {
	// Verify OllamaProvider implements LLMProvider interface
	var _ nxuskit.LLMProvider = (*nxuskit.OllamaProvider)(nil)

	// Verify OllamaProvider implements CapabilityDetector interface
	var _ nxuskit.CapabilityDetector = (*nxuskit.OllamaProvider)(nil)
}

func TestModelInfoMethods(t *testing.T) {
	// Test FormattedSize used in example
	size := int64(3_700_000_000) // 3.7GB
	info := nxuskit.ModelInfo{
		Name:      "llama3.2",
		SizeBytes: &size,
	}

	formatted := info.FormattedSize()
	if formatted == "" {
		t.Error("FormattedSize returned empty string")
	}
}

func TestStreamChunkMethods(t *testing.T) {
	// Test IsFinal used in example
	chunk := nxuskit.NewStreamChunk("hello")
	if chunk.IsFinal() {
		t.Error("regular chunk should not be final")
	}

	finalChunk := nxuskit.FinalChunk("done", nxuskit.FinishReasonStop, nil)
	if !finalChunk.IsFinal() {
		t.Error("final chunk should be final")
	}
}

func TestChatResponseFields(t *testing.T) {
	// Test fields accessed in example
	resp := &nxuskit.ChatResponse{
		Content: "test response",
		Usage: nxuskit.TokenUsage{
			Actual: &nxuskit.TokenCount{
				PromptTokens:     10,
				CompletionTokens: 20,
			},
		},
	}

	if resp.Content != "test response" {
		t.Error("Content field access failed")
	}

	if resp.Usage.Actual == nil {
		t.Error("Usage.Actual should be set")
	}

	if resp.Usage.Actual.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.Usage.Actual.PromptTokens)
	}
}

func TestThinkingResponse(t *testing.T) {
	// Test Thinking field used in example
	thinking := "Let me solve this step by step..."
	resp := &nxuskit.ChatResponse{
		Content:  "x = 5",
		Thinking: &thinking,
	}

	if resp.Thinking == nil {
		t.Error("Thinking should be set")
	}
	if *resp.Thinking != thinking {
		t.Errorf("expected thinking %q, got %q", thinking, *resp.Thinking)
	}
}

// TestOllamaIntegration runs actual API calls if OLLAMA_INTEGRATION=1 is set
func TestOllamaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Try to ping - skip if Ollama not running
	if err := provider.Ping(context.Background()); err != nil {
		t.Skipf("Ollama not available: %v", err)
	}

	// List models
	models, err := provider.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	t.Logf("Found %d models", len(models))
	for _, m := range models {
		t.Logf("  - %s (%s)", m.Name, m.FormattedSize())
	}
}
