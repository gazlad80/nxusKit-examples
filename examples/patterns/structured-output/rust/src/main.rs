//! Example: Structured Output (JSON Mode)
//!
//! ## nxusKit Features Demonstrated
//! - JSON schema-guided output generation
//! - Type-safe response parsing with serde
//! - Provider-agnostic structured output
//! - Schema validation and error handling
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show raw request/response data
//! - `--step` or `-s`: Pause at each step with explanations
//!
//! ## Why This Pattern Matters
//! Structured output enables reliable integration with downstream systems.
//! nxusKit handles the different JSON mode implementations across providers
//! (OpenAI's response_format, Claude's tool use, Ollama's format parameter).
//!
//! Usage:
//! ```bash
//! cargo run
//! cargo run -- --verbose    # Show request/response details
//! cargo run -- --step       # Step through with explanations
//! ```
//!
//! Demonstrates extracting typed structured data from LLM responses.

use nxuskit::builders::OllamaProvider;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
use structured_output::classify_log;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse interactive mode flags
    let mut config = InteractiveConfig::from_args();

    println!("=== Structured Output (JSON Mode) Demo ===\n");

    // Step: Creating provider
    if config.step_pause(
        "Creating Ollama provider for local development...",
        &[
            "nxusKit: Local provider for development - no API key needed",
            "JSON mode works the same way across all providers",
            "Ollama uses format parameter, OpenAI uses response_format",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // nxusKit: Local provider for development - no API key needed
    let provider = OllamaProvider::builder()
        .base_url("http://localhost:11434")
        .build()?;

    // Step: Preparing log entries
    if config.step_pause(
        "Preparing sample log entries for classification...",
        &[
            "Each log entry will be classified into structured fields",
            "The LLM extracts: severity, category, summary, and actionable flag",
            "serde handles type-safe deserialization from JSON",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let log_entries = [
        "2024-01-15 10:23:45 ERROR Failed login attempt for user admin from IP 192.168.1.100 after 5 retries",
        "2024-01-15 10:24:12 INFO User john.doe successfully authenticated",
        "2024-01-15 10:25:33 CRITICAL Database connection pool exhausted, all connections in use",
    ];

    for (i, log_entry) in log_entries.iter().enumerate() {
        // Step: Processing each log entry
        if config.step_pause(
            &format!("Processing log entry {}...", i + 1),
            &[
                "nxusKit: classify_log uses JSON mode to get typed LogClassification struct",
                "The schema is enforced by the provider's JSON mode",
                "Invalid JSON responses are handled gracefully",
            ],
        ) == StepAction::Quit
        {
            return Ok(());
        }

        println!("--- Log Entry {} ---", i + 1);
        println!("Input: {}\n", log_entry);

        // Verbose: Show what we're sending (simplified since classify_log builds the request internally)
        if config.verbose {
            println!("[VERBOSE] Classifying log entry with JSON mode enabled");
            println!("[VERBOSE] Model: llama3");
            println!();
        }

        // nxusKit: classify_log uses JSON mode to get typed LogClassification struct
        match classify_log(&provider, "llama3", log_entry).await {
            Ok(classification) => {
                println!("Classification:");
                println!("  Severity: {}", classification.severity);
                println!("  Category: {}", classification.category);
                println!("  Summary: {}", classification.summary);
                println!("  Actionable: {}", classification.actionable);
            }
            Err(e) => {
                println!("Error: {:?}", e);
            }
        }
        println!();
    }

    Ok(())
}
