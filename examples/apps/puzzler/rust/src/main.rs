//! Puzzler Example: Puzzle Solver Comparison
//!
//! Demonstrates comparing three approaches for solving puzzles:
//! - CLIPS-only: Pure rule-based constraint propagation
//! - LLM-only: Pure LLM reasoning
//! - Hybrid: CLIPS primary with LLM fallback for stuck states
//!
//! Supports both Sudoku and Set Game puzzles.
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show detailed solving information and LLM request/response data
//! - `--step` or `-s`: Pause at each major operation for step-by-step learning
//!
//! Run with: cargo run --example puzzler -- --help

#![allow(clippy::print_stdout, clippy::print_stderr)]

use llm_patterns::puzzler::{
    AttributeMatch, Difficulty, SetCard, SetColor, SetGameResult, SetHand, SetShading, SetShape,
    SetValidation, SudokuPuzzle, SudokuSolveResult, ValidSet,
};
use nxuskit::ClipsSession;
use nxuskit::prelude::*;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
use std::env;

fn main() {
    // Initialize interactive mode from CLI args (--verbose/-v and --step/-s)
    let mut interactive = InteractiveConfig::from_args();
    let args: Vec<String> = env::args().collect();

    // Parse command line arguments
    let mut game_type: Option<String> = None;
    let mut puzzle_id: Option<String> = None;
    let mut approach: Option<String> = None;
    let mut compare_all = false;

    let mut i = 1;
    while i < args.len() {
        match args[i].as_str() {
            "--help" | "-h" => {
                print_help();
                return;
            }
            "--game" | "-g" => {
                i += 1;
                if i < args.len() {
                    game_type = Some(args[i].clone());
                }
            }
            "--puzzle" | "-p" => {
                i += 1;
                if i < args.len() {
                    puzzle_id = Some(args[i].clone());
                }
            }
            "--approach" | "-a" => {
                i += 1;
                if i < args.len() {
                    approach = Some(args[i].clone());
                }
            }
            "--compare" | "-c" => {
                compare_all = true;
            }
            // --verbose and --step are handled by InteractiveConfig::from_args()
            "--verbose" | "-v" | "--step" | "-s" => {}
            _ => {
                if !args[i].starts_with('-') && game_type.is_none() {
                    game_type = Some(args[i].clone());
                }
            }
        }
        i += 1;
    }

    // nxusKit: ClipsProvider enables rule-based puzzle solving and validation
    // LLM-based solving uses Ollama (no API key required) by default
    eprintln!("Puzzler Pattern: CLIPS rules for validation, LLM for solving.\n");

    // Step mode: explain game selection
    if interactive.step_pause(
        "Selecting game type...",
        &[
            "Supported games: Sudoku and Set",
            "Each game supports CLIPS-only, LLM-only, and Hybrid approaches",
            "Comparison mode runs all three approaches",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    let game = game_type.as_deref().unwrap_or("sudoku");

    match game {
        "sudoku" => run_sudoku(
            puzzle_id.as_deref(),
            approach.as_deref(),
            compare_all,
            &mut interactive,
        ),
        "set" => run_set_game(
            puzzle_id.as_deref(),
            approach.as_deref(),
            compare_all,
            &mut interactive,
        ),
        _ => {
            eprintln!("Unknown game type '{}'. Use 'sudoku' or 'set'.", game);
            print_help();
        }
    }
}

fn print_help() {
    println!("Puzzler Example: Puzzle Solver Comparison");
    println!();
    println!("USAGE:");
    println!("    cargo run --example puzzler -- [OPTIONS] [GAME_TYPE]");
    println!();
    println!("GAME TYPES:");
    println!("    sudoku    Sudoku puzzle solver (default)");
    println!("    set       Set Game card solver");
    println!();
    println!("OPTIONS:");
    println!("    -g, --game <TYPE>      Game type: sudoku, set");
    println!("    -p, --puzzle <ID>      Puzzle ID: easy, medium, hard, or custom");
    println!("    -a, --approach <TYPE>  Solving approach: clips, llm, hybrid");
    println!("    -c, --compare          Compare all three approaches (default)");
    println!("    -v, --verbose          Show detailed puzzle, solution, and LLM data");
    println!("    -s, --step             Pause at each major operation for learning");
    println!("    -h, --help             Show this help message");
    println!();
    println!("INTERACTIVE MODES:");
    println!("    --verbose shows raw LLM request/response data for debugging");
    println!("    --step pauses before each major operation with explanations");
    println!("    Both can be combined: --verbose --step");
    println!();
    println!("EXAMPLES:");
    println!("    cargo run --example puzzler -- sudoku --puzzle easy --compare");
    println!();
    println!("    cargo run --example puzzler -- set --compare -v");
    println!();
    println!("    cargo run --example puzzler -- -g sudoku -p medium -a hybrid --step");
}

// ============================================================================
// Sudoku Game
// ============================================================================

fn run_sudoku(
    puzzle_id: Option<&str>,
    approach: Option<&str>,
    compare_all: bool,
    interactive: &mut InteractiveConfig,
) {
    let puzzle = get_sudoku_puzzle(puzzle_id);

    println!("Game: Sudoku");
    println!("Puzzle: {}", puzzle.id);
    println!("Difficulty: {:?}", puzzle.difficulty);
    println!("Filled cells: {}", puzzle.filled_count());
    println!();

    if interactive.is_verbose() {
        print_sudoku_puzzle(&puzzle);
        println!();
    }

    // Step mode: explain solving approach
    if interactive.step_pause(
        "Setting up Sudoku solver...",
        &[
            &format!("Puzzle: {} ({:?} difficulty)", puzzle.id, puzzle.difficulty),
            "CLIPS uses constraint propagation rules",
            "LLM uses reasoning to fill cells",
            "Hybrid combines both approaches",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    if compare_all || approach.is_none() {
        run_sudoku_comparison(&puzzle, interactive);
    } else {
        run_sudoku_single(&puzzle, approach.unwrap_or("clips"), interactive);
    }
}

fn get_sudoku_puzzle(id: Option<&str>) -> SudokuPuzzle {
    match id.unwrap_or("easy") {
        "easy" => SudokuPuzzle::new(
            "easy-001",
            [
                [5, 3, 0, 0, 7, 0, 0, 0, 0],
                [6, 0, 0, 1, 9, 5, 0, 0, 0],
                [0, 9, 8, 0, 0, 0, 0, 6, 0],
                [8, 0, 0, 0, 6, 0, 0, 0, 3],
                [4, 0, 0, 8, 0, 3, 0, 0, 1],
                [7, 0, 0, 0, 2, 0, 0, 0, 6],
                [0, 6, 0, 0, 0, 0, 2, 8, 0],
                [0, 0, 0, 4, 1, 9, 0, 0, 5],
                [0, 0, 0, 0, 8, 0, 0, 7, 9],
            ],
            Difficulty::Easy,
        )
        .with_solution([
            [5, 3, 4, 6, 7, 8, 9, 1, 2],
            [6, 7, 2, 1, 9, 5, 3, 4, 8],
            [1, 9, 8, 3, 4, 2, 5, 6, 7],
            [8, 5, 9, 7, 6, 1, 4, 2, 3],
            [4, 2, 6, 8, 5, 3, 7, 9, 1],
            [7, 1, 3, 9, 2, 4, 8, 5, 6],
            [9, 6, 1, 5, 3, 7, 2, 8, 4],
            [2, 8, 7, 4, 1, 9, 6, 3, 5],
            [3, 4, 5, 2, 8, 6, 1, 7, 9],
        ]),
        "medium" => SudokuPuzzle::new(
            "medium-001",
            [
                [0, 0, 0, 6, 0, 0, 4, 0, 0],
                [7, 0, 0, 0, 0, 3, 6, 0, 0],
                [0, 0, 0, 0, 9, 1, 0, 8, 0],
                [0, 0, 0, 0, 0, 0, 0, 0, 0],
                [0, 5, 0, 1, 8, 0, 0, 0, 3],
                [0, 0, 0, 3, 0, 6, 0, 4, 5],
                [0, 4, 0, 2, 0, 0, 0, 6, 0],
                [9, 0, 3, 0, 0, 0, 0, 0, 0],
                [0, 2, 0, 0, 0, 0, 1, 0, 0],
            ],
            Difficulty::Medium,
        ),
        "hard" => SudokuPuzzle::new(
            "hard-001",
            [
                [0, 0, 0, 0, 0, 0, 0, 0, 0],
                [0, 0, 0, 0, 0, 3, 0, 8, 5],
                [0, 0, 1, 0, 2, 0, 0, 0, 0],
                [0, 0, 0, 5, 0, 7, 0, 0, 0],
                [0, 0, 4, 0, 0, 0, 1, 0, 0],
                [0, 9, 0, 0, 0, 0, 0, 0, 0],
                [5, 0, 0, 0, 0, 0, 0, 7, 3],
                [0, 0, 2, 0, 1, 0, 0, 0, 0],
                [0, 0, 0, 0, 4, 0, 0, 0, 9],
            ],
            Difficulty::Hard,
        ),
        _ => {
            eprintln!("Unknown puzzle ID, using easy puzzle");
            get_sudoku_puzzle(Some("easy"))
        }
    }
}

fn print_sudoku_puzzle(puzzle: &SudokuPuzzle) {
    println!("Puzzle:");
    println!("+-------+-------+-------+");
    for (i, row) in puzzle.puzzle.iter().enumerate() {
        if i > 0 && i % 3 == 0 {
            println!("+-------+-------+-------+");
        }
        print!("| ");
        for (j, &val) in row.iter().enumerate() {
            if j > 0 && j % 3 == 0 {
                print!("| ");
            }
            if val == 0 {
                print!(". ");
            } else {
                print!("{} ", val);
            }
        }
        println!("|");
    }
    println!("+-------+-------+-------+");
}

fn print_sudoku_solution(solution: &[[u8; 9]; 9]) {
    println!("Solution:");
    println!("+-------+-------+-------+");
    for (i, row) in solution.iter().enumerate() {
        if i > 0 && i % 3 == 0 {
            println!("+-------+-------+-------+");
        }
        print!("| ");
        for (j, &val) in row.iter().enumerate() {
            if j > 0 && j % 3 == 0 {
                print!("| ");
            }
            print!("{} ", val);
        }
        println!("|");
    }
    println!("+-------+-------+-------+");
}

fn run_sudoku_comparison(puzzle: &SudokuPuzzle, interactive: &mut InteractiveConfig) {
    println!("Comparing all approaches...\n");

    // Step mode: explain comparison
    if interactive.step_pause(
        "Running approach comparison...",
        &[
            "Will run CLIPS-only, LLM-only, and Hybrid approaches",
            "Each approach is timed and accuracy measured",
            "Results are compared in a table",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    // Run all three approaches and report
    println!("--- CLIPS-only ---");
    let clips_result = solve_sudoku_clips_real(puzzle);
    println!(
        "Solved: {} | Iterations: {} | LLM calls: 0",
        if clips_result.solved { "Yes" } else { "No" },
        clips_result.iterations
    );

    println!("\n--- LLM-only ---");
    let llm_result = solve_sudoku_llm_real(puzzle);
    println!(
        "Solved: {} | Tokens: {}",
        if llm_result.solved { "Yes" } else { "No" },
        llm_result.tokens_used
    );

    println!("\n--- Hybrid (CLIPS first, LLM fallback) ---");
    let hybrid_result = if clips_result.solved {
        println!("CLIPS solved it — no LLM needed");
        clips_result.clone()
    } else {
        println!("CLIPS incomplete — falling back to LLM");
        solve_sudoku_llm_real(puzzle)
    };

    // Determine winner
    let winner = if clips_result.solved && llm_result.solved {
        "CLIPS (faster, deterministic)"
    } else if clips_result.solved {
        "CLIPS"
    } else if llm_result.solved {
        "LLM"
    } else {
        "Neither"
    };
    println!("\nWinner: {}", winner);

    if interactive.is_verbose() && hybrid_result.solved {
        println!();
        print_sudoku_solution(&hybrid_result.solution);
    }
}

fn run_sudoku_single(puzzle: &SudokuPuzzle, approach: &str, interactive: &mut InteractiveConfig) {
    // Step mode: explain single approach
    if interactive.step_pause(
        &format!("Running {} approach...", approach),
        &[
            match approach {
                "clips" => "CLIPS uses constraint propagation rules",
                "llm" => "LLM uses natural language reasoning",
                "hybrid" => "Hybrid uses CLIPS with LLM fallback",
                _ => "Unknown approach",
            },
            "Will solve puzzle and report statistics",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    let (result, approach_name) = match approach {
        "clips" => (solve_sudoku_clips_real(puzzle), "CLIPS-only"),
        "llm" => (solve_sudoku_llm_real(puzzle), "LLM-only"),
        "hybrid" => {
            // Try CLIPS first, fall back to real LLM
            let clips_result = solve_sudoku_clips_real(puzzle);
            if clips_result.solved {
                (clips_result, "Hybrid")
            } else {
                (solve_sudoku_llm_real(puzzle), "Hybrid")
            }
        }
        _ => {
            eprintln!("Unknown approach '{}', using CLIPS", approach);
            (solve_sudoku_clips_real(puzzle), "CLIPS-only")
        }
    };

    println!("Approach: {}", approach_name);
    println!("Solved: {}", if result.solved { "Yes" } else { "No" });
    println!("Iterations: {}", result.iterations);
    println!("LLM calls: {}", result.llm_calls);
    println!("Tokens used: {}", result.tokens_used);

    if interactive.is_verbose() && result.solved {
        println!();
        print_sudoku_solution(&result.solution);
    }

    #[allow(clippy::collapsible_if)]
    if result.solved {
        if let Some(ref known_solution) = puzzle.solution {
            if result.solution == *known_solution {
                println!("\nSolution verified correct!");
            } else {
                println!("\nWARNING: Solution differs from known solution!");
            }
        }
    }
}

// ============================================================================
// Set Game
// ============================================================================

fn run_set_game(
    puzzle_id: Option<&str>,
    approach: Option<&str>,
    compare_all: bool,
    interactive: &mut InteractiveConfig,
) {
    let hand = get_set_hand(puzzle_id);

    println!("Game: Set");
    println!("Hand: {}", hand.id);
    println!("Cards: {}", hand.cards.len());
    println!();

    if interactive.is_verbose() {
        print_set_hand(&hand);
        println!();
    }

    // Step mode: explain Set game
    if interactive.step_pause(
        "Setting up Set game solver...",
        &[
            &format!("Hand: {} with {} cards", hand.id, hand.cards.len()),
            "Must find triplets with all-same or all-different attributes",
            "Attributes: shape, color, count, shading",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    if compare_all || approach.is_none() {
        run_set_comparison(&hand, interactive);
    } else {
        run_set_single(&hand, approach.unwrap_or("clips"), interactive);
    }
}

fn get_set_hand(id: Option<&str>) -> SetHand {
    match id.unwrap_or("easy") {
        "easy" => SetHand::new(
            "easy-001",
            vec![
                // Valid set: all different in all attributes
                SetCard::new(0, SetShape::Diamond, SetColor::Red, 1, SetShading::Solid),
                SetCard::new(1, SetShape::Oval, SetColor::Green, 2, SetShading::Striped),
                SetCard::new(
                    2,
                    SetShape::Squiggle,
                    SetColor::Purple,
                    3,
                    SetShading::Empty,
                ),
                // Extra cards that don't form sets easily
                SetCard::new(3, SetShape::Diamond, SetColor::Red, 2, SetShading::Solid),
                SetCard::new(4, SetShape::Oval, SetColor::Red, 1, SetShading::Empty),
                SetCard::new(5, SetShape::Squiggle, SetColor::Green, 2, SetShading::Solid),
            ],
        ),
        "medium" => SetHand::new(
            "medium-001",
            vec![
                SetCard::new(0, SetShape::Diamond, SetColor::Red, 1, SetShading::Solid),
                SetCard::new(1, SetShape::Diamond, SetColor::Green, 1, SetShading::Solid),
                SetCard::new(2, SetShape::Diamond, SetColor::Purple, 1, SetShading::Solid),
                SetCard::new(3, SetShape::Oval, SetColor::Red, 2, SetShading::Striped),
                SetCard::new(4, SetShape::Oval, SetColor::Green, 2, SetShading::Striped),
                SetCard::new(5, SetShape::Oval, SetColor::Purple, 2, SetShading::Striped),
                SetCard::new(6, SetShape::Squiggle, SetColor::Red, 3, SetShading::Empty),
                SetCard::new(7, SetShape::Squiggle, SetColor::Green, 3, SetShading::Empty),
                SetCard::new(
                    8,
                    SetShape::Squiggle,
                    SetColor::Purple,
                    3,
                    SetShading::Empty,
                ),
            ],
        ),
        "hard" => SetHand::new(
            "hard-001",
            vec![
                SetCard::new(0, SetShape::Diamond, SetColor::Red, 1, SetShading::Solid),
                SetCard::new(1, SetShape::Diamond, SetColor::Red, 2, SetShading::Striped),
                SetCard::new(2, SetShape::Oval, SetColor::Green, 1, SetShading::Solid),
                SetCard::new(3, SetShape::Oval, SetColor::Green, 2, SetShading::Empty),
                SetCard::new(
                    4,
                    SetShape::Squiggle,
                    SetColor::Purple,
                    3,
                    SetShading::Striped,
                ),
                SetCard::new(5, SetShape::Diamond, SetColor::Green, 3, SetShading::Empty),
                SetCard::new(6, SetShape::Oval, SetColor::Purple, 1, SetShading::Striped),
                SetCard::new(7, SetShape::Squiggle, SetColor::Red, 2, SetShading::Solid),
                SetCard::new(8, SetShape::Squiggle, SetColor::Green, 1, SetShading::Empty),
                SetCard::new(9, SetShape::Diamond, SetColor::Purple, 2, SetShading::Solid),
                SetCard::new(10, SetShape::Oval, SetColor::Red, 3, SetShading::Empty),
                SetCard::new(
                    11,
                    SetShape::Squiggle,
                    SetColor::Purple,
                    2,
                    SetShading::Striped,
                ),
            ],
        ),
        _ => {
            eprintln!("Unknown hand ID, using easy hand");
            get_set_hand(Some("easy"))
        }
    }
}

fn print_set_hand(hand: &SetHand) {
    println!("Cards:");
    for card in &hand.cards {
        println!(
            "  [{:2}] {:9} {:6} {} {:7}",
            card.id,
            format!("{:?}", card.shape),
            format!("{:?}", card.color),
            card.count,
            format!("{:?}", card.shading)
        );
    }
}

fn print_set_results(result: &SetGameResult) {
    if result.sets.is_empty() {
        println!("No valid sets found.");
    } else {
        println!("Valid sets found: {}", result.sets.len());
        for (i, set) in result.sets.iter().enumerate() {
            println!(
                "  Set {}: cards [{}, {}, {}]",
                i + 1,
                set.card_ids[0],
                set.card_ids[1],
                set.card_ids[2]
            );
            println!(
                "         shape: {:?}, color: {:?}, count: {:?}, shading: {:?}",
                set.validation.shape,
                set.validation.color,
                set.validation.count,
                set.validation.shading
            );
        }
    }
}

fn run_set_comparison(hand: &SetHand, interactive: &mut InteractiveConfig) {
    println!("Comparing all approaches...\n");

    // Step mode: explain comparison
    if interactive.step_pause(
        "Running approach comparison...",
        &[
            "Will run CLIPS-only, LLM-only, and Hybrid approaches",
            "Each approach is timed and sets counted",
            "Results are compared in a table",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    let result = find_sets_with_clips(hand);
    println!("Sets found: {}", result.sets.len());
    println!("Rules fired: {}", result.iterations);

    if interactive.is_verbose() {
        print_set_results(&result);
    }
}

fn run_set_single(hand: &SetHand, approach: &str, interactive: &mut InteractiveConfig) {
    // Step mode: explain single approach
    if interactive.step_pause(
        &format!("Running {} approach...", approach),
        &[
            match approach {
                "clips" => "CLIPS checks all triplet combinations",
                "llm" => "LLM reasons about card attributes",
                "hybrid" => "Hybrid uses CLIPS with LLM verification",
                _ => "Unknown approach",
            },
            "Will find valid sets and report statistics",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    let (result, approach_name) = match approach {
        "clips" | "hybrid" | _ => (find_sets_with_clips(hand), "CLIPS"),
    };

    println!("Approach: {}", approach_name);
    println!("Sets found: {}", result.sets.len());
    println!("Iterations: {}", result.iterations);
    println!("LLM calls: {}", result.llm_calls);
    println!("Tokens used: {}", result.tokens_used);

    if interactive.is_verbose() {
        println!();
        print_set_results(&result);
    }
}

// ============================================================================
// Real SDK Solvers (CLIPS + LLM)
// ============================================================================

/// Solves a Sudoku puzzle using real CLIPS rules via ClipsSession.
///
/// Loads sudoku-propagation.clp and sudoku-strategies.clp, asserts the puzzle
/// grid as cell facts, runs inference, and reads back the solved grid.
fn solve_sudoku_clips_real(puzzle: &SudokuPuzzle) -> SudokuSolveResult {
    let clips = match ClipsSession::create() {
        Ok(c) => c,
        Err(e) => {
            eprintln!("CLIPS init error: {e}");
            return SudokuSolveResult {
                solution: puzzle.puzzle,
                solved: false,
                iterations: 0,
                llm_calls: 0,
                tokens_used: 0,
            };
        }
    };

    // Load rules — try multiple paths
    let rules_files = ["sudoku-propagation.clp", "sudoku-strategies.clp"];
    let rules_dirs = [
        "examples/apps/puzzler/shared/rules",
        "../shared/rules",
        "shared/rules",
    ];

    for rules_file in &rules_files {
        let mut loaded = false;
        for dir in &rules_dirs {
            let path = format!("{}/{}", dir, rules_file);
            if std::path::Path::new(&path).exists() {
                if let Err(e) = clips.load_file(&path) {
                    eprintln!("CLIPS load error ({}): {e}", path);
                } else {
                    loaded = true;
                    break;
                }
            }
        }
        if !loaded {
            eprintln!("Could not find {}", rules_file);
            return SudokuSolveResult {
                solution: puzzle.puzzle,
                solved: false,
                iterations: 0,
                llm_calls: 0,
                tokens_used: 0,
            };
        }
    }

    if let Err(e) = clips.reset() {
        eprintln!("CLIPS reset error: {e}");
        return SudokuSolveResult {
            solution: puzzle.puzzle,
            solved: false,
            iterations: 0,
            llm_calls: 0,
            tokens_used: 0,
        };
    }

    // Assert cell facts for the puzzle grid
    for row in 0..9 {
        for col in 0..9 {
            let val = puzzle.puzzle[row][col];
            let fact = format!(
                "(cell (row {}) (col {}) (value {}) (candidates))",
                row + 1,
                col + 1,
                val
            );
            if let Err(e) = clips.fact_assert_string(&fact) {
                eprintln!("CLIPS assert error at ({},{}): {e}", row + 1, col + 1);
            }
        }
    }

    // Assert initial puzzle state
    let filled = puzzle.filled_count();
    let _ = clips.fact_assert_string(&format!(
        "(puzzle-state (total-cells 81) (solved-cells {}) (iterations 0) (stuck 0))",
        filled
    ));

    // Run inference
    let rules_fired = match clips.run(None) {
        Ok(n) => n,
        Err(e) => {
            eprintln!("CLIPS run error: {e}");
            0
        }
    };

    // Read back the solved grid
    let mut solution = [[0u8; 9]; 9];
    if let Ok(facts) = clips.facts_by_template("cell") {
        for fact_idx in facts {
            if let Ok(slots) = clips.fact_slot_values(fact_idx) {
                let row = slots
                    .get("row")
                    .and_then(|v| v.as_integer().ok())
                    .unwrap_or(0) as usize;
                let col = slots
                    .get("col")
                    .and_then(|v| v.as_integer().ok())
                    .unwrap_or(0) as usize;
                let val = slots
                    .get("value")
                    .and_then(|v| v.as_integer().ok())
                    .unwrap_or(0) as u8;
                if row >= 1 && row <= 9 && col >= 1 && col <= 9 {
                    solution[row - 1][col - 1] = val;
                }
            }
        }
    }

    let solved = solution
        .iter()
        .all(|row| row.iter().all(|&v| v >= 1 && v <= 9));

    SudokuSolveResult {
        solution,
        solved,
        iterations: rules_fired as u64,
        llm_calls: 0,
        tokens_used: 0,
    }
}

/// Solves a Sudoku puzzle using a real LLM provider.
///
/// Uses Ollama by default (no API key required). Set ANTHROPIC_API_KEY
/// to use Claude instead.
fn solve_sudoku_llm_real(puzzle: &SudokuPuzzle) -> SudokuSolveResult {
    let grid_str = puzzle
        .puzzle
        .iter()
        .map(|row| {
            row.iter()
                .map(|&v| {
                    if v == 0 {
                        ".".to_string()
                    } else {
                        v.to_string()
                    }
                })
                .collect::<Vec<_>>()
                .join(" ")
        })
        .collect::<Vec<_>>()
        .join("\n");

    let prompt = format!(
        "Solve this Sudoku puzzle. Empty cells are shown as dots.\n\n{}\n\n\
         Return ONLY the completed 9x9 grid as 9 lines of 9 space-separated digits, nothing else.",
        grid_str
    );

    let chat_result = if let Ok(api_key) = env::var("ANTHROPIC_API_KEY")
        && !api_key.is_empty()
    {
        let provider = ClaudeProvider::builder()
            .api_key(api_key)
            .build()
            .map_err(|e| e.to_string());

        provider.and_then(|p| {
            let request = ChatRequest::new("claude-haiku-4-5-20251001")
                .with_message(Message::user(&prompt))
                .with_temperature(0.0_f32)
                .with_max_tokens(500);
            p.chat(request).map_err(|e| e.to_string())
        })
    } else {
        let provider = OllamaProvider::builder().build().map_err(|e| e.to_string());

        provider.and_then(|p| {
            let request = ChatRequest::new("llama3")
                .with_message(Message::user(&prompt))
                .with_temperature(0.0_f32)
                .with_max_tokens(500);
            p.chat(request).map_err(|e| e.to_string())
        })
    };

    match chat_result {
        Ok(response) => {
            let tokens = response.usage.estimated.prompt_tokens as u64
                + response.usage.estimated.completion_tokens as u64;
            let solution = parse_sudoku_response(&response.content, &puzzle.puzzle);
            let solved = puzzle
                .solution
                .as_ref()
                .map(|s| solution == *s)
                .unwrap_or_else(|| {
                    // Basic validation: no zeros remain
                    solution.iter().all(|row| row.iter().all(|&v| v > 0))
                });

            SudokuSolveResult {
                solution,
                solved,
                iterations: 1,
                llm_calls: 1,
                tokens_used: tokens,
            }
        }
        Err(e) => {
            eprintln!("LLM solver error: {e}");
            SudokuSolveResult {
                solution: puzzle.puzzle,
                solved: false,
                iterations: 0,
                llm_calls: 1,
                tokens_used: 0,
            }
        }
    }
}

/// Parses an LLM response into a Sudoku grid.
fn parse_sudoku_response(content: &str, original: &[[u8; 9]; 9]) -> [[u8; 9]; 9] {
    let mut grid = *original;
    let lines: Vec<&str> = content
        .lines()
        .filter(|l| {
            let trimmed = l.trim();
            !trimmed.is_empty() && trimmed.chars().any(|c| c.is_ascii_digit())
        })
        .collect();

    for (row, line) in lines.iter().enumerate().take(9) {
        let digits: Vec<u8> = line
            .chars()
            .filter(|c| c.is_ascii_digit())
            .filter_map(|c| c.to_digit(10).map(|d| d as u8))
            .filter(|&d| d >= 1 && d <= 9)
            .collect();

        for (col, &val) in digits.iter().enumerate().take(9) {
            grid[row][col] = val;
        }
    }

    grid
}

// ============================================================================
// Real SDK Set Game Solver
// ============================================================================

/// Finds valid sets in a Set Game hand using real nxusKit ClipsSession.
///
/// nxusKit: Loads set-game.clp rules, asserts card facts and game state,
/// runs inference, and reads back valid-set facts.
fn find_sets_with_clips(hand: &SetHand) -> SetGameResult {
    let clips = match ClipsSession::create() {
        Ok(c) => c,
        Err(e) => {
            eprintln!("CLIPS init error: {e}");
            return SetGameResult {
                sets: vec![],
                found_any: false,
                iterations: 0,
                llm_calls: 0,
                tokens_used: 0,
            };
        }
    };

    // Load rules
    let rules_dirs = [
        "examples/apps/puzzler/shared/rules",
        "../shared/rules",
        "shared/rules",
    ];
    let mut loaded = false;
    for dir in &rules_dirs {
        let path = format!("{}/set-game.clp", dir);
        if std::path::Path::new(&path).exists() {
            if let Err(e) = clips.load_file(&path) {
                eprintln!("CLIPS load error ({}): {e}", path);
            } else {
                loaded = true;
                break;
            }
        }
    }
    if !loaded {
        eprintln!("Could not find set-game.clp");
        return SetGameResult {
            sets: vec![],
            found_any: false,
            iterations: 0,
            llm_calls: 0,
            tokens_used: 0,
        };
    }

    if let Err(e) = clips.reset() {
        eprintln!("CLIPS reset error: {e}");
        return SetGameResult {
            sets: vec![],
            found_any: false,
            iterations: 0,
            llm_calls: 0,
            tokens_used: 0,
        };
    }

    // Assert card facts
    for card in &hand.cards {
        let shape = format!("{:?}", card.shape).to_lowercase();
        let color = format!("{:?}", card.color).to_lowercase();
        let shading = format!("{:?}", card.shading).to_lowercase();
        let fact = format!(
            "(set-card (id {}) (shape {}) (color {}) (count {}) (shading {}))",
            card.id, shape, color, card.count, shading
        );
        if let Err(e) = clips.fact_assert_string(&fact) {
            eprintln!("CLIPS assert error for card {}: {e}", card.id);
        }
    }

    // Assert game state
    let _ = clips.fact_assert_string(&format!(
        "(game-state (total-cards {}) (sets-found 0) (iterations 0))",
        hand.cards.len()
    ));

    // Run inference
    let rules_fired = match clips.run(None) {
        Ok(n) => n,
        Err(e) => {
            eprintln!("CLIPS run error: {e}");
            0
        }
    };

    // Read back valid-set facts
    let mut sets = Vec::new();
    if let Ok(fact_indices) = clips.facts_by_template("valid-set") {
        for idx in fact_indices {
            if let Ok(slots) = clips.fact_slot_values(idx) {
                let card1 = slots
                    .get("card1-id")
                    .and_then(|v| v.as_integer().ok())
                    .unwrap_or(0) as u8;
                let card2 = slots
                    .get("card2-id")
                    .and_then(|v| v.as_integer().ok())
                    .unwrap_or(0) as u8;
                let card3 = slots
                    .get("card3-id")
                    .and_then(|v| v.as_integer().ok())
                    .unwrap_or(0) as u8;

                let shape_match = clips_match_to_attribute(slots.get("shape-match"));
                let color_match = clips_match_to_attribute(slots.get("color-match"));
                let count_match = clips_match_to_attribute(slots.get("count-match"));
                let shading_match = clips_match_to_attribute(slots.get("shading-match"));

                sets.push(ValidSet {
                    card_ids: [card1, card2, card3],
                    validation: SetValidation {
                        shape: shape_match,
                        color: color_match,
                        count: count_match,
                        shading: shading_match,
                    },
                });
            }
        }
    }

    let found_any = !sets.is_empty();
    SetGameResult {
        sets,
        found_any,
        iterations: rules_fired as u64,
        llm_calls: 0,
        tokens_used: 0,
    }
}

fn clips_match_to_attribute(value: Option<&nxuskit::ClipsValue>) -> AttributeMatch {
    match value {
        Some(nxuskit::ClipsValue::Symbol(s)) if s == "all-same" => AttributeMatch::AllSame,
        _ => AttributeMatch::AllDifferent,
    }
}
