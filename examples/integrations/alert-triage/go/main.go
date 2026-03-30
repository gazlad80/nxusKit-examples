// Alert Triage Example
//
// Demonstrates LLM-powered alert triage for observability systems.
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
//	OLLAMA_MODEL            Optional. Model tag for triage (default: first matching llama3 / llama3.2 / phi3 / tinyllama)
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

func main() {
	config := interactive.FromArgs()

	fmt.Println("=== Alert Triage Demo ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Create provider
	action := config.StepPause("Creating Ollama provider...", []string{
		"Connects to local Ollama server (OLLAMA_HOST env var or localhost:11434)",
		"Will be used to call the LLM for alert triage",
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

	// Sample alerts (matching Alertmanager format)
	alerts := []Alert{
		{
			AlertName:   "HighMemoryUsage",
			Severity:    "warning",
			Instance:    "web-server-01",
			Description: "Memory usage above 85% for 5 minutes",
		},
		{
			AlertName:   "PodCrashLooping",
			Severity:    "critical",
			Instance:    "api-deployment-xyz",
			Description: "Pod restarted 5 times in last 10 minutes",
		},
		{
			AlertName:   "SSLCertExpiring",
			Severity:    "warning",
			Instance:    "loadbalancer-prod",
			Description: "SSL certificate expires in 7 days",
		},
	}

	// Step: Process alerts
	action = config.StepPause("Preparing to triage alerts...", []string{
		fmt.Sprintf("Will process %d sample alerts", len(alerts)),
		"LLM will analyze severity, suggest priority, and recommend actions",
		"Uses JSON mode for structured output",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	fmt.Printf("Processing %d alerts...\n\n", len(alerts))

	models, err := provider.ListModels(ctx)
	if err != nil {
		fmt.Printf("Failed to list models: %v\n", err)
		os.Exit(1)
	}
	modelTag := pickOllamaTriageModel(models)

	// Verbose: Show request details
	config.PrintRequest("POST", "ollama/api/chat", map[string]interface{}{
		"model":    modelTag,
		"messages": fmt.Sprintf("[system prompt + %d alerts as JSON]", len(alerts)),
	})

	results, err := TriageAlerts(ctx, provider, modelTag, alerts)
	if err != nil {
		fmt.Printf("Triage failed: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show response summary
	config.PrintResponse(200, 0, map[string]interface{}{
		"results_count": len(results),
		"alerts":        results,
	})

	// Step: Display results
	action = config.StepPause("Displaying triage results...", []string{
		fmt.Sprintf("Got %d triage results from LLM", len(results)),
		"Each result includes priority, summary, likely cause, and suggested actions",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	for _, result := range results {
		fmt.Printf("=== %s ===\n", result.AlertName)
		fmt.Printf("Priority: %d (1=highest, 5=lowest)\n", result.Priority)
		fmt.Printf("Summary: %s\n", result.Summary)
		fmt.Printf("Likely Cause: %s\n", result.LikelyCause)
		fmt.Println("Suggested Actions:")
		for _, action := range result.SuggestedActions {
			fmt.Printf("  - %s\n", action)
		}
		fmt.Println()
	}
}

// pickOllamaTriageModel selects a model that exists locally and follows common
// Ollama tags (avoids hardcoding "llama3" when only "llama3:latest" is installed).
func pickOllamaTriageModel(models []nxuskit.ModelInfo) string {
	if env := os.Getenv("OLLAMA_MODEL"); env != "" {
		return env
	}
	preferBase := []string{"llama3", "llama3.2", "phi3", "mistral", "tinyllama"}
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
	return "llama3:latest"
}
