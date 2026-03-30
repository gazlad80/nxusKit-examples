// Package main demonstrates the CLIPS+LLM hybrid pattern.
//
// ## nxusKit Features Demonstrated
// - Hybrid inference: LLM (probabilistic) + CLIPS (deterministic) integration
// - ClipsProvider for rule-based expert system execution
// - JSON-based fact assertion and conclusion extraction
// - LLMProvider interface for provider-agnostic code
// - JSONMode for structured LLM output
//
// ## Why This Pattern Matters
// LLMs excel at understanding unstructured input but can be unpredictable for
// business rules. CLIPS provides deterministic, auditable rule execution.
// Combining both gives you the best of both worlds: natural language understanding
// with predictable, explainable business logic.
//
// ## Architecture
// 1. LLM classifies unstructured ticket → structured facts
// 2. CLIPS applies business rules → routing decision
// 3. LLM generates human-friendly response
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// TicketClassification represents LLM-extracted classification from a ticket.
type TicketClassification struct {
	// Category: "security", "infrastructure", "application", "general"
	Category string `json:"category"`
	// Priority: "low", "medium", "high", "critical"
	Priority string `json:"priority"`
	// Sentiment: "positive", "neutral", "negative", "frustrated"
	Sentiment string `json:"sentiment"`
	// KeyEntities mentioned in the ticket
	KeyEntities []string `json:"key_entities"`
}

// RoutingDecision represents the output from CLIPS expert system.
// This structure matches the routing-decision deftemplate in ticket-routing.clp.
type RoutingDecision struct {
	// Team to handle the ticket (maps to CLIPS slot: team)
	Team string `json:"team"`
	// SLAHours is the SLA deadline in hours (maps to CLIPS slot: sla-hours)
	SLAHours int `json:"sla_hours"`
	// EscalationLevel: 0 = none, 1 = manager, 2 = director (maps to CLIPS slot: escalation-level)
	EscalationLevel int `json:"escalation_level"`
}

// TicketAnalysis represents the combined LLM + CLIPS result.
type TicketAnalysis struct {
	// From CLIPS (deterministic)
	Team            string `json:"team"`
	SLAHours        int    `json:"sla_hours"`
	EscalationLevel int    `json:"escalation_level"`

	// From LLM (probabilistic)
	Sentiment         string   `json:"sentiment"`
	KeyEntities       []string `json:"key_entities"`
	SuggestedResponse string   `json:"suggested_response"`
}

const classifyPrompt = `Classify this support ticket. Respond with JSON only.
Format: {"category": "security|infrastructure|application|general", "priority": "low|medium|high|critical", "sentiment": "positive|neutral|negative|frustrated", "key_entities": ["entity1", "entity2"]}`

const responsePrompt = `Generate a brief, empathetic initial response for this support ticket.
Keep it under 100 words. Acknowledge the issue and set expectations.`

// ApplyRoutingRules applies deterministic routing rules using the CLIPS expert system.
//
// This function:
// 1. Creates a ClipsProvider configured with the ticket-routing.clp rules
// 2. Asserts the ticket classification as a CLIPS fact
// 3. Runs the CLIPS inference engine
// 4. Extracts the routing-decision conclusion
//
// nxusKit Feature: ClipsProvider for deterministic rule execution
// The ClipsProvider implements LLMProvider, enabling CLIPS to be used anywhere
// an LLM provider is expected. This allows mixing rule-based and ML-based
// inference in the same pipeline.
func ApplyRoutingRules(ctx context.Context, classification *TicketClassification, rulesPath string) (*RoutingDecision, error) {
	// nxusKit: ClipsProvider with functional options pattern
	rulesDir := filepath.Dir(rulesPath)
	clips, err := nxuskit.NewClipsFFIProvider(rulesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIPS provider: %w", err)
	}

	// Determine if ticket has security keywords
	hasSecurityKeywords := "no"
	for _, entity := range classification.KeyEntities {
		lower := entity
		if contains(lower, "breach") || contains(lower, "hack") || contains(lower, "unauthorized") {
			hasSecurityKeywords = "yes"
			break
		}
	}

	// nxusKit: Format input as JSON for CLIPS provider (ClipsInput-shaped; see clips_wire.go).
	clipsInput := clipsInputWire{
		Facts: []clipsFactWire{
			{
				Template: "ticket-classification",
				Values: map[string]interface{}{
					"category":              classification.Category,
					"priority":              classification.Priority,
					"sentiment":             classification.Sentiment,
					"has-security-keywords": hasSecurityKeywords,
				},
			},
		},
	}

	inputJSON, err := json.Marshal(clipsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CLIPS input: %w", err)
	}

	// nxusKit: Use the rules file name as the "model" parameter
	model := filepath.Base(rulesPath)

	// nxusKit: ChatRequest works uniformly for CLIPS and LLM providers
	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(nxuskit.UserMessage(string(inputJSON))),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CLIPS request: %w", err)
	}

	// nxusKit: Same Chat() interface for rule execution as for LLM inference
	resp, err := clips.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("CLIPS inference failed: %w", err)
	}

	// Parse CLIPS output to extract routing decision
	var output clipsOutputWire
	if err := json.Unmarshal([]byte(resp.Content), &output); err != nil {
		return nil, fmt.Errorf("failed to parse CLIPS output: %w", err)
	}

	// Find the routing-decision conclusion
	for _, conclusion := range output.Conclusions {
		if conclusion.Template == "routing-decision" {
			// nxusKit: Extract values from CLIPS conclusion
			team := "general-support"
			if t, ok := conclusion.Values["team"].(string); ok {
				team = t
			}
			slaHours := 24
			if s, ok := conclusion.Values["sla-hours"].(float64); ok {
				slaHours = int(s)
			}
			escalationLevel := 0
			if e, ok := conclusion.Values["escalation-level"].(float64); ok {
				escalationLevel = int(e)
			}

			return &RoutingDecision{
				Team:            team,
				SLAHours:        slaHours,
				EscalationLevel: escalationLevel,
			}, nil
		}
	}

	return nil, fmt.Errorf("no routing-decision derived from CLIPS rules")
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldAt(s, i, substr) {
			return true
		}
	}
	return false
}

func equalFoldAt(s string, start int, substr string) bool {
	for j := 0; j < len(substr); j++ {
		c1, c2 := s[start+j], substr[j]
		if c1 != c2 && toLower(c1) != toLower(c2) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// ApplyRoutingRulesSync is a legacy synchronous routing function for testing.
// ApplyRoutingRulesSync provides a synchronous version for unit testing.
func ApplyRoutingRulesSync(classification *TicketClassification) *RoutingDecision {
	// Security tickets always go to security team with high priority
	if classification.Category == "security" {
		return &RoutingDecision{
			Team:            "security",
			SLAHours:        4,
			EscalationLevel: 2,
		}
	}

	// Infrastructure with critical priority goes to SRE
	if classification.Category == "infrastructure" && classification.Priority == "critical" {
		return &RoutingDecision{
			Team:            "sre",
			SLAHours:        2,
			EscalationLevel: 1,
		}
	}

	// High priority infrastructure goes to SRE
	if classification.Category == "infrastructure" && classification.Priority == "high" {
		return &RoutingDecision{
			Team:            "sre",
			SLAHours:        4,
			EscalationLevel: 1,
		}
	}

	// Application issues go to development
	if classification.Category == "application" {
		sla := 24
		escalation := 0
		switch classification.Priority {
		case "critical":
			sla = 4
			escalation = 1
		case "high":
			sla = 8
		}
		return &RoutingDecision{
			Team:            "development",
			SLAHours:        sla,
			EscalationLevel: escalation,
		}
	}

	// Default routing for general support
	return &RoutingDecision{
		Team:            "general-support",
		SLAHours:        24,
		EscalationLevel: 0,
	}
}

// AnalyzeTicket performs hybrid analysis using LLM + CLIPS pattern.
//
// ## nxusKit Features Demonstrated
// - LLMProvider interface enables provider-agnostic LLM usage
// - ClipsProvider for deterministic rule execution (via ApplyRoutingRules)
// - JSONMode for structured LLM output
// - Unified context-based interface for both LLM and CLIPS providers
//
// ## Three-step flow:
// 1. **LLM** classifies the ticket (extracts category, priority, sentiment, entities)
// 2. **CLIPS** applies deterministic routing rules (auditable, explainable)
// 3. **LLM** generates a suggested response (natural language)
func AnalyzeTicket(ctx context.Context, llm nxuskit.LLMProvider, model, ticketText, rulesPath string) (*TicketAnalysis, error) {
	// Step 1: LLM classification
	// nxusKit: ChatRequest with functional options pattern
	classifyReq, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.SystemMessage(classifyPrompt),
			nxuskit.UserMessage(ticketText),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create classify request: %w", err)
	}

	// nxusKit: JSONMode enables structured output mode
	classifyReq.JSONMode = true

	// nxusKit: Unified context-based interface - same pattern for all providers
	classifyResp, err := llm.Chat(ctx, classifyReq)
	if err != nil {
		return nil, fmt.Errorf("classification failed: %w", err)
	}

	var classification TicketClassification
	if err := json.Unmarshal([]byte(classifyResp.Content), &classification); err != nil {
		return nil, fmt.Errorf("failed to parse classification: %w", err)
	}

	// Step 2: Apply CLIPS routing rules (deterministic)
	// nxusKit: ClipsProvider integrates seamlessly - rules execute via same interface
	routing, err := ApplyRoutingRules(ctx, &classification, rulesPath)
	if err != nil {
		return nil, fmt.Errorf("CLIPS routing failed: %w", err)
	}

	// Step 3: LLM generates suggested response
	responseReq, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(
			nxuskit.SystemMessage(responsePrompt),
			nxuskit.UserMessage(ticketText),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create response request: %w", err)
	}

	responseResp, err := llm.Chat(ctx, responseReq)
	if err != nil {
		return nil, fmt.Errorf("response generation failed: %w", err)
	}

	return &TicketAnalysis{
		Team:              routing.Team,
		SLAHours:          routing.SLAHours,
		EscalationLevel:   routing.EscalationLevel,
		Sentiment:         classification.Sentiment,
		KeyEntities:       classification.KeyEntities,
		SuggestedResponse: responseResp.Content,
	}, nil
}
