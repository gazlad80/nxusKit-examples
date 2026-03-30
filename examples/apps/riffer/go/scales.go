// Package riffer provides music sequence analysis and transformation.
//
// Scale definitions and pitch-class membership.
// Provides scale interval patterns and membership checking.
package riffer

// ScaleInfo contains information about a scale
type ScaleInfo struct {
	ScaleType    ScaleType
	PitchClasses []PitchClass
	Intervals    []uint8
}

// GetScaleIntervals returns the semitone intervals for a scale type
func GetScaleIntervals(scaleType ScaleType) []uint8 {
	intervals := map[ScaleType][]uint8{
		ScaleMajor:         {0, 2, 4, 5, 7, 9, 11},
		ScaleNaturalMinor:  {0, 2, 3, 5, 7, 8, 10},
		ScaleHarmonicMinor: {0, 2, 3, 5, 7, 8, 11},
		ScaleMelodicMinor:  {0, 2, 3, 5, 7, 9, 11},
		ScalePentatonic:    {0, 2, 4, 7, 9},
		ScaleBlues:         {0, 3, 5, 6, 7, 10},
		ScaleDorian:        {0, 2, 3, 5, 7, 9, 10},
		ScalePhrygian:      {0, 1, 3, 5, 7, 8, 10},
		ScaleLydian:        {0, 2, 4, 6, 7, 9, 11},
		ScaleMixolydian:    {0, 2, 4, 5, 7, 9, 10},
		ScaleAeolian:       {0, 2, 3, 5, 7, 8, 10},
		ScaleLocrian:       {0, 1, 3, 5, 6, 8, 10},
	}
	if ints, ok := intervals[scaleType]; ok {
		return ints
	}
	return intervals[ScaleMajor]
}

// ModeToScaleType converts a mode to its corresponding scale type
func ModeToScaleType(mode Mode) ScaleType {
	switch mode {
	case Major:
		return ScaleMajor
	case Minor, Aeolian:
		return ScaleNaturalMinor
	case Dorian:
		return ScaleDorian
	case Phrygian:
		return ScalePhrygian
	case Lydian:
		return ScaleLydian
	case Mixolydian:
		return ScaleMixolydian
	case Locrian:
		return ScaleLocrian
	default:
		return ScaleMajor
	}
}

// IsInScale checks if a pitch class is in a scale
func IsInScale(pitchClass PitchClass, key KeySignature) bool {
	rootSemitone := key.Root.ToSemitone()
	scaleIntervals := GetScaleIntervals(ModeToScaleType(key.Mode))
	pitchSemitone := pitchClass.ToSemitone()

	// Calculate interval from root
	interval := (pitchSemitone + 12 - rootSemitone) % 12

	for _, scaleInterval := range scaleIntervals {
		if uint8(interval) == scaleInterval {
			return true
		}
	}
	return false
}

// GetScalePitchClasses returns all pitch classes in a scale
func GetScalePitchClasses(key KeySignature) []PitchClass {
	rootSemitone := key.Root.ToSemitone()
	intervals := GetScaleIntervals(ModeToScaleType(key.Mode))

	pitchClasses := make([]PitchClass, len(intervals))
	for i, interval := range intervals {
		semitone := (rootSemitone + int(interval)) % 12
		pitchClasses[i] = AllPitchClasses[semitone]
	}
	return pitchClasses
}

// BuildScaleInfo builds scale info for a given key
func BuildScaleInfo(key KeySignature) ScaleInfo {
	scaleType := ModeToScaleType(key.Mode)
	intervals := GetScaleIntervals(scaleType)
	pitchClasses := GetScalePitchClasses(key)

	return ScaleInfo{
		ScaleType:    scaleType,
		PitchClasses: pitchClasses,
		Intervals:    intervals,
	}
}

// CountInScale counts how many notes in a sequence are in the scale
func CountInScale(notes []Note, key KeySignature) (inScale int, total int) {
	total = len(notes)
	for _, n := range notes {
		pc := PitchClassFromMidi(n.Pitch)
		if IsInScale(pc, key) {
			inScale++
		}
	}
	return inScale, total
}

// HarmonicCoherence calculates harmonic coherence as percentage
func HarmonicCoherence(notes []Note, key KeySignature) float32 {
	inScale, total := CountInScale(notes, key)
	if total == 0 {
		return 100.0
	}
	return float32(inScale) / float32(total) * 100.0
}
