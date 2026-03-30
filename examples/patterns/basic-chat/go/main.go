// Example: Basic Chat
//
// ## nxusKit Features Demonstrated
// - Unified provider interface (LLMProvider interface)
// - Functional options pattern for provider configuration
// - Consistent error handling across providers
// - Normalized token tracking (Usage.TotalTokens)
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each API call with explanations
//
// ## Why This Pattern Matters
// This is the foundational pattern for all LLM interactions. nxusKit provides
// a consistent API across providers (Claude, OpenAI, Ollama) so you can switch
// providers without changing your application code.
//
// Usage:
//
//	export ANTHROPIC_API_KEY="your-key-here"
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
// Or with Ollama (no API key needed):
//
//	go run .  # Uses Ollama by default if no cloud API key is set
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

	fmt.Println("=== Basic Chat Example ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Getting API key
	if config.StepPause("Checking for API keys...", []string{
		"Checks ANTHROPIC_API_KEY, OPENAI_API_KEY, or falls back to Ollama",
		"This keeps secrets out of source code",
	}) == interactive.ActionQuit {
		return
	}

	// Try to create a provider based on available configuration
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
		"nxusKit: Functional options pattern for provider configuration",
		"The provider abstraction hides provider-specific details",
		"No HTTP connection is made yet - that happens on first request",
	}) == interactive.ActionQuit {
		return
	}

	// Step: Building request
	if config.StepPause("Building chat request...", []string{
		"nxusKit: Fluent request builder with functional options",
		"Messages support system, user, and assistant roles",
		"Parameters like temperature and max_tokens have sensible defaults",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: Fluent request builder with functional options pattern
	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.SystemMessage("You are a helpful programming assistant."),
			nxuskit.UserMessage("What is Go and why should I use it?"),
		),
		nxuskit.WithTemperature(0.7),
		nxuskit.WithMaxTokens(500),
	)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL(provider.ProviderName()), req)

	// Step: Sending request
	if config.StepPause("Sending request to "+provider.ProviderName()+" API...", []string{
		"nxusKit: Unified interface - same pattern works for all providers",
		"The request is serialized to JSON and sent via HTTPS",
		"Response is parsed and normalized to a common format",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Printf("Sending request to %s...\n\n", provider.ProviderName())

	// nxusKit: Unified async interface - same pattern works for all providers
	start := time.Now()
	resp, err := provider.Chat(ctx, req)
	elapsedMs := time.Since(start).Milliseconds()

	if err != nil {
		fmt.Printf("Request failed: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the response
	config.PrintResponse(200, elapsedMs, resp)

	// Display response
	fmt.Printf("Response:\n%s\n\n", resp.Content)
	fmt.Printf("Model: %s\n", resp.Model)
	// nxusKit: Unified token tracking - same format regardless of provider
	fmt.Println("Token usage:")
	fmt.Printf("  Prompt: %d tokens\n", resp.Usage.Estimated.PromptTokens)
	fmt.Printf("  Completion: %d tokens\n", resp.Usage.Estimated.CompletionTokens)
	fmt.Printf("  Total: %d tokens\n", resp.Usage.TotalTokens())
}

// createProvider creates an LLM provider based on available configuration.
// It checks for API keys in order of preference: Anthropic, OpenAI, then Ollama.
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

	// Fall back to Ollama (local, no API key needed)
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
