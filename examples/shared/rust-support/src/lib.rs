//! LLM Pattern Types
//!
//! Support types for nxusKit LLM integration pattern examples.
//! These types define common patterns for using LLMs with validation,
//! retry logic, and CLIPS expert system integration.
//!
//! # Patterns
//!
//! - **Arbiter**: Auto-retry LLM with CLIPS validation
//! - **Puzzler**: Puzzle-solving with LLM + constraint propagation
//! - **Ruler**: Progressive rule generation with LLM
//! - **Racer**: Benchmark competition between LLM and CLIPS solvers

pub mod arbiter;
pub mod puzzler;
pub mod racer;
pub mod ruler;

// Re-export commonly used types
pub use puzzler::{
    ComparisonReport, Difficulty, PerformanceMetrics, SetCard, SetHand, SetValidation,
    SolverApproach, SudokuPuzzle, ValidSet,
};

pub use arbiter::{
    AdjustAction, ConclusionType, EvalStatus, EvaluationResult, FailureStrategy, FailureType,
    KnobAdjustment, RetryAttempt, SolverConfig, SolverResult,
};

pub use ruler::{
    Complexity, ErrorType, GeneratedRules, ProgressiveExample, ProgressiveExamples,
    RuleDescription, SaveFormat, SavedRules, ValidationError, ValidationResult, ValidationStatus,
};

pub use racer::{
    BenchmarkReport, Problem, ProblemDifficulty, ProblemRegistry, ProblemType, RaceResult,
    RaceWinner, Runner, RunnerConfig, RunnerResult, RunnerStats, RunnerType, ScoringMode,
    ScoringWeights,
};
