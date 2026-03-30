// Package riffer provides music sequence analysis and transformation.
//
// MIDI file format support using gomidi/midi/v2.
package riffer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"
)

// ReadMIDI reads a MIDI file and converts to Sequence
func ReadMIDI(path string) (*Sequence, error) {
	s, err := smf.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, NewRifferError(ErrFileNotFound, path)
		}
		return nil, NewRifferErrorWithCause(ErrMidiParse, err.Error(), err)
	}

	// Extract timing information
	ticksPerQuarter := uint16(480) // Default
	if mt, ok := s.TimeFormat.(smf.MetricTicks); ok {
		ticksPerQuarter = uint16(mt)
	}

	// Check for multi-track and warn
	if len(s.Tracks) > 1 {
		fmt.Fprintf(os.Stderr, "Warning: Multi-track MIDI file detected. Using first track only. %d tracks ignored.\n", len(s.Tracks)-1)
	}

	// Find the first track with note data
	track := findNoteTrack(s.Tracks)
	if track == nil {
		return nil, NewRifferError(ErrMidiParse, "No note data found in MIDI file")
	}

	// Parse notes and context from track
	notes, context, err := parseTrack(track, ticksPerQuarter)
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
	sourceFile := path
	seq.Metadata = &Metadata{
		SourceFile: &sourceFile,
		Version:    "1.0",
	}

	return seq, nil
}

// findNoteTrack finds the first track that contains note events
func findNoteTrack(tracks []smf.Track) smf.Track {
	for _, track := range tracks {
		for _, ev := range track {
			if ev.Message.Is(midi.NoteOnMsg) {
				return track
			}
		}
	}
	return nil
}

// parseTrack parses a MIDI track into notes and context
func parseTrack(track smf.Track, ticksPerQuarter uint16) ([]Note, Context, error) {
	var notes []Note
	activeNotes := make(map[uint8]struct {
		startTick uint64
		velocity  uint8
	})
	var currentTick uint64
	tempo := uint16(120)
	timeSig := DefaultTimeSignature()

	for _, ev := range track {
		currentTick += uint64(ev.Delta)

		var channel, key, vel uint8
		if ev.Message.GetNoteOn(&channel, &key, &vel) {
			if vel > 0 {
				// Note on
				activeNotes[key] = struct {
					startTick uint64
					velocity  uint8
				}{currentTick, vel}
			} else {
				// Note off (velocity 0)
				if active, ok := activeNotes[key]; ok {
					duration := uint32(currentTick - active.startTick)
					notes = append(notes, NewNote(MidiNote(key), Ticks(duration), Velocity(active.velocity), active.startTick))
					delete(activeNotes, key)
				}
			}
		} else if ev.Message.GetNoteOff(&channel, &key, &vel) {
			if active, ok := activeNotes[key]; ok {
				duration := uint32(currentTick - active.startTick)
				notes = append(notes, NewNote(MidiNote(key), Ticks(duration), Velocity(active.velocity), active.startTick))
				delete(activeNotes, key)
			}
		} else if ev.Message.IsMeta() {
			// Check for tempo
			var bpm float64
			if ev.Message.GetMetaTempo(&bpm) {
				tempo = uint16(bpm)
			}
			// Check for time signature
			var num, denom, clocksPerClick, notesPerQuarter uint8
			if ev.Message.GetMetaTimeSig(&num, &denom, &clocksPerClick, &notesPerQuarter) {
				timeSig = TimeSignature{
					Numerator:   num,
					Denominator: 1 << denom,
				}
			}
		}
	}

	// Sort notes by start time
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].StartTick < notes[j].StartTick
	})

	context := Context{
		KeySignature:    nil,
		TimeSignature:   timeSig,
		Tempo:           tempo,
		TicksPerQuarter: ticksPerQuarter,
	}

	return notes, context, nil
}

// WriteMIDI writes a Sequence to a MIDI file
func WriteMIDI(sequence *Sequence, path string) error {
	tpq := sequence.Context.TicksPerQuarter

	// Create SMF
	s := smf.New()
	s.TimeFormat = smf.MetricTicks(tpq)

	var track smf.Track

	// Add tempo meta event
	track.Add(0, smf.MetaTempo(float64(sequence.Context.Tempo)))

	// Add time signature
	ts := &sequence.Context.TimeSignature
	denomPower := uint8(0)
	for d := ts.Denominator; d > 1; d >>= 1 {
		denomPower++
	}
	track.Add(0, smf.MetaTimeSig(ts.Numerator, denomPower, 24, 8))

	// Collect all note events with absolute times
	type noteEvent struct {
		tick  uint64
		isOn  bool
		pitch uint8
		vel   uint8
	}
	var events []noteEvent

	for _, note := range sequence.Notes {
		// Note on
		events = append(events, noteEvent{
			tick:  note.StartTick,
			isOn:  true,
			pitch: uint8(note.Pitch),
			vel:   uint8(note.Velocity),
		})
		// Note off
		events = append(events, noteEvent{
			tick:  note.EndTick(),
			isOn:  false,
			pitch: uint8(note.Pitch),
			vel:   0,
		})
	}

	// Sort by time
	sort.Slice(events, func(i, j int) bool {
		return events[i].tick < events[j].tick
	})

	// Convert to delta times
	var lastTick uint64
	for _, ev := range events {
		delta := uint32(ev.tick - lastTick)
		if ev.isOn {
			track.Add(delta, midi.NoteOn(0, ev.pitch, ev.vel))
		} else {
			track.Add(delta, midi.NoteOff(0, ev.pitch))
		}
		lastTick = ev.tick
	}

	// Close track (adds end of track marker)
	track.Close(0)

	s.Tracks = append(s.Tracks, track)

	// Write to file
	err := s.WriteFile(path)
	if err != nil {
		return NewRifferErrorWithCause(ErrIO, err.Error(), err)
	}

	return nil
}

// generateID generates a unique ID (simple implementation)
func generateID() string {
	return fmt.Sprintf("%d", os.Getpid())
}
