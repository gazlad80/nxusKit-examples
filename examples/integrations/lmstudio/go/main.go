// Example demonstrating LmStudioProvider usage with nxuskit.
//
// Prerequisites:
//   - LM Studio installed with server mode enabled
//   - A model loaded in LM Studio
//
// Run with: go run main.go
//
// ## Interactive Modes
//
// This example supports interactive debugging modes:
//
//	--verbose, -v    Show raw LLM request/response data
//	--step, -s       Pause at each step with explanations
//
// Environment variables:
//
//	NXUSKIT_VERBOSE=1       Enable verbose mode
//	NXUSKIT_STEP=1          Enable step mode
//go:build nxuskit

// NXUSKIT_VERBOSE_LIMIT   Max characters before truncation (default: 2000)
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// config is set in main() and used by helper functions for verbose/step output
var config *interactive.Config

func main() {
	config = interactive.FromArgs()

	// Step: Create provider
	action := config.StepPause("Creating LM Studio provider...", []string{
		"Connects to LM Studio server (default: localhost:1234/v1)",
		"LM Studio provides OpenAI-compatible API for local models",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Create provider with default settings (localhost:1234/v1)
	provider, err := nxuskit.NewLmStudioFFIProvider()
	if err != nil {
		log.Fatal(err)
	}

	// Step: Check server
	action = config.StepPause("Checking server availability...", []string{
		"Pings the LM Studio server to verify it's running",
		"Make sure LM Studio is running with local server enabled",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Check server availability
	if err := provider.Ping(context.Background()); err != nil {
		log.Fatalf("LM Studio server not available: %v", err)
	}
	fmt.Println("[OK] Connected to LM Studio")

	// Step: List models
	action = config.StepPause("Listing available models...", []string{
		"Fetches list of models loaded in LM Studio",
		"Will use the first available model for chat demos",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// List available models
	models, err := provider.ListModels(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\nAvailable models (%d):\n", len(models))
	for _, m := range models {
		fmt.Printf("  - %s\n", m.Name)
	}

	// Use first model or "local-model" as fallback
	modelName := "local-model"
	if len(models) > 0 {
		modelName = models[0].Name
	}

	// Step: Basic chat
	action = config.StepPause("Running basic chat...", []string{
		fmt.Sprintf("Will send a simple question to %s", modelName),
		"Demonstrates synchronous chat API",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Basic chat
	fmt.Printf("\n--- Basic Chat with %s ---\n", modelName)
	basicChat(provider, modelName)

	// Step: Streaming chat
	action = config.StepPause("Running streaming chat...", []string{
		"Demonstrates real-time streaming response",
		"Tokens are displayed as they're generated",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Streaming chat
	fmt.Printf("\n--- Streaming Chat ---\n")
	streamingChat(provider, modelName)

	// Step: Chat with options
	action = config.StepPause("Running chat with options...", []string{
		"Demonstrates temperature and max_tokens parameters",
		"Lower temperature (0.3) for more focused responses",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Chat with options
	fmt.Printf("\n--- Chat with Options ---\n")
	chatWithOptions(provider, modelName)
}

func basicChat(provider nxuskit.LLMProvider, model string) {
	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.UserMessage("What is the capital of France? Answer in one sentence."),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show request details
	config.PrintRequest("POST", "lmstudio/v1/chat/completions", map[string]interface{}{
		"model":   model,
		"message": "What is the capital of France? Answer in one sentence.",
	})

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show response
	config.PrintResponse(200, 0, map[string]interface{}{
		"content": resp.Content,
		"usage":   resp.Usage,
	})

	fmt.Printf("Response: %s\n", resp.Content)
	if resp.Usage.Actual != nil {
		fmt.Printf("Tokens: %d prompt, %d completion\n",
			resp.Usage.Actual.PromptTokens,
			resp.Usage.Actual.CompletionTokens)
	}
}

func streamingChat(provider nxuskit.LLMProvider, model string) {
	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.UserMessage("Write a haiku about coding."),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show request details
	config.PrintRequest("POST", "lmstudio/v1/chat/completions (streaming)", map[string]interface{}{
		"model":   model,
		"message": "Write a haiku about coding.",
		"stream":  true,
	})

	chunks, errs := provider.ChatStream(context.Background(), req)

	fmt.Print("Response: ")
	chunkNum := 0
	for chunk := range chunks {
		fmt.Print(chunk.Delta)
		chunkNum++
		// Verbose: Show first few chunks
		if chunkNum <= 3 {
			config.PrintStreamChunk(chunkNum, chunk.Delta)
		}
		if chunk.IsFinal() {
			fmt.Println()
		}
	}

	// Verbose: Show stream completion
	config.PrintStreamDone(0, chunkNum)

	if err := <-errs; err != nil {
		log.Fatal(err)
	}
}

func chatWithOptions(provider nxuskit.LLMProvider, model string) {
	temp := 0.3 // Lower temperature for more focused responses
	maxTokens := 50

	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.SystemMessage("You are a helpful assistant that gives concise answers."),
			nxuskit.UserMessage("What is 2+2?"),
		),
		nxuskit.WithTemperature(temp),
		nxuskit.WithMaxTokens(maxTokens),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show request details
	config.PrintRequest("POST", "lmstudio/v1/chat/completions", map[string]interface{}{
		"model":       model,
		"temperature": temp,
		"max_tokens":  maxTokens,
		"message":     "What is 2+2?",
	})

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show response
	config.PrintResponse(200, 0, map[string]interface{}{
		"content": resp.Content,
	})

	fmt.Printf("Response: %s\n", resp.Content)
}
