package ruler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// LLMGenerator generates CLIPS rules using a real LLM provider.
type LLMGenerator struct {
	provider     nxuskit.LLMProvider
	model        string
	validator    *Validator
	promptConfig *PromptConfig
	maxRetries   int
	timeout      time.Duration
}

// NewLLMGenerator creates a new LLM-based generator with the given provider.
func NewLLMGenerator(provider nxuskit.LLMProvider, model string) *LLMGenerator {
	return &LLMGenerator{
		provider:     provider,
		model:        model,
		validator:    NewValidator(),
		promptConfig: DefaultPromptConfig(),
		maxRetries:   5,
		timeout:      30 * time.Second,
	}
}

// NewLLMGeneratorWithFallback creates an LLM generator using the best available
// provider. Prefers Claude (if ANTHROPIC_API_KEY is set), falls back to Ollama.
func NewLLMGeneratorWithFallback() (*LLMGenerator, error) {
	var provider nxuskit.LLMProvider
	var err error

	// Prefer Claude if API key is available
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		provider, err = nxuskit.NewClaudeProvider()
		if err == nil {
			return &LLMGenerator{
				provider:     provider,
				model:        "claude-haiku-4-5-20251001",
				validator:    NewValidator(),
				promptConfig: DefaultPromptConfig(),
				maxRetries:   5,
				timeout:      30 * time.Second,
			}, nil
		}
	}

	// Fall back to auto-detection (Ollama, etc.)
	fallback := nxuskit.NewProviderFallback()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err = fallback.GetAvailableProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("no LLM provider available: %w", err)
	}

	return &LLMGenerator{
		provider:     provider,
		model:        "llama3",
		validator:    NewValidator(),
		promptConfig: DefaultPromptConfig(),
		maxRetries:   5,
		timeout:      30 * time.Second,
	}, nil
}

// WithMaxRetries sets the maximum retries.
func (g *LLMGenerator) WithMaxRetries(retries int) *LLMGenerator {
	g.maxRetries = retries
	return g
}

// WithModel sets the model to use for generation.
func (g *LLMGenerator) WithModel(model string) *LLMGenerator {
	g.model = model
	return g
}

// WithTimeout sets the timeout per attempt.
func (g *LLMGenerator) WithTimeout(timeout time.Duration) *LLMGenerator {
	g.timeout = timeout
	return g
}

// Generate generates CLIPS rules from a description using the real LLM.
func (g *LLMGenerator) Generate(ctx context.Context, desc *RuleDescription) (*GenerateResult, error) {
	result := &GenerateResult{
		Success:  false,
		Attempts: 0,
		Errors:   make([]error, 0),
	}

	startTime := time.Now()
	var previousErrors []string

	for attempt := 1; attempt <= g.maxRetries; attempt++ {
		result.Attempts = attempt

		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Build prompt with any previous errors for feedback
		prompt := g.buildPrompt(desc, previousErrors)

		// Create timeout context for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, g.timeout)

		// Call LLM
		req := &nxuskit.ChatRequest{
			Model: g.model,
			Messages: []nxuskit.Message{
				nxuskit.SystemMessage(g.getSystemPrompt(desc.Complexity)),
				nxuskit.UserMessage(prompt),
			},
			Temperature: floatPtr(0.2), // Low temperature for code generation
		}

		resp, err := g.provider.Chat(attemptCtx, req)
		cancel()

		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("attempt %d: LLM call failed: %w", attempt, err))
			continue
		}

		// Extract CLIPS code from response
		code := extractClipsCode(resp.Content)
		tokens := int64(resp.Usage.TotalTokens())

		// Validate the generated code
		validation := g.validator.Validate(code)
		result.Validation = validation

		if validation.IsValid() {
			elapsed := time.Since(startTime)
			result.Success = true
			result.Rules = NewGeneratedRules(desc.ID, code, g.model).
				WithAttempt(attempt).
				WithTokens(tokens).
				WithTimeMs(elapsed.Milliseconds())
			return result, nil
		}

		// Collect errors for next attempt feedback
		for _, verr := range validation.Errors {
			previousErrors = append(previousErrors, verr.Error())
		}

		errMsg := "validation failed"
		if len(validation.Errors) > 0 {
			errMsg = validation.Errors[0].Error()
		}
		result.Errors = append(result.Errors, fmt.Errorf("attempt %d: %s", attempt, errMsg))
	}

	return result, fmt.Errorf("generation failed after %d attempts", g.maxRetries)
}

func (g *LLMGenerator) getSystemPrompt(complexity Complexity) string {
	var sb strings.Builder
	sb.WriteString("You are an expert CLIPS (C Language Integrated Production System) programmer.\n")
	sb.WriteString("Generate valid CLIPS code based on the user's description.\n\n")

	switch complexity {
	case ComplexityBasic:
		sb.WriteString("Target complexity: BASIC\n")
		sb.WriteString("Use only: deftemplate, defrule with simple patterns\n")
		sb.WriteString("Avoid: deffunction, defmodule, salience, test patterns\n")
	case ComplexityIntermediate:
		sb.WriteString("Target complexity: INTERMEDIATE\n")
		sb.WriteString("You may use: deftemplate, defrule, salience, test patterns, constraints\n")
		sb.WriteString("Avoid: deffunction, defmodule\n")
	case ComplexityAdvanced:
		sb.WriteString("Target complexity: ADVANCED\n")
		sb.WriteString("You may use all CLIPS constructs: deftemplate, defrule, deffunction, defmodule, salience, test patterns, constraints\n")
	}

	sb.WriteString("\nIMPORTANT RULES:\n")
	sb.WriteString("1. Generate ONLY valid CLIPS code - no explanations outside comments\n")
	sb.WriteString("2. Start with ;;; comments describing the rules\n")
	sb.WriteString("3. Use proper CLIPS syntax with balanced parentheses\n")
	sb.WriteString("4. Include appropriate deftemplates before rules that use them\n")
	sb.WriteString("5. Do NOT use system functions like 'system', 'open', 'read', 'exec'\n")
	sb.WriteString("6. Wrap all code in a single code block marked with ```clips\n")

	return sb.String()
}

func (g *LLMGenerator) buildPrompt(desc *RuleDescription, previousErrors []string) string {
	var sb strings.Builder

	sb.WriteString("Generate CLIPS rules for the following:\n\n")
	sb.WriteString(desc.Description)
	sb.WriteString("\n")

	if len(desc.DomainHints) > 0 {
		sb.WriteString("\nDomain hints: ")
		sb.WriteString(strings.Join(desc.DomainHints, ", "))
		sb.WriteString("\n")
	}

	if len(previousErrors) > 0 {
		sb.WriteString("\n\nPREVIOUS ATTEMPT HAD THESE ERRORS - please fix them:\n")
		for _, err := range previousErrors {
			sb.WriteString("- ")
			sb.WriteString(err)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func extractClipsCode(response string) string {
	// Look for code block
	codeStart := strings.Index(response, "```clips")
	if codeStart == -1 {
		codeStart = strings.Index(response, "```clp")
	}
	if codeStart == -1 {
		codeStart = strings.Index(response, "```")
	}

	if codeStart != -1 {
		// Find the start of actual code
		newlineAfterStart := strings.Index(response[codeStart:], "\n")
		if newlineAfterStart != -1 {
			codeStart = codeStart + newlineAfterStart + 1
		}

		// Find end of code block
		codeEnd := strings.Index(response[codeStart:], "```")
		if codeEnd != -1 {
			return strings.TrimSpace(response[codeStart : codeStart+codeEnd])
		}
		return strings.TrimSpace(response[codeStart:])
	}

	// If no code block, look for CLIPS patterns
	if strings.Contains(response, "(deftemplate") || strings.Contains(response, "(defrule") {
		// Return everything from first ( to end
		parenStart := strings.Index(response, "(")
		if parenStart != -1 {
			return strings.TrimSpace(response[parenStart:])
		}
	}

	return response
}

func floatPtr(f float64) *float64 {
	return &f
}
