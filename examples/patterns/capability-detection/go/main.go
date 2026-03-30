// Example: Capability-Aware Model Selection
//
// ## nxusKit Features Demonstrated
// - ModelInfo metadata (Modalities, ContextWindow, MaxImages)
// - Capability-based model filtering
// - ListModels() discovery across providers
// - Task-to-model matching patterns
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each operation with explanations
//
// ## Why This Pattern Matters
// Different models have different capabilities (vision, context size, etc.).
// nxusKit normalizes model metadata across providers, enabling intelligent
// model selection based on task requirements rather than hardcoded model names.
//
// Usage:
//
//	# Check OpenAI models
//	OPENAI_API_KEY=your-key go run . openai
//	OPENAI_API_KEY=your-key go run . openai --verbose
//	OPENAI_API_KEY=your-key go run . openai --step
//
//	# Check Claude models
//	ANTHROPIC_API_KEY=your-key go run . claude
//
//	# Check Ollama models (requires Ollama running locally)
//	go run . ollama
//
//go:build nxuskit

// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"context"
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

	// Get provider from command line args (non-flag arguments)
	providerName := "openai"
	for _, arg := range os.Args[1:] {
		if !strings.HasPrefix(arg, "-") {
			providerName = arg
			break
		}
	}

	fmt.Println("Capability-Aware Model Selection Demo")
	fmt.Println()
	fmt.Printf("Provider: %s\n\n", providerName)

	ctx := context.Background()

	// Step: Introduction
	if config.StepPause("Understanding capability detection...", []string{
		"nxusKit: ModelInfo provides normalized metadata across providers",
		"Capabilities include: modalities, context window, vision support",
		"ListModels() returns detailed info for all available models",
		"Filter models by capability rather than hardcoding model names",
	}) == interactive.ActionQuit {
		return
	}

	// Step: Creating provider
	if config.StepPause("Creating "+providerName+" provider...", []string{
		"nxusKit: Provider creation based on argument",
		"API key loaded from environment variable",
		"For Ollama, no API key needed (local provider)",
	}) == interactive.ActionQuit {
		return
	}

	// Create provider based on argument
	start := time.Now()
	models, err := getModels(ctx, providerName)
	elapsedMs := time.Since(start).Milliseconds()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Verbose: Show the response metadata
	if config.IsVerbose() {
		fmt.Printf("[nxusKit] ListModels completed in %dms\n", elapsedMs)
		fmt.Printf("[nxusKit] Retrieved %d models from %s\n\n", len(models), providerName)
	}

	// Step: Displaying models
	if config.StepPause("Listing all models with capabilities...", []string{
		"nxusKit: ModelInfo includes Name, Modalities, SupportsVision",
		"Context window is normalized across providers",
		"Modalities show what input types the model accepts",
	}) == interactive.ActionQuit {
		return
	}

	// Display all models with their capabilities
	fmt.Println("All Available Models:")
	fmt.Println()
	fmt.Printf("%-40s %-20s %-15s %-15s\n", "Model", "Modalities", "Vision", "Context")
	fmt.Println(strings.Repeat("=", 90))

	for _, model := range models {
		modalities := strings.Join(model.Modalities(), ", ")
		vision := "No"
		if model.SupportsVision() {
			vision = "Yes"
		}
		context := model.FormattedContextWindow()
		if context == "" {
			context = "Unknown"
		}

		fmt.Printf("%-40s %-20s %-15s %-15s\n",
			truncate(model.Name, 40),
			truncate(modalities, 20),
			vision,
			context,
		)
	}

	// Step: Filtering by capability
	if config.StepPause("Filtering models by capability...", []string{
		"nxusKit: SupportsVision() is a convenience method on ModelInfo",
		"Filter slice with simple Go iteration",
		"Same pattern works for any capability check",
	}) == interactive.ActionQuit {
		return
	}

	// nxusKit: Filter models by capability - works identically across all providers
	var visionModels []nxuskit.ModelInfo
	for _, m := range models {
		if m.SupportsVision() {
			visionModels = append(visionModels, m)
		}
	}

	fmt.Println()
	fmt.Printf("\nVision-Capable Models (%d):\n\n", len(visionModels))

	if len(visionModels) == 0 {
		fmt.Println("   No vision-capable models found.")
		if providerName == "ollama" {
			fmt.Println("\n   Tip: If you have vision models installed (like llava),")
			fmt.Println("      vision detection depends on model metadata.")
		}
	} else {
		for _, model := range visionModels {
			fmt.Printf("   - %s\n", model.Name)
			fmt.Printf("     Modalities: %v\n", model.Modalities())
			if ctx := model.FormattedContextWindow(); ctx != "" {
				fmt.Printf("     Context window: %s\n", ctx)
			}
			fmt.Println()
		}
	}

	// Step: Task-based selection
	if config.StepPause("Demonstrating task-based model selection...", []string{
		"Match model capabilities to task requirements",
		"Task 1: Document with images -> vision + large context",
		"Task 2: Simple text -> any text model (cheaper)",
	}) == interactive.ActionQuit {
		return
	}

	// Example: Selecting the best model for a specific task
	fmt.Println("\nTask-Based Model Selection Examples:")
	fmt.Println()

	// Task 1: Need a vision model with large context
	fmt.Println("Task 1: Analyze a document with multiple images")
	var bestVisionModel *nxuskit.ModelInfo
	var maxContext int
	for i := range visionModels {
		m := &visionModels[i]
		if m.ContextWindow != nil && *m.ContextWindow >= 100000 {
			if *m.ContextWindow > maxContext {
				maxContext = *m.ContextWindow
				bestVisionModel = m
			}
		}
	}
	if bestVisionModel != nil {
		fmt.Printf("   Recommended: %s\n", bestVisionModel.Name)
		fmt.Printf("   Reason: Vision support + large context (%s)\n", bestVisionModel.FormattedContextWindow())
	} else {
		fmt.Println("   No vision models with 100K+ context found")
	}

	// Task 2: Need a text-only model for simple queries
	fmt.Println("\nTask 2: Simple text generation (no images needed)")
	for _, model := range models {
		if !model.SupportsVision() {
			fmt.Printf("   Recommended: %s\n", model.Name)
			fmt.Println("   Reason: Text-only model (faster and cheaper for text tasks)")
			break
		}
	}
	if len(models) > 0 && models[0].SupportsVision() && len(visionModels) == len(models) {
		fmt.Println("   All available models support vision")
	}

	// Step: Modality filtering
	if config.StepPause("Filtering by modality...", []string{
		"Modalities() returns slice of supported input types",
		"Common modalities: text, vision, audio",
		"Combine modality checks for multimodal requirements",
	}) == interactive.ActionQuit {
		return
	}

	// Demonstrate filtering by modality
	fmt.Println("\n\nAdvanced: Filtering by Modality:")
	fmt.Println()
	multimodalCount := 0
	for _, m := range models {
		mods := m.Modalities()
		hasText := false
		hasVision := false
		for _, mod := range mods {
			if mod == "text" {
				hasText = true
			}
			if mod == "vision" {
				hasVision = true
			}
		}
		if hasText && hasVision {
			multimodalCount++
		}
	}

	fmt.Printf("   Multimodal models (text + vision): %d\n", multimodalCount)
	fmt.Printf("   Text-only models: %d\n", len(models)-multimodalCount)

	// Example code snippet for users
	fmt.Println("\n\nExample Code:")
	fmt.Println()
	fmt.Println("```go")
	fmt.Println("// Filter for vision-capable models")
	fmt.Println("var visionModels []nxuskit.ModelInfo")
	fmt.Println("for _, m := range models {")
	fmt.Println("    if m.SupportsVision() {")
	fmt.Println("        visionModels = append(visionModels, m)")
	fmt.Println("    }")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("// Check modalities")
	fmt.Println("for _, model := range models {")
	printfLine := "    fmt.Printf" + `("%s supports: %v\n", model.Name, model.Modalities())`
	fmt.Println(printfLine)
	fmt.Println("}")
	fmt.Println("```")
}

func getModels(ctx context.Context, providerName string) ([]nxuskit.ModelInfo, error) {
	switch providerName {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
		}
		provider, err := nxuskit.NewOpenAIFFIProvider(nxuskit.WithOpenAIAPIKey(apiKey))
		if err != nil {
			return nil, err
		}
		return provider.ListModels(ctx)

	case "claude":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
		}
		provider, err := nxuskit.NewClaudeFFIProvider(nxuskit.WithClaudeAPIKey(apiKey))
		if err != nil {
			return nil, err
		}
		return provider.ListModels(ctx)

	case "ollama":
		provider, err := nxuskit.NewOllamaFFIProvider()
		if err != nil {
			return nil, err
		}
		return provider.ListModels(ctx)

	default:
		return nil, fmt.Errorf("unknown provider: %s. Supported: openai, claude, ollama", providerName)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
