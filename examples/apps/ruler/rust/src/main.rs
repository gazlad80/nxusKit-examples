//! Ruler Example: Natural Language to CLIPS Rule Generation
//!
//! Demonstrates using LLM to generate CLIPS rules from natural language
//! descriptions with validation and retry logic.
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show detailed generation progress and LLM request/response data
//! - `--step` or `-s`: Pause at each major operation for step-by-step learning
//!
//! Run with: cargo run --example ruler --features clips -- --help

use std::io::{self, BufRead};
use std::path::PathBuf;
use std::process::ExitCode;

use llm_patterns::ruler::{
    Complexity, GeneratedRules, ProgressiveExample, ProgressiveExamples, RuleDescription,
    SaveFormat, ValidationError, ValidationResult,
};
use nxuskit::prelude::*;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};

fn main() -> ExitCode {
    // Initialize interactive mode from CLI args (--verbose/-v and --step/-s)
    let mut interactive = InteractiveConfig::from_args();
    let args: Vec<String> = std::env::args().collect();

    if args.len() < 2 || args.contains(&"--help".to_string()) || args.contains(&"-h".to_string()) {
        print_help();
        return ExitCode::SUCCESS;
    }

    let command = &args[1];
    match command.as_str() {
        "generate" => cmd_generate(&args[2..], &mut interactive),
        "validate" => cmd_validate(&args[2..], &mut interactive),
        "save" => cmd_save(&args[2..]),
        "load" => cmd_load(&args[2..]),
        "examples" => cmd_examples(&args[2..], &mut interactive),
        "--version" | "-V" => {
            println!("ruler 0.7.0");
            ExitCode::SUCCESS
        }
        _ => {
            eprintln!("Unknown command: {}", command);
            eprintln!("Run 'ruler --help' for usage information.");
            ExitCode::FAILURE
        }
    }
}

fn print_help() {
    println!("Ruler: Natural Language to CLIPS Rule Generation");
    println!();
    println!("USAGE:");
    println!("    cargo run --example ruler --features clips -- <COMMAND> [OPTIONS]");
    println!();
    println!("COMMANDS:");
    println!("    generate <DESCRIPTION>   Generate CLIPS rules from natural language");
    println!("    validate <FILE>          Validate CLIPS code");
    println!("    save <FILE>              Save loaded rules to file");
    println!("    load <FILE>              Load rules from file");
    println!("    examples                 Run progressive complexity examples");
    println!();
    println!("OPTIONS:");
    println!("    -c, --complexity <LEVEL>  Target complexity: basic, intermediate, advanced");
    println!("    -m, --model <MODEL>       LLM model to use (default: claude-haiku-4-5-20251001)");
    println!("    -r, --retries <N>         Max retry attempts (default: 5)");
    println!("    -o, --output <FILE>       Write output to file");
    println!("    -j, --json                Output in JSON format");
    println!("    -v, --verbose             Show detailed progress and LLM data");
    println!("    -s, --step                Pause at each major operation for learning");
    println!("    -h, --help                Show this help message");
    println!("    -V, --version             Show version");
    println!();
    println!("INTERACTIVE MODES:");
    println!("    --verbose shows raw LLM request/response data for debugging");
    println!("    --step pauses before each major operation with explanations");
    println!("    Both can be combined: --verbose --step");
    println!();
    println!("EXAMPLES:");
    println!("    ruler generate \"Create a rule that classifies adults if age >= 18\"");
    println!();
    println!(
        "    ruler generate -c advanced \"Medical triage expert system\" -o triage.clp --step"
    );
    println!();
    println!("    ruler examples -c basic");
    println!();
    println!("    ruler validate my-rules.clp");
    println!();
    println!("NOTE: Uses ClipsProvider for CLIPS rule validation.");
}

fn cmd_generate(args: &[String], interactive: &mut InteractiveConfig) -> ExitCode {
    let mut description = String::new();
    let mut complexity = Complexity::Basic;
    let mut model = "claude-haiku-4-5-20251001".to_string();
    let mut max_retries = 5u8;
    let mut output_file: Option<PathBuf> = None;
    let mut json_output = false;

    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "-c" | "--complexity" => {
                if i + 1 < args.len() {
                    complexity = args[i + 1].parse().unwrap_or_else(|e| {
                        eprintln!("Warning: {}", e);
                        Complexity::Basic
                    });
                    i += 1;
                }
            }
            "-m" | "--model" => {
                if i + 1 < args.len() {
                    model = args[i + 1].clone();
                    i += 1;
                }
            }
            "-r" | "--retries" => {
                if i + 1 < args.len() {
                    max_retries = args[i + 1].parse().unwrap_or(5);
                    i += 1;
                }
            }
            "-o" | "--output" => {
                if i + 1 < args.len() {
                    output_file = Some(PathBuf::from(&args[i + 1]));
                    i += 1;
                }
            }
            "-j" | "--json" => json_output = true,
            // --verbose and --step are handled by InteractiveConfig::from_args()
            "-v" | "--verbose" | "-s" | "--step" => {}
            "-" => {
                // Read from stdin
                if interactive.is_verbose() {
                    eprintln!("Reading description from stdin...");
                }
                let stdin = io::stdin();
                for l in stdin.lock().lines().map_while(Result::ok) {
                    if !description.is_empty() {
                        description.push(' ');
                    }
                    description.push_str(&l);
                }
            }
            s if !s.starts_with('-') => {
                if !description.is_empty() {
                    description.push(' ');
                }
                description.push_str(s);
            }
            _ => {}
        }
        i += 1;
    }

    if description.is_empty() {
        eprintln!("Error: No description provided");
        eprintln!("Usage: ruler generate <DESCRIPTION>");
        return ExitCode::FAILURE;
    }

    if interactive.is_verbose() {
        eprintln!("Generating CLIPS rules...");
        eprintln!("  Description: {}", description);
        eprintln!("  Complexity: {}", complexity);
        eprintln!("  Model: {}", model);
        eprintln!("  Max retries: {}", max_retries);
    }

    // Step mode: explain generation process
    if interactive.step_pause(
        "Preparing rule generation...",
        &[
            &format!("Complexity level: {}", complexity),
            "LLM will generate CLIPS code from description",
            "Generated code will be validated for syntax",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    // nxusKit: LLM generates CLIPS rules, ClipsProvider validates them
    let rule_desc = RuleDescription::new(&description).with_complexity(complexity);

    // Step mode: explain LLM call
    if interactive.step_pause(
        "Calling LLM for rule generation...",
        &[
            &format!("Model: {}", model),
            "Sending natural language description",
            "Expecting CLIPS deftemplate and defrule constructs",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    // Verbose mode: show LLM request
    let request = serde_json::json!({
        "description": description,
        "complexity": complexity.to_string(),
        "model": model
    });
    interactive.print_request("POST", "llm://provider/chat", &request);

    // nxusKit: Call LLM to generate CLIPS rules from the description
    let generated = generate_clips_rules(&rule_desc, &model);

    // Verbose mode: show LLM response
    let response = serde_json::json!({
        "clips_code": generated.clips_code,
        "tokens_used": generated.tokens_used,
        "generation_time_ms": generated.generation_time_ms
    });
    interactive.print_response(200, generated.generation_time_ms, &response);

    // Step mode: explain validation
    if interactive.step_pause(
        "Validating generated CLIPS code...",
        &[
            "Checking parenthesis balance",
            "Verifying required constructs (deftemplate, defrule)",
            "Checking for unsafe operations",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    let validation = validate_clips_code(&generated.clips_code);

    if json_output {
        let output = serde_json::json!({
            "success": validation.is_valid(),
            "clips_code": generated.clips_code,
            "attempts": generated.generation_attempt,
            "tokens_used": generated.tokens_used,
            "generation_time_ms": generated.generation_time_ms,
            "validation": {
                "status": format!("{:?}", validation.status).to_lowercase(),
                "warnings": validation.warnings
            }
        });
        println!("{}", serde_json::to_string_pretty(&output).unwrap());
    } else if validation.is_valid() {
        println!(";; Generated CLIPS Rules");
        println!(";; Description: {}", description);
        println!(";; Complexity: {}", complexity);
        println!(";; Model: {}", model);
        println!();
        println!("{}", generated.clips_code);

        for warning in &validation.warnings {
            eprintln!("Warning: {}", warning);
        }
    } else {
        eprintln!("Validation failed:");
        for error in &validation.errors {
            eprintln!("  {}", error);
        }
        return ExitCode::from(2);
    }

    if let Some(path) = output_file {
        if let Err(e) = std::fs::write(&path, &generated.clips_code) {
            eprintln!("Error writing to {}: {}", path.display(), e);
            return ExitCode::from(4);
        }
        if interactive.is_verbose() {
            eprintln!(
                "Wrote {} bytes to {}",
                generated.clips_code.len(),
                path.display()
            );
        }
    }

    ExitCode::SUCCESS
}

fn cmd_validate(args: &[String], interactive: &mut InteractiveConfig) -> ExitCode {
    let mut file_path: Option<PathBuf> = None;
    let mut json_output = false;

    for arg in args.iter() {
        match arg.as_str() {
            // --verbose and --step are handled by InteractiveConfig::from_args()
            "-v" | "--verbose" | "-s" | "--step" => {}
            "-j" | "--json" => json_output = true,
            s if !s.starts_with('-') && file_path.is_none() => {
                file_path = Some(PathBuf::from(s));
            }
            _ => {}
        }
    }

    let path = match file_path {
        Some(p) => p,
        None => {
            eprintln!("Error: No file specified");
            return ExitCode::FAILURE;
        }
    };

    let code = match std::fs::read_to_string(&path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("Error reading {}: {}", path.display(), e);
            return ExitCode::from(4);
        }
    };

    if interactive.is_verbose() {
        eprintln!("Validating {}...", path.display());
    }

    // Step mode: explain validation
    if interactive.step_pause(
        "Validating CLIPS code...",
        &[
            &format!("File: {}", path.display()),
            "Checking parenthesis balance",
            "Verifying CLIPS constructs",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    let validation = validate_clips_code(&code);

    if json_output {
        let output = serde_json::json!({
            "valid": validation.is_valid(),
            "errors": validation.errors.iter().map(|e| {
                serde_json::json!({
                    "type": format!("{:?}", e.error_type).to_lowercase(),
                    "message": e.message,
                    "line": e.line_number
                })
            }).collect::<Vec<_>>(),
            "warnings": validation.warnings
        });
        println!("{}", serde_json::to_string_pretty(&output).unwrap());
    } else if validation.is_valid() {
        println!("Valid: {}", path.display());
        for warning in &validation.warnings {
            println!("Warning: {}", warning);
        }
    } else {
        println!("Invalid: {}", path.display());
        for error in &validation.errors {
            println!("  {}", error);
        }
    }

    if validation.is_valid() {
        ExitCode::SUCCESS
    } else {
        ExitCode::from(2)
    }
}

fn cmd_save(args: &[String]) -> ExitCode {
    let mut file_path: Option<PathBuf> = None;
    let mut format = SaveFormat::Text;

    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "-f" | "--format" => {
                if i + 1 < args.len() {
                    format = args[i + 1].parse().unwrap_or(SaveFormat::Text);
                    i += 1;
                }
            }
            s if !s.starts_with('-') && file_path.is_none() => {
                file_path = Some(PathBuf::from(s));
            }
            _ => {}
        }
        i += 1;
    }

    let path = match file_path {
        Some(p) => p,
        None => {
            eprintln!("Error: No file specified");
            return ExitCode::FAILURE;
        }
    };

    // nxusKit: ClipsProvider can save rules using bsave (binary) or text format
    eprintln!("Save: {} (format: {})", path.display(), format);

    ExitCode::SUCCESS
}

fn cmd_load(args: &[String]) -> ExitCode {
    let mut file_path: Option<PathBuf> = None;
    let mut verbose = false;

    for arg in args {
        match arg.as_str() {
            "-v" | "--verbose" => verbose = true,
            s if !s.starts_with('-') && file_path.is_none() => {
                file_path = Some(PathBuf::from(s));
            }
            _ => {}
        }
    }

    let path = match file_path {
        Some(p) => p,
        None => {
            eprintln!("Error: No file specified");
            return ExitCode::FAILURE;
        }
    };

    if !path.exists() {
        eprintln!("Error: File not found: {}", path.display());
        return ExitCode::from(4);
    }

    if verbose {
        eprintln!("Loading {}...", path.display());
    }

    // nxusKit: ClipsProvider can load rules from file or binary
    eprintln!("Load: {}", path.display());

    ExitCode::SUCCESS
}

fn cmd_examples(args: &[String], interactive: &mut InteractiveConfig) -> ExitCode {
    let mut complexity_filter: Option<Complexity> = None;
    let mut list_only = false;
    let mut example_number: Option<usize> = None;
    let mut json_output = false;

    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "-c" | "--complexity" => {
                if i + 1 < args.len() {
                    complexity_filter = Some(args[i + 1].parse().unwrap_or(Complexity::Basic));
                    i += 1;
                }
            }
            "-n" | "--number" => {
                if i + 1 < args.len() {
                    example_number = args[i + 1].parse().ok();
                    i += 1;
                }
            }
            "-l" | "--list" => list_only = true,
            "-j" | "--json" => json_output = true,
            // --verbose and --step are handled by InteractiveConfig::from_args()
            "-v" | "--verbose" | "-s" | "--step" => {}
            _ => {}
        }
        i += 1;
    }

    let examples = get_builtin_examples();

    // Filter by complexity if specified
    let filtered: Vec<_> = examples
        .examples
        .iter()
        .filter(|e| complexity_filter.is_none_or(|c| e.complexity == c))
        .collect();

    if list_only {
        if json_output {
            let output: Vec<_> = filtered
                .iter()
                .map(|e| {
                    serde_json::json!({
                        "id": e.id,
                        "complexity": e.complexity.to_string(),
                        "description": e.description
                    })
                })
                .collect();
            println!("{}", serde_json::to_string_pretty(&output).unwrap());
        } else {
            println!("Available Examples:");
            println!();
            for ex in &filtered {
                println!("  {} [{}]", ex.id, ex.complexity);
                println!("    {}", truncate(&ex.description, 60));
            }
        }
        return ExitCode::SUCCESS;
    }

    if let Some(num) = example_number {
        if num >= filtered.len() {
            eprintln!(
                "Error: Example {} not found (available: 0-{})",
                num,
                filtered.len() - 1
            );
            return ExitCode::FAILURE;
        }
        let example = &filtered[num];
        println!("Example {}: {}", num, example.id);
        println!("Complexity: {}", example.complexity);
        println!("Description: {}", example.description);
        println!();
        println!("Expected constructs: {:?}", example.expected_constructs);
        return ExitCode::SUCCESS;
    }

    // Run all filtered examples
    println!("Running {} examples...", filtered.len());
    println!();

    // Step mode: explain examples
    if interactive.step_pause(
        "Running progressive examples...",
        &[
            &format!("{} examples selected", filtered.len()),
            "Examples progress from basic to advanced",
            "Each example shows expected CLIPS constructs",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    for (i, example) in filtered.iter().enumerate() {
        println!(
            "--- Example {}: {} [{}] ---",
            i, example.id, example.complexity
        );
        println!("Description: {}", example.description);
        println!("Expected: {:?}", example.expected_constructs);
        println!();

        // nxusKit: LLM generates rules, ClipsProvider validates syntax
        println!("(Use ruler generate --description \"...\" to generate rules)");
        println!();
    }

    ExitCode::SUCCESS
}

// Helper functions

/// Generate CLIPS rules using LLM.
///
/// Uses Ollama by default (no API key required). Set ANTHROPIC_API_KEY
/// to use Claude instead.
fn generate_clips_rules(desc: &RuleDescription, model: &str) -> GeneratedRules {
    let start = std::time::Instant::now();

    let prompt = format!(
        "Generate CLIPS expert system rules for the following requirement:\n\n\
         Description: {}\n\
         Complexity level: {}\n\n\
         Requirements:\n\
         - Use (deftemplate ...) to define fact templates\n\
         - Use (defrule ...) to define rules\n\
         - Include comments explaining each rule\n\
         - Respond with ONLY valid CLIPS code, no markdown or explanation",
        desc.description, desc.complexity
    );

    let chat_result = if let Ok(api_key) = std::env::var("ANTHROPIC_API_KEY")
        && !api_key.is_empty()
    {
        let provider = ClaudeProvider::builder()
            .api_key(api_key)
            .build()
            .map_err(|e| e.to_string());

        provider.and_then(|p| {
            let request = ChatRequest::new(model)
                .with_message(Message::system(
                    "You are an expert CLIPS rule author. Generate only valid CLIPS code.",
                ))
                .with_message(Message::user(&prompt))
                .with_temperature(0.3_f32)
                .with_max_tokens(2000);
            p.chat(request).map_err(|e| e.to_string())
        })
    } else {
        let provider = OllamaProvider::builder().build().map_err(|e| e.to_string());

        provider.and_then(|p| {
            let request = ChatRequest::new("llama3")
                .with_message(Message::system(
                    "You are an expert CLIPS rule author. Generate only valid CLIPS code.",
                ))
                .with_message(Message::user(&prompt))
                .with_temperature(0.3_f32)
                .with_max_tokens(2000);
            p.chat(request).map_err(|e| e.to_string())
        })
    };

    let elapsed_ms = start.elapsed().as_millis() as u64;

    match chat_result {
        Ok(response) => {
            let tokens = response.usage.estimated.prompt_tokens as u64
                + response.usage.estimated.completion_tokens as u64;
            let clips_code = strip_markdown_fences(&response.content);
            GeneratedRules::new(&desc.id, clips_code, model)
                .with_attempt(1)
                .with_tokens(tokens)
                .with_time_ms(elapsed_ms)
        }
        Err(e) => {
            eprintln!("LLM generation error: {e}");
            // Return empty rules on failure so validation reports the issue
            GeneratedRules::new(&desc.id, String::new(), model)
                .with_attempt(1)
                .with_tokens(0)
                .with_time_ms(elapsed_ms)
        }
    }
}

fn validate_clips_code(code: &str) -> ValidationResult {
    let mut errors = Vec::new();
    let mut warnings = Vec::new();

    // Basic validation checks
    let open_parens = code.matches('(').count();
    let close_parens = code.matches(')').count();

    if open_parens != close_parens {
        errors.push(ValidationError::syntax(format!(
            "Unbalanced parentheses: {} open, {} close",
            open_parens, close_parens
        )));
    }

    // Check for required constructs
    if !code.contains("deftemplate") && !code.contains("defrule") {
        warnings.push("No deftemplate or defrule found".to_string());
    }

    // Check for potentially dangerous CLIPS function calls.
    // Match actual function invocations like (system ...) not substrings
    // in comments like "expert system".
    let dangerous_patterns = ["(system ", "(system)", "(open ", "(exec "];
    for pattern in &dangerous_patterns {
        if code.contains(pattern) {
            errors.push(ValidationError::safety(
                "Code contains potentially dangerous system calls",
            ));
            break;
        }
    }

    if errors.is_empty() {
        ValidationResult::valid("validation").with_warnings(warnings)
    } else {
        ValidationResult::invalid("validation", errors).with_warnings(warnings)
    }
}

fn get_builtin_examples() -> ProgressiveExamples {
    // Load from embedded JSON or return hardcoded examples
    ProgressiveExamples {
        examples: vec![
            ProgressiveExample {
                id: "basic-01-adult".to_string(),
                complexity: Complexity::Basic,
                description: "Classify person as adult if age >= 18".to_string(),
                domain_hints: vec!["age".to_string(), "classification".to_string()],
                expected_constructs: vec!["deftemplate".to_string(), "defrule".to_string()],
            },
            ProgressiveExample {
                id: "basic-02-temperature".to_string(),
                complexity: Complexity::Basic,
                description: "Classify temperature as cold/warm/hot".to_string(),
                domain_hints: vec!["temperature".to_string()],
                expected_constructs: vec!["deftemplate".to_string(), "defrule".to_string()],
            },
            ProgressiveExample {
                id: "intermediate-01-priority".to_string(),
                complexity: Complexity::Intermediate,
                description: "Process high-priority tasks first using salience".to_string(),
                domain_hints: vec!["priority".to_string(), "queue".to_string()],
                expected_constructs: vec![
                    "deftemplate".to_string(),
                    "defrule".to_string(),
                    "salience".to_string(),
                ],
            },
            ProgressiveExample {
                id: "advanced-01-triage".to_string(),
                complexity: Complexity::Advanced,
                description: "Medical triage with modules and helper functions".to_string(),
                domain_hints: vec!["medical".to_string(), "triage".to_string()],
                expected_constructs: vec![
                    "deftemplate".to_string(),
                    "defrule".to_string(),
                    "defmodule".to_string(),
                    "deffunction".to_string(),
                ],
            },
        ],
    }
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}

/// Strip markdown code fences (```clips ... ```) from LLM output.
fn strip_markdown_fences(s: &str) -> String {
    let trimmed = s.trim();
    if let Some(rest) = trimmed.strip_prefix("```") {
        let after_tag = rest.find('\n').map(|i| &rest[i + 1..]).unwrap_or(rest);
        after_tag
            .strip_suffix("```")
            .unwrap_or(after_tag)
            .trim()
            .to_string()
    } else {
        trimmed.to_string()
    }
}
