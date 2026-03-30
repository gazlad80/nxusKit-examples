//go:build nxuskit

// Package riffer provides music sequence analysis and transformation.
package riffer

import (
	"testing"
)

func TestNoteToFact(t *testing.T) {
	engine := &ClipsRuleEngine{
		rulesDir:  ".",
		available: true,
	}

	note := NewNote(60, 480, 80, 0) // Middle C
	fact := engine.noteToFact(&note, 0)

	// Check template
	if fact["template"] != "note" {
		t.Errorf("Expected template 'note', got %v", fact["template"])
	}

	// Check values
	values, ok := fact["values"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected values to be a map")
	}

	if values["pitch"] != MidiNote(60) {
		t.Errorf("Expected pitch 60, got %v", values["pitch"])
	}

	if values["index"] != 0 {
		t.Errorf("Expected index 0, got %v", values["index"])
	}
}

func TestContextToFact(t *testing.T) {
	engine := &ClipsRuleEngine{
		rulesDir:  ".",
		available: true,
	}

	key := NewKeySignature(C, Major)
	context := Context{
		KeySignature:    &key,
		TimeSignature:   TimeSignature{Numerator: 4, Denominator: 4},
		Tempo:           120,
		TicksPerQuarter: 480,
	}

	fact := engine.contextToFact(&context)

	if fact["template"] != "context" {
		t.Errorf("Expected template 'context', got %v", fact["template"])
	}

	values, ok := fact["values"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected values to be a map")
	}

	if values["tempo"] != uint16(120) {
		t.Errorf("Expected tempo 120, got %v", values["tempo"])
	}
}

func TestIntervalToFact(t *testing.T) {
	engine := &ClipsRuleEngine{
		rulesDir:  ".",
		available: true,
	}

	interval := IntervalInfo{
		FromIndex: 0,
		ToIndex:   1,
		Semitones: 4, // Major 3rd
		Quality:   ImperfectConsonance,
		Direction: Ascending,
	}

	fact := engine.intervalToFact(&interval)

	if fact["template"] != "interval" {
		t.Errorf("Expected template 'interval', got %v", fact["template"])
	}

	values, ok := fact["values"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected values to be a map")
	}

	if values["semitones"] != int8(4) {
		t.Errorf("Expected semitones 4, got %v", values["semitones"])
	}
}

func TestExtractAdjustments(t *testing.T) {
	engine := &ClipsRuleEngine{
		rulesDir:  ".",
		available: true,
	}

	output := map[string]interface{}{
		"conclusions": []interface{}{
			map[string]interface{}{
				"template": "scoring-adjustment",
				"values": map[string]interface{}{
					"dimension":  "harmonic-coherence",
					"adjustment": 5.0,
					"reason":     "Good use of leading tone",
					"rule-name":  "leading-tone-bonus",
				},
			},
		},
	}

	adjustments := engine.ExtractAdjustments(output)

	if len(adjustments) != 1 {
		t.Fatalf("Expected 1 adjustment, got %d", len(adjustments))
	}

	if adjustments[0].Dimension != "harmonic-coherence" {
		t.Errorf("Expected dimension 'harmonic-coherence', got %s", adjustments[0].Dimension)
	}

	if adjustments[0].Adjustment != 5.0 {
		t.Errorf("Expected adjustment 5.0, got %f", adjustments[0].Adjustment)
	}
}

func TestExtractSuggestions(t *testing.T) {
	engine := &ClipsRuleEngine{
		rulesDir:  ".",
		available: true,
	}

	output := map[string]interface{}{
		"conclusions": []interface{}{
			map[string]interface{}{
				"template": "suggestion",
				"values": map[string]interface{}{
					"category":  "resolution",
					"severity":  "warning",
					"message":   "Unresolved tritone at measure 4",
					"rule-name": "unresolved-tritone",
				},
			},
		},
	}

	suggestions := engine.ExtractSuggestions(output)

	if len(suggestions) != 1 {
		t.Fatalf("Expected 1 suggestion, got %d", len(suggestions))
	}

	if suggestions[0].Category != "resolution" {
		t.Errorf("Expected category 'resolution', got %s", suggestions[0].Category)
	}

	if suggestions[0].Severity != "warning" {
		t.Errorf("Expected severity 'warning', got %s", suggestions[0].Severity)
	}
}

func TestSequenceToFacts(t *testing.T) {
	engine := &ClipsRuleEngine{
		rulesDir:  ".",
		available: true,
	}

	// C major triad
	notes := []Note{
		NewNote(60, 480, 80, 0),   // C4
		NewNote(64, 480, 80, 480), // E4
		NewNote(67, 480, 80, 960), // G4
	}

	key := NewKeySignature(C, Major)
	sequence := &Sequence{
		ID:    "test",
		Notes: notes,
		Context: Context{
			KeySignature:    &key,
			TimeSignature:   TimeSignature{Numerator: 4, Denominator: 4},
			Tempo:           120,
			TicksPerQuarter: 480,
		},
	}

	factsJSON := engine.SequenceToFacts(sequence)

	facts, ok := factsJSON["facts"].([]map[string]interface{})
	if !ok {
		t.Fatal("Expected facts to be an array of maps")
	}

	// Should have:
	// - 1 context fact
	// - 3 note facts
	// - 2 interval facts
	// - 3 scale membership facts
	// Total: 9+ facts (may have dissonance facts too)
	if len(facts) < 9 {
		t.Errorf("Expected at least 9 facts, got %d", len(facts))
	}

	// Verify first fact is context
	if facts[0]["template"] != "context" {
		t.Errorf("Expected first fact to be context, got %v", facts[0]["template"])
	}
}

func TestApplyAdjustments(t *testing.T) {
	scores := map[string]float32{
		"harmonic-coherence": 80.0,
		"melodic-interest":   70.0,
	}

	adjustments := []ScoringAdjustment{
		{Dimension: "harmonic-coherence", Adjustment: 10.0},
		{Dimension: "melodic-interest", Adjustment: -5.0},
	}

	ApplyAdjustments(scores, adjustments)

	if scores["harmonic-coherence"] != 90.0 {
		t.Errorf("Expected harmonic-coherence 90.0, got %f", scores["harmonic-coherence"])
	}

	if scores["melodic-interest"] != 65.0 {
		t.Errorf("Expected melodic-interest 65.0, got %f", scores["melodic-interest"])
	}
}

func TestApplyAdjustmentsClamp(t *testing.T) {
	scores := map[string]float32{
		"harmonic-coherence": 95.0,
		"melodic-interest":   5.0,
	}

	adjustments := []ScoringAdjustment{
		{Dimension: "harmonic-coherence", Adjustment: 20.0}, // Should clamp to 100
		{Dimension: "melodic-interest", Adjustment: -10.0},  // Should clamp to 0
	}

	ApplyAdjustments(scores, adjustments)

	if scores["harmonic-coherence"] != 100.0 {
		t.Errorf("Expected harmonic-coherence clamped to 100.0, got %f", scores["harmonic-coherence"])
	}

	if scores["melodic-interest"] != 0.0 {
		t.Errorf("Expected melodic-interest clamped to 0.0, got %f", scores["melodic-interest"])
	}
}

func TestNewClipsRuleEngine(t *testing.T) {
	// Test with non-existent directory
	engine, err := NewClipsRuleEngine("/nonexistent/path")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if engine.IsAvailable() {
		t.Error("Expected engine to be unavailable for non-existent path")
	}

	// Test Analyze returns empty result when unavailable
	sequence := &Sequence{
		ID:      "test",
		Notes:   []Note{},
		Context: DefaultContext(),
	}

	result, err := engine.Analyze(sequence)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Adjustments) != 0 {
		t.Error("Expected empty adjustments")
	}

	if len(result.Suggestions) != 0 {
		t.Error("Expected empty suggestions")
	}
}
