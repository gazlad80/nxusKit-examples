package main

import (
	"context"
	"testing"

	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

func TestPolymorphic_ProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Register mock providers
	mock1 := nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
		Content: "Response from mock1",
	}))
	mock2 := nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
		Content: "Response from mock2",
	}))

	registry.Register("mock1", mock1)
	registry.Register("mock2", mock2)

	// Verify registration
	names := registry.Names()
	if len(names) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(names))
	}
}

func TestPolymorphic_GetProvider(t *testing.T) {
	registry := NewProviderRegistry()

	mock := nxuskit.NewMockProvider()
	registry.Register("test", mock)

	// Get existing provider
	provider, ok := registry.Get("test")
	if !ok {
		t.Error("Expected to find 'test' provider")
	}
	if provider == nil {
		t.Error("Provider should not be nil")
	}

	// Get non-existing provider
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Should not find 'nonexistent' provider")
	}
}

func TestPolymorphic_DiscoverAllModels(t *testing.T) {
	ctx := context.Background()

	registry := NewProviderRegistry()

	mock := nxuskit.NewMockProvider()
	loopback := nxuskit.NewLoopbackProvider()

	registry.Register("mock", mock)
	registry.Register("loopback", loopback)

	results := registry.DiscoverAllModels(ctx)

	// Should have entries for both providers
	if len(results) != 2 {
		t.Errorf("Expected results from 2 providers, got %d", len(results))
	}
}

func TestPolymorphic_InterfaceConformance(t *testing.T) {
	// Verify that different providers implement LLMProvider interface
	var _ nxuskit.LLMProvider = nxuskit.NewMockProvider()
	var _ nxuskit.LLMProvider = nxuskit.NewLoopbackProvider()

	// This test passes if it compiles - proves interface conformance
}

func TestPolymorphic_PolymorphicChat(t *testing.T) {
	ctx := context.Background()

	providers := map[string]nxuskit.LLMProvider{
		"mock": nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
			Content: "Mock response",
		})),
		"loopback": nxuskit.NewLoopbackProvider(),
	}

	req, err := nxuskit.NewChatRequest("any-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Hello")),
	)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Call Chat on each provider polymorphically
	for name, provider := range providers {
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			t.Errorf("Provider %s failed: %v", name, err)
			continue
		}
		if resp.Content == "" {
			t.Errorf("Provider %s returned empty content", name)
		}
	}
}

func TestPolymorphic_ConcurrentAccess(t *testing.T) {
	registry := NewProviderRegistry()

	// Register providers
	for i := 0; i < 5; i++ {
		name := string(rune('a' + i))
		registry.Register(name, nxuskit.NewMockProvider())
	}

	// Concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_ = registry.Names()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPolymorphic_MockProviderResponse(t *testing.T) {
	ctx := context.Background()

	expectedContent := "Custom mock response"
	mock := nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
		Content: expectedContent,
		Model:   "test-model",
	}))

	req, _ := nxuskit.NewChatRequest("test-model",
		nxuskit.WithMessages(nxuskit.UserMessage("Test")),
	)

	resp, err := mock.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	if resp.Content != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, resp.Content)
	}
}

func TestPolymorphic_LoopbackProvider(t *testing.T) {
	ctx := context.Background()

	loopback := nxuskit.NewLoopbackProvider()

	userMessage := "Echo this message"
	req, _ := nxuskit.NewChatRequest("any-model",
		nxuskit.WithMessages(nxuskit.UserMessage(userMessage)),
	)

	resp, err := loopback.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat failed: %v", err)
	}

	// Loopback echoes the input
	if resp.Content == "" {
		t.Error("Loopback should return non-empty content")
	}
}
