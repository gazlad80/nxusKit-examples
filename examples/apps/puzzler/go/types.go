// Package puzzler provides types and implementation for puzzle solving comparison.
package puzzler

import (
	"fmt"
	"sort"
	"strings"
)

// SolverApproach defines the solving approach used.
type SolverApproach string

const (
	// ClipsOnly is pure CLIPS rule-based solving.
	ClipsOnly SolverApproach = "clips_only"
	// LlmOnly is pure LLM reasoning.
	LlmOnly SolverApproach = "llm_only"
	// Hybrid is CLIPS primary with LLM for stuck states.
	Hybrid SolverApproach = "hybrid"
)

// String returns the string representation of the approach.
func (a SolverApproach) String() string {
	return string(a)
}

// Difficulty defines the puzzle difficulty level.
type Difficulty string

const (
	// Easy puzzles have 30-45 filled cells.
	Easy Difficulty = "easy"
	// Medium puzzles have 23-29 filled cells.
	Medium Difficulty = "medium"
	// Hard puzzles have 17-22 filled cells.
	Hard Difficulty = "hard"
)

// FilledCellsRange returns the typical range of filled cells for this difficulty.
func (d Difficulty) FilledCellsRange() (min, max int) {
	switch d {
	case Easy:
		return 30, 45
	case Medium:
		return 23, 29
	case Hard:
		return 17, 22
	default:
		return 0, 81
	}
}

// SudokuPuzzle represents a Sudoku puzzle.
type SudokuPuzzle struct {
	// ID is the unique puzzle identifier.
	ID string `json:"id"`
	// Puzzle is the 9x9 grid with 0 for empty cells.
	Puzzle [9][9]int `json:"puzzle"`
	// Difficulty of the puzzle.
	Difficulty Difficulty `json:"difficulty"`
	// Solution is the known correct solution (optional).
	Solution *[9][9]int `json:"solution,omitempty"`
	// Source is the puzzle attribution.
	Source string `json:"source,omitempty"`
}

// NewSudokuPuzzle creates a new puzzle with the given grid.
func NewSudokuPuzzle(id string, puzzle [9][9]int, difficulty Difficulty) *SudokuPuzzle {
	return &SudokuPuzzle{
		ID:         id,
		Puzzle:     puzzle,
		Difficulty: difficulty,
	}
}

// WithSolution sets the known solution.
func (p *SudokuPuzzle) WithSolution(solution [9][9]int) *SudokuPuzzle {
	p.Solution = &solution
	return p
}

// FilledCount returns the number of filled (non-zero) cells.
func (p *SudokuPuzzle) FilledCount() int {
	count := 0
	for row := range 9 {
		for col := range 9 {
			if p.Puzzle[row][col] != 0 {
				count++
			}
		}
	}
	return count
}

// Validate validates that the puzzle grid is well-formed.
func (p *SudokuPuzzle) Validate() error {
	// Check all values are 0-9
	for row := range 9 {
		for col := range 9 {
			val := p.Puzzle[row][col]
			if val < 0 || val > 9 {
				return fmt.Errorf("invalid value %d at (%d, %d): must be 0-9", val, row, col)
			}
		}
	}

	// Check minimum filled cells
	filled := p.FilledCount()
	if filled < 17 {
		return fmt.Errorf("only %d filled cells: minimum 17 required for unique solution", filled)
	}

	return nil
}

// ValidateSolution validates a solution against the puzzle.
func (p *SudokuPuzzle) ValidateSolution(solution *[9][9]int) error {
	// Check rows
	for row := range 9 {
		seen := make(map[int]bool)
		for col := range 9 {
			val := solution[row][col]
			if val < 1 || val > 9 {
				return fmt.Errorf("invalid value %d in row %d", val, row)
			}
			if seen[val] {
				return fmt.Errorf("duplicate %d in row %d", val, row)
			}
			seen[val] = true
		}
	}

	// Check columns
	for col := range 9 {
		seen := make(map[int]bool)
		for row := range 9 {
			val := solution[row][col]
			if seen[val] {
				return fmt.Errorf("duplicate %d in column %d", val, col)
			}
			seen[val] = true
		}
	}

	// Check 3x3 boxes
	for boxRow := range 3 {
		for boxCol := range 3 {
			seen := make(map[int]bool)
			for dr := range 3 {
				for dc := range 3 {
					val := solution[boxRow*3+dr][boxCol*3+dc]
					if seen[val] {
						return fmt.Errorf("duplicate %d in box (%d, %d)", val, boxRow, boxCol)
					}
					seen[val] = true
				}
			}
		}
	}

	// Verify solution matches puzzle constraints
	for row := range 9 {
		for col := range 9 {
			puzzleVal := p.Puzzle[row][col]
			if puzzleVal != 0 && puzzleVal != solution[row][col] {
				return fmt.Errorf("solution conflicts with puzzle at (%d, %d): puzzle has %d, solution has %d",
					row, col, puzzleVal, solution[row][col])
			}
		}
	}

	return nil
}

// SetShape is the shape attribute for Set Game cards.
type SetShape string

const (
	Diamond  SetShape = "diamond"
	Oval     SetShape = "oval"
	Squiggle SetShape = "squiggle"
)

// SetColor is the color attribute for Set Game cards.
type SetColor string

const (
	Red    SetColor = "red"
	Green  SetColor = "green"
	Purple SetColor = "purple"
)

// SetShading is the shading attribute for Set Game cards.
type SetShading string

const (
	Solid   SetShading = "solid"
	Striped SetShading = "striped"
	Empty   SetShading = "empty"
)

// SetCard represents a Set Game card.
type SetCard struct {
	// ID is the card position in hand (0-11).
	ID int `json:"id"`
	// Shape of the card.
	Shape SetShape `json:"shape"`
	// Color of the card.
	Color SetColor `json:"color"`
	// Count is the number of shapes (1, 2, or 3).
	Count int `json:"count"`
	// Shading is the fill pattern.
	Shading SetShading `json:"shading"`
}

// NewSetCard creates a new Set Game card.
func NewSetCard(id int, shape SetShape, color SetColor, count int, shading SetShading) SetCard {
	return SetCard{
		ID:      id,
		Shape:   shape,
		Color:   color,
		Count:   count,
		Shading: shading,
	}
}

// SetHand is a collection of cards for Set Game.
type SetHand struct {
	// ID is the hand identifier.
	ID string `json:"id"`
	// Cards are the 12 cards in play.
	Cards []SetCard `json:"cards"`
	// KnownSets are valid sets for validation.
	KnownSets [][3]int `json:"known_sets,omitempty"`
}

// NewSetHand creates a new hand with the given cards.
func NewSetHand(id string, cards []SetCard) *SetHand {
	return &SetHand{
		ID:    id,
		Cards: cards,
	}
}

// Validate validates the hand has correct structure.
func (h *SetHand) Validate() error {
	if len(h.Cards) != 12 {
		return fmt.Errorf("hand must have exactly 12 cards, has %d", len(h.Cards))
	}

	// Check for duplicate cards
	for i := 0; i < len(h.Cards); i++ {
		for j := i + 1; j < len(h.Cards); j++ {
			if cardsEqual(&h.Cards[i], &h.Cards[j]) {
				return fmt.Errorf("duplicate cards at positions %d and %d", h.Cards[i].ID, h.Cards[j].ID)
			}
		}
	}

	// Validate card attributes
	for _, card := range h.Cards {
		if card.ID < 0 || card.ID > 11 {
			return fmt.Errorf("card ID %d out of range (0-11)", card.ID)
		}
		if card.Count < 1 || card.Count > 3 {
			return fmt.Errorf("card %d count %d out of range (1-3)", card.ID, card.Count)
		}
	}

	return nil
}

func cardsEqual(a, b *SetCard) bool {
	return a.Shape == b.Shape && a.Color == b.Color && a.Count == b.Count && a.Shading == b.Shading
}

// AttributeMatch indicates how an attribute satisfies the set rule.
type AttributeMatch string

const (
	AllSame      AttributeMatch = "all_same"
	AllDifferent AttributeMatch = "all_different"
)

// SetValidation holds validation details for a set.
type SetValidation struct {
	// Shape attribute validation.
	Shape AttributeMatch `json:"shape"`
	// Color attribute validation.
	Color AttributeMatch `json:"color"`
	// Count attribute validation.
	Count AttributeMatch `json:"count"`
	// Shading attribute validation.
	Shading AttributeMatch `json:"shading"`
}

// ValidSet represents a valid set of three cards.
type ValidSet struct {
	// CardIDs are the IDs of three cards forming a set.
	CardIDs [3]int `json:"card_ids"`
	// Validation details how each attribute satisfies set rule.
	Validation SetValidation `json:"validation"`
}

// CheckSet checks if three cards form a valid set.
func CheckSet(cards [3]*SetCard) *ValidSet {
	shapeMatch := checkAttribute3(cards[0].Shape, cards[1].Shape, cards[2].Shape)
	if shapeMatch == nil {
		return nil
	}

	colorMatch := checkAttribute3(cards[0].Color, cards[1].Color, cards[2].Color)
	if colorMatch == nil {
		return nil
	}

	countMatch := checkAttribute3(cards[0].Count, cards[1].Count, cards[2].Count)
	if countMatch == nil {
		return nil
	}

	shadingMatch := checkAttribute3(cards[0].Shading, cards[1].Shading, cards[2].Shading)
	if shadingMatch == nil {
		return nil
	}

	return &ValidSet{
		CardIDs: [3]int{cards[0].ID, cards[1].ID, cards[2].ID},
		Validation: SetValidation{
			Shape:   *shapeMatch,
			Color:   *colorMatch,
			Count:   *countMatch,
			Shading: *shadingMatch,
		},
	}
}

func checkAttribute3[T comparable](a, b, c T) *AttributeMatch {
	if a == b && b == c {
		result := AllSame
		return &result
	}
	if a != b && b != c && a != c {
		result := AllDifferent
		return &result
	}
	return nil // Invalid: two same, one different
}

// PerformanceMetrics holds performance metrics for a solver run.
type PerformanceMetrics struct {
	// Approach used for solving.
	Approach SolverApproach `json:"approach"`
	// PuzzleID is the identifier of puzzle solved.
	PuzzleID string `json:"puzzle_id"`
	// Correct indicates whether solution is correct.
	Correct bool `json:"correct"`
	// TimeMS is the total solve time in milliseconds.
	TimeMS int64 `json:"time_ms"`
	// Iterations is the number of CLIPS rule firings or LLM reasoning steps.
	Iterations int64 `json:"iterations"`
	// LLMCalls is the number of LLM API calls.
	LLMCalls int64 `json:"llm_calls"`
	// TokensUsed is the total tokens consumed.
	TokensUsed int64 `json:"tokens_used"`
	// MemoryPeakKB is the peak memory usage in kilobytes (optional).
	MemoryPeakKB *int64 `json:"memory_peak_kb,omitempty"`
}

// ComparisonReport is an aggregated comparison across approaches.
type ComparisonReport struct {
	// PuzzleID is the puzzle that was solved.
	PuzzleID string `json:"puzzle_id"`
	// Results are metrics for each approach.
	Results []PerformanceMetrics `json:"results"`
	// Winner is the best approach by composite score.
	Winner *SolverApproach `json:"winner,omitempty"`
	// Summary is a human-readable comparison.
	Summary string `json:"summary"`
}

// DetermineWinner determines the winning approach based on correctness, tokens, and time.
func (r *ComparisonReport) DetermineWinner() {
	// Filter to only correct solutions
	var correct []PerformanceMetrics
	for _, result := range r.Results {
		if result.Correct {
			correct = append(correct, result)
		}
	}

	if len(correct) == 0 {
		r.Winner = nil
		return
	}

	// Sort by tokens (ascending), then by time (ascending)
	sort.Slice(correct, func(i, j int) bool {
		if correct[i].TokensUsed != correct[j].TokensUsed {
			return correct[i].TokensUsed < correct[j].TokensUsed
		}
		return correct[i].TimeMS < correct[j].TimeMS
	})

	r.Winner = &correct[0].Approach
}

// GenerateSummary generates a summary string.
func (r *ComparisonReport) GenerateSummary() {
	var parts []string

	for _, result := range r.Results {
		status := "✓"
		if !result.Correct {
			status = "✗"
		}
		parts = append(parts, fmt.Sprintf("%s %s in %dms (%d tokens)",
			result.Approach, status, result.TimeMS, result.TokensUsed))
	}

	if r.Winner != nil {
		parts = append(parts, fmt.Sprintf("Winner: %s", *r.Winner))
	}

	r.Summary = ""
	for i, part := range parts {
		if i > 0 {
			r.Summary += ". "
		}
		r.Summary += part
	}
}

// SudokuSolveResult contains the result of solving a Sudoku puzzle.
type SudokuSolveResult struct {
	// Solution is the solved grid (or partial if failed)
	Solution [9][9]int
	// Solved indicates whether the puzzle was completely solved
	Solved bool
	// Iterations is the number of rule firings/steps
	Iterations int64
	// LLMCalls is the number of LLM API calls (for hybrid/llm-only)
	LLMCalls int64
	// TokensUsed is the total tokens consumed
	TokensUsed int64
}

// ValidateSudokuSolution validates a solution against puzzle constraints.
func ValidateSudokuSolution(puzzle *SudokuPuzzle, solution *[9][9]int) bool {
	return puzzle.ValidateSolution(solution) == nil
}

func sudokuIsSolved(grid *[9][9]int) bool {
	for r := range 9 {
		for c := range 9 {
			if grid[r][c] < 1 || grid[r][c] > 9 {
				return false
			}
		}
	}
	return true
}

// FormatComparisonTable formats a ComparisonReport as a text table.
func FormatComparisonTable(report *ComparisonReport) string {
	var sb strings.Builder

	sb.WriteString("┌────────────┬─────────┬────────────┬────────┬─────────┐\n")
	sb.WriteString("│ Approach   │ Correct │ Time (ms)  │ Tokens │ Winner  │\n")
	sb.WriteString("├────────────┼─────────┼────────────┼────────┼─────────┤\n")

	for _, result := range report.Results {
		correct := "✗"
		if result.Correct {
			correct = "✓"
		}
		winner := ""
		if report.Winner != nil && *report.Winner == result.Approach {
			winner = "★"
		}
		sb.WriteString(fmt.Sprintf("│ %-10s │ %7s │ %10d │ %6d │ %7s │\n",
			result.Approach, correct, result.TimeMS, result.TokensUsed, winner))
	}

	sb.WriteString("└────────────┴─────────┴────────────┴────────┴─────────┘\n")

	return sb.String()
}
