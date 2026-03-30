//! Example: Streaming with Token Budget
//!
//! ## nxusKit Features Demonstrated
//! - Stream cancellation with budget tracking
//! - Normalized token counting across providers
//! - Graceful stream termination
//! - Cost estimation during streaming
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show raw request/response data
//! - `--step` or `-s`: Pause at each step with explanations
//!
//! ## Why This Pattern Matters
//! Token budgets enable cost control and prevent runaway API costs.
//! nxusKit's streaming interface supports cancellation at any point,
//! and provides estimated token counts even for partial responses.
//!
//! Usage:
//! ```bash
//! cargo run
//! cargo run -- --verbose    # Show request/response details
//! cargo run -- --step       # Step through with explanations
//! ```
//!
//! Demonstrates cost control by limiting tokens during streaming.

use nxuskit::prelude::*;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
use token_budget::stream_with_budget;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse interactive mode flags
    let mut config = InteractiveConfig::from_args();

    println!("=== Streaming with Token Budget Demo ===\n");

    // Step: Creating provider
    if config.step_pause(
        "Creating Ollama provider for local testing...",
        &[
            "nxusKit: Local provider is ideal for testing budget logic",
            "No API key needed - runs against local Ollama instance",
            "Same interface as cloud providers (OpenAI, Claude)",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // nxusKit: Local provider is ideal for testing budget logic
    let provider = OllamaProvider::builder()
        .base_url("http://localhost:11434")
        .build()?;

    // Step: Building request
    if config.step_pause(
        "Building chat request...",
        &[
            "nxusKit: Same request structure works across all providers",
            "The prompt is designed to generate a longer response",
            "This helps demonstrate the budget cutoff behavior",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let request = ChatRequest::new("llama3").with_message(Message::user(
        "Write a short story about a robot learning to paint. Be creative!",
    ));

    // Verbose: Show the request
    config.print_request("POST", "http://localhost:11434/api/chat", &request);

    // Step: Setting budget
    if config.step_pause(
        "Setting token budget limit...",
        &[
            "nxusKit: Budget limit - stream will be cancelled when exceeded",
            "50 tokens is intentionally low to demonstrate early termination",
            "In production, you might use larger limits for cost control",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // nxusKit: Budget limit - stream will be cancelled when exceeded
    let max_tokens = 50;
    println!("Token budget: {} tokens", max_tokens);
    println!("Streaming response...\n");

    // Step: Streaming with budget
    if config.step_pause(
        "Starting streaming with budget tracking...",
        &[
            "nxusKit: stream_with_budget wraps the stream and tracks token usage",
            "Tokens are estimated in real-time as chunks arrive",
            "Stream is gracefully cancelled when budget is exceeded",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // nxusKit: stream_with_budget wraps the stream and tracks token usage
    match stream_with_budget(&provider, &request, max_tokens) {
        Ok(result) => {
            println!("\n=== Result ===");
            println!("Content: {}", result.content);
            println!("Estimated tokens: {}", result.estimated_tokens);
            println!(
                "Budget reached: {}",
                if result.budget_reached { "Yes" } else { "No" }
            );
        }
        Err(e) => {
            println!("Error: {}", e);
        }
    }

    Ok(())
}
