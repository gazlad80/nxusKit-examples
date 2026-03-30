// Package main demonstrates streaming with token budget enforcement.
package main

import (
	"context"
	"errors"
	"strings"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// BudgetStreamResult contains the result of streaming with budget enforcement.
type BudgetStreamResult struct {
	// Content is the accumulated text from the stream
	Content string
	// EstimatedTokens is the approximate token count
	EstimatedTokens int
	// BudgetReached indicates if streaming stopped due to budget
	BudgetReached bool
}

// StreamWithBudget streams a response while enforcing a token budget.
// Uses ~4 characters per token as a rough estimate.
// Stops streaming when estimated tokens reach the budget.
func StreamWithBudget(ctx context.Context, provider nxuskit.LLMProvider, req *nxuskit.ChatRequest, maxTokens int) (*BudgetStreamResult, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	chunks, errs := provider.ChatStream(ctx, req)
	var output strings.Builder
	budgetReached := false

	for chunk := range chunks {
		output.WriteString(chunk.Delta)

		// Estimate tokens: ~4 characters per token
		estimatedTokens := output.Len() / 4

		if estimatedTokens >= maxTokens {
			budgetReached = true
			cancel() // Stop the stream
			break
		}
	}

	// Drain the error channel
	if err := <-errs; err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	return &BudgetStreamResult{
		Content:         output.String(),
		EstimatedTokens: output.Len() / 4,
		BudgetReached:   budgetReached,
	}, nil
}
