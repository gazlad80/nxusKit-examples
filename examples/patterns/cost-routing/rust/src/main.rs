//! Example: Model Router (Cost Tiers)
//!
//! ## nxusKit Features Demonstrated
//! - Task complexity classification
//! - Dynamic model selection based on task requirements
//! - Cost-tier routing (economy/standard/premium)
//! - Provider-agnostic request routing
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show raw request/response data
//! - `--step` or `-s`: Pause at each API call with explanations
//!
//! ## Why This Pattern Matters
//! Using expensive models for simple queries wastes money. This pattern
//! demonstrates how nxusKit's unified interface enables intelligent routing
//! to appropriate models based on task complexity, optimizing cost/quality.
//!
//! Usage:
//! ```bash
//! cargo run
//! cargo run -- --verbose    # Show request/response details
//! cargo run -- --step       # Step through with explanations
//! ```

use cost_routing::{CostTier, classify_task};
use nxuskit::builders::OllamaProvider;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse interactive mode flags
    let mut config = InteractiveConfig::from_args();

    println!("=== Model Router (Cost Tiers) Demo ===\n");

    // Step: Creating provider
    if config.step_pause(
        "Creating Ollama provider...",
        &[
            "nxusKit: Provider builder pattern with sensible defaults",
            "Connects to local Ollama instance at localhost:11434",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let provider = OllamaProvider::builder()
        .base_url("http://localhost:11434")
        .build()?;

    // Verbose: Show provider configuration
    if config.verbose {
        println!("[VERBOSE] Provider: Ollama");
        println!("[VERBOSE] Base URL: http://localhost:11434\n");
    }

    let prompts = vec![
        ("Simple", "What is 2+2?"),
        (
            "Medium",
            "Explain the concept of recursion in programming. Include an example of how it works and when you might use it in practice.",
        ),
        (
            "Complex",
            "Analyze the trade-offs between microservices and monolithic architectures. Compare their scalability, maintainability, deployment complexity, and team coordination requirements.",
        ),
    ];

    // Step: Processing prompts
    if config.step_pause(
        "Processing prompts through cost router...",
        &[
            "Each prompt is analyzed for complexity",
            "classify_task() determines the appropriate cost tier",
            "Tier maps to a specific model (economy/standard/premium)",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    for (label, prompt) in prompts {
        println!("--- {} Prompt ---", label);
        println!("Input: {}...", &prompt[..prompt.len().min(50)]);

        // nxusKit: classify_task determines appropriate cost tier for the prompt
        let tier = classify_task(prompt);
        println!(
            "Classified as: {} (would use: {})",
            tier.name(),
            tier.model_name()
        );

        // Verbose: Show classification details
        if config.verbose {
            println!("[VERBOSE] Prompt length: {} chars", prompt.len());
            println!("[VERBOSE] Selected tier: {:?}", tier.name());
            println!("[VERBOSE] Target model: {}", tier.model_name());
        }

        // For demo, we'll just show classification without making actual API calls
        // In production, uncomment this:
        // match routed_chat(&provider, prompt).await {
        //     Ok((response, tier)) => {
        //         println!("Response: {}", response.content);
        //     }
        //     Err(e) => println!("Error: {}", e),
        // }
        println!();
    }

    // Show tier breakdown
    println!("=== Tier Summary ===");
    println!(
        "Economy ({}): Short, simple queries",
        CostTier::Economy.model_name()
    );
    println!(
        "Standard ({}): Medium complexity",
        CostTier::Standard.model_name()
    );
    println!(
        "Premium ({}): Complex analysis",
        CostTier::Premium.model_name()
    );

    // Keep provider in scope to avoid unused warning
    let _ = provider;

    Ok(())
}
