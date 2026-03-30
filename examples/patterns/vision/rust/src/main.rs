//! Example: Vision / Multimodal
//!
//! ## nxusKit Features Demonstrated
//! - Provider builder pattern for Claude and OpenAI
//! - Model discovery with list_models()
//! - Chat requests with different providers
//! - Consistent error handling with NxuskitError
//!
//! ## Note on Vision Support
//! Full vision/multimodal features (image URLs, base64 images, detail levels)
//! are available through the nxuskit crate. This example demonstrates
//! the provider setup and text-based chat through the nxuskit SDK.
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show raw request/response data
//! - `--step` or `-s`: Pause at each step with explanations
//!
//! # Running the example
//!
//! ```bash
//! # With Claude
//! ANTHROPIC_API_KEY=your_key cargo run -- claude
//!
//! # With OpenAI
//! OPENAI_API_KEY=your_key cargo run -- openai
//!
//! # With verbose output
//! cargo run -- claude --verbose
//!
//! # Step-by-step mode
//! cargo run -- openai --step
//! ```

use nxuskit::prelude::*;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
use std::env;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Parse interactive mode flags
    let mut config = InteractiveConfig::from_args();

    // Get provider from command line args (excluding flags)
    let args: Vec<String> = env::args()
        .skip(1)
        .filter(|a| !a.starts_with('-'))
        .collect();
    let provider_name = args.first().map(|s| s.as_str()).unwrap_or("claude");

    println!("Vision Example - Using {} provider\n", provider_name);

    match provider_name {
        "claude" => run_claude_example(&mut config)?,
        "openai" => run_openai_example(&mut config)?,
        _ => {
            eprintln!(
                "Unknown provider: {}. Use 'claude' or 'openai'",
                provider_name
            );
            std::process::exit(1);
        }
    }

    Ok(())
}

fn run_claude_example(config: &mut InteractiveConfig) -> Result<(), Box<dyn std::error::Error>> {
    // Step: Getting API key
    if config.step_pause(
        "Getting API key from environment...",
        &[
            "Reads ANTHROPIC_API_KEY from environment variables",
            "This keeps secrets out of source code",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let api_key = env::var("ANTHROPIC_API_KEY").map_err(|_| NxuskitError::Configuration {
        message: "ANTHROPIC_API_KEY not set".into(),
    })?;

    // Step: Creating provider
    if config.step_pause(
        "Creating Claude provider...",
        &[
            "nxusKit: Type-safe builder ensures all required fields are set",
            "The builder pattern catches configuration errors at compile time",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let provider = ClaudeProvider::builder().api_key(api_key).build()?;

    // Step: Model discovery
    if config.step_pause(
        "Discovering available models...",
        &[
            "nxusKit: list_models() returns normalized ModelInfo across providers",
            "ModelInfo includes name, context_window, and size_bytes",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    println!("Discovering available models...");
    let models = provider.list_models()?;
    println!("Found {} models:\n", models.len());
    for model in models.iter().take(5) {
        println!("  - {}", model.name);
        if let Some(ctx) = model.context_window {
            println!("    Context window: {} tokens", ctx);
        }
    }
    if models.len() > 5 {
        println!("  ... and {} more", models.len() - 5);
    }
    println!();

    // Step: Text chat request
    if config.step_pause(
        "Sending a text chat request...",
        &[
            "nxusKit: Same ChatRequest pattern works for all providers",
            "The SDK routes the request to the correct provider API",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    println!("Example: Text chat with Claude");
    println!("{}", "-".repeat(35));

    let request = ChatRequest::new("claude-haiku-4-5-20251001")
        .with_message(Message::user(
            "Describe what a vision-capable AI model can do with images. Keep it brief.",
        ))
        .with_max_tokens(300);

    // Verbose: Show the request
    config.print_request("POST", "https://api.anthropic.com/v1/messages", &request);

    match provider.chat(request) {
        Ok(response) => {
            config.print_response(200, 0, &response);
            println!("Response: {}\n", response.content);
            println!(
                "Token usage: {} input, {} output\n",
                response.usage.estimated.prompt_tokens, response.usage.estimated.completion_tokens
            );
        }
        Err(e) => eprintln!("Error: {}\n", e),
    }

    println!("Note: Multimodal image input is available in the Go and Python SDKs.");
    println!("See the Go/Python variants of this example for image-sending demos.");
    println!("The Rust SDK's Message type currently supports text content only.\n");

    Ok(())
}

fn run_openai_example(config: &mut InteractiveConfig) -> Result<(), Box<dyn std::error::Error>> {
    // Step: Getting API key
    if config.step_pause(
        "Getting API key from environment...",
        &[
            "Reads OPENAI_API_KEY from environment variables",
            "This keeps secrets out of source code",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let api_key = env::var("OPENAI_API_KEY").map_err(|_| NxuskitError::Configuration {
        message: "OPENAI_API_KEY not set".into(),
    })?;

    // Step: Creating provider
    if config.step_pause(
        "Creating OpenAI provider...",
        &[
            "nxusKit: Type-safe builder ensures all required fields are set",
            "The builder pattern catches configuration errors at compile time",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    let provider = OpenAIProvider::builder().api_key(api_key).build()?;

    // Step: Model discovery
    if config.step_pause(
        "Discovering available models...",
        &[
            "nxusKit: list_models() returns normalized ModelInfo across providers",
            "OpenAI models expose metadata through the same API",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    println!("Discovering available models...");
    let models = provider.list_models()?;
    println!("Found {} models:\n", models.len());
    for model in models.iter().take(5) {
        println!("  - {}", model.name);
        if let Some(ctx) = model.context_window {
            println!("    Context window: {} tokens", ctx);
        }
    }
    if models.len() > 5 {
        println!("  ... and {} more", models.len() - 5);
    }
    println!();

    // Step: Text chat request
    if config.step_pause(
        "Sending a text chat request...",
        &[
            "nxusKit: Same ChatRequest pattern works for all providers",
            "The SDK routes the request to the correct provider API",
        ],
    ) == StepAction::Quit
    {
        return Ok(());
    }

    println!("Example: Text chat with OpenAI");
    println!("{}", "-".repeat(35));

    let request = ChatRequest::new("gpt-4o-mini")
        .with_message(Message::user(
            "Describe what a vision-capable AI model can do with images. Keep it brief.",
        ))
        .with_max_tokens(300);

    // Verbose: Show the request
    config.print_request(
        "POST",
        "https://api.openai.com/v1/chat/completions",
        &request,
    );

    match provider.chat(request) {
        Ok(response) => {
            config.print_response(200, 0, &response);
            println!("Response: {}\n", response.content);
            println!(
                "Token usage: {} input, {} output\n",
                response.usage.estimated.prompt_tokens, response.usage.estimated.completion_tokens
            );
        }
        Err(e) => eprintln!("Error: {}\n", e),
    }

    println!("Note: Multimodal image input is available in the Go and Python SDKs.");
    println!("See the Go/Python variants of this example for image-sending demos.");
    println!("The Rust SDK's Message type currently supports text content only.\n");

    Ok(())
}
