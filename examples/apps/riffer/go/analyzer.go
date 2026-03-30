// Package riffer provides music sequence analysis and transformation.
package riffer

import (
	"sort"
)

// AnalysisResult contains comprehensive analysis of a sequence.
type AnalysisResult struct {
	Name             string           `json:"name,omitempty"`
	NoteCount        int              `json:"note_count"`
	DurationTicks    uint64           `json:"duration_ticks"`
	KeyDetection     KeyDetection     `json:"key_detection"`
	ScaleAnalysis    ScaleAnalysis    `json:"scale_analysis"`
	IntervalAnalysis IntervalAnalysis `json:"interval_analysis"`
	ContourAnalysis  ContourAnalysis  `json:"contour_analysis"`
	RhythmAnalysis   RhythmAnalysis   `json:"rhythm_analysis"`
	DynamicsAnalysis DynamicsAnalysis `json:"dynamics_analysis"`
}

// ScaleAnalysis contains scale membership analysis.
type ScaleAnalysis struct {
	Key                  KeySignature          `json:"key"`
	ScaleNotes           []PitchClass          `json:"scale_notes"`
	InScaleCount         int                   `json:"in_scale_count"`
	OutOfScaleCount      int                   `json:"out_of_scale_count"`
	CoherencePercentage  float32               `json:"coherence_percentage"`
	PitchClassesUsed     []PitchClass          `json:"pitch_classes_used"`
	ChromaticAlterations []ChromaticAlteration `json:"chromatic_alterations"`
}

// ChromaticAlteration represents an out-of-scale note.
type ChromaticAlteration struct {
	NoteIndex  int        `json:"note_index"`
	PitchClass PitchClass `json:"pitch_class"`
	Pitch      uint8      `json:"pitch"`
}

// IntervalAnalysis contains interval analysis summary.
type IntervalAnalysis struct {
	Count            int                   `json:"count"`
	Intervals        []IntervalInfo        `json:"intervals,omitempty"`
	ByQuality        IntervalQualityCounts `json:"by_quality"`
	ByDirection      DirectionCounts       `json:"by_direction"`
	LargestInterval  int8                  `json:"largest_interval"`
	SmallestInterval int8                  `json:"smallest_interval"`
	AverageInterval  float32               `json:"average_interval"`
	IntervalVariety  int                   `json:"interval_variety"`
}

// IntervalQualityCounts contains counts by interval quality.
type IntervalQualityCounts struct {
	PerfectConsonance   int `json:"perfect_consonance"`
	ImperfectConsonance int `json:"imperfect_consonance"`
	MildDissonance      int `json:"mild_dissonance"`
	StrongDissonance    int `json:"strong_dissonance"`
}

// DirectionCounts contains counts by direction.
type DirectionCounts struct {
	Ascending  int `json:"ascending"`
	Descending int `json:"descending"`
	Unison     int `json:"unison"`
}

// ContourAnalysis contains melodic contour analysis.
type ContourAnalysis struct {
	ContourType      ContourType `json:"contour_type"`
	DirectionChanges int         `json:"direction_changes"`
	HighestPitch     uint8       `json:"highest_pitch"`
	LowestPitch      uint8       `json:"lowest_pitch"`
	PitchRange       uint8       `json:"pitch_range"`
	ClimaxPosition   float32     `json:"climax_position"`
	NadirPosition    float32     `json:"nadir_position"`
}

// RhythmAnalysis contains rhythm analysis.
type RhythmAnalysis struct {
	UniqueDurations    int     `json:"unique_durations"`
	MostCommonDuration uint32  `json:"most_common_duration"`
	DurationVariety    float32 `json:"duration_variety"`
	AverageDuration    float32 `json:"average_duration"`
	ShortestDuration   uint32  `json:"shortest_duration"`
	LongestDuration    uint32  `json:"longest_duration"`
}

// DynamicsAnalysis contains dynamics analysis.
type DynamicsAnalysis struct {
	MinVelocity      uint8   `json:"min_velocity"`
	MaxVelocity      uint8   `json:"max_velocity"`
	VelocityRange    uint8   `json:"velocity_range"`
	AverageVelocity  float32 `json:"average_velocity"`
	VelocityVariance float32 `json:"velocity_variance"`
	HasDynamics      bool    `json:"has_dynamics"`
}

// AnalyzeSequence performs comprehensive analysis of a sequence.
func AnalyzeSequence(sequence *Sequence, includeIntervals bool) (*AnalysisResult, error) {
	notes := sequence.Notes

	// Key detection
	keyDetection := DetectKey(notes)

	// Use detected or specified key
	key := keyDetection.Key
	if sequence.Context.KeySignature != nil {
		key = *sequence.Context.KeySignature
	}

	// Scale analysis
	scaleAnalysis := analyzeScale(notes, key)

	// Interval analysis
	intervalAnalysis := analyzeIntervalSummary(notes, includeIntervals)

	// Contour analysis
	contourAnalysis := AnalyzeContour(notes)

	// Rhythm analysis
	rhythmAnalysis := analyzeRhythm(notes)

	// Dynamics analysis
	dynamicsAnalysis := analyzeDynamics(notes)

	// Get name as string
	var name string
	if sequence.Name != nil {
		name = *sequence.Name
	}

	return &AnalysisResult{
		Name:             name,
		NoteCount:        len(notes),
		DurationTicks:    sequence.TotalDuration(),
		KeyDetection:     keyDetection,
		ScaleAnalysis:    scaleAnalysis,
		IntervalAnalysis: intervalAnalysis,
		ContourAnalysis:  contourAnalysis,
		RhythmAnalysis:   rhythmAnalysis,
		DynamicsAnalysis: dynamicsAnalysis,
	}, nil
}

// analyzeScale analyzes scale membership.
func analyzeScale(notes []Note, key KeySignature) ScaleAnalysis {
	inScale, total := CountInScale(notes, key)
	outOfScale := total - inScale
	coherence := HarmonicCoherence(notes, key)

	// Get scale notes
	scaleNotes := GetScalePitchClasses(key)

	// Find unique pitch classes used
	pcMap := make(map[PitchClass]bool)
	for _, n := range notes {
		pcMap[n.PitchClass()] = true
	}
	var pitchClassesUsed []PitchClass
	for pc := range pcMap {
		pitchClassesUsed = append(pitchClassesUsed, pc)
	}
	sort.Slice(pitchClassesUsed, func(i, j int) bool {
		return pitchClassesUsed[i].ToSemitone() < pitchClassesUsed[j].ToSemitone()
	})

	// Find chromatic alterations
	var chromatics []ChromaticAlteration
	for i, n := range notes {
		if !IsInScale(n.PitchClass(), key) {
			chromatics = append(chromatics, ChromaticAlteration{
				NoteIndex:  i,
				PitchClass: n.PitchClass(),
				Pitch:      uint8(n.Pitch),
			})
		}
	}

	return ScaleAnalysis{
		Key:                  key,
		ScaleNotes:           scaleNotes,
		InScaleCount:         inScale,
		OutOfScaleCount:      outOfScale,
		CoherencePercentage:  coherence,
		PitchClassesUsed:     pitchClassesUsed,
		ChromaticAlterations: chromatics,
	}
}

// analyzeIntervalSummary analyzes intervals.
func analyzeIntervalSummary(notes []Note, includeAll bool) IntervalAnalysis {
	intervals := AnalyzeIntervals(notes)

	if len(intervals) == 0 {
		return IntervalAnalysis{}
	}

	// Count by quality
	var byQuality IntervalQualityCounts
	for _, interval := range intervals {
		switch interval.Quality {
		case PerfectConsonance:
			byQuality.PerfectConsonance++
		case ImperfectConsonance:
			byQuality.ImperfectConsonance++
		case MildDissonance:
			byQuality.MildDissonance++
		case StrongDissonance:
			byQuality.StrongDissonance++
		}
	}

	// Count by direction
	var byDirection DirectionCounts
	for _, interval := range intervals {
		switch interval.Direction {
		case Ascending:
			byDirection.Ascending++
		case Descending:
			byDirection.Descending++
		case Unison:
			byDirection.Unison++
		}
	}

	// Statistics
	var largest, smallest int8
	var sum int32
	uniqueIntervals := make(map[int8]bool)

	for i, interval := range intervals {
		abs := interval.Semitones
		if abs < 0 {
			abs = -abs
		}
		if i == 0 || abs > largest {
			largest = abs
		}
		if abs > 0 && (i == 0 || abs < smallest) {
			smallest = abs
		}
		sum += int32(abs)
		uniqueIntervals[abs] = true
	}

	average := float32(sum) / float32(len(intervals))

	result := IntervalAnalysis{
		Count:            len(intervals),
		ByQuality:        byQuality,
		ByDirection:      byDirection,
		LargestInterval:  largest,
		SmallestInterval: smallest,
		AverageInterval:  average,
		IntervalVariety:  len(uniqueIntervals),
	}

	if includeAll {
		result.Intervals = intervals
	}

	return result
}

// AnalyzeContour detects melodic contour.
func AnalyzeContour(notes []Note) ContourAnalysis {
	if len(notes) == 0 {
		return ContourAnalysis{ContourType: ContourStatic}
	}

	pitches := make([]uint8, len(notes))
	for i, n := range notes {
		pitches[i] = uint8(n.Pitch)
	}

	// Find extremes
	highest := pitches[0]
	lowest := pitches[0]
	var highestIdx, lowestIdx int

	for i, p := range pitches {
		if p > highest {
			highest = p
			highestIdx = i
		}
		if p < lowest {
			lowest = p
			lowestIdx = i
		}
	}

	pitchRange := highest - lowest
	climaxPos := float32(highestIdx) / float32(max(1, len(pitches)-1))
	nadirPos := float32(lowestIdx) / float32(max(1, len(pitches)-1))

	// Count direction changes
	directionChanges := 0
	var prevDirection int8

	for i := 1; i < len(pitches); i++ {
		diff := int16(pitches[i]) - int16(pitches[i-1])
		var direction int8
		if diff > 0 {
			direction = 1
		} else if diff < 0 {
			direction = -1
		}

		if direction != 0 {
			if prevDirection != 0 && prevDirection != direction {
				directionChanges++
			}
			prevDirection = direction
		}
	}

	// Determine contour type
	contourType := detectContourType(pitches, climaxPos, nadirPos, directionChanges)

	return ContourAnalysis{
		ContourType:      contourType,
		DirectionChanges: directionChanges,
		HighestPitch:     highest,
		LowestPitch:      lowest,
		PitchRange:       pitchRange,
		ClimaxPosition:   climaxPos,
		NadirPosition:    nadirPos,
	}
}

// detectContourType determines the overall contour type.
func detectContourType(pitches []uint8, climaxPos, nadirPos float32, directionChanges int) ContourType {
	if len(pitches) < 2 {
		return ContourStatic
	}

	first := pitches[0]
	last := pitches[len(pitches)-1]

	// Check for predominantly ascending or descending
	if directionChanges == 0 {
		if last > first {
			return ContourAscending
		} else if last < first {
			return ContourDescending
		}
		return ContourStatic
	}

	// Check for arch (climax in middle)
	if climaxPos > 0.2 && climaxPos < 0.8 && directionChanges <= 2 {
		return ContourArch
	}

	// Check for inverse arch (nadir in middle)
	if nadirPos > 0.2 && nadirPos < 0.8 && directionChanges <= 2 {
		return ContourInverseArch
	}

	// Multiple direction changes = wave
	if directionChanges >= 2 {
		return ContourWave
	}

	// Default based on overall direction
	if last > first {
		return ContourAscending
	} else if last < first {
		return ContourDescending
	}
	return ContourStatic
}

// analyzeRhythm analyzes rhythm characteristics.
func analyzeRhythm(notes []Note) RhythmAnalysis {
	if len(notes) == 0 {
		return RhythmAnalysis{}
	}

	durations := make([]uint32, len(notes))
	for i, n := range notes {
		durations[i] = uint32(n.Duration)
	}

	// Find unique durations
	uniqueMap := make(map[uint32]int)
	for _, d := range durations {
		uniqueMap[d]++
	}

	// Find most common
	var mostCommon uint32
	var maxCount int
	for d, count := range uniqueMap {
		if count > maxCount {
			maxCount = count
			mostCommon = d
		}
	}

	// Statistics
	var sum uint64
	shortest := durations[0]
	longest := durations[0]

	for _, d := range durations {
		sum += uint64(d)
		if d < shortest {
			shortest = d
		}
		if d > longest {
			longest = d
		}
	}

	average := float32(sum) / float32(len(durations))

	// Duration variety (normalized)
	var variety float32
	if len(uniqueMap) > 1 {
		variety = float32(len(uniqueMap)) / float32(len(durations))
		if variety > 1.0 {
			variety = 1.0
		}
	}

	return RhythmAnalysis{
		UniqueDurations:    len(uniqueMap),
		MostCommonDuration: mostCommon,
		DurationVariety:    variety,
		AverageDuration:    average,
		ShortestDuration:   shortest,
		LongestDuration:    longest,
	}
}

// analyzeDynamics analyzes dynamics (velocity).
func analyzeDynamics(notes []Note) DynamicsAnalysis {
	if len(notes) == 0 {
		return DynamicsAnalysis{}
	}

	velocities := make([]uint8, len(notes))
	for i, n := range notes {
		velocities[i] = uint8(n.Velocity)
	}

	minV := velocities[0]
	maxV := velocities[0]
	var sum uint32

	for _, v := range velocities {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
		sum += uint32(v)
	}

	velRange := maxV - minV
	average := float32(sum) / float32(len(velocities))

	// Calculate variance
	var varianceSum float32
	for _, v := range velocities {
		diff := float32(v) - average
		varianceSum += diff * diff
	}
	variance := varianceSum / float32(len(velocities))

	return DynamicsAnalysis{
		MinVelocity:      minV,
		MaxVelocity:      maxV,
		VelocityRange:    velRange,
		AverageVelocity:  average,
		VelocityVariance: variance,
		HasDynamics:      velRange > 20,
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
