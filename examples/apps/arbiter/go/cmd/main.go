// Arbiter Example: Auto-Retry LLM with CLIPS Validation
//
// ## nxusKit Features Demonstrated
// - ClipsProvider for deterministic output validation
// - Auto-retry loop with CLIPS-driven parameter adjustment
// - JSON-based fact assertion for LLM output evaluation
// - Provider-agnostic solver pattern (works with any LLMProvider)
//
// ## Why This Pattern Matters
// LLM outputs can be inconsistent. Using CLIPS rules to validate outputs
// enables automatic retries with intelligent parameter adjustments.
// This ensures consistent, policy-compliant results.
//
// ## Interactive Modes
// - --verbose or -v: Shows raw request/response data for debugging
// - --step or -s: Pauses at each major operation with explanations
//
// Run with: go run ./cmd --help

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/arbiter"
	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
)

// Note: os is still used for os.Exit and os.Stderr

func main() {
	// Parse interactive mode flags first
	config := interactive.FromArgs()

	// Define command line flags
	configPath := os.Getenv("SOLVER_CONFIG")
	if configPath == "" {
		for i, arg := range os.Args[1:] {
			if arg == "--config" && i+2 < len(os.Args) {
				configPath = os.Args[i+2]
				break
			}
		}
	}
	inputText := ""
	conclusionType := "classification"
	categories := "high,medium,low"
	maxRetries := 3
	showHelp := false

	// Manual parsing (interactive.FromArgs scans os.Args without flag.Parse)
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case arg == "--config" && i+1 < len(os.Args):
			configPath = os.Args[i+1]
			i++
		case arg == "--input" && i+1 < len(os.Args):
			inputText = os.Args[i+1]
			i++
		case arg == "--type" && i+1 < len(os.Args):
			conclusionType = os.Args[i+1]
			i++
		case arg == "--categories" && i+1 < len(os.Args):
			categories = os.Args[i+1]
			i++
		case arg == "--max-retries" && i+1 < len(os.Args):
			fmt.Sscanf(os.Args[i+1], "%d", &maxRetries)
			i++
		case arg == "--help" || arg == "-h":
			showHelp = true
		case !strings.HasPrefix(arg, "-") && inputText == "":
			inputText = arg
		}
	}

	if showHelp {
		printHelp()
		return
	}

	// nxusKit: ClipsProvider is available for CLIPS validation
	// Build with -tags=clips to enable RealClipsValidator integration

	if inputText == "" {
		fmt.Fprintln(os.Stderr, "Error: No input provided. Use --input or provide text as argument.")
		printHelp()
		os.Exit(1)
	}

	// Parse conclusion type
	var ct arbiter.ConclusionType
	switch conclusionType {
	case "classification":
		ct = arbiter.Classification
	case "extraction":
		ct = arbiter.Extraction
	case "reasoning":
		ct = arbiter.Reasoning
	default:
		fmt.Fprintf(os.Stderr, "Unknown type: %s. Using classification.\n", conclusionType)
		ct = arbiter.Classification
	}

	// Parse categories
	cats := strings.Split(categories, ",")
	for i := range cats {
		cats[i] = strings.TrimSpace(cats[i])
	}

	// Step: Loading configuration
	if action := config.StepPause("Loading configuration...", []string{
		"Will load solver configuration from file or defaults",
		"Configuration includes retry strategies and validation rules",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// Load or create configuration
	var solverConfig arbiter.SolverConfig
	if configPath != "" {
		cfg, err := arbiter.LoadConfigFromFile(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		solverConfig = *cfg
	} else {
		solverConfig = arbiter.SolverConfig{
			MaxRetries:          maxRetries,
			Strategies:          arbiter.DefaultStrategies(),
			EvaluationRules:     "examples/rules/solver/classification-eval.clp",
			ConclusionType:      ct,
			ConfidenceThreshold: 0.7,
			TimeoutMS:           30000,
			ValidCategories:     cats,
		}
	}

	// Validate configuration
	if err := solverConfig.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Solver: %s Mode\n", solverConfig.ConclusionType)
	fmt.Printf("Input: \"%s\"\n", truncateStr(inputText, 60))
	if solverConfig.ConclusionType == arbiter.Classification {
		fmt.Printf("Valid categories: %v\n", solverConfig.ValidCategories)
	}
	fmt.Println()

	// Step: Running solver with CLIPS validation
	if action := config.StepPause("Running solver with CLIPS validation...", []string{
		"Will send input to LLM for classification",
		"CLIPS rules will validate the LLM output",
		"Auto-retry with parameter adjustment if validation fails",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// nxusKit: Create solver with real LLM provider and CLIPS validator
	llmSolver, err := arbiter.NewLLMSolverWithFallback()
	if err != nil {
		fmt.Fprintf(os.Stderr, "LLM provider error: %v\n", err)
		os.Exit(1)
	}

	validator, err := arbiter.NewClipsRulesValidator(
		"examples/apps/arbiter/shared/rules",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS validator error: %v\n", err)
		os.Exit(1)
	}

	s := arbiter.NewSolver(solverConfig, llmSolver, validator)
	result, err := s.Run(context.Background(), inputText, config.IsVerbose())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Solver error: %v\n", err)
		os.Exit(1)
	}

	// Print results
	printResult(result)

	if !result.Success {
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Arbiter Example: Auto-Retry LLM with CLIPS Validation")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("    go run ./cmd [OPTIONS] [INPUT]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("    --config <FILE>       Path to solver-config.json")
	fmt.Println("    --input <TEXT>        Input text to classify/extract/reason")
	fmt.Println("    --type <TYPE>         Conclusion type: classification, extraction, reasoning")
	fmt.Println("    --categories <LIST>   Comma-separated valid categories")
	fmt.Println("    --max-retries <N>     Maximum retry attempts (default: 3)")
	fmt.Println("    -v, --verbose         Show detailed retry information and raw requests/responses")
	fmt.Println("    -s, --step            Step through operations with pauses")
	fmt.Println("    -h, --help            Show this help message")
	fmt.Println()
	fmt.Println("INTERACTIVE MODES:")
	fmt.Println("    --verbose shows raw HTTP request/response data for debugging")
	fmt.Println("    --step pauses at each major operation for inspection")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    go run ./cmd --type classification \\")
	fmt.Println("        --categories \"high,medium,low\" \\")
	fmt.Println("        --input \"My account is hacked!\"")
	fmt.Println()
	fmt.Println("    go run ./cmd --config solver-config.json \\")
	fmt.Println("        --input \"Please reset my password\" --step")
}

func printResult(result *arbiter.SolverResult) {
	status := "SUCCESS"
	if !result.Success {
		status = "FAILED (max retries)"
	}

	fmt.Printf("Result: %s\n", status)

	// Parse final output
	var output map[string]any
	if err := json.Unmarshal(result.FinalOutput, &output); err == nil {
		if category, ok := output["category"]; ok {
			fmt.Printf("  Category: %v\n", category)
		}
		if conf, ok := output["confidence"]; ok {
			fmt.Printf("  Confidence: %v\n", conf)
		}
		if reasoning, ok := output["reasoning"].(string); ok && reasoning != "" {
			fmt.Printf("  Reasoning: %s\n", truncateStr(reasoning, 60))
		}
	}

	fmt.Printf("  Total attempts: %d\n", len(result.RetryHistory))
	fmt.Printf("  Total time: %dms\n", result.TotalDurationMS)
	fmt.Printf("  Total tokens: %d\n", result.TotalTokens)
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
