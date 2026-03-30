// Package ruler provides CLIPS code validation functionality.
package ruler

import (
	"regexp"
	"strings"
)

// Validator validates CLIPS code for syntax, semantics, and safety.
type Validator struct {
	// DangerousPatterns are regex patterns for dangerous constructs.
	DangerousPatterns []*regexp.Regexp
	// RequiredConstructs are constructs that should be present.
	RequiredConstructs []string
}

// NewValidator creates a new validator with default settings.
func NewValidator() *Validator {
	return &Validator{
		DangerousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\(\s*system\s+`),
			regexp.MustCompile(`\(\s*shell\s+`),
			regexp.MustCompile(`\(\s*open\s+`),
			regexp.MustCompile(`\(\s*close\s+`),
			regexp.MustCompile(`\(\s*remove\s+`),
		},
		RequiredConstructs: []string{"deftemplate", "defrule"},
	}
}

// Validate validates CLIPS code and returns a validation result.
func (v *Validator) Validate(code string) *ValidationResult {
	errors := v.checkAll(code)

	if len(errors) == 0 {
		result := NewValidResult("validation")
		result.Warnings = v.checkWarnings(code)
		return result
	}

	result := NewInvalidResult("validation", errors)
	result.Warnings = v.checkWarnings(code)
	return result
}

// checkAll performs all validation checks.
func (v *Validator) checkAll(code string) []*ValidationError {
	var errors []*ValidationError

	// Check balanced parentheses
	if err := v.checkBalancedParens(code); err != nil {
		errors = append(errors, err)
	}

	// Check for dangerous patterns
	if errs := v.checkDangerousPatterns(code); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	// Check for empty code
	if strings.TrimSpace(code) == "" {
		errors = append(errors, SyntaxError("Empty CLIPS code"))
	}

	return errors
}

// checkBalancedParens checks if parentheses are balanced.
func (v *Validator) checkBalancedParens(code string) *ValidationError {
	open := strings.Count(code, "(")
	close := strings.Count(code, ")")

	if open != close {
		return SyntaxError("Unbalanced parentheses: " +
			string(rune('0'+open)) + " open, " + string(rune('0'+close)) + " close").
			WithSuggestion("Check for missing opening or closing parentheses")
	}

	// Check for proper nesting
	depth := 0
	for i, ch := range code {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth < 0 {
				return SyntaxError("Unexpected closing parenthesis").
					WithLine(countLines(code[:i]))
			}
		}
	}

	return nil
}

// checkDangerousPatterns checks for potentially dangerous constructs.
func (v *Validator) checkDangerousPatterns(code string) []*ValidationError {
	var errors []*ValidationError

	for _, pattern := range v.DangerousPatterns {
		if loc := pattern.FindStringIndex(code); loc != nil {
			lineNum := countLines(code[:loc[0]])
			errors = append(errors, SafetyError("Potentially dangerous construct detected").
				WithLine(lineNum).
				WithSuggestion("Remove system calls and file operations"))
		}
	}

	return errors
}

// checkWarnings checks for non-fatal issues.
func (v *Validator) checkWarnings(code string) []string {
	var warnings []string

	// Check for required constructs
	hasTemplate := strings.Contains(code, "deftemplate")
	hasRule := strings.Contains(code, "defrule")

	if !hasTemplate && !hasRule {
		warnings = append(warnings, "No deftemplate or defrule found")
	} else if !hasTemplate {
		warnings = append(warnings, "No deftemplate found - rules may use ordered facts only")
	} else if !hasRule {
		warnings = append(warnings, "No defrule found - code defines templates but no rules")
	}

	// Check for comments
	if !strings.Contains(code, ";;") && !strings.Contains(code, ";") {
		warnings = append(warnings, "No comments found - consider adding documentation")
	}

	// Check for salience without explicit value
	if strings.Contains(code, "(declare") && !strings.Contains(code, "salience") {
		warnings = append(warnings, "declare block without salience - using default priority")
	}

	return warnings
}

// ValidateCode is a convenience function that validates CLIPS code.
func ValidateCode(code string) *ValidationResult {
	return NewValidator().Validate(code)
}

// countLines counts the number of lines up to a position in the code.
func countLines(code string) int {
	return strings.Count(code, "\n") + 1
}

// ExtractConstructs extracts CLIPS construct names from code.
func ExtractConstructs(code string) []string {
	var constructs []string

	patterns := map[string]*regexp.Regexp{
		"deftemplate": regexp.MustCompile(`\(\s*deftemplate\s+([a-zA-Z_][a-zA-Z0-9_-]*)`),
		"defrule":     regexp.MustCompile(`\(\s*defrule\s+([a-zA-Z_][a-zA-Z0-9_-]*)`),
		"deffunction": regexp.MustCompile(`\(\s*deffunction\s+([a-zA-Z_][a-zA-Z0-9_-]*)`),
		"defmodule":   regexp.MustCompile(`\(\s*defmodule\s+([a-zA-Z_][a-zA-Z0-9_-]*)`),
		"deffacts":    regexp.MustCompile(`\(\s*deffacts\s+([a-zA-Z_][a-zA-Z0-9_-]*)`),
		"defglobal":   regexp.MustCompile(`\(\s*defglobal\s+`),
	}

	for constructType, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(code, -1)
		for _, match := range matches {
			if len(match) > 1 {
				constructs = append(constructs, constructType+":"+match[1])
			} else {
				constructs = append(constructs, constructType)
			}
		}
	}

	return constructs
}

// HasConstruct checks if a specific construct type exists in the code.
func HasConstruct(code, constructType string) bool {
	pattern := regexp.MustCompile(`\(\s*` + constructType + `\s+`)
	return pattern.MatchString(code)
}
