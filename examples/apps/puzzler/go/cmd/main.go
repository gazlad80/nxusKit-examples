// Package main provides a CLI for comparing puzzle solving approaches.
//
// Demonstrates comparing three approaches for solving puzzles:
// - CLIPS-only: Pure rule-based constraint propagation
// - LLM-only: Pure LLM reasoning
// - Hybrid: CLIPS primary with LLM fallback for stuck states
//
// Supports both Sudoku and Set Game puzzles.
//
// ## Interactive Modes
// - --verbose or -v: Shows raw request/response data for debugging
// - --step or -s: Pauses at each major operation with explanations
//
// Run with: go run ./examples/puzzler/cmd --help
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nxus-SYSTEMS/nxusKit/examples/apps/puzzler"
	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// Global interactive config
var interactiveConfig *interactive.Config

func main() {
	// Parse interactive mode flags first
	interactiveConfig = interactive.FromArgs()

	// Define flags
	gameType := flag.String("game", "sudoku", "Game type: sudoku, set")
	gameShort := flag.String("g", "", "Game type (shorthand)")
	puzzleID := flag.String("puzzle", "easy", "Puzzle ID: easy, medium, hard")
	puzzleShort := flag.String("p", "", "Puzzle ID (shorthand)")
	approach := flag.String("approach", "", "Solving approach: clips, llm, hybrid")
	approachShort := flag.String("a", "", "Solving approach (shorthand)")
	compare := flag.Bool("compare", false, "Compare all three approaches")
	compareShort := flag.Bool("c", false, "Compare all approaches (shorthand)")
	detailedOutput := flag.Bool("detailed", false, "Show detailed puzzle and solution")
	help := flag.Bool("help", false, "Show help message")
	helpShort := flag.Bool("h", false, "Show help (shorthand)")

	flag.Parse()

	// Handle shorthand flags
	if *gameShort != "" {
		*gameType = *gameShort
	}
	if *puzzleShort != "" {
		*puzzleID = *puzzleShort
	}
	if *approachShort != "" {
		*approach = *approachShort
	}
	if *compareShort {
		*compare = true
	}
	if *helpShort {
		*help = true
	}

	// Handle positional argument as game type, then re-parse remaining args
	remaining := flag.Args()
	if len(remaining) > 0 && *gameShort == "" {
		*gameType = remaining[0]
		// Re-parse flags that come after the positional game type
		subFlags := flag.NewFlagSet("sub", flag.ContinueOnError)
		subPuzzle := subFlags.String("p", "", "")
		subApproach := subFlags.String("a", "", "")
		subCompare := subFlags.Bool("c", false, "")
		subFlags.Parse(remaining[1:])
		if *subPuzzle != "" && *puzzleShort == "" {
			*puzzleID = *subPuzzle
		}
		if *subApproach != "" && *approachShort == "" {
			*approach = *subApproach
		}
		if *subCompare {
			*compare = true
		}
	}

	if *help {
		printHelp()
		return
	}

	// Use detailed output or interactive verbose mode
	verbose := *detailedOutput || interactiveConfig.IsVerbose()

	// nxusKit: ClipsProvider enables rule-based puzzle solving and validation
	fmt.Fprintln(os.Stderr, "Puzzler Pattern: CLIPS rules for validation, LLM for solving.")
	fmt.Fprintln(os.Stderr)

	switch *gameType {
	case "sudoku":
		runSudoku(*puzzleID, *approach, *compare, verbose)
	case "set":
		runSetGame(*puzzleID, *approach, *compare, verbose)
	default:
		fmt.Fprintf(os.Stderr, "Unknown game type '%s'. Use 'sudoku' or 'set'.\n", *gameType)
		printHelp()
	}
}

func printHelp() {
	fmt.Println("Puzzler Example: Puzzle Solver Comparison")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("    go run ./examples/puzzler/cmd [OPTIONS] [GAME_TYPE]")
	fmt.Println()
	fmt.Println("GAME TYPES:")
	fmt.Println("    sudoku    Sudoku puzzle solver (default)")
	fmt.Println("    set       Set Game card solver")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("    -g, --game <TYPE>      Game type: sudoku, set")
	fmt.Println("    -p, --puzzle <ID>      Puzzle ID: easy, medium, hard")
	fmt.Println("    -a, --approach <TYPE>  Solving approach: clips, llm, hybrid")
	fmt.Println("    -c, --compare          Compare all three approaches (default)")
	fmt.Println("    --detailed             Show detailed puzzle and solution")
	fmt.Println("    -v, --verbose          Show raw request/response data")
	fmt.Println("    -s, --step             Step through operations with pauses")
	fmt.Println("    -h, --help             Show this help message")
	fmt.Println()
	fmt.Println("INTERACTIVE MODES:")
	fmt.Println("    --verbose shows raw HTTP request/response data for debugging")
	fmt.Println("    --step pauses at each major operation for inspection")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("    go run ./examples/puzzler/cmd sudoku --puzzle easy --compare")
	fmt.Println()
	fmt.Println("    go run ./examples/puzzler/cmd set --compare --detailed")
	fmt.Println()
	fmt.Println("    go run ./examples/puzzler/cmd -g sudoku -p medium -a hybrid --step")
}

// ============================================================================
// Sudoku Game
// ============================================================================

func runSudoku(puzzleID, approach string, compareAll, verbose bool) {
	puzzle := getSudokuPuzzle(puzzleID)

	fmt.Println("Game: Sudoku")
	fmt.Printf("Puzzle: %s\n", puzzle.ID)
	fmt.Printf("Difficulty: %s\n", puzzle.Difficulty)
	fmt.Printf("Filled cells: %d\n", puzzle.FilledCount())
	fmt.Println()

	if verbose {
		printSudokuPuzzle(puzzle)
		fmt.Println()
	}

	if compareAll || approach == "" {
		runSudokuComparison(puzzle, verbose)
	} else {
		runSudokuSingle(puzzle, approach, verbose)
	}
}

func getSudokuPuzzle(id string) *puzzler.SudokuPuzzle {
	switch id {
	case "easy":
		puzzle := puzzler.NewSudokuPuzzle("easy-001", [9][9]int{
			{5, 3, 0, 0, 7, 0, 0, 0, 0},
			{6, 0, 0, 1, 9, 5, 0, 0, 0},
			{0, 9, 8, 0, 0, 0, 0, 6, 0},
			{8, 0, 0, 0, 6, 0, 0, 0, 3},
			{4, 0, 0, 8, 0, 3, 0, 0, 1},
			{7, 0, 0, 0, 2, 0, 0, 0, 6},
			{0, 6, 0, 0, 0, 0, 2, 8, 0},
			{0, 0, 0, 4, 1, 9, 0, 0, 5},
			{0, 0, 0, 0, 8, 0, 0, 7, 9},
		}, puzzler.Easy)
		puzzle.Solution = &[9][9]int{
			{5, 3, 4, 6, 7, 8, 9, 1, 2},
			{6, 7, 2, 1, 9, 5, 3, 4, 8},
			{1, 9, 8, 3, 4, 2, 5, 6, 7},
			{8, 5, 9, 7, 6, 1, 4, 2, 3},
			{4, 2, 6, 8, 5, 3, 7, 9, 1},
			{7, 1, 3, 9, 2, 4, 8, 5, 6},
			{9, 6, 1, 5, 3, 7, 2, 8, 4},
			{2, 8, 7, 4, 1, 9, 6, 3, 5},
			{3, 4, 5, 2, 8, 6, 1, 7, 9},
		}
		return puzzle
	case "medium":
		return puzzler.NewSudokuPuzzle("medium-001", [9][9]int{
			{0, 0, 0, 6, 0, 0, 4, 0, 0},
			{7, 0, 0, 0, 0, 3, 6, 0, 0},
			{0, 0, 0, 0, 9, 1, 0, 8, 0},
			{0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 5, 0, 1, 8, 0, 0, 0, 3},
			{0, 0, 0, 3, 0, 6, 0, 4, 5},
			{0, 4, 0, 2, 0, 0, 0, 6, 0},
			{9, 0, 3, 0, 0, 0, 0, 0, 0},
			{0, 2, 0, 0, 0, 0, 1, 0, 0},
		}, puzzler.Medium)
	case "hard":
		return puzzler.NewSudokuPuzzle("hard-001", [9][9]int{
			{0, 0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 3, 0, 8, 5},
			{0, 0, 1, 0, 2, 0, 0, 0, 0},
			{0, 0, 0, 5, 0, 7, 0, 0, 0},
			{0, 0, 4, 0, 0, 0, 1, 0, 0},
			{0, 9, 0, 0, 0, 0, 0, 0, 0},
			{5, 0, 0, 0, 0, 0, 0, 7, 3},
			{0, 0, 2, 0, 1, 0, 0, 0, 0},
			{0, 0, 0, 0, 4, 0, 0, 0, 9},
		}, puzzler.Hard)
	default:
		fmt.Fprintf(os.Stderr, "Unknown puzzle ID '%s', using easy puzzle\n", id)
		return getSudokuPuzzle("easy")
	}
}

func printSudokuPuzzle(puzzle *puzzler.SudokuPuzzle) {
	fmt.Println("Puzzle:")
	fmt.Println("+-------+-------+-------+")
	for i, row := range puzzle.Puzzle {
		if i > 0 && i%3 == 0 {
			fmt.Println("+-------+-------+-------+")
		}
		fmt.Print("| ")
		for j, val := range row {
			if j > 0 && j%3 == 0 {
				fmt.Print("| ")
			}
			if val == 0 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", val)
			}
		}
		fmt.Println("|")
	}
	fmt.Println("+-------+-------+-------+")
}

func printSudokuSolution(solution [9][9]int) {
	fmt.Println("Solution:")
	fmt.Println("+-------+-------+-------+")
	for i, row := range solution {
		if i > 0 && i%3 == 0 {
			fmt.Println("+-------+-------+-------+")
		}
		fmt.Print("| ")
		for j, val := range row {
			if j > 0 && j%3 == 0 {
				fmt.Print("| ")
			}
			fmt.Printf("%d ", val)
		}
		fmt.Println("|")
	}
	fmt.Println("+-------+-------+-------+")
}

func runSudokuComparison(puzzle *puzzler.SudokuPuzzle, verbose bool) {
	// Step: Comparing solving approaches
	if action := interactiveConfig.StepPause("Comparing all solving approaches...", []string{
		"Will run CLIPS-only approach (rule-based constraint propagation)",
		"Will run LLM-only approach (pure language model reasoning)",
		"Will run Hybrid approach (CLIPS primary, LLM fallback)",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	fmt.Println("Comparing all approaches...")
	fmt.Println()

	// Run CLIPS solver (real ClipsSession)
	fmt.Println("--- CLIPS-only ---")
	clipsResult := solveSudokuWithClips(puzzle)
	solvedStr := "No"
	if clipsResult.Solved {
		solvedStr = "Yes"
	}
	fmt.Printf("Solved: %s | Rules fired: %d\n", solvedStr, clipsResult.Iterations)

	// Note: LLM Sudoku solving is unreliable — LLMs hallucinate digits
	// and violate row/column/box constraints. See internal/roadmap/ for
	// planned CLIPS vs LLM side-by-side comparison.
	fmt.Println("\n--- LLM-only ---")
	fmt.Println("(Skipped — LLM Sudoku solving is unreliable; see roadmap)")
	llmResult := &puzzler.SudokuSolveResult{Solution: puzzle.Puzzle}

	// Determine winner
	winner := "Neither"
	if clipsResult.Solved && llmResult.Solved {
		winner = "CLIPS (faster, deterministic)"
	} else if clipsResult.Solved {
		winner = "CLIPS"
	} else if llmResult.Solved {
		winner = "LLM"
	}
	fmt.Printf("\nWinner: %s\n", winner)

	if verbose && clipsResult.Solved {
		fmt.Println()
		printSudokuSolution(clipsResult.Solution)
	}
}

func runSudokuSingle(puzzle *puzzler.SudokuPuzzle, approach string, verbose bool) {
	var result *puzzler.SudokuSolveResult
	var approachName string

	// Step: Running single approach
	stepExplanation := []string{}
	switch approach {
	case "clips":
		stepExplanation = []string{
			"Using CLIPS-only approach",
			"Rule-based constraint propagation",
			"No LLM calls will be made",
		}
	case "llm":
		stepExplanation = []string{
			"Using LLM-only approach",
			"Pure language model reasoning",
			"Will make API calls to LLM provider",
		}
	case "hybrid":
		stepExplanation = []string{
			"Using Hybrid approach",
			"CLIPS runs first for constraint propagation",
			"LLM is called only when CLIPS gets stuck",
		}
	default:
		stepExplanation = []string{
			"Using CLIPS-only approach (default)",
			"Rule-based constraint propagation",
		}
	}

	if action := interactiveConfig.StepPause("Running puzzle solver...", stepExplanation); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	switch approach {
	case "clips":
		result = solveSudokuWithClips(puzzle)
		approachName = "CLIPS-only"
	case "llm":
		fmt.Println("Note: LLM Sudoku solving is unreliable. Use CLIPS for correct results.")
		result = &puzzler.SudokuSolveResult{Solution: puzzle.Puzzle}
		approachName = "LLM-only"
	case "hybrid":
		result = solveSudokuWithClips(puzzle)
		approachName = "Hybrid (CLIPS)"
	default:
		fmt.Fprintf(os.Stderr, "Unknown approach '%s', using CLIPS\n", approach)
		result = solveSudokuWithClips(puzzle)
		approachName = "CLIPS-only"
	}

	solvedStr := "No"
	if result.Solved {
		solvedStr = "Yes"
	}

	fmt.Printf("Approach: %s\n", approachName)
	fmt.Printf("Solved: %s\n", solvedStr)
	fmt.Printf("Iterations: %d\n", result.Iterations)
	fmt.Printf("LLM calls: %d\n", result.LLMCalls)
	fmt.Printf("Tokens used: %d\n", result.TokensUsed)

	if verbose && result.Solved {
		fmt.Println()
		printSudokuSolution(result.Solution)
	}

	if result.Solved && puzzle.Solution != nil {
		if result.Solution == *puzzle.Solution {
			fmt.Println("\nSolution verified correct!")
		} else {
			fmt.Println("\nWARNING: Solution differs from known solution!")
		}
	}
}

// ============================================================================
// Set Game
// ============================================================================

func runSetGame(puzzleID, approach string, compareAll, verbose bool) {
	hand := getSetHand(puzzleID)

	fmt.Println("Game: Set")
	fmt.Printf("Hand: %s\n", hand.ID)
	fmt.Printf("Cards: %d\n", len(hand.Cards))
	fmt.Println()

	if verbose {
		printSetHand(hand)
		fmt.Println()
	}

	if compareAll || approach == "" {
		runSetComparison(hand, verbose)
	} else {
		runSetSingle(hand, approach, verbose)
	}
}

func getSetHand(id string) *puzzler.SetHand {
	switch id {
	case "easy":
		return puzzler.NewSetHand("easy-001", []puzzler.SetCard{
			// Valid set: all different in all attributes
			puzzler.NewSetCard(0, puzzler.Diamond, puzzler.Red, 1, puzzler.Solid),
			puzzler.NewSetCard(1, puzzler.Oval, puzzler.Green, 2, puzzler.Striped),
			puzzler.NewSetCard(2, puzzler.Squiggle, puzzler.Purple, 3, puzzler.Empty),
			// Extra cards
			puzzler.NewSetCard(3, puzzler.Diamond, puzzler.Red, 2, puzzler.Solid),
			puzzler.NewSetCard(4, puzzler.Oval, puzzler.Red, 1, puzzler.Empty),
			puzzler.NewSetCard(5, puzzler.Squiggle, puzzler.Green, 2, puzzler.Solid),
		})
	case "medium":
		return puzzler.NewSetHand("medium-001", []puzzler.SetCard{
			puzzler.NewSetCard(0, puzzler.Diamond, puzzler.Red, 1, puzzler.Solid),
			puzzler.NewSetCard(1, puzzler.Diamond, puzzler.Green, 1, puzzler.Solid),
			puzzler.NewSetCard(2, puzzler.Diamond, puzzler.Purple, 1, puzzler.Solid),
			puzzler.NewSetCard(3, puzzler.Oval, puzzler.Red, 2, puzzler.Striped),
			puzzler.NewSetCard(4, puzzler.Oval, puzzler.Green, 2, puzzler.Striped),
			puzzler.NewSetCard(5, puzzler.Oval, puzzler.Purple, 2, puzzler.Striped),
			puzzler.NewSetCard(6, puzzler.Squiggle, puzzler.Red, 3, puzzler.Empty),
			puzzler.NewSetCard(7, puzzler.Squiggle, puzzler.Green, 3, puzzler.Empty),
			puzzler.NewSetCard(8, puzzler.Squiggle, puzzler.Purple, 3, puzzler.Empty),
		})
	case "hard":
		return puzzler.NewSetHand("hard-001", []puzzler.SetCard{
			puzzler.NewSetCard(0, puzzler.Diamond, puzzler.Red, 1, puzzler.Solid),
			puzzler.NewSetCard(1, puzzler.Diamond, puzzler.Red, 2, puzzler.Striped),
			puzzler.NewSetCard(2, puzzler.Oval, puzzler.Green, 1, puzzler.Solid),
			puzzler.NewSetCard(3, puzzler.Oval, puzzler.Green, 2, puzzler.Empty),
			puzzler.NewSetCard(4, puzzler.Squiggle, puzzler.Purple, 3, puzzler.Striped),
			puzzler.NewSetCard(5, puzzler.Diamond, puzzler.Green, 3, puzzler.Empty),
			puzzler.NewSetCard(6, puzzler.Oval, puzzler.Purple, 1, puzzler.Striped),
			puzzler.NewSetCard(7, puzzler.Squiggle, puzzler.Red, 2, puzzler.Solid),
			puzzler.NewSetCard(8, puzzler.Squiggle, puzzler.Green, 1, puzzler.Empty),
			puzzler.NewSetCard(9, puzzler.Diamond, puzzler.Purple, 2, puzzler.Solid),
			puzzler.NewSetCard(10, puzzler.Oval, puzzler.Red, 3, puzzler.Empty),
			puzzler.NewSetCard(11, puzzler.Squiggle, puzzler.Purple, 2, puzzler.Striped),
		})
	default:
		fmt.Fprintf(os.Stderr, "Unknown hand ID '%s', using easy hand\n", id)
		return getSetHand("easy")
	}
}

func printSetHand(hand *puzzler.SetHand) {
	fmt.Println("Cards:")
	for _, card := range hand.Cards {
		fmt.Printf("  [%2d] %-9s %-6s %d %-7s\n",
			card.ID, card.Shape, card.Color, card.Count, card.Shading)
	}
}

func printSetResults(result *puzzler.SetGameResult) {
	if len(result.Sets) == 0 {
		fmt.Println("No valid sets found.")
	} else {
		fmt.Printf("Valid sets found: %d\n", len(result.Sets))
		for i, set := range result.Sets {
			fmt.Printf("  Set %d: cards [%d, %d, %d]\n",
				i+1, set.CardIDs[0], set.CardIDs[1], set.CardIDs[2])
			fmt.Printf("         shape: %s, color: %s, count: %s, shading: %s\n",
				set.Validation.Shape, set.Validation.Color,
				set.Validation.Count, set.Validation.Shading)
		}
	}
}

func runSetComparison(hand *puzzler.SetHand, verbose bool) {
	// Step: Comparing Set Game solving approaches
	if action := interactiveConfig.StepPause("Comparing all solving approaches...", []string{
		"Will run CLIPS-only approach (rule-based pattern matching)",
		"Will run LLM-only approach (pure language model reasoning)",
		"Will run Hybrid approach (CLIPS primary, LLM fallback)",
	}); action == interactive.ActionQuit {
		fmt.Println("Exiting...")
		return
	}

	fmt.Println("Comparing all approaches...")
	fmt.Println()

	result := findSetsWithClips(hand)
	fmt.Printf("Sets found: %d\n", len(result.Sets))
	fmt.Printf("Rules fired: %d\n", result.Iterations)

	if verbose {
		printSetResults(result)
	}
}

func runSetSingle(hand *puzzler.SetHand, approach string, verbose bool) {
	var result *puzzler.SetGameResult
	var approachName string

	result = findSetsWithClips(hand)
	approachName = "CLIPS"

	fmt.Printf("Approach: %s\n", approachName)
	fmt.Printf("Sets found: %d\n", len(result.Sets))
	fmt.Printf("Iterations: %d\n", result.Iterations)
	fmt.Printf("LLM calls: %d\n", result.LLMCalls)
	fmt.Printf("Tokens used: %d\n", result.TokensUsed)

	if verbose {
		fmt.Println()
		printSetResults(result)
	}
}

// solveSudokuWithClips solves a Sudoku puzzle using real nxusKit ClipsSession.
//
// nxusKit: Creates a ClipsSession, loads sudoku-propagation.clp and
// sudoku-strategies.clp, asserts the puzzle grid as cell facts,
// runs inference, and reads the solved grid from CLIPS facts.
// Matches the Rust variant's ClipsSession-based solver.
func solveSudokuWithClips(puzzle *puzzler.SudokuPuzzle) *puzzler.SudokuSolveResult {
	clips, err := nxuskit.NewClipsSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS init error: %v\n", err)
		return &puzzler.SudokuSolveResult{Solution: puzzle.Puzzle}
	}
	defer clips.Close()

	// Load rules from shared directory
	rulesFiles := []string{"sudoku-propagation.clp", "sudoku-strategies.clp"}
	rulesDirs := []string{
		"examples/apps/puzzler/shared/rules",
		"../shared/rules",
		"shared/rules",
	}

	for _, rulesFile := range rulesFiles {
		loaded := false
		for _, dir := range rulesDirs {
			path := filepath.Join(dir, rulesFile)
			if _, statErr := os.Stat(path); statErr == nil {
				if loadErr := clips.LoadFile(path); loadErr != nil {
					fmt.Fprintf(os.Stderr, "CLIPS load error (%s): %v\n", path, loadErr)
				} else {
					loaded = true
					break
				}
			}
		}
		if !loaded {
			fmt.Fprintf(os.Stderr, "Could not find %s\n", rulesFile)
			return &puzzler.SudokuSolveResult{Solution: puzzle.Puzzle}
		}
	}

	if err := clips.Reset(); err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS reset error: %v\n", err)
		return &puzzler.SudokuSolveResult{Solution: puzzle.Puzzle}
	}

	// Assert cell facts for the puzzle grid
	for row := 0; row < 9; row++ {
		for col := 0; col < 9; col++ {
			val := puzzle.Puzzle[row][col]
			fact := fmt.Sprintf("(cell (row %d) (col %d) (value %d) (candidates))",
				row+1, col+1, val)
			if _, err := clips.FactAssertString(fact); err != nil {
				fmt.Fprintf(os.Stderr, "CLIPS assert error at (%d,%d): %v\n", row+1, col+1, err)
			}
		}
	}

	// Assert puzzle state
	filled := puzzle.FilledCount()
	clips.FactAssertString(fmt.Sprintf(
		"(puzzle-state (total-cells 81) (solved-cells %d) (iterations 0) (stuck 0))", filled))

	// Run inference
	rulesFired, err := clips.Run(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS run error: %v\n", err)
	}

	// Read back the solved grid using FactsByTemplate + FactSlotValues
	var solution [9][9]int
	factIndices, err := clips.FactsByTemplate("cell")
	if err == nil {
		for _, idx := range factIndices {
			slotsJSON, slotErr := clips.FactSlotValues(idx)
			if slotErr != nil {
				continue
			}
			var slots map[string]json.RawMessage
			if json.Unmarshal([]byte(slotsJSON), &slots) != nil {
				continue
			}
			row := unwrapClipsInt(slots["row"])
			col := unwrapClipsInt(slots["col"])
			val := unwrapClipsInt(slots["value"])
			if row >= 1 && row <= 9 && col >= 1 && col <= 9 {
				solution[row-1][col-1] = val
			}
		}
	}

	solved := true
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			if solution[r][c] < 1 || solution[r][c] > 9 {
				solved = false
			}
		}
	}

	return &puzzler.SudokuSolveResult{
		Solution:   solution,
		Solved:     solved,
		Iterations: rulesFired,
		TokensUsed: 0,
	}
}

// unwrapClipsInt extracts an integer from a ClipsValue JSON ({"type":"integer","value":5}).
func unwrapClipsInt(raw json.RawMessage) int {
	var typed struct {
		Type  string `json:"type"`
		Value int    `json:"value"`
	}
	if json.Unmarshal(raw, &typed) == nil && typed.Type != "" {
		return typed.Value
	}
	var plain int
	json.Unmarshal(raw, &plain)
	return plain
}

func unwrapClipsSymbol(raw json.RawMessage) string {
	var typed struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &typed) == nil && typed.Type != "" {
		return typed.Value
	}
	var plain string
	json.Unmarshal(raw, &plain)
	return plain
}

// findSetsWithClips finds valid sets using real nxusKit ClipsSession.
//
// nxusKit: Loads set-game.clp rules, asserts card facts and game state,
// runs inference, and reads back valid-set facts.
func findSetsWithClips(hand *puzzler.SetHand) *puzzler.SetGameResult {
	clips, err := nxuskit.NewClipsSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS init error: %v\n", err)
		return &puzzler.SetGameResult{}
	}
	defer clips.Close()

	// Load rules
	rulesDirs := []string{
		"examples/apps/puzzler/shared/rules",
		"../shared/rules",
		"shared/rules",
	}
	loaded := false
	for _, dir := range rulesDirs {
		path := filepath.Join(dir, "set-game.clp")
		if _, statErr := os.Stat(path); statErr == nil {
			if loadErr := clips.LoadFile(path); loadErr != nil {
				fmt.Fprintf(os.Stderr, "CLIPS load error (%s): %v\n", path, loadErr)
			} else {
				loaded = true
				break
			}
		}
	}
	if !loaded {
		fmt.Fprintln(os.Stderr, "Could not find set-game.clp")
		return &puzzler.SetGameResult{}
	}

	if err := clips.Reset(); err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS reset error: %v\n", err)
		return &puzzler.SetGameResult{}
	}

	// Assert card facts
	for _, card := range hand.Cards {
		fact := fmt.Sprintf("(set-card (id %d) (shape %s) (color %s) (count %d) (shading %s))",
			card.ID, card.Shape, card.Color, card.Count, card.Shading)
		if _, err := clips.FactAssertString(fact); err != nil {
			fmt.Fprintf(os.Stderr, "CLIPS assert error for card %d: %v\n", card.ID, err)
		}
	}

	// Assert game state
	clips.FactAssertString(fmt.Sprintf("(game-state (total-cards %d) (sets-found 0) (iterations 0))",
		len(hand.Cards)))

	// Run inference
	rulesFired, err := clips.Run(-1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "CLIPS run error: %v\n", err)
	}

	// Read back valid-set facts
	var sets []puzzler.ValidSet
	factIndices, err := clips.FactsByTemplate("valid-set")
	if err == nil {
		for _, idx := range factIndices {
			slotsJSON, slotErr := clips.FactSlotValues(idx)
			if slotErr != nil {
				continue
			}
			var slots map[string]json.RawMessage
			if json.Unmarshal([]byte(slotsJSON), &slots) != nil {
				continue
			}

			sets = append(sets, puzzler.ValidSet{
				CardIDs: [3]int{
					unwrapClipsInt(slots["card1-id"]),
					unwrapClipsInt(slots["card2-id"]),
					unwrapClipsInt(slots["card3-id"]),
				},
				Validation: puzzler.SetValidation{
					Shape:   puzzler.AttributeMatch(unwrapClipsSymbol(slots["shape-match"])),
					Color:   puzzler.AttributeMatch(unwrapClipsSymbol(slots["color-match"])),
					Count:   puzzler.AttributeMatch(unwrapClipsSymbol(slots["count-match"])),
					Shading: puzzler.AttributeMatch(unwrapClipsSymbol(slots["shading-match"])),
				},
			})
		}
	}

	return &puzzler.SetGameResult{
		Sets:       sets,
		FoundAny:   len(sets) > 0,
		Iterations: rulesFired,
	}
}
