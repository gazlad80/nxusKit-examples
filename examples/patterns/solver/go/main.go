//go:build nxuskit

// Example: Solver — Satisfaction, Optimization & What-If Analysis
//
// ## nxusKit Features Demonstrated
// - SolverSession lifecycle (create, close)
// - Variable and constraint model building
// - Satisfaction solving (feasibility check)
// - Single-objective optimization
// - Multi-objective weighted optimization
// - Soft constraints with weights
// - Push/Pop scoping for what-if analysis
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show solver stats and intermediate details
// - `--step` or `-s`: Pause at each phase with explanations
//
// ## Scenario Selection
// - `--scenario <name>`: Load a problem from ../scenarios/<name>/problem.json
// - Available scenarios: theme-park, space-colony, fantasy-draft
//
// Usage:
//
//	go run . --scenario theme-park
//	go run . --scenario space-colony --verbose
//	go run . --scenario fantasy-draft --step
//
// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"encoding/json"
	"fmt"
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
	SoftConstraints []nxuskit.ConstraintDef `json:"soft_constraints"`
	Objectives      []nxuskit.ObjectiveDef  `json:"objectives"`
	WhatIfScenarios []WhatIfScenario        `json:"what_if_scenarios"`
}

// WhatIfScenario describes a push/pop what-if analysis scenario.
type WhatIfScenario struct {
	Name                  string                  `json:"name"`
	Description           string                  `json:"description"`
	AdditionalConstraints []nxuskit.ConstraintDef `json:"additional_constraints"`
}

// stepResult holds the outcome of a solver step for the final summary.
type stepResult struct {
	name   string
	status nxuskit.SolveStatus
	detail string
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
		fmt.Fprintln(os.Stderr, "Error: --scenario <name> is required")
		fmt.Fprintln(os.Stderr)
		listAvailableScenarios()
		os.Exit(1)
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
	ensureConstraintParams(problem.SoftConstraints)
	for i := range problem.WhatIfScenarios {
		ensureConstraintParams(problem.WhatIfScenarios[i].AdditionalConstraints)
	}

	// ── Print problem summary ────────────────────────────────────
	fmt.Println("========================================")
	fmt.Printf("  Constraint Solver: %s\n", problem.Name)
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println(problem.Description)
	fmt.Println()
	fmt.Printf("Variables:        %d\n", len(problem.Variables))
	fmt.Printf("Hard constraints: %d\n", len(problem.Constraints))
	fmt.Printf("Soft constraints: %d\n", len(problem.SoftConstraints))
	fmt.Printf("Objectives:       %d\n", len(problem.Objectives))
	fmt.Printf("What-if scenarios: %d\n", len(problem.WhatIfScenarios))
	fmt.Println()

	// Step: introduction
	if config.StepPause("Problem loaded. Starting solver session...", []string{
		"nxusKit: SolverSession wraps the Z3 constraint solver via C ABI",
		"Variables, constraints, and objectives are added incrementally",
		"Push/Pop enables reversible what-if exploration",
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

	var results []stepResult

	// ── Step 1: Satisfaction ─────────────────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  Step 1: Satisfaction (feasibility)")
	fmt.Println("----------------------------------------")
	fmt.Println()

	if config.StepPause("Adding hard constraints and checking feasibility...", []string{
		"nxusKit: AddConstraints() adds hard constraints to the model",
		"Solve() with no objective checks satisfiability only",
		"Status 'sat' means a feasible assignment exists",
	}) == interactive.ActionQuit {
		return
	}

	if err := session.AddConstraints(problem.Constraints); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding constraints: %v\n", err)
		os.Exit(1)
	}

	satResult, err := session.Solve(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error solving: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", satResult.Status)
	if satResult.Status == nxuskit.SolveStatusSat || satResult.Status == nxuskit.SolveStatusOptimal {
		printAssignments(satResult.Assignments)
	}
	printStats(config, satResult.Stats)
	fmt.Println()

	results = append(results, stepResult{
		name:   "Satisfaction",
		status: satResult.Status,
		detail: fmt.Sprintf("%d assignments", len(satResult.Assignments)),
	})

	// ── Step 2: Single-objective optimization ────────────────────
	if len(problem.Objectives) > 0 {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 2: Optimization (single objective)")
		fmt.Println("----------------------------------------")
		fmt.Println()

		obj := problem.Objectives[0]
		fmt.Printf("Objective: %s %s\n", obj.Direction, obj.Name)
		if obj.Label != "" {
			fmt.Printf("  %s\n", obj.Label)
		}
		fmt.Println()

		if config.StepPause("Setting objective and solving for optimality...", []string{
			"nxusKit: SetObjective() sets a single optimization target",
			"Solver searches for the optimal value (minimize or maximize)",
			"Status 'optimal' means the best possible value was found",
		}) == interactive.ActionQuit {
			return
		}

		if err := session.SetObjective(obj); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting objective: %v\n", err)
			os.Exit(1)
		}

		optResult, err := session.Solve(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error solving: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Status: %s\n", optResult.Status)
		if optResult.ObjectiveValue != nil {
			fmt.Printf("Objective value: %.2f\n", *optResult.ObjectiveValue)
		}
		if optResult.Status == nxuskit.SolveStatusSat || optResult.Status == nxuskit.SolveStatusOptimal {
			printAssignments(optResult.Assignments)
		}
		printStats(config, optResult.Stats)
		fmt.Println()

		detail := fmt.Sprintf("%d assignments", len(optResult.Assignments))
		if optResult.ObjectiveValue != nil {
			detail = fmt.Sprintf("obj=%.2f, %s", *optResult.ObjectiveValue, detail)
		}
		results = append(results, stepResult{
			name:   "Optimization",
			status: optResult.Status,
			detail: detail,
		})
	}

	// ── Step 3: Multi-objective optimization ─────────────────────
	if len(problem.Objectives) > 1 {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 3: Multi-objective (weighted)")
		fmt.Println("----------------------------------------")
		fmt.Println()

		for _, obj := range problem.Objectives {
			w := 1.0
			if obj.Weight != nil {
				w = *obj.Weight
			}
			fmt.Printf("  - %s %s (weight=%.1f)\n", obj.Direction, obj.Name, w)
		}
		fmt.Println()

		if config.StepPause("Adding multiple objectives with weighted combination...", []string{
			"nxusKit: AddObjective() appends to the multi-objective list",
			"MultiObjectiveMode 'weighted' combines objectives by weight",
			"Each objective's weight determines its relative importance",
		}) == interactive.ActionQuit {
			return
		}

		for _, obj := range problem.Objectives {
			if err := session.AddObjective(obj); err != nil {
				fmt.Fprintf(os.Stderr, "Error adding objective: %v\n", err)
				os.Exit(1)
			}
		}

		mode := nxuskit.MultiObjectiveModeWeighted
		multiConfig := &nxuskit.SolverConfig{
			MultiObjectiveMode: &mode,
		}
		multiResult, err := session.Solve(multiConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error solving: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Status: %s\n", multiResult.Status)
		if multiResult.ObjectiveValue != nil {
			fmt.Printf("Combined objective value: %.2f\n", *multiResult.ObjectiveValue)
		}
		if len(multiResult.ObjectiveValues) > 0 {
			fmt.Println("Per-objective values:")
			for name, val := range multiResult.ObjectiveValues {
				fmt.Printf("  - %s: %.2f\n", name, val)
			}
		}
		if multiResult.Status == nxuskit.SolveStatusSat || multiResult.Status == nxuskit.SolveStatusOptimal {
			printAssignments(multiResult.Assignments)
		}
		printStats(config, multiResult.Stats)
		fmt.Println()

		detail := fmt.Sprintf("%d objectives, %d assignments", len(problem.Objectives), len(multiResult.Assignments))
		if multiResult.ObjectiveValue != nil {
			detail = fmt.Sprintf("combined=%.2f, %s", *multiResult.ObjectiveValue, detail)
		}
		results = append(results, stepResult{
			name:   "Multi-objective",
			status: multiResult.Status,
			detail: detail,
		})
	} else {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 3: Multi-objective (skipped)")
		fmt.Println("----------------------------------------")
		fmt.Println()
		fmt.Println("Only one objective defined; skipping multi-objective step.")
		fmt.Println()
	}

	// ── Step 4: Soft constraints ─────────────────────────────────
	if len(problem.SoftConstraints) > 0 {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 4: Soft constraints")
		fmt.Println("----------------------------------------")
		fmt.Println()

		for _, sc := range problem.SoftConstraints {
			w := 0.0
			if sc.Weight != nil {
				w = *sc.Weight
			}
			label := sc.Label
			if label == "" {
				label = sc.Name
			}
			fmt.Printf("  - %s (weight=%.1f)\n", label, w)
		}
		fmt.Println()

		if config.StepPause("Adding soft constraints with weights...", []string{
			"nxusKit: Soft constraints have a Weight field",
			"The solver satisfies them when possible, violates when necessary",
			"ViolatedSoftConstraints in the result shows which were relaxed",
		}) == interactive.ActionQuit {
			return
		}

		if err := session.AddConstraints(problem.SoftConstraints); err != nil {
			fmt.Fprintf(os.Stderr, "Error adding soft constraints: %v\n", err)
			os.Exit(1)
		}

		softResult, err := session.Solve(nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error solving: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Status: %s\n", softResult.Status)
		if softResult.ObjectiveValue != nil {
			fmt.Printf("Objective value: %.2f\n", *softResult.ObjectiveValue)
		}
		if len(softResult.ViolatedSoftConstraints) > 0 {
			fmt.Printf("Violated soft constraints: %s\n", strings.Join(softResult.ViolatedSoftConstraints, ", "))
		} else {
			fmt.Println("All soft constraints satisfied.")
		}
		if softResult.Status == nxuskit.SolveStatusSat || softResult.Status == nxuskit.SolveStatusOptimal {
			printAssignments(softResult.Assignments)
		}
		printStats(config, softResult.Stats)
		fmt.Println()

		violated := "none violated"
		if len(softResult.ViolatedSoftConstraints) > 0 {
			violated = fmt.Sprintf("%d violated", len(softResult.ViolatedSoftConstraints))
		}
		results = append(results, stepResult{
			name:   "Soft constraints",
			status: softResult.Status,
			detail: fmt.Sprintf("%d soft constraints, %s", len(problem.SoftConstraints), violated),
		})
	} else {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 4: Soft constraints (skipped)")
		fmt.Println("----------------------------------------")
		fmt.Println()
		fmt.Println("No soft constraints defined; skipping.")
		fmt.Println()
	}

	// ── Step 5: What-if analysis (push/pop) ──────────────────────
	if len(problem.WhatIfScenarios) > 0 {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 5: What-if analysis (push/pop)")
		fmt.Println("----------------------------------------")
		fmt.Println()

		for i, ws := range problem.WhatIfScenarios {
			fmt.Printf("Scenario %d: %s\n", i+1, ws.Name)
			fmt.Printf("  %s\n", ws.Description)
			fmt.Println()

			if config.StepPause(fmt.Sprintf("Push scope, add %d constraints, solve, then pop...", len(ws.AdditionalConstraints)), []string{
				"nxusKit: Push() saves the current model state",
				"Additional constraints are added temporarily",
				"Pop() restores the model to its pre-push state",
				"This enables side-by-side comparison without rebuilding",
			}) == interactive.ActionQuit {
				return
			}

			// Push scope
			if err := session.Push(); err != nil {
				fmt.Fprintf(os.Stderr, "Error pushing scope: %v\n", err)
				os.Exit(1)
			}

			// Add what-if constraints
			if err := session.AddConstraints(ws.AdditionalConstraints); err != nil {
				fmt.Fprintf(os.Stderr, "Error adding what-if constraints: %v\n", err)
				os.Exit(1)
			}

			if config.IsVerbose() {
				fmt.Println("[nxusKit] What-if constraints added:")
				for _, c := range ws.AdditionalConstraints {
					label := c.Label
					if label == "" {
						label = c.Name
					}
					fmt.Printf("  - %s (%s)\n", label, c.ConstraintType)
				}
				fmt.Println()
			}

			// Solve under what-if constraints
			wiResult, err := session.Solve(nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error solving what-if: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("Status: %s\n", wiResult.Status)
			if wiResult.ObjectiveValue != nil {
				fmt.Printf("Objective value: %.2f\n", *wiResult.ObjectiveValue)
			}
			if wiResult.Status == nxuskit.SolveStatusSat || wiResult.Status == nxuskit.SolveStatusOptimal {
				printAssignments(wiResult.Assignments)
			}
			printStats(config, wiResult.Stats)
			fmt.Println()

			// Pop scope to restore original model
			if err := session.Pop(); err != nil {
				fmt.Fprintf(os.Stderr, "Error popping scope: %v\n", err)
				os.Exit(1)
			}

			detail := fmt.Sprintf("%d extra constraints, %d assignments", len(ws.AdditionalConstraints), len(wiResult.Assignments))
			if wiResult.ObjectiveValue != nil {
				detail = fmt.Sprintf("obj=%.2f, %s", *wiResult.ObjectiveValue, detail)
			}
			results = append(results, stepResult{
				name:   fmt.Sprintf("What-if: %s", ws.Name),
				status: wiResult.Status,
				detail: detail,
			})
		}
	} else {
		fmt.Println("----------------------------------------")
		fmt.Println("  Step 5: What-if analysis (skipped)")
		fmt.Println("----------------------------------------")
		fmt.Println()
		fmt.Println("No what-if scenarios defined; skipping.")
		fmt.Println()
	}

	// ── Summary ──────────────────────────────────────────────────
	fmt.Println("========================================")
	fmt.Println("  Summary")
	fmt.Println("========================================")
	fmt.Println()
	for _, r := range results {
		icon := statusIcon(r.status)
		fmt.Printf("  %s %-25s [%s] %s\n", icon, r.name, r.status, r.detail)
	}
	fmt.Println()
	fmt.Println("Done.")
}

// printAssignments displays variable assignments in sorted order.
func printAssignments(assignments map[string]nxuskit.SolverValue) {
	if len(assignments) == 0 {
		return
	}
	fmt.Println("Assignments:")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(assignments))
	for k := range assignments {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := assignments[k]
		fmt.Printf("  %-25s = %v\n", k, formatValue(v))
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
		// Display integers without decimal points
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

// flagValue extracts the value for a flag like "--scenario theme-park" from
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
		fmt.Fprintln(os.Stderr, "Available scenarios: theme-park, space-colony, fantasy-draft")
		return
	}
	fmt.Fprintln(os.Stderr, "Available scenarios:")
	for _, e := range entries {
		if e.IsDir() {
			// Verify problem.json exists
			pPath := filepath.Join(scenariosDir, e.Name(), "problem.json")
			if _, err := os.Stat(pPath); err == nil {
				fmt.Fprintf(os.Stderr, "  - %s\n", e.Name())
			}
		}
	}
}
