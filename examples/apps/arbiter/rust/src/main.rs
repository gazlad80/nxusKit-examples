//! Arbiter Example: Auto-Retry LLM with CLIPS Validation
//!
//! ## nxusKit Features Demonstrated
//! - ClipsProvider for deterministic output validation
//! - Auto-retry loop with CLIPS-driven parameter adjustment
//! - JSON-based fact assertion for LLM output evaluation
//! - Provider-agnostic solver pattern (works with any AsyncProvider)
//!
//! ## Why This Pattern Matters
//! LLM outputs can be inconsistent. Using CLIPS rules to validate outputs
//! enables automatic retries with intelligent parameter adjustments.
//! This ensures consistent, policy-compliant results.
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show detailed retry information and LLM request/response data
//! - `--step` or `-s`: Pause at each major operation for step-by-step learning
//!
//! Run with: cargo run --example arbiter -- --help

use llm_patterns::arbiter::{
    ConclusionType, EvalStatus, EvaluationResult, FailureType, RetryAttempt, SolverConfig,
    SolverResult, apply_adjustments, default_strategies, find_strategy_for_failure,
    load_config_from_file, parse_evaluation_result, score_attempt, validate_config,
};
use nxuskit::ClipsSession;
use nxuskit::clips::ClipsValue;
use nxuskit::prelude::*;
use nxuskit_examples_interactive::{InteractiveConfig, StepAction};
use std::collections::HashMap;
use std::env;
use std::path::Path;
use std::time::Instant;

fn main() {
    // Initialize interactive mode from CLI args (--verbose/-v and --step/-s)
    let mut interactive = InteractiveConfig::from_args();
    let args: Vec<String> = env::args().collect();

    // Parse command line arguments
    let mut config_path: Option<String> = None;
    let mut input: Option<String> = None;
    let mut conclusion_type = ConclusionType::Classification;
    let mut categories: Vec<String> =
        vec!["high".to_string(), "medium".to_string(), "low".to_string()];
    let mut max_retries: u32 = 3;

    let mut i = 1;
    while i < args.len() {
        match args[i].as_str() {
            "--help" | "-h" => {
                print_help();
                return;
            }
            "--config" | "-c" => {
                i += 1;
                if i < args.len() {
                    config_path = Some(args[i].clone());
                }
            }
            "--input" | "-i" => {
                i += 1;
                if i < args.len() {
                    input = Some(args[i].clone());
                }
            }
            "--type" | "-t" => {
                i += 1;
                if i < args.len() {
                    conclusion_type = match args[i].as_str() {
                        "classification" => ConclusionType::Classification,
                        "extraction" => ConclusionType::Extraction,
                        "reasoning" => ConclusionType::Reasoning,
                        _ => {
                            eprintln!("Unknown type: {}. Using classification.", args[i]);
                            ConclusionType::Classification
                        }
                    };
                }
            }
            "--categories" => {
                i += 1;
                if i < args.len() {
                    categories = args[i].split(',').map(|s| s.trim().to_string()).collect();
                }
            }
            "--max-retries" => {
                i += 1;
                if i < args.len() {
                    max_retries = args[i].parse().unwrap_or(3);
                }
            }
            // --verbose and --step are handled by InteractiveConfig::from_args()
            "--verbose" | "-v" | "--step" | "-s" => {}
            _ => {
                // Treat as input if not a flag
                if !args[i].starts_with('-') && input.is_none() {
                    input = Some(args[i].clone());
                }
            }
        }
        i += 1;
    }

    // nxusKit: ClipsProvider enables rule-based validation of LLM outputs
    // The solver pattern uses CLIPS rules to evaluate output quality and
    // determine if retries with adjusted parameters are needed.
    eprintln!("Solver Pattern: Using CLIPS rules for LLM output validation.\n");

    // Step mode: explain configuration loading
    if interactive.step_pause(
        "Loading solver configuration...",
        &[
            "Configuration defines retry strategies and CLIPS rules",
            "Strategies map failure types to parameter adjustments",
            "CLIPS rules evaluate LLM output quality",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    // Load or create configuration
    let config = if let Some(path) = config_path {
        match load_config_from_file(&path) {
            Ok(c) => c,
            Err(e) => {
                eprintln!("Error loading config: {}", e);
                std::process::exit(1);
            }
        }
    } else {
        SolverConfig {
            max_retries,
            strategies: default_strategies(),
            evaluation_rules: "examples/rules/solver/classification-eval.clp".to_string(),
            conclusion_type,
            confidence_threshold: 0.7,
            timeout_ms: 30000,
            valid_categories: categories.clone(),
        }
    };

    // Validate configuration
    if let Err(e) = validate_config(&config) {
        eprintln!("Invalid configuration: {}", e);
        std::process::exit(1);
    }

    // Get input text
    let input_text = match input {
        Some(text) => text,
        None => {
            eprintln!("Error: No input provided. Use --input or provide text as argument.");
            print_help();
            std::process::exit(1);
        }
    };

    println!("Solver: {:?} Mode", config.conclusion_type);
    println!("Input: \"{}\"", truncate(&input_text, 60));
    if matches!(config.conclusion_type, ConclusionType::Classification) {
        println!("Valid categories: {:?}", config.valid_categories);
    }
    println!();

    // Step mode: explain solver loop
    if interactive.step_pause(
        "Starting solver loop...",
        &[
            "Will generate LLM responses and validate with CLIPS",
            "Retries with parameter adjustments on validation failure",
            "Loop terminates on valid output or max retries",
        ],
    ) == StepAction::Quit
    {
        return;
    }

    // nxusKit: Run the solver loop with CLIPS validation
    // Uses ClipsProvider for output validation and AsyncProvider for generation.
    let result = run_solver_loop(&config, &input_text, &mut interactive);

    // Print results
    print_result(&result);

    if !result.success {
        std::process::exit(1);
    }
}

fn print_help() {
    println!("Arbiter Example: Auto-Retry LLM with CLIPS Validation");
    println!();
    println!("USAGE:");
    println!("    cargo run --example arbiter -- [OPTIONS] [INPUT]");
    println!();
    println!("OPTIONS:");
    println!("    -c, --config <FILE>     Path to solver-config.json");
    println!("    -i, --input <TEXT>      Input text to classify/extract/reason");
    println!("    -t, --type <TYPE>       Conclusion type: classification, extraction, reasoning");
    println!("        --categories <LIST> Comma-separated valid categories");
    println!("        --max-retries <N>   Maximum retry attempts (default: 3)");
    println!("    -v, --verbose           Show detailed retry information and LLM data");
    println!("    -s, --step              Pause at each major operation for learning");
    println!("    -h, --help              Show this help message");
    println!();
    println!("INTERACTIVE MODES:");
    println!("    --verbose shows raw LLM request/response data for debugging");
    println!("    --step pauses before each major operation with explanations");
    println!("    Both can be combined: --verbose --step");
    println!();
    println!("EXAMPLES:");
    println!("    cargo run --example arbiter -- --type classification \\");
    println!("        --categories \"high,medium,low\" \\");
    println!("        --input \"My account is hacked!\"");
    println!();
    println!("    cargo run --example arbiter -- --config solver-config.json \\");
    println!("        --input \"Please reset my password\" --step");
}

/// Runs the solver loop with CLIPS validation.
///
/// ## nxusKit Features Demonstrated
/// - CLIPS rules evaluate LLM output quality (confidence, category, reasoning)
/// - Failure strategies drive automatic parameter adjustments
/// - Loop terminates on valid output or max retries
fn run_solver_loop(
    config: &SolverConfig,
    input: &str,
    interactive: &mut InteractiveConfig,
) -> SolverResult {
    // Resolve CLIPS rules path relative to the binary or working directory
    let rules_path = resolve_rules_path(&config.evaluation_rules);
    let start_time = Instant::now();
    let mut retry_history = Vec::new();
    let mut current_params: HashMap<String, serde_json::Value> = HashMap::new();

    // Initialize default parameters
    current_params.insert("temperature".to_string(), serde_json::json!(0.7));
    current_params.insert("max_tokens".to_string(), serde_json::json!(1000));
    current_params.insert("thinking_enabled".to_string(), serde_json::json!(0.0));

    let strategies = if config.strategies.is_empty() {
        default_strategies()
    } else {
        config.strategies.clone()
    };

    for attempt_num in 1..=config.max_retries {
        let attempt_start = Instant::now();

        // Step mode: explain each retry attempt
        if interactive.step_pause(
            &format!("Retry attempt {} of {}...", attempt_num, config.max_retries),
            &[
                &format!(
                    "Temperature: {:.1}",
                    current_params
                        .get("temperature")
                        .and_then(|v| v.as_f64())
                        .unwrap_or(0.7)
                ),
                "Generating LLM response with current parameters",
                "Will validate output with CLIPS rules",
            ],
        ) == StepAction::Quit
        {
            break;
        }

        if interactive.is_verbose() {
            println!("Attempt {}:", attempt_num);
            println!(
                "  Parameters: temperature={:.1}",
                current_params
                    .get("temperature")
                    .and_then(|v| v.as_f64())
                    .unwrap_or(0.7)
            );
        }

        // Verbose mode: show LLM request
        let request = serde_json::json!({
            "input": truncate(input, 100),
            "parameters": current_params,
            "attempt": attempt_num
        });
        interactive.print_request("POST", "llm://provider/chat", &request);

        // nxusKit: Generate LLM response using provider.chat()
        let llm_response = match generate_llm_response(input, config, &current_params) {
            Ok(r) => r,
            Err(e) => {
                eprintln!("  LLM error: {}", e);
                retry_history.push(RetryAttempt {
                    attempt_number: attempt_num,
                    parameters: current_params.clone(),
                    llm_response: String::new(),
                    evaluation: EvaluationResult {
                        status: EvalStatus::Invalid,
                        failure_type: Some(FailureType::ParseError),
                        suggested_adjustment: None,
                        confidence: None,
                        details: HashMap::new(),
                    },
                    duration_ms: attempt_start.elapsed().as_millis() as u64,
                    tokens_used: 0,
                });
                continue;
            }
        };

        // nxusKit: Evaluate LLM output with CLIPS rules
        let eval_json = match evaluate_with_clips(&llm_response, config, &rules_path) {
            Ok(e) => e,
            Err(e) => {
                eprintln!("  CLIPS evaluation error: {}", e);
                // Treat CLIPS errors as parse errors for retry logic
                r#"{"status":"retry","failure_type":"parse_error","suggested_adjustment":"","confidence":0.0}"#.to_string()
            }
        };

        // Verbose mode: show LLM response
        let response_data = serde_json::json!({
            "response": llm_response,
            "evaluation": eval_json
        });
        interactive.print_response(
            200,
            attempt_start.elapsed().as_millis() as u64,
            &response_data,
        );

        if interactive.is_verbose() {
            println!("  LLM Response: {}", truncate(&llm_response, 80));
        }

        // Parse evaluation result
        let evaluation = match parse_evaluation_result(&eval_json) {
            Ok(e) => e,
            Err(err) => {
                eprintln!("  Evaluation parse error: {}", err);
                continue;
            }
        };

        if interactive.is_verbose() {
            println!("  Evaluation: {:?}", evaluation.status);
            if let Some(ref ft) = evaluation.failure_type {
                println!("  Failure: {:?}", ft);
            }
        }

        let attempt = RetryAttempt {
            attempt_number: attempt_num,
            parameters: current_params.clone(),
            llm_response: llm_response.clone(),
            evaluation: evaluation.clone(),
            duration_ms: attempt_start.elapsed().as_millis() as u64,
            tokens_used: 150 + (attempt_num as u64 * 50), // Approximate; real counts require provider response metadata
        };

        retry_history.push(attempt);

        // Check if valid
        if evaluation.status == EvalStatus::Valid {
            // Success!
            let best = retry_history.last().unwrap().clone();
            let total_tokens: u64 = retry_history.iter().map(|a| a.tokens_used).sum();

            return SolverResult {
                success: true,
                final_output: serde_json::from_str(&strip_markdown_fences(&llm_response))
                    .unwrap_or(serde_json::json!({})),
                best_attempt: best,
                retry_history,
                total_duration_ms: start_time.elapsed().as_millis() as u64,
                total_tokens,
            };
        }

        // Apply adjustments for retry
        if let Some(failure_type) = evaluation.failure_type
            && let Some(strategy) = find_strategy_for_failure(&strategies, failure_type)
        {
            apply_adjustments(&mut current_params, strategy);
            if interactive.is_verbose() {
                println!(
                    "  Adjustment: {:?}",
                    strategy
                        .adjustments
                        .iter()
                        .map(|a| &a.knob)
                        .collect::<Vec<_>>()
                );
            }
        }

        if interactive.is_verbose() {
            println!();
        }
    }

    // Max retries exceeded - return best attempt
    let best = retry_history
        .iter()
        .max_by(|a, b| {
            score_attempt(a)
                .partial_cmp(&score_attempt(b))
                .unwrap_or(std::cmp::Ordering::Equal)
        })
        .cloned()
        .unwrap_or_else(|| retry_history.last().unwrap().clone());

    let total_tokens: u64 = retry_history.iter().map(|a| a.tokens_used).sum();

    SolverResult {
        success: false,
        final_output: serde_json::from_str(&strip_markdown_fences(&best.llm_response))
            .unwrap_or(serde_json::json!({})),
        best_attempt: best,
        retry_history,
        total_duration_ms: start_time.elapsed().as_millis() as u64,
        total_tokens,
    }
}

/// Generates an LLM classification response for the given input.
///
/// Uses Ollama by default (no API key required). Set ANTHROPIC_API_KEY
/// to use Claude instead.
fn generate_llm_response(
    input: &str,
    config: &SolverConfig,
    params: &HashMap<String, serde_json::Value>,
) -> Result<String, String> {
    let temperature = params
        .get("temperature")
        .and_then(|v| v.as_f64())
        .unwrap_or(0.7) as f32;
    let max_tokens = params
        .get("max_tokens")
        .and_then(|v| v.as_u64())
        .unwrap_or(1000) as u32;

    let system_prompt = match config.conclusion_type {
        ConclusionType::Classification => format!(
            "You are a classification system. Classify the following input into one of these categories: {}. \
             Respond with ONLY a JSON object containing: category (string), confidence (float 0-1), reasoning (string).",
            config.valid_categories.join(", ")
        ),
        ConclusionType::Extraction => "You are an extraction system. Extract structured fields from the input. \
             Respond with ONLY a JSON object containing the extracted fields.".to_string(),
        ConclusionType::Reasoning => "You are a reasoning system. Analyze the input using chain-of-thought. \
             Respond with ONLY a JSON object containing: conclusion (string), confidence (float 0-1), reasoning (string).".to_string(),
    };

    // Use Claude if ANTHROPIC_API_KEY is set, otherwise Ollama
    if let Ok(api_key) = env::var("ANTHROPIC_API_KEY")
        && !api_key.is_empty()
    {
        let provider = ClaudeProvider::builder()
            .api_key(api_key)
            .build()
            .map_err(|e| format!("Claude provider error: {e}"))?;

        let request = ChatRequest::new("claude-haiku-4-5-20251001")
            .with_message(Message::system(&system_prompt))
            .with_message(Message::user(input))
            .with_temperature(temperature)
            .with_max_tokens(max_tokens);

        let response = provider
            .chat(request)
            .map_err(|e| format!("LLM chat error: {e}"))?;
        Ok(response.content)
    } else {
        let provider = OllamaProvider::builder()
            .build()
            .map_err(|e| format!("Ollama provider error: {e}"))?;

        let request = ChatRequest::new("llama3")
            .with_message(Message::system(&system_prompt))
            .with_message(Message::user(input))
            .with_temperature(temperature)
            .with_max_tokens(max_tokens);

        let response = provider
            .chat(request)
            .map_err(|e| format!("LLM chat error: {e}"))?;
        Ok(response.content)
    }
}

/// Evaluates an LLM response using CLIPS rules.
///
/// Loads the classification-eval.clp rules, asserts the LLM output and
/// evaluation config as facts, runs the rules engine, and extracts the
/// evaluation result as JSON.
fn evaluate_with_clips(
    llm_response: &str,
    config: &SolverConfig,
    rules_path: &Path,
) -> Result<String, String> {
    let clips = ClipsSession::create().map_err(|e| format!("CLIPS init error: {e}"))?;
    clips
        .load_file(rules_path.to_str().unwrap_or(""))
        .map_err(|e| format!("CLIPS load error: {e}"))?;
    clips
        .reset()
        .map_err(|e| format!("CLIPS reset error: {e}"))?;

    // Strip markdown code fences if present
    let cleaned = strip_markdown_fences(llm_response);

    // Parse LLM response to extract classification fields
    let parsed: serde_json::Value = serde_json::from_str(&cleaned).map_err(|_| {
        r#"{"status":"retry","failure_type":"parse_error","suggested_adjustment":"","confidence":0.0}"#
            .to_string()
    })?;

    let category = parsed
        .get("category")
        .and_then(|v| v.as_str())
        .unwrap_or("unknown");
    let confidence = parsed
        .get("confidence")
        .and_then(|v| v.as_f64())
        .unwrap_or(0.0);
    let reasoning = parsed
        .get("reasoning")
        .and_then(|v| v.as_str())
        .unwrap_or("");

    // Assert classification-output fact
    let mut output_slots = HashMap::new();
    output_slots.insert("category".into(), ClipsValue::Symbol(category.into()));
    output_slots.insert("confidence".into(), ClipsValue::Float(confidence));
    output_slots.insert("reasoning".into(), ClipsValue::String(reasoning.into()));
    output_slots.insert(
        "raw-response".into(),
        ClipsValue::String(llm_response.into()),
    );
    clips
        .fact_assert_structured("classification-output", &output_slots)
        .map_err(|e| format!("CLIPS assert error: {e}"))?;

    // Assert eval-config fact using CLIPS syntax (multislot needs assert_string)
    let categories_symbols = config
        .valid_categories
        .iter()
        .map(|c| c.as_str())
        .collect::<Vec<_>>()
        .join(" ");
    let eval_config_fact = format!(
        "(eval-config (confidence-threshold {}) (valid-categories {}) (require-reasoning 1))",
        config.confidence_threshold, categories_symbols
    );
    clips
        .fact_assert_string(&eval_config_fact)
        .map_err(|e| format!("CLIPS config assert error: {e}"))?;

    // Run rules
    clips
        .run(None)
        .map_err(|e| format!("CLIPS run error: {e}"))?;

    // Extract evaluation-result fact
    let facts = clips
        .facts_by_template("evaluation-result")
        .map_err(|e| format!("CLIPS facts error: {e}"))?;

    if let Some(&fact_idx) = facts.first() {
        let slots = clips
            .fact_slot_values(fact_idx)
            .map_err(|e| format!("CLIPS slot error: {e}"))?;

        let status = clips_value_as_str(slots.get("status"), "invalid");
        let failure_type = clips_value_as_str(slots.get("failure-type"), "parse_error");
        let suggested = clips_value_as_str(slots.get("suggested-adjustment"), "");
        let conf = slots
            .get("extracted-confidence")
            .and_then(|v| v.as_float().ok())
            .unwrap_or(0.0);

        Ok(format!(
            r#"{{"status":"{status}","failure_type":"{failure_type}","suggested_adjustment":"{suggested}","confidence":{conf}}}"#
        ))
    } else {
        Ok(
            r#"{"status":"invalid","failure_type":"parse_error","suggested_adjustment":"","confidence":0.0}"#
                .to_string(),
        )
    }
}

/// Strip markdown code fences (```json ... ```) from LLM output.
fn strip_markdown_fences(s: &str) -> String {
    let trimmed = s.trim();
    if let Some(rest) = trimmed.strip_prefix("```") {
        // Skip optional language tag on the first line
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

/// Extract a string from a ClipsValue (Symbol or String variant).
fn clips_value_as_str(value: Option<&ClipsValue>, default: &str) -> String {
    match value {
        Some(ClipsValue::Symbol(s)) | Some(ClipsValue::String(s)) => s.clone(),
        _ => default.to_string(),
    }
}

fn print_result(result: &SolverResult) {
    let status = if result.success {
        "SUCCESS"
    } else {
        "FAILED (max retries)"
    };

    println!("Result: {}", status);

    if let Some(category) = result.final_output.get("category") {
        println!("  Category: {}", category);
    }
    if let Some(conf) = result.final_output.get("confidence") {
        println!("  Confidence: {}", conf);
    }
    if let Some(reasoning) = result.final_output.get("reasoning")
        && let Some(r) = reasoning.as_str()
        && !r.is_empty()
    {
        println!("  Reasoning: {}", truncate(r, 60));
    }

    println!("  Total attempts: {}", result.retry_history.len());
    println!("  Total time: {}ms", result.total_duration_ms);
    println!("  Total tokens: {}", result.total_tokens);
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}

/// Resolves a CLIPS rules path, searching common locations relative to the binary.
fn resolve_rules_path(configured_path: &str) -> std::path::PathBuf {
    let candidates = [
        std::path::PathBuf::from(configured_path),
        std::path::PathBuf::from("../shared/rules/classification-eval.clp"),
        std::path::PathBuf::from("shared/rules/classification-eval.clp"),
        std::path::PathBuf::from("examples/apps/arbiter/shared/rules/classification-eval.clp"),
    ];
    candidates
        .into_iter()
        .find(|p| p.exists())
        .unwrap_or_else(|| std::path::PathBuf::from(configured_path))
}
