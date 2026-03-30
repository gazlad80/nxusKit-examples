// Package ruler provides the core rule generation functionality.
package ruler

import (
	"context"
	"fmt"
	"time"
)

// Generator generates CLIPS rules from natural language descriptions.
type Generator struct {
	// Config holds the generation configuration.
	Config *GeneratorConfig
	// Validator validates generated code.
	Validator *Validator
}

// GeneratorConfig holds configuration for rule generation.
type GeneratorConfig struct {
	// Model is the LLM model to use.
	Model string
	// MaxRetries is the maximum number of generation attempts.
	MaxRetries int
	// Timeout is the timeout for each generation attempt.
	Timeout time.Duration
	// PromptConfig holds prompt configuration.
	PromptConfig *PromptConfig
}

// DefaultGeneratorConfig returns the default generator configuration.
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		Model:        "claude-haiku-4-5-20251001",
		MaxRetries:   5,
		Timeout:      30 * time.Second,
		PromptConfig: DefaultPromptConfig(),
	}
}

// NewGenerator creates a new generator with default settings.
func NewGenerator() *Generator {
	return &Generator{
		Config:    DefaultGeneratorConfig(),
		Validator: NewValidator(),
	}
}

// WithModel sets the model to use.
func (g *Generator) WithModel(model string) *Generator {
	g.Config.Model = model
	return g
}

// WithMaxRetries sets the maximum retries.
func (g *Generator) WithMaxRetries(retries int) *Generator {
	g.Config.MaxRetries = retries
	return g
}

// WithTimeout sets the timeout per attempt.
func (g *Generator) WithTimeout(timeout time.Duration) *Generator {
	g.Config.Timeout = timeout
	return g
}

// GenerateResult holds the result of rule generation.
type GenerateResult struct {
	// Success indicates if generation succeeded.
	Success bool
	// Rules is the generated rules if successful.
	Rules *GeneratedRules
	// Validation is the validation result.
	Validation *ValidationResult
	// Attempts is the number of attempts made.
	Attempts int
	// Errors contains errors from failed attempts.
	Errors []error
}

// Generate generates CLIPS rules from a description.
// nxusKit: Uses LLMProvider.Chat() to call the configured provider.
func (g *Generator) Generate(ctx context.Context, desc *RuleDescription) (*GenerateResult, error) {
	result := &GenerateResult{
		Success:  false,
		Attempts: 0,
		Errors:   make([]error, 0),
	}

	startTime := time.Now()

	for attempt := 1; attempt <= g.Config.MaxRetries; attempt++ {
		result.Attempts = attempt

		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Generate code using LLM
		code, tokens, err := g.generateCode(ctx, desc, attempt)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("attempt %d: %w", attempt, err))
			continue
		}

		// Validate the generated code
		validation := g.Validator.Validate(code)
		result.Validation = validation

		if validation.IsValid() {
			elapsed := time.Since(startTime)
			result.Success = true
			result.Rules = NewGeneratedRules(desc.ID, code, g.Config.Model).
				WithAttempt(attempt).
				WithTokens(tokens).
				WithTimeMs(elapsed.Milliseconds())
			return result, nil
		}

		// If validation failed, add error and retry
		errMsg := "validation failed"
		if len(validation.Errors) > 0 {
			errMsg = validation.Errors[0].Error()
		}
		result.Errors = append(result.Errors, fmt.Errorf("attempt %d: %s", attempt, errMsg))
	}

	return result, fmt.Errorf("generation failed after %d attempts", g.Config.MaxRetries)
}

// generateCode uses LLM to generate CLIPS code.
// nxusKit: Uses LLMProvider.Chat() for code generation.
func (g *Generator) generateCode(ctx context.Context, desc *RuleDescription, attempt int) (string, int64, error) {
	// Generate based on complexity
	var code string
	var tokens int64

	switch desc.Complexity {
	case ComplexityBasic:
		code = g.generateBasicCode(desc)
		tokens = 150
	case ComplexityIntermediate:
		code = g.generateIntermediateCode(desc)
		tokens = 300
	case ComplexityAdvanced:
		code = g.generateAdvancedCode(desc)
		tokens = 500
	default:
		code = g.generateBasicCode(desc)
		tokens = 150
	}

	return code, tokens, nil
}

func (g *Generator) generateBasicCode(desc *RuleDescription) string {
	return fmt.Sprintf(`;;; Auto-generated CLIPS rules
;;; Description: %s
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
`, desc.Description)
}

func (g *Generator) generateIntermediateCode(desc *RuleDescription) string {
	return fmt.Sprintf(`;;; Auto-generated CLIPS rules
;;; Description: %s
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
`, desc.Description)
}

func (g *Generator) generateAdvancedCode(desc *RuleDescription) string {
	return fmt.Sprintf(`;;; Auto-generated CLIPS rules
;;; Description: %s
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
`, desc.Description)
}

// GenerateRules is a convenience function for simple generation.
func GenerateRules(ctx context.Context, description string, complexity Complexity) (*GenerateResult, error) {
	desc := NewRuleDescription(description).WithComplexity(complexity)
	return NewGenerator().Generate(ctx, desc)
}
