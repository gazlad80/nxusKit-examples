//go:build nxuskit

package puzzler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// ClipsSolver uses CLIPS rules to solve puzzles
type ClipsSolver struct {
	provider nxuskit.LLMProvider
	rulesDir string
}

// NewClipsSolver creates a new CLIPS-based solver
func NewClipsSolver(rulesDir string) (*ClipsSolver, error) {
	provider, err := nxuskit.NewClipsFFIProvider(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIPS provider: %w", err)
	}

	return &ClipsSolver{
		provider: provider,
		rulesDir: rulesDir,
	}, nil
}

// SolveSudoku solves a Sudoku puzzle using CLIPS constraint propagation rules
func (s *ClipsSolver) SolveSudoku(ctx context.Context, puzzle *SudokuPuzzle) (*SudokuSolveResult, error) {
	// Convert puzzle to CLIPS facts
	input := s.buildSudokuInput(puzzle)

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req := &nxuskit.ChatRequest{
		Messages: []nxuskit.Message{
			nxuskit.UserMessage(string(inputJSON)),
		},
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("CLIPS execution failed: %w", err)
	}

	// Parse CLIPS output
	var output clipsOutputWire
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		return nil, fmt.Errorf("failed to parse CLIPS output: %w", err)
	}

	// Extract solution from conclusions
	solution := s.extractSudokuSolution(puzzle, output.Conclusions)

	return &SudokuSolveResult{
		Solution:   solution,
		Solved:     sudokuIsSolved(&solution),
		Iterations: int64(output.Stats.TotalRulesFired),
		LLMCalls:   0,
		TokensUsed: 0,
	}, nil
}

// SolveSet finds valid Set game sets using CLIPS rules
func (s *ClipsSolver) SolveSet(ctx context.Context, hand *SetHand) (*SetGameResult, error) {
	input := s.buildSetInput(hand)

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	req := &nxuskit.ChatRequest{
		Messages: []nxuskit.Message{
			nxuskit.UserMessage(string(inputJSON)),
		},
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("CLIPS execution failed: %w", err)
	}

	var output clipsOutputWire
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		return nil, fmt.Errorf("failed to parse CLIPS output: %w", err)
	}

	sets := s.extractSetSolution(output.Conclusions, hand)

	return &SetGameResult{
		Sets:       sets,
		FoundAny:   len(sets) > 0,
		Iterations: int64(output.Stats.TotalRulesFired),
		LLMCalls:   0,
		TokensUsed: 0,
	}, nil
}

func (s *ClipsSolver) buildSudokuInput(puzzle *SudokuPuzzle) clipsInputWire {
	var facts []clipsFactWire

	// Assert each cell as a fact
	for row := range 9 {
		for col := range 9 {
			val := puzzle.Puzzle[row][col]
			facts = append(facts, clipsFactWire{
				Template: "cell",
				Values: map[string]interface{}{
					"row":   row,
					"col":   col,
					"value": val,
					"fixed": val != 0,
				},
			})
		}
	}

	inc, der := true, true
	maxR := int64(10000)
	return clipsInputWire{
		Facts: facts,
		Config: &clipsRequestConfigWire{
			IncludeTrace:   &inc,
			MaxRules:       &maxR,
			DerivedOnlyNew: &der,
		},
	}
}

func (s *ClipsSolver) buildSetInput(hand *SetHand) clipsInputWire {
	var facts []clipsFactWire

	for i, card := range hand.Cards {
		facts = append(facts, clipsFactWire{
			Template: "card",
			Values: map[string]interface{}{
				"index":   i,
				"count":   card.Count,
				"color":   string(card.Color),
				"shape":   string(card.Shape),
				"shading": string(card.Shading),
			},
		})
	}

	inc, der := true, true
	maxR := int64(1000)
	return clipsInputWire{
		Facts: facts,
		Config: &clipsRequestConfigWire{
			IncludeTrace:   &inc,
			MaxRules:       &maxR,
			DerivedOnlyNew: &der,
		},
	}
}

func (s *ClipsSolver) extractSudokuSolution(puzzle *SudokuPuzzle, conclusions []clipsConclusionWire) [9][9]int {
	// Start with original puzzle
	solution := puzzle.Puzzle

	// Apply derived cell values
	for _, c := range conclusions {
		if c.Template == "cell-value" || c.Template == "solved-cell" {
			if row, ok := c.Values["row"].(float64); ok {
				if col, ok := c.Values["col"].(float64); ok {
					if val, ok := c.Values["value"].(float64); ok {
						solution[int(row)][int(col)] = int(val)
					}
				}
			}
		}
	}

	return solution
}

func (s *ClipsSolver) extractSetSolution(conclusions []clipsConclusionWire, hand *SetHand) []ValidSet {
	var sets []ValidSet

	for _, c := range conclusions {
		if c.Template == "valid-set" {
			var indices [3]int
			if i1, ok := c.Values["card1"].(float64); ok {
				indices[0] = int(i1)
			}
			if i2, ok := c.Values["card2"].(float64); ok {
				indices[1] = int(i2)
			}
			if i3, ok := c.Values["card3"].(float64); ok {
				indices[2] = int(i3)
			}
			// Create ValidSet from indices
			if indices[0] < len(hand.Cards) && indices[1] < len(hand.Cards) && indices[2] < len(hand.Cards) {
				sets = append(sets, ValidSet{
					CardIDs: [3]int{indices[0], indices[1], indices[2]},
				})
			}
		}
	}

	return sets
}
