// Command generate_fixtures creates test MIDI files for the riffer test suite.
//
// Run with: go run ./examples/riffer/cmd/generate_fixtures
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/riffer"
)

func main() {
	// Find testdata directory
	testdataDir := findTestdataDir()
	fmt.Printf("Generating test fixtures in: %s\n", testdataDir)

	// Ensure directories exist
	os.MkdirAll(testdataDir, 0755)
	os.MkdirAll(filepath.Join(testdataDir, "expected"), 0755)

	// Generate test files
	generateCMajorScale(testdataDir)
	generateEMinorRiff(testdataDir)
	generateChromaticTest(testdataDir)

	fmt.Println("Test fixtures generated successfully!")
}

func findTestdataDir() string {
	// Try relative paths
	candidates := []string{
		"testdata",
		"examples/riffer/testdata",
		"../testdata",
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	// Default to current directory
	return "./testdata"
}

func generateCMajorScale(dir string) {
	// C major scale: C D E F G A B C
	notes := []riffer.Note{
		riffer.NewNote(60, 480, 80, 0),    // C4
		riffer.NewNote(62, 480, 80, 480),  // D4
		riffer.NewNote(64, 480, 80, 960),  // E4
		riffer.NewNote(65, 480, 80, 1440), // F4
		riffer.NewNote(67, 480, 80, 1920), // G4
		riffer.NewNote(69, 480, 80, 2400), // A4
		riffer.NewNote(71, 480, 80, 2880), // B4
		riffer.NewNote(72, 480, 80, 3360), // C5
	}

	key := riffer.NewKeySignature(riffer.C, riffer.Major)
	name := "C Major Scale"
	seq := &riffer.Sequence{
		ID:    "c_major_scale",
		Name:  &name,
		Notes: notes,
		Context: riffer.Context{
			KeySignature:    &key,
			TimeSignature:   riffer.TimeSignature{Numerator: 4, Denominator: 4},
			Tempo:           120,
			TicksPerQuarter: 480,
		},
	}

	path := filepath.Join(dir, "c_major_scale.mid")
	if err := riffer.WriteMIDI(seq, path); err != nil {
		fmt.Printf("Error writing %s: %v\n", path, err)
		return
	}
	fmt.Printf("Generated: %s\n", path)
}

func generateEMinorRiff(dir string) {
	// E minor pentatonic riff with varying rhythms and dynamics
	notes := []riffer.Note{
		// Measure 1: E-G-A-B
		riffer.NewNote(64, 240, 90, 0),   // E4 (eighth)
		riffer.NewNote(67, 240, 85, 240), // G4 (eighth)
		riffer.NewNote(69, 480, 80, 480), // A4 (quarter)
		riffer.NewNote(71, 480, 95, 960), // B4 (quarter, accent)
		// Measure 2: D-E-G-E
		riffer.NewNote(74, 240, 75, 1440), // D5 (eighth)
		riffer.NewNote(76, 240, 70, 1680), // E5 (eighth)
		riffer.NewNote(79, 480, 85, 1920), // G5 (quarter)
		riffer.NewNote(76, 960, 90, 2400), // E5 (half, resolution)
		// Measure 3: descending pattern
		riffer.NewNote(74, 240, 80, 3360), // D5
		riffer.NewNote(71, 240, 80, 3600), // B4
		riffer.NewNote(69, 240, 80, 3840), // A4
		riffer.NewNote(67, 240, 80, 4080), // G4
		riffer.NewNote(64, 960, 85, 4320), // E4 (half, tonic resolution)
	}

	key := riffer.NewKeySignature(riffer.E, riffer.Minor)
	name := "E Minor Riff"
	seq := &riffer.Sequence{
		ID:    "e_minor_riff",
		Name:  &name,
		Notes: notes,
		Context: riffer.Context{
			KeySignature:    &key,
			TimeSignature:   riffer.TimeSignature{Numerator: 4, Denominator: 4},
			Tempo:           100,
			TicksPerQuarter: 480,
		},
	}

	path := filepath.Join(dir, "e_minor_riff.mid")
	if err := riffer.WriteMIDI(seq, path); err != nil {
		fmt.Printf("Error writing %s: %v\n", path, err)
		return
	}
	fmt.Printf("Generated: %s\n", path)
}

func generateChromaticTest(dir string) {
	// Chromatic passage with tritones and dissonances
	notes := []riffer.Note{
		riffer.NewNote(60, 480, 80, 0),    // C4
		riffer.NewNote(61, 480, 80, 480),  // C#4 (chromatic)
		riffer.NewNote(62, 480, 80, 960),  // D4
		riffer.NewNote(66, 480, 80, 1440), // F#4 (tritone from C)
		riffer.NewNote(67, 480, 80, 1920), // G4 (resolution)
		riffer.NewNote(71, 480, 80, 2400), // B4 (leading tone)
		riffer.NewNote(72, 480, 80, 2880), // C5 (resolution)
	}

	key := riffer.NewKeySignature(riffer.C, riffer.Major)
	name := "Chromatic Test"
	seq := &riffer.Sequence{
		ID:    "chromatic_test",
		Name:  &name,
		Notes: notes,
		Context: riffer.Context{
			KeySignature:    &key,
			TimeSignature:   riffer.TimeSignature{Numerator: 4, Denominator: 4},
			Tempo:           120,
			TicksPerQuarter: 480,
		},
	}

	path := filepath.Join(dir, "chromatic_test.mid")
	if err := riffer.WriteMIDI(seq, path); err != nil {
		fmt.Printf("Error writing %s: %v\n", path, err)
		return
	}
	fmt.Printf("Generated: %s\n", path)
}
