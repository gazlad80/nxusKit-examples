//! Ruler: Natural language to CLIPS rule generation.
//!
//! This module provides types and functions for generating CLIPS rules from
//! natural language descriptions using LLM-based code generation with
//! validation and retry logic.

use serde::{Deserialize, Serialize};

/// Target complexity level for rule generation.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize, Default)]
#[serde(rename_all = "lowercase")]
pub enum Complexity {
    /// Basic: deftemplate and simple defrule constructs.
    #[default]
    Basic,
    /// Intermediate: salience, test patterns, constraints.
    Intermediate,
    /// Advanced: deffunction, defmodule, complex patterns.
    Advanced,
}

impl std::fmt::Display for Complexity {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Basic => write!(f, "basic"),
            Self::Intermediate => write!(f, "intermediate"),
            Self::Advanced => write!(f, "advanced"),
        }
    }
}

impl std::str::FromStr for Complexity {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "basic" => Ok(Self::Basic),
            "intermediate" => Ok(Self::Intermediate),
            "advanced" => Ok(Self::Advanced),
            _ => Err(format!(
                "Invalid complexity: '{}'. Valid values: basic, intermediate, advanced",
                s
            )),
        }
    }
}

/// Natural language description of desired CLIPS behavior.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RuleDescription {
    /// Unique identifier.
    pub id: String,
    /// Natural language rule description.
    pub description: String,
    /// Target complexity level.
    pub complexity: Complexity,
    /// Optional domain keywords.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub domain_hints: Vec<String>,
}

impl RuleDescription {
    /// Create a new rule description.
    pub fn new(description: impl Into<String>) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            description: description.into(),
            complexity: Complexity::default(),
            domain_hints: Vec::new(),
        }
    }

    /// Set the complexity level.
    pub fn with_complexity(mut self, complexity: Complexity) -> Self {
        self.complexity = complexity;
        self
    }

    /// Add domain hints.
    pub fn with_domain_hints(mut self, hints: Vec<String>) -> Self {
        self.domain_hints = hints;
        self
    }
}

/// CLIPS source code produced by LLM.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GeneratedRules {
    /// Unique identifier.
    pub id: String,
    /// Reference to source RuleDescription.
    pub source_description_id: String,
    /// Generated CLIPS source code.
    pub clips_code: String,
    /// Attempt number (1-5).
    pub generation_attempt: u8,
    /// LLM model identifier.
    pub model_used: String,
    /// Total tokens consumed.
    pub tokens_used: u64,
    /// Time to generate in milliseconds.
    pub generation_time_ms: u64,
}

impl GeneratedRules {
    /// Create a new generated rules result.
    pub fn new(
        source_description_id: impl Into<String>,
        clips_code: impl Into<String>,
        model_used: impl Into<String>,
    ) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            source_description_id: source_description_id.into(),
            clips_code: clips_code.into(),
            generation_attempt: 1,
            model_used: model_used.into(),
            tokens_used: 0,
            generation_time_ms: 0,
        }
    }

    /// Set the attempt number.
    pub fn with_attempt(mut self, attempt: u8) -> Self {
        self.generation_attempt = attempt;
        self
    }

    /// Set token usage.
    pub fn with_tokens(mut self, tokens: u64) -> Self {
        self.tokens_used = tokens;
        self
    }

    /// Set generation time.
    pub fn with_time_ms(mut self, time_ms: u64) -> Self {
        self.generation_time_ms = time_ms;
        self
    }
}

/// Category of validation error.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ErrorType {
    /// Syntax error in CLIPS code.
    Syntax,
    /// Semantic error (valid syntax, invalid meaning).
    Semantic,
    /// Safety error (potentially dangerous constructs).
    Safety,
}

impl std::fmt::Display for ErrorType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Syntax => write!(f, "syntax"),
            Self::Semantic => write!(f, "semantic"),
            Self::Safety => write!(f, "safety"),
        }
    }
}

/// Details of a validation failure.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationError {
    /// Category of error.
    pub error_type: ErrorType,
    /// Human-readable description.
    pub message: String,
    /// Line where error occurred (if known).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub line_number: Option<u32>,
    /// Suggested fix (if available).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub suggestion: Option<String>,
}

impl ValidationError {
    /// Create a new validation error.
    pub fn new(error_type: ErrorType, message: impl Into<String>) -> Self {
        Self {
            error_type,
            message: message.into(),
            line_number: None,
            suggestion: None,
        }
    }

    /// Set the line number.
    pub fn with_line(mut self, line: u32) -> Self {
        self.line_number = Some(line);
        self
    }

    /// Set a suggestion.
    pub fn with_suggestion(mut self, suggestion: impl Into<String>) -> Self {
        self.suggestion = Some(suggestion.into());
        self
    }

    /// Create a syntax error.
    pub fn syntax(message: impl Into<String>) -> Self {
        Self::new(ErrorType::Syntax, message)
    }

    /// Create a semantic error.
    pub fn semantic(message: impl Into<String>) -> Self {
        Self::new(ErrorType::Semantic, message)
    }

    /// Create a safety error.
    pub fn safety(message: impl Into<String>) -> Self {
        Self::new(ErrorType::Safety, message)
    }
}

impl std::fmt::Display for ValidationError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        if let Some(line) = self.line_number {
            write!(f, "[{}] Line {}: {}", self.error_type, line, self.message)
        } else {
            write!(f, "[{}] {}", self.error_type, self.message)
        }
    }
}

/// Validation status outcome.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum ValidationStatus {
    /// CLIPS code is valid.
    Valid,
    /// CLIPS code has errors.
    Invalid,
    /// CLIPS code was rejected (safety concerns).
    Rejected,
}

/// Outcome of syntax and semantic checks on generated rules.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ValidationResult {
    /// Unique identifier.
    pub id: String,
    /// Reference to GeneratedRules.
    pub generated_rules_id: String,
    /// Validation outcome.
    pub status: ValidationStatus,
    /// List of errors found.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub errors: Vec<ValidationError>,
    /// Non-fatal warnings.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub warnings: Vec<String>,
    /// When validation ran.
    pub validated_at: chrono::DateTime<chrono::Utc>,
}

impl ValidationResult {
    /// Create a valid result.
    pub fn valid(generated_rules_id: impl Into<String>) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            generated_rules_id: generated_rules_id.into(),
            status: ValidationStatus::Valid,
            errors: Vec::new(),
            warnings: Vec::new(),
            validated_at: chrono::Utc::now(),
        }
    }

    /// Create an invalid result with errors.
    pub fn invalid(generated_rules_id: impl Into<String>, errors: Vec<ValidationError>) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            generated_rules_id: generated_rules_id.into(),
            status: ValidationStatus::Invalid,
            errors,
            warnings: Vec::new(),
            validated_at: chrono::Utc::now(),
        }
    }

    /// Create a rejected result.
    pub fn rejected(generated_rules_id: impl Into<String>, reason: impl Into<String>) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            generated_rules_id: generated_rules_id.into(),
            status: ValidationStatus::Rejected,
            errors: vec![ValidationError::safety(reason)],
            warnings: Vec::new(),
            validated_at: chrono::Utc::now(),
        }
    }

    /// Add warnings to the result.
    pub fn with_warnings(mut self, warnings: Vec<String>) -> Self {
        self.warnings = warnings;
        self
    }

    /// Check if the result is valid.
    pub fn is_valid(&self) -> bool {
        self.status == ValidationStatus::Valid
    }
}

/// Storage format for saved rules.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum SaveFormat {
    /// Text (.clp) format.
    Text,
    /// Binary (bsave) format.
    Binary,
}

impl std::fmt::Display for SaveFormat {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Text => write!(f, "text"),
            Self::Binary => write!(f, "binary"),
        }
    }
}

impl std::str::FromStr for SaveFormat {
    type Err = String;

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "text" | "clp" => Ok(Self::Text),
            "binary" | "bin" => Ok(Self::Binary),
            _ => Err(format!(
                "Invalid format: '{}'. Valid values: text, binary",
                s
            )),
        }
    }
}

/// Persisted rules in text or binary format.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SavedRules {
    /// Unique identifier.
    pub id: String,
    /// Reference to source GeneratedRules.
    pub source_rules_id: String,
    /// Storage format.
    pub format: SaveFormat,
    /// Path to saved file.
    pub file_path: String,
    /// When saved.
    pub saved_at: chrono::DateTime<chrono::Utc>,
    /// Size of saved file in bytes.
    pub file_size_bytes: u64,
}

impl SavedRules {
    /// Create a new saved rules record.
    pub fn new(
        source_rules_id: impl Into<String>,
        format: SaveFormat,
        file_path: impl Into<String>,
        file_size_bytes: u64,
    ) -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            source_rules_id: source_rules_id.into(),
            format,
            file_path: file_path.into(),
            saved_at: chrono::Utc::now(),
            file_size_bytes,
        }
    }
}

/// A progressive example for the Ruler.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProgressiveExample {
    /// Example identifier.
    pub id: String,
    /// Complexity level.
    pub complexity: Complexity,
    /// Natural language description.
    pub description: String,
    /// Optional domain hints.
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub domain_hints: Vec<String>,
    /// Expected CLIPS constructs to be generated.
    pub expected_constructs: Vec<String>,
}

/// Collection of progressive examples.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProgressiveExamples {
    /// List of examples.
    pub examples: Vec<ProgressiveExample>,
}

impl ProgressiveExamples {
    /// Filter examples by complexity level.
    pub fn by_complexity(&self, complexity: Complexity) -> Vec<&ProgressiveExample> {
        self.examples
            .iter()
            .filter(|e| e.complexity == complexity)
            .collect()
    }

    /// Get an example by ID.
    pub fn get(&self, id: &str) -> Option<&ProgressiveExample> {
        self.examples.iter().find(|e| e.id == id)
    }
}

// ============================================================================
// Prompts
// ============================================================================

/// System prompt for CLIPS code generation.
pub const SYSTEM_PROMPT: &str = r#"You are an expert CLIPS (C Language Integrated Production System) programmer.
Your task is to generate valid CLIPS code from natural language descriptions.

GUIDELINES:
1. Always include necessary deftemplates before defrules
2. Use clear, descriptive names for templates, rules, and slots
3. Include comments explaining the logic
4. Follow CLIPS best practices for performance
5. Ensure parentheses are balanced
6. Use appropriate slot types and constraints

COMPLEXITY LEVELS:
- basic: Use only deftemplate and defrule
- intermediate: Include salience, test patterns, and constraints
- advanced: Include deffunction, defmodule, and complex patterns

OUTPUT FORMAT:
Return only valid CLIPS code. No explanations outside of CLIPS comments.
The code must be immediately loadable into a CLIPS environment."#;

/// Generate prompt template.
pub const GENERATE_PROMPT_TEMPLATE: &str = r#"Generate CLIPS rules for the following requirement:

Description: {description}
Complexity Level: {complexity}
Domain Hints: {domain_hints}

Requirements:
1. Create appropriate deftemplates for the domain
2. Create defrules that implement the described behavior
3. Include comments explaining the logic
4. Ensure the code is valid and can be loaded into CLIPS

Generate the CLIPS code now:"#;

/// Format a generate prompt from a rule description.
pub fn format_generate_prompt(desc: &RuleDescription) -> String {
    let domain_hints = if desc.domain_hints.is_empty() {
        "(none)".to_string()
    } else {
        desc.domain_hints.join(", ")
    };

    GENERATE_PROMPT_TEMPLATE
        .replace("{description}", &desc.description)
        .replace("{complexity}", &desc.complexity.to_string())
        .replace("{domain_hints}", &domain_hints)
}

// ============================================================================
// Validator
// ============================================================================

/// Dangerous patterns that should not appear in generated CLIPS code.
const DANGEROUS_PATTERNS: &[&str] = &["(system ", "(shell ", "(open ", "(close ", "(remove "];

/// Validator for CLIPS code.
#[derive(Debug, Clone)]
pub struct Validator {
    /// Patterns considered dangerous.
    pub dangerous_patterns: Vec<String>,
    /// Required constructs for valid code.
    pub required_constructs: Vec<String>,
}

impl Default for Validator {
    fn default() -> Self {
        Self::new()
    }
}

impl Validator {
    /// Create a new validator with default settings.
    pub fn new() -> Self {
        Self {
            dangerous_patterns: DANGEROUS_PATTERNS.iter().map(|s| s.to_string()).collect(),
            required_constructs: vec!["deftemplate".to_string(), "defrule".to_string()],
        }
    }

    /// Validate CLIPS code.
    pub fn validate(&self, code: &str) -> ValidationResult {
        let mut errors = Vec::new();
        let mut warnings = Vec::new();

        // Check for empty code
        if code.trim().is_empty() {
            errors.push(ValidationError::syntax("Empty CLIPS code"));
            return ValidationResult::invalid("validation", errors);
        }

        // Check balanced parentheses
        if let Some(err) = self.check_balanced_parens(code) {
            errors.push(err);
        }

        // Check for dangerous patterns
        errors.extend(self.check_dangerous_patterns(code));

        // Check for warnings
        warnings.extend(self.check_warnings(code));

        if errors.is_empty() {
            ValidationResult::valid("validation").with_warnings(warnings)
        } else {
            // Check if any error is a safety error
            let has_safety = errors.iter().any(|e| e.error_type == ErrorType::Safety);
            if has_safety {
                ValidationResult::rejected("validation", "Dangerous constructs detected")
                    .with_warnings(warnings)
            } else {
                ValidationResult::invalid("validation", errors).with_warnings(warnings)
            }
        }
    }

    /// Check if parentheses are balanced.
    fn check_balanced_parens(&self, code: &str) -> Option<ValidationError> {
        let open = code.matches('(').count();
        let close = code.matches(')').count();

        if open != close {
            return Some(
                ValidationError::syntax(format!(
                    "Unbalanced parentheses: {} open, {} close",
                    open, close
                ))
                .with_suggestion("Check for missing opening or closing parentheses"),
            );
        }

        // Check proper nesting
        let mut depth = 0i32;
        let mut byte_pos = 0usize;
        for ch in code.chars() {
            match ch {
                '(' => depth += 1,
                ')' => {
                    depth -= 1;
                    if depth < 0 {
                        let line = code[..byte_pos].matches('\n').count() + 1;
                        return Some(
                            ValidationError::syntax("Unexpected closing parenthesis")
                                .with_line(line as u32),
                        );
                    }
                }
                _ => {}
            }
            byte_pos += ch.len_utf8();
        }

        None
    }

    /// Check for dangerous patterns.
    fn check_dangerous_patterns(&self, code: &str) -> Vec<ValidationError> {
        let mut errors = Vec::new();
        let code_lower = code.to_lowercase();

        for pattern in &self.dangerous_patterns {
            if let Some(pos) = code_lower.find(&pattern.to_lowercase()) {
                let line = code[..pos].matches('\n').count() + 1;
                errors.push(
                    ValidationError::safety("Potentially dangerous construct detected")
                        .with_line(line as u32)
                        .with_suggestion("Remove system calls and file operations"),
                );
            }
        }

        errors
    }

    /// Check for non-fatal warnings.
    fn check_warnings(&self, code: &str) -> Vec<String> {
        let mut warnings = Vec::new();

        let has_template = code.contains("deftemplate");
        let has_rule = code.contains("defrule");

        if !has_template && !has_rule {
            warnings.push("No deftemplate or defrule found".to_string());
        } else if !has_template {
            warnings.push("No deftemplate found - rules may use ordered facts only".to_string());
        } else if !has_rule {
            warnings.push("No defrule found - code defines templates but no rules".to_string());
        }

        if !code.contains(";;") && !code.contains(';') {
            warnings.push("No comments found - consider adding documentation".to_string());
        }

        if code.contains("(declare") && !code.contains("salience") {
            warnings.push("declare block without salience - using default priority".to_string());
        }

        warnings
    }
}

/// Validate CLIPS code using the default validator.
pub fn validate_code(code: &str) -> ValidationResult {
    Validator::new().validate(code)
}

/// Extract construct names from CLIPS code.
pub fn extract_constructs(code: &str) -> Vec<String> {
    let mut constructs = Vec::new();

    let patterns = [
        (
            "deftemplate",
            r"\(\s*deftemplate\s+([a-zA-Z_][a-zA-Z0-9_-]*)",
        ),
        ("defrule", r"\(\s*defrule\s+([a-zA-Z_][a-zA-Z0-9_-]*)"),
        (
            "deffunction",
            r"\(\s*deffunction\s+([a-zA-Z_][a-zA-Z0-9_-]*)",
        ),
        ("defmodule", r"\(\s*defmodule\s+([a-zA-Z_][a-zA-Z0-9_-]*)"),
        ("deffacts", r"\(\s*deffacts\s+([a-zA-Z_][a-zA-Z0-9_-]*)"),
    ];

    for (construct_type, pattern) in patterns {
        if let Ok(re) = regex::Regex::new(pattern) {
            for cap in re.captures_iter(code) {
                if let Some(name) = cap.get(1) {
                    constructs.push(format!("{}:{}", construct_type, name.as_str()));
                }
            }
        }
    }

    constructs
}

/// Check if a specific construct type exists in the code.
pub fn has_construct(code: &str, construct_type: &str) -> bool {
    let pattern = format!(r"\(\s*{}\s+", regex::escape(construct_type));
    regex::Regex::new(&pattern)
        .map(|re| re.is_match(code))
        .unwrap_or(false)
}

// ============================================================================
// Generator
// ============================================================================

/// Configuration for rule generation.
#[derive(Debug, Clone)]
pub struct GeneratorConfig {
    /// LLM model to use.
    pub model: String,
    /// Maximum number of generation attempts.
    pub max_retries: u8,
    /// Timeout per attempt in milliseconds.
    pub timeout_ms: u64,
}

impl Default for GeneratorConfig {
    fn default() -> Self {
        Self {
            model: "claude-haiku-4-5-20251001".to_string(),
            max_retries: 5,
            timeout_ms: 30_000,
        }
    }
}

/// Result of rule generation.
#[derive(Debug, Clone)]
pub struct GenerateResult {
    /// Whether generation succeeded.
    pub success: bool,
    /// Generated rules if successful.
    pub rules: Option<GeneratedRules>,
    /// Validation result.
    pub validation: Option<ValidationResult>,
    /// Number of attempts made.
    pub attempts: u8,
    /// Errors from failed attempts.
    pub errors: Vec<String>,
}

/// Generator for CLIPS rules from natural language.
#[derive(Debug, Clone)]
pub struct Generator {
    /// Configuration.
    pub config: GeneratorConfig,
    /// Validator for generated code.
    pub validator: Validator,
}

impl Default for Generator {
    fn default() -> Self {
        Self::new()
    }
}

impl Generator {
    /// Create a new generator with default settings.
    pub fn new() -> Self {
        Self {
            config: GeneratorConfig::default(),
            validator: Validator::new(),
        }
    }

    /// Set the model.
    pub fn with_model(mut self, model: impl Into<String>) -> Self {
        self.config.model = model.into();
        self
    }

    /// Set maximum retries.
    pub fn with_max_retries(mut self, retries: u8) -> Self {
        self.config.max_retries = retries;
        self
    }

    /// Set timeout per attempt.
    pub fn with_timeout_ms(mut self, timeout_ms: u64) -> Self {
        self.config.timeout_ms = timeout_ms;
        self
    }

    /// Generate CLIPS rules from a description.
    ///
    /// This is a simulation - in production, it would call the LLM API.
    pub fn generate(&self, desc: &RuleDescription) -> GenerateResult {
        let start_time = std::time::Instant::now();
        let mut result = GenerateResult {
            success: false,
            rules: None,
            validation: None,
            attempts: 0,
            errors: Vec::new(),
        };

        for attempt in 1..=self.config.max_retries {
            result.attempts = attempt;

            // Generate code (simulated)
            let (code, tokens) = self.generate_code(desc, attempt);

            // Validate
            let validation = self.validator.validate(&code);
            result.validation = Some(validation.clone());

            if validation.is_valid() {
                let elapsed = start_time.elapsed().as_millis() as u64;
                result.success = true;
                result.rules = Some(
                    GeneratedRules::new(&desc.id, code, &self.config.model)
                        .with_attempt(attempt)
                        .with_tokens(tokens)
                        .with_time_ms(elapsed),
                );
                return result;
            }

            // Record error
            let err_msg = if validation.errors.is_empty() {
                "validation failed".to_string()
            } else {
                validation.errors[0].message.clone()
            };
            result
                .errors
                .push(format!("attempt {}: {}", attempt, err_msg));
        }

        result
    }

    /// Generate CLIPS code (simulated).
    fn generate_code(&self, desc: &RuleDescription, _attempt: u8) -> (String, u64) {
        let code = match desc.complexity {
            Complexity::Basic => self.generate_basic_code(desc),
            Complexity::Intermediate => self.generate_intermediate_code(desc),
            Complexity::Advanced => self.generate_advanced_code(desc),
        };

        let tokens = match desc.complexity {
            Complexity::Basic => 150,
            Complexity::Intermediate => 300,
            Complexity::Advanced => 500,
        };

        (code, tokens)
    }

    fn generate_basic_code(&self, desc: &RuleDescription) -> String {
        format!(
            r#";;; Auto-generated CLIPS rules
;;; Description: {}
;;; Complexity: basic

(deftemplate entity
  "A generic entity for demonstration"
  (slot id (type STRING))
  (slot name (type STRING))
  (slot value (type INTEGER) (default 0)))

(deftemplate result
  "Result of rule processing"
  (slot entity-id (type STRING))
  (slot status (type SYMBOL) (allowed-symbols pending processed)))

(defrule process-entity
  "Process entities with positive values"
  (entity (id ?id) (name ?name) (value ?v&:(> ?v 0)))
  (not (result (entity-id ?id)))
  =>
  (assert (result (entity-id ?id) (status processed)))
  (printout t "Processed entity: " ?name " with value: " ?v crlf))
"#,
            desc.description
        )
    }

    fn generate_intermediate_code(&self, desc: &RuleDescription) -> String {
        format!(
            r#";;; Auto-generated CLIPS rules
;;; Description: {}
;;; Complexity: intermediate

(deftemplate task
  "A task with priority"
  (slot id (type STRING))
  (slot name (type STRING))
  (slot priority (type INTEGER) (range 1 10) (default 5))
  (slot status (type SYMBOL) (allowed-symbols pending running completed) (default pending)))

(deftemplate task-result
  "Result of task processing"
  (slot task-id (type STRING))
  (slot completed-at (type STRING)))

;;; High priority tasks first
(defrule process-high-priority
  "Process high priority tasks first"
  (declare (salience 100))
  ?task <- (task (id ?id) (name ?name) (priority ?p&:(>= ?p 8)) (status pending))
  =>
  (modify ?task (status running))
  (printout t "Starting high-priority task: " ?name " (priority " ?p ")" crlf))

;;; Normal priority tasks
(defrule process-normal-priority
  "Process normal priority tasks"
  (declare (salience 50))
  ?task <- (task (id ?id) (name ?name) (priority ?p&:(< ?p 8)) (status pending))
  (not (task (status running)))
  =>
  (modify ?task (status running))
  (printout t "Starting task: " ?name " (priority " ?p ")" crlf))

;;; Complete running tasks
(defrule complete-task
  "Complete running tasks"
  (declare (salience 10))
  ?task <- (task (id ?id) (status running))
  =>
  (modify ?task (status completed))
  (assert (task-result (task-id ?id) (completed-at "now")))
  (printout t "Completed task: " ?id crlf))
"#,
            desc.description
        )
    }

    fn generate_advanced_code(&self, desc: &RuleDescription) -> String {
        format!(
            r#";;; Auto-generated CLIPS rules
;;; Description: {}
;;; Complexity: advanced

;;; =============================================
;;; Module: MAIN
;;; =============================================
(defmodule MAIN (export ?ALL))

(deftemplate MAIN::entity
  "Base entity template"
  (slot id (type STRING))
  (slot type (type SYMBOL))
  (slot score (type FLOAT) (default 0.0)))

(deftemplate MAIN::processing-state
  "Current processing state"
  (slot phase (type SYMBOL) (allowed-symbols init analyze decide complete))
  (slot entity-count (type INTEGER) (default 0)))

;;; =============================================
;;; Module: ANALYSIS
;;; =============================================
(defmodule ANALYSIS (import MAIN ?ALL))

(deftemplate ANALYSIS::analysis-result
  "Result of entity analysis"
  (slot entity-id (type STRING))
  (slot category (type SYMBOL))
  (slot confidence (type FLOAT)))

;;; =============================================
;;; Helper Functions
;;; =============================================
(deffunction MAIN::calculate-score (?base ?multiplier)
  "Calculate weighted score"
  (* ?base ?multiplier))

(deffunction MAIN::categorize (?score)
  "Categorize based on score"
  (if (>= ?score 0.8) then high
   else (if (>= ?score 0.5) then medium
         else low)))

;;; =============================================
;;; Rules
;;; =============================================
(defrule MAIN::start-processing
  "Initialize processing"
  (declare (salience 1000))
  (not (processing-state))
  =>
  (assert (processing-state (phase init) (entity-count 0)))
  (printout t "Starting processing..." crlf))

(defrule MAIN::transition-to-analysis
  "Move to analysis phase"
  (declare (salience 500))
  ?state <- (processing-state (phase init))
  =>
  (modify ?state (phase analyze))
  (focus ANALYSIS))

(defrule ANALYSIS::analyze-entity
  "Analyze each entity"
  ?entity <- (entity (id ?id) (score ?s))
  (not (analysis-result (entity-id ?id)))
  =>
  (bind ?category (categorize ?s))
  (assert (analysis-result
    (entity-id ?id)
    (category ?category)
    (confidence (calculate-score ?s 1.2))))
  (printout t "Analyzed entity " ?id ": " ?category crlf))

(defrule MAIN::complete-processing
  "Complete processing when done"
  (declare (salience -1000))
  ?state <- (processing-state (phase analyze))
  =>
  (modify ?state (phase complete))
  (printout t "Processing complete." crlf))
"#,
            desc.description
        )
    }
}

/// Convenience function for simple generation.
pub fn generate_rules(description: &str, complexity: Complexity) -> GenerateResult {
    let desc = RuleDescription::new(description).with_complexity(complexity);
    Generator::new().generate(&desc)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_complexity_parsing() {
        assert_eq!("basic".parse::<Complexity>().unwrap(), Complexity::Basic);
        assert_eq!(
            "intermediate".parse::<Complexity>().unwrap(),
            Complexity::Intermediate
        );
        assert_eq!(
            "advanced".parse::<Complexity>().unwrap(),
            Complexity::Advanced
        );
        assert!("invalid".parse::<Complexity>().is_err());
    }

    #[test]
    fn test_rule_description_builder() {
        let desc = RuleDescription::new("Classify adults")
            .with_complexity(Complexity::Basic)
            .with_domain_hints(vec!["age".to_string(), "classification".to_string()]);

        assert_eq!(desc.description, "Classify adults");
        assert_eq!(desc.complexity, Complexity::Basic);
        assert_eq!(desc.domain_hints.len(), 2);
    }

    #[test]
    fn test_validation_error_display() {
        let err = ValidationError::syntax("Unexpected token")
            .with_line(5)
            .with_suggestion("Add closing parenthesis");

        let display = format!("{}", err);
        assert!(display.contains("syntax"));
        assert!(display.contains("Line 5"));
        assert!(display.contains("Unexpected token"));
    }

    #[test]
    fn test_validation_result_states() {
        let valid = ValidationResult::valid("test-id");
        assert!(valid.is_valid());
        assert_eq!(valid.status, ValidationStatus::Valid);

        let invalid = ValidationResult::invalid("test-id", vec![ValidationError::syntax("Error")]);
        assert!(!invalid.is_valid());
        assert_eq!(invalid.errors.len(), 1);

        let rejected = ValidationResult::rejected("test-id", "Dangerous construct");
        assert!(!rejected.is_valid());
        assert_eq!(rejected.status, ValidationStatus::Rejected);
    }

    #[test]
    fn test_validator_balanced_parens() {
        let validator = Validator::new();

        // Valid balanced code
        let valid = "(deftemplate test (slot name))";
        let result = validator.validate(valid);
        assert!(result.is_valid());

        // Unbalanced - missing close
        let unbalanced = "(deftemplate test (slot name)";
        let result = validator.validate(unbalanced);
        assert!(!result.is_valid());
        assert!(
            result
                .errors
                .iter()
                .any(|e| e.message.contains("Unbalanced"))
        );
    }

    #[test]
    fn test_validator_dangerous_patterns() {
        let validator = Validator::new();

        // Safe code
        let safe = "(deftemplate test (slot name))\n(defrule r => (printout t \"hello\"))";
        let result = validator.validate(safe);
        assert!(result.is_valid());

        // Dangerous - system call
        let dangerous = "(defrule r => (system \"rm -rf /\"))";
        let result = validator.validate(dangerous);
        assert!(!result.is_valid());
        assert!(
            result
                .errors
                .iter()
                .any(|e| e.error_type == ErrorType::Safety)
        );
    }

    #[test]
    fn test_validator_warnings() {
        let validator = Validator::new();

        // Code without comments
        let no_comments = "(deftemplate test (slot name))\n(defrule r (test) => (assert (done)))";
        let result = validator.validate(no_comments);
        assert!(result.is_valid());
        assert!(result.warnings.iter().any(|w| w.contains("No comments")));

        // Code with comments
        let with_comments =
            ";;; Test\n(deftemplate test (slot name))\n(defrule r (test) => (assert (done)))";
        let result = validator.validate(with_comments);
        assert!(result.is_valid());
        assert!(!result.warnings.iter().any(|w| w.contains("No comments")));
    }

    #[test]
    fn test_extract_constructs() {
        let code = r#"
(deftemplate entity (slot id))
(deftemplate result (slot status))
(defrule process-entity (entity (id ?id)) => (assert (result)))
(deffunction helper () (return 1))
"#;
        let constructs = extract_constructs(code);

        assert!(constructs.contains(&"deftemplate:entity".to_string()));
        assert!(constructs.contains(&"deftemplate:result".to_string()));
        assert!(constructs.contains(&"defrule:process-entity".to_string()));
        assert!(constructs.contains(&"deffunction:helper".to_string()));
    }

    #[test]
    fn test_has_construct() {
        let code = "(deftemplate test (slot name))\n(defrule r => (assert (done)))";

        assert!(has_construct(code, "deftemplate"));
        assert!(has_construct(code, "defrule"));
        assert!(!has_construct(code, "deffunction"));
    }

    #[test]
    fn test_generator_config() {
        let config = GeneratorConfig::default();
        assert_eq!(config.model, "claude-haiku-4-5-20251001");
        assert_eq!(config.max_retries, 5);
    }

    #[test]
    fn test_generator_generate() {
        let generator = Generator::new();
        let desc = RuleDescription::new("Create a simple rule").with_complexity(Complexity::Basic);

        let result = generator.generate(&desc);
        assert!(result.success);
        assert!(result.rules.is_some());
        assert!(result.attempts >= 1);

        let rules = result.rules.unwrap();
        assert!(!rules.clips_code.is_empty());
    }

    #[test]
    fn test_generate_rules_convenience() {
        let result = generate_rules("Test rule", Complexity::Basic);
        assert!(result.success);
        assert!(result.rules.is_some());
    }

    #[test]
    fn test_format_generate_prompt() {
        let desc = RuleDescription::new("Classify adults")
            .with_complexity(Complexity::Basic)
            .with_domain_hints(vec!["age".to_string()]);

        let prompt = format_generate_prompt(&desc);
        assert!(prompt.contains("Classify adults"));
        assert!(prompt.contains("basic"));
        assert!(prompt.contains("age"));
    }
}
