// Package main demonstrates basic CLIPS integration with nxuskit.
//
// CLIPS (C Language Integrated Production System) is an expert system tool
// that provides a complete environment for constructing rule-based systems.
// This example shows how to use the nxuskit CLIPS provider for:
//   - Loading rule files
//   - Asserting facts
//   - Running inference
//   - Extracting conclusions
//
// Usage:
//
//	go run .
//
// See ../rust/src/ for Rust reference implementations:
//   - animal_classification.rs: Animal classification using CLIPS rules
//   - basic.rs: Basic CLIPS integration overview
//   - inventory.rs: Inventory management with CLIPS
//   - medical_triage.rs: Medical triage using rule-based reasoning
//   - pipeline.rs: CLIPS pipeline processing
//   - scheduler.rs: Task scheduling with CLIPS
//
// ## Interactive Modes
//
// This example supports interactive debugging modes:
//
//	--verbose, -v    Show raw request/response data
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
	"encoding/json"
	"fmt"
	"os"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// config is set in main() and used for verbose/step output
var config *interactive.Config

func main() {
	config = interactive.FromArgs()

	fmt.Println("=== CLIPS Expert System Integration Demo ===")
	fmt.Println()

	ctx := context.Background()

	// Step: Create provider
	action := config.StepPause("Creating CLIPS provider...", []string{
		"CLIPS is a rule-based expert system",
		"Provides deterministic, auditable rule execution",
		"Looking for .clp rule files in rules directory",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Create CLIPS provider
	// The rules directory would typically contain .clp rule files
	rulesDir := "../shared/rules"
	if _, err := os.Stat(rulesDir); os.IsNotExist(err) {
		rulesDir = "./rules" // Try local directory
	}

	provider, err := nxuskit.NewClipsFFIProvider(rulesDir)
	if err != nil {
		if llmErr, ok := err.(*nxuskit.LLMError); ok {
			fmt.Printf("CLIPS provider error: %s\n", llmErr.Message)
		} else {
			fmt.Printf("Failed to create CLIPS provider: %v\n", err)
		}
		fmt.Println("\nRunning in demo mode instead...")
		fmt.Println()
		runDemoMode()
		return
	}

	// Step: Example 1
	action = config.StepPause("Running Example 1: Animal Classification...", []string{
		"Will classify a dog based on its characteristics",
		"Facts: has-backbone=yes, body-temperature=warm, has-fur=yes",
		"CLIPS rules will derive the animal classification",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Example 1: Animal Classification
	fmt.Println("--- Example 1: Animal Classification ---")
	fmt.Println()

	// Build fact input for a dog
	dogInput := clipsInputWire{
		Facts: []clipsFactWire{
			{
				Template: "animal",
				Values: map[string]interface{}{
					"name":             "Buddy",
					"has-backbone":     map[string]string{"symbol": "yes"},
					"body-temperature": map[string]string{"symbol": "warm"},
					"has-feathers":     map[string]string{"symbol": "no"},
					"has-fur":          map[string]string{"symbol": "yes"},
					"has-scales":       map[string]string{"symbol": "no"},
					"lives-in-water":   map[string]string{"symbol": "no"},
					"can-fly":          map[string]string{"symbol": "no"},
					"lays-eggs":        map[string]string{"symbol": "no"},
				},
			},
		},
	}

	inputJSON, _ := json.Marshal(dogInput)

	req, err := nxuskit.NewChatRequest("animal-classification",
		nxuskit.WithMessages(nxuskit.UserMessage(string(inputJSON))),
	)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	// Verbose: Show request details
	config.PrintRequest("CLIPS", "Chat (animal-classification)", dogInput)

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		fmt.Printf("CLIPS execution failed: %v\n", err)
		return
	}

	// Parse and display results
	var output clipsOutputWire
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		fmt.Printf("Failed to parse output: %v\n", err)
		return
	}

	// Verbose: Show response
	config.PrintResponse(200, int64(output.Stats.ExecutionTimeMs), output)

	fmt.Printf("Dog classification results:\n")
	fmt.Printf("  Rules fired: %d\n", output.Stats.TotalRulesFired)
	fmt.Printf("  Conclusions: %d\n", output.Stats.ConclusionsCount)
	fmt.Printf("  Time: %dms\n", output.Stats.ExecutionTimeMs)
	fmt.Println()

	for _, c := range output.Conclusions {
		fmt.Printf("  - %s: %v\n", c.Template, c.Values)
	}
	fmt.Println()

	// Step: Example 2
	action = config.StepPause("Running Example 2: Multiple Animals...", []string{
		"Will classify Frog, Penguin, and Spider (same facts as Rust animal_classification.rs)",
		"Demonstrates batch inference with CLIPS",
	})
	if action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Example 2: Multiple animals
	fmt.Println("--- Example 2: Multiple Animals ---")
	fmt.Println()

	// Same multi-animal payload as rust/src/animal_classification.rs (Frog, Penguin, Spider).
	multiInput := clipsInputWire{
		Facts: []clipsFactWire{
			{
				Template: "animal",
				Values: map[string]interface{}{
					"name":             "Frog",
					"has-backbone":     map[string]string{"symbol": "yes"},
					"body-temperature": map[string]string{"symbol": "cold"},
					"has-feathers":     map[string]string{"symbol": "no"},
					"has-fur":          map[string]string{"symbol": "no"},
					"has-scales":       map[string]string{"symbol": "no"},
					"lives-in-water":   map[string]string{"symbol": "partial"},
					"can-fly":          map[string]string{"symbol": "no"},
					"lays-eggs":        map[string]string{"symbol": "yes"},
				},
			},
			{
				Template: "animal",
				Values: map[string]interface{}{
					"name":             "Penguin",
					"has-backbone":     map[string]string{"symbol": "yes"},
					"body-temperature": map[string]string{"symbol": "warm"},
					"has-feathers":     map[string]string{"symbol": "yes"},
					"has-fur":          map[string]string{"symbol": "no"},
					"has-scales":       map[string]string{"symbol": "no"},
					"lives-in-water":   map[string]string{"symbol": "partial"},
					"can-fly":          map[string]string{"symbol": "no"},
					"lays-eggs":        map[string]string{"symbol": "yes"},
				},
			},
			{
				Template: "animal",
				Values: map[string]interface{}{
					"name":             "Spider",
					"has-backbone":     map[string]string{"symbol": "no"},
					"body-temperature": map[string]string{"symbol": "cold"},
					"has-feathers":     map[string]string{"symbol": "no"},
					"has-fur":          map[string]string{"symbol": "no"},
					"has-scales":       map[string]string{"symbol": "no"},
					"lives-in-water":   map[string]string{"symbol": "no"},
					"can-fly":          map[string]string{"symbol": "no"},
					"lays-eggs":        map[string]string{"symbol": "yes"},
				},
			},
		},
	}

	inputJSON, _ = json.Marshal(multiInput)
	req, _ = nxuskit.NewChatRequest("animal-classification",
		nxuskit.WithMessages(nxuskit.UserMessage(string(inputJSON))),
	)

	// Verbose: Show request details
	config.PrintRequest("CLIPS", "Chat (animal-classification)", multiInput)

	resp, err = provider.Chat(ctx, req)
	if err != nil {
		fmt.Printf("CLIPS execution failed: %v\n", err)
		return
	}

	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		fmt.Printf("Failed to parse output: %v\n", err)
		return
	}

	// Verbose: Show response
	config.PrintResponse(200, int64(output.Stats.ExecutionTimeMs), output)

	fmt.Printf("Multiple animal results:\n")
	fmt.Printf("  Total conclusions: %d\n", output.Stats.ConclusionsCount)
	for _, c := range output.Conclusions {
		if name, ok := c.Values["name"]; ok {
			fmt.Printf("  - %s classified as: %v\n", name, c.Template)
		}
	}

	fmt.Println()
	fmt.Println("=== CLIPS Demo Complete ===")
}

// runDemoMode demonstrates API patterns without executing rules
func runDemoMode() {
	fmt.Println("=== CLIPS API Pattern Demo ===")
	fmt.Println()

	// Show how to structure CLIPS input
	fmt.Println("CLIPS Input Structure:")
	fmt.Println("----------------------")

	incDemo, derDemo := true, true
	maxRulesDemo := int64(1000)
	exampleInput := clipsInputWire{
		Facts: []clipsFactWire{
			{
				Template: "animal",
				Values: map[string]interface{}{
					"name":         "Buddy",
					"has-backbone": map[string]string{"symbol": "yes"},
					"has-fur":      map[string]string{"symbol": "yes"},
				},
			},
		},
		Templates: []clipsTemplateWire{
			{
				Name: "classification",
				Slots: []clipsSlotWire{
					{Name: "animal-name", Type: "STRING"},
					{Name: "category", Type: "SYMBOL"},
					{Name: "confidence", Type: "SYMBOL"},
				},
			},
		},
		Config: &clipsRequestConfigWire{
			IncludeTrace:   &incDemo,
			MaxRules:       &maxRulesDemo,
			DerivedOnlyNew: &derDemo,
		},
	}

	inputJSON, _ := json.MarshalIndent(exampleInput, "", "  ")
	fmt.Println(string(inputJSON))
	fmt.Println()

	// Show expected output structure
	fmt.Println("Expected CLIPS Output Structure:")
	fmt.Println("--------------------------------")

	exampleOutput := clipsOutputWire{
		Conclusions: []clipsConclusionWire{
			{
				Template: "classification",
				Values: map[string]interface{}{
					"animal-name": "Buddy",
					"category":    "mammal",
					"confidence":  "high",
				},
				FactIndex: 2,
				Derived:   true,
			},
		},
		Stats: clipsExecStatsWire{
			TotalRulesFired:  5,
			ConclusionsCount: 1,
			ExecutionTimeMs:  2,
		},
		Trace: &clipsTraceWire{
			RulesFired: []clipsRuleFiringWire{
				{RuleName: "identify-mammal", FireCount: 1},
				{RuleName: "classify-by-fur", FireCount: 1},
			},
		},
	}

	outputJSON, _ := json.MarshalIndent(exampleOutput, "", "  ")
	fmt.Println(string(outputJSON))
	fmt.Println()

	// Show usage code
	fmt.Println("Go Usage Example:")
	fmt.Println("-----------------")
	fmt.Println(`// Create provider (FFI-backed)
provider, err := nxuskit.NewClipsFFIProvider("./rules")

// Build input — local wire types in clips_wire.go (JSON matches nxuskit CLIPS engine)
input := clipsInputWire{
    Facts: []clipsFactWire{
        {Template: "animal", Values: map[string]interface{}{...}},
    },
}
inputJSON, _ := json.Marshal(input)

// Execute rules
req, _ := nxuskit.NewChatRequest("animal-classification",
    nxuskit.WithMessages(nxuskit.UserMessage(string(inputJSON))),
)
resp, err := provider.Chat(ctx, req)

// Parse conclusions
var output clipsOutputWire
json.Unmarshal([]byte(resp.Content), &output)`)

	fmt.Println()
	fmt.Println("Available Rust Examples (../rust/src/):")
	fmt.Println("  - animal_classification.rs: Animal classification using CLIPS rules")
	fmt.Println("  - basic.rs: Basic CLIPS integration overview")
	fmt.Println("  - inventory.rs: Inventory management with CLIPS")
	fmt.Println("  - medical_triage.rs: Medical triage using rule-based reasoning")
	fmt.Println("  - pipeline.rs: CLIPS pipeline processing")
	fmt.Println("  - scheduler.rs: Task scheduling with CLIPS")
	fmt.Println()
	fmt.Println("To run CLIPS inference:")
	fmt.Println("  go run .")
}
