// Package arbiter provides the auto-retry LLM pattern with CLIPS validation.
package arbiter

import (
	"encoding/json"
	"fmt"
	"os"
)

// ValidKnobs contains the allowed knob names for adjustments.
var ValidKnobs = []string{
	"temperature",
	"top_p",
	"top_k",
	"presence_penalty",
	"frequency_penalty",
	"max_tokens",
	"thinking_enabled",
}

// IsValidKnob checks if a knob name is valid.
func IsValidKnob(knob string) bool {
	for _, k := range ValidKnobs {
		if k == knob {
			return true
		}
	}
	return false
}

// LoadConfigFromFile loads a SolverConfig from a JSON file.
func LoadConfigFromFile(path string) (*SolverConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config SolverConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
