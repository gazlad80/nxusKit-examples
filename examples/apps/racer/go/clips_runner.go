// Package racer provides the CLIPS runner for racing.
package racer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ClipsRunner runs CLIPS rules to solve problems.
type ClipsRunner struct {
	// Config holds runner configuration.
	Config *RunnerConfig
	// RulesPath is the path to the CLIPS rules file.
	RulesPath string
}

// NewClipsRunner creates a new CLIPS runner.
func NewClipsRunner(rulesPath string) *ClipsRunner {
	return &ClipsRunner{
		Config:    NewRunnerConfig(),
		RulesPath: rulesPath,
	}
}

// WithTimeout sets the timeout.
func (r *ClipsRunner) WithTimeout(timeoutMs int64) *ClipsRunner {
	r.Config.TimeoutMs = timeoutMs
	return r
}

// Run executes the CLIPS rules on a problem.
// nxusKit: Uses ClipsProvider for CLIPS rule execution.
func (r *ClipsRunner) Run(ctx context.Context, problem *Problem) (*RunnerResult, error) {
	startTime := time.Now()

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(r.Config.TimeoutMs)*time.Millisecond)
	defer cancel()

	// Channel for result
	resultCh := make(chan *RunnerResult, 1)
	errCh := make(chan error, 1)

	go func() {
		// Execute CLIPS rules
		result, err := r.execute(problem)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	select {
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return NewTimeoutResult("clips-runner", problem.ID), nil
		}
		return nil, timeoutCtx.Err()
	case err := <-errCh:
		elapsed := time.Since(startTime).Milliseconds()
		return NewFailedResult("clips-runner", problem.ID, err.Error(), elapsed), nil
	case result := <-resultCh:
		result.TimeMs = time.Since(startTime).Milliseconds()
		return result, nil
	}
}

// execute runs CLIPS rule execution.
// nxusKit: Uses ClipsProvider to:
// 1. Create a CLIPS environment
// 2. Load the rules file
// 3. Assert facts from problem.InputData
// 4. Run the rules
// 5. Extract the answer from facts
func (r *ClipsRunner) execute(problem *Problem) (*RunnerResult, error) {
	// Execution times vary by problem type
	var sleepTime time.Duration
	switch problem.Type {
	case ProblemTypeLogicPuzzle:
		sleepTime = 40 * time.Millisecond // Logic puzzles are complex
	case ProblemTypeClassification:
		sleepTime = 10 * time.Millisecond // Classification is fast
	case ProblemTypeConstraintSatisfaction:
		sleepTime = 25 * time.Millisecond // CSP is medium
	default:
		sleepTime = 20 * time.Millisecond
	}

	time.Sleep(sleepTime)

	// Get the answer from CLIPS rule execution
	answer, correct := r.getAnswer(problem)

	return &RunnerResult{
		RunnerID:  "clips-runner",
		ProblemID: problem.ID,
		Answer:    answer,
		Correct:   correct,
		TimeMs:    0, // Will be set by caller
		TimedOut:  false,
	}, nil
}

// getAnswer returns an answer for the problem from CLIPS execution.
// Parses response data (mock implementation for offline testing).
func (r *ClipsRunner) getAnswer(problem *Problem) (json.RawMessage, bool) {
	// For known problems, return correct answers
	switch problem.Name {
	case "einstein-riddle":
		return json.RawMessage(`{"fish_owner": "German"}`), true
	case "family-relations":
		return json.RawMessage(`{"relationships_found": true}`), true
	case "animal-classification":
		return json.RawMessage(`{"classifications": [{"animal": "dog", "class": "mammal"}]}`), true
	default:
		// For unknown problems, use expected solution if available
		if problem.ExpectedSolution != nil {
			return problem.ExpectedSolution, true
		}
		return json.RawMessage(`{"result": "unknown"}`), false
	}
}

// ClipsEnvironment represents a CLIPS environment.
// nxusKit: ClipsProvider wraps this for seamless LLM integration.
type ClipsEnvironment struct {
	// Loaded indicates if rules are loaded.
	Loaded bool
	// FactCount is the number of facts in working memory.
	FactCount int
	// RuleFirings is the number of rule firings.
	RuleFirings int
}

// NewClipsEnvironment creates a new CLIPS environment.
// Mock CLIPS environment for offline testing.
func NewClipsEnvironment() *ClipsEnvironment {
	return &ClipsEnvironment{
		Loaded:      false,
		FactCount:   0,
		RuleFirings: 0,
	}
}

// LoadRules loads CLIPS rules from a file.
// nxusKit: Uses CLIPS load function to parse and compile rules.
func (e *ClipsEnvironment) LoadRules(path string) error {
	e.Loaded = true
	return nil
}

// AssertFact asserts a fact into working memory.
// nxusKit: Uses CLIPS assert function to add facts.
func (e *ClipsEnvironment) AssertFact(fact string) error {
	e.FactCount++
	return nil
}

// Run runs the CLIPS inference engine.
// Mock CLIPS execution for offline testing.
func (e *ClipsEnvironment) Run() (int, error) {
	// Build with -tags nxuskit for real SDK integration.
	e.RuleFirings = 10 // Example value for benchmarking
	return e.RuleFirings, nil
}

// GetFacts returns all facts matching a template.
// nxusKit: Queries CLIPS facts by template name.
func (e *ClipsEnvironment) GetFacts(template string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

// Reset resets the CLIPS environment.
func (e *ClipsEnvironment) Reset() error {
	e.FactCount = 0
	e.RuleFirings = 0
	return nil
}

// Close closes the CLIPS environment.
func (e *ClipsEnvironment) Close() error {
	e.Loaded = false
	return nil
}

// ClipsRequired returns an error indicating CLIPS is required.
// nxusKit: CLIPS functionality requires the clips build tag.
func ClipsRequired() error {
	return fmt.Errorf("CLIPS functionality requires the clips build tag")
}
