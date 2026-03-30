// Example demonstrating OllamaProvider usage with nxuskit.
//
// Prerequisites:
//   - Ollama installed and running (`ollama serve`)
//   - A model pulled (e.g., `ollama pull llama3.2`)
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
//	OLLAMA_MODEL            Optional. Model tag for chat demos (default: small / common tag from ListModels)
//	NXUSKIT_VERBOSE=1       Enable verbose mode
//	NXUSKIT_STEP=1          Enable step mode
//go:build nxuskit

// NXUSKIT_VERBOSE_LIMIT   Max characters before truncation (default: 2000)
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// config is set in main() and used by helper functions for verbose/step output
var config *interactive.Config

func main() {
	config = interactive.FromArgs()

	// Step: Create provider
	action := config.StepPause("Creating Ollama provider...", []string{
		"Connects to local Ollama server (OLLAMA_HOST env var or localhost:11434)",
		"Ollama runs open-source LLMs locally",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Create provider with default settings (localhost:11434)
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		log.Fatal(err)
	}

	// Step: Check server
	action = config.StepPause("Checking server availability...", []string{
		"Pings the Ollama server to verify it's running",
		"Make sure Ollama is running: ollama serve",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Check server availability
	if err := provider.Ping(context.Background()); err != nil {
		log.Fatalf("Ollama server not available: %v", err)
	}
	fmt.Println("[OK] Connected to Ollama")

	// Step: List models
	action = config.StepPause("Listing available models...", []string{
		"Fetches list of models pulled in Ollama",
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
		fmt.Printf("  - %s (%s)\n", m.Name, m.FormattedSize())
	}

	// Prefer small / common tags so CI and first-time runs don't pick the first
	// tag in API order (often a large model). Override with OLLAMA_MODEL.
	modelName := pickOllamaDemoModel(models)

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

	// Thinking demo calls a heavy model; skip when stdin is not a TTY (e.g. smoke harness).
	if hasThinkingModel(models) && config.IsTTYMode() {
		// Step: Thinking mode
		action = config.StepPause("Running thinking mode...", []string{
			"Demonstrates extended thinking with qwen3 or deepseek-r1",
			"Model shows its reasoning process before answering",
		})
		if action == interactive.ActionQuit {
			fmt.Println("Exiting...")
			return
		}

		fmt.Printf("\n--- Thinking Mode ---\n")
		thinkingMode(provider)
	}
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
	config.PrintRequest("POST", "ollama/api/chat", map[string]interface{}{
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
			nxuskit.UserMessage("Tell me a very short joke about programming."),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show request details
	config.PrintRequest("POST", "ollama/api/chat (streaming)", map[string]interface{}{
		"model":   model,
		"message": "Tell me a very short joke about programming.",
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

func thinkingMode(provider nxuskit.LLMProvider) {
	// Use qwen3 or deepseek-r1 for thinking mode
	req, err := nxuskit.NewChatRequest("qwen3:latest",
		nxuskit.WithMessages(
			nxuskit.UserMessage("Solve step by step: 2x + 5 = 15"),
		),
		nxuskit.WithThinkingMode(nxuskit.ThinkingModeEnabled),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Verbose: Show request details
	config.PrintRequest("POST", "ollama/api/chat", map[string]interface{}{
		"model":         "qwen3:latest",
		"message":       "Solve step by step: 2x + 5 = 15",
		"thinking_mode": "enabled",
	})

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Note: Thinking mode requires a compatible model like qwen3\n")
		return
	}

	// Verbose: Show response
	config.PrintResponse(200, 0, map[string]interface{}{
		"content":  resp.Content,
		"thinking": resp.Thinking,
	})

	if resp.Thinking != nil {
		fmt.Printf("Thinking: %s\n\n", *resp.Thinking)
	}
	fmt.Printf("Answer: %s\n", resp.Content)
}

// pickOllamaDemoModel chooses a model for chat/stream demos. Order favors small
// models that are likely pulled for quick runs; set OLLAMA_MODEL to force one.
func pickOllamaDemoModel(models []nxuskit.ModelInfo) string {
	if env := os.Getenv("OLLAMA_MODEL"); env != "" {
		return env
	}
	preferBase := []string{"tinyllama", "llama3.2", "phi3", "llama3", "mistral"}
	for _, base := range preferBase {
		prefix := strings.ToLower(base) + ":"
		for _, m := range models {
			n := m.Name
			low := strings.ToLower(n)
			if low == strings.ToLower(base) || strings.HasPrefix(low, prefix) {
				return n
			}
		}
	}
	if len(models) > 0 {
		return models[0].Name
	}
	return "llama3.2"
}

func hasThinkingModel(models []nxuskit.ModelInfo) bool {
	for _, m := range models {
		if m.Name == "qwen3:latest" || m.Name == "deepseek-r1:latest" {
			return true
		}
	}
	return false
}
