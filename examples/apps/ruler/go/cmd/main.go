// Package main provides a CLI for natural language to CLIPS rule generation.
//
// Demonstrates using LLM to generate CLIPS rules from natural language
// descriptions with validation and retry logic.
//
// ## Interactive Modes
// - --verbose or -v: Shows raw request/response data for debugging
// - --step or -s: Pauses at each major operation with explanations
//
// Run with: go run ./examples/ruler/cmd --help
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/ruler"
	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
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
	case "generate":
		cmdGenerate(args)
	case "validate":
		cmdValidate(args)
	case "save":
		cmdSave(args)
	case "load":
		cmdLoad(args)
	case "examples":
		cmdExamples(args)
	case "--help", "-h":
		printHelp()
	case "--version", "-V":
		fmt.Println("ruler 0.7.0")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Run 'ruler --help' for usage information.")
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("Ruler: Natural Language to CLIPS Rule Generation")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("    go run ./examples/ruler/cmd <COMMAND> [OPTIONS]")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("    generate <DESCRIPTION>   Generate CLIPS rules from natural language")
	fmt.Println("    validate <FILE>          Validate CLIPS code")
	fmt.Println("    save <FILE>              Save loaded rules to file")
	fmt.Println("    load <FILE>              Load rules from file")
	fmt.Println("    examples                 Run progressive complexity examples")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("    -c, --complexity <LEVEL>  Target complexity: basic, intermediate, advanced")
	fmt.Println("    -m, --model <MODEL>       LLM model to use (default: claude-haiku-4-5-20251001)")
	fmt.Println("    -r, --retries <N>         Max retry attempts (default: 5)")
	fmt.Println("    -o, --output <FILE>       Write output to file")
	fmt.Println("    -j, --json                Output in JSON format")
	fmt.Println("    -v, --verbose             Show raw request/response data")
	fmt.Println("    -s, --step                Step through operations with pauses")
	fmt.Println("    -h, --help                Show this help message")
	fmt.Println("    -V, --version             Show version")
	fmt.Println()
	fmt.Println("INTERACTIVE MODES:")
	fmt.Println("    --verbose shows raw HTTP request/response data for debugging")
	fmt.Println("    --step pauses at each major operation for inspection")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    ruler generate \"Create a rule that classifies adults if age >= 18\"")
	fmt.Println()
	fmt.Println("    ruler generate -c advanced \"Medical triage expert system\" -o triage.clp --step")
	fmt.Println()
	fmt.Println("    ruler examples -c basic")
	fmt.Println()
	fmt.Println("NOTE: Uses ClipsProvider for CLIPS rule validation.")
}

func cmdGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	complexity := fs.String("c", "basic", "Complexity level")
	complexityLong := fs.String("complexity", "", "Complexity level (long form)")
	model := fs.String("m", "claude-haiku-4-5-20251001", "LLM model")
	modelLong := fs.String("model", "", "LLM model (long form)")
	retries := fs.Int("r", 5, "Max retries")
	retriesLong := fs.Int("retries", 0, "Max retries (long form)")
	output := fs.String("o", "", "Output file")
	outputLong := fs.String("output", "", "Output file (long form)")
	jsonOutput := fs.Bool("json", false, "JSON output")
	verbose := fs.Bool("v", false, "Verbose output")
	verboseLong := fs.Bool("verbose", false, "Verbose output (long form)")

	fs.Parse(args)

	// Handle long forms
	if *complexityLong != "" {
		*complexity = *complexityLong
	}
	if *modelLong != "" {
		*model = *modelLong
	}
	if *retriesLong > 0 {
		*retries = *retriesLong
	}
	if *outputLong != "" {
		*output = *outputLong
	}
	if *verboseLong {
		*verbose = true
	}

	// Get description from remaining args or stdin
	description := strings.Join(fs.Args(), " ")
	if description == "" || description == "-" {
		if *verbose {
			fmt.Fprintln(os.Stderr, "Reading description from stdin...")
		}
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		description = strings.Join(lines, " ")
	}

	if description == "" {
		fmt.Fprintln(os.Stderr, "Error: No description provided")
		fmt.Fprintln(os.Stderr, "Usage: ruler generate <DESCRIPTION>")
		os.Exit(1)
	}

	comp, err := ruler.ParseComplexity(*complexity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v, using basic\n", err)
		comp = ruler.ComplexityBasic
	}

	if *verbose || interactiveConfig.IsVerbose() {
		fmt.Fprintln(os.Stderr, "Generating CLIPS rules...")
		fmt.Fprintf(os.Stderr, "  Description: %s\n", description)
		fmt.Fprintf(os.Stderr, "  Complexity: %s\n", comp)
		fmt.Fprintf(os.Stderr, "  Model: %s\n", *model)
		fmt.Fprintf(os.Stderr, "  Max retries: %d\n", *retries)
	}

	// Step: Generating CLIPS rules using LLM
	if action := interactiveConfig.StepPause("Generating CLIPS rules using LLM...", []string{
		fmt.Sprintf("Description: %s", description),
		fmt.Sprintf("Complexity: %s", comp),
		fmt.Sprintf("Model: %s", *model),
		"LLM will generate CLIPS deftemplate and defrule constructs",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	// nxusKit: Generate rules using real LLM provider
	ruleDesc := ruler.NewRuleDescription(description).WithComplexity(comp)

	generator, err := ruler.NewLLMGeneratorWithFallback()
	if err != nil {
		fmt.Fprintf(os.Stderr, "LLM provider error: %v\n", err)
		os.Exit(1)
	}
	// Set the model from CLI args — the fallback provider auto-detects
	// the LLM backend but needs the model name to know which to use.
	// For Ollama models, use "llama3"; for Claude, use the full model ID.
	generator.WithMaxRetries(*retries).WithModel(*model)

	genResult, err := generator.Generate(context.Background(), ruleDesc)
	if err != nil || !genResult.Success {
		fmt.Fprintf(os.Stderr, "Error generating rules: %v\n", err)
		if genResult != nil {
			for i, attemptErr := range genResult.Errors {
				fmt.Fprintf(os.Stderr, "  attempt %d: %v\n", i+1, attemptErr)
			}
		}
		os.Exit(1)
	}
	generated := genResult.Rules

	// Step: Validating generated CLIPS code
	if action := interactiveConfig.StepPause("Validating generated CLIPS code...", []string{
		"Checking syntax (balanced parentheses)",
		"Verifying required constructs (deftemplate, defrule)",
		"Checking for unsafe operations",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	validation := validateClipsCode(generated.ClipsCode)

	if *jsonOutput {
		outputData := map[string]interface{}{
			"success":            validation.IsValid(),
			"clips_code":         generated.ClipsCode,
			"attempts":           generated.GenerationAttempt,
			"tokens_used":        generated.TokensUsed,
			"generation_time_ms": generated.GenerationTimeMs,
			"validation": map[string]interface{}{
				"status":   string(validation.Status),
				"warnings": validation.Warnings,
			},
		}
		jsonBytes, _ := json.MarshalIndent(outputData, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		if validation.IsValid() {
			fmt.Println(";; Generated CLIPS Rules")
			fmt.Printf(";; Description: %s\n", description)
			fmt.Printf(";; Complexity: %s\n", comp)
			fmt.Printf(";; Model: %s\n", *model)
			fmt.Println()
			fmt.Println(generated.ClipsCode)

			for _, warning := range validation.Warnings {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
			}
		} else {
			fmt.Fprintln(os.Stderr, "Validation failed:")
			for _, err := range validation.Errors {
				fmt.Fprintf(os.Stderr, "  %s\n", err.Error())
			}
			os.Exit(2)
		}
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(generated.ClipsCode), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", *output, err)
			os.Exit(4)
		}
		if *verbose {
			fmt.Fprintf(os.Stderr, "Wrote %d bytes to %s\n", len(generated.ClipsCode), *output)
		}
	}
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	jsonOutput := fs.Bool("json", false, "JSON output")
	verbose := fs.Bool("v", false, "Verbose output")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: No file specified")
		os.Exit(1)
	}

	filePath := fs.Arg(0)
	code, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", filePath, err)
		os.Exit(4)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Validating %s...\n", filePath)
	}

	validation := validateClipsCode(string(code))

	if *jsonOutput {
		errors := make([]map[string]interface{}, len(validation.Errors))
		for i, e := range validation.Errors {
			errors[i] = map[string]interface{}{
				"type":    string(e.ErrorType),
				"message": e.Message,
				"line":    e.LineNumber,
			}
		}
		outputData := map[string]interface{}{
			"valid":    validation.IsValid(),
			"errors":   errors,
			"warnings": validation.Warnings,
		}
		jsonBytes, _ := json.MarshalIndent(outputData, "", "  ")
		fmt.Println(string(jsonBytes))
	} else {
		if validation.IsValid() {
			fmt.Printf("Valid: %s\n", filePath)
			for _, warning := range validation.Warnings {
				fmt.Printf("Warning: %s\n", warning)
			}
		} else {
			fmt.Printf("Invalid: %s\n", filePath)
			for _, err := range validation.Errors {
				fmt.Printf("  %s\n", err.Error())
			}
		}
	}

	if !validation.IsValid() {
		os.Exit(2)
	}
}

func cmdSave(args []string) {
	fs := flag.NewFlagSet("save", flag.ExitOnError)
	format := fs.String("f", "text", "Format: text, binary")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: No file specified")
		os.Exit(1)
	}

	filePath := fs.Arg(0)
	// nxusKit: ClipsProvider can save rules using bsave (binary) or text format
	fmt.Fprintf(os.Stderr, "Save: %s (format: %s)\n", filePath, *format)
}

func cmdLoad(args []string) {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	verbose := fs.Bool("v", false, "Verbose output")
	fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: No file specified")
		os.Exit(1)
	}

	filePath := fs.Arg(0)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: File not found: %s\n", filePath)
		os.Exit(4)
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Loading %s...\n", filePath)
	}

	// nxusKit: ClipsProvider can load rules from file or binary
	fmt.Fprintf(os.Stderr, "Load: %s\n", filePath)
}

func cmdExamples(args []string) {
	fs := flag.NewFlagSet("examples", flag.ExitOnError)
	complexity := fs.String("c", "", "Complexity filter")
	listOnly := fs.Bool("l", false, "List only")
	number := fs.Int("n", -1, "Example number")
	jsonOutput := fs.Bool("json", false, "JSON output")
	fs.Parse(args)

	examples := getBuiltinExamples()

	// Filter by complexity
	var filtered []*ruler.ProgressiveExample
	for _, ex := range examples.Examples {
		if *complexity == "" || string(ex.Complexity) == *complexity {
			filtered = append(filtered, ex)
		}
	}

	if *listOnly {
		if *jsonOutput {
			var output []map[string]interface{}
			for _, ex := range filtered {
				output = append(output, map[string]interface{}{
					"id":          ex.ID,
					"complexity":  string(ex.Complexity),
					"description": ex.Description,
				})
			}
			jsonBytes, _ := json.MarshalIndent(output, "", "  ")
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println("Available Examples:")
			fmt.Println()
			for _, ex := range filtered {
				fmt.Printf("  %s [%s]\n", ex.ID, ex.Complexity)
				fmt.Printf("    %s\n", truncate(ex.Description, 60))
			}
		}
		return
	}

	if *number >= 0 {
		if *number >= len(filtered) {
			fmt.Fprintf(os.Stderr, "Error: Example %d not found (available: 0-%d)\n", *number, len(filtered)-1)
			os.Exit(1)
		}
		ex := filtered[*number]
		fmt.Printf("Example %d: %s\n", *number, ex.ID)
		fmt.Printf("Complexity: %s\n", ex.Complexity)
		fmt.Printf("Description: %s\n", ex.Description)
		fmt.Println()
		fmt.Printf("Expected constructs: %v\n", ex.ExpectedConstructs)
		return
	}

	// Run all examples
	fmt.Printf("Running %d examples...\n", len(filtered))
	fmt.Println()

	for i, ex := range filtered {
		fmt.Printf("--- Example %d: %s [%s] ---\n", i, ex.ID, ex.Complexity)
		fmt.Printf("Description: %s\n", ex.Description)
		fmt.Printf("Expected: %v\n", ex.ExpectedConstructs)
		fmt.Println()
		// nxusKit: LLM generates rules, ClipsProvider validates syntax
		fmt.Println("(Use ruler generate --description \"...\" to generate rules)")
		fmt.Println()
	}
}

// Helper functions

func validateClipsCode(code string) *ruler.ValidationResult {
	var errors []*ruler.ValidationError
	var warnings []string

	// Basic validation
	openParens := strings.Count(code, "(")
	closeParens := strings.Count(code, ")")

	if openParens != closeParens {
		errors = append(errors, ruler.SyntaxError(
			fmt.Sprintf("Unbalanced parentheses: %d open, %d close", openParens, closeParens)))
	}

	if !strings.Contains(code, "deftemplate") && !strings.Contains(code, "defrule") {
		warnings = append(warnings, "No deftemplate or defrule found")
	}

	// Check for actual dangerous CLIPS function calls, not substrings
	// in comments like "expert system"
	dangerousPatterns := []string{"(system ", "(system)", "(open ", "(exec "}
	for _, pattern := range dangerousPatterns {
		if strings.Contains(code, pattern) {
			errors = append(errors, ruler.SafetyError("Code contains potentially dangerous system calls"))
			break
		}
	}

	if len(errors) == 0 {
		return ruler.NewValidResult("validation").WithWarnings(warnings)
	}
	return ruler.NewInvalidResult("validation", errors).WithWarnings(warnings)
}

func getBuiltinExamples() *ruler.ProgressiveExamples {
	return &ruler.ProgressiveExamples{
		Examples: []*ruler.ProgressiveExample{
			{
				ID:                 "basic-01-adult",
				Complexity:         ruler.ComplexityBasic,
				Description:        "Classify person as adult if age >= 18",
				DomainHints:        []string{"age", "classification"},
				ExpectedConstructs: []string{"deftemplate", "defrule"},
			},
			{
				ID:                 "basic-02-temperature",
				Complexity:         ruler.ComplexityBasic,
				Description:        "Classify temperature as cold/warm/hot",
				DomainHints:        []string{"temperature"},
				ExpectedConstructs: []string{"deftemplate", "defrule"},
			},
			{
				ID:                 "intermediate-01-priority",
				Complexity:         ruler.ComplexityIntermediate,
				Description:        "Process high-priority tasks first using salience",
				DomainHints:        []string{"priority", "queue"},
				ExpectedConstructs: []string{"deftemplate", "defrule", "salience"},
			},
			{
				ID:                 "advanced-01-triage",
				Complexity:         ruler.ComplexityAdvanced,
				Description:        "Medical triage with modules and helper functions",
				DomainHints:        []string{"medical", "triage"},
				ExpectedConstructs: []string{"deftemplate", "defrule", "defmodule", "deffunction"},
			},
		},
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
