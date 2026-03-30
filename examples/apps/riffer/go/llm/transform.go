//go:build nxuskit

// Package llm provides LLM-powered features for riffer.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/riffer"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// TransformRequest represents the parsed transformation parameters
type TransformRequest struct {
	Transpose   *TransposeParams  `json:"transpose,omitempty"`
	Tempo       *TempoParams      `json:"tempo,omitempty"`
	Invert      *InvertParams     `json:"invert,omitempty"`
	Retrograde  *RetrogradeParams `json:"retrograde,omitempty"`
	Augment     *AugmentParams    `json:"augment,omitempty"`
	Diminish    *DiminishParams   `json:"diminish,omitempty"`
	KeyChange   *KeyChangeParams  `json:"key_change,omitempty"`
	Description string            `json:"description,omitempty"`
}

// TransposeParams for transpose transformation
type TransposeParams struct {
	Semitones int `json:"semitones"`
}

// TempoParams for tempo change
type TempoParams struct {
	BPM int `json:"bpm"`
}

// InvertParams for inversion
type InvertParams struct {
	Pivot *int `json:"pivot,omitempty"`
}

// RetrogradeParams for retrograde
type RetrogradeParams struct {
	Enabled bool `json:"enabled"`
}

// AugmentParams for augmentation
type AugmentParams struct {
	Factor float64 `json:"factor"`
}

// DiminishParams for diminution
type DiminishParams struct {
	Factor float64 `json:"factor"`
}

// KeyChangeParams for key change
type KeyChangeParams struct {
	Root string `json:"root"`
	Mode string `json:"mode"`
}

const transformSystemPrompt = `You are a music transformation assistant. Given a natural language description of a musical transformation, you will respond with a JSON object containing the transformation parameters.

Available transformations:
- "transpose": Shift all notes by semitones. Use "semitones" field (-12 to +12).
  Examples: "up a major third" = +4, "down a perfect fifth" = -7, "up an octave" = +12
- "tempo": Change the tempo. Use "bpm" field (20-300).
- "invert": Mirror the melody around a pivot pitch. Use "pivot" field (MIDI note, 0-127, or null for first note).
- "retrograde": Reverse the note order. Use "enabled" field (true/false).
- "augment": Stretch durations. Use "factor" field (e.g., 2.0 = twice as long).
- "diminish": Compress durations. Use "factor" field (e.g., 2.0 = half as long).
- "key_change": Change key. Use "root" (C, C#, D, etc.) and "mode" (major/minor) fields.

Respond ONLY with a JSON object. Example:
{"transpose": {"semitones": 7}, "description": "Transposed up a perfect fifth"}

For multiple transformations:
{"transpose": {"semitones": 5}, "tempo": {"bpm": 90}, "description": "Transposed up and slowed down"}

If the prompt is unclear, respond with your best interpretation.`

// TransformWithPrompt transforms a sequence based on a natural language prompt.
// It uses an LLM to interpret the prompt and apply the appropriate transformations.
func TransformWithPrompt(ctx context.Context, seq *riffer.Sequence, prompt string) (string, error) {
	keyStr := "Unknown"
	if seq.Context.KeySignature != nil {
		keyStr = seq.Context.KeySignature.String()
	}

	userPrompt := fmt.Sprintf(
		"Transform request: \"%s\"\n\nCurrent sequence info:\n- Notes: %d\n- Key: %s\n- Tempo: %d BPM",
		prompt,
		len(seq.Notes),
		keyStr,
		seq.Context.Tempo,
	)

	// Call LLM
	response, err := callLLM(ctx, transformSystemPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	// Extract JSON from response (might be in code blocks)
	jsonStr := extractJSON(response)

	// Parse the response
	var req TransformRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w. Response: %s", err, response)
	}

	// Apply transformations
	var applied []string

	if req.Transpose != nil {
		if err := riffer.Transpose(seq, int8(req.Transpose.Semitones)); err != nil {
			return "", fmt.Errorf("transpose failed: %w", err)
		}
		applied = append(applied, fmt.Sprintf("Transposed by %d semitones", req.Transpose.Semitones))
	}

	if req.Tempo != nil {
		if err := riffer.ChangeTempo(seq, uint16(req.Tempo.BPM)); err != nil {
			return "", fmt.Errorf("tempo change failed: %w", err)
		}
		applied = append(applied, fmt.Sprintf("Changed tempo to %d BPM", req.Tempo.BPM))
	}

	if req.Invert != nil {
		var pivot *uint8
		if req.Invert.Pivot != nil {
			p := uint8(*req.Invert.Pivot)
			pivot = &p
		}
		if err := riffer.Invert(seq, pivot); err != nil {
			return "", fmt.Errorf("invert failed: %w", err)
		}
		applied = append(applied, "Inverted melody")
	}

	if req.Retrograde != nil && req.Retrograde.Enabled {
		if err := riffer.Retrograde(seq); err != nil {
			return "", fmt.Errorf("retrograde failed: %w", err)
		}
		applied = append(applied, "Applied retrograde")
	}

	if req.Augment != nil {
		if err := riffer.Augment(seq, float32(req.Augment.Factor)); err != nil {
			return "", fmt.Errorf("augment failed: %w", err)
		}
		applied = append(applied, fmt.Sprintf("Augmented by factor %v", req.Augment.Factor))
	}

	if req.Diminish != nil {
		if err := riffer.Diminish(seq, float32(req.Diminish.Factor)); err != nil {
			return "", fmt.Errorf("diminish failed: %w", err)
		}
		applied = append(applied, fmt.Sprintf("Diminished by factor %v", req.Diminish.Factor))
	}

	if req.KeyChange != nil {
		targetKey, ok := riffer.ParseKey(req.KeyChange.Root + modeToSuffix(req.KeyChange.Mode))
		if !ok {
			return "", fmt.Errorf("invalid key: %s %s", req.KeyChange.Root, req.KeyChange.Mode)
		}
		if err := riffer.KeyChange(seq, *targetKey); err != nil {
			return "", fmt.Errorf("key change failed: %w", err)
		}
		applied = append(applied, fmt.Sprintf("Changed key to %s", targetKey.String()))
	}

	// Build description
	if len(applied) == 0 {
		if req.Description != "" {
			return req.Description, nil
		}
		return "No transformations applied", nil
	}

	return strings.Join(applied, "; "), nil
}

func modeToSuffix(mode string) string {
	mode = strings.ToLower(mode)
	if strings.Contains(mode, "minor") {
		return "m"
	}
	return ""
}

func extractJSON(response string) string {
	// Try to find JSON in code blocks
	if idx := strings.Index(response, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(response[start:], "```"); end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	if idx := strings.Index(response, "```"); idx != -1 {
		start := idx + 3
		if end := strings.Index(response[start:], "```"); end != -1 {
			content := strings.TrimSpace(response[start : start+end])
			// Skip language identifier if present
			if newline := strings.Index(content, "\n"); newline != -1 {
				return strings.TrimSpace(content[newline+1:])
			}
		}
	}
	return strings.TrimSpace(response)
}

func callLLM(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// Try providers in order: Claude, OpenAI, Ollama
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		return callClaude(ctx, apiKey, systemPrompt, userPrompt)
	}

	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		return callOpenAI(ctx, apiKey, systemPrompt, userPrompt)
	}

	// Try Ollama as fallback
	return callOllama(ctx, systemPrompt, userPrompt)
}

func callClaude(ctx context.Context, apiKey, systemPrompt, userPrompt string) (string, error) {
	provider, err := nxuskit.NewClaudeFFIProvider(
		nxuskit.WithClaudeAPIKey(apiKey),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Claude provider: %w", err)
	}

	req := &nxuskit.ChatRequest{
		Model: "claude-haiku-4-5-20251001",
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage(systemPrompt),
			nxuskit.UserMessage(userPrompt),
		},
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("Claude API error: %w", err)
	}

	return resp.Content, nil
}

func callOpenAI(ctx context.Context, apiKey, systemPrompt, userPrompt string) (string, error) {
	provider, err := nxuskit.NewOpenAIFFIProvider(
		nxuskit.WithOpenAIAPIKey(apiKey),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create OpenAI provider: %w", err)
	}

	req := &nxuskit.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage(systemPrompt),
			nxuskit.UserMessage(userPrompt),
		},
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("OpenAI API error: %w", err)
	}

	return resp.Content, nil
}

func callOllama(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	provider, err := nxuskit.NewOllamaFFIProvider()
	if err != nil {
		return "", fmt.Errorf("failed to create Ollama provider: %w", err)
	}

	req := &nxuskit.ChatRequest{
		Model: "llama3.2",
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage(systemPrompt),
			nxuskit.UserMessage(userPrompt),
		},
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("no LLM available. Set ANTHROPIC_API_KEY or OPENAI_API_KEY, or run Ollama locally. Error: %w", err)
	}

	return resp.Content, nil
}
