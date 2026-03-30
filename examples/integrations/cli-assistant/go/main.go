// CLI Assistant Example
//
// Converts natural language to shell commands with streaming output.
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
	"os"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

const systemPrompt = `You are a CLI assistant. Convert natural language to shell commands.
Rules:
- Output ONLY the command, no explanation
- Use common Unix commands (ls, grep, find, etc.)
- For dangerous operations, add a comment warning
- If unclear, output a best-guess command with a clarifying comment`

// config is set in main() and used by GenerateCommand for verbose/step output
var config *interactive.Config

func init() {
	// Ensure config is never nil (tests don't call main)
	if config == nil {
		config = &interactive.Config{}
	}
}

// GenerateCommand generates a shell command from natural language using streaming.
func GenerateCommand(ctx context.Context, provider nxuskit.LLMProvider, query string) (string, error) {
	req, err := nxuskit.NewChatRequest("llama3",
		nxuskit.WithMessages(
			nxuskit.SystemMessage(systemPrompt),
			nxuskit.UserMessage(query),
		),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Verbose: Show request details
	config.PrintRequest("POST", "ollama/api/chat (streaming)", map[string]interface{}{
		"model":  "llama3",
		"system": "[system prompt for CLI assistant]",
		"query":  query,
	})

	chunks, errs := provider.ChatStream(ctx, req)
	var output strings.Builder

	fmt.Print("$ ")

	chunkNum := 0
	for chunk := range chunks {
		fmt.Print(chunk.Delta)
		output.WriteString(chunk.Delta)
		chunkNum++
		// Verbose: Show first few chunks
		if chunkNum <= 3 {
			config.PrintStreamChunk(chunkNum, chunk.Delta)
		}
	}
	fmt.Println()

	// Verbose: Show stream completion
	config.PrintStreamDone(0, chunkNum)

	if err := <-errs; err != nil {
		return "", err
	}

	return output.String(), nil
}

func main() {
	config = interactive.FromArgs()

	fmt.Println("=== CLI Assistant Demo ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Create provider
	action := config.StepPause("Creating Ollama provider...", []string{
		"Connects to local Ollama server (OLLAMA_HOST env var or localhost:11434)",
		"Will be used to stream command generation from the LLM",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// NewOllamaProvider() reads OLLAMA_HOST env var automatically
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		fmt.Printf("Failed to create provider: %v\n", err)
		os.Exit(1)
	}

	queries := []string{
		"find all rust files modified in the last week",
		"show disk usage sorted by size",
		"list all running docker containers",
	}

	for i, query := range queries {
		// Step: Process each query
		action := config.StepPause(fmt.Sprintf("Processing query %d/%d...", i+1, len(queries)), []string{
			fmt.Sprintf("Query: %q", query),
			"LLM will convert this to a shell command",
			"Response is streamed for real-time output",
		})
		if action == interactive.ActionQuit {
			fmt.Println("Exiting...")
			return
		}

		fmt.Printf("Query: \"%s\"\n", query)
		_, err := GenerateCommand(ctx, provider, query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		fmt.Println()
	}

	// Interactive mode hint
	fmt.Println("Tip: For interactive use, pass your query as a command-line argument:")
	fmt.Println("  go run . \"your natural language query here\"")
}
