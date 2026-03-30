//! Racer: CLIPS vs LLM head-to-head competition.
//!
//! This module provides types and functions for running head-to-head races
//! between CLIPS rule-based solving and LLM reasoning on logic problems.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Problem category for racing.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ProblemType {
    /// Logic puzzles (e.g., Einstein's riddle).
    LogicPuzzle,
    /// Classification tasks.
    Classification,
    /// Constraint satisfaction problems.
    ConstraintSatisfaction,
}

impl std::fmt::Display for ProblemType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::LogicPuzzle => write!(f, "logic_puzzle"),
            Self::Classification => write!(f, "classification"),
            Self::ConstraintSatisfaction => write!(f, "constraint_satisfaction"),
        }
    }
}

impl std::str::FromStr for ProblemType {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().replace('-', "_").as_str() {
            "logic_puzzle" | "logic" => Ok(Self::LogicPuzzle),
            "classification" | "classify" => Ok(Self::Classification),
            "constraint_satisfaction" | "constraint" | "csp" => Ok(Self::ConstraintSatisfaction),
            _ => Err(format!(
                "Invalid problem type: '{}'. Valid: logic_puzzle, classification, constraint_satisfaction",
                s
            )),
        }
    }
}

/// Problem difficulty level.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ProblemDifficulty {
    Easy,
    Medium,
    Hard,
}

impl std::fmt::Display for ProblemDifficulty {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Easy => write!(f, "easy"),
            Self::Medium => write!(f, "medium"),
            Self::Hard => write!(f, "hard"),
        }
    }
}

impl std::str::FromStr for ProblemDifficulty {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "easy" => Ok(Self::Easy),
            "medium" => Ok(Self::Medium),
            "hard" => Ok(Self::Hard),
            _ => Err(format!(
                "Invalid difficulty: '{}'. Valid: easy, medium, hard",
                s
            )),
        }
    }
}

/// A challenge for the Racer with known solution.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Problem {
    /// Unique identifier.
    pub id: String,
    /// Human-readable name.
    pub name: String,
    /// Problem category.
    #[serde(rename = "type")]
    pub problem_type: ProblemType,
    /// Problem description.
    pub description: String,
    /// Problem-specific input data.
    pub input_data: serde_json::Value,
    /// Correct answer.
    pub expected_solution: serde_json::Value,
    /// Path to CLIPS rules.
    pub clips_rules_path: String,
    /// Problem difficulty.
    pub difficulty: ProblemDifficulty,
}

impl Problem {
    /// Create a new problem.
    pub fn new(
        name: impl Into<String>,
        problem_type: ProblemType,
        description: impl Into<String>,
    ) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            name: name.into(),
            problem_type,
            description: description.into(),
            input_data: serde_json::Value::Null,
            expected_solution: serde_json::Value::Null,
            clips_rules_path: String::new(),
            difficulty: ProblemDifficulty::Medium,
        }
    }

    /// Set input data.
    pub fn with_input(mut self, input: serde_json::Value) -> Self {
        self.input_data = input;
        self
    }

    /// Set expected solution.
    pub fn with_solution(mut self, solution: serde_json::Value) -> Self {
        self.expected_solution = solution;
        self
    }

    /// Set CLIPS rules path.
    pub fn with_rules_path(mut self, path: impl Into<String>) -> Self {
        self.clips_rules_path = path.into();
        self
    }

    /// Set difficulty.
    pub fn with_difficulty(mut self, difficulty: ProblemDifficulty) -> Self {
        self.difficulty = difficulty;
        self
    }
}

/// Type of runner approach.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RunnerType {
    /// CLIPS rule-based solving.
    Clips,
    /// LLM reasoning.
    Llm,
}

impl std::fmt::Display for RunnerType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Clips => write!(f, "clips"),
            Self::Llm => write!(f, "llm"),
        }
    }
}

/// Configuration for a runner.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerConfig {
    /// Max execution time in milliseconds.
    #[serde(default = "default_timeout")]
    pub timeout_ms: u64,
    /// LLM model (for LLM runner).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub model: Option<String>,
    /// CLIPS rules path (for CLIPS runner).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub rules_path: Option<String>,
}

fn default_timeout() -> u64 {
    60_000 // 60 seconds
}

impl Default for RunnerConfig {
    fn default() -> Self {
        Self {
            timeout_ms: default_timeout(),
            model: None,
            rules_path: None,
        }
    }
}

impl RunnerConfig {
    /// Create a new runner config with default timeout.
    pub fn new() -> Self {
        Self::default()
    }

    /// Set timeout.
    pub fn with_timeout_ms(mut self, timeout_ms: u64) -> Self {
        self.timeout_ms = timeout_ms;
        self
    }

    /// Set LLM model.
    pub fn with_model(mut self, model: impl Into<String>) -> Self {
        self.model = Some(model.into());
        self
    }

    /// Set CLIPS rules path.
    pub fn with_rules_path(mut self, path: impl Into<String>) -> Self {
        self.rules_path = Some(path.into());
        self
    }
}

/// An approach that attempts to solve a problem.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Runner {
    /// Unique identifier.
    pub id: String,
    /// Runner type.
    #[serde(rename = "type")]
    pub runner_type: RunnerType,
    /// Display name.
    pub name: String,
    /// Runner configuration.
    pub config: RunnerConfig,
}

impl Runner {
    /// Create a CLIPS runner.
    pub fn clips(name: impl Into<String>, rules_path: impl Into<String>) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            runner_type: RunnerType::Clips,
            name: name.into(),
            config: RunnerConfig::new().with_rules_path(rules_path),
        }
    }

    /// Create an LLM runner.
    pub fn llm(name: impl Into<String>, model: impl Into<String>) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            runner_type: RunnerType::Llm,
            name: name.into(),
            config: RunnerConfig::new().with_model(model),
        }
    }

    /// Set custom timeout.
    pub fn with_timeout_ms(mut self, timeout_ms: u64) -> Self {
        self.config.timeout_ms = timeout_ms;
        self
    }
}

/// Output from a single runner execution.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerResult {
    /// Reference to Runner.
    pub runner_id: String,
    /// Reference to Problem.
    pub problem_id: String,
    /// Runner's answer.
    pub answer: serde_json::Value,
    /// Whether answer matches expected.
    pub correct: bool,
    /// Execution time in milliseconds.
    pub time_ms: u64,
    /// Whether runner timed out.
    pub timed_out: bool,
    /// Tokens consumed (LLM only).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tokens_used: Option<u64>,
    /// Intermediate reasoning steps.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reasoning: Option<String>,
    /// Error message if failed.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<String>,
}

impl RunnerResult {
    /// Create a successful result.
    pub fn success(
        runner_id: impl Into<String>,
        problem_id: impl Into<String>,
        answer: serde_json::Value,
        correct: bool,
        time_ms: u64,
    ) -> Self {
        Self {
            runner_id: runner_id.into(),
            problem_id: problem_id.into(),
            answer,
            correct,
            time_ms,
            timed_out: false,
            tokens_used: None,
            reasoning: None,
            error: None,
        }
    }

    /// Create a timeout result.
    pub fn timeout(runner_id: impl Into<String>, problem_id: impl Into<String>) -> Self {
        Self {
            runner_id: runner_id.into(),
            problem_id: problem_id.into(),
            answer: serde_json::Value::Null,
            correct: false,
            time_ms: 60_000,
            timed_out: true,
            tokens_used: None,
            reasoning: None,
            error: Some("Execution timed out".to_string()),
        }
    }

    /// Create a failed result.
    pub fn failed(
        runner_id: impl Into<String>,
        problem_id: impl Into<String>,
        error: impl Into<String>,
        time_ms: u64,
    ) -> Self {
        Self {
            runner_id: runner_id.into(),
            problem_id: problem_id.into(),
            answer: serde_json::Value::Null,
            correct: false,
            time_ms,
            timed_out: false,
            tokens_used: None,
            reasoning: None,
            error: Some(error.into()),
        }
    }

    /// Set tokens used (for LLM runner).
    pub fn with_tokens(mut self, tokens: u64) -> Self {
        self.tokens_used = Some(tokens);
        self
    }

    /// Set reasoning steps.
    pub fn with_reasoning(mut self, reasoning: impl Into<String>) -> Self {
        self.reasoning = Some(reasoning.into());
        self
    }
}

/// How race winner is determined.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum ScoringMode {
    /// Fastest correct answer wins.
    #[default]
    Speed,
    /// Most complete answer wins.
    Accuracy,
    /// Weighted combination of speed and accuracy.
    Composite,
}

impl std::fmt::Display for ScoringMode {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Speed => write!(f, "speed"),
            Self::Accuracy => write!(f, "accuracy"),
            Self::Composite => write!(f, "composite"),
        }
    }
}

impl std::str::FromStr for ScoringMode {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "speed" => Ok(Self::Speed),
            "accuracy" => Ok(Self::Accuracy),
            "composite" => Ok(Self::Composite),
            _ => Err(format!(
                "Invalid scoring mode: '{}'. Valid: speed, accuracy, composite",
                s
            )),
        }
    }
}

/// Weights for composite scoring.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ScoringWeights {
    /// Weight for speed (0.0-1.0).
    pub time_weight: f64,
    /// Weight for correctness (0.0-1.0).
    pub accuracy_weight: f64,
}

impl Default for ScoringWeights {
    fn default() -> Self {
        Self {
            time_weight: 0.5,
            accuracy_weight: 0.5,
        }
    }
}

/// Race winner.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum RaceWinner {
    Clips,
    Llm,
    Tie,
    None,
}

impl std::fmt::Display for RaceWinner {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Clips => write!(f, "clips"),
            Self::Llm => write!(f, "llm"),
            Self::Tie => write!(f, "tie"),
            Self::None => write!(f, "none"),
        }
    }
}

/// Combined outcome of a head-to-head race.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RaceResult {
    /// Unique identifier.
    pub id: String,
    /// Reference to Problem.
    pub problem_id: String,
    /// CLIPS runner outcome.
    pub clips_result: RunnerResult,
    /// LLM runner outcome.
    pub llm_result: RunnerResult,
    /// Race winner.
    pub winner: RaceWinner,
    /// How winner was determined.
    pub scoring_mode: ScoringMode,
    /// When judged.
    pub scored_at: chrono::DateTime<chrono::Utc>,
    /// Margin of victory in milliseconds.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub margin_ms: Option<i64>,
}

impl RaceResult {
    /// Create a new race result and determine winner.
    pub fn new(
        problem_id: impl Into<String>,
        clips_result: RunnerResult,
        llm_result: RunnerResult,
        scoring_mode: ScoringMode,
    ) -> Self {
        let winner = Self::determine_winner(&clips_result, &llm_result, scoring_mode);
        let margin_ms = if clips_result.correct && llm_result.correct {
            Some(llm_result.time_ms as i64 - clips_result.time_ms as i64)
        } else {
            None
        };

        Self {
            id: uuid::Uuid::new_v4().to_string(),
            problem_id: problem_id.into(),
            clips_result,
            llm_result,
            winner,
            scoring_mode,
            scored_at: chrono::Utc::now(),
            margin_ms,
        }
    }

    /// Determine the winner based on scoring mode.
    fn determine_winner(
        clips: &RunnerResult,
        llm: &RunnerResult,
        scoring_mode: ScoringMode,
    ) -> RaceWinner {
        match (clips.correct, llm.correct) {
            (false, false) => RaceWinner::None,
            (true, false) => RaceWinner::Clips,
            (false, true) => RaceWinner::Llm,
            (true, true) => match scoring_mode {
                ScoringMode::Speed | ScoringMode::Composite => {
                    if clips.time_ms < llm.time_ms {
                        RaceWinner::Clips
                    } else if llm.time_ms < clips.time_ms {
                        RaceWinner::Llm
                    } else {
                        RaceWinner::Tie
                    }
                }
                ScoringMode::Accuracy => {
                    // For accuracy mode with both correct, compare by completeness
                    // (here we just use time as tiebreaker since both are correct)
                    if clips.time_ms < llm.time_ms {
                        RaceWinner::Clips
                    } else if llm.time_ms < clips.time_ms {
                        RaceWinner::Llm
                    } else {
                        RaceWinner::Tie
                    }
                }
            },
        }
    }
}

/// Aggregate statistics for a runner across benchmark runs.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerStats {
    /// Average execution time.
    pub mean_time_ms: f64,
    /// Standard deviation of time.
    pub std_dev_time_ms: f64,
    /// Fastest run.
    pub min_time_ms: u64,
    /// Slowest run.
    pub max_time_ms: u64,
    /// Percentage of successful runs.
    pub success_rate: f64,
    /// Percentage of timeouts.
    pub timeout_rate: f64,
}

impl RunnerStats {
    /// Calculate statistics from a list of runner results.
    pub fn from_results(results: &[RunnerResult]) -> Self {
        if results.is_empty() {
            return Self {
                mean_time_ms: 0.0,
                std_dev_time_ms: 0.0,
                min_time_ms: 0,
                max_time_ms: 0,
                success_rate: 0.0,
                timeout_rate: 0.0,
            };
        }

        let times: Vec<f64> = results.iter().map(|r| r.time_ms as f64).collect();
        let mean = mean(&times);
        let std_dev = std_dev(&times);
        let min = results.iter().map(|r| r.time_ms).min().unwrap_or(0);
        let max = results.iter().map(|r| r.time_ms).max().unwrap_or(0);

        let success_count = results.iter().filter(|r| r.correct).count();
        let timeout_count = results.iter().filter(|r| r.timed_out).count();
        let total = results.len() as f64;

        Self {
            mean_time_ms: mean,
            std_dev_time_ms: std_dev,
            min_time_ms: min,
            max_time_ms: max,
            success_rate: success_count as f64 / total,
            timeout_rate: timeout_count as f64 / total,
        }
    }
}

/// Aggregate statistics from multiple race runs.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BenchmarkReport {
    /// Unique identifier.
    pub id: String,
    /// Reference to Problem.
    pub problem_id: String,
    /// Number of iterations.
    pub total_runs: u32,
    /// CLIPS aggregate stats.
    pub clips_stats: RunnerStats,
    /// LLM aggregate stats.
    pub llm_stats: RunnerStats,
    /// CLIPS win percentage.
    pub clips_win_rate: f64,
    /// LLM win percentage.
    pub llm_win_rate: f64,
    /// Tie percentage.
    pub tie_rate: f64,
    /// Report generation time.
    pub created_at: chrono::DateTime<chrono::Utc>,
}

impl BenchmarkReport {
    /// Create a benchmark report from race results.
    pub fn from_races(problem_id: impl Into<String>, races: &[RaceResult]) -> Self {
        let clips_results: Vec<_> = races.iter().map(|r| r.clips_result.clone()).collect();
        let llm_results: Vec<_> = races.iter().map(|r| r.llm_result.clone()).collect();

        let clips_wins = races
            .iter()
            .filter(|r| r.winner == RaceWinner::Clips)
            .count();
        let llm_wins = races.iter().filter(|r| r.winner == RaceWinner::Llm).count();
        let ties = races.iter().filter(|r| r.winner == RaceWinner::Tie).count();
        let total = races.len() as f64;

        Self {
            id: uuid::Uuid::new_v4().to_string(),
            problem_id: problem_id.into(),
            total_runs: races.len() as u32,
            clips_stats: RunnerStats::from_results(&clips_results),
            llm_stats: RunnerStats::from_results(&llm_results),
            clips_win_rate: clips_wins as f64 / total,
            llm_win_rate: llm_wins as f64 / total,
            tie_rate: ties as f64 / total,
            created_at: chrono::Utc::now(),
        }
    }
}

/// Built-in problem registry.
#[derive(Debug, Default)]
pub struct ProblemRegistry {
    problems: HashMap<String, Problem>,
}

impl ProblemRegistry {
    /// Create a new empty registry.
    pub fn new() -> Self {
        Self::default()
    }

    /// Register a problem.
    pub fn register(&mut self, problem: Problem) {
        self.problems.insert(problem.name.clone(), problem);
    }

    /// Get a problem by name.
    pub fn get(&self, name: &str) -> Option<&Problem> {
        self.problems.get(name)
    }

    /// List all problem names.
    pub fn list(&self) -> Vec<&str> {
        self.problems.keys().map(|s| s.as_str()).collect()
    }

    /// Filter problems by type.
    pub fn by_type(&self, problem_type: ProblemType) -> Vec<&Problem> {
        self.problems
            .values()
            .filter(|p| p.problem_type == problem_type)
            .collect()
    }

    /// Filter problems by difficulty.
    pub fn by_difficulty(&self, difficulty: ProblemDifficulty) -> Vec<&Problem> {
        self.problems
            .values()
            .filter(|p| p.difficulty == difficulty)
            .collect()
    }

    /// Find similar problem names using string similarity.
    pub fn find_similar(&self, name: &str, threshold: f64) -> Vec<&str> {
        self.problems
            .keys()
            .filter(|k| {
                strsim::normalized_levenshtein(k.to_lowercase().as_str(), &name.to_lowercase())
                    >= threshold
            })
            .map(|s| s.as_str())
            .collect()
    }
}

// Statistics utility functions

/// Calculate mean of a slice of f64 values.
pub fn mean(values: &[f64]) -> f64 {
    if values.is_empty() {
        return 0.0;
    }
    values.iter().sum::<f64>() / values.len() as f64
}

/// Calculate standard deviation of a slice of f64 values.
pub fn std_dev(values: &[f64]) -> f64 {
    if values.len() < 2 {
        return 0.0;
    }
    let m = mean(values);
    let variance = values.iter().map(|v| (v - m).powi(2)).sum::<f64>() / (values.len() - 1) as f64;
    variance.sqrt()
}

/// Calculate minimum of a slice of u64 values.
pub fn min_u64(values: &[u64]) -> u64 {
    values.iter().copied().min().unwrap_or(0)
}

/// Calculate maximum of a slice of u64 values.
pub fn max_u64(values: &[u64]) -> u64 {
    values.iter().copied().max().unwrap_or(0)
}

// ============================================================================
// CLIPS Runner
// ============================================================================

/// CLIPS-based runner for solving problems.
#[derive(Debug, Clone)]
pub struct ClipsRunner {
    /// Runner configuration.
    pub config: RunnerConfig,
    /// Path to CLIPS rules file.
    pub rules_path: String,
}

impl ClipsRunner {
    /// Create a new CLIPS runner.
    pub fn new(rules_path: impl Into<String>) -> Self {
        Self {
            config: RunnerConfig::default(),
            rules_path: rules_path.into(),
        }
    }

    /// Set timeout.
    pub fn with_timeout_ms(mut self, timeout_ms: u64) -> Self {
        self.config.timeout_ms = timeout_ms;
        self
    }

    /// Run the CLIPS solver on a problem.
    ///
    /// This is a simulation - in production, it would use CLIPS FFI bindings.
    pub fn run(&self, problem: &Problem) -> RunnerResult {
        let start = std::time::Instant::now();

        // Simulate CLIPS execution time based on problem type
        let sleep_time = match problem.problem_type {
            ProblemType::LogicPuzzle => std::time::Duration::from_millis(40),
            ProblemType::Classification => std::time::Duration::from_millis(10),
            ProblemType::ConstraintSatisfaction => std::time::Duration::from_millis(25),
        };

        std::thread::sleep(sleep_time);

        // Get simulated answer
        let (answer, correct) = self.simulate_answer(problem);
        let elapsed = start.elapsed().as_millis() as u64;

        RunnerResult::success("clips-runner", &problem.id, answer, correct, elapsed)
    }

    /// Simulate an answer for a problem.
    fn simulate_answer(&self, problem: &Problem) -> (serde_json::Value, bool) {
        match problem.name.as_str() {
            "einstein-riddle" => (serde_json::json!({"fish_owner": "German"}), true),
            "family-relations" => (serde_json::json!({"relationships_found": true}), true),
            "animal-classification" => (
                serde_json::json!({"classifications": [{"animal": "dog", "class": "mammal"}]}),
                true,
            ),
            _ => {
                if problem.expected_solution != serde_json::Value::Null {
                    (problem.expected_solution.clone(), true)
                } else {
                    (serde_json::json!({"result": "unknown"}), false)
                }
            }
        }
    }
}

// ============================================================================
// LLM Runner
// ============================================================================

/// LLM-based runner for solving problems.
#[derive(Debug, Clone)]
pub struct LlmRunner {
    /// Runner configuration.
    pub config: RunnerConfig,
    /// Model to use.
    pub model: String,
}

impl LlmRunner {
    /// Create a new LLM runner.
    pub fn new(model: impl Into<String>) -> Self {
        Self {
            config: RunnerConfig::default(),
            model: model.into(),
        }
    }

    /// Set timeout.
    pub fn with_timeout_ms(mut self, timeout_ms: u64) -> Self {
        self.config.timeout_ms = timeout_ms;
        self
    }

    /// Run the LLM solver on a problem.
    ///
    /// This is a simulation - in production, it would call the LLM API.
    pub fn run(&self, problem: &Problem) -> RunnerResult {
        let start = std::time::Instant::now();

        // Simulate LLM execution time based on difficulty
        let (sleep_time, tokens) = match problem.difficulty {
            ProblemDifficulty::Easy => (std::time::Duration::from_millis(1500), 500u64),
            ProblemDifficulty::Medium => (std::time::Duration::from_millis(2500), 1000),
            ProblemDifficulty::Hard => (std::time::Duration::from_millis(3500), 1500),
        };

        std::thread::sleep(sleep_time);

        // Get simulated answer
        let (answer, correct, reasoning) = self.simulate_answer(problem);
        let elapsed = start.elapsed().as_millis() as u64;

        RunnerResult::success("llm-runner", &problem.id, answer, correct, elapsed)
            .with_tokens(tokens)
            .with_reasoning(reasoning)
    }

    /// Simulate an answer for a problem.
    fn simulate_answer(&self, problem: &Problem) -> (serde_json::Value, bool, String) {
        match problem.name.as_str() {
            "einstein-riddle" => (
                serde_json::json!({"fish_owner": "German"}),
                true,
                "Through constraint propagation, the German owns the fish.".to_string(),
            ),
            "family-relations" => (
                serde_json::json!({"relationships_found": true}),
                true,
                "Analyzed family tree to find relationships.".to_string(),
            ),
            "animal-classification" => (
                serde_json::json!({"classifications": [{"animal": "dog", "class": "mammal"}]}),
                true,
                "Classified animals by characteristics.".to_string(),
            ),
            _ => {
                if problem.expected_solution != serde_json::Value::Null {
                    (
                        problem.expected_solution.clone(),
                        true,
                        "Solved by pattern matching".to_string(),
                    )
                } else {
                    (
                        serde_json::json!({"result": "unknown"}),
                        false,
                        "Unable to determine solution".to_string(),
                    )
                }
            }
        }
    }
}

// ============================================================================
// Racer
// ============================================================================

/// Racer orchestrates head-to-head races between CLIPS and LLM.
#[derive(Debug, Clone)]
pub struct Racer {
    /// CLIPS runner.
    pub clips_runner: ClipsRunner,
    /// LLM runner.
    pub llm_runner: LlmRunner,
    /// Scoring mode for determining winners.
    pub scoring_mode: ScoringMode,
    /// Timeout in milliseconds.
    pub timeout_ms: u64,
}

impl Racer {
    /// Create a new racer.
    pub fn new(clips_rules_path: impl Into<String>, llm_model: impl Into<String>) -> Self {
        Self {
            clips_runner: ClipsRunner::new(clips_rules_path),
            llm_runner: LlmRunner::new(llm_model),
            scoring_mode: ScoringMode::Speed,
            timeout_ms: 60_000,
        }
    }

    /// Set scoring mode.
    pub fn with_scoring_mode(mut self, mode: ScoringMode) -> Self {
        self.scoring_mode = mode;
        self
    }

    /// Set timeout.
    pub fn with_timeout_ms(mut self, timeout_ms: u64) -> Self {
        self.timeout_ms = timeout_ms;
        self.clips_runner.config.timeout_ms = timeout_ms;
        self.llm_runner.config.timeout_ms = timeout_ms;
        self
    }

    /// Run a race between CLIPS and LLM.
    pub fn race(&self, problem: &Problem) -> RaceResult {
        // Run both (in production, these would be concurrent)
        let clips_result = self.clips_runner.run(problem);
        let llm_result = self.llm_runner.run(problem);

        RaceResult::new(&problem.id, clips_result, llm_result, self.scoring_mode)
    }

    /// Run multiple races for benchmarking.
    pub fn benchmark(&self, problem: &Problem, runs: u32) -> BenchmarkReport {
        let races: Vec<RaceResult> = (0..runs).map(|_| self.race(problem)).collect();
        BenchmarkReport::from_races(&problem.id, &races)
    }
}

/// Run a single race (convenience function).
pub fn run_race(
    problem: &Problem,
    clips_rules_path: &str,
    llm_model: &str,
    scoring_mode: ScoringMode,
) -> RaceResult {
    Racer::new(clips_rules_path, llm_model)
        .with_scoring_mode(scoring_mode)
        .race(problem)
}

// ============================================================================
// Judge
// ============================================================================

/// Judge for determining race winners.
#[derive(Debug, Clone)]
pub struct Judge {
    /// Scoring mode.
    pub scoring_mode: ScoringMode,
    /// Scoring weights for composite mode.
    pub weights: ScoringWeights,
}

impl Default for Judge {
    fn default() -> Self {
        Self::new()
    }
}

impl Judge {
    /// Create a new judge.
    pub fn new() -> Self {
        Self {
            scoring_mode: ScoringMode::Speed,
            weights: ScoringWeights::default(),
        }
    }

    /// Set scoring mode.
    pub fn with_scoring_mode(mut self, mode: ScoringMode) -> Self {
        self.scoring_mode = mode;
        self
    }

    /// Determine winner between two results.
    pub fn determine_winner(&self, clips: &RunnerResult, llm: &RunnerResult) -> RaceWinner {
        match (clips.correct, llm.correct) {
            (false, false) => RaceWinner::None,
            (true, false) => RaceWinner::Clips,
            (false, true) => RaceWinner::Llm,
            (true, true) => self.compare_correct_results(clips, llm),
        }
    }

    fn compare_correct_results(&self, clips: &RunnerResult, llm: &RunnerResult) -> RaceWinner {
        match self.scoring_mode {
            ScoringMode::Speed => self.compare_by_speed(clips, llm),
            ScoringMode::Accuracy => self.compare_by_accuracy(clips, llm),
            ScoringMode::Composite => self.compare_by_composite(clips, llm),
        }
    }

    fn compare_by_speed(&self, clips: &RunnerResult, llm: &RunnerResult) -> RaceWinner {
        match clips.time_ms.cmp(&llm.time_ms) {
            std::cmp::Ordering::Less => RaceWinner::Clips,
            std::cmp::Ordering::Greater => RaceWinner::Llm,
            std::cmp::Ordering::Equal => RaceWinner::Tie,
        }
    }

    fn compare_by_accuracy(&self, clips: &RunnerResult, llm: &RunnerResult) -> RaceWinner {
        // For accuracy mode with both correct, use answer size as proxy
        let clips_size = clips.answer.to_string().len();
        let llm_size = llm.answer.to_string().len();

        match clips_size.cmp(&llm_size) {
            std::cmp::Ordering::Greater => RaceWinner::Clips,
            std::cmp::Ordering::Less => RaceWinner::Llm,
            std::cmp::Ordering::Equal => self.compare_by_speed(clips, llm),
        }
    }

    fn compare_by_composite(&self, clips: &RunnerResult, llm: &RunnerResult) -> RaceWinner {
        let clips_score = self.calculate_composite_score(clips);
        let llm_score = self.calculate_composite_score(llm);

        if (clips_score - llm_score).abs() < 0.001 {
            RaceWinner::Tie
        } else if clips_score > llm_score {
            RaceWinner::Clips
        } else {
            RaceWinner::Llm
        }
    }

    fn calculate_composite_score(&self, result: &RunnerResult) -> f64 {
        let correctness = if result.correct { 1.0 } else { 0.0 };
        let speed = if result.time_ms > 0 {
            1000.0 / result.time_ms as f64
        } else {
            0.0
        };

        (correctness * self.weights.accuracy_weight) + (speed * self.weights.time_weight)
    }
}

/// Compare two JSON answers for equality.
pub fn compare_answers(a: &serde_json::Value, b: &serde_json::Value) -> bool {
    a == b
}

/// Calculate margin of victory in milliseconds.
pub fn calculate_margin(clips: &RunnerResult, llm: &RunnerResult) -> i64 {
    llm.time_ms as i64 - clips.time_ms as i64
}

// ============================================================================
// Problem Loading
// ============================================================================

/// Load a problem from a JSON file.
pub fn load_problem_from_file(path: &std::path::Path) -> Result<Problem, String> {
    let content =
        std::fs::read_to_string(path).map_err(|e| format!("Failed to read file: {}", e))?;

    serde_json::from_str(&content).map_err(|e| format!("Failed to parse JSON: {}", e))
}

/// Load all problems from a directory.
pub fn load_problems_from_directory(dir: &std::path::Path) -> Result<Vec<Problem>, String> {
    let mut problems = Vec::new();

    let entries = std::fs::read_dir(dir).map_err(|e| format!("Failed to read directory: {}", e))?;

    for entry in entries.flatten() {
        let path = entry.path();
        if path.extension().is_some_and(|ext| ext == "json")
            && let Ok(problem) = load_problem_from_file(&path)
        {
            problems.push(problem);
        }
    }

    Ok(problems)
}

/// Get the built-in problem registry with default problems.
pub fn get_builtin_registry() -> ProblemRegistry {
    let mut registry = ProblemRegistry::new();

    registry.register(
        Problem::new(
            "einstein-riddle",
            ProblemType::LogicPuzzle,
            "Five houses puzzle: determine who owns the fish",
        )
        .with_difficulty(ProblemDifficulty::Hard)
        .with_solution(serde_json::json!({"fish_owner": "German"})),
    );

    registry.register(
        Problem::new(
            "family-relations",
            ProblemType::ConstraintSatisfaction,
            "Infer family relationships from parent-child facts",
        )
        .with_difficulty(ProblemDifficulty::Medium)
        .with_solution(serde_json::json!({"relationships_found": true})),
    );

    registry.register(
        Problem::new(
            "animal-classification",
            ProblemType::Classification,
            "Classify animals by their characteristics",
        )
        .with_difficulty(ProblemDifficulty::Easy)
        .with_solution(
            serde_json::json!({"classifications": [{"animal": "dog", "class": "mammal"}]}),
        ),
    );

    registry
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_problem_type_parsing() {
        assert_eq!(
            "logic_puzzle".parse::<ProblemType>().unwrap(),
            ProblemType::LogicPuzzle
        );
        assert_eq!(
            "classification".parse::<ProblemType>().unwrap(),
            ProblemType::Classification
        );
        assert!("invalid".parse::<ProblemType>().is_err());
    }

    #[test]
    fn test_scoring_mode_parsing() {
        assert_eq!("speed".parse::<ScoringMode>().unwrap(), ScoringMode::Speed);
        assert_eq!(
            "accuracy".parse::<ScoringMode>().unwrap(),
            ScoringMode::Accuracy
        );
        assert_eq!(
            "composite".parse::<ScoringMode>().unwrap(),
            ScoringMode::Composite
        );
    }

    #[test]
    fn test_runner_creation() {
        let clips = Runner::clips("CLIPS Runner", "rules/test.clp");
        assert_eq!(clips.runner_type, RunnerType::Clips);
        assert_eq!(clips.config.rules_path, Some("rules/test.clp".to_string()));

        let llm = Runner::llm("LLM Runner", "claude-haiku-4-5-20251001");
        assert_eq!(llm.runner_type, RunnerType::Llm);
        assert_eq!(
            llm.config.model,
            Some("claude-haiku-4-5-20251001".to_string())
        );
    }

    #[test]
    fn test_race_winner_determination() {
        // Both correct, CLIPS faster
        let clips = RunnerResult::success("c", "p", serde_json::json!({"a": 1}), true, 100);
        let llm = RunnerResult::success("l", "p", serde_json::json!({"a": 1}), true, 200);
        let race = RaceResult::new("p", clips, llm, ScoringMode::Speed);
        assert_eq!(race.winner, RaceWinner::Clips);

        // Only LLM correct
        let clips = RunnerResult::success("c", "p", serde_json::json!({"a": 1}), false, 100);
        let llm = RunnerResult::success("l", "p", serde_json::json!({"a": 1}), true, 200);
        let race = RaceResult::new("p", clips, llm, ScoringMode::Speed);
        assert_eq!(race.winner, RaceWinner::Llm);

        // Neither correct
        let clips = RunnerResult::success("c", "p", serde_json::json!(null), false, 100);
        let llm = RunnerResult::success("l", "p", serde_json::json!(null), false, 200);
        let race = RaceResult::new("p", clips, llm, ScoringMode::Speed);
        assert_eq!(race.winner, RaceWinner::None);
    }

    #[test]
    fn test_statistics() {
        let values = vec![10.0, 20.0, 30.0, 40.0, 50.0];
        assert!((mean(&values) - 30.0).abs() < 0.001);
        assert!((std_dev(&values) - 15.811).abs() < 0.01);
    }

    #[test]
    fn test_runner_stats() {
        let results = vec![
            RunnerResult::success("r", "p", serde_json::json!(1), true, 100),
            RunnerResult::success("r", "p", serde_json::json!(1), true, 200),
            RunnerResult::success("r", "p", serde_json::json!(1), false, 150),
        ];

        let stats = RunnerStats::from_results(&results);
        assert!((stats.mean_time_ms - 150.0).abs() < 0.1);
        assert_eq!(stats.min_time_ms, 100);
        assert_eq!(stats.max_time_ms, 200);
        assert!((stats.success_rate - 0.666).abs() < 0.01);
    }

    #[test]
    fn test_clips_runner() {
        let runner = ClipsRunner::new("rules/test.clp").with_timeout_ms(5000);
        assert_eq!(runner.config.timeout_ms, 5000);

        let problem = Problem::new("einstein-riddle", ProblemType::LogicPuzzle, "Test");
        let result = runner.run(&problem);

        assert!(result.correct);
        assert!(result.time_ms > 0);
    }

    #[test]
    fn test_llm_runner() {
        let runner = LlmRunner::new("test-model").with_timeout_ms(5000);
        assert_eq!(runner.model, "test-model");

        let problem = Problem::new("einstein-riddle", ProblemType::LogicPuzzle, "Test")
            .with_difficulty(ProblemDifficulty::Easy);
        let result = runner.run(&problem);

        assert!(result.correct);
        assert!(result.tokens_used.is_some());
        assert!(result.reasoning.is_some());
    }

    #[test]
    fn test_racer() {
        let racer = Racer::new("rules/test.clp", "test-model")
            .with_scoring_mode(ScoringMode::Speed)
            .with_timeout_ms(5000);

        assert_eq!(racer.scoring_mode, ScoringMode::Speed);
        assert_eq!(racer.timeout_ms, 5000);
    }

    #[test]
    fn test_judge() {
        let judge = Judge::new().with_scoring_mode(ScoringMode::Speed);

        // CLIPS faster
        let clips = RunnerResult::success("c", "p", serde_json::json!(1), true, 100);
        let llm = RunnerResult::success("l", "p", serde_json::json!(1), true, 200);
        assert_eq!(judge.determine_winner(&clips, &llm), RaceWinner::Clips);

        // LLM faster
        let clips = RunnerResult::success("c", "p", serde_json::json!(1), true, 200);
        let llm = RunnerResult::success("l", "p", serde_json::json!(1), true, 100);
        assert_eq!(judge.determine_winner(&clips, &llm), RaceWinner::Llm);

        // Neither correct
        let clips = RunnerResult::success("c", "p", serde_json::json!(1), false, 100);
        let llm = RunnerResult::success("l", "p", serde_json::json!(1), false, 100);
        assert_eq!(judge.determine_winner(&clips, &llm), RaceWinner::None);
    }

    #[test]
    fn test_compare_answers() {
        assert!(compare_answers(
            &serde_json::json!({"a": 1}),
            &serde_json::json!({"a": 1})
        ));
        assert!(!compare_answers(
            &serde_json::json!({"a": 1}),
            &serde_json::json!({"a": 2})
        ));
    }

    #[test]
    fn test_calculate_margin() {
        let clips = RunnerResult::success("c", "p", serde_json::json!(1), true, 100);
        let llm = RunnerResult::success("l", "p", serde_json::json!(1), true, 250);

        assert_eq!(calculate_margin(&clips, &llm), 150);
    }

    #[test]
    fn test_builtin_registry() {
        let registry = get_builtin_registry();

        assert!(registry.get("einstein-riddle").is_some());
        assert!(registry.get("family-relations").is_some());
        assert!(registry.get("animal-classification").is_some());
        assert!(registry.get("nonexistent").is_none());

        let einstein = registry.get("einstein-riddle").unwrap();
        assert_eq!(einstein.difficulty, ProblemDifficulty::Hard);
    }

    #[test]
    fn test_problem_registry_filters() {
        let registry = get_builtin_registry();

        let logic = registry.by_type(ProblemType::LogicPuzzle);
        assert!(!logic.is_empty());

        let easy = registry.by_difficulty(ProblemDifficulty::Easy);
        assert!(!easy.is_empty());
    }
}
