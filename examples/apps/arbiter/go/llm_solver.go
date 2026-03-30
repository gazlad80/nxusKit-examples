package arbiter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// LLMSolver wraps an LLM provider for the solver's retry loop.
type LLMSolver struct {
	provider nxuskit.LLMProvider
	model    string
}

// NewLLMSolver creates a new LLM solver with the given provider.
func NewLLMSolver(provider nxuskit.LLMProvider, model string) *LLMSolver {
	return &LLMSolver{
		provider: provider,
		model:    model,
	}
}

// NewLLMSolverWithFallback creates an LLM solver using the best available
// provider. Prefers Claude (if ANTHROPIC_API_KEY is set), falls back to Ollama.
func NewLLMSolverWithFallback() (*LLMSolver, error) {
	// Prefer Claude if API key is available
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		provider, err := nxuskit.NewClaudeProvider()
		if err == nil {
			return &LLMSolver{
				provider: provider,
				model:    "claude-haiku-4-5-20251001",
			}, nil
		}
	}

	// Fall back to auto-detection (Ollama, etc.)
	fallback := nxuskit.NewProviderFallback()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := fallback.GetAvailableProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("no LLM provider available: %w", err)
	}

	return &LLMSolver{
		provider: provider,
		model:    "llama3",
	}, nil
}

// GenerateResponse calls the LLM with the given input and parameters.
func (s *LLMSolver) GenerateResponse(ctx context.Context, input string, systemPrompt string, params map[string]any) (*LLMResponse, error) {
	temp := 0.7
	if v, ok := params["temperature"].(float64); ok {
		temp = v
	}

	maxTokens := 1000
	if v, ok := params["max_tokens"].(float64); ok {
		maxTokens = int(v)
	}

	req := &nxuskit.ChatRequest{
		Model: s.model,
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage(systemPrompt),
			nxuskit.UserMessage(input),
		},
		Temperature: floatPtr(temp),
		MaxTokens:   &maxTokens,
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return &LLMResponse{
		Content:    resp.Content,
		TokensUsed: int64(resp.Usage.TotalTokens()),
	}, nil
}

// LLMResponse holds the response from an LLM call.
type LLMResponse struct {
	Content    string
	TokensUsed int64
}

// Solver is a solver that uses real LLM and CLIPS providers.
type Solver struct {
	Config     SolverConfig
	Strategies []FailureStrategy
	llmSolver  *LLMSolver
	validator  ClipsValidator
}

// ClipsValidator validates LLM responses using CLIPS rules.
type ClipsValidator interface {
	Validate(ctx context.Context, llmResponse string, config *SolverConfig) (*EvaluationResult, error)
}

// NewSolver creates a solver with real LLM integration.
func NewSolver(config SolverConfig, llmSolver *LLMSolver, validator ClipsValidator) *Solver {
	strategies := config.Strategies
	if len(strategies) == 0 {
		strategies = DefaultStrategies()
	}

	return &Solver{
		Config:     config,
		Strategies: strategies,
		llmSolver:  llmSolver,
		validator:  validator,
	}
}

// Run executes the solver loop with real LLM calls and CLIPS validation.
func (s *Solver) Run(ctx context.Context, input string, verbose bool) (*SolverResult, error) {
	if err := s.Config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	startTime := time.Now()
	var retryHistory []RetryAttempt
	currentParams := s.initializeParams()

	systemPrompt := s.buildSystemPrompt()

	for attemptNum := 1; attemptNum <= s.Config.MaxRetries; attemptNum++ {
		attemptStart := time.Now()

		if verbose {
			temp := 0.7
			if v, ok := currentParams["temperature"].(float64); ok {
				temp = v
			}
			fmt.Printf("Attempt %d:\n", attemptNum)
			fmt.Printf("  Parameters: temperature=%.1f\n", temp)
		}

		// Call real LLM
		llmResponse, err := s.llmSolver.GenerateResponse(ctx, input, systemPrompt, currentParams)
		if err != nil {
			if verbose {
				fmt.Printf("  LLM Error: %v\n", err)
			}
			continue
		}

		if verbose {
			fmt.Printf("  LLM Response: %s\n", truncateSolver(llmResponse.Content, 80))
		}

		// Validate with CLIPS
		evaluation, err := s.validator.Validate(ctx, llmResponse.Content, &s.Config)
		if err != nil {
			if verbose {
				fmt.Printf("  Validation error: %v\n", err)
			}
			continue
		}

		if verbose {
			fmt.Printf("  Evaluation: %s\n", evaluation.Status)
			if evaluation.FailureType != nil {
				fmt.Printf("  Failure: %s\n", *evaluation.FailureType)
			}
		}

		attempt := RetryAttempt{
			AttemptNumber: attemptNum,
			Parameters:    copyParamsSolver(currentParams),
			LLMResponse:   llmResponse.Content,
			Evaluation:    *evaluation,
			DurationMS:    time.Since(attemptStart).Milliseconds(),
			TokensUsed:    llmResponse.TokensUsed,
		}

		retryHistory = append(retryHistory, attempt)

		// Check if valid
		if evaluation.Status == Valid {
			totalTokens := sumTokensSolver(retryHistory)
			var finalOutput json.RawMessage
			finalOutput = []byte(llmResponse.Content)

			return &SolverResult{
				Success:         true,
				FinalOutput:     finalOutput,
				BestAttempt:     attempt,
				RetryHistory:    retryHistory,
				TotalDurationMS: time.Since(startTime).Milliseconds(),
				TotalTokens:     totalTokens,
			}, nil
		}

		// Apply adjustments for retry
		if evaluation.FailureType != nil {
			if strategy := s.findStrategy(*evaluation.FailureType); strategy != nil {
				ApplyAdjustments(currentParams, strategy)
				if verbose {
					knobs := make([]string, 0, len(strategy.Adjustments))
					for _, adj := range strategy.Adjustments {
						knobs = append(knobs, adj.Knob)
					}
					fmt.Printf("  Adjustment: %v\n", knobs)
				}
			}
		}

		if verbose {
			fmt.Println()
		}
	}

	// Max retries exceeded - return best attempt
	best := findBestAttemptSolver(retryHistory)
	totalTokens := sumTokensSolver(retryHistory)
	var finalOutput json.RawMessage
	finalOutput = []byte(best.LLMResponse)

	return &SolverResult{
		Success:         false,
		FinalOutput:     finalOutput,
		BestAttempt:     best,
		RetryHistory:    retryHistory,
		TotalDurationMS: time.Since(startTime).Milliseconds(),
		TotalTokens:     totalTokens,
	}, nil
}

func (s *Solver) initializeParams() map[string]any {
	return map[string]any{
		"temperature":      0.7,
		"max_tokens":       1000.0,
		"thinking_enabled": 0.0,
	}
}

func (s *Solver) findStrategy(failureType FailureType) *FailureStrategy {
	for i := range s.Strategies {
		if s.Strategies[i].FailureType == failureType {
			return &s.Strategies[i]
		}
	}
	return nil
}

func (s *Solver) buildSystemPrompt() string {
	var sb strings.Builder
	sb.WriteString("You are an AI assistant that classifies and analyzes input.\n")
	sb.WriteString("Provide your response as a JSON object with the following structure:\n")
	sb.WriteString(`{"category": "<category>", "confidence": <0.0-1.0>, "reasoning": "<explanation>"}`)
	sb.WriteString("\n\n")

	if len(s.Config.ValidCategories) > 0 {
		sb.WriteString("Valid categories: ")
		sb.WriteString(strings.Join(s.Config.ValidCategories, ", "))
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("Minimum confidence threshold: %.1f\n", s.Config.ConfidenceThreshold))

	return sb.String()
}

func floatPtr(f float64) *float64 {
	return &f
}

func truncateSolver(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func copyParamsSolver(params map[string]any) map[string]any {
	cp := make(map[string]any)
	for k, v := range params {
		cp[k] = v
	}
	return cp
}

func sumTokensSolver(history []RetryAttempt) int64 {
	var total int64
	for _, attempt := range history {
		total += attempt.TokensUsed
	}
	return total
}

func findBestAttemptSolver(history []RetryAttempt) RetryAttempt {
	if len(history) == 0 {
		return RetryAttempt{}
	}

	best := history[0]
	bestScore := ScoreAttempt(&best)

	for i := 1; i < len(history); i++ {
		score := ScoreAttempt(&history[i])
		if score > bestScore {
			best = history[i]
			bestScore = score
		}
	}

	return best
}
