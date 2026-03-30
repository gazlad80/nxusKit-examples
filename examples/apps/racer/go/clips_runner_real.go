//go:build nxuskit

package racer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// RealClipsRunner runs actual CLIPS rules to solve problems using nxuskit ClipsProvider.
type RealClipsRunner struct {
	provider nxuskit.LLMProvider
	rulesDir string
	timeout  time.Duration
}

// NewRealClipsRunner creates a new CLIPS runner with the given rules directory.
func NewRealClipsRunner(rulesDir string) (*RealClipsRunner, error) {
	provider, err := nxuskit.NewClipsFFIProvider(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIPS provider: %w", err)
	}

	return &RealClipsRunner{
		provider: provider,
		rulesDir: rulesDir,
		timeout:  time.Duration(DefaultTimeout) * time.Millisecond,
	}, nil
}

// WithTimeout sets the timeout for CLIPS execution.
func (r *RealClipsRunner) WithTimeout(d time.Duration) *RealClipsRunner {
	r.timeout = d
	return r
}

// Run executes actual CLIPS rules on a problem.
func (r *RealClipsRunner) Run(ctx context.Context, problem *Problem) (*RunnerResult, error) {
	startTime := time.Now()

	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Convert problem to CLIPS facts
	input := buildClipsInput(problem)

	inputJSON, err := json.Marshal(input)
	if err != nil {
		elapsed := time.Since(startTime).Milliseconds()
		return NewFailedResult("clips-runner", problem.ID, fmt.Sprintf("failed to marshal input: %v", err), elapsed), nil
	}

	req := &nxuskit.ChatRequest{
		Messages: []nxuskit.Message{
			nxuskit.UserMessage(string(inputJSON)),
		},
	}

	resp, err := r.provider.Chat(timeoutCtx, req)
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return NewTimeoutResult("clips-runner", problem.ID), nil
		}
		elapsed := time.Since(startTime).Milliseconds()
		return NewFailedResult("clips-runner", problem.ID, err.Error(), elapsed), nil
	}

	elapsed := time.Since(startTime).Milliseconds()

	// Parse CLIPS output
	var output clipsOutputWire
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		return NewFailedResult("clips-runner", problem.ID, fmt.Sprintf("failed to parse CLIPS output: %v", err), elapsed), nil
	}

	// Extract answer from conclusions
	answer, correct := extractClipsAnswer(output.Conclusions, problem)

	return &RunnerResult{
		RunnerID:  "clips-runner",
		ProblemID: problem.ID,
		Answer:    answer,
		Correct:   correct,
		TimeMs:    elapsed,
		TimedOut:  false,
	}, nil
}

func buildClipsInput(problem *Problem) clipsInputWire {
	var facts []clipsFactWire

	// Add problem metadata as facts
	facts = append(facts, clipsFactWire{
		Template: "problem",
		Values: map[string]interface{}{
			"id":   problem.ID,
			"name": problem.Name,
			"type": string(problem.Type),
		},
	})

	// Parse and add input data as facts if available
	if problem.InputData != nil {
		var inputData map[string]interface{}
		if err := json.Unmarshal(problem.InputData, &inputData); err == nil {
			for key, value := range inputData {
				facts = append(facts, clipsFactWire{
					Template: "input-data",
					Values: map[string]interface{}{
						"key":   key,
						"value": value,
					},
				})
			}
		}
	}

	inc, der := true, true
	maxR := int64(5000)
	return clipsInputWire{
		Facts: facts,
		Config: &clipsRequestConfigWire{
			IncludeTrace:   &inc,
			MaxRules:       &maxR,
			DerivedOnlyNew: &der,
		},
	}
}

func extractClipsAnswer(conclusions []clipsConclusionWire, problem *Problem) (json.RawMessage, bool) {
	// Look for solution/answer conclusions
	for _, c := range conclusions {
		if c.Template == "solution" || c.Template == "answer" || c.Template == "result" {
			answerJSON, err := json.Marshal(c.Values)
			if err != nil {
				continue
			}
			correct := validateAnswer(answerJSON, problem.ExpectedSolution)
			return answerJSON, correct
		}
	}

	// If no explicit answer found, return the last conclusion
	if len(conclusions) > 0 {
		last := conclusions[len(conclusions)-1]
		answerJSON, _ := json.Marshal(last.Values)
		return answerJSON, false
	}

	return json.RawMessage(`{"error": "no solution found"}`), false
}
