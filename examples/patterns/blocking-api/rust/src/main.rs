//! Example: Synchronous (Blocking) API
//!
//! ## nxusKit Features Demonstrated
//! - BlockingProvider for synchronous chat without an async runtime
//! - NxuskitProvider for direct synchronous calls via the C ABI
//! - LoopbackProvider for request inspection
//! - Thread-safe provider access
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show raw request/response data
//! - `--step` or `-s`: Pause at each API call with explanations
//!
//! ## Why This Pattern Matters
//! Not all code can be async — immediate-mode UIs, simple scripts, and legacy
//! code often need synchronous APIs. nxusKit provides two synchronous options:
//! - `BlockingProvider`: owns an isolated single-threaded tokio runtime
//! - `NxuskitProvider`: direct C ABI calls, no async runtime at all
//!
//! Usage:
//! ```bash
//! cargo run
//! cargo run -- --verbose    # Show request/response details
//! cargo run -- --step       # Step through with explanations
//! ```

fn main() -> Result<(), Box<dyn std::error::Error>> {
    use nxuskit::builders::LoopbackProvider;
    use nxuskit::{BlockingProvider, ChatRequest, Message, ProviderConfig};
    use nxuskit_examples_interactive::{InteractiveConfig, StepAction};

    // Parse interactive mode flags
    let mut config = InteractiveConfig::from_args();

    println!("=== Blocking API Usage Example ===\n");
    println!("Demonstrates synchronous LLM calls without an async runtime.\n");

    // Step: Creating BlockingProvider
    if config.step_pause(
        "Example 1: BlockingProvider with mock backend...",
        &[
            "nxusKit: BlockingProvider wraps any provider for synchronous access",
            "It owns an isolated single-threaded tokio runtime internally",
            "Your code stays fully synchronous — no async/await needed",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Example 1: BlockingProvider — synchronous chat
    println!("1. Synchronous chat with BlockingProvider:");
    println!("{}", "-".repeat(40));

    // nxusKit: BlockingProvider makes any provider synchronous
    let provider = BlockingProvider::new(ProviderConfig {
        provider_type: "mock".to_string(),
        ..Default::default()
    })?;

    let request =
        ChatRequest::new("mock-model").with_message(Message::user("Tell me something interesting"));

    config.print_request("BLOCKING", "mock-provider", &request);

    let start = std::time::Instant::now();
    // nxusKit: .chat() blocks until complete — no async, no runtime, no .await
    let response = provider.chat(request)?;
    let elapsed_ms = start.elapsed().as_millis() as u64;

    config.print_response(200, elapsed_ms, &response);
    println!("Response: {}\n", response.content);

    // Step: Listing models
    if config.step_pause(
        "Example 2: Listing models synchronously...",
        &[
            "list_models() returns available models — synchronous call",
            "Same API surface whether blocking or async",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Example 2: List models (synchronous)
    println!("2. List models:");
    println!("{}", "-".repeat(40));

    let models = provider.list_models()?;
    println!("Available models ({}):", models.len());
    for model in &models {
        println!("  - {} (id: {})", model.name, model.id);
    }
    println!();

    // Step: LoopbackProvider
    if config.step_pause(
        "Example 3: LoopbackProvider for request inspection...",
        &[
            "LoopbackProvider echoes back request details",
            "Useful for debugging and testing request construction",
            "Already synchronous — .chat() returns directly",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Example 3: LoopbackProvider for request inspection
    println!("3. Request inspection with LoopbackProvider:");
    println!("{}", "-".repeat(40));

    let loopback = LoopbackProvider::builder().build()?;

    let request = ChatRequest::new("u-turn-summary")
        .with_message(Message::user("Inspect this request"))
        .with_temperature(0.7_f32)
        .with_max_tokens(100);

    config.print_request("BLOCKING", "loopback-provider", &request);

    let response = loopback.chat(request)?;
    println!("Request summary:\n{}\n", response.content);

    // Step: Multiple sequential calls
    if config.step_pause(
        "Example 4: Sequential blocking calls (simulating UI)...",
        &[
            "Multiple sequential blocking calls — no async coordination",
            "Perfect for immediate-mode UIs and simple scripts",
            "Each call completes before the next begins",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    // Example 4: Multiple sequential calls
    println!("4. Sequential UI interactions:");
    println!("{}", "-".repeat(40));

    for i in 1..=3 {
        let request = ChatRequest::new("mock-model")
            .with_message(Message::user(format!("Button {} clicked", i)));

        let start = std::time::Instant::now();
        let response = provider.chat(request)?;
        let elapsed = start.elapsed();

        println!(
            "  Interaction {}: \"{}\" ({:?})",
            i, response.content, elapsed
        );
    }

    println!("\n=== Example Complete ===");
    Ok(())
}
