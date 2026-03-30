// Package riffer provides music sequence analysis and transformation.
//
// Interval classification and analysis.
// Classifies intervals by semitone distance, name, and consonance/dissonance quality.
package riffer

import "fmt"

// IntervalInfo contains information about an interval between two notes
type IntervalInfo struct {
	FromIndex int             `json:"from_index"`
	ToIndex   int             `json:"to_index"`
	Semitones int8            `json:"semitones"`
	Name      string          `json:"name"`
	Quality   IntervalQuality `json:"quality"`
	Direction Direction       `json:"direction"`
}

// NewIntervalInfo creates a new interval info from two consecutive notes
func NewIntervalInfo(fromIndex int, fromPitch, toPitch uint8) IntervalInfo {
	semitones := int8(toPitch) - int8(fromPitch)
	var direction Direction
	switch {
	case semitones > 0:
		direction = Ascending
	case semitones < 0:
		direction = Descending
	default:
		direction = Unison
	}

	return IntervalInfo{
		FromIndex: fromIndex,
		ToIndex:   fromIndex + 1,
		Semitones: semitones,
		Name:      IntervalName(semitones),
		Quality:   ClassifyInterval(semitones),
		Direction: direction,
	}
}

// ClassifyInterval classifies an interval by its consonance/dissonance quality
func ClassifyInterval(semitones int8) IntervalQuality {
	absSemitones := semitones
	if absSemitones < 0 {
		absSemitones = -absSemitones
	}
	absSemitones = absSemitones % 12

	switch absSemitones {
	case 0, 7, 12:
		return PerfectConsonance // Unison, P5, Octave
	case 3, 4, 8, 9:
		return ImperfectConsonance // m3, M3, m6, M6
	case 2, 10:
		return MildDissonance // M2, m7
	case 1, 6, 11:
		return StrongDissonance // m2, tritone, M7
	case 5:
		return ImperfectConsonance // P4 (context-dependent, treat as consonant)
	default:
		return MildDissonance
	}
}

// IntervalName returns the name of an interval by semitone distance
func IntervalName(semitones int8) string {
	absSemitones := semitones
	if absSemitones < 0 {
		absSemitones = -absSemitones
	}
	modSemitones := absSemitones % 12

	direction := ""
	if semitones < 0 {
		direction = "descending "
	}

	names := map[int8]string{
		0:  "unison",
		1:  "minor 2nd",
		2:  "major 2nd",
		3:  "minor 3rd",
		4:  "major 3rd",
		5:  "perfect 4th",
		6:  "tritone",
		7:  "perfect 5th",
		8:  "minor 6th",
		9:  "major 6th",
		10: "minor 7th",
		11: "major 7th",
	}

	name := names[modSemitones]
	if name == "" {
		name = "octave"
	}

	// Handle octave+ intervals
	octaves := absSemitones / 12

	if octaves > 0 && modSemitones == 0 {
		return direction + "octave"
	} else if octaves > 0 {
		suffix := ""
		if octaves > 1 {
			suffix = "s"
		}
		return fmt.Sprintf("%s%s + %d octave%s", direction, name, octaves, suffix)
	}

	return direction + name
}

// IsDissonant checks if an interval is dissonant
func IsDissonant(semitones int8) bool {
	quality := ClassifyInterval(semitones)
	return quality == MildDissonance || quality == StrongDissonance
}

// IsStronglyDissonant checks if an interval is strongly dissonant (needs resolution)
func IsStronglyDissonant(semitones int8) bool {
	return ClassifyInterval(semitones) == StrongDissonance
}

// AnalyzeIntervals analyzes all consecutive intervals in a note sequence
func AnalyzeIntervals(notes []Note) []IntervalInfo {
	if len(notes) < 2 {
		return nil
	}

	intervals := make([]IntervalInfo, 0, len(notes)-1)
	for i := 0; i < len(notes)-1; i++ {
		intervals = append(intervals, NewIntervalInfo(i, uint8(notes[i].Pitch), uint8(notes[i+1].Pitch)))
	}
	return intervals
}
