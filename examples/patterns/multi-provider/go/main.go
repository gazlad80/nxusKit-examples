// Example: Multi-Provider Comparison
//
// ## nxusKit Features Demonstrated
// - Provider abstraction layer (LLMProvider interface)
// - Concurrent request execution with goroutines
// - Unified response structure across different providers
// - Provider-agnostic error handling
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each step with explanations
//
// ## Why This Pattern Matters
// Running the same prompt across providers enables A/B testing, cost comparison,
// and fallback strategies. nxusKit's unified interface makes this trivial -
// all providers return the same ChatResponse type with normalized token usage.
//
// Usage:
//
//	export ANTHROPIC_API_KEY="your-key-here"
//	export OPENAI_API_KEY="your-key-here"
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
// Note: Providers with missing API keys will show errors but won't stop
// other providers from responding.
//
//go:build nxuskit

// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// providerResult holds the result from a single provider call.
type providerResult struct {
	name     string
	response *nxuskit.ChatResponse
	err      error
}

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()

	fmt.Println("=== Multi-Provider Comparison Example ===")
	fmt.Println()

	ctx := context.Background()
	question := "In one sentence, what makes Go unique among programming languages?"

	// Step: Setting up providers
	if config.StepPause("Creating multiple LLM providers...", []string{
		"nxusKit: Each provider uses the same factory pattern",
		"Providers are created independently and can fail gracefully",
		"All providers implement the same LLMProvider interface",
	}) == interactive.ActionQuit {
		return
	}

	// Create providers (errors are captured, not fatal)
	providers := createProviders()

	if len(providers) == 0 {
		fmt.Println("No providers available!")
		fmt.Println("\nTo use this example, configure at least one provider:")
		fmt.Println("  - Set ANTHROPIC_API_KEY for Claude")
		fmt.Println("  - Set OPENAI_API_KEY for OpenAI")
		fmt.Println("  - Start Ollama locally (ollama serve)")
		os.Exit(1)
	}

	// Step: Building requests
	if config.StepPause("Building identical requests for each provider...", []string{
		"nxusKit: Request structure is provider-agnostic",
		"Only the model name differs between providers",
		"Same parameters work across all providers",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Printf("Question: %s\n\n", question)
	fmt.Println(strings.Repeat("=", 80))

	// Step: Sending concurrent requests
	if config.StepPause("Sending concurrent requests to all providers...", []string{
		"nxusKit: Unified interface enables easy concurrent execution with goroutines",
		"Each request runs in its own goroutine",
		"Results are collected via channels",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: Unified interface enables easy concurrent execution with goroutines
	results := make(chan providerResult, len(providers))
	var wg sync.WaitGroup

	for name, p := range providers {
		wg.Add(1)
		go func(providerName string, provider nxuskit.LLMProvider, model string) {
			defer wg.Done()

			req, err := nxuskit.NewChatRequest(model,
				nxuskit.WithMessages(nxuskit.UserMessage(question)),
				nxuskit.WithTemperature(0.5),
				nxuskit.WithMaxTokens(100),
			)
			if err != nil {
				results <- providerResult{name: providerName, err: err}
				return
			}

			// Verbose: Show request
			config.PrintRequest("POST", getProviderURL(providerName), req)

			resp, err := provider.Chat(ctx, req)
			results <- providerResult{name: providerName, response: resp, err: err}
		}(name, p.provider, p.model)
	}

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Display results as they arrive
	for result := range results {
		fmt.Println()
		if result.err != nil {
			fmt.Printf("%s: Error - %v\n", result.name, result.err)
		} else {
			// Verbose: Show response
			config.PrintResponse(200, 0, result.response)

			fmt.Printf("%s (%s)\n", result.name, result.response.Model)
			fmt.Printf("%s\n", result.response.Content)
			fmt.Printf("Tokens: %d\n", result.response.Usage.TotalTokens())
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
}

type providerInfo struct {
	provider nxuskit.LLMProvider
	model    string
}

// createProviders creates all available providers.
func createProviders() map[string]providerInfo {
	providers := make(map[string]providerInfo)

	// Try Claude
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		provider, err := nxuskit.NewClaudeFFIProvider(nxuskit.WithClaudeAPIKey(apiKey))
		if err == nil {
			providers["Claude"] = providerInfo{provider: provider, model: "claude-haiku-4-5-20251001"}
		}
	}

	// Try OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		provider, err := nxuskit.NewOpenAIFFIProvider(nxuskit.WithOpenAIAPIKey(apiKey))
		if err == nil {
			providers["OpenAI"] = providerInfo{provider: provider, model: "gpt-4o"}
		}
	}

	// Try Ollama (always attempt, it's local)
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err == nil {
		providers["Ollama"] = providerInfo{provider: provider, model: "llama3"}
	}

	return providers
}

// getProviderURL returns the API URL for verbose output based on provider name.
func getProviderURL(providerName string) string {
	switch providerName {
	case "Claude":
		return "https://api.anthropic.com/v1/messages"
	case "OpenAI":
		return "https://api.openai.com/v1/chat/completions"
	case "Ollama":
		return "http://localhost:11434/api/chat"
	default:
		return "https://api.example.com/chat"
	}
}
