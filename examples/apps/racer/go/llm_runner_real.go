package racer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// RealLLMRunner runs actual LLM inference to solve problems using nxuskit.
type RealLLMRunner struct {
	provider nxuskit.LLMProvider
	model    string
	timeout  time.Duration
}

// NewRealLLMRunner creates a new LLM runner with the given provider.
func NewRealLLMRunner(provider nxuskit.LLMProvider, model string) *RealLLMRunner {
	return &RealLLMRunner{
		provider: provider,
		model:    model,
		timeout:  time.Duration(DefaultTimeout) * time.Millisecond,
	}
}

// NewRealLLMRunnerWithFallback creates an LLM runner using the fallback provider.
func NewRealLLMRunnerWithFallback() (*RealLLMRunner, error) {
	fallback := nxuskit.NewProviderFallback()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := fallback.GetAvailableProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("no LLM provider available: %w", err)
	}

	return &RealLLMRunner{
		provider: provider,
		model:    "", // Use provider default
		timeout:  time.Duration(DefaultTimeout) * time.Millisecond,
	}, nil
}

// WithModel sets the model to use for LLM calls.
func (r *RealLLMRunner) WithModel(model string) *RealLLMRunner {
	r.model = model
	return r
}

// WithTimeout sets the timeout for LLM calls.
func (r *RealLLMRunner) WithTimeout(d time.Duration) *RealLLMRunner {
	r.timeout = d
	return r
}

// Run executes actual LLM inference on a problem.
func (r *RealLLMRunner) Run(ctx context.Context, problem *Problem) (*RunnerResult, error) {
	startTime := time.Now()

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Format the problem as a prompt
	prompt := formatProblemForLLM(problem)

	req := &nxuskit.ChatRequest{
		Model: r.model,
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage(getSystemPromptForProblem(problem)),
			nxuskit.UserMessage(prompt),
		},
		Temperature: floatPtr(0.0), // Deterministic for problem solving
	}

	resp, err := r.provider.Chat(timeoutCtx, req)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return NewTimeoutResult("llm-runner", problem.ID), nil
		}
		elapsed := time.Since(startTime).Milliseconds()
		return NewFailedResult("llm-runner", problem.ID, err.Error(), elapsed), nil
	}

	elapsed := time.Since(startTime).Milliseconds()

	// Parse the LLM response
	answer, correct, reasoning := parseLLMAnswer(resp.Content, problem)

	tokens := int64(resp.Usage.TotalTokens())
	result := &RunnerResult{
		RunnerID:   "llm-runner",
		ProblemID:  problem.ID,
		Answer:     answer,
		Correct:    correct,
		TimeMs:     elapsed,
		TimedOut:   false,
		TokensUsed: &tokens,
		Reasoning:  reasoning,
	}

	return result, nil
}

func getSystemPromptForProblem(problem *Problem) string {
	switch problem.Type {
	case ProblemTypeLogicPuzzle:
		return `You are an expert logic puzzle solver. Analyze the puzzle carefully using deductive reasoning.
Provide your reasoning step by step, then give your final answer in JSON format.`
	case ProblemTypeClassification:
		return `You are an expert classifier. Analyze the input and categorize it according to the given rules.
Provide clear reasoning, then give your answer in JSON format.`
	case ProblemTypeConstraintSatisfaction:
		return `You are an expert at solving constraint satisfaction problems. Apply systematic constraint propagation.
Show your work step by step, then provide the solution in JSON format.`
	default:
		return `You are a problem solver. Analyze the problem carefully and provide a solution in JSON format.`
	}
}

func formatProblemForLLM(problem *Problem) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Problem: %s\n\n", problem.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", problem.Description))

	if problem.InputData != nil {
		sb.WriteString("Input Data:\n")
		sb.WriteString(string(problem.InputData))
		sb.WriteString("\n\n")
	}

	// NOTE: LLMs are very flexible with the inputs we use compared to CLIPS,
	// but this often requires us to be explicit about the output format so
	// that results can be compared directly with CLIPS output.
	sb.WriteString("Return ONLY a flat JSON object with the answer. ")
	sb.WriteString("For example, if the answer is that the German owns the fish, return: ")
	sb.WriteString(`{"fish-owner": "German"}`)
	sb.WriteString("\nDo not nest the answer. Do not include explanations. ONLY the JSON object.")

	return sb.String()
}

func parseLLMAnswer(content string, problem *Problem) (json.RawMessage, bool, string) {
	// Extract JSON from the response
	jsonStr := extractJSONFromResponse(content)

	// Extract reasoning (everything before JSON)
	reasoning := content
	if idx := strings.Index(content, "{"); idx > 0 {
		reasoning = strings.TrimSpace(content[:idx])
	}

	if jsonStr == "" {
		return json.RawMessage(`{"error": "no JSON found"}`), false, reasoning
	}

	// Validate the answer against expected solution if available
	correct := validateAnswer([]byte(jsonStr), problem.ExpectedSolution)

	return json.RawMessage(jsonStr), correct, reasoning
}

func extractJSONFromResponse(content string) string {
	// Find the first { and match to closing }
	start := strings.Index(content, "{")
	if start == -1 {
		return ""
	}

	depth := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}

	return content[start:]
}

func validateAnswer(actual json.RawMessage, expected json.RawMessage) bool {
	if expected == nil {
		return false
	}

	// Parse both as generic maps for comparison
	var actualMap, expectedMap map[string]interface{}
	if err := json.Unmarshal(actual, &actualMap); err != nil {
		return false
	}
	if err := json.Unmarshal(expected, &expectedMap); err != nil {
		return false
	}

	// Simple key-value comparison for expected fields
	for key, expectedVal := range expectedMap {
		actualVal, exists := actualMap[key]
		if !exists {
			return false
		}
		// Compare string representations for simplicity
		if fmt.Sprintf("%v", expectedVal) != fmt.Sprintf("%v", actualVal) {
			return false
		}
	}

	return true
}

func floatPtr(f float64) *float64 {
	return &f
}
