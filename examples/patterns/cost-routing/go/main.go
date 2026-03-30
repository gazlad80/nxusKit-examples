// Example: Model Router (Cost Tiers)
//
// ## nxusKit Features Demonstrated
// - Task complexity classification
// - Dynamic model selection based on task requirements
// - Cost-tier routing (economy/standard/premium)
// - Provider-agnostic request routing
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show classification details
// - `--step` or `-s`: Pause at each classification with explanations
//
// ## Why This Pattern Matters
// Using expensive models for simple queries wastes money. This pattern
// demonstrates how nxusKit's unified interface enables intelligent routing
// to appropriate models based on task complexity, optimizing cost/quality.
//
// Usage:
//
//	go run .
//	go run . --verbose    # Show classification details
//	go run . --step       # Step through with explanations
//
// Demonstrates routing requests to different models based on complexity.
package main

import (
	"fmt"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
)

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()

	fmt.Println("=== Model Router (Cost Tiers) Demo ===")
	fmt.Println()

	// Step: Introduction
	if config.StepPause("Understanding cost-tier routing...", []string{
		"nxusKit: Task complexity determines which model tier to use",
		"Economy tier (gpt-4o-mini): Fast, cheap for simple queries",
		"Standard tier (gpt-4o): Balanced for general tasks",
		"Premium tier (gpt-4-turbo): Best quality for complex analysis",
	}) == interactive.ActionQuit {
		return
	}

	prompts := []struct {
		label  string
		prompt string
	}{
		{"Simple", "What is 2+2?"},
		{"Medium", "Explain the concept of recursion in programming. Include an example of how it works and when you might use it in practice."},
		{"Complex", "Analyze the trade-offs between microservices and monolithic architectures. Compare their scalability, maintainability, deployment complexity, and team coordination requirements."},
	}

	for _, p := range prompts {
		fmt.Printf("--- %s Prompt ---\n", p.label)
		truncated := p.prompt
		if len(truncated) > 50 {
			truncated = truncated[:50] + "..."
		}
		fmt.Printf("Input: %s\n", truncated)

		// Step: Classification
		if config.StepPause("Classifying prompt complexity...", []string{
			"nxusKit: ClassifyTask analyzes prompt characteristics",
			"Checks for complex keywords (analyze, compare, evaluate)",
			"Considers prompt length (>1000 chars = complex, >200 = medium)",
			"Returns appropriate cost tier for the task",
		}) == interactive.ActionQuit {
			return
		}

		// nxusKit: ClassifyTask determines appropriate cost tier for the prompt
		tier := ClassifyTask(p.prompt)

		// Verbose: Show classification reasoning
		if config.IsVerbose() {
			fmt.Printf("[nxusKit] Prompt length: %d chars\n", len(p.prompt))
			fmt.Printf("[nxusKit] Classification: %s -> %s\n", tier.Name(), tier.ModelName())
		}

		fmt.Printf("Classified as: %s (would use: %s)\n", tier.Name(), tier.ModelName())

		// For demo, we'll just show classification without making actual API calls
		// In production, use RoutedChat() to make the actual request
		fmt.Println()
	}

	// Step: Summary
	if config.StepPause("Reviewing tier summary...", []string{
		"Each tier maps to a specific model optimized for that complexity level",
		"In production, RoutedChat() makes the actual API call with the selected model",
		"This pattern saves money by matching model capability to task requirements",
	}) == interactive.ActionQuit {
		return
	}

	// Show tier breakdown
	fmt.Println("=== Tier Summary ===")
	fmt.Printf("Economy (%s): Short, simple queries\n", TierEconomy.ModelName())
	fmt.Printf("Standard (%s): Medium complexity\n", TierStandard.ModelName())
	fmt.Printf("Premium (%s): Complex analysis\n", TierPremium.ModelName())
}
