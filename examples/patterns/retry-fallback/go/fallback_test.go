// Package main tests for the multi-provider fallback pattern.
package main

import (
	"context"
	"errors"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestChatWithFallback_FirstProviderSucceeds(t *testing.T) {
	ctx := context.Background()

	// Create mock provider that succeeds
	successProvider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Success from first provider",
			Model:   "test-model",
		}),
	)

	failProvider := nxuskit.NewMockProvider(
		nxuskit.WithMockError(errors.New("provider failed")),
	)

	providers := []nxuskit.LLMProvider{successProvider, failProvider}

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := ChatWithFallback(ctx, providers, req)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if resp.Content != "Success from first provider" {
		t.Errorf("Expected 'Success from first provider', got '%s'", resp.Content)
	}
}

func TestChatWithFallback_FirstFailsSecondSucceeds(t *testing.T) {
	ctx := context.Background()

	failProvider := nxuskit.NewMockProvider(
		nxuskit.WithMockError(errors.New("first provider failed")),
	)

	successProvider := nxuskit.NewMockProvider(
		nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Success from second provider",
			Model:   "test-model",
		}),
	)

	providers := []nxuskit.LLMProvider{failProvider, successProvider}

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := ChatWithFallback(ctx, providers, req)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if resp.Content != "Success from second provider" {
		t.Errorf("Expected 'Success from second provider', got '%s'", resp.Content)
	}
}

func TestChatWithFallback_AllProvidersFail(t *testing.T) {
	ctx := context.Background()

	fail1 := nxuskit.NewMockProvider(
		nxuskit.WithMockError(errors.New("provider 1 failed")),
	)

	fail2 := nxuskit.NewMockProvider(
		nxuskit.WithMockError(errors.New("provider 2 failed")),
	)

	providers := []nxuskit.LLMProvider{fail1, fail2}

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = ChatWithFallback(ctx, providers, req)
	if err == nil {
		t.Fatal("Expected error when all providers fail")
	}
}

func TestChatWithFallback_NoProviders(t *testing.T) {
	ctx := context.Background()

	providers := []nxuskit.LLMProvider{}

	req, err := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("test")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = ChatWithFallback(ctx, providers, req)
	if err == nil {
		t.Fatal("Expected error with no providers")
	}
}
