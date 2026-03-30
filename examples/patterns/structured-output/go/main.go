// Example: Structured Output (JSON Mode)
//
// ## nxusKit Features Demonstrated
// - JSON schema-guided output generation
// - Type-safe response parsing with json.Unmarshal
// - Provider-agnostic structured output
// - Schema validation and error handling
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each step with explanations
//
// ## Why This Pattern Matters
// Structured output enables reliable integration with downstream systems.
// nxusKit handles the different JSON mode implementations across providers
// (OpenAI's response_format, Claude's tool use, Ollama's format parameter).
//
// Usage:
//
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
//go:build nxuskit

// Demonstrates extracting typed structured data from LLM responses.
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

	fmt.Println("=== Structured Output (JSON Mode) Demo ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Creating provider
	if config.StepPause("Creating Ollama provider...", []string{
		"nxusKit: Local provider for development - no API key needed",
		"Ollama supports JSON mode via format parameter",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: Local provider for development - no API key needed
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		fmt.Printf("Failed to create provider: %v\n", err)
		os.Exit(1)
	}

	// Step: Setting up log entries
	if config.StepPause("Setting up test log entries...", []string{
		"Three sample log entries with different severities",
		"We'll classify each one using JSON mode",
		"Output will be parsed into a typed Go struct",
	}) == interactive.ActionQuit {
		return
	}

	logEntries := []string{
		"2024-01-15 10:23:45 ERROR Failed login attempt for user admin from IP 192.168.1.100 after 5 retries",
		"2024-01-15 10:24:12 INFO User john.doe successfully authenticated",
		"2024-01-15 10:25:33 CRITICAL Database connection pool exhausted, all connections in use",
	}

	for i, logEntry := range logEntries {
		fmt.Printf("--- Log Entry %d ---\n", i+1)
		fmt.Printf("Input: %s\n\n", logEntry)

		// Step: Classifying log entry
		if config.StepPause(fmt.Sprintf("Classifying log entry %d...", i+1), []string{
			"nxusKit: ClassifyLog uses JSON mode to get typed LogClassification struct",
			"LLM output is constrained to valid JSON matching our schema",
			"Parsed directly into Go struct with json.Unmarshal",
		}) == interactive.ActionQuit {
			return
		}

		// Verbose: Show what we're sending
		config.PrintRequest("POST", getProviderURL("ollama"), map[string]interface{}{
			"model":     "llama3",
			"log_entry": logEntry,
			"format":    "json",
		})

		// nxusKit: ClassifyLog uses JSON mode to get typed LogClassification struct
		start := time.Now()
		classification, err := ClassifyLog(ctx, provider, "llama3", logEntry)
		elapsedMs := time.Since(start).Milliseconds()

		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, classification)

		fmt.Println("Classification:")
		fmt.Printf("  Severity: %s\n", classification.Severity)
		fmt.Printf("  Category: %s\n", classification.Category)
		fmt.Printf("  Summary: %s\n", classification.Summary)
		fmt.Printf("  Actionable: %t\n", classification.Actionable)
		fmt.Println()
	}
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
