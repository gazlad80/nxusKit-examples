//! Example: Polymorphic Provider Usage
//!
//! ## nxusKit Features Demonstrated
//! - Trait object polymorphism (Box<dyn AsyncProvider>)
//! - Provider registry pattern for dynamic provider management
//! - Safe async trait dispatch with object safety
//! - Runtime provider discovery and selection
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show raw request/response data
//! - `--step` or `-s`: Pause at each API call with explanations
//!
//! ## Why This Pattern Matters
//! Polymorphic providers enable runtime provider selection, plugin architectures,
//! and dynamic configuration. nxusKit's trait design ensures type safety while
//! supporting dynamic dispatch through carefully designed trait bounds.
//!
//! Usage:
//! ```bash
//! cargo run
//! cargo run -- --verbose    # Show request/response details
//! cargo run -- --step       # Step through with explanations
//! ```

use nxuskit::ModelInfo;
use nxuskit::prelude::*;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
use std::collections::HashMap;

/// nxusKit: Provider registry using trait objects for runtime polymorphism
struct ProviderRegistry {
    providers: HashMap<String, Box<dyn AsyncProvider>>,
}

impl ProviderRegistry {
    fn new() -> Self {
        Self {
            providers: HashMap::new(),
        }
    }

    /// nxusKit: Type erasure allows any AsyncProvider impl to be registered
    fn register(&mut self, name: impl Into<String>, provider: impl AsyncProvider + 'static) {
        self.providers.insert(name.into(), Box::new(provider));
    }

    /// List all models from all providers
    async fn discover_all_models(&self) -> Vec<(String, Vec<ModelInfo>)> {
        let mut results = Vec::new();

        for (name, provider) in &self.providers {
            match provider.list_models().await {
                Ok(models) => {
                    results.push((name.clone(), models));
                }
                Err(e) => {
                    eprintln!("Warning: Failed to list models from {}: {}", name, e);
                }
            }
        }

        results
    }

    /// Get a specific provider by name
    fn get(&self, name: &str) -> Option<&dyn AsyncProvider> {
        self.providers.get(name).map(|p| p.as_ref())
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse interactive mode flags
    let mut config = InteractiveConfig::from_args();

    println!("=== Polymorphic Provider Usage Example ===\n");

    // Step: Creating registry
    if config.step_pause(
        "Creating provider registry...",
        &[
            "nxusKit: Provider registry using trait objects for runtime polymorphism",
            "Box<dyn AsyncProvider> enables storing different provider types",
            "Type erasure allows any AsyncProvider impl to be registered",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Create a provider registry
    let mut registry = ProviderRegistry::new();

    // Register different providers
    // Each provider implements AsyncProvider, so they can be stored as trait objects
    let mock = MockProvider::new("Hello from mock!");

    let loopback = LoopbackProvider::builder().build()?;

    registry.register("mock", mock);
    registry.register("loopback", loopback);

    println!("Registered {} providers\n", registry.providers.len());

    // Verbose: Show registered providers
    if config.verbose {
        println!("[VERBOSE] Registered providers:");
        for name in registry.providers.keys() {
            println!("[VERBOSE]   - {}", name);
        }
        println!();
    }

    // Step: Discovering models
    if config.step_pause(
        "Discovering models from all providers...",
        &[
            "Iterates through all registered providers",
            "Calls list_available_models() on each via trait object",
            "Demonstrates runtime polymorphism with async traits",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Discover all models from all providers
    println!("Discovering models from all providers...\n");
    let all_models = registry.discover_all_models().await;

    for (provider_name, models) in &all_models {
        println!("Provider: {} ({} models)", provider_name, models.len());
        println!("{}", "-".repeat(40));

        for model in models.iter().take(5) {
            // Show first 5 models
            println!("  - {}", model.name);
        }

        if models.len() > 5 {
            println!("  ... and {} more", models.len() - 5);
        }
        println!();
    }

    // Verbose: Show total model count
    if config.verbose {
        let total: usize = all_models.iter().map(|(_, m)| m.len()).sum();
        println!("[VERBOSE] Total models discovered: {}\n", total);
    }

    // Step: Query specific provider
    if config.step_pause(
        "Querying specific provider 'mock'...",
        &[
            "Registry.get() returns Option<&dyn AsyncProvider>",
            "Can query any provider by name at runtime",
            "Demonstrates dynamic provider selection",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Example: Query a specific provider
    println!("Querying specific provider 'mock'...");
    if let Some(mock_provider) = registry.get("mock") {
        let models = mock_provider.list_models().await?;
        println!("Mock provider has {} models:", models.len());
        for model in &models {
            println!("  - {}", model.name);
        }
    }

    println!("\n=== Example Complete ===");
    Ok(())
}
