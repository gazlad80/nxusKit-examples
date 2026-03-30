// CLIPS+LLM Hybrid Example
//
// Demonstrates combining CLIPS expert system with LLM for superior results.
// nxuskit provides native CLIPS support via ClipsProvider.
//
// ## nxusKit Features Demonstrated
// - ClipsProvider for deterministic business rule execution
// - LLMProvider for natural language understanding
// - Hybrid pattern: LLM → CLIPS → LLM workflow
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

func main() {
	config := interactive.FromArgs()

	fmt.Println("=== CLIPS+LLM Hybrid Demo ===")
	fmt.Println()
	fmt.Println("This example demonstrates the hybrid AI pattern:")
	fmt.Println("1. LLM classifies ticket (category, priority, sentiment, entities)")
	fmt.Println("2. CLIPS applies deterministic routing rules (team, SLA, escalation)")
	fmt.Println("3. LLM generates empathetic response suggestion")
	fmt.Println()

	ctx := context.Background()

	// Step: Create provider
	action := config.StepPause("Creating Ollama provider...", []string{
		"Connects to local Ollama server (OLLAMA_HOST env var or localhost:11434)",
		"Will be used for ticket classification and response generation",
		"CLIPS rules handle deterministic routing separately",
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

	tickets := []struct {
		label string
		text  string
	}{
		{"Security Incident", "URGENT: We've detected unauthorized access attempts on our production database. Multiple failed login attempts from unknown IPs. Need immediate investigation!"},
		{"Infrastructure Issue", "Database connection timeouts are causing checkout failures. Customers are complaining they can't complete purchases. This started after last night's deployment."},
		{"Application Bug", "The login button on the mobile app is not responding. Users have to force close and reopen the app. Started happening after the latest update."},
		{"General Inquiry", "Hi, I was wondering if you could help me understand how to export my data? The documentation is a bit unclear."},
	}

	for i, ticket := range tickets {
		// Step: Process each ticket
		action := config.StepPause(fmt.Sprintf("Processing ticket %d/%d: %s", i+1, len(tickets), ticket.label), []string{
			"Step 1: LLM classifies ticket (category, priority, sentiment)",
			"Step 2: CLIPS applies deterministic routing rules",
			"Step 3: LLM generates empathetic response",
		})
		if action == interactive.ActionQuit {
			fmt.Println("Exiting...")
			return
		}

		fmt.Printf("=== %s ===\n", ticket.label)
		truncated := ticket.text
		if len(truncated) > 80 {
			truncated = truncated[:80] + "..."
		}
		fmt.Printf("Ticket: %s\n\n", truncated)

		// Verbose: Show request details
		config.PrintRequest("POST", "ollama/api/chat", map[string]interface{}{
			"model":  "llama3",
			"step":   "classify ticket",
			"ticket": truncated,
		})

		// Rules file is in parent directory (shared between Rust and Go implementations)
		rulesPath := "../ticket-routing.clp"
		analysis, err := AnalyzeTicket(ctx, provider, "llama3", ticket.text, rulesPath)
		if err != nil {
			fmt.Printf("Analysis failed: %v\n", err)
			fmt.Println()
			fmt.Println(strings.Repeat("-", 60))
			fmt.Println()
			continue
		}

		// Verbose: Show response summary
		config.PrintResponse(200, 0, map[string]interface{}{
			"team":            analysis.Team,
			"sla_hours":       analysis.SLAHours,
			"escalation":      analysis.EscalationLevel,
			"sentiment":       analysis.Sentiment,
			"key_entities":    analysis.KeyEntities,
			"response_length": len(analysis.SuggestedResponse),
		})

		fmt.Println("Routing (from CLIPS rules - deterministic):")
		fmt.Printf("  Team: %s\n", analysis.Team)
		fmt.Printf("  SLA: %d hours\n", analysis.SLAHours)
		fmt.Printf("  Escalation: Level %d\n", analysis.EscalationLevel)
		fmt.Println()
		fmt.Println("Analysis (from LLM - probabilistic):")
		fmt.Printf("  Sentiment: %s\n", analysis.Sentiment)
		fmt.Printf("  Key Entities: %v\n", analysis.KeyEntities)
		fmt.Println()
		fmt.Println("Suggested Response:")
		fmt.Printf("  %s\n", analysis.SuggestedResponse)

		fmt.Println()
		fmt.Println(strings.Repeat("-", 60))
		fmt.Println()
	}

	fmt.Println("=== Why Hybrid is Better ===")
	fmt.Println("- LLM alone: May miss SLA policies, inconsistent routing")
	fmt.Println("- CLIPS alone: Can't understand natural language input")
	fmt.Println("- CLIPS + LLM: Best of both - understanding AND policy compliance")
}
