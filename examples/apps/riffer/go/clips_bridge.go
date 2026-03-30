//go:build nxuskit

// Package riffer provides music sequence analysis and transformation.
//
// CLIPS Rule Engine Bridge for Riffer.
// Provides integration with the CLIPS expert system for scoring adjustments
// and context-aware suggestions.
package riffer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// ScoringAdjustment represents a scoring adjustment from CLIPS rules
type ScoringAdjustment struct {
	Dimension  string  `json:"dimension"`
	Adjustment float32 `json:"adjustment"`
	Reason     string  `json:"reason"`
	RuleName   string  `json:"rule_name"`
}

// ClipsSuggestion represents a suggestion from CLIPS rules
type ClipsSuggestion struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Message     string `json:"message"`
	RuleName    string `json:"rule_name"`
	NoteIndices []int  `json:"note_indices,omitempty"`
}

// ClipsResult contains the results from CLIPS rule execution
type ClipsResult struct {
	Adjustments     []ScoringAdjustment `json:"adjustments"`
	Suggestions     []ClipsSuggestion   `json:"suggestions"`
	RulesFired      uint32              `json:"rules_fired"`
	ExecutionTimeMs uint64              `json:"execution_time_ms"`
}

// ClipsRuleEngine provides CLIPS integration for music analysis
type ClipsRuleEngine struct {
	rulesDir  string
	available bool
}

// NewClipsRuleEngine creates a new CLIPS rule engine
func NewClipsRuleEngine(rulesDir string) (*ClipsRuleEngine, error) {
	// Check if rules directory exists
	info, err := os.Stat(rulesDir)
	available := err == nil && info.IsDir()

	if !available {
		fmt.Fprintf(os.Stderr, "Warning: CLIPS rules directory not found at %s. Using deterministic scoring only.\n", rulesDir)
	}

	return &ClipsRuleEngine{
		rulesDir:  rulesDir,
		available: available,
	}, nil
}

// IsAvailable returns true if CLIPS is available
func (e *ClipsRuleEngine) IsAvailable() bool {
	return e.available
}

// SequenceToFacts converts a sequence to CLIPS facts JSON
func (e *ClipsRuleEngine) SequenceToFacts(sequence *Sequence) map[string]interface{} {
	facts := make([]map[string]interface{}, 0)

	// Add context fact
	contextFact := e.contextToFact(&sequence.Context)
	facts = append(facts, contextFact)

	// Add note facts
	for idx, note := range sequence.Notes {
		noteFact := e.noteToFact(&note, idx)
		facts = append(facts, noteFact)
	}

	// Add interval facts
	intervals := AnalyzeIntervals(sequence.Notes)
	for _, interval := range intervals {
		intervalFact := e.intervalToFact(&interval)
		facts = append(facts, intervalFact)
	}

	// Add scale membership facts
	if sequence.Context.KeySignature != nil {
		key := NewKeySignature(sequence.Context.KeySignature.Root, sequence.Context.KeySignature.Mode)
		for idx, note := range sequence.Notes {
			pc := PitchClassFromMidi(note.Pitch)
			inScale := IsInScale(pc, key)
			membershipFact := map[string]interface{}{
				"template": "scale-membership",
				"values": map[string]interface{}{
					"note-index":  idx,
					"pitch-class": pc.String(),
					"in-scale":    inScale,
				},
			}
			facts = append(facts, membershipFact)
		}
	}

	// Add dissonance facts for strong dissonances
	for _, interval := range intervals {
		if interval.Quality == StrongDissonance {
			dissonanceFact := map[string]interface{}{
				"template": "dissonance",
				"values": map[string]interface{}{
					"note-index":         interval.FromIndex,
					"next-index":         interval.ToIndex,
					"interval-semitones": interval.Semitones,
					"needs-resolution":   true,
				},
			}
			facts = append(facts, dissonanceFact)
		}
	}

	return map[string]interface{}{
		"facts": facts,
		"config": map[string]interface{}{
			"include_trace": true,
			"max_rules":     100,
		},
	}
}

// contextToFact converts context to a CLIPS fact
func (e *ClipsRuleEngine) contextToFact(context *Context) map[string]interface{} {
	keyRoot := "unknown"
	keyMode := "unknown"

	if context.KeySignature != nil {
		keyRoot = context.KeySignature.Root.String()
		switch context.KeySignature.Mode {
		case Major:
			keyMode = "major"
		case Minor:
			keyMode = "minor"
		default:
			keyMode = "other"
		}
	}

	return map[string]interface{}{
		"template": "context",
		"values": map[string]interface{}{
			"key-root":          map[string]interface{}{"symbol": keyRoot},
			"key-mode":          map[string]interface{}{"symbol": keyMode},
			"time-num":          context.TimeSignature.Numerator,
			"time-denom":        context.TimeSignature.Denominator,
			"tempo":             context.Tempo,
			"ticks-per-quarter": context.TicksPerQuarter,
		},
	}
}

// noteToFact converts a note to a CLIPS fact
func (e *ClipsRuleEngine) noteToFact(note *Note, index int) map[string]interface{} {
	pc := PitchClassFromMidi(note.Pitch)
	octave := int(note.Pitch)/12 - 1

	return map[string]interface{}{
		"template": "note",
		"values": map[string]interface{}{
			"index":       index,
			"pitch":       note.Pitch,
			"pitch-class": map[string]interface{}{"symbol": pc.String()},
			"octave":      octave,
			"duration":    note.Duration,
			"velocity":    note.Velocity,
			"start-tick":  note.StartTick,
		},
	}
}

// intervalToFact converts an interval to a CLIPS fact
func (e *ClipsRuleEngine) intervalToFact(interval *IntervalInfo) map[string]interface{} {
	var quality string
	switch interval.Quality {
	case PerfectConsonance:
		quality = "perfect-consonance"
	case ImperfectConsonance:
		quality = "imperfect-consonance"
	case MildDissonance:
		quality = "mild-dissonance"
	case StrongDissonance:
		quality = "strong-dissonance"
	}

	var direction string
	switch interval.Direction {
	case Ascending:
		direction = "ascending"
	case Descending:
		direction = "descending"
	case Unison:
		direction = "unison"
	}

	return map[string]interface{}{
		"template": "interval",
		"values": map[string]interface{}{
			"from-index": interval.FromIndex,
			"to-index":   interval.ToIndex,
			"semitones":  interval.Semitones,
			"quality":    map[string]interface{}{"symbol": quality},
			"direction":  map[string]interface{}{"symbol": direction},
		},
	}
}

// ExtractAdjustments extracts scoring adjustments from CLIPS output
func (e *ClipsRuleEngine) ExtractAdjustments(output map[string]interface{}) []ScoringAdjustment {
	var adjustments []ScoringAdjustment

	conclusions, ok := output["conclusions"].([]interface{})
	if !ok {
		return adjustments
	}

	for _, conclusion := range conclusions {
		c, ok := conclusion.(map[string]interface{})
		if !ok {
			continue
		}

		template, _ := c["template"].(string)
		if template != "scoring-adjustment" {
			continue
		}

		values, ok := c["values"].(map[string]interface{})
		if !ok {
			continue
		}

		// Handle dimension which might be a symbol (object with "symbol" key) or a string
		dimension := "unknown"
		if d, ok := values["dimension"]; ok {
			if dm, ok := d.(map[string]interface{}); ok {
				if sym, ok := dm["symbol"].(string); ok {
					dimension = sym
				}
			} else if s, ok := d.(string); ok {
				dimension = s
			}
		}

		// Handle amount (the CLIPS template uses 'amount', not 'adjustment')
		adjustment := getFloat32OrDefault(values, "amount", 0)
		if adjustment == 0 {
			adjustment = getFloat32OrDefault(values, "adjustment", 0)
		}

		adj := ScoringAdjustment{
			Dimension:  dimension,
			Adjustment: adjustment,
			Reason:     getStringOrDefault(values, "reason", ""),
			RuleName:   getStringOrDefault(values, "rule-name", "unknown"),
		}
		adjustments = append(adjustments, adj)
	}

	return adjustments
}

// ExtractSuggestions extracts suggestions from CLIPS output
func (e *ClipsRuleEngine) ExtractSuggestions(output map[string]interface{}) []ClipsSuggestion {
	var suggestions []ClipsSuggestion

	conclusions, ok := output["conclusions"].([]interface{})
	if !ok {
		return suggestions
	}

	for _, conclusion := range conclusions {
		c, ok := conclusion.(map[string]interface{})
		if !ok {
			continue
		}

		template, _ := c["template"].(string)
		if template != "suggestion" {
			continue
		}

		values, ok := c["values"].(map[string]interface{})
		if !ok {
			continue
		}

		sug := ClipsSuggestion{
			Category: getStringOrDefault(values, "category", "general"),
			Severity: getStringOrDefault(values, "severity", "info"),
			Message:  getStringOrDefault(values, "message", ""),
			RuleName: getStringOrDefault(values, "rule-name", "unknown"),
		}

		// Extract note indices if present
		if indices, ok := values["note-indices"].([]interface{}); ok {
			for _, idx := range indices {
				if n, ok := idx.(float64); ok {
					sug.NoteIndices = append(sug.NoteIndices, int(n))
				}
			}
		}

		suggestions = append(suggestions, sug)
	}

	return suggestions
}

// Analyze runs CLIPS rules on a sequence and returns results (synchronous wrapper)
func (e *ClipsRuleEngine) Analyze(sequence *Sequence) (*ClipsResult, error) {
	return e.AnalyzeContext(context.Background(), sequence)
}

// AnalyzeContext runs CLIPS rules on a sequence with context support
func (e *ClipsRuleEngine) AnalyzeContext(ctx context.Context, sequence *Sequence) (*ClipsResult, error) {
	start := time.Now()

	if !e.available {
		return &ClipsResult{
			Adjustments:     []ScoringAdjustment{},
			Suggestions:     []ClipsSuggestion{},
			RulesFired:      0,
			ExecutionTimeMs: uint64(time.Since(start).Milliseconds()),
		}, nil
	}

	// Convert sequence to facts
	factsJSON := e.SequenceToFacts(sequence)

	// Log facts for debugging if CLIPS_DEBUG is set
	if os.Getenv("CLIPS_DEBUG") != "" {
		data, _ := json.MarshalIndent(factsJSON, "", "  ")
		fmt.Fprintf(os.Stderr, "CLIPS Facts:\n%s\n", string(data))
	}

	// Create CLIPS provider (FFI-backed)
	provider, err := nxuskit.NewClipsFFIProvider(e.rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIPS provider: %w", err)
	}

	// Build the request - model field specifies which rule files to load
	ruleFiles := "templates.clp,music-theory.clp,scoring-adjustments.clp,suggestions.clp"
	factsData, _ := json.Marshal(factsJSON)

	req := &nxuskit.ChatRequest{
		Model: ruleFiles,
		Messages: []nxuskit.Message{
			nxuskit.UserMessage(string(factsData)),
		},
	}

	// Execute CLIPS rules
	resp, err := provider.Chat(ctx, req)
	if err != nil {
		// Check if it's a license error - if so, fall back gracefully
		if errors.Is(err, nxuskit.ErrLicenseRequired) {
			if os.Getenv("CLIPS_DEBUG") != "" {
				fmt.Fprintf(os.Stderr, "CLIPS: License required, using deterministic scoring\n")
			}
			return &ClipsResult{
				Adjustments:     []ScoringAdjustment{},
				Suggestions:     []ClipsSuggestion{},
				RulesFired:      0,
				ExecutionTimeMs: uint64(time.Since(start).Milliseconds()),
			}, nil
		}
		return nil, fmt.Errorf("CLIPS execution failed: %w", err)
	}

	executionTimeMs := uint64(time.Since(start).Milliseconds())

	// Parse the response
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		return nil, fmt.Errorf("failed to parse CLIPS output: %w", err)
	}

	// Extract adjustments and suggestions
	adjustments := e.ExtractAdjustments(output)
	suggestions := e.ExtractSuggestions(output)

	// Get rules fired count
	var rulesFired uint32
	if stats, ok := output["stats"].(map[string]interface{}); ok {
		if rf, ok := stats["total_rules_fired"].(float64); ok {
			rulesFired = uint32(rf)
		}
	}

	return &ClipsResult{
		Adjustments:     adjustments,
		Suggestions:     suggestions,
		RulesFired:      rulesFired,
		ExecutionTimeMs: executionTimeMs,
	}, nil
}

// ApplyAdjustments applies scoring adjustments to dimension scores
func ApplyAdjustments(scores map[string]float32, adjustments []ScoringAdjustment) {
	for _, adj := range adjustments {
		if score, ok := scores[adj.Dimension]; ok {
			newScore := score + adj.Adjustment
			if newScore < 0 {
				newScore = 0
			} else if newScore > 100 {
				newScore = 100
			}
			scores[adj.Dimension] = newScore
		}
	}
}

// Helper functions

func getStringOrDefault(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultVal
}

func getFloat32OrDefault(m map[string]interface{}, key string, defaultVal float32) float32 {
	if v, ok := m[key].(float64); ok {
		return float32(v)
	}
	return defaultVal
}

// LoadRulesDir returns the default rules directory path
func LoadRulesDir() string {
	// Check environment variable first
	if dir := os.Getenv("RIFFER_RULES_DIR"); dir != "" {
		return dir
	}

	// Try relative to executable
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "rules")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}

	// Default to current directory
	return "./rules"
}
