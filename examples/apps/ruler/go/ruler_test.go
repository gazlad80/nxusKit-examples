// Package ruler provides tests for CLIPS rule generation.
package ruler

import (
	"context"
	"testing"
)

func TestComplexityParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected Complexity
		wantErr  bool
	}{
		{"basic", ComplexityBasic, false},
		{"intermediate", ComplexityIntermediate, false},
		{"advanced", ComplexityAdvanced, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseComplexity(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRuleDescriptionBuilder(t *testing.T) {
	desc := NewRuleDescription("Classify adults").
		WithComplexity(ComplexityBasic).
		WithDomainHints([]string{"age", "classification"})

	if desc.Description != "Classify adults" {
		t.Errorf("got description %q, want %q", desc.Description, "Classify adults")
	}
	if desc.Complexity != ComplexityBasic {
		t.Errorf("got complexity %v, want %v", desc.Complexity, ComplexityBasic)
	}
	if len(desc.DomainHints) != 2 {
		t.Errorf("got %d domain hints, want 2", len(desc.DomainHints))
	}
}

func TestValidatorBalancedParens(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name    string
		code    string
		isValid bool
	}{
		{
			name:    "balanced simple",
			code:    "(deftemplate test (slot name))",
			isValid: true,
		},
		{
			name:    "balanced nested",
			code:    "(defrule test (entity (id ?id)) => (assert (result (entity-id ?id))))",
			isValid: true,
		},
		{
			name:    "unbalanced missing close",
			code:    "(deftemplate test (slot name)",
			isValid: false,
		},
		{
			name:    "unbalanced extra close",
			code:    "(deftemplate test))",
			isValid: false,
		},
		{
			name:    "empty",
			code:    "",
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.code)
			if result.IsValid() != tt.isValid {
				t.Errorf("Validate(%q) = %v, want %v", tt.code, result.IsValid(), tt.isValid)
				if len(result.Errors) > 0 {
					t.Logf("Errors: %v", result.Errors)
				}
			}
		})
	}
}

func TestValidatorDangerousPatterns(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name      string
		code      string
		hasSafety bool
	}{
		{
			name:      "safe code",
			code:      "(deftemplate test (slot name))\n(defrule r => (printout t \"hello\"))",
			hasSafety: false,
		},
		{
			name:      "system call",
			code:      "(defrule r => (system \"rm -rf /\"))",
			hasSafety: true,
		},
		{
			name:      "shell call",
			code:      "(defrule r => (shell \"echo hello\"))",
			hasSafety: true,
		},
		{
			name:      "open file",
			code:      "(defrule r => (open \"file.txt\" file \"w\"))",
			hasSafety: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.code)
			hasSafetyError := false
			for _, err := range result.Errors {
				if err.ErrorType == ErrorTypeSafety {
					hasSafetyError = true
					break
				}
			}
			if hasSafetyError != tt.hasSafety {
				t.Errorf("expected safety error: %v, got: %v", tt.hasSafety, hasSafetyError)
			}
		})
	}
}

func TestValidatorWarnings(t *testing.T) {
	validator := NewValidator()

	// Code with no comments
	code := "(deftemplate test (slot name))\n(defrule r (test) => (assert (done)))"
	result := validator.Validate(code)

	hasNoCommentWarning := false
	for _, w := range result.Warnings {
		if w == "No comments found - consider adding documentation" {
			hasNoCommentWarning = true
			break
		}
	}

	if !hasNoCommentWarning {
		t.Error("expected warning about no comments")
	}

	// Code with comments
	codeWithComments := ";;; Test rules\n(deftemplate test (slot name))\n(defrule r (test) => (assert (done)))"
	result2 := validator.Validate(codeWithComments)

	hasNoCommentWarning2 := false
	for _, w := range result2.Warnings {
		if w == "No comments found - consider adding documentation" {
			hasNoCommentWarning2 = true
			break
		}
	}

	if hasNoCommentWarning2 {
		t.Error("should not have warning about no comments when comments exist")
	}
}

func TestValidationResultStates(t *testing.T) {
	valid := NewValidResult("test-id")
	if !valid.IsValid() {
		t.Error("NewValidResult should be valid")
	}
	if valid.Status != ValidationStatusValid {
		t.Errorf("got status %v, want %v", valid.Status, ValidationStatusValid)
	}

	invalid := NewInvalidResult("test-id", []*ValidationError{SyntaxError("test error")})
	if invalid.IsValid() {
		t.Error("NewInvalidResult should not be valid")
	}
	if len(invalid.Errors) != 1 {
		t.Errorf("got %d errors, want 1", len(invalid.Errors))
	}
}

func TestValidationErrorTypes(t *testing.T) {
	syntax := SyntaxError("unexpected token")
	if syntax.ErrorType != ErrorTypeSyntax {
		t.Errorf("got type %v, want %v", syntax.ErrorType, ErrorTypeSyntax)
	}

	semantic := SemanticError("undefined template")
	if semantic.ErrorType != ErrorTypeSemantic {
		t.Errorf("got type %v, want %v", semantic.ErrorType, ErrorTypeSemantic)
	}

	safety := SafetyError("dangerous construct")
	if safety.ErrorType != ErrorTypeSafety {
		t.Errorf("got type %v, want %v", safety.ErrorType, ErrorTypeSafety)
	}

	// Test with line and suggestion
	errWithDetails := SyntaxError("error").WithLine(10).WithSuggestion("fix it")
	if errWithDetails.LineNumber != 10 {
		t.Errorf("got line %d, want 10", errWithDetails.LineNumber)
	}
	if errWithDetails.Suggestion != "fix it" {
		t.Errorf("got suggestion %q, want %q", errWithDetails.Suggestion, "fix it")
	}
}

func TestExtractConstructs(t *testing.T) {
	code := `
(deftemplate entity (slot id))
(deftemplate result (slot status))
(defrule process-entity (entity (id ?id)) => (assert (result)))
(deffunction helper () (return 1))
`
	constructs := ExtractConstructs(code)

	expected := map[string]bool{
		"deftemplate:entity":     true,
		"deftemplate:result":     true,
		"defrule:process-entity": true,
		"deffunction:helper":     true,
	}

	for _, c := range constructs {
		if !expected[c] {
			t.Errorf("unexpected construct: %s", c)
		}
		delete(expected, c)
	}

	if len(expected) > 0 {
		t.Errorf("missing constructs: %v", expected)
	}
}

func TestHasConstruct(t *testing.T) {
	code := "(deftemplate test (slot name))\n(defrule r => (assert (done)))"

	if !HasConstruct(code, "deftemplate") {
		t.Error("should find deftemplate")
	}
	if !HasConstruct(code, "defrule") {
		t.Error("should find defrule")
	}
	if HasConstruct(code, "deffunction") {
		t.Error("should not find deffunction")
	}
}

func TestGeneratorConfig(t *testing.T) {
	config := DefaultGeneratorConfig()

	if config.Model != "claude-haiku-4-5-20251001" {
		t.Errorf("got model %q, want claude-haiku-4-5-20251001", config.Model)
	}
	if config.MaxRetries != 5 {
		t.Errorf("got max retries %d, want 5", config.MaxRetries)
	}
}

func TestGeneratorGenerate(t *testing.T) {
	generator := NewGenerator()
	ctx := context.Background()

	desc := NewRuleDescription("Create a simple rule").
		WithComplexity(ComplexityBasic)

	result, err := generator.Generate(ctx, desc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("generation should succeed")
	}
	if result.Rules == nil {
		t.Error("rules should not be nil")
	}
	if result.Rules.ClipsCode == "" {
		t.Error("clips code should not be empty")
	}
	if result.Attempts < 1 {
		t.Error("should have at least 1 attempt")
	}
}

func TestPromptConfig(t *testing.T) {
	config := DefaultPromptConfig()

	if config.SystemPrompt == "" {
		t.Error("system prompt should not be empty")
	}

	desc := NewRuleDescription("test description").WithComplexity(ComplexityBasic)
	prompt := FormatGeneratePrompt(desc)
	if prompt == "" {
		t.Error("formatted prompt should not be empty")
	}
}
