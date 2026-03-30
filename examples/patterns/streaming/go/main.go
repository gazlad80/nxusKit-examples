// Example: Streaming Chat
//
// ## nxusKit Features Demonstrated
// - Unified streaming interface across all providers
// - Go channel-based streaming (idiomatic Go pattern)
// - Structured chunk types with delta content and metadata
// - Final chunk detection with accumulated token usage
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw SSE chunks as they arrive
// - `--step` or `-s`: Pause at each step with explanations
//
// ## Why This Pattern Matters
// Streaming enables real-time response display, reducing perceived latency.
// nxusKit normalizes the different streaming formats from Claude, OpenAI,
// and Ollama into a consistent channel-based interface.
//
// Usage:
//
//	export ANTHROPIC_API_KEY="your-key-here"
//	go run .
//	go run . --verbose    # Show SSE chunks
//	go run . --step       # Step through with explanations
//
// Or with Ollama (no API key needed):
//
//	go run .
//
//go:build nxuskit

// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()

	fmt.Println("=== Streaming Chat Example ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Checking API keys
	if config.StepPause("Checking for API keys...", []string{
		"Checks ANTHROPIC_API_KEY, OPENAI_API_KEY, or falls back to Ollama",
		"This keeps secrets out of source code",
	}) == interactive.ActionQuit {
		return
	}

	// Create provider based on available configuration
	provider, model, err := createProvider()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("\nTo fix this, either:")
		fmt.Println("  1. Set ANTHROPIC_API_KEY for Claude")
		fmt.Println("  2. Set OPENAI_API_KEY for OpenAI")
		fmt.Println("  3. Start Ollama locally (ollama serve)")
		os.Exit(1)
	}

	// Step: Creating provider
	if config.StepPause("Creating "+provider.ProviderName()+" provider...", []string{
		"nxusKit: Same pattern as non-streaming requests",
		"Streaming is just a different method call on the provider",
		"No HTTP connection is made yet - that happens on first request",
	}) == interactive.ActionQuit {
		return
	}

	// Step: Building request
	if config.StepPause("Building chat request...", []string{
		"nxusKit: Same request type works for both streaming and non-streaming",
		"The provider determines how to handle the request",
		"Streaming adds a 'stream: true' parameter automatically",
	}) == interactive.ActionQuit {
		return
	}

	// Create request
	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.UserMessage("Write a short poem about Go programming."),
		),
		nxuskit.WithTemperature(0.8),
		nxuskit.WithMaxTokens(300),
	)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL(provider.ProviderName()), req)

	// Step: Starting stream
	if config.StepPause("Starting streaming request...", []string{
		"nxusKit: Unified streaming API - returns channels for chunks and errors",
		"Server-Sent Events (SSE) arrive as they're generated",
		"Go channels provide natural backpressure handling",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Printf("Streaming response from %s:\n\n", provider.ProviderName())

	// nxusKit: Unified streaming API - returns channels for chunks and errors
	start := time.Now()
	chunks, errs := provider.ChatStream(ctx, req)

	var totalUsage *nxuskit.TokenUsage
	chunkCount := 0

	// nxusKit: Standard channel processing - range over chunks as they arrive
	for chunk := range chunks {
		chunkCount++

		// Verbose: Show each SSE chunk
		if chunk.Delta != "" {
			config.PrintStreamChunk(chunkCount, chunk.Delta)
		}

		// nxusKit: Normalized chunk structure - Delta contains new content
		if chunk.Delta != "" {
			fmt.Print(chunk.Delta)
		}

		// nxusKit: IsFinal() detects stream completion across all providers
		if chunk.IsFinal() && chunk.Usage != nil {
			totalUsage = chunk.Usage
		}
	}

	// Check for streaming errors
	if err := <-errs; err != nil {
		fmt.Printf("\n\nStream error: %v\n", err)
		os.Exit(1)
	}

	elapsedMs := time.Since(start).Milliseconds()

	fmt.Println()
	fmt.Println()

	// Verbose: Show stream completion summary
	config.PrintStreamDone(elapsedMs, chunkCount)

	// Display token usage if available
	if totalUsage != nil {
		fmt.Println("Token usage:")
		fmt.Printf("  Prompt: %d tokens\n", totalUsage.Estimated.PromptTokens)
		fmt.Printf("  Completion: %d tokens\n", totalUsage.Estimated.CompletionTokens)
		fmt.Printf("  Total: %d tokens\n", totalUsage.TotalTokens())
	}
}

// createProvider creates an LLM provider based on available configuration.
func createProvider() (nxuskit.LLMProvider, string, error) {
	// Try Anthropic (Claude)
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider, err := nxuskit.NewClaudeFFIProvider(nxuskit.WithClaudeAPIKey(apiKey))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Claude provider: %w", err)
		}
		return provider, "claude-haiku-4-5-20251001", nil
	}

	// Try OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider, err := nxuskit.NewOpenAIFFIProvider(nxuskit.WithOpenAIAPIKey(apiKey))
		if err != nil {
			return nil, "", fmt.Errorf("failed to create OpenAI provider: %w", err)
		}
		return provider, "gpt-4o", nil
	}

	// Fall back to Ollama
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		return nil, "", fmt.Errorf("no provider available: set ANTHROPIC_API_KEY, OPENAI_API_KEY, or start Ollama")
	}
	return provider, "llama3", nil
}

// getProviderURL returns the API URL for verbose output based on provider name.
func getProviderURL(providerName string) string {
	switch providerName {
	case "claude":
		return "https://api.anthropic.com/v1/messages"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	case "ollama":
		return "http://localhost:11434/api/chat"
	default:
		return "https://api.example.com/chat"
	}
}
