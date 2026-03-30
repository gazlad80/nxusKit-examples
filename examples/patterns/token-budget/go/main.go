// Example: Streaming with Token Budget
//
// ## nxusKit Features Demonstrated
// - Stream cancellation with budget tracking
// - Normalized token counting across providers
// - Graceful stream termination
// - Cost estimation during streaming
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each step with explanations
//
// ## Why This Pattern Matters
// Token budgets enable cost control and prevent runaway API costs.
// nxusKit's streaming interface supports cancellation at any point,
// and provides estimated token counts even for partial responses.
//
// Usage:
//
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
//go:build nxuskit

// Demonstrates cost control by limiting tokens during streaming.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()

	fmt.Println("=== Streaming with Token Budget Demo ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Creating provider
	if config.StepPause("Creating Ollama provider...", []string{
		"nxusKit: NewOllamaProvider() reads OLLAMA_HOST env var automatically",
		"Local provider - no API key needed",
		"Streaming is supported for token-by-token output",
	}) == interactive.ActionQuit {
		return
	}

	// NewOllamaProvider() reads OLLAMA_HOST env var automatically
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		fmt.Printf("Failed to create provider: %v\n", err)
		os.Exit(1)
	}

	// Step: Building request
	if config.StepPause("Building chat request...", []string{
		"nxusKit: Standard request builder pattern",
		"Requesting creative content that would normally be long",
		"We'll cut it short with a token budget",
	}) == interactive.ActionQuit {
		return
	}

	req, err := nxuskit.NewChatRequest("llama3",
		nxuskit.WithMessages(nxuskit.UserMessage("Write a short story about a robot learning to paint. Be creative!")),
	)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("ollama"), req)

	// nxusKit: Budget limit - stream will be cancelled when exceeded
	maxTokens := 50

	// Step: Setting up budget
	if config.StepPause("Setting up token budget...", []string{
		fmt.Sprintf("Budget limit: %d tokens", maxTokens),
		"nxusKit: StreamWithBudget tracks token usage in real-time",
		"Stream will be cancelled when budget is exceeded",
		"Enables cost control for streaming responses",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Printf("Token budget: %d tokens\n", maxTokens)
	fmt.Println("Streaming response...")
	fmt.Println()

	// Step: Starting stream
	if config.StepPause("Starting budget-limited stream...", []string{
		"nxusKit: StreamWithBudget wraps the stream and tracks token usage",
		"Tokens are estimated as chunks arrive",
		"Stream automatically cancels when budget is reached",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: StreamWithBudget wraps the stream and tracks token usage
	result, err := StreamWithBudget(ctx, provider, req, maxTokens)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the result
	config.PrintResponse(200, 0, result)

	fmt.Println("\n=== Result ===")
	fmt.Printf("Content: %s\n", result.Content)
	fmt.Printf("Estimated tokens: %d\n", result.EstimatedTokens)
	budgetStr := "No"
	if result.BudgetReached {
		budgetStr = "Yes"
	}
	fmt.Printf("Budget reached: %s\n", budgetStr)
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
