package puzzler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// LLMSolver uses an LLM provider to solve puzzles
type LLMSolver struct {
	provider nxuskit.LLMProvider
	model    string
}

// NewLLMSolver creates a new LLM-based solver
func NewLLMSolver(provider nxuskit.LLMProvider, model string) *LLMSolver {
	return &LLMSolver{
		provider: provider,
		model:    model,
	}
}

// NewLLMSolverWithFallback creates an LLM solver using the fallback provider
func NewLLMSolverWithFallback() (*LLMSolver, error) {
	fallback := nxuskit.NewProviderFallback()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	provider, err := fallback.GetAvailableProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("no LLM provider available: %w", err)
	}

	return &LLMSolver{
		provider: provider,
		model:    "", // Use provider default
	}, nil
}

// SolveSudoku solves a Sudoku puzzle using the LLM
func (s *LLMSolver) SolveSudoku(ctx context.Context, puzzle *SudokuPuzzle) (*SudokuSolveResult, error) {
	prompt := s.formatSudokuPrompt(puzzle)

	req := &nxuskit.ChatRequest{
		Model: s.model,
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage("You are an expert Sudoku solver. Solve the puzzle and return the complete 9x9 grid as JSON."),
			nxuskit.UserMessage(prompt),
		},
		Temperature: floatPtr(0.0), // Deterministic for puzzle solving
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse the solution from LLM response
	solution, err := s.parseSudokuSolution(resp.Content)
	if err != nil {
		// Return partial result with error
		return &SudokuSolveResult{
			Solution:   puzzle.Puzzle,
			Solved:     false,
			Iterations: 1,
			LLMCalls:   1,
			TokensUsed: int64(resp.Usage.TotalTokens()),
		}, nil
	}

	return &SudokuSolveResult{
		Solution:   solution,
		Solved:     sudokuIsSolved(&solution),
		Iterations: 1,
		LLMCalls:   1,
		TokensUsed: int64(resp.Usage.TotalTokens()),
	}, nil
}

// SolveSet finds valid Set game sets using the LLM
func (s *LLMSolver) SolveSet(ctx context.Context, hand *SetHand) (*SetGameResult, error) {
	prompt := s.formatSetPrompt(hand)

	req := &nxuskit.ChatRequest{
		Model: s.model,
		Messages: []nxuskit.Message{
			nxuskit.SystemMessage("You are an expert Set game player. Find all valid sets in the given hand. Return as JSON array of card index triplets."),
			nxuskit.UserMessage(prompt),
		},
		Temperature: floatPtr(0.0),
	}

	resp, err := s.provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse the sets from LLM response
	sets, err := s.parseSetSolution(resp.Content, hand)
	if err != nil {
		return &SetGameResult{
			Sets:       nil,
			FoundAny:   false,
			Iterations: 1,
			LLMCalls:   1,
			TokensUsed: int64(resp.Usage.TotalTokens()),
		}, nil
	}

	return &SetGameResult{
		Sets:       sets,
		FoundAny:   len(sets) > 0,
		Iterations: 1,
		LLMCalls:   1,
		TokensUsed: int64(resp.Usage.TotalTokens()),
	}, nil
}

func (s *LLMSolver) formatSudokuPrompt(puzzle *SudokuPuzzle) string {
	var sb strings.Builder
	sb.WriteString("Solve this Sudoku puzzle. Each row, column, and 3x3 box must contain digits 1-9.\n\n")
	sb.WriteString("Current puzzle (0 = empty):\n")

	for row := range 9 {
		for col := range 9 {
			if col > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(fmt.Sprintf("%d", puzzle.Puzzle[row][col]))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nReturn the complete solution as a JSON object with a 'grid' field containing a 9x9 array of integers.\n")
	sb.WriteString("Example format: {\"grid\": [[1,2,3,...], [4,5,6,...], ...]}")

	return sb.String()
}

func (s *LLMSolver) formatSetPrompt(hand *SetHand) string {
	var sb strings.Builder
	sb.WriteString("Find all valid Sets in this hand of cards.\n\n")
	sb.WriteString("A valid Set has 3 cards where each attribute (number, color, shape, fill) is either all the same or all different across the 3 cards.\n\n")
	sb.WriteString("Cards:\n")

	for i, card := range hand.Cards {
		sb.WriteString(fmt.Sprintf("%d: %d %s %s %s\n", i, card.Count, card.Shape, card.Color, card.Shading))
	}

	sb.WriteString("\nReturn all valid sets as a JSON object with a 'sets' field containing an array of [i, j, k] triplets (0-indexed).\n")
	sb.WriteString("Example format: {\"sets\": [[0, 1, 2], [3, 5, 7]]}")

	return sb.String()
}

func (s *LLMSolver) parseSudokuSolution(content string) ([9][9]int, error) {
	// Try to extract JSON from the response
	content = extractJSON(content)

	var result struct {
		Grid [9][9]int `json:"grid"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return [9][9]int{}, fmt.Errorf("failed to parse solution: %w", err)
	}

	return result.Grid, nil
}

func (s *LLMSolver) parseSetSolution(content string, hand *SetHand) ([]ValidSet, error) {
	content = extractJSON(content)

	var result struct {
		Sets [][3]int `json:"sets"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse sets: %w", err)
	}

	// Convert to ValidSet format
	var validSets []ValidSet
	for _, indices := range result.Sets {
		if indices[0] >= 0 && indices[0] < len(hand.Cards) &&
			indices[1] >= 0 && indices[1] < len(hand.Cards) &&
			indices[2] >= 0 && indices[2] < len(hand.Cards) {
			validSets = append(validSets, ValidSet{
				CardIDs: [3]int{indices[0], indices[1], indices[2]},
			})
		}
	}

	return validSets, nil
}

// extractJSON tries to find JSON content in a string
func extractJSON(content string) string {
	// Look for JSON object
	start := strings.Index(content, "{")
	if start == -1 {
		return content
	}

	// Find matching closing brace
	depth := 0
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return content[start : i+1]
			}
		}
	}

	return content[start:]
}

func floatPtr(f float64) *float64 {
	return &f
}
