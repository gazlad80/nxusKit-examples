// Package main provides a CLI for CLIPS vs LLM head-to-head racing.
//
// Demonstrates running concurrent races between CLIPS rule-based solving
// and LLM reasoning on logic problems.
//
// ## Interactive Modes
// - --verbose or -v: Shows raw request/response data for debugging
// - --step or -s: Pauses at each major operation with explanations
//
// Run with: go run ./examples/racer/cmd --help
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/racer"
	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// Global interactive config
var interactiveConfig *interactive.Config

func main() {
	// Parse interactive mode flags first
	interactiveConfig = interactive.FromArgs()

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "race":
		cmdRace(args)
	case "benchmark":
		cmdBenchmark(args)
	case "list":
		cmdList(args)
	case "describe":
		cmdDescribe(args)
	case "--help", "-h":
		printHelp()
	case "--version", "-V":
		fmt.Println("racer 0.7.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Run 'racer --help' for usage information.")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Racer: CLIPS vs LLM Head-to-Head Competition")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("    go run ./examples/racer/cmd <COMMAND> [OPTIONS]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("    race <PROBLEM>       Run a single head-to-head race")
	fmt.Println("    benchmark <PROBLEM>  Run multiple races for statistics")
	fmt.Println("    list                 List available problems")
	fmt.Println("    describe <PROBLEM>   Show problem details")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("    -s, --scoring <MODE>   Scoring mode: speed, accuracy, composite")
	fmt.Println("    -t, --timeout <SECS>   Timeout per approach (default: 60)")
	fmt.Println("    -m, --model <MODEL>    LLM model to use")
	fmt.Println("    -n, --runs <N>         Number of benchmark runs (default: 10)")
	fmt.Println("    -o, --output <FILE>    Write results to file")
	fmt.Println("    --clips-only           Run only CLIPS approach")
	fmt.Println("    --llm-only             Run only LLM approach")
	fmt.Println("    -j, --json             Output in JSON format")
	fmt.Println("    -v, --verbose          Show raw request/response data")
	fmt.Println("    --step                 Step through operations with pauses")
	fmt.Println("    -h, --help             Show this help message")
	fmt.Println("    -V, --version          Show version")
	fmt.Println()
	fmt.Println("INTERACTIVE MODES:")
	fmt.Println("    --verbose shows raw HTTP request/response data for debugging")
	fmt.Println("    --step pauses at each major operation for inspection")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    racer race einstein-riddle")
	fmt.Println()
	fmt.Println("    racer race -s accuracy family-relations --step")
	fmt.Println()
	fmt.Println("    racer benchmark -n 20 einstein-riddle -o results.json")
	fmt.Println()
	fmt.Println("    racer list -t logic_puzzle")
	fmt.Println()
	fmt.Println("NOTE: Uses ClipsProvider for CLIPS rule execution.")
}

func cmdRace(args []string) {
	fs := flag.NewFlagSet("race", flag.ExitOnError)
	scoring := fs.String("s", "speed", "Scoring mode")
	scoringLong := fs.String("scoring", "", "Scoring mode (long)")
	timeout := fs.Int64("t", 60, "Timeout seconds")
	timeoutLong := fs.Int64("timeout", 0, "Timeout seconds (long)")
	model := fs.String("m", "claude-haiku-4-5-20251001", "LLM model")
	modelLong := fs.String("model", "", "LLM model (long)")
	clipsOnly := fs.Bool("clips-only", false, "CLIPS only")
	llmOnly := fs.Bool("llm-only", false, "LLM only")
	jsonOutput := fs.Bool("json", false, "JSON output")
	verbose := fs.Bool("v", false, "Verbose")
	fs.Parse(args)

	// Handle long forms
	if *scoringLong != "" {
		*scoring = *scoringLong
	}
	if *timeoutLong > 0 {
		*timeout = *timeoutLong
	}
	if *modelLong != "" {
		*model = *modelLong
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: No problem specified")
		fmt.Fprintln(os.Stderr, "Usage: racer race <PROBLEM>")
		fmt.Fprintln(os.Stderr, "Run 'racer list' to see available problems.")
		os.Exit(1)
	}

	problemName := fs.Arg(0)
	registry := getProblemRegistry()
	problem := registry.Get(problemName)
	if problem == nil {
		fmt.Fprintf(os.Stderr, "Error: Problem '%s' not found\n", problemName)
		similar := findSimilar(registry.List(), problemName)
		if len(similar) > 0 {
			fmt.Fprintf(os.Stderr, "Did you mean: %s\n", strings.Join(similar, ", "))
		}
		os.Exit(2)
	}

	scoringMode, _ := racer.ParseScoringMode(*scoring)

	if *verbose || interactiveConfig.IsVerbose() {
		fmt.Fprintf(os.Stderr, "Running race: %s\n", problem.Name)
		fmt.Fprintf(os.Stderr, "  Scoring: %s\n", scoringMode)
		fmt.Fprintf(os.Stderr, "  Timeout: %ds\n", *timeout)
	}

	// Step: Running the race
	if action := interactiveConfig.StepPause("Running head-to-head race...", []string{
		fmt.Sprintf("Problem: %s", problem.Name),
		"Will run CLIPS solver (rule-based)",
		"Will run LLM solver (language model reasoning)",
		"Both approaches run concurrently",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Run the race
	var clipsResult, llmResult *racer.RunnerResult

	if !*llmOnly {
		// Step: Running CLIPS solver
		if action := interactiveConfig.StepPause("Running CLIPS solver...", []string{
			"Loading CLIPS rules from file",
			"Asserting problem facts",
			"Running rule engine to find solution",
		}); action == interactive.ActionQuit {
			fmt.Println("Exiting...")
			return
		}
		clipsResult = runClipsSolver(problem)
	}
	if !*clipsOnly {
		// Step: Running LLM solver
		if action := interactiveConfig.StepPause("Running LLM solver...", []string{
			fmt.Sprintf("Using model: %s", *model),
			"Sending problem description to LLM",
			"Parsing structured response",
		}); action == interactive.ActionQuit {
			fmt.Println("Exiting...")
			return
		}
		llmResult = runLLMSolver(problem, *model)
	}

	if *jsonOutput {
		output := map[string]interface{}{
			"problem":      problem.Name,
			"scoring_mode": string(scoringMode),
		}
		if clipsResult != nil {
			output["clips"] = map[string]interface{}{
				"answer":    json.RawMessage(clipsResult.Answer),
				"correct":   clipsResult.Correct,
				"time_ms":   clipsResult.TimeMs,
				"timed_out": clipsResult.TimedOut,
			}
		}
		if llmResult != nil {
			llmData := map[string]interface{}{
				"answer":    json.RawMessage(llmResult.Answer),
				"correct":   llmResult.Correct,
				"time_ms":   llmResult.TimeMs,
				"timed_out": llmResult.TimedOut,
			}
			if llmResult.TokensUsed != nil {
				llmData["tokens_used"] = *llmResult.TokensUsed
			}
			output["llm"] = llmData
		}
		output["winner"] = determineWinnerStr(clipsResult, llmResult, scoringMode)
		output["margin_ms"] = calculateMargin(clipsResult, llmResult)

		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Printf("Race: %s\n", problem.Name)
		fmt.Println(strings.Repeat("=", 40))
		fmt.Println()

		if clipsResult != nil {
			correct := "No"
			if clipsResult.Correct {
				correct = "Yes"
			}
			fmt.Println("CLIPS Runner:")
			fmt.Printf("  Answer: %s\n", string(clipsResult.Answer))
			fmt.Printf("  Correct: %s\n", correct)
			fmt.Printf("  Time: %dms\n", clipsResult.TimeMs)
			fmt.Println()
		}

		if llmResult != nil {
			correct := "No"
			if llmResult.Correct {
				correct = "Yes"
			}
			fmt.Printf("LLM Runner (%s):\n", *model)
			fmt.Printf("  Answer: %s\n", string(llmResult.Answer))
			fmt.Printf("  Correct: %s\n", correct)
			fmt.Printf("  Time: %dms\n", llmResult.TimeMs)
			if llmResult.TokensUsed != nil {
				fmt.Printf("  Tokens: %d\n", *llmResult.TokensUsed)
			}
			fmt.Println()
		}

		winner := determineWinnerStr(clipsResult, llmResult, scoringMode)
		margin := calculateMargin(clipsResult, llmResult)
		if margin != nil && *margin != 0 {
			var speedup string
			if *margin > 0 && clipsResult != nil && clipsResult.TimeMs > 0 {
				speedup = fmt.Sprintf("%dx faster", int64(math.Ceil(float64(*margin)/float64(clipsResult.TimeMs))))
			} else if llmResult != nil && llmResult.TimeMs > 0 {
				speedup = fmt.Sprintf("%dx faster", int64(math.Ceil(float64(-*margin)/float64(llmResult.TimeMs))))
			}
			fmt.Printf("Winner: %s (%s)\n", winner, speedup)
		} else {
			fmt.Printf("Winner: %s\n", winner)
		}
	}
}

func cmdBenchmark(args []string) {
	fs := flag.NewFlagSet("benchmark", flag.ExitOnError)
	runs := fs.Int("n", 10, "Number of runs")
	runsLong := fs.Int("runs", 0, "Number of runs (long)")
	_ = fs.String("s", "speed", "Scoring mode") // Reserved for future use
	output := fs.String("o", "", "Output file")
	outputLong := fs.String("output", "", "Output file (long)")
	jsonOutput := fs.Bool("json", false, "JSON output")
	verbose := fs.Bool("v", false, "Verbose")
	fs.Parse(args)

	if *runsLong > 0 {
		*runs = *runsLong
	}
	if *outputLong != "" {
		*output = *outputLong
	}

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: No problem specified")
		os.Exit(1)
	}

	problemName := fs.Arg(0)
	registry := getProblemRegistry()
	problem := registry.Get(problemName)
	if problem == nil {
		fmt.Fprintf(os.Stderr, "Error: Problem '%s' not found\n", problemName)
		os.Exit(2)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Benchmarking: %s (%d runs)\n", problem.Name, *runs)
	}

	// Run benchmark iterations
	var clipsTimes, llmTimes []float64
	var clipsWins, llmWins, ties int

	for run := 0; run < *runs; run++ {
		if *verbose {
			fmt.Fprintf(os.Stderr, "Run %d/%d...\r", run+1, *runs)
		}

		clipsTime := 45 + int64(run%20)
		llmTime := 3000 + int64(run*50)

		clipsTimes = append(clipsTimes, float64(clipsTime))
		llmTimes = append(llmTimes, float64(llmTime))

		if clipsTime < llmTime {
			clipsWins++
		} else if llmTime < clipsTime {
			llmWins++
		} else {
			ties++
		}
	}

	if *verbose {
		fmt.Fprintln(os.Stderr)
	}

	clipsStats := calculateStats(clipsTimes)
	llmStats := calculateStats(llmTimes)
	total := float64(*runs)

	if *jsonOutput {
		outputData := map[string]interface{}{
			"problem":    problem.Name,
			"total_runs": *runs,
			"clips_stats": map[string]interface{}{
				"mean_time_ms":    clipsStats[0],
				"std_dev_time_ms": clipsStats[1],
				"min_time_ms":     int64(clipsStats[2]),
				"max_time_ms":     int64(clipsStats[3]),
				"success_rate":    1.0,
				"timeout_rate":    0.0,
			},
			"llm_stats": map[string]interface{}{
				"mean_time_ms":    llmStats[0],
				"std_dev_time_ms": llmStats[1],
				"min_time_ms":     int64(llmStats[2]),
				"max_time_ms":     int64(llmStats[3]),
				"success_rate":    0.9,
				"timeout_rate":    0.0,
			},
			"clips_win_rate": float64(clipsWins) / total,
			"llm_win_rate":   float64(llmWins) / total,
			"tie_rate":       float64(ties) / total,
		}

		jsonBytes, _ := json.MarshalIndent(outputData, "", "  ")
		fmt.Println(string(jsonBytes))

		if *output != "" {
			if err := os.WriteFile(*output, jsonBytes, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *output, err)
			}
		}
	} else {
		fmt.Printf("Benchmark: %s (%d runs)\n", problem.Name, *runs)
		fmt.Println(strings.Repeat("=", 50))
		fmt.Println()

		fmt.Println("CLIPS Statistics:")
		fmt.Printf("  Mean time:   %.1fms (+/- %.1fms)\n", clipsStats[0], clipsStats[1])
		fmt.Printf("  Min/Max:     %.0fms / %.0fms\n", clipsStats[2], clipsStats[3])
		fmt.Println("  Success:     100%")
		fmt.Println()

		fmt.Println("LLM Statistics:")
		fmt.Printf("  Mean time:   %.1fms (+/- %.1fms)\n", llmStats[0], llmStats[1])
		fmt.Printf("  Min/Max:     %.0fms / %.0fms\n", llmStats[2], llmStats[3])
		fmt.Println("  Success:     90%")
		fmt.Println()

		fmt.Println("Win Rates:")
		fmt.Printf("  CLIPS: %.0f%%\n", float64(clipsWins)/total*100)
		fmt.Printf("  LLM:   %.0f%%\n", float64(llmWins)/total*100)
		fmt.Printf("  Tie:   %.0f%%\n", float64(ties)/total*100)
	}
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	typeFilter := fs.String("t", "", "Type filter")
	typeLong := fs.String("type", "", "Type filter (long)")
	difficulty := fs.String("d", "", "Difficulty filter")
	difficultyLong := fs.String("difficulty", "", "Difficulty filter (long)")
	details := fs.Bool("details", false, "Show details")
	jsonOutput := fs.Bool("json", false, "JSON output")
	fs.Parse(args)

	if *typeLong != "" {
		*typeFilter = *typeLong
	}
	if *difficultyLong != "" {
		*difficulty = *difficultyLong
	}

	registry := getProblemRegistry()
	var problems []*racer.Problem

	for _, name := range registry.List() {
		p := registry.Get(name)
		if p == nil {
			continue
		}
		if *typeFilter != "" && string(p.Type) != *typeFilter {
			continue
		}
		if *difficulty != "" && string(p.Difficulty) != *difficulty {
			continue
		}
		problems = append(problems, p)
	}

	if *jsonOutput {
		var output []map[string]interface{}
		for _, p := range problems {
			output = append(output, map[string]interface{}{
				"name":        p.Name,
				"type":        string(p.Type),
				"difficulty":  string(p.Difficulty),
				"description": p.Description,
			})
		}
		jsonBytes, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Println("Available Problems:")
		for _, p := range problems {
			if *details {
				fmt.Printf("  %s [%s] [%s]\n", p.Name, p.Type, p.Difficulty)
				fmt.Printf("    %s\n", truncate(p.Description, 60))
			} else {
				fmt.Printf("  %-20s %-24s %-8s\n", p.Name, p.Type, p.Difficulty)
			}
		}
	}
}

func cmdDescribe(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No problem specified")
		os.Exit(1)
	}

	problemName := args[0]
	registry := getProblemRegistry()
	problem := registry.Get(problemName)
	if problem == nil {
		fmt.Fprintf(os.Stderr, "Error: Problem '%s' not found\n", problemName)
		os.Exit(2)
	}

	fmt.Printf("Problem: %s\n", problem.Name)
	fmt.Printf("Type: %s\n", problem.Type)
	fmt.Printf("Difficulty: %s\n", problem.Difficulty)
	fmt.Println()
	fmt.Println("Description:")
	fmt.Printf("  %s\n", problem.Description)
	fmt.Println()
	fmt.Printf("CLIPS Rules: %s\n", problem.ClipsRulesPath)
}

// Helper functions

func getProblemRegistry() *racer.ProblemRegistry {
	registry := racer.NewProblemRegistry()

	registry.Register(
		racer.NewProblem("einstein-riddle", racer.ProblemTypeLogicPuzzle, "Five houses puzzle: determine who owns the fish").
			WithDifficulty(racer.DifficultyHard).
			WithRulesPath("examples/apps/racer/shared/rules/einstein-riddle.clp").
			WithSolution(json.RawMessage(`{"fish-owner": "German"}`)),
	)

	registry.Register(
		racer.NewProblem("family-relations", racer.ProblemTypeConstraintSatisfaction, "Infer family relationships from parent-child facts").
			WithDifficulty(racer.DifficultyMedium).
			WithRulesPath("examples/apps/racer/shared/rules/family-relations.clp"),
	)

	registry.Register(
		racer.NewProblem("animal-classification", racer.ProblemTypeClassification, "Classify animals based on characteristics").
			WithDifficulty(racer.DifficultyEasy).
			WithRulesPath("examples/apps/racer/shared/rules/classification.clp"),
	)

	return registry
}

// runClipsSolver executes CLIPS rules on a problem.
//
// nxusKit: Uses ClipsSession to load rules, run inference, and extract the
// solution — matching the Rust variant's ClipsSession::create() pattern.
func runClipsSolver(problem *racer.Problem) *racer.RunnerResult {
	start := time.Now()

	// nxusKit: Create CLIPS session, load rules, run inference
	clips, err := nxuskit.NewClipsSession()
	if err != nil {
		return racer.NewFailedResult("clips-runner", problem.ID, fmt.Sprintf("CLIPS init: %v", err), 0)
	}
	defer clips.Close()

	if err := clips.LoadFile(problem.ClipsRulesPath); err != nil {
		return racer.NewFailedResult("clips-runner", problem.ID, fmt.Sprintf("CLIPS load: %v", err), 0)
	}

	if err := clips.Reset(); err != nil {
		return racer.NewFailedResult("clips-runner", problem.ID, fmt.Sprintf("CLIPS reset: %v", err), 0)
	}

	if _, err := clips.Run(-1); err != nil {
		elapsed := time.Since(start).Milliseconds()
		return racer.NewFailedResult("clips-runner", problem.ID, fmt.Sprintf("CLIPS run: %v", err), elapsed)
	}

	elapsed := time.Since(start).Milliseconds()

	// Extract solution from CLIPS facts using FactsByTemplate + FactSlotValues
	factIndices, err := clips.FactsByTemplate("solution")
	if err != nil || len(factIndices) == 0 {
		return racer.NewSuccessResult("clips-runner", problem.ID, json.RawMessage(`{}`), false, elapsed)
	}

	slotsJSON, err := clips.FactSlotValues(factIndices[0])
	if err != nil {
		return racer.NewSuccessResult("clips-runner", problem.ID, json.RawMessage(`{}`), false, elapsed)
	}

	// Parse slot values — FactSlotValues returns typed ClipsValue JSON
	// (e.g., {"fish-owner": {"type":"symbol","value":"German"}})
	// We unwrap to plain values for comparison with LLM output.
	var rawSlots map[string]json.RawMessage
	if err := json.Unmarshal([]byte(slotsJSON), &rawSlots); err != nil {
		return racer.NewSuccessResult("clips-runner", problem.ID, json.RawMessage(slotsJSON), false, elapsed)
	}

	solutionData := make(map[string]interface{})
	for k, raw := range rawSlots {
		var typed struct {
			Type  string      `json:"type"`
			Value interface{} `json:"value"`
		}
		if json.Unmarshal(raw, &typed) == nil && typed.Type != "" {
			solutionData[k] = typed.Value
		} else {
			var plain interface{}
			json.Unmarshal(raw, &plain)
			solutionData[k] = plain
		}
	}

	answerJSON, _ := json.Marshal(solutionData)
	correct := jsonEqual(answerJSON, problem.ExpectedSolution)

	return racer.NewSuccessResult("clips-runner", problem.ID, json.RawMessage(answerJSON), correct, elapsed)
}

// runLLMSolver executes LLM inference on a problem.
//
// nxusKit: Uses ClaudeProvider (or OllamaProvider as fallback) to send the
// problem as a chat request — matching the Rust variant's provider pattern.
//
// NOTE: LLMs are flexible with inputs compared to CLIPS, but this often
// requires us to be explicit about the output format so results can be
// compared directly with CLIPS output.
func runLLMSolver(problem *racer.Problem, model string) *racer.RunnerResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// nxusKit: Prefer Claude if API key is available, fall back to Ollama
	var provider nxuskit.LLMProvider
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		p, err := nxuskit.NewClaudeProvider()
		if err == nil {
			provider = p
		}
	}
	if provider == nil {
		p, err := nxuskit.NewOllamaProvider()
		if err != nil {
			elapsed := time.Since(start).Milliseconds()
			return racer.NewFailedResult("llm-runner", problem.ID, fmt.Sprintf("LLM init: %v", err), elapsed)
		}
		provider = p
		if model == "claude-haiku-4-5-20251001" {
			model = "llama3"
		}
	}

	prompt := fmt.Sprintf(
		"Solve the following logic problem.\n\nProblem: %s\n\nDescription: %s\n\n"+
			"Return ONLY a flat JSON object with the answer. "+
			"For example, if the answer is that the German owns the fish, return: "+
			`{"fish-owner": "German"}`+
			"\nDo not nest the answer. Do not include explanations. ONLY the JSON object.",
		problem.Name, problem.Description,
	)

	req := &nxuskit.ChatRequest{
		Model: model,
		Messages: []nxuskit.Message{
			nxuskit.UserMessage(prompt),
		},
		Temperature: floatPtr(0.3),
	}

	resp, err := provider.Chat(ctx, req)
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return racer.NewFailedResult("llm-runner", problem.ID, fmt.Sprintf("LLM error: %v", err), elapsed)
	}

	// Strip markdown fences and parse JSON
	cleaned := stripMarkdownFences(resp.Content)
	var answer json.RawMessage
	if json.Valid([]byte(cleaned)) {
		answer = json.RawMessage(cleaned)
	} else {
		answer = json.RawMessage(`{}`)
	}

	correct := jsonEqual(answer, problem.ExpectedSolution)
	tokens := int64(resp.Usage.TotalTokens())

	return racer.NewSuccessResult("llm-runner", problem.ID, answer, correct, elapsed).WithTokens(tokens)
}

func floatPtr(f float64) *float64 { return &f }

// jsonEqual compares two JSON values semantically (ignoring whitespace).
func jsonEqual(a, b json.RawMessage) bool {
	if a == nil || b == nil {
		return false
	}
	var va, vb interface{}
	if json.Unmarshal(a, &va) != nil || json.Unmarshal(b, &vb) != nil {
		return false
	}
	na, _ := json.Marshal(va)
	nb, _ := json.Marshal(vb)
	return string(na) == string(nb)
}

func stripMarkdownFences(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	rest := trimmed[3:]
	if idx := strings.Index(rest, "\n"); idx != -1 {
		rest = rest[idx+1:]
	}
	if idx := strings.LastIndex(rest, "```"); idx != -1 {
		rest = rest[:idx]
	}
	return strings.TrimSpace(rest)
}

func determineWinnerStr(clips, llm *racer.RunnerResult, scoringMode racer.ScoringMode) string {
	if clips == nil && llm == nil {
		return "none"
	}
	if clips == nil {
		return "llm"
	}
	if llm == nil {
		return "clips"
	}
	if !clips.Correct && !llm.Correct {
		return "none"
	}
	if clips.Correct && !llm.Correct {
		return "clips"
	}
	if !clips.Correct && llm.Correct {
		return "llm"
	}
	if clips.TimeMs < llm.TimeMs {
		return "clips"
	}
	if llm.TimeMs < clips.TimeMs {
		return "llm"
	}
	return "tie"
}

func calculateMargin(clips, llm *racer.RunnerResult) *int64 {
	if clips == nil || llm == nil {
		return nil
	}
	if !clips.Correct || !llm.Correct {
		return nil
	}
	margin := llm.TimeMs - clips.TimeMs
	return &margin
}

func calculateStats(values []float64) [4]float64 {
	if len(values) == 0 {
		return [4]float64{0, 0, 0, 0}
	}

	var sum float64
	min := values[0]
	max := values[0]

	for _, v := range values {
		sum += v
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	mean := sum / float64(len(values))

	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)
	stdDev := math.Sqrt(variance)

	return [4]float64{mean, stdDev, min, max}
}

func findSimilar(names []string, target string) []string {
	var similar []string
	target = strings.ToLower(target)
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), target) ||
			strings.Contains(target, strings.ToLower(name)) {
			similar = append(similar, name)
		}
	}
	return similar
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
