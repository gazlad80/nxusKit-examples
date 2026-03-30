//go:build nxuskit

// Example: Multi-Provider Pipeline — BN Prediction + Solver Optimization + CLIPS Safety
//
// ## nxusKit Features Demonstrated
// - BnNetwork lifecycle (load BIF, set evidence, infer)
// - Variable Elimination exact inference for crowd/demand prediction
// - SolverSession lifecycle (create, add variables/constraints/objective, solve)
// - Single-objective optimization using BN-predicted values
// - ClipsProvider lifecycle (create with rules dir, assert facts via Chat, collect alerts)
// - 3-stage pipeline: prediction -> optimization -> safety enforcement
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show intermediate results and raw data
// - `--step` or `-s`: Pause at each pipeline stage with explanations
//
// ## Scenario Selection
// - `--scenario <name>`: Load a scenario from ../scenarios/<name>/
// - Available scenarios: festival, rescue, bakery
//
// Usage:
//
//	go run . --scenario festival
//	go run . --scenario rescue --verbose
//	go run . --scenario bakery --step
//
// See the solver and bayesian-inference Go examples for the
// individual pattern references.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// scenarioConfig maps scenario names to their CLIPS template/alert names.
type scenarioConfig struct {
	factTemplate  string // e.g. "stage-assignment"
	alertTemplate string // e.g. "safety-alert"
}

// knownScenarios defines the CLIPS template mapping per scenario.
var knownScenarios = map[string]scenarioConfig{
	"festival": {factTemplate: "stage-assignment", alertTemplate: "safety-alert"},
	"rescue":   {factTemplate: "rescue-assignment", alertTemplate: "protocol-alert"},
	"bakery":   {factTemplate: "baking-assignment", alertTemplate: "health-alert"},
}

// Problem represents the JSON structure of a scenario problem file.
type Problem struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Variables   []nxuskit.VariableDef   `json:"variables"`
	Constraints []nxuskit.ConstraintDef `json:"constraints"`
	Objectives  []nxuskit.ObjectiveDef  `json:"objectives"`
}

// stageResult holds the outcome of a pipeline stage for the final summary.
type stageResult struct {
	name   string
	status string
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

	// Validate scenario against known configurations
	scConfig, ok := knownScenarios[scenario]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown scenario %q\n", scenario)
		fmt.Fprintln(os.Stderr)
		listAvailableScenarios()
		os.Exit(1)
	}

	// Resolve scenario paths relative to the binary location
	scenarioDir := filepath.Join("..", "scenarios", scenario)
	modelPath := filepath.Join(scenarioDir, "model.bif")
	evidencePath := filepath.Join(scenarioDir, "evidence.json")
	problemPath := filepath.Join(scenarioDir, "problem.json")
	rulesDir := scenarioDir // rules.clp is in the scenario directory

	// Verify all required files exist
	requiredFiles := map[string]string{
		"model.bif":     modelPath,
		"evidence.json": evidencePath,
		"problem.json":  problemPath,
		"rules.clp":     filepath.Join(scenarioDir, "rules.clp"),
	}
	for name, path := range requiredFiles {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: cannot load scenario %q: %s not found\n", scenario, name)
			fmt.Fprintln(os.Stderr)
			listAvailableScenarios()
			os.Exit(1)
		}
	}

	// Load evidence
	evidenceMap, err := loadEvidence(evidencePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot load evidence: %v\n", err)
		os.Exit(1)
	}

	// Load problem
	problemData, err := os.ReadFile(problemPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot load problem: %v\n", err)
		os.Exit(1)
	}
	var problem Problem
	if err := json.Unmarshal(problemData, &problem); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid problem.json: %v\n", err)
		os.Exit(1)
	}

	// Ensure constraint parameters are never nil (C ABI requires the field)
	ensureConstraintParams(problem.Constraints)

	var results []stageResult

	// ── Pipeline Header ─────────────────────────────────────────
	fmt.Println("========================================")
	fmt.Printf("  Multi-Provider Pipeline: %s\n", scenario)
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println(problem.Description)
	fmt.Println()
	fmt.Println("Pipeline stages:")
	fmt.Println("  1. BN Prediction    - Probabilistic inference for demand/crowd estimation")
	fmt.Println("  2. Solver Optimize  - Constraint optimization using BN predictions")
	fmt.Println("  3. CLIPS Safety     - Rule-based safety enforcement on solver output")
	fmt.Println()

	if config.StepPause("Starting 3-stage pipeline...", []string{
		"Stage 1 uses Bayesian Network inference to predict demand/crowd levels",
		"Stage 2 feeds predictions into a Z3 constraint solver for optimal assignments",
		"Stage 3 validates assignments against domain-specific safety rules via CLIPS",
	}) == interactive.ActionQuit {
		return
	}

	// ══════════════════════════════════════════════════════════════
	// Stage 1: BN Prediction
	// ══════════════════════════════════════════════════════════════
	fmt.Println("========================================")
	fmt.Println("  Stage 1: BN Prediction")
	fmt.Println("========================================")
	fmt.Println()

	if config.StepPause("Loading Bayesian Network and running inference...", []string{
		"nxusKit: LoadBnNetwork() parses a BIF file into a network handle",
		"Evidence is set from the scenario's evidence.json",
		"Variable Elimination computes exact posterior distributions",
	}) == interactive.ActionQuit {
		return
	}

	// Load network
	net, err := nxuskit.LoadBnNetwork(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading BN network: %v\n", err)
		os.Exit(1)
	}
	defer net.Close()

	// Get network variables
	variables, err := net.Variables()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading network variables: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Network: %s\n", modelPath)
	fmt.Printf("Variables: %d\n", len(variables))
	fmt.Println()

	if config.IsVerbose() {
		fmt.Println("[nxusKit] Network variables:")
		sorted := make([]string, len(variables))
		copy(sorted, variables)
		sort.Strings(sorted)
		for _, v := range sorted {
			states, err := net.VariableStates(v)
			if err != nil {
				fmt.Printf("  - %s (error: %v)\n", v, err)
				continue
			}
			fmt.Printf("  - %s: [%s]\n", v, strings.Join(states, ", "))
		}
		fmt.Println()
	}

	// Set evidence
	fmt.Println("Evidence:")
	evidenceKeys := sortedKeys(evidenceMap)
	for _, k := range evidenceKeys {
		fmt.Printf("  %s = %s\n", k, evidenceMap[k])
	}
	fmt.Println()

	ev, err := nxuskit.NewBnEvidence()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating evidence: %v\n", err)
		os.Exit(1)
	}
	defer ev.Close()

	for _, k := range evidenceKeys {
		if err := ev.SetDiscrete(net, k, evidenceMap[k]); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting evidence %s=%s: %v\n", k, evidenceMap[k], err)
			os.Exit(1)
		}
	}

	// Run Variable Elimination
	veResult, err := net.Infer(ev, "ve")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error (VE inference): %v\n", err)
		os.Exit(1)
	}

	// Extract prediction variable — find the first non-evidence variable
	predictionVar := findPredictionVariable(variables, evidenceMap)
	if predictionVar == "" {
		fmt.Fprintln(os.Stderr, "Error: no prediction variable found (all variables are evidence)")
		os.Exit(1)
	}

	predDist, err := veResult.Marginal(predictionVar)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading marginal for %s: %v\n", predictionVar, err)
		os.Exit(1)
	}
	veResult.Close()

	// Display prediction
	fmt.Printf("Prediction variable: %s\n", predictionVar)
	fmt.Println("Posterior distribution (VE):")
	predStates := sortedKeys(predDist)
	maxProb := 0.0
	maxState := ""
	for _, s := range predStates {
		prob := predDist[s]
		bar := strings.Repeat("#", int(prob*40))
		fmt.Printf("  %-12s %.4f  %s\n", s, prob, bar)
		if prob > maxProb {
			maxProb = prob
			maxState = s
		}
	}
	fmt.Printf("\nMost likely: %s (%.1f%%)\n", maxState, maxProb*100)
	fmt.Println()

	results = append(results, stageResult{
		name:   "BN Prediction",
		status: "OK",
		detail: fmt.Sprintf("%s -> %s (%.1f%%)", predictionVar, maxState, maxProb*100),
	})

	// ══════════════════════════════════════════════════════════════
	// Stage 2: Solver Optimization
	// ══════════════════════════════════════════════════════════════
	fmt.Println("========================================")
	fmt.Println("  Stage 2: Solver Optimization")
	fmt.Println("========================================")
	fmt.Println()

	if config.StepPause("Creating solver session and optimizing assignments...", []string{
		"nxusKit: NewSolverSession() creates a Z3 constraint solver session",
		"Variables and constraints are loaded from the scenario's problem.json",
		"The solver finds optimal assignments respecting all constraints",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Printf("Problem: %s\n", problem.Name)
	fmt.Printf("Variables: %d\n", len(problem.Variables))
	fmt.Printf("Constraints: %d\n", len(problem.Constraints))
	fmt.Printf("Objectives: %d\n", len(problem.Objectives))
	fmt.Println()

	if config.IsVerbose() {
		fmt.Println("[nxusKit] Problem variables:")
		for _, v := range problem.Variables {
			label := v.Label
			if label == "" {
				label = string(v.VarType)
			}
			fmt.Printf("  - %s (%s): %s\n", v.Name, v.VarType, label)
		}
		fmt.Println()
	}

	// Create solver session
	// TODO(v0.8.1): Migrate to FFI solver when available
	session, err := nxuskit.NewSolverSession(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating solver session: %v\n", err)
		os.Exit(1)
	}
	defer session.Close()

	// Add variables
	if err := session.AddVariables(problem.Variables); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding variables: %v\n", err)
		os.Exit(1)
	}

	// Add constraints
	if err := session.AddConstraints(problem.Constraints); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding constraints: %v\n", err)
		os.Exit(1)
	}

	// Set objective (use first objective)
	if len(problem.Objectives) > 0 {
		obj := problem.Objectives[0]
		fmt.Printf("Objective: %s %s\n", obj.Direction, obj.Name)
		if obj.Label != "" {
			fmt.Printf("  %s\n", obj.Label)
		}
		fmt.Println()

		if err := session.SetObjective(obj); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting objective: %v\n", err)
			os.Exit(1)
		}
	}

	// Solve
	solveResult, err := session.Solve(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error solving: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", solveResult.Status)
	if solveResult.ObjectiveValue != nil {
		fmt.Printf("Objective value: %.2f\n", *solveResult.ObjectiveValue)
	}

	solverOK := solveResult.Status == nxuskit.SolveStatusSat || solveResult.Status == nxuskit.SolveStatusOptimal
	if solverOK {
		printAssignments(solveResult.Assignments)
	}

	if config.IsVerbose() {
		fmt.Printf("[nxusKit] Solve stats: %dms, %d vars, %d constraints\n",
			solveResult.Stats.SolveTimeMs, solveResult.Stats.NumVariables, solveResult.Stats.NumConstraints)
	}
	fmt.Println()

	solverDetail := fmt.Sprintf("%s, %d assignments", solveResult.Status, len(solveResult.Assignments))
	if solveResult.ObjectiveValue != nil {
		solverDetail = fmt.Sprintf("%s, obj=%.2f, %d assignments", solveResult.Status, *solveResult.ObjectiveValue, len(solveResult.Assignments))
	}
	results = append(results, stageResult{
		name:   "Solver Optimization",
		status: statusIcon(solveResult.Status),
		detail: solverDetail,
	})

	// ══════════════════════════════════════════════════════════════
	// Stage 3: CLIPS Safety Enforcement
	// ══════════════════════════════════════════════════════════════
	fmt.Println("========================================")
	fmt.Println("  Stage 3: CLIPS Safety Enforcement")
	fmt.Println("========================================")
	fmt.Println()

	if !solverOK {
		fmt.Println("Skipping CLIPS stage: solver did not find a feasible solution.")
		fmt.Println()
		results = append(results, stageResult{
			name:   "CLIPS Safety",
			status: "[--]",
			detail: "skipped (no feasible solution from solver)",
		})
	} else {
		if config.StepPause("Loading safety rules and validating solver assignments...", []string{
			"nxusKit: NewClipsProvider() creates a CLIPS rule engine environment",
			"Facts are asserted based on solver output via the Chat API",
			"Rules fire to detect safety violations and generate alerts",
		}) == interactive.ActionQuit {
			return
		}

		// Create CLIPS provider (FFI-backed)
		provider, err := nxuskit.NewClipsFFIProvider(rulesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating CLIPS provider: %v\n", err)
			os.Exit(1)
		}

		// Build facts from solver assignments
		facts := buildClipsFacts(scConfig, solveResult.Assignments)
		fmt.Printf("Asserting %d facts as %s templates...\n", len(facts), scConfig.factTemplate)

		if config.IsVerbose() {
			fmt.Println("[nxusKit] Facts to assert:")
			for _, f := range facts {
				factJSON, _ := json.MarshalIndent(f, "  ", "  ")
				fmt.Printf("  %s\n", factJSON)
			}
			fmt.Println()
		}

		// Send facts via Chat API
		derivedOnly := true
		verbose := config.IsVerbose()
		input := clipsInputWire{
			Facts: facts,
			Config: &clipsRequestConfigWire{
				DerivedOnlyNew: &derivedOnly,
				IncludeTrace:   &verbose,
			},
		}
		inputJSON, err := json.Marshal(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling CLIPS input: %v\n", err)
			os.Exit(1)
		}

		req := &nxuskit.ChatRequest{
			Messages: []nxuskit.Message{nxuskit.UserMessage(string(inputJSON))},
		}
		resp, err := provider.Chat(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running CLIPS rules: %v\n", err)
			os.Exit(1)
		}

		// Parse CLIPS output
		var output clipsOutputWire
		if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing CLIPS output: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Rules fired: %d\n", output.Stats.TotalRulesFired)
		fmt.Printf("Alerts generated: %d\n", output.Stats.ConclusionsCount)
		fmt.Println()

		// Display alerts grouped by severity
		if len(output.Conclusions) == 0 {
			fmt.Println("No safety alerts generated. All assignments pass safety checks.")
		} else {
			fmt.Println("Safety Alerts:")
			printAlerts(output.Conclusions, scConfig.alertTemplate)
		}

		if config.IsVerbose() && output.Trace != nil {
			fmt.Println()
			fmt.Println("[nxusKit] Rule trace:")
			for _, firing := range output.Trace.RulesFired {
				fmt.Printf("  - %s\n", firing.RuleName)
			}
		}
		fmt.Println()

		// Count alerts by severity
		criticalCount, warningCount, infoCount := countAlertsBySeverity(output.Conclusions)
		alertDetail := fmt.Sprintf("%d alerts (%d critical, %d warning, %d info)",
			len(output.Conclusions), criticalCount, warningCount, infoCount)

		clipsStatus := "[OK]"
		if criticalCount > 0 {
			clipsStatus = "[!!]"
		} else if warningCount > 0 {
			clipsStatus = "[WN]"
		}

		results = append(results, stageResult{
			name:   "CLIPS Safety",
			status: clipsStatus,
			detail: alertDetail,
		})
	}

	// ── Pipeline Summary ─────────────────────────────────────────
	fmt.Println("========================================")
	fmt.Println("  Pipeline Summary")
	fmt.Println("========================================")
	fmt.Println()
	for i, r := range results {
		fmt.Printf("  %d. %-5s %-25s %s\n", i+1, r.status, r.name, r.detail)
	}
	fmt.Println()
	fmt.Println("Done.")
}

// ── Helper Functions ──────────────────────────────────────────────

// loadEvidence reads evidence.json as a map[string]string.
func loadEvidence(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid evidence JSON: %w", err)
	}
	return m, nil
}

// findPredictionVariable returns the first non-evidence variable in the network.
// For the pipeline, this is the variable whose distribution we want to predict.
func findPredictionVariable(variables []string, evidence map[string]string) string {
	sorted := make([]string, len(variables))
	copy(sorted, variables)
	sort.Strings(sorted)
	for _, v := range sorted {
		if _, isEvidence := evidence[v]; !isEvidence {
			return v
		}
	}
	return ""
}

// buildClipsFacts converts solver assignments into CLIPS facts for the
// appropriate scenario template. Each scenario has a different fact structure.
func buildClipsFacts(sc scenarioConfig, assignments map[string]nxuskit.SolverValue) []clipsFactWire {
	switch sc.factTemplate {
	case "stage-assignment":
		return buildFestivalFacts(assignments)
	case "rescue-assignment":
		return buildRescueFacts(assignments)
	case "baking-assignment":
		return buildBakeryFacts(assignments)
	default:
		return nil
	}
}

// buildFestivalFacts creates stage-assignment facts from solver output.
// Extracts band-to-stage assignments and predicted crowd sizes.
func buildFestivalFacts(assignments map[string]nxuskit.SolverValue) []clipsFactWire {
	// Collect band stage assignments
	type bandInfo struct {
		stage int
		name  string
	}
	bands := make(map[int]bandInfo)

	for k, v := range assignments {
		if strings.HasPrefix(k, "band_") && strings.HasSuffix(k, "_stage") {
			idx := extractIndex(k, "band_", "_stage")
			if idx > 0 {
				stage := toInt(v)
				bands[idx] = bandInfo{stage: stage, name: fmt.Sprintf("Band_%d", idx)}
			}
		}
	}

	// Collect stage crowd predictions
	stageCrowds := make(map[int]int)
	for k, v := range assignments {
		if strings.HasPrefix(k, "stage_") && strings.HasSuffix(k, "_crowd") {
			idx := extractIndex(k, "stage_", "_crowd")
			if idx > 0 {
				stageCrowds[idx] = toInt(v)
			}
		}
	}

	// Build facts — one per band assignment
	var facts []clipsFactWire
	bandIDs := sortedIntKeys(bands)
	for _, bID := range bandIDs {
		b := bands[bID]
		crowd := stageCrowds[b.stage]

		// Alternate pyro and stage material for variety
		hasPyro := "no"
		if bID%3 == 0 {
			hasPyro = "yes"
		}
		stageMaterial := "concrete"
		switch b.stage {
		case 1:
			stageMaterial = "wood"
		case 2:
			stageMaterial = "metal"
		case 3:
			stageMaterial = "concrete"
		}

		facts = append(facts, clipsFactWire{
			Template: "stage-assignment",
			Values: map[string]interface{}{
				"stage-id":        b.stage,
				"band-name":       b.name,
				"predicted-crowd": crowd,
				"has-pyro":        hasPyro,
				"stage-material":  stageMaterial,
			},
		})
	}

	return facts
}

// buildRescueFacts creates rescue-assignment facts from solver output.
func buildRescueFacts(assignments map[string]nxuskit.SolverValue) []clipsFactWire {
	// Collect team zone assignments
	type teamInfo struct {
		zone int
	}
	teams := make(map[int]teamInfo)

	for k, v := range assignments {
		if strings.HasPrefix(k, "team_") && strings.HasSuffix(k, "_zone") {
			idx := extractIndex(k, "team_", "_zone")
			if idx > 0 {
				teams[idx] = teamInfo{zone: toInt(v)}
			}
		}
	}

	// Build facts — one per team assignment
	var facts []clipsFactWire
	teamIDs := sortedIntKeys(teams)
	for _, tID := range teamIDs {
		t := teams[tID]

		// Assign team types based on team ID
		teamType := "ground"
		switch tID {
		case 1, 2:
			teamType = "ground"
		case 3:
			teamType = "helicopter"
		case 4:
			teamType = "drone"
		}

		// Assign terrain and wind based on zone
		terrain := "urban"
		switch t.zone % 3 {
		case 0:
			terrain = "mountainous"
		case 1:
			terrain = "urban"
		case 2:
			terrain = "rural"
		}

		windSpeed := 20 + (t.zone * 8) // Simulate varying wind conditions

		facts = append(facts, clipsFactWire{
			Template: "rescue-assignment",
			Values: map[string]interface{}{
				"team-id":      tID,
				"zone-id":      t.zone,
				"team-type":    teamType,
				"wind-speed":   windSpeed,
				"zone-terrain": terrain,
			},
		})
	}

	return facts
}

// buildBakeryFacts creates baking-assignment facts from solver output.
func buildBakeryFacts(assignments map[string]nxuskit.SolverValue) []clipsFactWire {
	// Collect item assignments
	type itemInfo struct {
		oven int
		slot int
	}
	items := make(map[int]itemInfo)

	for k, v := range assignments {
		if strings.HasPrefix(k, "item_") && strings.HasSuffix(k, "_oven") {
			idx := extractIndex(k, "item_", "_oven")
			if idx > 0 {
				info := items[idx]
				info.oven = toInt(v)
				items[idx] = info
			}
		}
		if strings.HasPrefix(k, "item_") && strings.HasSuffix(k, "_slot") {
			idx := extractIndex(k, "item_", "_slot")
			if idx > 0 {
				info := items[idx]
				info.slot = toInt(v)
				items[idx] = info
			}
		}
	}

	// Item names and allergen properties
	itemNames := map[int]string{
		1: "Sourdough_Bread", 2: "Almond_Croissant", 3: "Gluten_Free_Muffin",
		4: "Chocolate_Cake", 5: "Walnut_Brownie", 6: "Plain_Bagel",
	}
	nutItems := map[int]bool{2: true, 5: true} // Almond Croissant, Walnut Brownie
	glutenFreeItems := map[int]bool{3: true}   // Gluten Free Muffin

	var facts []clipsFactWire
	itemIDs := sortedIntKeys(items)
	for _, iID := range itemIDs {
		item := items[iID]
		name := itemNames[iID]
		if name == "" {
			name = fmt.Sprintf("Item_%d", iID)
		}

		containsNuts := "no"
		if nutItems[iID] {
			containsNuts = "yes"
		}
		containsGluten := "yes"
		if glutenFreeItems[iID] {
			containsGluten = "no"
		}

		// Simulate last allergen in oven based on oven ID
		ovenLastAllergen := "none"
		switch item.oven {
		case 1:
			ovenLastAllergen = "gluten"
		case 2:
			ovenLastAllergen = "nuts"
		case 3:
			ovenLastAllergen = "none"
		}

		facts = append(facts, clipsFactWire{
			Template: "baking-assignment",
			Values: map[string]interface{}{
				"item-name":          name,
				"oven-id":            item.oven,
				"time-slot":          item.slot,
				"contains-nuts":      containsNuts,
				"contains-gluten":    containsGluten,
				"oven-last-allergen": ovenLastAllergen,
			},
		})
	}

	return facts
}

// printAssignments displays variable assignments in sorted order.
func printAssignments(assignments map[string]nxuskit.SolverValue) {
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
		fmt.Printf("  %-30s = %v\n", k, formatValue(v))
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

// printAlerts displays CLIPS conclusions grouped by severity.
func printAlerts(conclusions []clipsConclusionWire, alertTemplate string) {
	// Group by severity
	severityOrder := []string{"critical", "warning", "info"}
	grouped := make(map[string][]clipsConclusionWire)
	for _, c := range conclusions {
		if c.Template != alertTemplate {
			continue
		}
		sev := "info"
		if s, ok := c.Values["severity"].(string); ok {
			sev = s
		}
		grouped[sev] = append(grouped[sev], c)
	}

	// Also include any conclusions from other templates
	for _, c := range conclusions {
		if c.Template == alertTemplate {
			continue
		}
		grouped["info"] = append(grouped["info"], c)
	}

	for _, sev := range severityOrder {
		alerts := grouped[sev]
		if len(alerts) == 0 {
			continue
		}

		icon := "[i]"
		switch sev {
		case "critical":
			icon = "[!!]"
		case "warning":
			icon = "[WN]"
		}

		fmt.Printf("\n  %s %s (%d):\n", icon, strings.ToUpper(sev), len(alerts))
		for _, a := range alerts {
			msg := ""
			if m, ok := a.Values["message"].(string); ok {
				msg = m
			}
			ruleName := ""
			if r, ok := a.Values["rule-name"].(string); ok {
				ruleName = r
			}
			if ruleName != "" {
				fmt.Printf("      [%s] %s\n", ruleName, msg)
			} else {
				fmt.Printf("      %s\n", msg)
			}
		}
	}
}

// countAlertsBySeverity counts alerts by severity level.
func countAlertsBySeverity(conclusions []clipsConclusionWire) (critical, warning, info int) {
	for _, c := range conclusions {
		sev, _ := c.Values["severity"].(string)
		switch sev {
		case "critical":
			critical++
		case "warning":
			warning++
		default:
			info++
		}
	}
	return
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

// toInt converts a SolverValue to an integer.
func toInt(v nxuskit.SolverValue) int {
	switch val := v.Value.(type) {
	case float64:
		return int(val)
	case int64:
		return int(val)
	case int:
		return val
	default:
		return 0
	}
}

// extractIndex extracts the numeric index from a variable name like "band_3_stage".
func extractIndex(name, prefix, suffix string) int {
	s := strings.TrimPrefix(name, prefix)
	s = strings.TrimSuffix(s, suffix)
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return n
}

// sortedKeys returns the keys of a map[string]V in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedIntKeys returns the keys of a map[int]V in sorted order.
func sortedIntKeys[V any](m map[int]V) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// flagValue extracts the value for a flag like "--scenario festival" from
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
// A scenario is valid if it contains all four required files:
// model.bif, evidence.json, problem.json, and rules.clp.
func listAvailableScenarios() {
	scenariosDir := filepath.Join("..", "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Available scenarios: festival, rescue, bakery")
		return
	}
	fmt.Fprintln(os.Stderr, "Available scenarios:")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Verify all required files exist
		required := []string{"model.bif", "evidence.json", "problem.json", "rules.clp"}
		allPresent := true
		for _, f := range required {
			fPath := filepath.Join(scenariosDir, e.Name(), f)
			if _, err := os.Stat(fPath); err != nil {
				allPresent = false
				break
			}
		}
		if allPresent {
			fmt.Fprintf(os.Stderr, "  - %s\n", e.Name())
		}
	}
}
