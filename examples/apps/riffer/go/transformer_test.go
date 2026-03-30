// Package riffer provides music sequence analysis and transformation.
package riffer

import "testing"

func createTransformTestSequence() *Sequence {
	notes := []Note{
		NewNote(60, 480, 80, 0),    // C4
		NewNote(64, 480, 80, 480),  // E4
		NewNote(67, 480, 80, 960),  // G4
		NewNote(72, 480, 80, 1440), // C5
	}

	seq := NewSequence("test", notes)
	key := NewKeySignature(C, Major)
	seq.Context.KeySignature = &key
	seq.Context.Tempo = 120
	return seq
}

func TestTransposeUp(t *testing.T) {
	seq := createTransformTestSequence()
	err := Transpose(seq, 5)
	if err != nil {
		t.Fatalf("Transpose failed: %v", err)
	}

	if seq.Notes[0].Pitch != 65 { // F4
		t.Errorf("Expected pitch 65, got %d", seq.Notes[0].Pitch)
	}
	if seq.Notes[1].Pitch != 69 { // A4
		t.Errorf("Expected pitch 69, got %d", seq.Notes[1].Pitch)
	}

	// Key signature should be updated
	if seq.Context.KeySignature.Root != F {
		t.Errorf("Expected key root F, got %v", seq.Context.KeySignature.Root)
	}
}

func TestTransposeDown(t *testing.T) {
	seq := createTransformTestSequence()
	err := Transpose(seq, -2)
	if err != nil {
		t.Fatalf("Transpose failed: %v", err)
	}

	if seq.Notes[0].Pitch != 58 { // Bb3
		t.Errorf("Expected pitch 58, got %d", seq.Notes[0].Pitch)
	}
}

func TestTransposeClampsHigh(t *testing.T) {
	seq := NewSequence("test", []Note{NewNote(120, 480, 80, 0)})
	Transpose(seq, 20)
	if seq.Notes[0].Pitch != 127 {
		t.Errorf("Expected clamped pitch 127, got %d", seq.Notes[0].Pitch)
	}
}

func TestTransposeClampsLow(t *testing.T) {
	seq := NewSequence("test", []Note{NewNote(10, 480, 80, 0)})
	Transpose(seq, -20)
	if seq.Notes[0].Pitch != 0 {
		t.Errorf("Expected clamped pitch 0, got %d", seq.Notes[0].Pitch)
	}
}

func TestChangeTempo(t *testing.T) {
	seq := createTransformTestSequence()
	err := ChangeTempo(seq, 140)
	if err != nil {
		t.Fatalf("ChangeTempo failed: %v", err)
	}
	if seq.Context.Tempo != 140 {
		t.Errorf("Expected tempo 140, got %d", seq.Context.Tempo)
	}
}

func TestTempoValidation(t *testing.T) {
	seq := createTransformTestSequence()
	if err := ChangeTempo(seq, 10); err == nil {
		t.Error("Expected error for too slow tempo")
	}
	if err := ChangeTempo(seq, 400); err == nil {
		t.Error("Expected error for too fast tempo")
	}
}

func TestInvert(t *testing.T) {
	seq := createTransformTestSequence()
	pivot := uint8(60)
	err := Invert(seq, &pivot)
	if err != nil {
		t.Fatalf("Invert failed: %v", err)
	}

	if seq.Notes[0].Pitch != 60 { // C4 stays same (pivot)
		t.Errorf("Expected pitch 60, got %d", seq.Notes[0].Pitch)
	}
	if seq.Notes[1].Pitch != 56 { // E4 -> Ab3
		t.Errorf("Expected pitch 56, got %d", seq.Notes[1].Pitch)
	}
	if seq.Notes[2].Pitch != 53 { // G4 -> F3
		t.Errorf("Expected pitch 53, got %d", seq.Notes[2].Pitch)
	}
}

func TestInvertDefaultPivot(t *testing.T) {
	seq := createTransformTestSequence()
	err := Invert(seq, nil)
	if err != nil {
		t.Fatalf("Invert failed: %v", err)
	}

	// First note should be pivot (unchanged)
	if seq.Notes[0].Pitch != 60 {
		t.Errorf("Expected pitch 60 (pivot), got %d", seq.Notes[0].Pitch)
	}
}

func TestRetrograde(t *testing.T) {
	seq := createTransformTestSequence()
	err := Retrograde(seq)
	if err != nil {
		t.Fatalf("Retrograde failed: %v", err)
	}

	// Order should be reversed
	if seq.Notes[0].Pitch != 72 { // Was last (C5)
		t.Errorf("Expected pitch 72, got %d", seq.Notes[0].Pitch)
	}
	if seq.Notes[3].Pitch != 60 { // Was first (C4)
		t.Errorf("Expected pitch 60, got %d", seq.Notes[3].Pitch)
	}

	// Start times should be recalculated
	if seq.Notes[0].StartTick != 0 {
		t.Errorf("Expected start tick 0, got %d", seq.Notes[0].StartTick)
	}
	if seq.Notes[1].StartTick != 480 {
		t.Errorf("Expected start tick 480, got %d", seq.Notes[1].StartTick)
	}
}

func TestAugment(t *testing.T) {
	seq := createTransformTestSequence()
	err := Augment(seq, 2.0)
	if err != nil {
		t.Fatalf("Augment failed: %v", err)
	}

	if seq.Notes[0].Duration != 960 { // 480 * 2
		t.Errorf("Expected duration 960, got %d", seq.Notes[0].Duration)
	}
	if seq.Notes[1].StartTick != 960 { // 480 * 2
		t.Errorf("Expected start tick 960, got %d", seq.Notes[1].StartTick)
	}
}

func TestDiminish(t *testing.T) {
	seq := createTransformTestSequence()
	err := Diminish(seq, 2.0)
	if err != nil {
		t.Fatalf("Diminish failed: %v", err)
	}

	if seq.Notes[0].Duration != 240 { // 480 / 2
		t.Errorf("Expected duration 240, got %d", seq.Notes[0].Duration)
	}
	if seq.Notes[1].StartTick != 240 { // 480 / 2
		t.Errorf("Expected start tick 240, got %d", seq.Notes[1].StartTick)
	}
}

func TestDiminishMinimumDuration(t *testing.T) {
	seq := NewSequence("test", []Note{NewNote(60, 1, 80, 0)})
	Diminish(seq, 8.0)
	if seq.Notes[0].Duration < 1 {
		t.Errorf("Duration should not go below 1, got %d", seq.Notes[0].Duration)
	}
}

func TestKeyChange(t *testing.T) {
	seq := createTransformTestSequence()
	target := NewKeySignature(G, Major)
	err := KeyChange(seq, target)
	if err != nil {
		t.Fatalf("KeyChange failed: %v", err)
	}

	// Should transpose (either +7 or -5)
	pitchDiff := int(seq.Notes[0].Pitch) - 60
	if pitchDiff != 7 && pitchDiff != -5 {
		t.Errorf("Expected transpose of +7 or -5, got %d", pitchDiff)
	}

	if seq.Context.KeySignature.Root != G {
		t.Errorf("Expected key root G, got %v", seq.Context.KeySignature.Root)
	}
}

func TestTransformChain(t *testing.T) {
	seq := createTransformTestSequence()
	ops := []TransformOp{
		TransposeOp{Semitones: 2},
		TempoOp{BPM: 140},
	}

	err := TransformChain(seq, ops)
	if err != nil {
		t.Fatalf("TransformChain failed: %v", err)
	}

	if seq.Notes[0].Pitch != 62 { // D4 (transposed up 2)
		t.Errorf("Expected pitch 62, got %d", seq.Notes[0].Pitch)
	}
	if seq.Context.Tempo != 140 {
		t.Errorf("Expected tempo 140, got %d", seq.Context.Tempo)
	}
}

func TestValidateDurationFactor(t *testing.T) {
	if err := ValidateDurationFactor(0.05); err == nil {
		t.Error("Expected error for too small factor")
	}
	if err := ValidateDurationFactor(10.0); err == nil {
		t.Error("Expected error for too large factor")
	}
	if err := ValidateDurationFactor(1.0); err != nil {
		t.Errorf("Expected no error for valid factor: %v", err)
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		input    string
		wantRoot PitchClass
		wantMode Mode
		wantOk   bool
	}{
		{"C", C, Major, true},
		{"Am", A, Minor, true},
		{"F#", Fs, Major, true},
		{"Bbm", As, Minor, true},
		{"G", G, Major, true},
		{"Em", E, Minor, true},
		{"", C, Major, false},
		{"X", C, Major, false},
	}

	for _, tt := range tests {
		key, ok := ParseKey(tt.input)
		if ok != tt.wantOk {
			t.Errorf("ParseKey(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
			continue
		}
		if !ok {
			continue
		}
		if key.Root != tt.wantRoot {
			t.Errorf("ParseKey(%q) root = %v, want %v", tt.input, key.Root, tt.wantRoot)
		}
		if key.Mode != tt.wantMode {
			t.Errorf("ParseKey(%q) mode = %v, want %v", tt.input, key.Mode, tt.wantMode)
		}
	}
}
