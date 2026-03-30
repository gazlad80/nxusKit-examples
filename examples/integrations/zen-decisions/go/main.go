//go:build nxuskit

// Example: ZEN Decision Tables -- Personality Variants & Hit Policies
//
// ## nxusKit Features Demonstrated
// - ZEN JSON Decision Model (JDM) evaluation via C ABI
// - Decision tables with "first" hit policy (maze-rat)
// - Decision tables with "collect" hit policy (potion)
// - Expression nodes for computed outputs (food-truck)
// - Personality variant comparison (same input, different models)
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw JSON results
// - `--step` or `-s`: Pause at each pipeline step with explanations
//
// ## Scenario Selection
// - `--scenario <name>`: Load a scenario from ../scenarios/<name>/
// - Available scenarios: maze-rat, potion, food-truck
//
// Usage:
//
//	go run . --scenario maze-rat
//	go run . --scenario potion --verbose
//	go run . --scenario food-truck --step
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func main() {
	config := interactive.FromArgs()

	scenario := flagValue("--scenario")
	if scenario == "" {
		scenario = flagValue("-scenario")
	}
	if scenario == "" {
		scenario = "maze-rat"
	}

	scenariosDir := filepath.Join("..", "scenarios")
	if _, err := os.Stat(scenariosDir); os.IsNotExist(err) {
		// Try alternative paths
		for _, alt := range []string{"scenarios", "examples/integrations/zen-decisions/scenarios"} {
			if _, err := os.Stat(alt); err == nil {
				scenariosDir = alt
				break
			}
		}
	}

	available := availableScenarios(scenariosDir)
	found := false
	for _, s := range available {
		if s == scenario {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "Error: unknown scenario %q\n", scenario)
		if len(available) > 0 {
			fmt.Fprintf(os.Stderr, "Available scenarios: %s\n", strings.Join(available, ", "))
		}
		os.Exit(1)
	}

	totalStart := time.Now()

	switch scenario {
	case "maze-rat":
		runMazeRat(scenariosDir, config)
	case "potion":
		runPotion(scenariosDir, config)
	case "food-truck":
		runFoodTruck(scenariosDir, config)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown scenario %q\n", scenario)
		os.Exit(1)
	}

	totalElapsed := time.Since(totalStart)

	fmt.Println("=== Summary ===")
	fmt.Printf("Scenario:   %s\n", scenario)
	fmt.Printf("Total time: %dms\n", totalElapsed.Milliseconds())
	fmt.Println()
	fmt.Println("Done.")
}

// ── Scenario Runners ────────────────────────────────────────────────

func runMazeRat(scenariosDir string, config *interactive.Config) {
	scenarioDir := filepath.Join(scenariosDir, "maze-rat")
	inputJSON := mustReadFile(filepath.Join(scenarioDir, "input.json"))

	var input map[string]interface{}
	mustUnmarshal(inputJSON, &input)

	fmt.Println("========================================")
	fmt.Println("  ZEN Decision Tables: Maze Rat")
	fmt.Println("========================================")
	fmt.Println("Personality variant comparison with first-hit policy")
	fmt.Println()

	fmt.Println("Input:")
	printResultFields(input, "  ")
	fmt.Println()

	// Evaluate all 3 personality JDMs
	type personality struct {
		name     string
		filename string
	}
	personalities := []personality{
		{"cautious", "decision-model.json"},
		{"greedy", "greedy.json"},
		{"explorer", "explorer.json"},
	}

	type evalResult struct {
		name    string
		result  map[string]interface{}
		elapsed time.Duration
	}
	var results []evalResult

	for _, p := range personalities {
		fmt.Printf("--- Personality: %s ---\n", p.name)

		if config.StepPause(
			fmt.Sprintf("Evaluating %s personality decision table...", p.name),
			[]string{
				"Loads the JDM file for this personality variant",
				"Evaluates against the same input using first-hit policy",
				"First matching rule determines the action",
			},
		) == interactive.ActionQuit {
			return
		}

		modelJSON := mustReadFile(filepath.Join(scenarioDir, p.filename))

		start := time.Now()
		result, err := nxuskit.ZenEvaluate(string(modelJSON), string(inputJSON))
		elapsed := time.Since(start)

		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
			fmt.Println()
			continue
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			fmt.Fprintf(os.Stderr, "  Error parsing result: %v\n", err)
			fmt.Println()
			continue
		}

		fmt.Println("  Result:")
		printResultFields(parsed, "    ")
		fmt.Printf("  Time: %dus\n", elapsed.Microseconds())

		if config.IsVerbose() {
			raw, _ := json.MarshalIndent(parsed, "  ", "  ")
			fmt.Printf("\n  [verbose] Raw result:\n  %s\n", raw)
		}

		results = append(results, evalResult{p.name, parsed, elapsed})
		fmt.Println()
	}

	// Compare personality outcomes
	fmt.Println("--- Personality Comparison ---")

	if config.StepPause(
		"Comparing decisions across personality variants...",
		[]string{
			"Same input, different decision tables produce different actions",
			"Cautious avoids risk, greedy follows scent, explorer seeks new paths",
			"Confidence values reflect each personality's certainty",
		},
	) == interactive.ActionQuit {
		return
	}

	fmt.Printf("  %-12s %-15s %-12s %8s\n", "Personality", "Action", "Confidence", "Time")
	fmt.Printf("  %s\n", strings.Repeat("-", 50))

	for _, r := range results {
		action := "?"
		if v, ok := r.result["action"]; ok {
			action = fmt.Sprintf("%v", v)
		}
		confidence := 0.0
		if v, ok := r.result["confidence"]; ok {
			if f, ok := v.(float64); ok {
				confidence = f
			}
		}
		fmt.Printf("  %-12s %-15s %-12.2f %5dus\n",
			r.name, action, confidence, r.elapsed.Microseconds())
	}
	fmt.Println()
}

func runPotion(scenariosDir string, config *interactive.Config) {
	scenarioDir := filepath.Join(scenariosDir, "potion")
	inputJSON := mustReadFile(filepath.Join(scenarioDir, "input.json"))
	modelJSON := mustReadFile(filepath.Join(scenarioDir, "decision-model.json"))

	var input map[string]interface{}
	mustUnmarshal(inputJSON, &input)

	fmt.Println("========================================")
	fmt.Println("  ZEN Decision Tables: Potion Recipes")
	fmt.Println("========================================")
	fmt.Println("Collect hit policy -- returns all matching recipes")
	fmt.Println()

	fmt.Println("Input:")
	printResultFields(input, "  ")
	fmt.Println()

	fmt.Println("--- Evaluate Potion Recipes ---")

	if config.StepPause(
		"Evaluating potion decision table with collect hit policy...",
		[]string{
			"Collect hit policy returns ALL matching rules, not just the first",
			"Multiple recipes can match the same input",
			"Each result includes recipe name, steps, and warnings",
		},
	) == interactive.ActionQuit {
		return
	}

	start := time.Now()
	result, err := nxuskit.ZenEvaluate(string(modelJSON), string(inputJSON))
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}

	// Collect policy may return an array or a single object
	var rawResult interface{}
	if err := json.Unmarshal([]byte(result), &rawResult); err != nil {
		fmt.Fprintf(os.Stderr, "  Error parsing result: %v\n", err)
		return
	}

	switch v := rawResult.(type) {
	case []interface{}:
		fmt.Printf("  Matching recipes: %d\n\n", len(v))
		for i, item := range v {
			fmt.Printf("  Recipe %d:\n", i+1)
			if m, ok := item.(map[string]interface{}); ok {
				printResultFields(m, "    ")
			}
			fmt.Println()
		}
	case map[string]interface{}:
		fmt.Println("  Result:")
		printResultFields(v, "    ")
		fmt.Println()
	}

	fmt.Printf("  Time: %dus\n", elapsed.Microseconds())

	if config.IsVerbose() {
		fmt.Printf("\n  [verbose] Raw result:\n  %s\n", result)
	}
	fmt.Println()
}

func runFoodTruck(scenariosDir string, config *interactive.Config) {
	scenarioDir := filepath.Join(scenariosDir, "food-truck")
	inputJSON := mustReadFile(filepath.Join(scenarioDir, "input.json"))
	modelJSON := mustReadFile(filepath.Join(scenarioDir, "decision-model.json"))

	var input map[string]interface{}
	mustUnmarshal(inputJSON, &input)

	fmt.Println("========================================")
	fmt.Println("  ZEN Decision Tables: Food Truck Planner")
	fmt.Println("========================================")
	fmt.Println("Decision table + expression node pipeline")
	fmt.Println()

	fmt.Println("Input:")
	printResultFields(input, "  ")
	fmt.Println()

	fmt.Println("--- Evaluate Food Truck Decision ---")

	if config.StepPause(
		"Evaluating food truck decision pipeline...",
		[]string{
			"Decision table selects location and base price multiplier",
			"Expression node computes menu adjustment and restock alert",
			"Pipeline: inputNode -> decisionTableNode -> expressionNode -> outputNode",
		},
	) == interactive.ActionQuit {
		return
	}

	start := time.Now()
	result, err := nxuskit.ZenEvaluate(string(modelJSON), string(inputJSON))
	elapsed := time.Since(start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
		return
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "  Error parsing result: %v\n", err)
		return
	}

	fmt.Println("  Decision output:")
	fmt.Printf("    location:         %v\n", parsed["location"])
	fmt.Printf("    price_multiplier: %v\n", parsed["price_multiplier"])
	fmt.Printf("    menu_adjustment:  %v\n", parsed["menu_adjustment"])
	fmt.Printf("    restock_alert:    %v\n", parsed["restock_alert"])
	fmt.Println()
	fmt.Printf("  Time: %dus\n", elapsed.Microseconds())

	if config.IsVerbose() {
		raw, _ := json.MarshalIndent(parsed, "  ", "  ")
		fmt.Printf("\n  [verbose] Raw result:\n  %s\n", raw)
	}
	fmt.Println()
}

// ── Helper Functions ────────────────────────────────────────────────

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
		os.Exit(1)
	}
	return data
}

func mustUnmarshal(data []byte, v interface{}) {
	if err := json.Unmarshal(data, v); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}
}

func printResultFields(m map[string]interface{}, indent string) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s%s: %v\n", indent, k, m[k])
	}
}

func availableScenarios(scenariosDir string) []string {
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		inputFile := filepath.Join(scenariosDir, e.Name(), "input.json")
		if _, err := os.Stat(inputFile); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

func flagValue(name string) string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, name+"=") {
			return strings.TrimPrefix(arg, name+"=")
		}
	}
	return ""
}
