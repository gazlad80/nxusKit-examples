package main

import (
	"context"
	"sync"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestMultiProvider_ConcurrentCalls(t *testing.T) {
	ctx := context.Background()

	// Create multiple mock providers
	providers := map[string]*nxuskit.MockProvider{
		"provider1": nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Response from provider 1",
			Model:   "model-1",
		})),
		"provider2": nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Response from provider 2",
			Model:   "model-2",
		})),
		"provider3": nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Response from provider 3",
			Model:   "model-3",
		})),
	}

	req, _ := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Hello")),
	)

	// Call all providers concurrently
	var wg sync.WaitGroup
	results := make(map[string]string)
	var mu sync.Mutex

	for name, provider := range providers {
		wg.Add(1)
		go func(name string, p *nxuskit.MockProvider) {
			defer wg.Done()
			resp, err := p.Chat(ctx, req)
			if err != nil {
				t.Errorf("Provider %s failed: %v", name, err)
				return
			}
			mu.Lock()
			results[name] = resp.Content
			mu.Unlock()
		}(name, provider)
	}

	wg.Wait()

	// Verify all providers responded
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for name := range providers {
		if _, ok := results[name]; !ok {
			t.Errorf("Missing result from provider %s", name)
		}
	}
}

func TestMultiProvider_PartialFailure(t *testing.T) {
	ctx := context.Background()

	// Create providers - one will fail
	successProvider := nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
		Content: "Success",
		Model:   "model-1",
	}))
	failProvider := nxuskit.NewMockProvider(nxuskit.WithMockError(
		nxuskit.NewProviderError("mock", "simulated failure", 500, nil),
	))

	req, _ := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Hello")),
	)

	// Success provider should work
	resp, err := successProvider.Chat(ctx, req)
	if err != nil {
		t.Errorf("Success provider should not fail: %v", err)
	}
	if resp.Content != "Success" {
		t.Errorf("Expected 'Success', got '%s'", resp.Content)
	}

	// Fail provider should return error
	_, err = failProvider.Chat(ctx, req)
	if err == nil {
		t.Error("Fail provider should return error")
	}
}

func TestMultiProvider_ResponseComparison(t *testing.T) {
	ctx := context.Background()

	// Create providers with different response characteristics
	providers := []*nxuskit.MockProvider{
		nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Short response",
			Model:   "fast-model",
			Usage: nxuskit.TokenUsage{
				Actual: &nxuskit.TokenCount{
					PromptTokens:     10,
					CompletionTokens: 5,
				},
			},
		})),
		nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "This is a much longer and more detailed response from a different provider",
			Model:   "detailed-model",
			Usage: nxuskit.TokenUsage{
				Actual: &nxuskit.TokenCount{
					PromptTokens:     10,
					CompletionTokens: 20,
				},
			},
		})),
	}

	req, _ := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Test")),
	)

	// Collect responses
	var responses []*nxuskit.ChatResponse
	for _, provider := range providers {
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			t.Fatalf("Provider failed: %v", err)
		}
		responses = append(responses, resp)
	}

	// Verify we got different responses
	if responses[0].Content == responses[1].Content {
		t.Error("Expected different responses from different providers")
	}

	// Verify token counts differ
	if responses[0].Usage.TotalTokens() == responses[1].Usage.TotalTokens() {
		t.Error("Expected different token counts")
	}
}
