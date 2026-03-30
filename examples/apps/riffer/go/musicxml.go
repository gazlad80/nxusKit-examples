// Package riffer provides music sequence analysis and transformation.
//
// MusicXML file format support using encoding/xml.
package riffer

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// MusicXML structures

// ScorePartwise represents the root MusicXML element
type ScorePartwise struct {
	XMLName  xml.Name `xml:"score-partwise"`
	PartList PartList `xml:"part-list"`
	Parts    []Part   `xml:"part"`
}

// PartList contains score-part elements
type PartList struct {
	ScoreParts []ScorePart `xml:"score-part"`
}

// ScorePart represents a part definition
type ScorePart struct {
	ID       string `xml:"id,attr"`
	PartName string `xml:"part-name,omitempty"`
}

// Part represents a musical part
type Part struct {
	ID       string    `xml:"id,attr"`
	Measures []Measure `xml:"measure"`
}

// Measure represents a musical measure
type Measure struct {
	Number     string          `xml:"number,attr"`
	Attributes *MXMLAttributes `xml:"attributes,omitempty"`
	Directions []MXMLDirection `xml:"direction"`
	Notes      []MXMLNote      `xml:"note"`
}

// MXMLAttributes contains measure attributes
type MXMLAttributes struct {
	Divisions int       `xml:"divisions,omitempty"`
	Key       *MXMLKey  `xml:"key,omitempty"`
	Time      *MXMLTime `xml:"time,omitempty"`
}

// MXMLKey represents a key signature
type MXMLKey struct {
	Fifths int    `xml:"fifths"`
	Mode   string `xml:"mode,omitempty"`
}

// MXMLTime represents a time signature
type MXMLTime struct {
	Beats    string `xml:"beats"`
	BeatType string `xml:"beat-type"`
}

// MXMLDirection contains musical directions
type MXMLDirection struct {
	Sound *MXMLSound `xml:"sound,omitempty"`
}

// MXMLSound contains sound-related info (tempo, etc.)
type MXMLSound struct {
	Tempo float32 `xml:"tempo,attr,omitempty"`
}

// MXMLNote represents a MusicXML note
type MXMLNote struct {
	Pitch    *MXMLPitch `xml:"pitch,omitempty"`
	Rest     *MXMLRest  `xml:"rest,omitempty"`
	Duration int        `xml:"duration"`
	Voice    string     `xml:"voice,omitempty"`
	Type     string     `xml:"type,omitempty"`
}

// MXMLPitch represents note pitch
type MXMLPitch struct {
	Step   string `xml:"step"`
	Alter  *int   `xml:"alter,omitempty"`
	Octave int    `xml:"octave"`
}

// MXMLRest represents a rest
type MXMLRest struct{}

// ReadMusicXML reads a MusicXML file and converts to Sequence
func ReadMusicXML(path string) (*Sequence, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewRifferError(ErrFileNotFound, path)
		}
		return nil, NewRifferErrorWithCause(ErrIO, err.Error(), err)
	}

	var score ScorePartwise
	if err := xml.Unmarshal(data, &score); err != nil {
		return nil, NewRifferErrorWithCause(ErrMusicXMLParse, err.Error(), err)
	}

	// Get first part
	if len(score.Parts) == 0 {
		return nil, NewRifferError(ErrMusicXMLParse, "No parts found in MusicXML")
	}

	part := &score.Parts[0]

	// Parse notes and context
	notes, context, err := parseMusicXMLPart(part)
	if err != nil {
		return nil, err
	}

	if len(notes) == 0 {
		return nil, NewRifferError(ErrEmptySequence, "")
	}

	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]

	seq := NewSequence(generateID(), notes)
	seq.Context = context
	seq.Name = &name

	return seq, nil
}

// parseMusicXMLPart parses a MusicXML part into notes and context
func parseMusicXMLPart(part *Part) ([]Note, Context, error) {
	var notes []Note
	var currentTick uint64
	divisions := 1 // Divisions per quarter note
	tempo := uint16(120)
	timeSig := DefaultTimeSignature()
	var keySig *KeySignature

	for _, measure := range part.Measures {
		// Check attributes
		if measure.Attributes != nil {
			if measure.Attributes.Divisions > 0 {
				divisions = measure.Attributes.Divisions
			}
			if measure.Attributes.Key != nil {
				keySig = fifthsToKey(measure.Attributes.Key.Fifths, measure.Attributes.Key.Mode)
			}
			if measure.Attributes.Time != nil {
				num, _ := strconv.Atoi(measure.Attributes.Time.Beats)
				denom, _ := strconv.Atoi(measure.Attributes.Time.BeatType)
				if num > 0 && denom > 0 {
					timeSig = TimeSignature{
						Numerator:   uint8(num),
						Denominator: uint8(denom),
					}
				}
			}
		}

		// Check directions for tempo
		for _, dir := range measure.Directions {
			if dir.Sound != nil && dir.Sound.Tempo > 0 {
				tempo = uint16(dir.Sound.Tempo)
			}
		}

		// Parse notes
		for _, note := range measure.Notes {
			if note.Rest != nil {
				// Rest - advance time but don't create note
				ticks := uint64(note.Duration * 480 / divisions)
				currentTick += ticks
				continue
			}

			if note.Pitch != nil {
				midiPitch := pitchToMIDI(note.Pitch)
				durationTicks := uint32(note.Duration * 480 / divisions)
				velocity := uint8(80) // Default velocity

				notes = append(notes, NewNote(MidiNote(midiPitch), Ticks(durationTicks), Velocity(velocity), currentTick))
				currentTick += uint64(durationTicks)
			}
		}
	}

	context := Context{
		KeySignature:    keySig,
		TimeSignature:   timeSig,
		Tempo:           tempo,
		TicksPerQuarter: 480,
	}

	return notes, context, nil
}

// pitchToMIDI converts a pitch element to MIDI note number
func pitchToMIDI(pitch *MXMLPitch) uint8 {
	stepSemitone := map[string]int{
		"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11,
	}[pitch.Step]

	alter := 0
	if pitch.Alter != nil {
		alter = *pitch.Alter
	}

	// MIDI note: (octave + 1) * 12 + step + alter
	return uint8((pitch.Octave+1)*12 + stepSemitone + alter)
}

// fifthsToKey converts fifths to KeySignature
func fifthsToKey(fifths int, mode string) *KeySignature {
	modeVal := Major
	if mode == "minor" {
		modeVal = Minor
	}

	// Circle of fifths: F C G D A E B / Gb Db Ab Eb Bb
	rootMap := map[int]PitchClass{
		-7: B,  // Cb -> B
		-6: Fs, // Gb -> F#
		-5: Cs, // Db -> C#
		-4: Gs, // Ab -> G#
		-3: Ds, // Eb -> D#
		-2: As, // Bb -> A#
		-1: F,  // F
		0:  C,  // C
		1:  G,  // G
		2:  D,  // D
		3:  A,  // A
		4:  E,  // E
		5:  B,  // B
		6:  Fs, // F#
		7:  Cs, // C#
	}

	root, ok := rootMap[fifths]
	if !ok {
		root = C // Default to C
	}

	return &KeySignature{Root: root, Mode: modeVal}
}

// WriteMusicXML writes a Sequence to a MusicXML file
func WriteMusicXML(sequence *Sequence, path string) error {
	divisions := 1 // 1 division = 1 quarter note = 480 ticks

	// Convert notes to MusicXML notes
	var mxlNotes []MXMLNote
	for _, note := range sequence.Notes {
		step, alter, octave := midiToPitch(uint8(note.Pitch))
		duration := int(note.Duration) / 480 // Convert to divisions
		if duration < 1 {
			duration = 1
		}

		mxlNote := MXMLNote{
			Pitch: &MXMLPitch{
				Step:   step,
				Octave: octave,
			},
			Duration: duration,
			Voice:    "1",
			Type:     durationToType(uint32(note.Duration)),
		}
		if alter != 0 {
			mxlNote.Pitch.Alter = &alter
		}
		mxlNotes = append(mxlNotes, mxlNote)
	}

	// Build attributes
	attrs := &MXMLAttributes{
		Divisions: divisions,
		Time: &MXMLTime{
			Beats:    fmt.Sprintf("%d", sequence.Context.TimeSignature.Numerator),
			BeatType: fmt.Sprintf("%d", sequence.Context.TimeSignature.Denominator),
		},
	}
	if sequence.Context.KeySignature != nil {
		mode := "major"
		if sequence.Context.KeySignature.Mode == Minor {
			mode = "minor"
		}
		attrs.Key = &MXMLKey{
			Fifths: keyToFifths(sequence.Context.KeySignature),
			Mode:   mode,
		}
	}

	// Build measure
	measure := Measure{
		Number:     "1",
		Attributes: attrs,
		Directions: []MXMLDirection{
			{Sound: &MXMLSound{Tempo: float32(sequence.Context.Tempo)}},
		},
		Notes: mxlNotes,
	}

	// Build part
	partName := ""
	if sequence.Name != nil {
		partName = *sequence.Name
	}
	part := Part{
		ID:       "P1",
		Measures: []Measure{measure},
	}

	// Build score
	score := ScorePartwise{
		PartList: PartList{
			ScoreParts: []ScorePart{
				{ID: "P1", PartName: partName},
			},
		},
		Parts: []Part{part},
	}

	// Serialize to XML
	xmlData, err := xml.MarshalIndent(score, "", "  ")
	if err != nil {
		return NewRifferErrorWithCause(ErrMusicXMLParse, err.Error(), err)
	}

	// Write with XML declaration
	fullXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE score-partwise PUBLIC "-//Recordare//DTD MusicXML 4.0 Partwise//EN" "http://www.musicxml.org/dtds/partwise.dtd">
%s`, string(xmlData))

	if err := os.WriteFile(path, []byte(fullXML), 0644); err != nil {
		return NewRifferErrorWithCause(ErrIO, err.Error(), err)
	}

	return nil
}

// midiToPitch converts MIDI note to pitch components
func midiToPitch(midi uint8) (string, int, int) {
	octave := int(midi/12) - 1
	note := midi % 12

	stepAlter := map[uint8]struct {
		step  string
		alter int
	}{
		0:  {"C", 0},
		1:  {"C", 1},
		2:  {"D", 0},
		3:  {"D", 1},
		4:  {"E", 0},
		5:  {"F", 0},
		6:  {"F", 1},
		7:  {"G", 0},
		8:  {"G", 1},
		9:  {"A", 0},
		10: {"A", 1},
		11: {"B", 0},
	}

	sa := stepAlter[note]
	return sa.step, sa.alter, octave
}

// keyToFifths converts key signature to fifths
func keyToFifths(key *KeySignature) int {
	baseMap := map[PitchClass]int{
		C:  0,
		G:  1,
		D:  2,
		A:  3,
		E:  4,
		B:  5,
		Fs: 6,
		Cs: 7,
		F:  -1,
		As: -2,
		Ds: -3,
		Gs: -4,
	}

	base, ok := baseMap[key.Root]
	if !ok {
		base = 0
	}

	if key.Mode == Minor {
		return base - 3 // Relative minor is 3 fifths back
	}
	return base
}

// durationToType converts duration in ticks to note type
func durationToType(ticks uint32) string {
	switch {
	case ticks >= 1920:
		return "whole"
	case ticks >= 960:
		return "half"
	case ticks >= 480:
		return "quarter"
	case ticks >= 240:
		return "eighth"
	case ticks >= 120:
		return "16th"
	default:
		return "32nd"
	}
}
