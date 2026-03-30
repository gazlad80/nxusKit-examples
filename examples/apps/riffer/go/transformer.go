// Package riffer provides music sequence analysis and transformation.
package riffer

import "fmt"

// TransformOp represents a transformation operation.
type TransformOp interface {
	Apply(seq *Sequence) error
}

// TransposeOp transposes all notes by semitones.
type TransposeOp struct {
	Semitones int8
}

func (op TransposeOp) Apply(seq *Sequence) error {
	return Transpose(seq, op.Semitones)
}

// TempoOp changes the tempo.
type TempoOp struct {
	BPM uint16
}

func (op TempoOp) Apply(seq *Sequence) error {
	return ChangeTempo(seq, op.BPM)
}

// InvertOp inverts the melody around a pivot pitch.
type InvertOp struct {
	Pivot *uint8 // nil means use first note
}

func (op InvertOp) Apply(seq *Sequence) error {
	return Invert(seq, op.Pivot)
}

// RetrogradeOp reverses note order.
type RetrogradeOp struct{}

func (op RetrogradeOp) Apply(seq *Sequence) error {
	return Retrograde(seq)
}

// AugmentOp scales durations by a factor.
type AugmentOp struct {
	Factor float32
}

func (op AugmentOp) Apply(seq *Sequence) error {
	return Augment(seq, op.Factor)
}

// DiminishOp scales durations by dividing by a factor.
type DiminishOp struct {
	Factor float32
}

func (op DiminishOp) Apply(seq *Sequence) error {
	return Diminish(seq, op.Factor)
}

// KeyChangeOp changes the key.
type KeyChangeOp struct {
	TargetKey KeySignature
}

func (op KeyChangeOp) Apply(seq *Sequence) error {
	return KeyChange(seq, op.TargetKey)
}

// Transform applies a single transformation to a sequence.
func Transform(seq *Sequence, op TransformOp) error {
	return op.Apply(seq)
}

// TransformChain applies multiple transformations in sequence.
func TransformChain(seq *Sequence, ops []TransformOp) error {
	for _, op := range ops {
		if err := op.Apply(seq); err != nil {
			return err
		}
	}
	return nil
}

// Transpose transposes all notes by semitones.
func Transpose(seq *Sequence, semitones int8) error {
	for i := range seq.Notes {
		newPitch := int(seq.Notes[i].Pitch) + int(semitones)
		if newPitch < 0 {
			newPitch = 0
		}
		if newPitch > 127 {
			newPitch = 127
		}
		seq.Notes[i].Pitch = MidiNote(newPitch)
	}

	// Update key signature if present
	if seq.Context.KeySignature != nil {
		newRootSemitone := (seq.Context.KeySignature.Root.ToSemitone() + int(semitones)) % 12
		if newRootSemitone < 0 {
			newRootSemitone += 12
		}
		seq.Context.KeySignature.Root = AllPitchClasses[newRootSemitone]
	}

	return nil
}

// ChangeTempo changes the tempo.
func ChangeTempo(seq *Sequence, bpm uint16) error {
	if err := ValidateTempo(bpm); err != nil {
		return err
	}
	seq.Context.Tempo = bpm
	return nil
}

// Invert inverts the melody around a pivot pitch.
func Invert(seq *Sequence, pivot *uint8) error {
	if len(seq.Notes) == 0 {
		return nil
	}

	// Use first note as pivot if not specified
	var pivotPitch uint8
	if pivot != nil {
		pivotPitch = *pivot
	} else {
		pivotPitch = uint8(seq.Notes[0].Pitch)
	}

	for i := range seq.Notes {
		diff := int(seq.Notes[i].Pitch) - int(pivotPitch)
		newPitch := int(pivotPitch) - diff
		if newPitch < 0 {
			newPitch = 0
		}
		if newPitch > 127 {
			newPitch = 127
		}
		seq.Notes[i].Pitch = MidiNote(newPitch)
	}

	return nil
}

// Retrograde reverses the note order.
func Retrograde(seq *Sequence) error {
	if len(seq.Notes) < 2 {
		return nil
	}

	// Reverse the notes
	for i, j := 0, len(seq.Notes)-1; i < j; i, j = i+1, j-1 {
		seq.Notes[i], seq.Notes[j] = seq.Notes[j], seq.Notes[i]
	}

	// Recalculate start times
	var currentTime uint64
	for i := range seq.Notes {
		seq.Notes[i].StartTick = currentTime
		currentTime += uint64(seq.Notes[i].Duration)
	}

	return nil
}

// Augment scales durations by multiplying by a factor.
func Augment(seq *Sequence, factor float32) error {
	if err := ValidateDurationFactor(factor); err != nil {
		return err
	}

	for i := range seq.Notes {
		seq.Notes[i].Duration = Ticks(float32(seq.Notes[i].Duration) * factor)
		seq.Notes[i].StartTick = uint64(float32(seq.Notes[i].StartTick) * factor)
	}

	return nil
}

// Diminish scales durations by dividing by a factor.
func Diminish(seq *Sequence, factor float32) error {
	if err := ValidateDurationFactor(factor); err != nil {
		return err
	}

	for i := range seq.Notes {
		newDuration := Ticks(float32(seq.Notes[i].Duration) / factor)
		if newDuration < 1 {
			newDuration = 1
		}
		seq.Notes[i].Duration = newDuration
		seq.Notes[i].StartTick = uint64(float32(seq.Notes[i].StartTick) / factor)
	}

	return nil
}

// KeyChange changes the key (diatonic transposition).
func KeyChange(seq *Sequence, targetKey KeySignature) error {
	// Detect current key if not specified
	var sourceKey KeySignature
	if seq.Context.KeySignature != nil {
		sourceKey = *seq.Context.KeySignature
	} else {
		detection := DetectKey(seq.Notes)
		sourceKey = detection.Key
	}

	// Calculate semitone difference between root notes
	sourceRoot := int8(sourceKey.Root.ToSemitone())
	targetRoot := int8(targetKey.Root.ToSemitone())
	semitoneShift := targetRoot - sourceRoot

	// Adjust for mode change
	if sourceKey.Mode == Major && targetKey.Mode == Minor {
		semitoneShift -= 3
	} else if sourceKey.Mode == Minor && targetKey.Mode == Major {
		semitoneShift += 3
	}

	// Normalize to -6..+5 range
	if semitoneShift > 6 {
		semitoneShift -= 12
	} else if semitoneShift < -6 {
		semitoneShift += 12
	}

	// Apply transpose
	if err := Transpose(seq, semitoneShift); err != nil {
		return err
	}

	// Set the new key signature
	seq.Context.KeySignature = &targetKey

	return nil
}

// ValidatePitch validates MIDI pitch range (0-127).
func ValidatePitch(pitch int16) (uint8, error) {
	if pitch < 0 || pitch > 127 {
		return 0, fmt.Errorf("pitch %d is out of MIDI range (0-127)", pitch)
	}
	return uint8(pitch), nil
}

// ValidateTempo validates tempo range (20-300 BPM).
func ValidateTempo(bpm uint16) error {
	if bpm < 20 || bpm > 300 {
		return fmt.Errorf("tempo %d is out of valid range (20-300 BPM)", bpm)
	}
	return nil
}

// ValidateDurationFactor validates duration factor (0.125 to 8.0).
func ValidateDurationFactor(factor float32) error {
	if factor < 0.125 || factor > 8.0 {
		return fmt.Errorf("duration factor %f is out of valid range (0.125-8.0)", factor)
	}
	return nil
}

// ParseKey parses a key string like "C", "Am", "F#", "Bbm" into a KeySignature.
func ParseKey(s string) (*KeySignature, bool) {
	if s == "" {
		return nil, false
	}

	// Check if it ends with 'm' for minor mode
	var rootStr string
	var mode Mode

	if len(s) > 1 && s[len(s)-1] == 'm' {
		rootStr = s[:len(s)-1]
		mode = Minor
	} else {
		rootStr = s
		mode = Major
	}

	// Parse the root pitch class
	root, ok := ParsePitchClass(rootStr)
	if !ok {
		return nil, false
	}

	return &KeySignature{Root: root, Mode: mode}, true
}
