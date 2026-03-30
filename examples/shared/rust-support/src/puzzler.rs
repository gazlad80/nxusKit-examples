//! Puzzler pattern types for puzzle solving comparison.
//!
//! The Puzzler pattern compares three approaches for solving puzzles:
//! CLIPS-only (pure rule-based), LLM-only (pure reasoning), and
//! Hybrid (CLIPS primary with LLM fallback).

#![allow(clippy::needless_range_loop)]

use serde::{Deserialize, Serialize};

/// Solving approach enumeration.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SolverApproach {
    /// Pure CLIPS rule-based solving
    ClipsOnly,
    /// Pure LLM reasoning
    LlmOnly,
    /// CLIPS primary, LLM for stuck states
    Hybrid,
}

impl std::fmt::Display for SolverApproach {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SolverApproach::ClipsOnly => write!(f, "clips_only"),
            SolverApproach::LlmOnly => write!(f, "llm_only"),
            SolverApproach::Hybrid => write!(f, "hybrid"),
        }
    }
}

/// Puzzle difficulty level.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Difficulty {
    /// 30-45 filled cells, basic constraint propagation sufficient
    Easy,
    /// 23-29 filled cells, may require hidden singles
    Medium,
    /// 17-22 filled cells, requires advanced strategies or guessing
    Hard,
}

impl Difficulty {
    /// Returns the typical range of filled cells for this difficulty.
    pub fn filled_cells_range(&self) -> (u8, u8) {
        match self {
            Difficulty::Easy => (30, 45),
            Difficulty::Medium => (23, 29),
            Difficulty::Hard => (17, 22),
        }
    }
}

/// Sudoku puzzle representation.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SudokuPuzzle {
    /// Unique puzzle identifier
    pub id: String,

    /// 9x9 grid, 0 for empty cells
    pub puzzle: [[u8; 9]; 9],

    /// Puzzle difficulty
    pub difficulty: Difficulty,

    /// Known correct solution (optional, for validation)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub solution: Option<[[u8; 9]; 9]>,

    /// Puzzle attribution/source
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source: Option<String>,
}

impl SudokuPuzzle {
    /// Creates a new puzzle with the given grid.
    pub fn new(id: impl Into<String>, puzzle: [[u8; 9]; 9], difficulty: Difficulty) -> Self {
        Self {
            id: id.into(),
            puzzle,
            difficulty,
            solution: None,
            source: None,
        }
    }

    /// Sets the known solution.
    pub fn with_solution(mut self, solution: [[u8; 9]; 9]) -> Self {
        self.solution = Some(solution);
        self
    }

    /// Counts the number of filled (non-zero) cells.
    pub fn filled_count(&self) -> usize {
        self.puzzle
            .iter()
            .flat_map(|row| row.iter())
            .filter(|&&v| v != 0)
            .count()
    }

    /// Validates that the puzzle grid is well-formed.
    pub fn validate(&self) -> Result<(), String> {
        // Check all values are 0-9
        for (row_idx, row) in self.puzzle.iter().enumerate() {
            for (col_idx, &val) in row.iter().enumerate() {
                if val > 9 {
                    return Err(format!(
                        "Invalid value {} at ({}, {}): must be 0-9",
                        val, row_idx, col_idx
                    ));
                }
            }
        }

        // Check minimum filled cells (17 for unique solution)
        let filled = self.filled_count();
        if filled < 17 {
            return Err(format!(
                "Only {} filled cells: minimum 17 required for unique solution",
                filled
            ));
        }

        Ok(())
    }

    /// Validates a solution against the puzzle.
    pub fn validate_solution(&self, solution: &[[u8; 9]; 9]) -> Result<(), String> {
        // Check rows
        for (row_idx, row) in solution.iter().enumerate() {
            let mut seen = [false; 10];
            for &val in row.iter() {
                if !(1..=9).contains(&val) {
                    return Err(format!("Invalid value {} in row {}", val, row_idx));
                }
                if seen[val as usize] {
                    return Err(format!("Duplicate {} in row {}", val, row_idx));
                }
                seen[val as usize] = true;
            }
        }

        // Check columns
        for col_idx in 0..9 {
            let mut seen = [false; 10];
            for row in solution.iter() {
                let val = row[col_idx];
                if seen[val as usize] {
                    return Err(format!("Duplicate {} in column {}", val, col_idx));
                }
                seen[val as usize] = true;
            }
        }

        // Check 3x3 boxes
        for box_row in 0..3 {
            for box_col in 0..3 {
                let mut seen = [false; 10];
                for dr in 0..3 {
                    for dc in 0..3 {
                        let val = solution[box_row * 3 + dr][box_col * 3 + dc];
                        if seen[val as usize] {
                            return Err(format!(
                                "Duplicate {} in box ({}, {})",
                                val, box_row, box_col
                            ));
                        }
                        seen[val as usize] = true;
                    }
                }
            }
        }

        // Verify solution matches puzzle constraints
        for row_idx in 0..9 {
            for col_idx in 0..9 {
                let puzzle_val = self.puzzle[row_idx][col_idx];
                if puzzle_val != 0 && puzzle_val != solution[row_idx][col_idx] {
                    return Err(format!(
                        "Solution conflicts with puzzle at ({}, {}): puzzle has {}, solution has {}",
                        row_idx, col_idx, puzzle_val, solution[row_idx][col_idx]
                    ));
                }
            }
        }

        Ok(())
    }
}

/// Shape attribute for Set Game cards.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SetShape {
    Diamond,
    Oval,
    Squiggle,
}

/// Color attribute for Set Game cards.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SetColor {
    Red,
    Green,
    Purple,
}

/// Shading attribute for Set Game cards.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SetShading {
    Solid,
    Striped,
    Empty,
}

/// Set Game card representation.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub struct SetCard {
    /// Card position in hand (0-11)
    pub id: u8,

    /// Card shape
    pub shape: SetShape,

    /// Card color
    pub color: SetColor,

    /// Number of shapes (1, 2, or 3)
    pub count: u8,

    /// Fill pattern
    pub shading: SetShading,
}

impl SetCard {
    /// Creates a new Set Game card.
    pub fn new(id: u8, shape: SetShape, color: SetColor, count: u8, shading: SetShading) -> Self {
        Self {
            id,
            shape,
            color,
            count,
            shading,
        }
    }
}

/// Collection of cards for Set Game.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetHand {
    /// Hand identifier
    pub id: String,

    /// 12 cards in play
    pub cards: Vec<SetCard>,

    /// Known valid sets for validation
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub known_sets: Vec<[u8; 3]>,
}

impl SetHand {
    /// Creates a new hand with the given cards.
    pub fn new(id: impl Into<String>, cards: Vec<SetCard>) -> Self {
        Self {
            id: id.into(),
            cards,
            known_sets: Vec::new(),
        }
    }

    /// Validates the hand has correct structure.
    pub fn validate(&self) -> Result<(), String> {
        if self.cards.len() != 12 {
            return Err(format!(
                "Hand must have exactly 12 cards, has {}",
                self.cards.len()
            ));
        }

        // Check for duplicate cards
        for i in 0..self.cards.len() {
            for j in (i + 1)..self.cards.len() {
                if self.cards[i] == self.cards[j] {
                    return Err(format!(
                        "Duplicate cards at positions {} and {}",
                        self.cards[i].id, self.cards[j].id
                    ));
                }
            }
        }

        // Validate card IDs
        for card in &self.cards {
            if card.id > 11 {
                return Err(format!("Card ID {} out of range (0-11)", card.id));
            }
            if card.count < 1 || card.count > 3 {
                return Err(format!(
                    "Card {} count {} out of range (1-3)",
                    card.id, card.count
                ));
            }
        }

        Ok(())
    }
}

/// Attribute validation result for a set.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum AttributeMatch {
    AllSame,
    AllDifferent,
}

/// Validation details for a set.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SetValidation {
    /// Shape attribute validation
    pub shape: AttributeMatch,

    /// Color attribute validation
    pub color: AttributeMatch,

    /// Count attribute validation
    pub count: AttributeMatch,

    /// Shading attribute validation
    pub shading: AttributeMatch,
}

/// A valid set of three cards.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidSet {
    /// IDs of three cards forming a set
    pub card_ids: [u8; 3],

    /// How each attribute satisfies set rule
    pub validation: SetValidation,
}

impl ValidSet {
    /// Checks if three cards form a valid set.
    pub fn check(cards: [&SetCard; 3]) -> Option<Self> {
        let shape = Self::check_attribute(cards[0].shape, cards[1].shape, cards[2].shape)?;
        let color = Self::check_attribute(cards[0].color, cards[1].color, cards[2].color)?;
        let count = Self::check_attribute(cards[0].count, cards[1].count, cards[2].count)?;
        let shading = Self::check_attribute(cards[0].shading, cards[1].shading, cards[2].shading)?;

        Some(ValidSet {
            card_ids: [cards[0].id, cards[1].id, cards[2].id],
            validation: SetValidation {
                shape,
                color,
                count,
                shading,
            },
        })
    }

    fn check_attribute<T: Eq>(a: T, b: T, c: T) -> Option<AttributeMatch> {
        if a == b && b == c {
            Some(AttributeMatch::AllSame)
        } else if a != b && b != c && a != c {
            Some(AttributeMatch::AllDifferent)
        } else {
            None // Invalid: two same, one different
        }
    }
}

/// Performance metrics for a solver run.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerformanceMetrics {
    /// Which approach was used
    pub approach: SolverApproach,

    /// Identifier of puzzle solved
    pub puzzle_id: String,

    /// Whether solution is correct
    pub correct: bool,

    /// Total solve time in milliseconds
    pub time_ms: u64,

    /// CLIPS rule firings or LLM reasoning steps
    pub iterations: u64,

    /// Number of LLM API calls
    pub llm_calls: u64,

    /// Total tokens consumed
    pub tokens_used: u64,

    /// Peak memory usage in kilobytes (optional)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub memory_peak_kb: Option<u64>,
}

/// Aggregated comparison across approaches.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ComparisonReport {
    /// Puzzle that was solved
    pub puzzle_id: String,

    /// Metrics for each approach
    pub results: Vec<PerformanceMetrics>,

    /// Best approach by composite score
    #[serde(skip_serializing_if = "Option::is_none")]
    pub winner: Option<SolverApproach>,

    /// Human-readable comparison summary
    pub summary: String,
}

impl ComparisonReport {
    /// Determines the winning approach based on correctness, time, and tokens.
    pub fn determine_winner(&mut self) {
        // First, filter to only correct solutions
        let correct_results: Vec<_> = self.results.iter().filter(|r| r.correct).collect();

        if correct_results.is_empty() {
            self.winner = None;
            return;
        }

        // Among correct solutions, prefer: lowest tokens, then lowest time
        let best = correct_results
            .iter()
            .min_by(|a, b| {
                // Primary: fewer tokens is better
                let token_cmp = a.tokens_used.cmp(&b.tokens_used);
                if token_cmp != std::cmp::Ordering::Equal {
                    return token_cmp;
                }
                // Secondary: faster is better
                a.time_ms.cmp(&b.time_ms)
            })
            .unwrap();

        self.winner = Some(best.approach);
    }

    /// Generates a summary string.
    pub fn generate_summary(&mut self) {
        let mut parts = Vec::new();

        for result in &self.results {
            let status = if result.correct { "✓" } else { "✗" };
            parts.push(format!(
                "{} {} in {}ms ({} tokens)",
                result.approach, status, result.time_ms, result.tokens_used
            ));
        }

        if let Some(winner) = self.winner {
            parts.push(format!("Winner: {}", winner));
        }

        self.summary = parts.join(". ");
    }
}

// ============================================================================
// Sudoku Solver Implementations
// ============================================================================

/// Result of solving a Sudoku puzzle.
#[derive(Debug, Clone)]
pub struct SudokuSolveResult {
    /// The solved grid (or partial if failed)
    pub solution: [[u8; 9]; 9],
    /// Whether the puzzle was solved completely
    pub solved: bool,
    /// Number of iterations/rule firings
    pub iterations: u64,
    /// Number of LLM calls (for hybrid/llm-only)
    pub llm_calls: u64,
    /// Tokens used (for hybrid/llm-only)
    pub tokens_used: u64,
}

/// Validates a Sudoku solution against the puzzle constraints.
pub fn validate_sudoku_solution(puzzle: &SudokuPuzzle, solution: &[[u8; 9]; 9]) -> bool {
    puzzle.validate_solution(solution).is_ok()
}

// Sudoku solver functions removed — real solving now uses nxusKit ClipsSession
// in each app's main.rs. See internal/reference/puzzler-pure-rust-solver.rs.

// ============================================================================
// Set Game Solver Implementations
// ============================================================================

/// Result of finding sets in a Set Game hand.
#[derive(Debug, Clone)]
pub struct SetGameResult {
    /// Valid sets found
    pub sets: Vec<ValidSet>,
    /// Whether at least one set was found
    pub found_any: bool,
    /// Number of iterations/rule firings
    pub iterations: u64,
    /// Number of LLM calls (for hybrid/llm-only)
    pub llm_calls: u64,
    /// Tokens used (for hybrid/llm-only)
    pub tokens_used: u64,
}

// Set game solver functions removed — real solving now uses nxusKit ClipsSession
// in each app's main.rs. See internal/reference/puzzler-pure-rust-solver.rs.
