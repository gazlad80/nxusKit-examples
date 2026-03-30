// Package riffer provides music sequence analysis and transformation.
package riffer

import (
	"context"
	"fmt"
	"os"
)

// MusicScore contains complete musicality scores for a sequence.
type MusicScore struct {
	Overall          float32           `json:"overall"`
	Dimensions       ScoreDimensions   `json:"dimensions"`
	Summary          ScoreSummary      `json:"summary"`
	Suggestions      []string          `json:"suggestions"`
	ClipsAdjustments []ClipsAdjustment `json:"clips_adjustments,omitempty"`
}

// ScoreDimensions contains individual dimension scores.
type ScoreDimensions struct {
	HarmonicCoherence  ScoreDimension `json:"harmonic_coherence"`
	MelodicInterest    ScoreDimension `json:"melodic_interest"`
	RhythmicVariety    ScoreDimension `json:"rhythmic_variety"`
	ResolutionQuality  ScoreDimension `json:"resolution_quality"`
	DynamicsExpression ScoreDimension `json:"dynamics_expression"`
	StructuralBalance  ScoreDimension `json:"structural_balance"`
}

// ScoreDimension represents a single dimension score.
type ScoreDimension struct {
	Score       float32 `json:"score"`
	Weight      float32 `json:"weight"`
	Rating      string  `json:"rating"`
	Explanation string  `json:"explanation"`
}

// ScoreSummary contains score summary.
type ScoreSummary struct {
	Rating    string `json:"rating"`
	Strongest string `json:"strongest"`
	Weakest   string `json:"weakest"`
	Summary   string `json:"summary"`
}

// ClipsAdjustment represents a CLIPS-based adjustment.
type ClipsAdjustment struct {
	Dimension  string  `json:"dimension"`
	Adjustment float32 `json:"adjustment"`
	Reason     string  `json:"reason"`
}

// ScoreSequence scores a sequence and returns comprehensive musicality scores.
// For CLIPS support, use ScoreSequenceWithClips instead.
func ScoreSequence(sequence *Sequence, useClips bool) (*MusicScore, error) {
	return scoreSequenceInternal(sequence, nil, useClips)
}

// ScoreSequenceWithClips scores a sequence with optional CLIPS rule engine support.
// This is the Go equivalent of Rust's score_sequence_async.
func ScoreSequenceWithClips(ctx context.Context, sequence *Sequence, rulesDir string) (*MusicScore, error) {
	var clipsAdjustments []ClipsAdjustment

	if rulesDir != "" {
		engine, err := NewClipsRuleEngine(rulesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create CLIPS engine: %v\n", err)
		} else if engine.IsAvailable() {
			result, err := engine.AnalyzeContext(ctx, sequence)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: CLIPS analysis failed: %v\n", err)
			} else if len(result.Adjustments) > 0 {
				// Convert ScoringAdjustment to ClipsAdjustment
				for _, adj := range result.Adjustments {
					clipsAdjustments = append(clipsAdjustments, ClipsAdjustment{
						Dimension:  adj.Dimension,
						Adjustment: adj.Adjustment,
						Reason:     adj.Reason,
					})
				}
			}
		}
	}

	return scoreSequenceInternal(sequence, clipsAdjustments, rulesDir != "")
}

// scoreSequenceInternal is the internal scoring logic shared by sync and CLIPS variants.
func scoreSequenceInternal(sequence *Sequence, clipsAdjustments []ClipsAdjustment, useClips bool) (*MusicScore, error) {
	// First, get the analysis
	analysis, err := AnalyzeSequence(sequence, true)
	if err != nil {
		return nil, err
	}

	// Calculate each dimension
	harmonic := scoreHarmonicCoherence(analysis)
	melodic := scoreMelodicInterest(analysis)
	rhythmic := scoreRhythmicVariety(analysis)
	resolution := scoreResolutionQuality(analysis, sequence.Notes)
	dynamics := scoreDynamicsExpression(analysis)
	structural := scoreStructuralBalance(analysis)

	// Apply CLIPS adjustments if available
	if len(clipsAdjustments) > 0 {
		for _, adj := range clipsAdjustments {
			// Map CLIPS dimension names to internal dimension references
			switch adj.Dimension {
			case "harmonic-coherence", "harmony":
				harmonic.Score = clamp(harmonic.Score+adj.Adjustment, 0, 100)
			case "melodic-interest", "melody":
				melodic.Score = clamp(melodic.Score+adj.Adjustment, 0, 100)
			case "rhythmic-variety", "rhythm":
				rhythmic.Score = clamp(rhythmic.Score+adj.Adjustment, 0, 100)
			case "resolution-quality", "resolution":
				resolution.Score = clamp(resolution.Score+adj.Adjustment, 0, 100)
			case "dynamics-expression", "dynamics":
				dynamics.Score = clamp(dynamics.Score+adj.Adjustment, 0, 100)
			case "structural-balance", "structure":
				structural.Score = clamp(structural.Score+adj.Adjustment, 0, 100)
			}
		}
	}

	dimensions := ScoreDimensions{
		HarmonicCoherence:  harmonic,
		MelodicInterest:    melodic,
		RhythmicVariety:    rhythmic,
		ResolutionQuality:  resolution,
		DynamicsExpression: dynamics,
		StructuralBalance:  structural,
	}

	// Calculate weighted overall score
	overall := calculateOverall(&dimensions)

	// Generate suggestions
	suggestions := generateSuggestions(&dimensions, analysis)

	// Get summary
	summary := generateSummary(&dimensions, overall)

	return &MusicScore{
		Overall:          overall,
		Dimensions:       dimensions,
		Summary:          summary,
		Suggestions:      suggestions,
		ClipsAdjustments: clipsAdjustments,
	}, nil
}

// clamp restricts a value to be within min and max bounds.
func clamp(value, min, max float32) float32 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// scoreHarmonicCoherence scores harmonic coherence.
func scoreHarmonicCoherence(analysis *AnalysisResult) ScoreDimension {
	coherence := analysis.ScaleAnalysis.CoherencePercentage

	var rating, explanation string
	if coherence >= 95.0 {
		rating = "excellent"
		explanation = "All or nearly all notes fit the detected key"
	} else if coherence >= 80.0 {
		rating = "good"
		explanation = "Most notes fit the key with some chromatic alterations"
	} else if coherence >= 60.0 {
		rating = "fair"
		explanation = "Moderate key adherence with significant alterations"
	} else {
		rating = "poor"
		explanation = "Many notes fall outside the detected key"
	}

	return ScoreDimension{
		Score:       coherence,
		Weight:      0.20,
		Rating:      rating,
		Explanation: explanation,
	}
}

// scoreMelodicInterest scores melodic interest.
func scoreMelodicInterest(analysis *AnalysisResult) ScoreDimension {
	intervalVariety := analysis.IntervalAnalysis.IntervalVariety
	contour := &analysis.ContourAnalysis

	// Base score on interval variety
	varietyScore := float32(intervalVariety) * 12.0
	if varietyScore > 50.0 {
		varietyScore = 50.0
	}

	// Bonus for interesting contour
	var contourBonus float32
	switch contour.ContourType {
	case ContourArch, ContourInverseArch:
		contourBonus = 25.0
	case ContourWave:
		contourBonus = 30.0
	case ContourAscending, ContourDescending:
		contourBonus = 15.0
	}

	// Bonus for direction changes
	var changeBonus float32
	if contour.DirectionChanges >= 1 && contour.DirectionChanges <= 4 {
		changeBonus = float32(contour.DirectionChanges) * 5.0
	} else if contour.DirectionChanges > 4 {
		changeBonus = 15.0
	}

	score := varietyScore + contourBonus + changeBonus
	if score > 100.0 {
		score = 100.0
	}

	var rating, explanation string
	if score >= 80.0 {
		rating = "excellent"
		explanation = "High interval variety with engaging contour"
	} else if score >= 60.0 {
		rating = "good"
		explanation = "Good melodic movement and variety"
	} else if score >= 40.0 {
		rating = "fair"
		explanation = "Moderate melodic interest"
	} else {
		rating = "poor"
		explanation = "Limited melodic variety or static contour"
	}

	return ScoreDimension{
		Score:       score,
		Weight:      0.20,
		Rating:      rating,
		Explanation: explanation,
	}
}

// scoreRhythmicVariety scores rhythmic variety.
func scoreRhythmicVariety(analysis *AnalysisResult) ScoreDimension {
	rhythm := &analysis.RhythmAnalysis

	// Base on unique durations
	uniqueScore := float32(rhythm.UniqueDurations) * 20.0
	if uniqueScore > 60.0 {
		uniqueScore = 60.0
	}

	// Duration variety bonus
	varietyBonus := rhythm.DurationVariety * 40.0

	score := uniqueScore + varietyBonus
	if score > 100.0 {
		score = 100.0
	}

	var rating, explanation string
	if score >= 70.0 {
		rating = "excellent"
		explanation = "Rich rhythmic variety with multiple note values"
	} else if score >= 50.0 {
		rating = "good"
		explanation = "Good rhythmic variety"
	} else if score >= 30.0 {
		rating = "fair"
		explanation = "Some rhythmic variation"
	} else {
		rating = "poor"
		explanation = "Uniform rhythm with little variety"
	}

	return ScoreDimension{
		Score:       score,
		Weight:      0.15,
		Rating:      rating,
		Explanation: explanation,
	}
}

// scoreResolutionQuality scores resolution quality.
func scoreResolutionQuality(analysis *AnalysisResult, notes []Note) ScoreDimension {
	if len(notes) < 2 {
		return ScoreDimension{
			Score:       50.0,
			Weight:      0.15,
			Rating:      "fair",
			Explanation: "Too few notes to assess resolution",
		}
	}

	intervals := AnalyzeIntervals(notes)
	var score float32 = 50.0 // Start at neutral

	// Check for resolution patterns
	lastNote := notes[len(notes)-1]
	secondLast := notes[len(notes)-2]

	// Leading tone resolution
	key := analysis.ScaleAnalysis.Key
	tonicPC := uint8(key.Root.ToSemitone())
	lastPC := uint8(lastNote.Pitch) % 12
	secondLastPC := uint8(secondLast.Pitch) % 12

	// Check if ends on tonic
	if lastPC == tonicPC {
		score += 20.0

		// Check for leading tone
		leadingTone := uint8((tonicPC + 11) % 12)
		if secondLastPC == leadingTone {
			score += 15.0
		}
	}

	// Check for consonant ending
	if len(intervals) > 0 {
		lastInterval := intervals[len(intervals)-1]
		switch lastInterval.Quality {
		case PerfectConsonance, ImperfectConsonance:
			score += 10.0
		case StrongDissonance:
			score -= 10.0
		}
	}

	// Clamp score
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	var rating, explanation string
	if score >= 80.0 {
		rating = "excellent"
		explanation = "Strong resolution with proper voice leading"
	} else if score >= 60.0 {
		rating = "good"
		explanation = "Good resolution tendencies"
	} else if score >= 40.0 {
		rating = "fair"
		explanation = "Adequate resolution"
	} else {
		rating = "poor"
		explanation = "Weak or unresolved ending"
	}

	return ScoreDimension{
		Score:       score,
		Weight:      0.15,
		Rating:      rating,
		Explanation: explanation,
	}
}

// scoreDynamicsExpression scores dynamics expression.
func scoreDynamicsExpression(analysis *AnalysisResult) ScoreDimension {
	dynamics := &analysis.DynamicsAnalysis

	var score float32 = 30.0 // Base score

	// Velocity range bonus
	if dynamics.HasDynamics {
		score += 30.0
	}

	// Velocity variance bonus
	varianceNorm := dynamics.VelocityVariance / 400.0
	if varianceNorm > 1.0 {
		varianceNorm = 1.0
	}
	score += varianceNorm * 30.0

	// Range bonus
	rangeBonus := float32(dynamics.VelocityRange) / 127.0 * 20.0
	score += rangeBonus

	if score > 100.0 {
		score = 100.0
	}

	var rating, explanation string
	if score >= 70.0 {
		rating = "excellent"
		explanation = "Expressive dynamics with good variation"
	} else if score >= 50.0 {
		rating = "good"
		explanation = "Noticeable dynamic variation"
	} else if score >= 30.0 {
		rating = "fair"
		explanation = "Some dynamic variation"
	} else {
		rating = "poor"
		explanation = "Flat dynamics with little expression"
	}

	return ScoreDimension{
		Score:       score,
		Weight:      0.15,
		Rating:      rating,
		Explanation: explanation,
	}
}

// scoreStructuralBalance scores structural balance.
func scoreStructuralBalance(analysis *AnalysisResult) ScoreDimension {
	contour := &analysis.ContourAnalysis

	var score float32 = 50.0

	// Pitch range assessment
	pitchRange := contour.PitchRange
	if pitchRange >= 5 && pitchRange <= 24 {
		score += 25.0
	} else if pitchRange >= 3 && pitchRange <= 36 {
		score += 15.0
	}

	// Climax position
	if contour.ClimaxPosition >= 0.4 && contour.ClimaxPosition <= 0.8 {
		score += 15.0
	}

	// Note count
	if analysis.NoteCount >= 4 && analysis.NoteCount <= 32 {
		score += 10.0
	}

	if score > 100.0 {
		score = 100.0
	}

	var rating, explanation string
	if score >= 75.0 {
		rating = "excellent"
		explanation = "Well-balanced structure with good proportions"
	} else if score >= 55.0 {
		rating = "good"
		explanation = "Good structural balance"
	} else if score >= 35.0 {
		rating = "fair"
		explanation = "Adequate structure"
	} else {
		rating = "poor"
		explanation = "Unbalanced structure"
	}

	return ScoreDimension{
		Score:       score,
		Weight:      0.15,
		Rating:      rating,
		Explanation: explanation,
	}
}

// calculateOverall calculates the weighted overall score.
func calculateOverall(dimensions *ScoreDimensions) float32 {
	weightedSum := dimensions.HarmonicCoherence.Score*dimensions.HarmonicCoherence.Weight +
		dimensions.MelodicInterest.Score*dimensions.MelodicInterest.Weight +
		dimensions.RhythmicVariety.Score*dimensions.RhythmicVariety.Weight +
		dimensions.ResolutionQuality.Score*dimensions.ResolutionQuality.Weight +
		dimensions.DynamicsExpression.Score*dimensions.DynamicsExpression.Weight +
		dimensions.StructuralBalance.Score*dimensions.StructuralBalance.Weight

	totalWeight := dimensions.HarmonicCoherence.Weight +
		dimensions.MelodicInterest.Weight +
		dimensions.RhythmicVariety.Weight +
		dimensions.ResolutionQuality.Weight +
		dimensions.DynamicsExpression.Weight +
		dimensions.StructuralBalance.Weight

	return weightedSum / totalWeight
}

// generateSuggestions generates improvement suggestions.
func generateSuggestions(dimensions *ScoreDimensions, analysis *AnalysisResult) []string {
	var suggestions []string

	// Harmonic suggestions
	if dimensions.HarmonicCoherence.Score < 70.0 {
		outOfScale := analysis.ScaleAnalysis.OutOfScaleCount
		suggestions = append(suggestions, fmt.Sprintf(
			"Consider reducing chromatic alterations (%d out-of-scale notes detected)",
			outOfScale,
		))
	}

	// Melodic suggestions
	if dimensions.MelodicInterest.Score < 50.0 {
		if analysis.IntervalAnalysis.IntervalVariety < 3 {
			suggestions = append(suggestions, "Add more interval variety for melodic interest")
		}
		if analysis.ContourAnalysis.DirectionChanges < 2 {
			suggestions = append(suggestions, "Consider adding direction changes to the melody")
		}
	}

	// Rhythmic suggestions
	if dimensions.RhythmicVariety.Score < 40.0 {
		suggestions = append(suggestions, "Add rhythmic variety with different note durations")
	}

	// Resolution suggestions
	if dimensions.ResolutionQuality.Score < 50.0 {
		suggestions = append(suggestions, "Consider ending on a stronger resolution (tonic or dominant)")
	}

	// Dynamics suggestions
	if dimensions.DynamicsExpression.Score < 40.0 {
		suggestions = append(suggestions, "Add dynamic variation (velocity changes) for expression")
	}

	// Structural suggestions
	if dimensions.StructuralBalance.Score < 40.0 {
		pitchRange := analysis.ContourAnalysis.PitchRange
		if pitchRange < 5 {
			suggestions = append(suggestions, "Expand the pitch range for better structural balance")
		} else if pitchRange > 24 {
			suggestions = append(suggestions, "Consider reducing the pitch range for better focus")
		}
	}

	return suggestions
}

// generateSummary generates a score summary.
func generateSummary(dimensions *ScoreDimensions, overall float32) ScoreSummary {
	dimScores := []struct {
		name  string
		score float32
	}{
		{"Harmonic Coherence", dimensions.HarmonicCoherence.Score},
		{"Melodic Interest", dimensions.MelodicInterest.Score},
		{"Rhythmic Variety", dimensions.RhythmicVariety.Score},
		{"Resolution Quality", dimensions.ResolutionQuality.Score},
		{"Dynamics Expression", dimensions.DynamicsExpression.Score},
		{"Structural Balance", dimensions.StructuralBalance.Score},
	}

	strongest := dimScores[0].name
	weakest := dimScores[0].name
	highestScore := dimScores[0].score
	lowestScore := dimScores[0].score

	for _, d := range dimScores[1:] {
		if d.score > highestScore {
			highestScore = d.score
			strongest = d.name
		}
		if d.score < lowestScore {
			lowestScore = d.score
			weakest = d.name
		}
	}

	var rating string
	if overall >= 80.0 {
		rating = "excellent"
	} else if overall >= 60.0 {
		rating = "good"
	} else if overall >= 40.0 {
		rating = "fair"
	} else {
		rating = "poor"
	}

	summary := fmt.Sprintf(
		"Overall %s musicality. Strongest: %s. Consider improving: %s.",
		rating, strongest, weakest,
	)

	return ScoreSummary{
		Rating:    rating,
		Strongest: strongest,
		Weakest:   weakest,
		Summary:   summary,
	}
}
