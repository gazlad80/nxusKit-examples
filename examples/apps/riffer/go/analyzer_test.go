// Package riffer provides music sequence analysis and transformation.
package riffer

import "testing"

func createCMajorScaleForTest() *Sequence {
	notes := []Note{
		NewNote(60, 480, 80, 0),
		NewNote(62, 480, 80, 480),
		NewNote(64, 480, 80, 960),
		NewNote(65, 480, 80, 1440),
		NewNote(67, 480, 80, 1920),
		NewNote(69, 480, 80, 2400),
		NewNote(71, 480, 80, 2880),
		NewNote(72, 480, 80, 3360),
	}

	seq := NewSequence("test", notes)
	key := NewKeySignature(C, Major)
	seq.Context.KeySignature = &key
	name := "C Major Scale"
	seq.Name = &name
	return seq
}

func createVariedSequenceForTest() *Sequence {
	// More varied sequence with dynamics and rhythm
	notes := []Note{
		NewNote(60, 240, 70, 0),    // C4 eighth
		NewNote(64, 480, 85, 240),  // E4 quarter
		NewNote(67, 240, 90, 720),  // G4 eighth
		NewNote(72, 960, 95, 960),  // C5 half
		NewNote(71, 240, 80, 1920), // B4 eighth
		NewNote(67, 480, 75, 2160), // G4 quarter
		NewNote(64, 240, 70, 2640), // E4 eighth
		NewNote(60, 960, 85, 2880), // C4 half
	}

	seq := NewSequence("varied", notes)
	key := NewKeySignature(C, Major)
	seq.Context.KeySignature = &key
	name := "Varied Sequence"
	seq.Name = &name
	return seq
}

func TestAnalyzeCMajorScale(t *testing.T) {
	seq := createCMajorScaleForTest()
	result, err := AnalyzeSequence(seq, false)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	if result.NoteCount != 8 {
		t.Errorf("Expected 8 notes, got %d", result.NoteCount)
	}

	if result.ScaleAnalysis.InScaleCount != 8 {
		t.Errorf("Expected 8 in-scale notes, got %d", result.ScaleAnalysis.InScaleCount)
	}

	if result.ScaleAnalysis.OutOfScaleCount != 0 {
		t.Errorf("Expected 0 out-of-scale notes, got %d", result.ScaleAnalysis.OutOfScaleCount)
	}

	if result.ScaleAnalysis.CoherencePercentage < 99.0 {
		t.Errorf("Expected ~100%% coherence, got %.1f%%", result.ScaleAnalysis.CoherencePercentage)
	}
}

func TestContourAscending(t *testing.T) {
	seq := createCMajorScaleForTest()
	result, err := AnalyzeSequence(seq, false)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	if result.ContourAnalysis.ContourType != ContourAscending {
		t.Errorf("Expected ContourAscending, got %v", result.ContourAnalysis.ContourType)
	}

	if result.ContourAnalysis.DirectionChanges != 0 {
		t.Errorf("Expected 0 direction changes, got %d", result.ContourAnalysis.DirectionChanges)
	}
}

func TestIntervalAnalysis(t *testing.T) {
	seq := createCMajorScaleForTest()
	result, err := AnalyzeSequence(seq, true)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	if result.IntervalAnalysis.Count != 7 {
		t.Errorf("Expected 7 intervals, got %d", result.IntervalAnalysis.Count)
	}

	// All seconds (some major, some minor)
	if result.IntervalAnalysis.ByQuality.MildDissonance == 0 &&
		result.IntervalAnalysis.ByQuality.StrongDissonance == 0 {
		t.Error("Expected some mild or strong dissonances for seconds")
	}
}

func TestRhythmAnalysis(t *testing.T) {
	seq := createCMajorScaleForTest()
	result, err := AnalyzeSequence(seq, false)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	// All same duration
	if result.RhythmAnalysis.UniqueDurations != 1 {
		t.Errorf("Expected 1 unique duration, got %d", result.RhythmAnalysis.UniqueDurations)
	}

	if result.RhythmAnalysis.MostCommonDuration != 480 {
		t.Errorf("Expected most common duration 480, got %d", result.RhythmAnalysis.MostCommonDuration)
	}
}

func TestDynamicsUniform(t *testing.T) {
	seq := createCMajorScaleForTest()
	result, err := AnalyzeSequence(seq, false)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	// All same velocity
	if result.DynamicsAnalysis.VelocityRange != 0 {
		t.Errorf("Expected velocity range 0, got %d", result.DynamicsAnalysis.VelocityRange)
	}

	if result.DynamicsAnalysis.HasDynamics {
		t.Error("Expected HasDynamics to be false")
	}
}

func TestVariedSequenceRhythm(t *testing.T) {
	seq := createVariedSequenceForTest()
	result, err := AnalyzeSequence(seq, false)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	// Should have multiple duration types
	if result.RhythmAnalysis.UniqueDurations < 3 {
		t.Errorf("Expected at least 3 unique durations, got %d", result.RhythmAnalysis.UniqueDurations)
	}
}

func TestVariedSequenceDynamics(t *testing.T) {
	seq := createVariedSequenceForTest()
	result, err := AnalyzeSequence(seq, false)
	if err != nil {
		t.Fatalf("AnalyzeSequence failed: %v", err)
	}

	// Should have dynamic variation
	if result.DynamicsAnalysis.VelocityRange < 20 {
		t.Errorf("Expected velocity range >= 20, got %d", result.DynamicsAnalysis.VelocityRange)
	}

	if !result.DynamicsAnalysis.HasDynamics {
		t.Error("Expected HasDynamics to be true")
	}
}
