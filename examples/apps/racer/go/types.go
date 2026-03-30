// Package racer provides CLIPS vs LLM head-to-head competition.
//
// Racer runs CLIPS and LLM concurrently on logic problems, measures time,
// validates correctness, and determines the winner.
package racer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProblemType is the category of problem for racing.
type ProblemType string

const (
	// ProblemTypeLogicPuzzle is for logic puzzles (e.g., Einstein's riddle).
	ProblemTypeLogicPuzzle ProblemType = "logic_puzzle"
	// ProblemTypeClassification is for classification tasks.
	ProblemTypeClassification ProblemType = "classification"
	// ProblemTypeConstraintSatisfaction is for constraint satisfaction problems.
	ProblemTypeConstraintSatisfaction ProblemType = "constraint_satisfaction"
)

// String returns the string representation of the problem type.
func (p ProblemType) String() string {
	return string(p)
}

// ParseProblemType parses a string into a ProblemType value.
func ParseProblemType(s string) (ProblemType, error) {
	switch s {
	case "logic_puzzle", "logic":
		return ProblemTypeLogicPuzzle, nil
	case "classification", "classify":
		return ProblemTypeClassification, nil
	case "constraint_satisfaction", "constraint", "csp":
		return ProblemTypeConstraintSatisfaction, nil
	default:
		return "", fmt.Errorf("invalid problem type: '%s'. Valid: logic_puzzle, classification, constraint_satisfaction", s)
	}
}

// ProblemDifficulty represents problem difficulty level.
type ProblemDifficulty string

const (
	// DifficultyEasy is for easy problems.
	DifficultyEasy ProblemDifficulty = "easy"
	// DifficultyMedium is for medium problems.
	DifficultyMedium ProblemDifficulty = "medium"
	// DifficultyHard is for hard problems.
	DifficultyHard ProblemDifficulty = "hard"
)

// String returns the string representation of the difficulty.
func (d ProblemDifficulty) String() string {
	return string(d)
}

// ParseDifficulty parses a string into a ProblemDifficulty value.
func ParseDifficulty(s string) (ProblemDifficulty, error) {
	switch s {
	case "easy":
		return DifficultyEasy, nil
	case "medium":
		return DifficultyMedium, nil
	case "hard":
		return DifficultyHard, nil
	default:
		return "", fmt.Errorf("invalid difficulty: '%s'. Valid: easy, medium, hard", s)
	}
}

// Problem is a challenge for the Racer with known solution.
type Problem struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// Type is the problem category.
	Type ProblemType `json:"type"`
	// Description is the problem description.
	Description string `json:"description"`
	// InputData is the problem-specific input data.
	InputData json.RawMessage `json:"input_data"`
	// ExpectedSolution is the correct answer.
	ExpectedSolution json.RawMessage `json:"expected_solution"`
	// ClipsRulesPath is the path to CLIPS rules.
	ClipsRulesPath string `json:"clips_rules_path"`
	// Difficulty is the problem difficulty.
	Difficulty ProblemDifficulty `json:"difficulty"`
}

// NewProblem creates a new problem.
func NewProblem(name string, problemType ProblemType, description string) *Problem {
	return &Problem{
		ID:               uuid.New().String(),
		Name:             name,
		Type:             problemType,
		Description:      description,
		InputData:        nil,
		ExpectedSolution: nil,
		ClipsRulesPath:   "",
		Difficulty:       DifficultyMedium,
	}
}

// WithInput sets the input data.
func (p *Problem) WithInput(input json.RawMessage) *Problem {
	p.InputData = input
	return p
}

// WithSolution sets the expected solution.
func (p *Problem) WithSolution(solution json.RawMessage) *Problem {
	p.ExpectedSolution = solution
	return p
}

// WithRulesPath sets the CLIPS rules path.
func (p *Problem) WithRulesPath(path string) *Problem {
	p.ClipsRulesPath = path
	return p
}

// WithDifficulty sets the difficulty.
func (p *Problem) WithDifficulty(d ProblemDifficulty) *Problem {
	p.Difficulty = d
	return p
}

// RunnerType represents the type of runner approach.
type RunnerType string

const (
	// RunnerTypeClips is CLIPS rule-based solving.
	RunnerTypeClips RunnerType = "clips"
	// RunnerTypeLLM is LLM reasoning.
	RunnerTypeLLM RunnerType = "llm"
)

// String returns the string representation of the runner type.
func (r RunnerType) String() string {
	return string(r)
}

// RunnerConfig is the configuration for a runner.
type RunnerConfig struct {
	// TimeoutMs is the max execution time in milliseconds.
	TimeoutMs int64 `json:"timeout_ms"`
	// Model is the LLM model (for LLM runner).
	Model string `json:"model,omitempty"`
	// RulesPath is the CLIPS rules path (for CLIPS runner).
	RulesPath string `json:"rules_path,omitempty"`
}

// DefaultTimeout is the default timeout in milliseconds (60 seconds).
const DefaultTimeout int64 = 60_000

// NewRunnerConfig creates a new runner config with default timeout.
func NewRunnerConfig() *RunnerConfig {
	return &RunnerConfig{
		TimeoutMs: DefaultTimeout,
	}
}

// WithTimeout sets the timeout.
func (c *RunnerConfig) WithTimeout(timeoutMs int64) *RunnerConfig {
	c.TimeoutMs = timeoutMs
	return c
}

// WithModel sets the LLM model.
func (c *RunnerConfig) WithModel(model string) *RunnerConfig {
	c.Model = model
	return c
}

// WithRulesPath sets the CLIPS rules path.
func (c *RunnerConfig) WithRulesPath(path string) *RunnerConfig {
	c.RulesPath = path
	return c
}

// Runner is an approach that attempts to solve a problem.
type Runner struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// Type is the runner type.
	Type RunnerType `json:"type"`
	// Name is the display name.
	Name string `json:"name"`
	// Config is the runner configuration.
	Config *RunnerConfig `json:"config"`
}

// NewClipsRunnerConfig creates a CLIPS runner configuration.
func NewClipsRunnerConfig(name, rulesPath string) *Runner {
	return &Runner{
		ID:     uuid.New().String(),
		Type:   RunnerTypeClips,
		Name:   name,
		Config: NewRunnerConfig().WithRulesPath(rulesPath),
	}
}

// NewLLMRunnerConfig creates an LLM runner configuration.
func NewLLMRunnerConfig(name, model string) *Runner {
	return &Runner{
		ID:     uuid.New().String(),
		Type:   RunnerTypeLLM,
		Name:   name,
		Config: NewRunnerConfig().WithModel(model),
	}
}

// WithTimeout sets a custom timeout.
func (r *Runner) WithTimeout(timeoutMs int64) *Runner {
	r.Config.TimeoutMs = timeoutMs
	return r
}

// RunnerResult is the output from a single runner execution.
type RunnerResult struct {
	// RunnerID references the Runner.
	RunnerID string `json:"runner_id"`
	// ProblemID references the Problem.
	ProblemID string `json:"problem_id"`
	// Answer is the runner's answer.
	Answer json.RawMessage `json:"answer"`
	// Correct indicates whether answer matches expected.
	Correct bool `json:"correct"`
	// TimeMs is the execution time in milliseconds.
	TimeMs int64 `json:"time_ms"`
	// TimedOut indicates whether runner timed out.
	TimedOut bool `json:"timed_out"`
	// TokensUsed is the tokens consumed (LLM only).
	TokensUsed *int64 `json:"tokens_used,omitempty"`
	// Reasoning is intermediate reasoning steps.
	Reasoning string `json:"reasoning,omitempty"`
	// Error is the error message if failed.
	Error string `json:"error,omitempty"`
}

// NewSuccessResult creates a successful result.
func NewSuccessResult(runnerID, problemID string, answer json.RawMessage, correct bool, timeMs int64) *RunnerResult {
	return &RunnerResult{
		RunnerID:  runnerID,
		ProblemID: problemID,
		Answer:    answer,
		Correct:   correct,
		TimeMs:    timeMs,
		TimedOut:  false,
	}
}

// NewTimeoutResult creates a timeout result.
func NewTimeoutResult(runnerID, problemID string) *RunnerResult {
	return &RunnerResult{
		RunnerID:  runnerID,
		ProblemID: problemID,
		Answer:    nil,
		Correct:   false,
		TimeMs:    DefaultTimeout,
		TimedOut:  true,
		Error:     "Execution timed out",
	}
}

// NewFailedResult creates a failed result.
func NewFailedResult(runnerID, problemID, errorMsg string, timeMs int64) *RunnerResult {
	return &RunnerResult{
		RunnerID:  runnerID,
		ProblemID: problemID,
		Answer:    nil,
		Correct:   false,
		TimeMs:    timeMs,
		TimedOut:  false,
		Error:     errorMsg,
	}
}

// WithTokens sets the tokens used (for LLM runner).
func (r *RunnerResult) WithTokens(tokens int64) *RunnerResult {
	r.TokensUsed = &tokens
	return r
}

// WithReasoning sets the reasoning steps.
func (r *RunnerResult) WithReasoning(reasoning string) *RunnerResult {
	r.Reasoning = reasoning
	return r
}

// ScoringMode determines how race winner is determined.
type ScoringMode string

const (
	// ScoringModeSpeed means fastest correct answer wins.
	ScoringModeSpeed ScoringMode = "speed"
	// ScoringModeAccuracy means most complete answer wins.
	ScoringModeAccuracy ScoringMode = "accuracy"
	// ScoringModeComposite means weighted combination of speed and accuracy.
	ScoringModeComposite ScoringMode = "composite"
)

// String returns the string representation of the scoring mode.
func (s ScoringMode) String() string {
	return string(s)
}

// ParseScoringMode parses a string into a ScoringMode value.
func ParseScoringMode(str string) (ScoringMode, error) {
	switch str {
	case "speed":
		return ScoringModeSpeed, nil
	case "accuracy":
		return ScoringModeAccuracy, nil
	case "composite":
		return ScoringModeComposite, nil
	default:
		return "", fmt.Errorf("invalid scoring mode: '%s'. Valid: speed, accuracy, composite", str)
	}
}

// ScoringWeights holds weights for composite scoring.
type ScoringWeights struct {
	// TimeWeight is the weight for speed (0.0-1.0).
	TimeWeight float64 `json:"time_weight"`
	// AccuracyWeight is the weight for correctness (0.0-1.0).
	AccuracyWeight float64 `json:"accuracy_weight"`
}

// DefaultScoringWeights returns the default weights.
func DefaultScoringWeights() *ScoringWeights {
	return &ScoringWeights{
		TimeWeight:     0.5,
		AccuracyWeight: 0.5,
	}
}

// RaceWinner indicates the race winner.
type RaceWinner string

const (
	// WinnerClips means CLIPS won.
	WinnerClips RaceWinner = "clips"
	// WinnerLLM means LLM won.
	WinnerLLM RaceWinner = "llm"
	// WinnerTie means it was a tie.
	WinnerTie RaceWinner = "tie"
	// WinnerNone means no winner (both failed).
	WinnerNone RaceWinner = "none"
)

// String returns the string representation of the winner.
func (w RaceWinner) String() string {
	return string(w)
}

// RaceResult is the combined outcome of a head-to-head race.
type RaceResult struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// ProblemID references the Problem.
	ProblemID string `json:"problem_id"`
	// ClipsResult is the CLIPS runner outcome.
	ClipsResult *RunnerResult `json:"clips_result"`
	// LLMResult is the LLM runner outcome.
	LLMResult *RunnerResult `json:"llm_result"`
	// Winner is the race winner.
	Winner RaceWinner `json:"winner"`
	// ScoringMode is how winner was determined.
	ScoringMode ScoringMode `json:"scoring_mode"`
	// ScoredAt is when judged.
	ScoredAt time.Time `json:"scored_at"`
	// MarginMs is the margin of victory in milliseconds.
	MarginMs *int64 `json:"margin_ms,omitempty"`
}

// NewRaceResult creates a new race result and determines the winner.
func NewRaceResult(problemID string, clipsResult, llmResult *RunnerResult, scoringMode ScoringMode) *RaceResult {
	winner := determineWinner(clipsResult, llmResult, scoringMode)

	var marginMs *int64
	if clipsResult.Correct && llmResult.Correct {
		margin := llmResult.TimeMs - clipsResult.TimeMs
		marginMs = &margin
	}

	return &RaceResult{
		ID:          uuid.New().String(),
		ProblemID:   problemID,
		ClipsResult: clipsResult,
		LLMResult:   llmResult,
		Winner:      winner,
		ScoringMode: scoringMode,
		ScoredAt:    time.Now().UTC(),
		MarginMs:    marginMs,
	}
}

// determineWinner determines the winner based on scoring mode.
func determineWinner(clips, llm *RunnerResult, scoringMode ScoringMode) RaceWinner {
	if !clips.Correct && !llm.Correct {
		return WinnerNone
	}
	if clips.Correct && !llm.Correct {
		return WinnerClips
	}
	if !clips.Correct && llm.Correct {
		return WinnerLLM
	}

	// Both correct - determine by scoring mode
	switch scoringMode {
	case ScoringModeSpeed, ScoringModeComposite:
		if clips.TimeMs < llm.TimeMs {
			return WinnerClips
		} else if llm.TimeMs < clips.TimeMs {
			return WinnerLLM
		}
		return WinnerTie
	case ScoringModeAccuracy:
		// For accuracy mode with both correct, compare by completeness
		// Here we just use time as tiebreaker since both are correct
		if clips.TimeMs < llm.TimeMs {
			return WinnerClips
		} else if llm.TimeMs < clips.TimeMs {
			return WinnerLLM
		}
		return WinnerTie
	default:
		return WinnerTie
	}
}

// RunnerStats holds aggregate statistics for a runner across benchmark runs.
type RunnerStats struct {
	// MeanTimeMs is the average execution time.
	MeanTimeMs float64 `json:"mean_time_ms"`
	// StdDevTimeMs is the standard deviation of time.
	StdDevTimeMs float64 `json:"std_dev_time_ms"`
	// MinTimeMs is the fastest run.
	MinTimeMs int64 `json:"min_time_ms"`
	// MaxTimeMs is the slowest run.
	MaxTimeMs int64 `json:"max_time_ms"`
	// SuccessRate is the percentage of successful runs.
	SuccessRate float64 `json:"success_rate"`
	// TimeoutRate is the percentage of timeouts.
	TimeoutRate float64 `json:"timeout_rate"`
}

// BenchmarkReport holds aggregate statistics from multiple race runs.
type BenchmarkReport struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// ProblemID references the Problem.
	ProblemID string `json:"problem_id"`
	// TotalRuns is the number of iterations.
	TotalRuns int `json:"total_runs"`
	// ClipsStats is the CLIPS aggregate stats.
	ClipsStats *RunnerStats `json:"clips_stats"`
	// LLMStats is the LLM aggregate stats.
	LLMStats *RunnerStats `json:"llm_stats"`
	// ClipsWinRate is the CLIPS win percentage.
	ClipsWinRate float64 `json:"clips_win_rate"`
	// LLMWinRate is the LLM win percentage.
	LLMWinRate float64 `json:"llm_win_rate"`
	// TieRate is the tie percentage.
	TieRate float64 `json:"tie_rate"`
	// CreatedAt is the report generation time.
	CreatedAt time.Time `json:"created_at"`
}

// ProblemRegistry is a built-in problem registry.
type ProblemRegistry struct {
	problems map[string]*Problem
}

// NewProblemRegistry creates a new empty registry.
func NewProblemRegistry() *ProblemRegistry {
	return &ProblemRegistry{
		problems: make(map[string]*Problem),
	}
}

// Register adds a problem to the registry.
func (r *ProblemRegistry) Register(problem *Problem) {
	r.problems[problem.Name] = problem
}

// Get retrieves a problem by name.
func (r *ProblemRegistry) Get(name string) *Problem {
	return r.problems[name]
}

// List returns all problem names.
func (r *ProblemRegistry) List() []string {
	names := make([]string, 0, len(r.problems))
	for name := range r.problems {
		names = append(names, name)
	}
	return names
}

// ByType filters problems by type.
func (r *ProblemRegistry) ByType(problemType ProblemType) []*Problem {
	var result []*Problem
	for _, p := range r.problems {
		if p.Type == problemType {
			result = append(result, p)
		}
	}
	return result
}

// ByDifficulty filters problems by difficulty.
func (r *ProblemRegistry) ByDifficulty(difficulty ProblemDifficulty) []*Problem {
	var result []*Problem
	for _, p := range r.problems {
		if p.Difficulty == difficulty {
			result = append(result, p)
		}
	}
	return result
}
