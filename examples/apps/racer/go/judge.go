// Package racer provides judging logic for CLIPS vs LLM races.
package racer

import (
	"bytes"
	"encoding/json"
)

// Judge evaluates race results and determines winners.
type Judge struct {
	// ScoringMode determines how winner is determined.
	ScoringMode ScoringMode
	// Weights for composite scoring.
	Weights *ScoringWeights
}

// NewJudge creates a new judge with default settings.
func NewJudge() *Judge {
	return &Judge{
		ScoringMode: ScoringModeSpeed,
		Weights:     DefaultScoringWeights(),
	}
}

// WithScoringMode sets the scoring mode.
func (j *Judge) WithScoringMode(mode ScoringMode) *Judge {
	j.ScoringMode = mode
	return j
}

// WithWeights sets the scoring weights.
func (j *Judge) WithWeights(weights *ScoringWeights) *Judge {
	j.Weights = weights
	return j
}

// DetermineWinner compares two results and returns the winner.
func (j *Judge) DetermineWinner(clips, llm *RunnerResult) RaceWinner {
	// Check for failures
	if !clips.Correct && !llm.Correct {
		return WinnerNone
	}
	if clips.Correct && !llm.Correct {
		return WinnerClips
	}
	if !clips.Correct && llm.Correct {
		return WinnerLLM
	}

	// Both correct - apply scoring mode
	switch j.ScoringMode {
	case ScoringModeSpeed:
		return j.judgeBySpeed(clips, llm)
	case ScoringModeAccuracy:
		return j.judgeByAccuracy(clips, llm)
	case ScoringModeComposite:
		return j.judgeByComposite(clips, llm)
	default:
		return j.judgeBySpeed(clips, llm)
	}
}

// judgeBySpeed determines winner by execution time.
func (j *Judge) judgeBySpeed(clips, llm *RunnerResult) RaceWinner {
	if clips.TimeMs < llm.TimeMs {
		return WinnerClips
	} else if llm.TimeMs < clips.TimeMs {
		return WinnerLLM
	}
	return WinnerTie
}

// judgeByAccuracy determines winner by answer completeness.
// Since both are correct, use answer size as proxy for completeness.
func (j *Judge) judgeByAccuracy(clips, llm *RunnerResult) RaceWinner {
	clipsSize := len(clips.Answer)
	llmSize := len(llm.Answer)

	if clipsSize > llmSize {
		return WinnerClips
	} else if llmSize > clipsSize {
		return WinnerLLM
	}
	// Tie on accuracy - use speed as tiebreaker
	return j.judgeBySpeed(clips, llm)
}

// judgeByComposite uses weighted combination of speed and accuracy.
func (j *Judge) judgeByComposite(clips, llm *RunnerResult) RaceWinner {
	clipsScore := j.calculateCompositeScore(clips)
	llmScore := j.calculateCompositeScore(llm)

	// Higher score wins
	if clipsScore > llmScore {
		return WinnerClips
	} else if llmScore > clipsScore {
		return WinnerLLM
	}
	return WinnerTie
}

// calculateCompositeScore calculates a weighted score for a result.
// Higher is better.
func (j *Judge) calculateCompositeScore(result *RunnerResult) float64 {
	// Correctness base score (0 or 1)
	correctnessScore := 0.0
	if result.Correct {
		correctnessScore = 1.0
	}

	// Speed score (inverse of time, normalized)
	// Lower time = higher score
	speedScore := 0.0
	if result.TimeMs > 0 {
		speedScore = 1.0 / float64(result.TimeMs) * 1000 // Normalize to reasonable range
	}

	// Weighted combination
	return (correctnessScore * j.Weights.AccuracyWeight) + (speedScore * j.Weights.TimeWeight)
}

// CompareAnswers checks if two answers are equivalent.
func CompareAnswers(a, b json.RawMessage) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Normalize JSON by unmarshaling and remarshaling
	var aObj, bObj interface{}
	if err := json.Unmarshal(a, &aObj); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bObj); err != nil {
		return false
	}

	aNorm, err := json.Marshal(aObj)
	if err != nil {
		return false
	}
	bNorm, err := json.Marshal(bObj)
	if err != nil {
		return false
	}

	return bytes.Equal(aNorm, bNorm)
}

// ValidateAnswer checks if an answer matches the expected solution.
func ValidateAnswer(answer, expected json.RawMessage) bool {
	return CompareAnswers(answer, expected)
}

// CalculateMargin returns the time difference between two results.
// Positive means CLIPS was faster, negative means LLM was faster.
func CalculateMargin(clips, llm *RunnerResult) int64 {
	return llm.TimeMs - clips.TimeMs
}

// JudgeRace evaluates a complete race and populates the winner.
func JudgeRace(clips, llm *RunnerResult, scoringMode ScoringMode) RaceWinner {
	judge := NewJudge().WithScoringMode(scoringMode)
	return judge.DetermineWinner(clips, llm)
}
