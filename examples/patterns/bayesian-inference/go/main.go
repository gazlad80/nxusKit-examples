//go:build nxuskit

// Example: Bayesian Inference — Probabilistic Reasoning with Multiple Algorithms
//
// ## nxusKit Features Demonstrated
// - BnNetwork lifecycle (load BIF, query structure, close)
// - BnEvidence creation and discrete observation setting
// - Variable Elimination (VE) exact inference
// - Junction Tree (JT) exact inference
// - Loopy Belief Propagation (LBP) approximate inference
// - Gibbs Sampling with configurable samples, burn-in, and seed
// - Algorithm comparison across all four methods
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show network details and intermediate results
// - `--step` or `-s`: Pause at each inference step with explanations
//
// ## Scenario Selection
// - `--scenario <name>`: Load a model from ../scenarios/<name>/
// - Available scenarios: haunted-house, coffee-shop, plant-doctor
//
// Usage:
//
//	go run . --scenario haunted-house
//	go run . --scenario coffee-shop --verbose
//	go run . --scenario plant-doctor --step
//
// See the solver Go example for the Rust reference pattern.
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

// algorithmResult holds inference output for one algorithm.
type algorithmResult struct {
	name      string
	marginals map[string]map[string]float64 // variable -> state -> probability
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
	modelPath := filepath.Join(scenarioDir, "model.bif")
	evidencePath := filepath.Join(scenarioDir, "evidence.json")

	// Verify model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: cannot load scenario %q: model.bif not found\n", scenario)
		fmt.Fprintln(os.Stderr)
		listAvailableScenarios()
		os.Exit(1)
	}

	// Load evidence
	evidenceMap, err := loadEvidence(evidencePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot load evidence: %v\n", err)
		os.Exit(1)
	}

	// ── Load Bayesian Network ────────────────────────────────────
	fmt.Println("========================================")
	fmt.Printf("  Bayesian Inference: %s\n", scenario)
	fmt.Println("========================================")
	fmt.Println()

	if config.StepPause("Loading Bayesian Network from BIF file...", []string{
		"nxusKit: LoadBnNetwork() parses a BIF file into a network handle",
		"The network defines variables, their states, and conditional probability tables",
		"Evidence can be set to condition inference on observed values",
	}) == interactive.ActionQuit {
		return
	}

	net, err := nxuskit.LoadBnNetwork(modelPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading network: %v\n", err)
		os.Exit(1)
	}
	defer net.Close()

	// Print network summary
	variables, err := net.Variables()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading variables: %v\n", err)
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

	// ── Set Evidence ─────────────────────────────────────────────
	fmt.Println("Evidence:")
	evidenceKeys := sortedKeys(evidenceMap)
	for _, k := range evidenceKeys {
		fmt.Printf("  %s = %s\n", k, evidenceMap[k])
	}
	fmt.Println()

	if config.StepPause("Creating evidence and setting observations...", []string{
		"nxusKit: NewBnEvidence() creates an empty evidence set",
		"SetDiscrete() conditions inference on observed variable states",
		"Multiple observations can be set before running inference",
	}) == interactive.ActionQuit {
		return
	}

	// TODO(v0.8.1): Migrate to FFI BN when available
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

	// Collect all results for comparison
	var allResults []algorithmResult

	// ── Step 1: Variable Elimination ─────────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  Step 1: Variable Elimination (VE)")
	fmt.Println("----------------------------------------")
	fmt.Println()

	if config.StepPause("Running exact inference with Variable Elimination...", []string{
		"nxusKit: Infer() with algorithm 've' runs Variable Elimination",
		"VE is an exact method that sums out variables one at a time",
		"Optimal for small to medium networks",
	}) == interactive.ActionQuit {
		return
	}

	veResult, err := net.Infer(ev, "ve")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error (VE inference): %v\n", err)
		os.Exit(1)
	}
	veMarginals := extractMarginals(veResult, variables)
	veResult.Close()

	printMarginals("VE", veMarginals, variables, evidenceMap)
	allResults = append(allResults, algorithmResult{name: "VE", marginals: veMarginals})
	fmt.Println()

	// ── Step 2: Junction Tree ────────────────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  Step 2: Junction Tree (JT)")
	fmt.Println("----------------------------------------")
	fmt.Println()

	if config.StepPause("Running exact inference with Junction Tree...", []string{
		"nxusKit: Infer() with algorithm 'jt' runs Junction Tree",
		"JT compiles the network into a tree of cliques for message passing",
		"Also exact — results should match VE",
	}) == interactive.ActionQuit {
		return
	}

	jtResult, err := net.Infer(ev, "jt")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error (JT inference): %v\n", err)
		os.Exit(1)
	}
	jtMarginals := extractMarginals(jtResult, variables)
	jtResult.Close()

	printMarginals("JT", jtMarginals, variables, evidenceMap)
	allResults = append(allResults, algorithmResult{name: "JT", marginals: jtMarginals})
	fmt.Println()

	// ── Step 3: Loopy Belief Propagation ─────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  Step 3: Loopy Belief Propagation (LBP)")
	fmt.Println("----------------------------------------")
	fmt.Println()

	if config.StepPause("Running approximate inference with Loopy Belief Propagation...", []string{
		"nxusKit: Infer() with algorithm 'lbp' runs Loopy Belief Propagation",
		"LBP passes messages iteratively until convergence",
		"Approximate — results may differ slightly from exact methods",
	}) == interactive.ActionQuit {
		return
	}

	lbpResult, err := net.Infer(ev, "lbp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error (LBP inference): %v\n", err)
		os.Exit(1)
	}
	lbpMarginals := extractMarginals(lbpResult, variables)
	lbpResult.Close()

	printMarginals("LBP", lbpMarginals, variables, evidenceMap)
	allResults = append(allResults, algorithmResult{name: "LBP", marginals: lbpMarginals})
	fmt.Println()

	// ── Step 4: Gibbs Sampling ───────────────────────────────────
	fmt.Println("----------------------------------------")
	fmt.Println("  Step 4: Gibbs Sampling")
	fmt.Println("----------------------------------------")
	fmt.Println()

	const (
		gibbsSamples = 10000
		gibbsBurnIn  = 1000
		gibbsSeed    = 42
	)

	fmt.Printf("Samples: %d, Burn-in: %d, Seed: %d\n", gibbsSamples, gibbsBurnIn, gibbsSeed)
	fmt.Println()

	if config.StepPause("Running approximate inference with Gibbs Sampling...", []string{
		"nxusKit: InferWithOptions() configures Gibbs sampling parameters",
		"Gibbs is a Markov Chain Monte Carlo (MCMC) method",
		"More samples improve accuracy; burn-in discards initial instability",
	}) == interactive.ActionQuit {
		return
	}

	gibbsResult, err := net.InferWithOptions(ev, "gibbs", gibbsSamples, gibbsBurnIn, gibbsSeed)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error (Gibbs inference): %v\n", err)
		os.Exit(1)
	}
	gibbsMarginals := extractMarginals(gibbsResult, variables)
	gibbsResult.Close()

	printMarginals("Gibbs", gibbsMarginals, variables, evidenceMap)
	allResults = append(allResults, algorithmResult{name: "Gibbs", marginals: gibbsMarginals})
	fmt.Println()

	// ── Algorithm Comparison ─────────────────────────────────────
	fmt.Println("========================================")
	fmt.Println("  Algorithm Comparison")
	fmt.Println("========================================")
	fmt.Println()

	printComparisonTable(allResults, variables, evidenceMap)

	fmt.Println()
	fmt.Println("Done.")
}

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

// extractMarginals queries the marginal distribution for each variable.
func extractMarginals(result *nxuskit.BnResult, variables []string) map[string]map[string]float64 {
	marginals := make(map[string]map[string]float64)
	for _, v := range variables {
		dist, err := result.Marginal(v)
		if err != nil {
			continue
		}
		marginals[v] = dist
	}
	return marginals
}

// printMarginals displays posterior distributions for all non-evidence variables.
func printMarginals(algo string, marginals map[string]map[string]float64, variables []string, evidence map[string]string) {
	sorted := make([]string, len(variables))
	copy(sorted, variables)
	sort.Strings(sorted)

	fmt.Printf("Posterior marginals (%s):\n", algo)
	for _, v := range sorted {
		// Skip evidence variables — their marginal is trivially 1.0/0.0
		if _, isEvidence := evidence[v]; isEvidence {
			continue
		}
		dist, ok := marginals[v]
		if !ok {
			continue
		}
		states := sortedKeys(dist)
		parts := make([]string, 0, len(states))
		for _, s := range states {
			parts = append(parts, fmt.Sprintf("%s=%.4f", s, dist[s]))
		}
		fmt.Printf("  %-25s %s\n", v, strings.Join(parts, "  "))
	}
}

// printComparisonTable displays a side-by-side comparison of algorithm results.
func printComparisonTable(results []algorithmResult, variables []string, evidence map[string]string) {
	sorted := make([]string, len(variables))
	copy(sorted, variables)
	sort.Strings(sorted)

	// Use VE as reference for max-deviation calculation
	var veResult *algorithmResult
	for i := range results {
		if results[i].name == "VE" {
			veResult = &results[i]
			break
		}
	}

	// Header
	algoNames := make([]string, len(results))
	for i, r := range results {
		algoNames[i] = r.name
	}
	fmt.Printf("  %-20s %-8s", "Variable", "State")
	for _, name := range algoNames {
		fmt.Printf("  %-10s", name)
	}
	fmt.Println()

	divider := "  " + strings.Repeat("-", 20+8+len(results)*12)
	fmt.Println(divider)

	for _, v := range sorted {
		if _, isEvidence := evidence[v]; isEvidence {
			continue
		}
		// Gather all states across algorithms
		stateSet := make(map[string]bool)
		for _, r := range results {
			for s := range r.marginals[v] {
				stateSet[s] = true
			}
		}
		states := sortedKeysFromSet(stateSet)

		for _, s := range states {
			fmt.Printf("  %-20s %-8s", v, s)
			for _, r := range results {
				prob := 0.0
				if dist, ok := r.marginals[v]; ok {
					if p, ok := dist[s]; ok {
						prob = p
					}
				}
				fmt.Printf("  %-10.4f", prob)
			}
			fmt.Println()
		}
	}

	// Print max deviation from VE for each approximate algorithm
	if veResult != nil {
		fmt.Println()
		fmt.Println("  Max deviation from VE (exact):")
		for _, r := range results {
			if r.name == "VE" {
				continue
			}
			maxDev := 0.0
			for v, veDist := range veResult.marginals {
				if _, isEvidence := evidence[v]; isEvidence {
					continue
				}
				for s, veProb := range veDist {
					otherProb := 0.0
					if dist, ok := r.marginals[v]; ok {
						if p, ok := dist[s]; ok {
							otherProb = p
						}
					}
					dev := math.Abs(veProb - otherProb)
					if dev > maxDev {
						maxDev = dev
					}
				}
			}
			fmt.Printf("    %-10s %.6f\n", r.name, maxDev)
		}
	}
}

// sortedKeys returns the keys of a map[string]string in sorted order.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// sortedKeysFromSet returns sorted keys from a set (map[string]bool).
func sortedKeysFromSet(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// flagValue extracts the value for a flag like "--scenario haunted-house" from
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
		fmt.Fprintln(os.Stderr, "Available scenarios: haunted-house, coffee-shop, plant-doctor")
		return
	}
	fmt.Fprintln(os.Stderr, "Available scenarios:")
	for _, e := range entries {
		if e.IsDir() {
			// Verify model.bif exists
			mPath := filepath.Join(scenariosDir, e.Name(), "model.bif")
			if _, err := os.Stat(mPath); err == nil {
				fmt.Fprintf(os.Stderr, "  - %s\n", e.Name())
			}
		}
	}
}
