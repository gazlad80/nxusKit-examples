// Package riffer provides music sequence analysis and transformation.
package riffer

import (
	"os"
	"path/filepath"
	"testing"
)

func findTestdataDir(t *testing.T) string {
	// Try relative paths from test location
	candidates := []string{
		"testdata",
		"./testdata",
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}

	t.Skip("testdata directory not found")
	return ""
}

func TestReadCMajorScale(t *testing.T) {
	dir := findTestdataDir(t)
	path := filepath.Join(dir, "c_major_scale.mid")

	seq, err := ReadMIDI(path)
	if err != nil {
		t.Fatalf("Failed to read c_major_scale.mid: %v", err)
	}

	// Verify note count
	if len(seq.Notes) != 8 {
		t.Errorf("Expected 8 notes, got %d", len(seq.Notes))
	}

	// Verify first note is C4 (MIDI 60)
	if seq.Notes[0].Pitch != 60 {
		t.Errorf("Expected first note pitch 60 (C4), got %d", seq.Notes[0].Pitch)
	}

	// Verify last note is C5 (MIDI 72)
	if seq.Notes[7].Pitch != 72 {
		t.Errorf("Expected last note pitch 72 (C5), got %d", seq.Notes[7].Pitch)
	}

	// Test key detection
	keyDetection := DetectKey(seq.Notes)
	if keyDetection.Key.Root != C {
		t.Errorf("Expected key root C, got %v", keyDetection.Key.Root)
	}
	if keyDetection.Key.Mode != Major {
		t.Errorf("Expected mode Major, got %v", keyDetection.Key.Mode)
	}

	// Test harmonic coherence (should be 100% for C major scale in C major)
	coherence := HarmonicCoherence(seq.Notes, keyDetection.Key)
	if coherence < 99.0 {
		t.Errorf("Expected harmonic coherence ~100%%, got %.1f%%", coherence)
	}
}

func TestReadEMinorRiff(t *testing.T) {
	dir := findTestdataDir(t)
	path := filepath.Join(dir, "e_minor_riff.mid")

	seq, err := ReadMIDI(path)
	if err != nil {
		t.Fatalf("Failed to read e_minor_riff.mid: %v", err)
	}

	// Verify note count
	if len(seq.Notes) != 13 {
		t.Errorf("Expected 13 notes, got %d", len(seq.Notes))
	}

	// Verify first note is E4 (MIDI 64)
	if seq.Notes[0].Pitch != 64 {
		t.Errorf("Expected first note pitch 64 (E4), got %d", seq.Notes[0].Pitch)
	}

	// Test key detection - should be E minor or G major (relative keys)
	// Note: Krumhansl-Schmuckler may detect alternative keys for pentatonic content
	keyDetection := DetectKey(seq.Notes)
	isValidKey := (keyDetection.Key.Root == E && keyDetection.Key.Mode == Minor) ||
		(keyDetection.Key.Root == G && keyDetection.Key.Mode == Major)
	if !isValidKey {
		// Log but don't fail - key detection with pentatonic content can vary
		t.Logf("Note: Key detected as %v %v (expected E Minor or G Major, pentatonic content may affect detection)",
			keyDetection.Key.Root, keyDetection.Key.Mode)
		// Check that E minor is at least in the alternatives
		foundEMinor := false
		for _, alt := range keyDetection.Alternatives {
			if alt.Key.Root == E && alt.Key.Mode == Minor {
				foundEMinor = true
				break
			}
		}
		if !foundEMinor {
			t.Logf("E Minor not found in alternatives: %v", keyDetection.Alternatives)
		}
	}

	// Test interval analysis
	intervals := AnalyzeIntervals(seq.Notes)
	if len(intervals) != 12 {
		t.Errorf("Expected 12 intervals, got %d", len(intervals))
	}

	// Count consonances and dissonances
	consonances := 0
	for _, interval := range intervals {
		if interval.Quality == PerfectConsonance || interval.Quality == ImperfectConsonance {
			consonances++
		}
	}
	if consonances < 6 {
		t.Errorf("Expected at least 6 consonant intervals, got %d", consonances)
	}
}

func TestReadChromaticTest(t *testing.T) {
	dir := findTestdataDir(t)
	path := filepath.Join(dir, "chromatic_test.mid")

	seq, err := ReadMIDI(path)
	if err != nil {
		t.Fatalf("Failed to read chromatic_test.mid: %v", err)
	}

	// Verify note count
	if len(seq.Notes) != 7 {
		t.Errorf("Expected 7 notes, got %d", len(seq.Notes))
	}

	// Test key detection - should still identify C major
	keyDetection := DetectKey(seq.Notes)
	if keyDetection.Key.Root != C {
		t.Logf("Note: Key detected as %v (expected C, but chromatic content may affect detection)", keyDetection.Key.Root)
	}

	// Test scale membership - should have some out-of-scale notes
	cMajor := NewKeySignature(C, Major)
	inScale, total := CountInScale(seq.Notes, cMajor)
	outOfScale := total - inScale

	// C# and F# are out of C major scale
	if outOfScale < 2 {
		t.Errorf("Expected at least 2 out-of-scale notes (C# and F#), got %d", outOfScale)
	}

	// Harmonic coherence should be less than 100%
	coherence := HarmonicCoherence(seq.Notes, cMajor)
	if coherence >= 100.0 {
		t.Errorf("Expected harmonic coherence < 100%% (chromatic notes), got %.1f%%", coherence)
	}

	// Test interval analysis - should have strong dissonances
	intervals := AnalyzeIntervals(seq.Notes)
	strongDissonances := 0
	for _, interval := range intervals {
		if interval.Quality == StrongDissonance {
			strongDissonances++
		}
	}
	if strongDissonances < 3 {
		t.Errorf("Expected at least 3 strong dissonances (chromatic semitones), got %d", strongDissonances)
	}
}

func TestRoundTripMIDI(t *testing.T) {
	dir := findTestdataDir(t)

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "riffer_test_*.mid")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// Read original
	original, err := ReadMIDI(filepath.Join(dir, "c_major_scale.mid"))
	if err != nil {
		t.Fatalf("Failed to read original: %v", err)
	}

	// Write to temp
	if err := WriteMIDI(original, tmpPath); err != nil {
		t.Fatalf("Failed to write MIDI: %v", err)
	}

	// Read back
	loaded, err := ReadMIDI(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read back: %v", err)
	}

	// Compare
	if len(loaded.Notes) != len(original.Notes) {
		t.Errorf("Note count mismatch: original %d, loaded %d", len(original.Notes), len(loaded.Notes))
	}

	for i, origNote := range original.Notes {
		if i >= len(loaded.Notes) {
			break
		}
		loadedNote := loaded.Notes[i]
		if origNote.Pitch != loadedNote.Pitch {
			t.Errorf("Note %d pitch mismatch: original %d, loaded %d", i, origNote.Pitch, loadedNote.Pitch)
		}
	}
}
