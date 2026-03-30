// Example: Vision / Multimodal
//
// ## nxusKit Features Demonstrated
// - Multimodal message construction (text + images)
// - Capability detection (SupportsVision, MaxImages)
// - Provider-specific image handling abstraction
// - URL-based image support with detail level options
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each step with explanations
//
// ## Why This Pattern Matters
// Vision APIs differ significantly between providers (different formats, limits,
// detail levels). nxusKit abstracts these differences while exposing provider-
// specific options (like OpenAI's detail level) through a consistent interface.
//
// Usage:
//
//	# With Claude
//	export ANTHROPIC_API_KEY="your-key-here"
//	go run . claude
//	go run . claude --verbose    # Show request/response details
//	go run . claude --step       # Step through with explanations
//
//	# With OpenAI
//	export OPENAI_API_KEY="your-key-here"
//	go run . openai
//
//go:build nxuskit

// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()
	flag.Parse()

	// Get provider from remaining command line args (after flags are parsed)
	providerName := "claude"
	args := flag.Args()
	if len(args) > 0 {
		providerName = args[0]
	}

	fmt.Printf("Vision Example - Using %s provider\n\n", providerName)

	ctx := context.Background()

	var err error
	switch providerName {
	case "claude":
		err = runClaudeExample(ctx, config)
	case "openai":
		err = runOpenAIExample(ctx, config)
	default:
		fmt.Fprintf(os.Stderr, "Unknown provider: %s. Use 'claude' or 'openai'\n", providerName)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runClaudeExample(ctx context.Context, config *interactive.Config) error {
	// Step: Checking for API key
	if config.StepPause("Checking for Anthropic API key...", []string{
		"Reads ANTHROPIC_API_KEY from environment",
		"This keeps secrets out of source code",
	}) == interactive.ActionQuit {
		return nil
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	// Step: Creating provider
	if config.StepPause("Creating Claude provider...", []string{
		"nxusKit: Functional options pattern for provider configuration",
		"The provider abstraction hides API-specific details",
	}) == interactive.ActionQuit {
		return nil
	}

	provider, err := nxuskit.NewClaudeFFIProvider(nxuskit.WithClaudeAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Claude provider: %w", err)
	}

	// Step: Checking vision capabilities
	if config.StepPause("Checking for vision-capable models...", []string{
		"nxusKit: Capability detection - query models before making requests",
		"SupportsVision() method checks if model can handle images",
	}) == interactive.ActionQuit {
		return nil
	}

	// nxusKit: Capability detection - query models before making requests
	fmt.Println("Checking for vision-capable models...")
	models, err := provider.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	var visionModels []nxuskit.ModelInfo
	for _, m := range models {
		if m.SupportsVision() {
			visionModels = append(visionModels, m)
		}
	}

	if len(visionModels) == 0 {
		return fmt.Errorf("no vision-capable models found")
	}

	fmt.Printf("Found %d vision-capable models:\n", len(visionModels))
	for _, m := range visionModels {
		fmt.Printf("   - %s\n", m.Name)
	}
	fmt.Println()

	// Example 1: Image from URL
	fmt.Println("Example 1: Image from URL")
	fmt.Println(strings.Repeat("-", 40))

	// Step: Building multimodal request
	if config.StepPause("Building multimodal request with image URL...", []string{
		"nxusKit: Fluent API for multimodal messages - chain text and images",
		"WithImageURL() adds an image from a URL to the message",
		"Provider handles fetching and encoding the image",
	}) == interactive.ActionQuit {
		return nil
	}

	// nxusKit: Fluent API for multimodal messages - chain text and images
	msg := nxuskit.UserMessage("What's in this image? Describe it briefly.").
		WithImageURL("https://upload.wikimedia.org/wikipedia/commons/thumb/d/d5/Rust_programming_language_black_logo.svg/800px-Rust_programming_language_black_logo.svg.png")

	req, err := nxuskit.NewChatRequest("claude-haiku-4-5-20251001",
		nxuskit.WithMessages(msg),
		nxuskit.WithMaxTokens(300),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("claude"), req)

	// Step: Sending request
	if config.StepPause("Sending vision request to Claude API...", []string{
		"nxusKit: Same Chat() method works for text and multimodal",
		"The request includes the image URL for Claude to process",
	}) == interactive.ActionQuit {
		return nil
	}

	start := time.Now()
	resp, err := provider.Chat(ctx, req)
	elapsedMs := time.Since(start).Milliseconds()

	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)

		usage := resp.Usage.BestAvailable()
		fmt.Printf("Response: %s\n", resp.Content)
		fmt.Printf("Token usage: %d input, %d output\n\n",
			usage.PromptTokens, usage.CompletionTokens)
	}

	// Example 2: Multiple images for comparison
	fmt.Println("Example 2: Multiple images for comparison")
	fmt.Println(strings.Repeat("-", 40))

	// Step: Building multi-image request
	if config.StepPause("Building request with multiple images...", []string{
		"nxusKit: Chain multiple WithImageURL() calls for comparison tasks",
		"Claude can analyze and compare multiple images in one request",
	}) == interactive.ActionQuit {
		return nil
	}

	msg = nxuskit.UserMessage("Compare these two logos. What do they have in common?").
		WithImageURL("https://upload.wikimedia.org/wikipedia/commons/thumb/d/d5/Rust_programming_language_black_logo.svg/800px-Rust_programming_language_black_logo.svg.png").
		WithImageURL("https://upload.wikimedia.org/wikipedia/commons/thumb/1/18/ISO_C%2B%2B_Logo.svg/800px-ISO_C%2B%2B_Logo.svg.png")

	req, err = nxuskit.NewChatRequest("claude-haiku-4-5-20251001",
		nxuskit.WithMessages(msg),
		nxuskit.WithMaxTokens(300),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("claude"), req)

	// Step: Sending multi-image request
	if config.StepPause("Sending multi-image comparison request...", []string{
		"Claude will analyze both images and find commonalities",
	}) == interactive.ActionQuit {
		return nil
	}

	start = time.Now()
	resp, err = provider.Chat(ctx, req)
	elapsedMs = time.Since(start).Milliseconds()

	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)

		fmt.Printf("Response: %s\n\n", resp.Content)
	}

	return nil
}

func runOpenAIExample(ctx context.Context, config *interactive.Config) error {
	// Step: Checking for API key
	if config.StepPause("Checking for OpenAI API key...", []string{
		"Reads OPENAI_API_KEY from environment",
		"This keeps secrets out of source code",
	}) == interactive.ActionQuit {
		return nil
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Step: Creating provider
	if config.StepPause("Creating OpenAI provider...", []string{
		"nxusKit: Same factory pattern as Claude provider",
		"Provider abstraction hides OpenAI-specific details",
	}) == interactive.ActionQuit {
		return nil
	}

	provider, err := nxuskit.NewOpenAIFFIProvider(nxuskit.WithOpenAIAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create OpenAI provider: %w", err)
	}

	// Step: Checking vision capabilities
	if config.StepPause("Checking for vision-capable models...", []string{
		"nxusKit: Capability detection works across providers",
		"OpenAI's model list may not expose all capabilities",
	}) == interactive.ActionQuit {
		return nil
	}

	// Check for vision-capable models
	fmt.Println("Checking for vision-capable models...")
	models, err := provider.ListModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to list models: %w", err)
	}

	var visionModels []nxuskit.ModelInfo
	for _, m := range models {
		if m.SupportsVision() {
			visionModels = append(visionModels, m)
		}
	}

	if len(visionModels) == 0 {
		fmt.Println("Note: OpenAI model list doesn't expose vision capability metadata.")
		fmt.Println("Using gpt-4o which supports vision.")
		fmt.Println()
	} else {
		fmt.Printf("Found %d vision-capable models:\n", len(visionModels))
		for _, m := range visionModels {
			fmt.Printf("   - %s\n", m.Name)
		}
		fmt.Println()
	}

	// Example 1: Image from URL (low detail)
	fmt.Println("Example 1: Image from URL (low detail)")
	fmt.Println(strings.Repeat("-", 40))

	// Step: Building low-detail request
	if config.StepPause("Building request with low detail level...", []string{
		"nxusKit: WithDetail() sets OpenAI's image detail level",
		"'low' is faster and cheaper, uses fewer tokens",
		"Provider-specific options through a consistent interface",
	}) == interactive.ActionQuit {
		return nil
	}

	msg := nxuskit.UserMessage("What's in this image? Describe it briefly.").
		WithImageURL("https://upload.wikimedia.org/wikipedia/commons/thumb/d/d5/Rust_programming_language_black_logo.svg/800px-Rust_programming_language_black_logo.svg.png").
		WithDetail("low") // Faster and cheaper

	req, err := nxuskit.NewChatRequest("gpt-4o-mini",
		nxuskit.WithMessages(msg),
		nxuskit.WithMaxTokens(300),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("openai"), req)

	// Step: Sending request
	if config.StepPause("Sending low-detail vision request to OpenAI...", []string{
		"Using gpt-4o-mini for cost-effective vision tasks",
	}) == interactive.ActionQuit {
		return nil
	}

	start := time.Now()
	resp, err := provider.Chat(ctx, req)
	elapsedMs := time.Since(start).Milliseconds()

	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)

		usage := resp.Usage.BestAvailable()
		fmt.Printf("Response: %s\n", resp.Content)
		fmt.Printf("Token usage: %d input, %d output\n\n",
			usage.PromptTokens, usage.CompletionTokens)
	}

	// Example 2: High-detail analysis
	fmt.Println("Example 2: High-detail analysis")
	fmt.Println(strings.Repeat("-", 40))

	// Step: Building high-detail request
	if config.StepPause("Building request with high detail level...", []string{
		"nxusKit: 'high' detail uses more tokens for detailed analysis",
		"OpenAI processes the image at higher resolution",
	}) == interactive.ActionQuit {
		return nil
	}

	msg = nxuskit.UserMessage("Analyze this diagram in detail. What elements does it contain?").
		WithImageURL("https://upload.wikimedia.org/wikipedia/commons/thumb/d/d5/Rust_programming_language_black_logo.svg/800px-Rust_programming_language_black_logo.svg.png").
		WithDetail("high") // More detailed analysis

	req, err = nxuskit.NewChatRequest("gpt-4o",
		nxuskit.WithMessages(msg),
		nxuskit.WithMaxTokens(500),
	)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Verbose: Show the request
	config.PrintRequest("POST", getProviderURL("openai"), req)

	// Step: Sending high-detail request
	if config.StepPause("Sending high-detail vision request to OpenAI...", []string{
		"Using gpt-4o for more detailed image analysis",
	}) == interactive.ActionQuit {
		return nil
	}

	start = time.Now()
	resp, err = provider.Chat(ctx, req)
	elapsedMs = time.Since(start).Milliseconds()

	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)

		fmt.Printf("Response: %s\n\n", resp.Content)
	}

	return nil
}

// getProviderURL returns the API URL for verbose output based on provider name.
func getProviderURL(providerName string) string {
	switch providerName {
	case "claude":
		return "https://api.anthropic.com/v1/messages"
	case "openai":
		return "https://api.openai.com/v1/chat/completions"
	default:
		return "https://api.example.com/chat"
	}
}
