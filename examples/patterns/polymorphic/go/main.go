// Example: Polymorphic Provider Usage
//
// ## nxusKit Features Demonstrated
// - Interface-based polymorphism (nxuskit.LLMProvider)
// - Provider registry pattern for dynamic provider management
// - Go's structural typing (implicit interface satisfaction)
// - Runtime provider discovery and selection
//
// ## Interactive Modes
// - `--verbose` or `-v`: Show raw request/response data
// - `--step` or `-s`: Pause at each operation with explanations
//
// ## Why This Pattern Matters
// Polymorphic providers enable runtime provider selection, plugin architectures,
// and dynamic configuration. Go's implicit interfaces make this natural - any
// type implementing the required methods automatically satisfies LLMProvider.
//
// Usage:
//
//	go run .
//	go run . --verbose    # Show request/response details
//	go run . --step       # Step through with explanations
//
// See ../rust/src/main.rs for the Rust reference implementation.
package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive"
	nxuskit "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go"
)

// nxusKit: Provider registry using Go interfaces for runtime polymorphism
// Any type implementing LLMProvider methods is automatically compatible.
type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]nxuskit.LLMProvider
}

// NewProviderRegistry creates a new empty provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]nxuskit.LLMProvider),
	}
}

// Register adds a provider with a given name.
func (r *ProviderRegistry) Register(name string, provider nxuskit.LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

// Get retrieves a provider by name.
func (r *ProviderRegistry) Get(name string) (nxuskit.LLMProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// Names returns all registered provider names.
func (r *ProviderRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// DiscoverAllModels lists all models from all providers.
func (r *ProviderRegistry) DiscoverAllModels(ctx context.Context) map[string][]nxuskit.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	results := make(map[string][]nxuskit.ModelInfo)

	for name, provider := range r.providers {
		models, err := provider.ListModels(ctx)
		if err != nil {
			fmt.Printf("Warning: Failed to list models from %s: %v\n", name, err)
			continue
		}
		results[name] = models
	}

	return results
}

func main() {
	// Parse interactive mode flags
	config := interactive.FromArgs()

	fmt.Println("=== Polymorphic Provider Usage Example ===")
	fmt.Println()

	// Step: Introduction
	if config.StepPause("Understanding polymorphic providers...", []string{
		"nxusKit: nxuskit.LLMProvider is the core interface",
		"Any type implementing Chat() and ListModels() satisfies LLMProvider",
		"This enables runtime provider selection and plugin architectures",
		"Go's structural typing means no explicit 'implements' declaration",
	}) == interactive.ActionQuit {
		return
	}

	// Create a provider registry
	registry := NewProviderRegistry()

	// Step: Registry creation
	if config.StepPause("Creating provider registry...", []string{
		"The registry stores providers by name for dynamic lookup",
		"Thread-safe with sync.RWMutex for concurrent access",
		"Enables plugin-style provider loading at runtime",
	}) == interactive.ActionQuit {
		return
	}

	// Register different providers
	// Each provider implements LLMProvider, so they can be stored as interfaces

	// Mock provider - always available
	mock := nxuskit.NewMockProvider(nxuskit.WithMockResponse(&nxuskit.ChatResponse{
		Content: "Hello from mock!",
	}))
	registry.Register("mock", mock)

	// Loopback provider - echoes input back
	loopback := nxuskit.NewLoopbackProvider()
	registry.Register("loopback", loopback)

	// Verbose: Show registered providers
	if config.IsVerbose() {
		fmt.Printf("[nxusKit] Registered providers: %v\n", registry.Names())
	}

	fmt.Printf("Registered %d providers\n\n", len(registry.Names()))

	// Step: Model discovery
	if config.StepPause("Discovering models from all providers...", []string{
		"nxusKit: ListModels() works identically on any provider",
		"The registry iterates through all providers polymorphically",
		"Results are aggregated into a map by provider name",
	}) == interactive.ActionQuit {
		return
	}

	// Discover all models from all providers
	fmt.Println("Discovering models from all providers...")
	fmt.Println()

	ctx := context.Background()
	allModels := registry.DiscoverAllModels(ctx)

	for providerName, models := range allModels {
		fmt.Printf("Provider: %s (%d models)\n", providerName, len(models))
		fmt.Println(strings.Repeat("-", 40))

		// Show first 5 models
		for i, model := range models {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(models)-5)
				break
			}
			desc := ""
			if model.Description != nil && *model.Description != "" {
				desc = fmt.Sprintf(" - %s", *model.Description)
			}
			fmt.Printf("  - %s%s\n", model.Name, desc)
		}
		fmt.Println()
	}

	// Step: Querying specific provider
	if config.StepPause("Querying specific provider by name...", []string{
		"registry.Get() retrieves a provider by name",
		"Returns (provider, true) if found, (nil, false) otherwise",
		"The provider is still the concrete type behind the interface",
	}) == interactive.ActionQuit {
		return
	}

	// Example: Query a specific provider
	fmt.Println("Querying specific provider 'mock'...")
	if mockProvider, ok := registry.Get("mock"); ok {
		models, err := mockProvider.ListModels(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Mock provider has %d models:\n", len(models))
			for _, model := range models {
				vision := ""
				if model.SupportsVision() {
					vision = " [vision]"
				}
				fmt.Printf("  - %s%s\n", model.Name, vision)
			}
		}
	}

	// Step: Polymorphic chat
	if config.StepPause("Demonstrating polymorphic chat...", []string{
		"nxusKit: Same code works for any provider type",
		"Chat() is called through the interface, not concrete type",
		"Request/response handling is identical across providers",
	}) == interactive.ActionQuit {
		return
	}

	// Demonstrate polymorphic chat
	fmt.Println()
	fmt.Println("Demonstrating polymorphic chat:")
	fmt.Println()

	for _, name := range registry.Names() {
		provider, _ := registry.Get(name)

		req, err := nxuskit.NewChatRequest("any-model",
			nxuskit.WithMessages(nxuskit.UserMessage("Hello!")),
		)
		if err != nil {
			fmt.Printf("  %s: Error creating request: %v\n", name, err)
			continue
		}

		// Verbose: Show the request
		config.PrintRequest("POST", "mock://"+name+"/chat", req)

		start := time.Now()
		resp, err := provider.Chat(ctx, req)
		elapsedMs := time.Since(start).Milliseconds()

		if err != nil {
			fmt.Printf("  %s: Error: %v\n", name, err)
			continue
		}

		// Verbose: Show the response
		config.PrintResponse(200, elapsedMs, resp)

		// Truncate long responses for display
		content := resp.Content
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		fmt.Printf("  %s: %q\n", name, content)
	}

	// Step: Key concepts
	if config.StepPause("Reviewing Go interface concepts...", []string{
		"Structural typing: No explicit 'implements' declaration needed",
		"Interface variables: Store any compatible type",
		"Type assertions: Check for additional capabilities at runtime",
	}) == interactive.ActionQuit {
		return
	}

	fmt.Println()
	fmt.Println("=== Key Go Interface Concepts ===")
	fmt.Println()
	fmt.Println("1. Structural typing: Any type implementing LLMProvider methods")
	fmt.Println("   automatically satisfies the interface - no explicit declaration needed.")
	fmt.Println()
	fmt.Println("2. Interface variables: Store any provider as nxuskit.LLMProvider")
	fmt.Println("   and call methods polymorphically.")
	fmt.Println()
	fmt.Println("3. Type assertions: Use provider.(nxuskit.ModelLister) to check")
	fmt.Println("   if a provider implements additional interfaces.")
	fmt.Println()
	fmt.Println("Example:")
	fmt.Println("```go")
	fmt.Println("// Store any provider as interface")
	fmt.Println("var provider nxuskit.LLMProvider = nxuskit.NewMockProvider()")
	fmt.Println()
	fmt.Println("// Call methods polymorphically")
	fmt.Println("resp, err := provider.Chat(ctx, req)")
	fmt.Println()
	fmt.Println("// Check for additional capabilities")
	fmt.Println("if lister, ok := provider.(nxuskit.ModelLister); ok {")
	fmt.Println("    models, _ := lister.ListAvailableModels(ctx)")
	fmt.Println("}")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("=== Example Complete ===")
}
