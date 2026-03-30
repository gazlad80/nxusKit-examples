// Package riffer provides music sequence analysis and transformation.
package riffer

import "fmt"

// MidiNote represents a MIDI note number (0-127)
type MidiNote uint8

// Ticks represents duration in MIDI ticks
type Ticks uint32

// Velocity represents attack velocity (0-127)
type Velocity uint8

// PitchClass represents one of the 12 chromatic pitch classes
type PitchClass int

const (
	C PitchClass = iota
	Cs
	D
	Ds
	E
	F
	Fs
	G
	Gs
	A
	As
	B
)

// AllPitchClasses returns all pitch classes in chromatic order
var AllPitchClasses = []PitchClass{C, Cs, D, Ds, E, F, Fs, G, Gs, A, As, B}

// PitchClassNames maps pitch classes to their string representations
var PitchClassNames = map[PitchClass]string{
	C: "C", Cs: "C#", D: "D", Ds: "D#", E: "E", F: "F",
	Fs: "F#", G: "G", Gs: "G#", A: "A", As: "A#", B: "B",
}

// String returns the string representation of a pitch class
func (pc PitchClass) String() string {
	if name, ok := PitchClassNames[pc]; ok {
		return name
	}
	return "?"
}

// PitchClassFromMidi converts a MIDI note number to its pitch class
func PitchClassFromMidi(note MidiNote) PitchClass {
	return PitchClass(note % 12)
}

// ToSemitone returns the semitone offset from C
func (pc PitchClass) ToSemitone() int {
	return int(pc)
}

// ParsePitchClass parses a pitch class from a string
func ParsePitchClass(s string) (PitchClass, bool) {
	switch s {
	case "C", "c":
		return C, true
	case "Cs", "C#", "Db", "cs", "c#", "db":
		return Cs, true
	case "D", "d":
		return D, true
	case "Ds", "D#", "Eb", "ds", "d#", "eb":
		return Ds, true
	case "E", "e":
		return E, true
	case "F", "f":
		return F, true
	case "Fs", "F#", "Gb", "fs", "f#", "gb":
		return Fs, true
	case "G", "g":
		return G, true
	case "Gs", "G#", "Ab", "gs", "g#", "ab":
		return Gs, true
	case "A", "a":
		return A, true
	case "As", "A#", "Bb", "as", "a#", "bb":
		return As, true
	case "B", "b":
		return B, true
	default:
		return C, false
	}
}

// Mode represents a musical mode (scale type)
type Mode int

const (
	Major Mode = iota
	Minor      // Natural minor (Aeolian)
	Dorian
	Phrygian
	Lydian
	Mixolydian
	Aeolian
	Locrian
)

// ModeNames maps modes to their string representations
var ModeNames = map[Mode]string{
	Major:      "Major",
	Minor:      "Minor",
	Dorian:     "Dorian",
	Phrygian:   "Phrygian",
	Lydian:     "Lydian",
	Mixolydian: "Mixolydian",
	Aeolian:    "Aeolian",
	Locrian:    "Locrian",
}

func (m Mode) String() string {
	if name, ok := ModeNames[m]; ok {
		return name
	}
	return "Unknown"
}

// KeySignature represents a musical key (root + mode)
type KeySignature struct {
	Root PitchClass `json:"root"`
	Mode Mode       `json:"mode"`
}

// NewKeySignature creates a new key signature
func NewKeySignature(root PitchClass, mode Mode) KeySignature {
	return KeySignature{Root: root, Mode: mode}
}

func (k KeySignature) String() string {
	return k.Root.String() + " " + k.Mode.String()
}

// TimeSignature represents a time signature
type TimeSignature struct {
	Numerator   uint8 `json:"numerator"`
	Denominator uint8 `json:"denominator"`
}

// DefaultTimeSignature returns 4/4 time
func DefaultTimeSignature() TimeSignature {
	return TimeSignature{Numerator: 4, Denominator: 4}
}

func (ts TimeSignature) String() string {
	return fmt.Sprintf("%d/%d", ts.Numerator, ts.Denominator)
}

// Note represents a single musical note
type Note struct {
	Pitch     MidiNote `json:"pitch"`
	Duration  Ticks    `json:"duration"`
	Velocity  Velocity `json:"velocity"`
	StartTick uint64   `json:"start_tick"`
}

// NewNote creates a new note
func NewNote(pitch MidiNote, duration Ticks, velocity Velocity, startTick uint64) Note {
	return Note{
		Pitch:     pitch,
		Duration:  duration,
		Velocity:  velocity,
		StartTick: startTick,
	}
}

// PitchClass returns the pitch class of this note
func (n Note) PitchClass() PitchClass {
	return PitchClassFromMidi(n.Pitch)
}

// Octave returns the octave of this note (-1 to 9)
func (n Note) Octave() int {
	return int(n.Pitch)/12 - 1
}

// EndTick returns the end position (start + duration)
func (n Note) EndTick() uint64 {
	return n.StartTick + uint64(n.Duration)
}

// Context represents musical context for a sequence
type Context struct {
	KeySignature    *KeySignature `json:"key_signature,omitempty"`
	TimeSignature   TimeSignature `json:"time_signature"`
	Tempo           uint16        `json:"tempo"`
	TicksPerQuarter uint16        `json:"ticks_per_quarter"`
}

// DefaultContext returns a default context (4/4, 120 BPM, 480 PPQN)
func DefaultContext() Context {
	return Context{
		KeySignature:    nil,
		TimeSignature:   DefaultTimeSignature(),
		Tempo:           120,
		TicksPerQuarter: 480,
	}
}

// Metadata contains optional metadata about a sequence
type Metadata struct {
	Created    *string `json:"created,omitempty"`
	Author     *string `json:"author,omitempty"`
	SourceFile *string `json:"source_file,omitempty"`
	Version    string  `json:"version"`
}

// DefaultMetadata returns default metadata
func DefaultMetadata() Metadata {
	return Metadata{Version: "1.0"}
}

// Sequence represents a musical sequence (collection of notes with context)
type Sequence struct {
	ID       string    `json:"id"`
	Name     *string   `json:"name,omitempty"`
	Notes    []Note    `json:"notes"`
	Context  Context   `json:"context"`
	Metadata *Metadata `json:"metadata,omitempty"`
}

// NewSequence creates a new sequence
func NewSequence(id string, notes []Note) *Sequence {
	return &Sequence{
		ID:      id,
		Notes:   notes,
		Context: DefaultContext(),
	}
}

// WithContext sets the context
func (s *Sequence) WithContext(ctx Context) *Sequence {
	s.Context = ctx
	return s
}

// WithName sets the name
func (s *Sequence) WithName(name string) *Sequence {
	s.Name = &name
	return s
}

// IsEmpty returns true if the sequence has no notes
func (s *Sequence) IsEmpty() bool {
	return len(s.Notes) == 0
}

// Len returns the number of notes
func (s *Sequence) Len() int {
	return len(s.Notes)
}

// TotalDuration returns the total duration in ticks
func (s *Sequence) TotalDuration() uint64 {
	var maxEnd uint64
	for _, n := range s.Notes {
		if end := n.EndTick(); end > maxEnd {
			maxEnd = end
		}
	}
	return maxEnd
}

// IntervalQuality represents the consonance/dissonance of an interval
type IntervalQuality int

const (
	PerfectConsonance IntervalQuality = iota
	ImperfectConsonance
	MildDissonance
	StrongDissonance
)

var IntervalQualityNames = map[IntervalQuality]string{
	PerfectConsonance:   "Perfect Consonance",
	ImperfectConsonance: "Imperfect Consonance",
	MildDissonance:      "Mild Dissonance",
	StrongDissonance:    "Strong Dissonance",
}

func (iq IntervalQuality) String() string {
	if name, ok := IntervalQualityNames[iq]; ok {
		return name
	}
	return "Unknown"
}

// Direction represents the direction of an interval
type Direction int

const (
	Ascending Direction = iota
	Descending
	Unison
)

// ContourType represents the melodic contour
type ContourType int

const (
	ContourAscending ContourType = iota
	ContourDescending
	ContourArch
	ContourInverseArch
	ContourWave
	ContourStatic
)

var ContourTypeNames = map[ContourType]string{
	ContourAscending:   "Ascending",
	ContourDescending:  "Descending",
	ContourArch:        "Arch",
	ContourInverseArch: "Inverse Arch",
	ContourWave:        "Wave",
	ContourStatic:      "Static",
}

func (ct ContourType) String() string {
	if name, ok := ContourTypeNames[ct]; ok {
		return name
	}
	return "Unknown"
}

// ScaleType represents the type of scale
type ScaleType int

const (
	ScaleMajor ScaleType = iota
	ScaleNaturalMinor
	ScaleHarmonicMinor
	ScaleMelodicMinor
	ScalePentatonic
	ScaleBlues
	ScaleDorian
	ScalePhrygian
	ScaleLydian
	ScaleMixolydian
	ScaleAeolian
	ScaleLocrian
)

// SuggestionCategory represents the category of a suggestion
type SuggestionCategory int

const (
	CategoryHarmony SuggestionCategory = iota
	CategoryMelody
	CategoryRhythm
	CategoryDynamics
	CategoryResolution
	CategoryStructure
)

// Severity represents the severity of a suggestion
type Severity int

const (
	SeverityInfo Severity = iota
	SeveritySuggestion
	SeverityWarning
)

// OutputFormat represents the output format
type OutputFormat int

const (
	FormatJSON OutputFormat = iota
	FormatMarkdown
	FormatMIDI
	FormatMusicXML
)
