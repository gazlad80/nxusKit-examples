// Package ruler provides natural language to CLIPS rule generation.
//
// Ruler accepts natural language descriptions, uses LLM to generate valid
// CLIPS code, validates it, and loads into CLIPS environment.
package ruler

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Complexity represents the target complexity level for rule generation.
type Complexity string

const (
	// Basic: deftemplate and simple defrule constructs.
	ComplexityBasic Complexity = "basic"
	// Intermediate: salience, test patterns, constraints.
	ComplexityIntermediate Complexity = "intermediate"
	// Advanced: deffunction, defmodule, complex patterns.
	ComplexityAdvanced Complexity = "advanced"
)

// String returns the string representation of the complexity.
func (c Complexity) String() string {
	return string(c)
}

// ParseComplexity parses a string into a Complexity value.
func ParseComplexity(s string) (Complexity, error) {
	switch s {
	case "basic":
		return ComplexityBasic, nil
	case "intermediate":
		return ComplexityIntermediate, nil
	case "advanced":
		return ComplexityAdvanced, nil
	default:
		return "", fmt.Errorf("invalid complexity: '%s'. Valid values: basic, intermediate, advanced", s)
	}
}

// RuleDescription is a natural language description of desired CLIPS behavior.
type RuleDescription struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// Description is the natural language rule description.
	Description string `json:"description"`
	// Complexity is the target complexity level.
	Complexity Complexity `json:"complexity"`
	// DomainHints are optional domain keywords.
	DomainHints []string `json:"domain_hints,omitempty"`
}

// NewRuleDescription creates a new rule description with basic complexity.
func NewRuleDescription(description string) *RuleDescription {
	return &RuleDescription{
		ID:          uuid.New().String(),
		Description: description,
		Complexity:  ComplexityBasic,
		DomainHints: nil,
	}
}

// WithComplexity sets the complexity level.
func (r *RuleDescription) WithComplexity(c Complexity) *RuleDescription {
	r.Complexity = c
	return r
}

// WithDomainHints sets the domain hints.
func (r *RuleDescription) WithDomainHints(hints []string) *RuleDescription {
	r.DomainHints = hints
	return r
}

// GeneratedRules represents CLIPS source code produced by LLM.
type GeneratedRules struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// SourceDescriptionID references the source RuleDescription.
	SourceDescriptionID string `json:"source_description_id"`
	// ClipsCode is the generated CLIPS source code.
	ClipsCode string `json:"clips_code"`
	// GenerationAttempt is the attempt number (1-5).
	GenerationAttempt int `json:"generation_attempt"`
	// ModelUsed is the LLM model identifier.
	ModelUsed string `json:"model_used"`
	// TokensUsed is the total tokens consumed.
	TokensUsed int64 `json:"tokens_used"`
	// GenerationTimeMs is the time to generate in milliseconds.
	GenerationTimeMs int64 `json:"generation_time_ms"`
}

// NewGeneratedRules creates a new generated rules result.
func NewGeneratedRules(sourceID, clipsCode, model string) *GeneratedRules {
	return &GeneratedRules{
		ID:                  uuid.New().String(),
		SourceDescriptionID: sourceID,
		ClipsCode:           clipsCode,
		GenerationAttempt:   1,
		ModelUsed:           model,
		TokensUsed:          0,
		GenerationTimeMs:    0,
	}
}

// WithAttempt sets the attempt number.
func (g *GeneratedRules) WithAttempt(attempt int) *GeneratedRules {
	g.GenerationAttempt = attempt
	return g
}

// WithTokens sets the token usage.
func (g *GeneratedRules) WithTokens(tokens int64) *GeneratedRules {
	g.TokensUsed = tokens
	return g
}

// WithTimeMs sets the generation time.
func (g *GeneratedRules) WithTimeMs(timeMs int64) *GeneratedRules {
	g.GenerationTimeMs = timeMs
	return g
}

// ErrorType categorizes validation errors.
type ErrorType string

const (
	// ErrorTypeSyntax indicates a syntax error in CLIPS code.
	ErrorTypeSyntax ErrorType = "syntax"
	// ErrorTypeSemantic indicates a semantic error (valid syntax, invalid meaning).
	ErrorTypeSemantic ErrorType = "semantic"
	// ErrorTypeSafety indicates a safety error (potentially dangerous constructs).
	ErrorTypeSafety ErrorType = "safety"
)

// String returns the string representation of the error type.
func (e ErrorType) String() string {
	return string(e)
}

// ValidationError represents details of a validation failure.
type ValidationError struct {
	// ErrorType is the category of error.
	ErrorType ErrorType `json:"error_type"`
	// Message is the human-readable description.
	Message string `json:"message"`
	// LineNumber is the line where error occurred (0 if unknown).
	LineNumber int `json:"line_number,omitempty"`
	// Suggestion is the suggested fix (empty if none).
	Suggestion string `json:"suggestion,omitempty"`
}

// NewValidationError creates a new validation error.
func NewValidationError(errorType ErrorType, message string) *ValidationError {
	return &ValidationError{
		ErrorType: errorType,
		Message:   message,
	}
}

// WithLine sets the line number.
func (v *ValidationError) WithLine(line int) *ValidationError {
	v.LineNumber = line
	return v
}

// WithSuggestion sets a suggestion.
func (v *ValidationError) WithSuggestion(suggestion string) *ValidationError {
	v.Suggestion = suggestion
	return v
}

// SyntaxError creates a syntax error.
func SyntaxError(message string) *ValidationError {
	return NewValidationError(ErrorTypeSyntax, message)
}

// SemanticError creates a semantic error.
func SemanticError(message string) *ValidationError {
	return NewValidationError(ErrorTypeSemantic, message)
}

// SafetyError creates a safety error.
func SafetyError(message string) *ValidationError {
	return NewValidationError(ErrorTypeSafety, message)
}

// Error implements the error interface.
func (v *ValidationError) Error() string {
	if v.LineNumber > 0 {
		return fmt.Sprintf("[%s] Line %d: %s", v.ErrorType, v.LineNumber, v.Message)
	}
	return fmt.Sprintf("[%s] %s", v.ErrorType, v.Message)
}

// ValidationStatus represents the validation outcome.
type ValidationStatus string

const (
	// ValidationStatusValid indicates CLIPS code is valid.
	ValidationStatusValid ValidationStatus = "valid"
	// ValidationStatusInvalid indicates CLIPS code has errors.
	ValidationStatusInvalid ValidationStatus = "invalid"
	// ValidationStatusRejected indicates CLIPS code was rejected for safety.
	ValidationStatusRejected ValidationStatus = "rejected"
)

// ValidationResult is the outcome of validation checks on generated rules.
type ValidationResult struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// GeneratedRulesID references the GeneratedRules.
	GeneratedRulesID string `json:"generated_rules_id"`
	// Status is the validation outcome.
	Status ValidationStatus `json:"status"`
	// Errors is the list of errors found.
	Errors []*ValidationError `json:"errors,omitempty"`
	// Warnings is the list of non-fatal warnings.
	Warnings []string `json:"warnings,omitempty"`
	// ValidatedAt is when validation ran.
	ValidatedAt time.Time `json:"validated_at"`
}

// NewValidResult creates a valid result.
func NewValidResult(generatedRulesID string) *ValidationResult {
	return &ValidationResult{
		ID:               uuid.New().String(),
		GeneratedRulesID: generatedRulesID,
		Status:           ValidationStatusValid,
		Errors:           nil,
		Warnings:         nil,
		ValidatedAt:      time.Now().UTC(),
	}
}

// NewInvalidResult creates an invalid result with errors.
func NewInvalidResult(generatedRulesID string, errors []*ValidationError) *ValidationResult {
	return &ValidationResult{
		ID:               uuid.New().String(),
		GeneratedRulesID: generatedRulesID,
		Status:           ValidationStatusInvalid,
		Errors:           errors,
		Warnings:         nil,
		ValidatedAt:      time.Now().UTC(),
	}
}

// NewRejectedResult creates a rejected result.
func NewRejectedResult(generatedRulesID, reason string) *ValidationResult {
	return &ValidationResult{
		ID:               uuid.New().String(),
		GeneratedRulesID: generatedRulesID,
		Status:           ValidationStatusRejected,
		Errors:           []*ValidationError{SafetyError(reason)},
		Warnings:         nil,
		ValidatedAt:      time.Now().UTC(),
	}
}

// WithWarnings adds warnings to the result.
func (v *ValidationResult) WithWarnings(warnings []string) *ValidationResult {
	v.Warnings = warnings
	return v
}

// IsValid checks if the result is valid.
func (v *ValidationResult) IsValid() bool {
	return v.Status == ValidationStatusValid
}

// SaveFormat represents the storage format for saved rules.
type SaveFormat string

const (
	// SaveFormatText is text (.clp) format.
	SaveFormatText SaveFormat = "text"
	// SaveFormatBinary is binary (bsave) format.
	SaveFormatBinary SaveFormat = "binary"
)

// String returns the string representation of the format.
func (s SaveFormat) String() string {
	return string(s)
}

// ParseSaveFormat parses a string into a SaveFormat value.
func ParseSaveFormat(str string) (SaveFormat, error) {
	switch str {
	case "text", "clp":
		return SaveFormatText, nil
	case "binary", "bin":
		return SaveFormatBinary, nil
	default:
		return "", fmt.Errorf("invalid format: '%s'. Valid values: text, binary", str)
	}
}

// SavedRules represents persisted rules in text or binary format.
type SavedRules struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// SourceRulesID references the source GeneratedRules.
	SourceRulesID string `json:"source_rules_id"`
	// Format is the storage format.
	Format SaveFormat `json:"format"`
	// FilePath is the path to saved file.
	FilePath string `json:"file_path"`
	// SavedAt is when the rules were saved.
	SavedAt time.Time `json:"saved_at"`
	// FileSizeBytes is the size of saved file in bytes.
	FileSizeBytes int64 `json:"file_size_bytes"`
}

// NewSavedRules creates a new saved rules record.
func NewSavedRules(sourceRulesID string, format SaveFormat, filePath string, sizeBytes int64) *SavedRules {
	return &SavedRules{
		ID:            uuid.New().String(),
		SourceRulesID: sourceRulesID,
		Format:        format,
		FilePath:      filePath,
		SavedAt:       time.Now().UTC(),
		FileSizeBytes: sizeBytes,
	}
}

// ProgressiveExample represents a progressive example for the Ruler.
type ProgressiveExample struct {
	// ID is the example identifier.
	ID string `json:"id"`
	// Complexity is the complexity level.
	Complexity Complexity `json:"complexity"`
	// Description is the natural language description.
	Description string `json:"description"`
	// DomainHints are optional domain hints.
	DomainHints []string `json:"domain_hints,omitempty"`
	// ExpectedConstructs are the expected CLIPS constructs to be generated.
	ExpectedConstructs []string `json:"expected_constructs"`
}

// ProgressiveExamples is a collection of progressive examples.
type ProgressiveExamples struct {
	// Examples is the list of examples.
	Examples []*ProgressiveExample `json:"examples"`
}

// ByComplexity filters examples by complexity level.
func (p *ProgressiveExamples) ByComplexity(c Complexity) []*ProgressiveExample {
	var result []*ProgressiveExample
	for _, ex := range p.Examples {
		if ex.Complexity == c {
			result = append(result, ex)
		}
	}
	return result
}

// Get retrieves an example by ID.
func (p *ProgressiveExamples) Get(id string) *ProgressiveExample {
	for _, ex := range p.Examples {
		if ex.ID == id {
			return ex
		}
	}
	return nil
}
