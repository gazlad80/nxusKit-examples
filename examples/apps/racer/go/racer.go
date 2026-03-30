// Package racer provides concurrent race execution for CLIPS vs LLM.
package racer

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
)

// Racer orchestrates head-to-head races between CLIPS and LLM.
type Racer struct {
	// ClipsRunner is the CLIPS-based solver.
	ClipsRunner *ClipsRunner
	// LLMRunner is the LLM-based solver.
	LLMRunner *LLMRunner
	// ScoringMode determines how winner is determined.
	ScoringMode ScoringMode
	// TimeoutMs is the max execution time per runner.
	TimeoutMs int64
}

// NewRacer creates a new racer with default settings.
func NewRacer(clipsRulesPath, llmModel string) *Racer {
	return &Racer{
		ClipsRunner: NewClipsRunner(clipsRulesPath),
		LLMRunner:   NewLLMRunner(llmModel),
		ScoringMode: ScoringModeSpeed,
		TimeoutMs:   DefaultTimeout,
	}
}

// WithScoringMode sets the scoring mode.
func (r *Racer) WithScoringMode(mode ScoringMode) *Racer {
	r.ScoringMode = mode
	return r
}

// WithTimeout sets the timeout in milliseconds.
func (r *Racer) WithTimeout(timeoutMs int64) *Racer {
	r.TimeoutMs = timeoutMs
	r.ClipsRunner.Config.TimeoutMs = timeoutMs
	r.LLMRunner.Config.TimeoutMs = timeoutMs
	return r
}

// Race runs CLIPS and LLM concurrently on a problem.
func (r *Racer) Race(ctx context.Context, problem *Problem) (*RaceResult, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(r.TimeoutMs)*time.Millisecond)
	defer cancel()

	// Results channels
	var clipsResult, llmResult *RunnerResult
	var clipsErr, llmErr error

	// Run both concurrently using errgroup
	g, gCtx := errgroup.WithContext(timeoutCtx)

	// CLIPS runner
	g.Go(func() error {
		result, err := r.ClipsRunner.Run(gCtx, problem)
		clipsResult = result
		clipsErr = err
		return nil // Don't fail the group on runner errors
	})

	// LLM runner
	g.Go(func() error {
		result, err := r.LLMRunner.Run(gCtx, problem)
		llmResult = result
		llmErr = err
		return nil // Don't fail the group on runner errors
	})

	// Wait for both to complete
	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("race execution failed: %w", err)
	}

	// Handle errors - create failed results if needed
	if clipsErr != nil && clipsResult == nil {
		clipsResult = NewFailedResult("clips-runner", problem.ID, clipsErr.Error(), 0)
	}
	if llmErr != nil && llmResult == nil {
		llmResult = NewFailedResult("llm-runner", problem.ID, llmErr.Error(), 0)
	}

	// Create race result
	return NewRaceResult(problem.ID, clipsResult, llmResult, r.ScoringMode), nil
}

// Benchmark runs multiple races and calculates statistics.
func (r *Racer) Benchmark(ctx context.Context, problem *Problem, runs int) (*BenchmarkReport, error) {
	if runs < 1 {
		runs = 1
	}

	var races []*RaceResult
	for i := 0; i < runs; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result, err := r.Race(ctx, problem)
		if err != nil {
			return nil, fmt.Errorf("race %d failed: %w", i+1, err)
		}
		races = append(races, result)
	}

	return FromRaces(problem.ID, races), nil
}

// RaceConfig holds configuration for a race session.
type RaceConfig struct {
	// Problem is the problem to solve.
	Problem *Problem
	// ClipsRulesPath is the path to CLIPS rules.
	ClipsRulesPath string
	// LLMModel is the LLM model to use.
	LLMModel string
	// ScoringMode determines how winner is determined.
	ScoringMode ScoringMode
	// TimeoutMs is the max execution time.
	TimeoutMs int64
	// Runs is the number of benchmark iterations.
	Runs int
}

// DefaultRaceConfig returns the default race configuration.
func DefaultRaceConfig() *RaceConfig {
	return &RaceConfig{
		ScoringMode: ScoringModeSpeed,
		TimeoutMs:   DefaultTimeout,
		Runs:        1,
	}
}

// RunRace executes a race with the given configuration.
func RunRace(ctx context.Context, config *RaceConfig) (*RaceResult, error) {
	racer := NewRacer(config.ClipsRulesPath, config.LLMModel).
		WithScoringMode(config.ScoringMode).
		WithTimeout(config.TimeoutMs)

	return racer.Race(ctx, config.Problem)
}

// RunBenchmark executes multiple races and returns statistics.
func RunBenchmark(ctx context.Context, config *RaceConfig) (*BenchmarkReport, error) {
	racer := NewRacer(config.ClipsRulesPath, config.LLMModel).
		WithScoringMode(config.ScoringMode).
		WithTimeout(config.TimeoutMs)

	return racer.Benchmark(ctx, config.Problem, config.Runs)
}
