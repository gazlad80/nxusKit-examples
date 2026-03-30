// Example: Multi-Provider Fallback
//
// ## nxusKit Features Demonstrated
// - Provider failover chains ([]nxuskit.LLMProvider)
// - Unified error handling across providers
// - Resilient request handling with automatic retry
// - Provider-agnostic fallback logic
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each step with explanations
//
// ## Why This Pattern Matters
// Production systems need resilience. nxusKit's interface-based design enables
// easy construction of fallback chains - if one provider fails, the request
// automatically routes to the next available provider.
//
// Usage:
//
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
//go:build nxuskit

// Demonstrates automatic failover between LLM providers.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// Global config for interactive mode
var config *interactive.Config

func main() {
	// Parse interactive mode flags
	config = interactive.FromArgs()

	fmt.Println("=== Multi-Provider Fallback Demo ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Creating providers
	if config.StepPause("Creating multiple provider instances...", []string{
		"nxusKit: Creating 3 Ollama providers for fallback chain",
		"In production, you might use different providers (OpenAI, Claude, Ollama)",
		"NewOllamaProvider() reads OLLAMA_HOST env var automatically",
	}) == interactive.ActionQuit {
		return
	}

	// Create multiple provider instances
	// In production, you might use different providers (OpenAI, Claude, Ollama)
	// NewOllamaProvider() reads OLLAMA_HOST env var automatically
	provider1, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		fmt.Printf("Failed to create provider 1: %v\n", err)
		os.Exit(1)
	}

	provider2, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		fmt.Printf("Failed to create provider 2: %v\n", err)
		os.Exit(1)
	}

	provider3, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		fmt.Printf("Failed to create provider 3: %v\n", err)
		os.Exit(1)
	}

	// Step: Building fallback chain
	if config.StepPause("Building fallback chain...", []string{
		"nxusKit: Interfaces enable heterogeneous provider collections",
		"All providers implement the same LLMProvider interface",
		"Chain will try providers in order until one succeeds",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: Interfaces enable heterogeneous provider collections
	providers := []nxuskit.LLMProvider{provider1, provider2, provider3}

	// Step: Building request
	if config.StepPause("Building chat request...", []string{
		"nxusKit: Same request works with any provider in the chain",
		"Request is provider-agnostic",
	}) == interactive.ActionQuit {
		return
	}

	// Create a simple request
	req, err := nxuskit.NewChatRequest("llama3",
		nxuskit.WithMessages(nxuskit.UserMessage("What is 2 + 2? Answer briefly.")),
	)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("ollama"), req)

	// Step: Sending with fallback
	if config.StepPause("Sending request with fallback chain...", []string{
		"nxusKit: ChatWithFallback tries providers in order until one succeeds",
		"If a provider fails, it automatically tries the next one",
		"Provides resilience for production systems",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Println("Sending request with 3-provider fallback chain...")
	fmt.Println()

	// nxusKit: ChatWithFallback tries providers in order until one succeeds
	start := time.Now()
	resp, err := ChatWithFallback(ctx, providers, req)
	elapsedMs := time.Since(start).Milliseconds()

	if err != nil {
		fmt.Println("\n=== All Providers Failed ===")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the response
	config.PrintResponse(200, elapsedMs, resp)

	fmt.Println("\n=== Success ===")
	fmt.Printf("Response: %s\n", resp.Content)
}

// getProviderURL returns the API URL for verbose output based on provider name.
func getProviderURL(providerName string) string {
	switch providerName {
	case "ollama":
		return "http://localhost:11434/api/chat"
	default:
		return "https://api.example.com/chat"
	}
}
