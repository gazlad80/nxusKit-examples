// Example: Timeout Configuration
//
// ## nxusKit Features Demonstrated
// - Context-based timeout control (Go idiomatic pattern)
// - WithTimeout option for convenience API
// - Provider-specific timeout recommendations
// - Graceful timeout handling
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each API call with explanations
//
// ## Why This Pattern Matters
// Network conditions and response times vary. Go's context.Context provides
// natural timeout control. nxusKit integrates with this pattern while also
// providing convenience options for simpler use cases.
//
// Usage:
//
//	export ANTHROPIC_API_KEY="your-key-here"
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
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

	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY environment variable not set")
		os.Exit(1)
	}

	fmt.Println("=== Timeout Configuration Examples ===")
	fmt.Println()

	// Step: Introduction
	if config.StepPause("Understanding timeout configuration...", []string{
		"nxusKit: Go uses context.Context for timeout control",
		"This is different from Rust's builder pattern",
		"Timeouts protect against slow networks and unresponsive servers",
	}) == interactive.ActionQuit {
		return
	}

	// Example 1: Using default timeouts
	fmt.Println("1. Using default timeouts:")

	// Step: Default timeout
	if config.StepPause("Creating provider with default timeout...", []string{
		"nxusKit: NewClaudeProvider uses nxuskit.DefaultTimeout (60s)",
		"This is suitable for most chat interactions",
		"No explicit timeout configuration needed",
	}) == interactive.ActionQuit {
		return
	}

	_, err := nxuskit.NewClaudeFFIProvider(nxuskit.WithClaudeAPIKey(apiKey))
	if err != nil {
		fmt.Printf("   Error creating provider: %v\n", err)
		return
	}
	fmt.Println("   Default timeout: 60s (from nxuskit.DefaultTimeout)")
	fmt.Println()

	// Example 2: Using context with timeout
	fmt.Println("2. Using context with timeout (Go idiomatic approach):")
	fmt.Println("   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)")
	fmt.Println("   provider.Chat(ctx, req)")
	fmt.Println("   // Context controls the timeout")
	fmt.Println()

	// Example 3: Using WithTimeout option in convenience API
	fmt.Println("3. Using WithTimeout option (convenience API):")
	fmt.Println("   nxuskit.Completion(ctx, model, prompt,")
	fmt.Println("       nxuskit.WithTimeout(30*time.Second),")
	fmt.Println("   )")
	fmt.Println()

	// Example 4: Different timeouts for different use cases
	fmt.Println("4. Timeout recommendations by use case:")
	fmt.Println("   - Quick queries (simple questions): 30s")
	fmt.Println("   - Standard chat: 60s (default)")
	fmt.Println("   - Streaming responses: 120-300s")
	fmt.Println("   - Ollama (local, may need model load): 120-180s")
	fmt.Println()

	// Step: Real request
	if config.StepPause("Testing with a real request...", []string{
		"nxusKit: Using context.WithTimeout for 30-second timeout",
		"The context is passed to provider.Chat()",
		"If timeout expires, ctx.Err() returns context.DeadlineExceeded",
	}) == interactive.ActionQuit {
		return
	}

	// Example 5: Test with a real request using context timeout
	fmt.Println("5. Testing with a real request:")
	fmt.Println("   Using 30-second timeout...")

	provider, err := nxuskit.NewClaudeFFIProvider(nxuskit.WithClaudeAPIKey(apiKey))
	if err != nil {
		fmt.Printf("   Error: %v\n", err)
		return
	}

	// nxusKit: Go's context.WithTimeout is the idiomatic timeout pattern
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := nxuskit.NewChatRequest("claude-3-5-haiku-20241022",
		nxuskit.WithMessages(nxuskit.UserMessage("What is Go in one sentence?")),
		nxuskit.WithMaxTokens(100),
	)
	if err != nil {
		fmt.Printf("   Error creating request: %v\n", err)
		return
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("claude"), req)

	start := time.Now()
	resp, err := provider.Chat(ctx, req)
	elapsedMs := time.Since(start).Milliseconds()

	if err != nil {
		fmt.Printf("   Request failed: %v\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)

		fmt.Println("   Request succeeded!")
		fmt.Printf("   Response: %s\n", resp.Content)
		fmt.Printf("   Tokens used: %d\n", resp.Usage.TotalTokens())
	}

	// Step: Best practices
	if config.StepPause("Reviewing best practices...", []string{
		"Use context.WithTimeout for fine-grained control",
		"Set shorter timeouts (10-30s) to detect network issues quickly",
		"Set longer timeouts (120-300s) for streaming or large responses",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Println()
	fmt.Println("=== Configuration Best Practices ===")
	fmt.Println()
	fmt.Println("Go uses context.Context for timeout control (different from Rust's builder pattern):")
	fmt.Println()
	fmt.Println("  // Method 1: Context with timeout")
	fmt.Println("  ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)")
	fmt.Println("  defer cancel()")
	fmt.Println("  provider.Chat(ctx, req)")
	fmt.Println()
	fmt.Println("  // Method 2: Convenience API with WithTimeout option")
	fmt.Println("  nxuskit.Completion(ctx, model, prompt,")
	fmt.Println("      nxuskit.WithTimeout(30*time.Second),")
	fmt.Println("  )")
	fmt.Println()
	fmt.Println("Tips:")
	fmt.Println("  - Use context.WithTimeout for fine-grained control")
	fmt.Println("  - Set shorter timeouts (10-30s) to detect network issues quickly")
	fmt.Println("  - Set longer timeouts (120-300s) for streaming or large responses")
	fmt.Println("  - For Ollama (local), use longer timeouts (may need model loading)")
	fmt.Println("  - For Claude/OpenAI (remote), shorter timeouts work well (60s default)")
	fmt.Println()
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
