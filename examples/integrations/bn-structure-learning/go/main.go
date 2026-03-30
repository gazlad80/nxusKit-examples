//go:build nxuskit

// Example: BN Structure Learning -- Causal Discovery from CSV Data
//
// ## nxusKit Features Demonstrated
// - BnNetwork lifecycle (create, search structure, learn parameters)
// - Hill-Climb + BDeu structure learning algorithm
// - K2 + BDeu structure learning algorithm
// - MLE parameter learning with Laplace smoothing
// - Log-likelihood model fit evaluation
// - Variable Elimination inference on learned models
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw JSON results and intermediate data
// - `--step` or `-s`: Pause at each pipeline step with explanations
//
// ## Scenario Selection
// - `--scenario <name>`: Load a scenario from ../scenarios/<name>/
// - Available scenarios: golf, bmx, sourdough
//
// Usage:
//
//	go run . --scenario golf
//	go run . --scenario bmx --verbose
//	go run . --scenario sourdough --step
package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// scenarioConfig holds scenario-specific settings for inference demo.
type scenarioConfig struct {
	title    string
	evidence map[string]string // sample evidence for inference
	queryVar string            // variable to query
}

var knownScenarios = map[string]scenarioConfig{
	"golf": {
		title:    "Golf Course Conditions",
		evidence: map[string]string{"weather": "rainy", "fertilizer": "heavy"},
		queryVar: "green_speed",
	},
	"bmx": {
		title:    "BMX Rider Performance",
		evidence: map[string]string{"skill": "pro", "pump_timing": "perfect"},
		queryVar: "lap_time",
	},
	"sourdough": {
		title:    "Sourdough Baking",
		evidence: map[string]string{"flour_type": "rye", "ambient_temp": "warm"},
		queryVar: "flavor_profile",
	},
}

func main() {
	config := interactive.FromArgs()

	scenario := flagValue("--scenario")
	if scenario == "" {
		scenario = flagValue("-scenario")
	}
	if scenario == "" {
		scenario = "golf"
	}

	scConfig, ok := knownScenarios[scenario]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: unknown scenario %q\n", scenario)
		fmt.Fprintln(os.Stderr)
		listAvailableScenarios()
		os.Exit(1)
	}

	scenarioDir := filepath.Join("..", "scenarios", scenario)
	csvPath := filepath.Join(scenarioDir, "data.csv")

	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: cannot find %s\n", csvPath)
		fmt.Fprintln(os.Stderr)
		listAvailableScenarios()
		os.Exit(1)
	}

	totalStart := time.Now()

	// ── Header ──────────────────────────────────────────────────
	fmt.Println("========================================")
	fmt.Printf("  BN Structure Learning: %s\n", scConfig.title)
	fmt.Println("========================================")
	fmt.Printf("Scenario: %s\n\n", scenario)

	// ════════════════════════════════════════════════════════════
	// Step 1: Load CSV Data
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 1: Load CSV Data ---")
	fmt.Println()

	if config.StepPause("Loading and inspecting CSV data...", []string{
		"Reads the CSV file to discover column names and row count",
		"Column names become Bayesian Network variable names",
		"Row count affects structure learning quality",
	}) == interactive.ActionQuit {
		return
	}

	columns, rowCount, err := csvInfo(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading CSV: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File: %s\n", csvPath)
	fmt.Printf("Columns: %d\n", len(columns))
	for _, col := range columns {
		fmt.Printf("  - %s\n", col)
	}
	fmt.Printf("Rows: %d\n\n", rowCount)

	if rowCount == 0 {
		fmt.Fprintln(os.Stderr, "Error: CSV file has no data rows.")
		os.Exit(1)
	}

	// ════════════════════════════════════════════════════════════
	// Step 2: Hill-Climb + BDeu Structure Learning
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 2: Hill-Climb + BDeu Structure Learning ---")
	fmt.Println()

	if config.StepPause("Running Hill-Climb search with BDeu scoring...", []string{
		"Hill-Climb is a greedy search that adds, removes, or reverses edges",
		"BDeu (Bayesian Dirichlet equivalent uniform) penalizes model complexity",
		"Discovers causal structure from observed data correlations",
	}) == interactive.ActionQuit {
		return
	}

	step2Start := time.Now()

	// TODO(v0.8.1): Migrate to FFI BN when available
	hcNet, err := nxuskit.NewBnNetwork()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating BN network: %v\n", err)
		os.Exit(1)
	}
	defer hcNet.Close()

	hcResult, err := hcNet.SearchStructure(csvPath, nxuskit.BnSearchStructureConfig{
		Algorithm:  "hill_climb",
		Scoring:    "bdeu",
		MaxParents: 5,
		MaxSteps:   1000,
	})

	step2Elapsed := time.Since(step2Start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Hill-Climb failed: %v\n", err)
	} else {
		fmt.Println("Algorithm: Hill-Climb")
		fmt.Println("Scoring: BDeu")
		fmt.Printf("Edges discovered: %d\n", len(hcResult.Edges))
		fmt.Printf("Score: %.2f\n", hcResult.Score)
		fmt.Printf("Iterations: %d\n", hcResult.Iterations)
		fmt.Println("Discovered edges:")
		printEdges(hcResult.Edges)

		if config.IsVerbose() {
			raw, _ := json.MarshalIndent(hcResult, "  ", "  ")
			fmt.Printf("\n[verbose] Raw HC result:\n  %s\n", raw)
		}
	}
	fmt.Printf("Time: %dms\n\n", step2Elapsed.Milliseconds())

	// ════════════════════════════════════════════════════════════
	// Step 3: K2 + BDeu Structure Learning
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 3: K2 + BDeu Structure Learning ---")
	fmt.Println()

	if config.StepPause("Running K2 search with BDeu scoring...", []string{
		"K2 is an order-based algorithm that requires a variable ordering",
		"BDeu (Bayesian Dirichlet equivalent uniform) is a Bayesian score",
		"Compares structure to Hill-Climb to assess algorithm sensitivity",
	}) == interactive.ActionQuit {
		return
	}

	step3Start := time.Now()

	k2Net, err := nxuskit.NewBnNetwork()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating BN network for K2: %v\n", err)
		os.Exit(1)
	}
	defer k2Net.Close()

	k2Result, err := k2Net.SearchStructure(csvPath, nxuskit.BnSearchStructureConfig{
		Algorithm:  "k2",
		Scoring:    "bdeu",
		MaxParents: 3,
		ESS:        10.0,
		Ordering:   columns, // use CSV column order
	})

	step3Elapsed := time.Since(step3Start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: K2 failed: %v\n", err)
	} else {
		fmt.Println("Algorithm: K2")
		fmt.Println("Scoring: BDeu (ESS=10.0)")
		fmt.Printf("Edges discovered: %d\n", len(k2Result.Edges))
		fmt.Printf("Score: %.2f\n", k2Result.Score)
		fmt.Println("Discovered edges:")
		printEdges(k2Result.Edges)

		if config.IsVerbose() {
			raw, _ := json.MarshalIndent(k2Result, "  ", "  ")
			fmt.Printf("\n[verbose] Raw K2 result:\n  %s\n", raw)
		}
	}
	fmt.Printf("Time: %dms\n\n", step3Elapsed.Milliseconds())

	// ════════════════════════════════════════════════════════════
	// Step 4: MLE Parameter Learning on Hill-Climb Structure
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 4: MLE Parameter Learning ---")
	fmt.Println()

	if config.StepPause("Learning CPT parameters via Maximum Likelihood Estimation...", []string{
		"Uses the Hill-Climb discovered structure as the model skeleton",
		"MLE estimates conditional probability tables from the CSV data",
		"Laplace smoothing (pseudocount=1.0) prevents zero-probability entries",
	}) == interactive.ActionQuit {
		return
	}

	step4Start := time.Now()
	err = hcNet.LearnMLE(csvPath, 1.0)
	step4Elapsed := time.Since(step4Start)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: MLE parameter learning failed: %v\n", err)
	} else {
		numVars := hcNet.NumVariables()
		fmt.Println("Structure: Hill-Climb (from Step 2)")
		fmt.Println("Pseudocount: 1.0 (Laplace smoothing)")
		fmt.Printf("Variables with learned CPTs: %d\n", numVars)
		fmt.Println("Status: OK")
	}
	fmt.Printf("Time: %dms\n\n", step4Elapsed.Milliseconds())

	// ════════════════════════════════════════════════════════════
	// Step 5: Log-Likelihood Fit Evaluation
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 5: Log-Likelihood Fit Evaluation ---")
	fmt.Println()

	if config.StepPause("Computing log-likelihood to evaluate model fit...", []string{
		"Log-likelihood measures how well the model explains the observed data",
		"Higher (less negative) values indicate better fit",
		"Normalized per-sample LL allows comparison across datasets",
	}) == interactive.ActionQuit {
		return
	}

	step5Start := time.Now()
	ll, err := hcNet.LogLikelihood(csvPath)
	step5Elapsed := time.Since(step5Start)

	perSample := math.Inf(-1)
	if rowCount > 0 {
		perSample = ll / float64(rowCount)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Log-likelihood computation failed: %v\n", err)
	} else {
		fmt.Printf("Log-likelihood: %.4f\n", ll)
		fmt.Printf("Per-sample LL: %.4f\n", perSample)
		fmt.Printf("Data rows: %d\n", rowCount)
		if !math.IsInf(ll, -1) {
			fmt.Println("Status: OK")
		} else {
			fmt.Println("Status: WARNING (non-finite log-likelihood)")
		}
	}
	fmt.Printf("Time: %dms\n\n", step5Elapsed.Milliseconds())

	// ════════════════════════════════════════════════════════════
	// Step 6: Inference on Learned Model
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 6: Inference on Learned Model ---")
	fmt.Println()

	if config.StepPause("Running Variable Elimination on the learned model...", []string{
		"Sets sample evidence to test the learned model",
		"Variable Elimination computes exact posteriors",
		"Demonstrates that the learned model supports standard BN queries",
	}) == interactive.ActionQuit {
		return
	}

	step6Start := time.Now()

	ev, err := nxuskit.NewBnEvidence()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating evidence: %v\n", err)
		os.Exit(1)
	}
	defer ev.Close()

	fmt.Println("Evidence:")
	evKeys := sortedKeys(scConfig.evidence)
	for _, k := range evKeys {
		if err := ev.SetDiscrete(hcNet, k, scConfig.evidence[k]); err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: Failed to set evidence %s=%s: %v\n", k, scConfig.evidence[k], err)
			continue
		}
		fmt.Printf("  %s = %s\n", k, scConfig.evidence[k])
	}
	fmt.Println()

	veResult, err := hcNet.Infer(ev, "ve")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Inference failed: %v\n", err)
	} else {
		defer veResult.Close()

		fmt.Printf("Query: P(%s | evidence)\n", scConfig.queryVar)
		dist, err := veResult.Marginal(scConfig.queryVar)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: Could not read marginal for %s: %v\n", scConfig.queryVar, err)
		} else {
			// Sort by probability descending
			type stateProb struct {
				state string
				prob  float64
			}
			var entries []stateProb
			for s, p := range dist {
				entries = append(entries, stateProb{s, p})
			}
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].prob > entries[j].prob
			})

			for _, e := range entries {
				bar := strings.Repeat("#", int(e.prob*40))
				fmt.Printf("  %-15s %.4f  %s\n", e.state, e.prob, bar)
			}
		}

		if config.IsVerbose() {
			fullJSON, _ := veResult.JSON()
			fmt.Printf("\n[verbose] Full inference result:\n  %s\n", fullJSON)
		}
	}

	step6Elapsed := time.Since(step6Start)
	fmt.Printf("Time: %dms\n\n", step6Elapsed.Milliseconds())

	// ════════════════════════════════════════════════════════════
	// Step 7: Structure Comparison
	// ════════════════════════════════════════════════════════════
	fmt.Println("--- Step 7: Structure Comparison ---")
	fmt.Println()

	if config.StepPause("Comparing Hill-Climb and K2 discovered structures...", []string{
		"Counts edges unique to each algorithm and shared edges",
		"Different algorithms may discover different causal relationships",
		"Shared edges are more likely to represent true causal links",
	}) == interactive.ActionQuit {
		return
	}

	var hcEdgeSet, k2EdgeSet []edgeKey
	if hcResult != nil {
		for _, e := range hcResult.Edges {
			hcEdgeSet = append(hcEdgeSet, edgeKey{e.From, e.To})
		}
	}
	if k2Result != nil {
		for _, e := range k2Result.Edges {
			k2EdgeSet = append(k2EdgeSet, edgeKey{e.From, e.To})
		}
	}

	shared, hcOnly, k2Only := compareEdges(hcEdgeSet, k2EdgeSet)

	fmt.Printf("Hill-Climb edges: %d\n", len(hcEdgeSet))
	fmt.Printf("K2 edges: %d\n", len(k2EdgeSet))
	fmt.Printf("Shared edges: %d\n", len(shared))

	if len(shared) > 0 {
		fmt.Println("Shared (high-confidence causal links):")
		for _, e := range shared {
			fmt.Printf("  %s -> %s\n", e.from, e.to)
		}
	}
	if len(hcOnly) > 0 {
		fmt.Println("Hill-Climb only:")
		for _, e := range hcOnly {
			fmt.Printf("  %s -> %s\n", e.from, e.to)
		}
	}
	if len(k2Only) > 0 {
		fmt.Println("K2 only:")
		for _, e := range k2Only {
			fmt.Printf("  %s -> %s\n", e.from, e.to)
		}
	}
	fmt.Println()

	// ── Summary ─────────────────────────────────────────────────
	totalElapsed := time.Since(totalStart)

	fmt.Println("========================================")
	fmt.Println("  Summary")
	fmt.Println("========================================")
	fmt.Printf("Scenario:       %s (%s)\n", scConfig.title, scenario)
	fmt.Printf("Data:           %d rows x %d columns\n", rowCount, len(columns))
	fmt.Printf("HC edges:       %d\n", len(hcEdgeSet))
	fmt.Printf("K2 edges:       %d\n", len(k2EdgeSet))
	fmt.Printf("Shared edges:   %d\n", len(shared))
	fmt.Printf("Log-likelihood: %.4f\n", ll)
	fmt.Printf("Per-sample LL:  %.4f\n", perSample)
	fmt.Printf("Total time:     %dms\n", totalElapsed.Milliseconds())
	fmt.Println()
	fmt.Println("Done.")
}

// ── Helper Functions ──────────────────────────────────────────────

// csvInfo reads a CSV file and returns column names and row count.
func csvInfo(path string) ([]string, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, fmt.Errorf("cannot parse CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, 0, fmt.Errorf("CSV file is empty")
	}

	columns := records[0]
	rowCount := len(records) - 1 // exclude header
	return columns, rowCount, nil
}

// printEdges displays edges in "from -> to" format.
func printEdges(edges []nxuskit.BnEdge) {
	if len(edges) == 0 {
		fmt.Println("  (no edges discovered)")
		return
	}
	for _, e := range edges {
		fmt.Printf("  %s -> %s\n", e.From, e.To)
	}
}

type edgeKey struct {
	from, to string
}

// compareEdges returns shared, hcOnly, and k2Only edge sets.
func compareEdges(hc, k2 []edgeKey) (shared, hcOnly, k2Only []edgeKey) {
	k2Set := make(map[edgeKey]bool)
	for _, e := range k2 {
		k2Set[e] = true
	}
	hcSet := make(map[edgeKey]bool)
	for _, e := range hc {
		hcSet[e] = true
	}

	for _, e := range hc {
		if k2Set[e] {
			shared = append(shared, e)
		} else {
			hcOnly = append(hcOnly, e)
		}
	}
	for _, e := range k2 {
		if !hcSet[e] {
			k2Only = append(k2Only, e)
		}
	}

	sort.Slice(shared, func(i, j int) bool { return edgeLess(shared[i], shared[j]) })
	sort.Slice(hcOnly, func(i, j int) bool { return edgeLess(hcOnly[i], hcOnly[j]) })
	sort.Slice(k2Only, func(i, j int) bool { return edgeLess(k2Only[i], k2Only[j]) })
	return
}

func edgeLess(a, b edgeKey) bool {
	if a.from != b.from {
		return a.from < b.from
	}
	return a.to < b.to
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// flagValue extracts the value for a flag like "--scenario golf" from os.Args.
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

// listAvailableScenarios prints scenario directories containing data.csv.
func listAvailableScenarios() {
	scenariosDir := filepath.Join("..", "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Available scenarios: golf, bmx, sourdough")
		return
	}
	fmt.Fprintln(os.Stderr, "Available scenarios:")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		csvFile := filepath.Join(scenariosDir, e.Name(), "data.csv")
		if _, err := os.Stat(csvFile); err == nil {
			fmt.Fprintf(os.Stderr, "  - %s\n", e.Name())
		}
	}
}
