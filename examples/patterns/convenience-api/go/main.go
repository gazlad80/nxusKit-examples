// Example: Convenience API (LiteLLM-style)
//
// ## nxusKit Features Demonstrated
// - One-liner completions with automatic provider detection
// - Model name routing (gpt-4 -> OpenAI, claude-* -> Anthropic)
// - Explicit provider prefixes (openai/gpt-4, anthropic/claude-*)
// - Streaming convenience function
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each API call with explanations
//
// ## Why This Pattern Matters
// For simple use cases, explicit provider setup is overhead. nxusKit's
// convenience API auto-detects providers from model names and environment
// variables, providing a "just works" experience for rapid prototyping.
//
// Usage:
//
//	# With OpenAI
//	export OPENAI_API_KEY="your-key-here"
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
//	# With Claude
//	export ANTHROPIC_API_KEY="your-key-here"
//	go run .
//
//	# With Ollama (no API key needed)
//	go run .
//
// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()

	fmt.Println("=== nxuskit Convenience API Examples ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Introduction
	if config.StepPause("Understanding the convenience API...", []string{
		"nxusKit: One-liner API auto-detects provider from model name",
		"gpt-* models route to OpenAI, claude-* to Anthropic",
		"Credentials are auto-detected from environment variables",
		"No explicit provider setup required",
	}) == interactive.ActionQuit {
		return
	}

	// ========================================
	// Example 1: Simple completion with auto-detection
	// ========================================
	fmt.Println("Example 1: Auto-detected provider")
	fmt.Println("Asking: What is Go programming language?")
	fmt.Println()

	// Step: OpenAI auto-detection
	if config.StepPause("Making OpenAI request with auto-detection...", []string{
		"nxusKit: Completion() detects 'gpt-4o' as OpenAI model",
		"OPENAI_API_KEY is read from environment automatically",
		"Request is routed to api.openai.com/v1/chat/completions",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: One-liner API - provider auto-detected from model name
	start := time.Now()
	resp, err := nxuskit.Completion(ctx, "gpt-4o", "Explain Go in one sentence.")
	elapsedMs := time.Since(start).Milliseconds()
	if err != nil {
		fmt.Printf("  OpenAI example failed (OK if OPENAI_API_KEY not set): %v\n\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)
		fmt.Printf("Response: %s\n\n", resp.Content)
	}

	// ========================================
	// Example 2: Explicit provider specification
	// ========================================
	fmt.Println("Example 2: Explicit provider (anthropic/claude)")
	fmt.Println()

	// Step: Explicit provider
	if config.StepPause("Making Claude request with explicit provider...", []string{
		"nxusKit: 'anthropic/claude-haiku-4-5-20251001' explicitly specifies Anthropic",
		"Useful when model names are ambiguous or custom",
		"ANTHROPIC_API_KEY is read from environment automatically",
	}) == interactive.ActionQuit {
		return
	}

	start = time.Now()
	resp, err = nxuskit.Completion(ctx, "anthropic/claude-haiku-4-5-20251001",
		"What makes Go memory-safe? Answer in one sentence.")
	elapsedMs = time.Since(start).Milliseconds()
	if err != nil {
		fmt.Printf("  Anthropic example failed (OK if ANTHROPIC_API_KEY not set): %v\n\n", err)
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)
		fmt.Printf("Response: %s\n\n", resp.Content)
	}

	// ========================================
	// Example 3: Streaming response
	// ========================================
	fmt.Println("Example 3: Streaming response")

	// Step: Streaming
	if config.StepPause("Making streaming request...", []string{
		"nxusKit: CompletionStream returns channels for chunks and errors",
		"Tokens arrive incrementally as they're generated",
		"Great for real-time display of long responses",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Print("Streaming response: ")

	chunkCount := 0
	start = time.Now()
	chunks, errs := nxuskit.CompletionStream(ctx, "gpt-4o", "Count from 1 to 5 with brief comments.")

	for chunk := range chunks {
		fmt.Print(chunk.Delta)
		chunkCount++
		// Verbose: Show chunk
		config.PrintStreamChunk(chunkCount, chunk.Delta)
	}
	elapsedMs = time.Since(start).Milliseconds()
	if err := <-errs; err != nil {
		fmt.Printf("\n  Streaming example failed: %v\n", err)
	} else {
		// Verbose: Show stream completion
		config.PrintStreamDone(elapsedMs, chunkCount)
	}
	fmt.Println()
	fmt.Println()

	// ========================================
	// Example 4: Ollama (local model)
	// ========================================
	fmt.Println("Example 4: Ollama local model")
	fmt.Println()

	// Step: Ollama
	if config.StepPause("Making Ollama local request...", []string{
		"nxusKit: 'llama3' routes to Ollama (localhost:11434)",
		"No API key needed for local models",
		"WithTimeout extends timeout for model loading",
	}) == interactive.ActionQuit {
		return
	}

	start = time.Now()
	resp, err = nxuskit.Completion(ctx, "llama3", "What is the capital of France? One word only.",
		nxuskit.WithTimeout(30*time.Second),
	)
	elapsedMs = time.Since(start).Milliseconds()
	if err != nil {
		fmt.Printf("  Ollama example failed (OK if Ollama not running): %v\n", err)
		fmt.Println("   To use Ollama:")
		fmt.Println("   1. Install from https://ollama.ai")
		fmt.Println("   2. Run: ollama pull llama3")
		fmt.Println("   3. Ollama starts automatically")
		fmt.Println()
	} else {
		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)
		fmt.Printf("Response: %s\n\n", resp.Content)
	}

	// ========================================
	// Example 5: Provider routing examples
	// ========================================
	fmt.Println("Example 5: Provider routing examples")
	fmt.Println()

	// Step: Routing summary
	if config.StepPause("Understanding provider routing...", []string{
		"Auto-detection uses model name prefixes (gpt-, claude-, llama)",
		"Explicit format: 'provider/model' for unambiguous routing",
		"Environment variables: OPENAI_API_KEY, ANTHROPIC_API_KEY",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Println("These all work with auto-detection:")
	fmt.Println("  - Completion(ctx, \"gpt-4o\", ...) -> OpenAI")
	fmt.Println("  - Completion(ctx, \"gpt-3.5-turbo\", ...) -> OpenAI")
	fmt.Println("  - Completion(ctx, \"claude-haiku-4-5-20251001\", ...) -> Anthropic")
	fmt.Println("  - Completion(ctx, \"claude-3-opus\", ...) -> Anthropic")
	fmt.Println("  - Completion(ctx, \"llama3\", ...) -> Ollama")
	fmt.Println()

	fmt.Println("Or use explicit provider/model format:")
	fmt.Println("  - Completion(ctx, \"openai/gpt-4o\", ...)")
	fmt.Println("  - Completion(ctx, \"anthropic/claude-haiku-4-5-20251001\", ...)")
	fmt.Println("  - Completion(ctx, \"ollama/llama3\", ...)")
	fmt.Println()

	// ========================================
	// Summary
	// ========================================
	fmt.Println("=== Summary ===")
	fmt.Println()
	fmt.Println("The convenience API provides:")
	fmt.Println("  - Automatic provider detection from model names")
	fmt.Println("  - Automatic credential detection from environment")
	fmt.Println("  - Simple streaming with channels")
	fmt.Println("  - Unified interface across all providers")
	fmt.Println("  - Minimal boilerplate - just model and prompt")
	fmt.Println()
	fmt.Println("For more control, use the provider-specific APIs.")
	fmt.Println("See examples: basic-chat, streaming, multi-provider")
}
