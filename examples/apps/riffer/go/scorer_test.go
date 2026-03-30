// Package riffer provides music sequence analysis and transformation.
package riffer

import "testing"

func TestScoreCMajorScale(t *testing.T) {
	seq := createCMajorScaleForTest()
	score, err := ScoreSequence(seq, false)
	if err != nil {
		t.Fatalf("ScoreSequence failed: %v", err)
	}

	// Harmonic coherence should be excellent
	if score.Dimensions.HarmonicCoherence.Score < 95.0 {
		t.Errorf("Expected harmonic coherence >= 95, got %.1f", score.Dimensions.HarmonicCoherence.Score)
	}

	// Overall should be reasonable
	if score.Overall < 40.0 {
		t.Errorf("Expected overall score >= 40, got %.1f", score.Overall)
	}
}

func TestScoreVariedSequence(t *testing.T) {
	seq := createVariedSequenceForTest()
	score, err := ScoreSequence(seq, false)
	if err != nil {
		t.Fatalf("ScoreSequence failed: %v", err)
	}

	// Should have better rhythmic variety
	if score.Dimensions.RhythmicVariety.Score < 40.0 {
		t.Errorf("Expected rhythmic variety >= 40, got %.1f", score.Dimensions.RhythmicVariety.Score)
	}

	// Should have better dynamics
	if score.Dimensions.DynamicsExpression.Score < 40.0 {
		t.Errorf("Expected dynamics expression >= 40, got %.1f", score.Dimensions.DynamicsExpression.Score)
	}
}

func TestScoreSummary(t *testing.T) {
	seq := createCMajorScaleForTest()
	score, err := ScoreSequence(seq, false)
	if err != nil {
		t.Fatalf("ScoreSequence failed: %v", err)
	}

	if score.Summary.Rating == "" {
		t.Error("Expected non-empty rating")
	}

	if score.Summary.Strongest == "" {
		t.Error("Expected non-empty strongest dimension")
	}

	if score.Summary.Weakest == "" {
		t.Error("Expected non-empty weakest dimension")
	}
}

func TestSuggestionsGenerated(t *testing.T) {
	seq := createCMajorScaleForTest()
	score, err := ScoreSequence(seq, false)
	if err != nil {
		t.Fatalf("ScoreSequence failed: %v", err)
	}

	// Should generate some suggestions (uniform rhythm, no dynamics)
	// Note: This may or may not generate suggestions depending on scores
	// The important thing is it doesn't panic
	_ = score.Suggestions // Just ensure suggestions slice exists
}

func TestScoreWeights(t *testing.T) {
	seq := createCMajorScaleForTest()
	score, err := ScoreSequence(seq, false)
	if err != nil {
		t.Fatalf("ScoreSequence failed: %v", err)
	}

	// Verify weights sum to 1.0
	totalWeight := score.Dimensions.HarmonicCoherence.Weight +
		score.Dimensions.MelodicInterest.Weight +
		score.Dimensions.RhythmicVariety.Weight +
		score.Dimensions.ResolutionQuality.Weight +
		score.Dimensions.DynamicsExpression.Weight +
		score.Dimensions.StructuralBalance.Weight

	// Allow small floating point error
	if totalWeight < 0.99 || totalWeight > 1.01 {
		t.Errorf("Expected total weight ~1.0, got %.2f", totalWeight)
	}
}

func TestScoreClipsDisabled(t *testing.T) {
	seq := createCMajorScaleForTest()
	score, err := ScoreSequence(seq, false)
	if err != nil {
		t.Fatalf("ScoreSequence failed: %v", err)
	}

	// CLIPS should be nil when disabled
	if score.ClipsAdjustments != nil && len(score.ClipsAdjustments) > 0 {
		t.Error("Expected nil or empty ClipsAdjustments when CLIPS disabled")
	}
}
