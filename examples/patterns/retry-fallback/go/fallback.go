// Package main demonstrates the multi-provider fallback pattern.
package main

import (
	"context"
	"fmt"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// ChatWithFallback attempts to send a chat request through multiple providers
// in sequence. Returns the response from the first provider that succeeds,
// or an error if all providers fail.
func ChatWithFallback(ctx context.Context, providers []nxuskit.LLMProvider, req *nxuskit.ChatRequest) (*nxuskit.ChatResponse, error) {
	var lastErr error

	for i, provider := range providers {
		resp, err := provider.Chat(ctx, req)
		if err == nil {
			fmt.Printf("Provider %d (%s) succeeded\n", i+1, provider.ProviderName())
			return resp, nil
		}
		fmt.Printf("Provider %d (%s) failed: %v\n", i+1, provider.ProviderName(), err)
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all providers failed, last error: %w", lastErr)
	}
	return nil, fmt.Errorf("no providers available")
}
