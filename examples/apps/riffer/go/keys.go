// Package riffer provides music sequence analysis and transformation.
//
// Key detection using Krumhansl-Schmuckler algorithm.
// Detects the most likely key of a musical sequence using pitch-class distribution.
package riffer

import "math"

// Key profiles from Krumhansl & Kessler (1982)
var majorProfile = [12]float32{
	6.35, 2.23, 3.48, 2.33, 4.38, 4.09, 2.52, 5.19, 2.39, 3.66, 2.29, 2.88,
}

var minorProfile = [12]float32{
	6.33, 2.68, 3.52, 5.38, 2.60, 3.53, 2.54, 4.75, 3.98, 2.69, 3.34, 3.17,
}

// KeyDetection contains the result of key detection
type KeyDetection struct {
	Key          KeySignature     `json:"key"`
	Confidence   float32          `json:"confidence"`
	Alternatives []KeyCorrelation `json:"alternatives"`
}

// KeyCorrelation represents a key with its correlation score
type KeyCorrelation struct {
	Key         KeySignature `json:"key"`
	Correlation float32      `json:"correlation"`
}

// DetectKey detects the key of a sequence using Krumhansl-Schmuckler algorithm
func DetectKey(notes []Note) KeyDetection {
	if len(notes) == 0 {
		return KeyDetection{
			Key:          NewKeySignature(C, Major),
			Confidence:   0.0,
			Alternatives: nil,
		}
	}

	// Count pitch class occurrences
	var pitchCounts [12]uint32
	for _, note := range notes {
		pc := note.Pitch % 12
		pitchCounts[pc]++
	}

	// Normalize to distribution
	var total uint32
	for _, count := range pitchCounts {
		total += count
	}
	distribution := make([]float32, 12)
	for i, count := range pitchCounts {
		distribution[i] = float32(count) / float32(total)
	}

	// Calculate correlation with all possible keys
	var correlations []KeyCorrelation

	for rootIdx := range 12 {
		root := AllPitchClasses[rootIdx]

		// Major key correlation
		majorCorr := pearsonCorrelation(distribution, rotateProfile(&majorProfile, rootIdx))
		correlations = append(correlations, KeyCorrelation{
			Key:         NewKeySignature(root, Major),
			Correlation: majorCorr,
		})

		// Minor key correlation
		minorCorr := pearsonCorrelation(distribution, rotateProfile(&minorProfile, rootIdx))
		correlations = append(correlations, KeyCorrelation{
			Key:         NewKeySignature(root, Minor),
			Correlation: minorCorr,
		})
	}

	// Sort by correlation (descending)
	for i := 0; i < len(correlations)-1; i++ {
		for j := i + 1; j < len(correlations); j++ {
			if correlations[j].Correlation > correlations[i].Correlation {
				correlations[i], correlations[j] = correlations[j], correlations[i]
			}
		}
	}

	// Calculate confidence based on difference between top 2 correlations
	confidence := float32(0.0)
	if len(correlations) >= 2 {
		diff := correlations[0].Correlation - correlations[1].Correlation
		// Normalize: diff of 0.15+ = high confidence
		confidence = diff / 0.15
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < 0.0 {
			confidence = 0.0
		}
	}

	bestKey := correlations[0].Key
	maxAlts := min(4, len(correlations)-1)
	alternatives := correlations[1 : maxAlts+1]

	return KeyDetection{
		Key:          bestKey,
		Confidence:   confidence,
		Alternatives: alternatives,
	}
}

// rotateProfile rotates a profile by a number of semitones
func rotateProfile(profile *[12]float32, semitones int) [12]float32 {
	var rotated [12]float32
	for i := range 12 {
		newIdx := (i + 12 - semitones) % 12
		rotated[newIdx] = profile[i]
	}
	return rotated
}

// pearsonCorrelation calculates Pearson correlation coefficient between two distributions
func pearsonCorrelation(x []float32, y [12]float32) float32 {
	n := 12

	var sumX, sumY float32
	for i := range n {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / float32(n)
	meanY := sumY / float32(n)

	var numerator, denomX, denomY float32
	for i := range n {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	denominator := float32(math.Sqrt(float64(denomX * denomY)))
	if denominator == 0 {
		return 0.0
	}

	return numerator / denominator
}
