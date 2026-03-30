// Package racer provides tests for CLIPS vs LLM racing.
package racer

import (
	"context"
	"encoding/json"
	"testing"
)

func TestProblemTypeParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected ProblemType
		wantErr  bool
	}{
		{"logic_puzzle", ProblemTypeLogicPuzzle, false},
		{"logic", ProblemTypeLogicPuzzle, false},
		{"classification", ProblemTypeClassification, false},
		{"classify", ProblemTypeClassification, false},
		{"constraint_satisfaction", ProblemTypeConstraintSatisfaction, false},
		{"csp", ProblemTypeConstraintSatisfaction, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseProblemType(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestScoringModeParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected ScoringMode
		wantErr  bool
	}{
		{"speed", ScoringModeSpeed, false},
		{"accuracy", ScoringModeAccuracy, false},
		{"composite", ScoringModeComposite, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseScoringMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestJudgeDetermineWinner(t *testing.T) {
	tests := []struct {
		name         string
		clipsCorrect bool
		clipsTime    int64
		llmCorrect   bool
		llmTime      int64
		scoringMode  ScoringMode
		expected     RaceWinner
	}{
		{
			name:         "both correct clips faster",
			clipsCorrect: true,
			clipsTime:    100,
			llmCorrect:   true,
			llmTime:      200,
			scoringMode:  ScoringModeSpeed,
			expected:     WinnerClips,
		},
		{
			name:         "both correct llm faster",
			clipsCorrect: true,
			clipsTime:    200,
			llmCorrect:   true,
			llmTime:      100,
			scoringMode:  ScoringModeSpeed,
			expected:     WinnerLLM,
		},
		{
			name:         "both correct same time",
			clipsCorrect: true,
			clipsTime:    100,
			llmCorrect:   true,
			llmTime:      100,
			scoringMode:  ScoringModeSpeed,
			expected:     WinnerTie,
		},
		{
			name:         "only clips correct",
			clipsCorrect: true,
			clipsTime:    100,
			llmCorrect:   false,
			llmTime:      50,
			scoringMode:  ScoringModeSpeed,
			expected:     WinnerClips,
		},
		{
			name:         "only llm correct",
			clipsCorrect: false,
			clipsTime:    50,
			llmCorrect:   true,
			llmTime:      100,
			scoringMode:  ScoringModeSpeed,
			expected:     WinnerLLM,
		},
		{
			name:         "neither correct",
			clipsCorrect: false,
			clipsTime:    100,
			llmCorrect:   false,
			llmTime:      100,
			scoringMode:  ScoringModeSpeed,
			expected:     WinnerNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			judge := NewJudge().WithScoringMode(tt.scoringMode)

			clipsResult := &RunnerResult{
				RunnerID:  "clips",
				ProblemID: "test",
				Correct:   tt.clipsCorrect,
				TimeMs:    tt.clipsTime,
			}
			llmResult := &RunnerResult{
				RunnerID:  "llm",
				ProblemID: "test",
				Correct:   tt.llmCorrect,
				TimeMs:    tt.llmTime,
			}

			winner := judge.DetermineWinner(clipsResult, llmResult)
			if winner != tt.expected {
				t.Errorf("got winner %v, want %v", winner, tt.expected)
			}
		})
	}
}

func TestJudgeRace(t *testing.T) {
	clipsResult := &RunnerResult{
		RunnerID:  "clips",
		ProblemID: "test",
		Correct:   true,
		TimeMs:    100,
	}
	llmResult := &RunnerResult{
		RunnerID:  "llm",
		ProblemID: "test",
		Correct:   true,
		TimeMs:    200,
	}

	winner := JudgeRace(clipsResult, llmResult, ScoringModeSpeed)
	if winner != WinnerClips {
		t.Errorf("got winner %v, want %v", winner, WinnerClips)
	}
}

func TestCompareAnswers(t *testing.T) {
	tests := []struct {
		name     string
		a        json.RawMessage
		b        json.RawMessage
		expected bool
	}{
		{
			name:     "equal objects",
			a:        json.RawMessage(`{"key": "value"}`),
			b:        json.RawMessage(`{"key": "value"}`),
			expected: true,
		},
		{
			name:     "different order same content",
			a:        json.RawMessage(`{"a": 1, "b": 2}`),
			b:        json.RawMessage(`{"b": 2, "a": 1}`),
			expected: true,
		},
		{
			name:     "different values",
			a:        json.RawMessage(`{"key": "value1"}`),
			b:        json.RawMessage(`{"key": "value2"}`),
			expected: false,
		},
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "one nil",
			a:        json.RawMessage(`{}`),
			b:        nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareAnswers(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("CompareAnswers() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateMargin(t *testing.T) {
	clips := &RunnerResult{TimeMs: 100}
	llm := &RunnerResult{TimeMs: 250}

	margin := CalculateMargin(clips, llm)
	if margin != 150 {
		t.Errorf("got margin %d, want 150", margin)
	}

	// LLM faster
	clips2 := &RunnerResult{TimeMs: 300}
	llm2 := &RunnerResult{TimeMs: 100}

	margin2 := CalculateMargin(clips2, llm2)
	if margin2 != -200 {
		t.Errorf("got margin %d, want -200", margin2)
	}
}

func TestRaceResult(t *testing.T) {
	clipsResult := NewSuccessResult("clips", "problem-1", json.RawMessage(`{"answer": 42}`), true, 100)
	llmResult := NewSuccessResult("llm", "problem-1", json.RawMessage(`{"answer": 42}`), true, 200)

	race := NewRaceResult("problem-1", clipsResult, llmResult, ScoringModeSpeed)

	if race.Winner != WinnerClips {
		t.Errorf("got winner %v, want %v", race.Winner, WinnerClips)
	}
	if race.MarginMs == nil || *race.MarginMs != 100 {
		t.Errorf("got margin %v, want 100", race.MarginMs)
	}
}

func TestRunnerStats(t *testing.T) {
	results := []*RunnerResult{
		{Correct: true, TimeMs: 100, TimedOut: false},
		{Correct: true, TimeMs: 200, TimedOut: false},
		{Correct: false, TimeMs: 150, TimedOut: false},
		{Correct: false, TimeMs: 0, TimedOut: true},
	}

	stats := FromResults(results)

	// Mean should be (100+200+150+0)/4 = 112.5
	if stats.MeanTimeMs < 112 || stats.MeanTimeMs > 113 {
		t.Errorf("got mean %f, want ~112.5", stats.MeanTimeMs)
	}
	if stats.MinTimeMs != 0 {
		t.Errorf("got min %d, want 0", stats.MinTimeMs)
	}
	if stats.MaxTimeMs != 200 {
		t.Errorf("got max %d, want 200", stats.MaxTimeMs)
	}
	if stats.SuccessRate != 0.5 {
		t.Errorf("got success rate %f, want 0.5", stats.SuccessRate)
	}
	if stats.TimeoutRate != 0.25 {
		t.Errorf("got timeout rate %f, want 0.25", stats.TimeoutRate)
	}
}

func TestStatisticsFunctions(t *testing.T) {
	values := []float64{10, 20, 30, 40, 50}

	m := Mean(values)
	if m != 30.0 {
		t.Errorf("Mean() = %f, want 30.0", m)
	}

	sd := StdDev(values)
	// Sample std dev of [10,20,30,40,50] is ~15.81
	if sd < 15.8 || sd > 15.9 {
		t.Errorf("StdDev() = %f, want ~15.81", sd)
	}

	min := MinInt64([]int64{5, 3, 8, 1, 9})
	if min != 1 {
		t.Errorf("MinInt64() = %d, want 1", min)
	}

	max := MaxInt64([]int64{5, 3, 8, 1, 9})
	if max != 9 {
		t.Errorf("MaxInt64() = %d, want 9", max)
	}
}

func TestProblemRegistry(t *testing.T) {
	registry := NewProblemRegistry()

	p1 := NewProblem("test-1", ProblemTypeLogicPuzzle, "Test problem 1").
		WithDifficulty(DifficultyEasy)
	p2 := NewProblem("test-2", ProblemTypeClassification, "Test problem 2").
		WithDifficulty(DifficultyHard)

	registry.Register(p1)
	registry.Register(p2)

	// Test Get
	found := registry.Get("test-1")
	if found == nil {
		t.Error("should find test-1")
	}
	if found.Name != "test-1" {
		t.Errorf("got name %q, want test-1", found.Name)
	}

	notFound := registry.Get("nonexistent")
	if notFound != nil {
		t.Error("should not find nonexistent")
	}

	// Test List
	names := registry.List()
	if len(names) != 2 {
		t.Errorf("got %d names, want 2", len(names))
	}

	// Test ByType
	logicProblems := registry.ByType(ProblemTypeLogicPuzzle)
	if len(logicProblems) != 1 {
		t.Errorf("got %d logic problems, want 1", len(logicProblems))
	}

	// Test ByDifficulty
	easyProblems := registry.ByDifficulty(DifficultyEasy)
	if len(easyProblems) != 1 {
		t.Errorf("got %d easy problems, want 1", len(easyProblems))
	}
}

func TestBuiltinProblems(t *testing.T) {
	registry := GetBuiltinProblemRegistry()

	einstein := registry.Get("einstein-riddle")
	if einstein == nil {
		t.Error("should have einstein-riddle problem")
	}
	if einstein.Difficulty != DifficultyHard {
		t.Errorf("einstein should be hard, got %v", einstein.Difficulty)
	}

	family := registry.Get("family-relations")
	if family == nil {
		t.Error("should have family-relations problem")
	}

	animal := registry.Get("animal-classification")
	if animal == nil {
		t.Error("should have animal-classification problem")
	}
	if animal.Type != ProblemTypeClassification {
		t.Errorf("animal should be classification, got %v", animal.Type)
	}
}

func TestRacer(t *testing.T) {
	racer := NewRacer("rules/test.clp", "claude-haiku-4-5-20251001").
		WithScoringMode(ScoringModeSpeed).
		WithTimeout(5000)

	if racer.ScoringMode != ScoringModeSpeed {
		t.Errorf("got scoring mode %v, want %v", racer.ScoringMode, ScoringModeSpeed)
	}
	if racer.TimeoutMs != 5000 {
		t.Errorf("got timeout %d, want 5000", racer.TimeoutMs)
	}
}

func TestRacerRace(t *testing.T) {
	racer := NewRacer("rules/test.clp", "test-model").
		WithTimeout(5000)

	problem := NewProblem("test-problem", ProblemTypeLogicPuzzle, "Test").
		WithSolution(json.RawMessage(`{"answer": 42}`))

	ctx := context.Background()
	result, err := racer.Race(ctx, problem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ClipsResult == nil {
		t.Error("clips result should not be nil")
	}
	if result.LLMResult == nil {
		t.Error("llm result should not be nil")
	}
	// Winner should be determined
	if result.Winner == "" {
		t.Error("winner should be set")
	}
}

func TestTimeoutResult(t *testing.T) {
	result := NewTimeoutResult("runner-1", "problem-1")

	if !result.TimedOut {
		t.Error("should be timed out")
	}
	if result.Correct {
		t.Error("timed out result should not be correct")
	}
	if result.Error == "" {
		t.Error("should have error message")
	}
}

func TestFailedResult(t *testing.T) {
	result := NewFailedResult("runner-1", "problem-1", "something went wrong", 500)

	if result.Correct {
		t.Error("failed result should not be correct")
	}
	if result.Error != "something went wrong" {
		t.Errorf("got error %q, want %q", result.Error, "something went wrong")
	}
	if result.TimeMs != 500 {
		t.Errorf("got time %d, want 500", result.TimeMs)
	}
}

func TestFindProblemByName(t *testing.T) {
	registry := GetBuiltinProblemRegistry()

	// Exact match
	problem, suggestions := FindProblemByName(registry, "einstein-riddle")
	if problem == nil {
		t.Error("should find exact match")
	}
	if len(suggestions) != 0 {
		t.Error("should have no suggestions for exact match")
	}

	// No match, should have suggestions
	problem2, suggestions2 := FindProblemByName(registry, "einstein")
	if problem2 != nil {
		t.Error("should not find partial match")
	}
	// May or may not have suggestions depending on similarity threshold
	_ = suggestions2
}
