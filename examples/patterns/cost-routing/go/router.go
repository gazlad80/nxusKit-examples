// Package main demonstrates the model router (cost tiers) pattern.
package main

import (
	"context"
	"fmt"
	"strings"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// CostTier represents a cost/quality tier for model selection.
type CostTier int

const (
	// TierEconomy uses fast, cheap models for simple tasks
	TierEconomy CostTier = iota
	// TierStandard uses balanced models for general tasks
	TierStandard
	// TierPremium uses high-quality models for complex reasoning
	TierPremium
)

// ModelName returns the recommended model for this tier.
func (t CostTier) ModelName() string {
	switch t {
	case TierEconomy:
		return "gpt-4o-mini"
	case TierStandard:
		return "gpt-4o"
	case TierPremium:
		return "gpt-4-turbo"
	default:
		return "gpt-4o"
	}
}

// Name returns the tier name as a string.
func (t CostTier) Name() string {
	switch t {
	case TierEconomy:
		return "Economy"
	case TierStandard:
		return "Standard"
	case TierPremium:
		return "Premium"
	default:
		return "Unknown"
	}
}

// ClassifyTask determines the appropriate cost tier based on prompt characteristics.
//
// Uses simple heuristics:
// - Premium: Contains "analyze", "compare", or > 1000 chars
// - Standard: > 200 chars
// - Economy: Simple/short prompts
func ClassifyTask(prompt string) CostTier {
	promptLower := strings.ToLower(prompt)

	// Premium tier indicators
	complexKeywords := []string{"analyze", "compare", "evaluate", "synthesize", "critique"}
	for _, keyword := range complexKeywords {
		if strings.Contains(promptLower, keyword) {
			return TierPremium
		}
	}

	// Long prompts get premium tier
	if len(prompt) > 1000 {
		return TierPremium
	}

	// Medium length gets standard tier
	if len(prompt) > 200 {
		return TierStandard
	}

	// Short/simple prompts get economy tier
	return TierEconomy
}

// RoutedChatResult contains the response and the tier that was used.
type RoutedChatResult struct {
	Response *nxuskit.ChatResponse
	Tier     CostTier
}

// RoutedChat sends a chat request using the appropriate model based on task complexity.
func RoutedChat(ctx context.Context, provider nxuskit.LLMProvider, prompt string) (*RoutedChatResult, error) {
	tier := ClassifyTask(prompt)
	model := tier.ModelName()

	fmt.Printf("Task classified as: %s (using %s)\n", tier.Name(), model)

	req, err := nxuskit.NewChatRequest(model,
		nxuskit.WithMessages(nxuskit.UserMessage(prompt)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("chat request failed: %w", err)
	}

	return &RoutedChatResult{
		Response: resp,
		Tier:     tier,
	}, nil
}
