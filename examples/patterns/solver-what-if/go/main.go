//go:build nxuskit

// Example: Solver What-If — Push/Pop Scoping & Assumption-Based Solving
//
// ## nxusKit Features Demonstrated
// - SolverSession lifecycle (create, close)
// - Variable, constraint, and objective model building
// - Optimal base solve
// - Push/Pop scoping for what-if analysis
// - Explanation / unsat-core retrieval
// - Delta comparison (base vs. what-if)
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show solver stats and intermediate details
// - `--step` or `-s`: Pause at each phase with explanations
//
// ## Scenario Selection
// - `--scenario <name>`: Load a problem from ../scenarios/<name>/problem.json
// - Available scenarios: wedding, mars, recipe
//
// Usage:
//
//	go run . --scenario wedding
//	go run . --scenario mars --verbose
//	go run . --scenario recipe --step
//
// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// Problem represents the JSON structure of a scenario problem file.
type Problem struct {
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Variables       []nxuskit.VariableDef   `json:"variables"`
	Constraints     []nxuskit.ConstraintDef `json:"constraints"`
	Objectives      []nxuskit.ObjectiveDef  `json:"objectives"`
	WhatIfScenarios []WhatIfScenario        `json:"what_if_scenarios"`
}

// WhatIfScenario describes a push/pop what-if analysis scenario.
type WhatIfScenario struct {
	Name                  string                  `json:"name"`
	Description           string                  `json:"description"`
	AdditionalConstraints []nxuskit.ConstraintDef `json:"additional_constraints"`
}

// scenarioResult holds the outcome of a scenario for the final summary.
type scenarioResult struct {
	name           string
	status         nxuskit.SolveStatus
	objectiveValue *float64
}

func main() {
	// Parse interactive mode flags (consumes --verbose/-v and --step/-s)
	config := interactive.FromArgs()

	// Parse --scenario flag manually (flag package already parsed by FromArgs)
	scenario := flagValue("--scenario")
	if scenario == "" {
		scenario = flagValue("-scenario")
	}
	if scenario == "" {
		scenario = "wedding"
	}

	// Resolve scenario path relative to the binary location
	scenarioDir := filepath.Join("..", "scenarios", scenario)
	problemPath := filepath.Join(scenarioDir, "problem.json")

	// Load and parse the problem file
	data, err := os.ReadFile(problemPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot load scenario %q: %v\n", scenario, err)
		fmt.Fprintln(os.Stderr)
		listAvailableScenarios()
		os.Exit(1)
	}

	var problem Problem
	if err := json.Unmarshal(data, &problem); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid problem.json: %v\n", err)
		os.Exit(1)
	}

	// Ensure constraint parameters are never nil (C ABI requires the field)
	ensureConstraintParams(problem.Constraints)
	for i := range problem.WhatIfScenarios {
		ensureConstraintParams(problem.WhatIfScenarios[i].AdditionalConstraints)
	}

	// ── Print problem summary ────────────────────────────────────
	fmt.Println("========================================")
	fmt.Printf("  Solver What-If: %s\n", problem.Name)
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println(problem.Description)
	fmt.Println()
	fmt.Printf("Variables:          %d\n", len(problem.Variables))
	fmt.Printf("Hard constraints:   %d\n", len(problem.Constraints))
	fmt.Printf("What-if scenarios:  %d\n", len(problem.WhatIfScenarios))
	fmt.Println()

	// Step: introduction
	if config.StepPause("Problem loaded. Creating solver session...", []string{
		"nxusKit: SolverSession wraps the Z3 constraint solver via C ABI",
		"Push/Pop enables reversible what-if exploration",
		"Explanation() retrieves unsat cores when constraints conflict",
	}) == interactive.ActionQuit {
		return
	}

	// ── Create solver session ────────────────────────────────────
	// TODO(v0.8.1): Migrate to FFI solver when available
	session, err := nxuskit.NewSolverSession(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating solver session: %v\n", err)
		os.Exit(1)
	}
	defer session.Close()

	// ── Add variables ────────────────────────────────────────────
	if err := session.AddVariables(problem.Variables); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding variables: %v\n", err)
		os.Exit(1)
	}

	if config.IsVerbose() {
		fmt.Println("[nxusKit] Variables added:")
		for _, v := range problem.Variables {
			label := v.Label
			if label == "" {
				label = string(v.VarType)
			}
			fmt.Printf("  - %s (%s): %s\n", v.Name, v.VarType, label)
		}
		fmt.Println()
	}

	// ── Add hard constraints ─────────────────────────────────────
	fmt.Printf("Adding %d hard constraint(s)...\n", len(problem.Constraints))
	if err := session.AddConstraints(problem.Constraints); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding constraints: %v\n", err)
		os.Exit(1)
	}

	// ── Set objective ────────────────────────────────────────────
	if len(problem.Objectives) > 0 {
		obj := problem.Objectives[0]
		fmt.Printf("Setting objective: %s %s\n", obj.Direction, obj.Name)
		if err := session.SetObjective(obj); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting objective: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println()

	// ── Solve base problem ───────────────────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  Base Problem")
	fmt.Println("----------------------------------------")
	fmt.Println()

	if config.StepPause("Solving the base problem with all hard constraints and objective...", []string{
		"This establishes the baseline optimal solution",
		"What-if scenarios will be compared against this result",
	}) == interactive.ActionQuit {
		return
	}

	explain := true
	baseResult, err := session.Solve(&nxuskit.SolverConfig{ProduceExplanation: &explain})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error solving: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", baseResult.Status)
	if baseResult.Status == nxuskit.SolveStatusSat || baseResult.Status == nxuskit.SolveStatusOptimal {
		printAssignments(baseResult.Assignments)
	}
	if baseResult.ObjectiveValue != nil {
		fmt.Printf("Objective value: %.0f\n", *baseResult.ObjectiveValue)
	}
	printStats(config, baseResult.Stats)
	fmt.Println()

	baseAssignments := extractAssignments(baseResult.Assignments)

	var results []scenarioResult
	results = append(results, scenarioResult{
		name:           "Base",
		status:         baseResult.Status,
		objectiveValue: baseResult.ObjectiveValue,
	})

	// ── What-If Scenarios ────────────────────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  What-If Analysis")
	fmt.Println("----------------------------------------")
	fmt.Println()

	for i, ws := range problem.WhatIfScenarios {
		fmt.Printf("Scenario %d: \"%s\"\n", i+1, ws.Name)
		fmt.Printf("  %s\n\n", ws.Description)

		if config.StepPause(fmt.Sprintf("What-if: \"%s\"", ws.Name), []string{
			ws.Description,
			"Push saves the current model state",
			"Additional constraints are added temporarily",
			"Pop restores the base model after the experiment",
		}) == interactive.ActionQuit {
			return
		}

		// Push scope
		fmt.Println("  Push scope...")
		if err := session.Push(); err != nil {
			fmt.Fprintf(os.Stderr, "  Error pushing scope: %v\n", err)
			os.Exit(1)
		}

		// Add what-if constraints
		fmt.Printf("  Adding %d temporary constraint(s)...\n", len(ws.AdditionalConstraints))

		if config.IsVerbose() {
			for _, c := range ws.AdditionalConstraints {
				label := c.Label
				if label == "" {
					label = c.Name
				}
				fmt.Printf("  [nxusKit] %s (%s)\n", label, c.ConstraintType)
			}
		}

		if err := session.AddConstraints(ws.AdditionalConstraints); err != nil {
			fmt.Fprintf(os.Stderr, "  Error adding what-if constraints: %v\n", err)
			os.Exit(1)
		}

		// Solve under what-if constraints
		fmt.Println("  Solving...")
		wiResult, err := session.Solve(&nxuskit.SolverConfig{ProduceExplanation: &explain})
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error solving what-if: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("  Status: %s\n", wiResult.Status)

		if wiResult.Status == nxuskit.SolveStatusUnsat {
			// Try to get explanation / unsat core
			fmt.Println("  Attempting to retrieve explanation...")
			expl, err := session.Explanation()
			if err == nil && expl != nil {
				if len(expl.UnsatCoreLabels) > 0 {
					fmt.Printf("  Unsat core: [%s]\n", strings.Join(expl.UnsatCoreLabels, ", "))
				}
				if config.IsVerbose() {
					explJSON, _ := json.MarshalIndent(expl, "  ", "  ")
					fmt.Printf("  [verbose] Explanation: %s\n", string(explJSON))
				}
			} else {
				fmt.Println("  (no explanation available)")
			}

			results = append(results, scenarioResult{
				name:           ws.Name,
				status:         wiResult.Status,
				objectiveValue: nil,
			})
		} else {
			if wiResult.Status == nxuskit.SolveStatusSat || wiResult.Status == nxuskit.SolveStatusOptimal {
				printAssignmentsIndented(wiResult.Assignments, "    ")
			}

			if wiResult.ObjectiveValue != nil {
				fmt.Printf("  Objective value: %.0f\n", *wiResult.ObjectiveValue)
			}

			// Show delta from base
			wiAssignments := extractAssignments(wiResult.Assignments)
			fmt.Println("  Delta from base:")
			printDelta(baseAssignments, wiAssignments, "    ")

			// Objective delta
			if baseResult.ObjectiveValue != nil && wiResult.ObjectiveValue != nil {
				diff := *wiResult.ObjectiveValue - *baseResult.ObjectiveValue
				sign := ""
				if diff > 0 {
					sign = "+"
				}
				fmt.Printf("  Objective delta: %.0f -> %.0f (%s%.0f)\n",
					*baseResult.ObjectiveValue, *wiResult.ObjectiveValue, sign, diff)
			}

			printStats(config, wiResult.Stats)

			results = append(results, scenarioResult{
				name:           ws.Name,
				status:         wiResult.Status,
				objectiveValue: wiResult.ObjectiveValue,
			})
		}

		// Pop scope to restore original model
		fmt.Println("  Pop scope (restoring base model)")
		if err := session.Pop(); err != nil {
			fmt.Fprintf(os.Stderr, "  Error popping scope: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	}

	// ── Summary ──────────────────────────────────────────────────
	fmt.Println("========================================")
	fmt.Printf("  Summary: %s (%d what-if variants)\n", problem.Name, len(problem.WhatIfScenarios))
	fmt.Println("========================================")
	fmt.Println()

	nameWidth := 10
	for _, r := range results {
		if len(r.name) > nameWidth {
			nameWidth = len(r.name)
		}
	}

	fmt.Printf("  %-*s  %10s  %15s\n", nameWidth, "Variant", "Status", "Objective")
	fmt.Printf("  %s  %s  %s\n",
		strings.Repeat("-", nameWidth),
		strings.Repeat("-", 10),
		strings.Repeat("-", 15))

	for _, r := range results {
		icon := statusIcon(r.status)
		objStr := "-"
		if r.objectiveValue != nil {
			objStr = fmt.Sprintf("%.0f", *r.objectiveValue)
		}
		fmt.Printf("  %-*s  %s %-5s  %15s\n", nameWidth, r.name, icon, string(r.status), objStr)
	}
	fmt.Println()
	fmt.Println("Done.")
}

// extractAssignments returns a sorted map of variable name -> float64 value.
func extractAssignments(assignments map[string]nxuskit.SolverValue) map[string]float64 {
	m := make(map[string]float64, len(assignments))
	for k, v := range assignments {
		switch val := v.Value.(type) {
		case float64:
			m[k] = val
		case bool:
			if val {
				m[k] = 1
			} else {
				m[k] = 0
			}
		}
	}
	return m
}

// printDelta shows differences between base and what-if assignments.
func printDelta(base, whatIf map[string]float64, indent string) {
	keys := make([]string, 0, len(base))
	for k := range base {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	anyDelta := false
	for _, k := range keys {
		baseVal := base[k]
		wiVal, ok := whatIf[k]
		if !ok {
			continue
		}
		diff := wiVal - baseVal
		if math.Abs(diff) > 0.001 {
			sign := ""
			if diff > 0 {
				sign = "+"
			}
			fmt.Printf("%s%s: %.0f -> %.0f (%s%.0f)\n", indent, k, baseVal, wiVal, sign, diff)
			anyDelta = true
		}
	}
	if !anyDelta {
		fmt.Printf("%s(no changes from base)\n", indent)
	}
}

// printAssignments displays variable assignments in sorted order.
func printAssignments(assignments map[string]nxuskit.SolverValue) {
	printAssignmentsIndented(assignments, "  ")
}

// printAssignmentsIndented displays variable assignments with a custom indent.
func printAssignmentsIndented(assignments map[string]nxuskit.SolverValue, indent string) {
	if len(assignments) == 0 {
		return
	}
	fmt.Println("Assignments:")

	keys := make([]string, 0, len(assignments))
	for k := range assignments {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := assignments[k]
		fmt.Printf("%s%-25s = %v\n", indent, k, formatValue(v))
	}
}

// formatValue renders a SolverValue for display.
func formatValue(v nxuskit.SolverValue) string {
	switch val := v.Value.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if v.Type == "integer" || val == float64(int64(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%.4f", val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v.Value)
	}
}

// printStats prints solver performance stats when verbose mode is enabled.
func printStats(config *interactive.Config, stats nxuskit.SolverStats) {
	if !config.IsVerbose() {
		return
	}
	fmt.Printf("[nxusKit] Solve stats: %dms, %d vars, %d constraints",
		stats.SolveTimeMs, stats.NumVariables, stats.NumConstraints)
	if stats.NumConflicts != nil {
		fmt.Printf(", %d conflicts", *stats.NumConflicts)
	}
	if stats.NumDecisions != nil {
		fmt.Printf(", %d decisions", *stats.NumDecisions)
	}
	fmt.Println()
}

// statusIcon returns a text indicator for the solve status.
func statusIcon(status nxuskit.SolveStatus) string {
	switch status {
	case nxuskit.SolveStatusSat, nxuskit.SolveStatusOptimal:
		return "[OK]"
	case nxuskit.SolveStatusUnsat:
		return "[!!]"
	case nxuskit.SolveStatusTimeout:
		return "[TO]"
	default:
		return "[??]"
	}
}

// ensureConstraintParams ensures the Parameters field is non-nil for every
// constraint, as the C ABI requires the JSON field to be present.
func ensureConstraintParams(constraints []nxuskit.ConstraintDef) {
	for i := range constraints {
		if constraints[i].Parameters == nil {
			constraints[i].Parameters = map[string]any{}
		}
	}
}

// flagValue extracts the value for a flag like "--scenario wedding" from
// os.Args. Returns "" if not found.
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

// listAvailableScenarios prints the available scenario directories.
func listAvailableScenarios() {
	scenariosDir := filepath.Join("..", "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Available scenarios: wedding, mars, recipe")
		return
	}
	fmt.Fprintln(os.Stderr, "Available scenarios:")
	for _, e := range entries {
		if e.IsDir() {
			pPath := filepath.Join(scenariosDir, e.Name(), "problem.json")
			if _, err := os.Stat(pPath); err == nil {
				fmt.Fprintf(os.Stderr, "  - %s\n", e.Name())
			}
		}
	}
}
