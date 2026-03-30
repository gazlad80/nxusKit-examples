// Package ruler provides prompts for LLM-based CLIPS code generation.
package ruler

import "fmt"

// SystemPrompt is the system prompt for CLIPS code generation.
const SystemPrompt = `You are an expert CLIPS (C Language Integrated Production System) programmer.
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
The code must be immediately loadable into a CLIPS environment.`

// GeneratePromptTemplate is the template for generating CLIPS from a description.
const GeneratePromptTemplate = `Generate CLIPS rules for the following requirement:

Description: %s
Complexity Level: %s
Domain Hints: %s

Requirements:
1. Create appropriate deftemplates for the domain
2. Create defrules that implement the described behavior
3. Include comments explaining the logic
4. Ensure the code is valid and can be loaded into CLIPS

Generate the CLIPS code now:`

// ValidationPromptTemplate is used to ask the LLM to fix invalid code.
const ValidationPromptTemplate = `The following CLIPS code has validation errors:

CODE:
%s

ERRORS:
%s

Please fix the code to address these errors while maintaining the original functionality.
Return only the corrected CLIPS code.`

// PromptConfig holds configuration for prompt generation.
type PromptConfig struct {
	// SystemPrompt is the system prompt to use.
	SystemPrompt string
	// MaxTokens is the maximum tokens for the response.
	MaxTokens int
	// Temperature for generation (0.0-1.0).
	Temperature float64
}

// DefaultPromptConfig returns the default prompt configuration.
func DefaultPromptConfig() *PromptConfig {
	return &PromptConfig{
		SystemPrompt: SystemPrompt,
		MaxTokens:    4096,
		Temperature:  0.2,
	}
}

// FormatGeneratePrompt creates a generate prompt from a rule description.
func FormatGeneratePrompt(desc *RuleDescription) string {
	domainHints := ""
	if len(desc.DomainHints) > 0 {
		for i, hint := range desc.DomainHints {
			if i > 0 {
				domainHints += ", "
			}
			domainHints += hint
		}
	} else {
		domainHints = "(none)"
	}

	return fmt.Sprintf(GeneratePromptTemplate, desc.Description, desc.Complexity, domainHints)
}

// FormatValidationPrompt creates a validation fix prompt.
func FormatValidationPrompt(code string, errors []*ValidationError) string {
	errorStr := ""
	for _, err := range errors {
		if errorStr != "" {
			errorStr += "\n"
		}
		errorStr += err.Error()
	}
	return fmt.Sprintf(ValidationPromptTemplate, code, errorStr)
}
