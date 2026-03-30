// Package racer provides statistics utility functions for benchmark analysis.
package racer

import (
	"math"
)

// Mean calculates the arithmetic mean of a slice of float64 values.
func Mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// StdDev calculates the sample standard deviation of a slice of float64 values.
func StdDev(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	m := Mean(values)
	var variance float64
	for _, v := range values {
		diff := v - m
		variance += diff * diff
	}
	variance /= float64(len(values) - 1)
	return math.Sqrt(variance)
}

// MinInt64 returns the minimum value from a slice of int64 values.
func MinInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// MaxInt64 returns the maximum value from a slice of int64 values.
func MaxInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// FromResults calculates statistics from a list of runner results.
func FromResults(results []*RunnerResult) *RunnerStats {
	if len(results) == 0 {
		return &RunnerStats{
			MeanTimeMs:   0,
			StdDevTimeMs: 0,
			MinTimeMs:    0,
			MaxTimeMs:    0,
			SuccessRate:  0,
			TimeoutRate:  0,
		}
	}

	times := make([]float64, len(results))
	timesInt := make([]int64, len(results))
	var successCount, timeoutCount int

	for i, r := range results {
		times[i] = float64(r.TimeMs)
		timesInt[i] = r.TimeMs
		if r.Correct {
			successCount++
		}
		if r.TimedOut {
			timeoutCount++
		}
	}

	total := float64(len(results))

	return &RunnerStats{
		MeanTimeMs:   Mean(times),
		StdDevTimeMs: StdDev(times),
		MinTimeMs:    MinInt64(timesInt),
		MaxTimeMs:    MaxInt64(timesInt),
		SuccessRate:  float64(successCount) / total,
		TimeoutRate:  float64(timeoutCount) / total,
	}
}

// FromRaces creates a benchmark report from race results.
func FromRaces(problemID string, races []*RaceResult) *BenchmarkReport {
	if len(races) == 0 {
		return nil
	}

	clipsResults := make([]*RunnerResult, len(races))
	llmResults := make([]*RunnerResult, len(races))
	var clipsWins, llmWins, ties int

	for i, r := range races {
		clipsResults[i] = r.ClipsResult
		llmResults[i] = r.LLMResult

		switch r.Winner {
		case WinnerClips:
			clipsWins++
		case WinnerLLM:
			llmWins++
		case WinnerTie:
			ties++
		}
	}

	total := float64(len(races))

	return &BenchmarkReport{
		ID:           "", // Will be set by caller
		ProblemID:    problemID,
		TotalRuns:    len(races),
		ClipsStats:   FromResults(clipsResults),
		LLMStats:     FromResults(llmResults),
		ClipsWinRate: float64(clipsWins) / total,
		LLMWinRate:   float64(llmWins) / total,
		TieRate:      float64(ties) / total,
	}
}
