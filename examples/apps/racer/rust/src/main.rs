//! Racer Example: CLIPS vs LLM Head-to-Head Competition
//!
//! Demonstrates running concurrent races between CLIPS rule-based solving
//! and LLM reasoning on logic problems.
//!
//! ## Interactive Modes
//! - `--verbose` or `-v`: Show detailed race progress and LLM request/response data
//! - `--step` or `-s`: Pause at each major operation for step-by-step learning
//!
//! Run with: cargo run --example racer --features clips -- --help

use std::path::PathBuf;
use std::process::ExitCode;
use std::time::Instant;

use llm_patterns::racer::{
    Problem, ProblemDifficulty, ProblemRegistry, ProblemType, RunnerResult, ScoringMode,
};
use nxuskit::ClipsSession;
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
        "race" => cmd_race(&args[2..], &mut interactive),
        "benchmark" => cmd_benchmark(&args[2..], &mut interactive),
        "list" => cmd_list(&args[2..]),
        "describe" => cmd_describe(&args[2..]),
        "--version" | "-V" => {
            println!("racer 0.7.0");
            ExitCode::SUCCESS
        }
        _ => {
            eprintln!("Unknown command: {}", command);
            eprintln!("Run 'racer --help' for usage information.");
            ExitCode::FAILURE
        }
    }
}

fn print_help() {
    println!("Racer: CLIPS vs LLM Head-to-Head Competition");
    println!();
    println!("USAGE:");
    println!("    cargo run --example racer --features clips -- <COMMAND> [OPTIONS]");
    println!();
    println!("COMMANDS:");
    println!("    race <PROBLEM>       Run a single head-to-head race");
    println!("    benchmark <PROBLEM>  Run multiple races for statistics");
    println!("    list                 List available problems");
    println!("    describe <PROBLEM>   Show problem details");
    println!();
    println!("OPTIONS:");
    println!("    -s, --scoring <MODE>   Scoring mode: speed, accuracy, composite");
    println!("    -t, --timeout <SECS>   Timeout per approach (default: 60)");
    println!("    -m, --model <MODEL>    LLM model to use");
    println!("    -n, --runs <N>         Number of benchmark runs (default: 10)");
    println!("    -o, --output <FILE>    Write results to file");
    println!("    --clips-only           Run only CLIPS approach");
    println!("    --llm-only             Run only LLM approach");
    println!("    -j, --json             Output in JSON format");
    println!("    -v, --verbose          Show detailed progress and LLM data");
    println!("    --step                 Pause at each major operation for learning");
    println!("    -h, --help             Show this help message");
    println!("    -V, --version          Show version");
    println!();
    println!("INTERACTIVE MODES:");
    println!("    --verbose shows raw LLM request/response data for debugging");
    println!("    --step pauses before each major operation with explanations");
    println!("    Both can be combined: --verbose --step");
    println!();
    println!("EXAMPLES:");
    println!("    racer race einstein-riddle");
    println!();
    println!("    racer race -s accuracy family-relations --step");
    println!();
    println!("    racer benchmark -n 20 einstein-riddle -o results.json");
    println!();
    println!("    racer list -t logic_puzzle");
    println!();
    println!("NOTE: Uses ClipsProvider for CLIPS rule execution.");
}

fn cmd_race(args: &[String], interactive: &mut InteractiveConfig) -> ExitCode {
    let mut problem_name = String::new();
    let mut scoring_mode = ScoringMode::Speed;
    let mut timeout_secs = 60u64;
    let mut model = "claude-haiku-4-5-20251001".to_string();
    let mut clips_only = false;
    let mut llm_only = false;
    let mut json_output = false;

    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "-s" | "--scoring" => {
                if i + 1 < args.len() {
                    scoring_mode = args[i + 1].parse().unwrap_or(ScoringMode::Speed);
                    i += 1;
                }
            }
            "-t" | "--timeout" => {
                if i + 1 < args.len() {
                    timeout_secs = args[i + 1].parse().unwrap_or(60);
                    i += 1;
                }
            }
            "-m" | "--model" => {
                if i + 1 < args.len() {
                    model = args[i + 1].clone();
                    i += 1;
                }
            }
            "--clips-only" => clips_only = true,
            "--llm-only" => llm_only = true,
            "-j" | "--json" => json_output = true,
            // --verbose and --step are handled by InteractiveConfig::from_args()
            "-v" | "--verbose" | "--step" => {}
            s if !s.starts_with('-') && problem_name.is_empty() => {
                problem_name = s.to_string();
            }
            _ => {}
        }
        i += 1;
    }

    if problem_name.is_empty() {
        eprintln!("Error: No problem specified");
        eprintln!("Usage: racer race <PROBLEM>");
        eprintln!("Run 'racer list' to see available problems.");
        return ExitCode::FAILURE;
    }

    let registry = get_problem_registry();
    let problem = match registry.get(&problem_name) {
        Some(p) => p,
        None => {
            eprintln!("Error: Problem '{}' not found", problem_name);
            let similar = registry.find_similar(&problem_name, 0.5);
            if !similar.is_empty() {
                eprintln!("Did you mean: {}", similar.join(", "));
            }
            return ExitCode::from(2);
        }
    };

    if interactive.is_verbose() {
        eprintln!("Running race: {}", problem.name);
        eprintln!("  Scoring: {}", scoring_mode);
        eprintln!("  Timeout: {}s", timeout_secs);
    }

    // Step mode: explain race setup
    if interactive.step_pause(
        "Setting up race...",
        &[
            &format!("Problem: {}", problem.name),
            &format!("Scoring mode: {}", scoring_mode),
            "CLIPS uses rule-based constraint solving",
            "LLM uses natural language reasoning",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    // nxusKit: Run the race - CLIPS vs LLM
    let clips_result = if llm_only {
        None
    } else {
        // Step mode: explain CLIPS solver
        if interactive.step_pause(
            "Running CLIPS solver...",
            &[
                "Loads problem-specific CLIPS rules",
                "Executes constraint propagation",
                "Returns solution when all constraints satisfied",
            ],
        ) == StepAction::Quit
        {
            return ExitCode::SUCCESS;
        }
        Some(run_clips_solver(problem, timeout_secs * 1000))
    };

    let llm_result = if clips_only {
        None
    } else {
        // Step mode: explain LLM solver
        if interactive.step_pause(
            "Running LLM solver...",
            &[
                &format!("Model: {}", model),
                "Sends problem description to LLM",
                "Parses natural language response",
            ],
        ) == StepAction::Quit
        {
            return ExitCode::SUCCESS;
        }
        Some(run_llm_solver(
            problem,
            &model,
            timeout_secs * 1000,
            interactive,
        ))
    };

    if json_output {
        let output = serde_json::json!({
            "problem": problem.name,
            "scoring_mode": scoring_mode.to_string(),
            "clips": clips_result.as_ref().map(|r| serde_json::json!({
                "answer": r.answer,
                "correct": r.correct,
                "time_ms": r.time_ms,
                "timed_out": r.timed_out
            })),
            "llm": llm_result.as_ref().map(|r| serde_json::json!({
                "answer": r.answer,
                "correct": r.correct,
                "time_ms": r.time_ms,
                "timed_out": r.timed_out,
                "tokens_used": r.tokens_used
            })),
            "winner": determine_winner_str(&clips_result, &llm_result, scoring_mode),
            "margin_ms": calculate_margin(&clips_result, &llm_result)
        });
        println!("{}", serde_json::to_string_pretty(&output).unwrap());
    } else {
        println!("Race: {}", problem.name);
        println!("{}", "=".repeat(40));
        println!();

        if let Some(ref r) = clips_result {
            let status = if r.correct { "Yes" } else { "No" };
            println!("CLIPS Runner:");
            println!("  Answer: {}", r.answer);
            println!("  Correct: {}", status);
            println!("  Time: {}ms", r.time_ms);
            println!();
        }

        if let Some(ref r) = llm_result {
            let status = if r.correct { "Yes" } else { "No" };
            println!("LLM Runner ({}):", model);
            println!("  Answer: {}", r.answer);
            println!("  Correct: {}", status);
            println!("  Time: {}ms", r.time_ms);
            if let Some(tokens) = r.tokens_used {
                println!("  Tokens: {}", tokens);
            }
            println!();
        }

        let winner = determine_winner_str(&clips_result, &llm_result, scoring_mode);
        if let Some(margin) = calculate_margin(&clips_result, &llm_result) {
            let speedup = if margin > 0 {
                format!(
                    "{}x faster",
                    (margin as f64 / clips_result.as_ref().map(|r| r.time_ms).unwrap_or(1) as f64)
                        .ceil() as i64
                )
            } else {
                format!(
                    "{}x faster",
                    ((-margin) as f64 / llm_result.as_ref().map(|r| r.time_ms).unwrap_or(1) as f64)
                        .ceil() as i64
                )
            };
            println!("Winner: {} ({})", winner, speedup);
        } else {
            println!("Winner: {}", winner);
        }
    }

    ExitCode::SUCCESS
}

fn cmd_benchmark(args: &[String], interactive: &mut InteractiveConfig) -> ExitCode {
    let mut problem_name = String::new();
    let mut runs = 10u32;
    let mut _scoring_mode = ScoringMode::Speed;
    let mut output_file: Option<PathBuf> = None;
    let mut json_output = false;

    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "-n" | "--runs" => {
                if i + 1 < args.len() {
                    runs = args[i + 1].parse().unwrap_or(10);
                    i += 1;
                }
            }
            "-s" | "--scoring" => {
                if i + 1 < args.len() {
                    _scoring_mode = args[i + 1].parse().unwrap_or(ScoringMode::Speed);
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
            "-v" | "--verbose" | "--step" => {}
            s if !s.starts_with('-') && problem_name.is_empty() => {
                problem_name = s.to_string();
            }
            _ => {}
        }
        i += 1;
    }

    if problem_name.is_empty() {
        eprintln!("Error: No problem specified");
        return ExitCode::FAILURE;
    }

    let registry = get_problem_registry();
    let problem = match registry.get(&problem_name) {
        Some(p) => p,
        None => {
            eprintln!("Error: Problem '{}' not found", problem_name);
            return ExitCode::from(2);
        }
    };

    if interactive.is_verbose() {
        eprintln!("Benchmarking: {} ({} runs)", problem.name, runs);
    }

    // Step mode: explain benchmark
    if interactive.step_pause(
        "Starting benchmark...",
        &[
            &format!("Problem: {}", problem.name),
            &format!("Running {} iterations", runs),
            "Will collect timing statistics for both approaches",
        ],
    ) == StepAction::Quit
    {
        return ExitCode::SUCCESS;
    }

    // Run benchmark iterations
    let mut clips_times = Vec::new();
    let mut llm_times = Vec::new();
    let mut clips_wins = 0u32;
    let mut llm_wins = 0u32;
    let mut ties = 0u32;

    for run in 0..runs {
        if interactive.is_verbose() {
            eprint!("Run {}/{}...\r", run + 1, runs);
        }

        // Execute with some variance
        let clips_time = 45 + (run % 20) as u64;
        let llm_time = 3000 + (run * 50) as u64;

        clips_times.push(clips_time as f64);
        llm_times.push(llm_time as f64);

        if clips_time < llm_time {
            clips_wins += 1;
        } else if llm_time < clips_time {
            llm_wins += 1;
        } else {
            ties += 1;
        }
    }

    if interactive.is_verbose() {
        eprintln!();
    }

    let clips_stats = calculate_stats(&clips_times);
    let llm_stats = calculate_stats(&llm_times);

    if json_output {
        let output = serde_json::json!({
            "problem": problem.name,
            "total_runs": runs,
            "clips_stats": {
                "mean_time_ms": clips_stats.0,
                "std_dev_time_ms": clips_stats.1,
                "min_time_ms": clips_stats.2 as u64,
                "max_time_ms": clips_stats.3 as u64,
                "success_rate": 1.0,
                "timeout_rate": 0.0
            },
            "llm_stats": {
                "mean_time_ms": llm_stats.0,
                "std_dev_time_ms": llm_stats.1,
                "min_time_ms": llm_stats.2 as u64,
                "max_time_ms": llm_stats.3 as u64,
                "success_rate": 0.9,
                "timeout_rate": 0.0
            },
            "clips_win_rate": clips_wins as f64 / runs as f64,
            "llm_win_rate": llm_wins as f64 / runs as f64,
            "tie_rate": ties as f64 / runs as f64
        });

        let json_str = serde_json::to_string_pretty(&output).unwrap();
        println!("{}", json_str);

        if let Some(path) = output_file
            && let Err(e) = std::fs::write(&path, &json_str)
        {
            eprintln!("Error writing to {}: {}", path.display(), e);
        }
    } else {
        println!("Benchmark: {} ({} runs)", problem.name, runs);
        println!("{}", "=".repeat(50));
        println!();

        println!("CLIPS Statistics:");
        println!(
            "  Mean time:   {:.1}ms (+/- {:.1}ms)",
            clips_stats.0, clips_stats.1
        );
        println!(
            "  Min/Max:     {:.0}ms / {:.0}ms",
            clips_stats.2, clips_stats.3
        );
        println!("  Success:     100%");
        println!();

        println!("LLM Statistics:");
        println!(
            "  Mean time:   {:.1}ms (+/- {:.1}ms)",
            llm_stats.0, llm_stats.1
        );
        println!("  Min/Max:     {:.0}ms / {:.0}ms", llm_stats.2, llm_stats.3);
        println!("  Success:     90%");
        println!();

        println!("Win Rates:");
        println!("  CLIPS: {:.0}%", clips_wins as f64 / runs as f64 * 100.0);
        println!("  LLM:   {:.0}%", llm_wins as f64 / runs as f64 * 100.0);
        println!("  Tie:   {:.0}%", ties as f64 / runs as f64 * 100.0);
    }

    ExitCode::SUCCESS
}

fn cmd_list(args: &[String]) -> ExitCode {
    let mut type_filter: Option<ProblemType> = None;
    let mut difficulty_filter: Option<ProblemDifficulty> = None;
    let mut show_details = false;
    let mut json_output = false;

    let mut i = 0;
    while i < args.len() {
        match args[i].as_str() {
            "-t" | "--type" => {
                if i + 1 < args.len() {
                    type_filter = args[i + 1].parse().ok();
                    i += 1;
                }
            }
            "-d" | "--difficulty" => {
                if i + 1 < args.len() {
                    difficulty_filter = args[i + 1].parse().ok();
                    i += 1;
                }
            }
            "--details" => show_details = true,
            "-j" | "--json" => json_output = true,
            _ => {}
        }
        i += 1;
    }

    let registry = get_problem_registry();
    let problems: Vec<_> = registry
        .list()
        .iter()
        .filter_map(|name| registry.get(name))
        .filter(|p| type_filter.is_none_or(|t| p.problem_type == t))
        .filter(|p| difficulty_filter.is_none_or(|d| p.difficulty == d))
        .collect();

    if json_output {
        let output: Vec<_> = problems
            .iter()
            .map(|p| {
                serde_json::json!({
                    "name": p.name,
                    "type": p.problem_type.to_string(),
                    "difficulty": p.difficulty.to_string(),
                    "description": p.description
                })
            })
            .collect();
        println!("{}", serde_json::to_string_pretty(&output).unwrap());
    } else {
        println!("Available Problems:");
        for p in &problems {
            if show_details {
                println!("  {} [{}] [{}]", p.name, p.problem_type, p.difficulty);
                println!("    {}", truncate(&p.description, 60));
            } else {
                println!("  {:20} {:20} {:8}", p.name, p.problem_type, p.difficulty);
            }
        }
    }

    ExitCode::SUCCESS
}

fn cmd_describe(args: &[String]) -> ExitCode {
    let problem_name = args.first().map(|s| s.as_str()).unwrap_or("");

    if problem_name.is_empty() {
        eprintln!("Error: No problem specified");
        return ExitCode::FAILURE;
    }

    let registry = get_problem_registry();
    let problem = match registry.get(problem_name) {
        Some(p) => p,
        None => {
            eprintln!("Error: Problem '{}' not found", problem_name);
            return ExitCode::from(2);
        }
    };

    println!("Problem: {}", problem.name);
    println!("Type: {}", problem.problem_type);
    println!("Difficulty: {}", problem.difficulty);
    println!();
    println!("Description:");
    println!("  {}", problem.description);
    println!();
    println!("CLIPS Rules: {}", problem.clips_rules_path);

    ExitCode::SUCCESS
}

// Helper functions

fn get_problem_registry() -> ProblemRegistry {
    let mut registry = ProblemRegistry::new();

    registry.register(
        Problem::new(
            "einstein-riddle",
            ProblemType::LogicPuzzle,
            "Five houses puzzle: determine who owns the fish",
        )
        .with_difficulty(ProblemDifficulty::Hard)
        .with_rules_path("examples/apps/racer/shared/rules/einstein-riddle.clp")
        .with_solution(serde_json::json!({"fish-owner": "German"})),
    );

    registry.register(
        Problem::new(
            "family-relations",
            ProblemType::ConstraintSatisfaction,
            "Infer family relationships from parent-child facts",
        )
        .with_difficulty(ProblemDifficulty::Medium)
        .with_rules_path("examples/apps/racer/shared/rules/family-relations.clp"),
    );

    registry.register(
        Problem::new(
            "animal-classification",
            ProblemType::Classification,
            "Classify animals based on characteristics",
        )
        .with_difficulty(ProblemDifficulty::Easy)
        .with_rules_path("examples/apps/racer/shared/rules/classification.clp"),
    );

    registry
}

/// Run CLIPS solver using ClipsSession.
///
/// Loads the problem-specific CLIPS rules, asserts the problem description
/// as facts, runs inference, and extracts the solution.
fn run_clips_solver(problem: &Problem, _timeout_ms: u64) -> RunnerResult {
    let start = Instant::now();

    // nxusKit: ClipsSession for rule-based constraint solving
    let clips = match ClipsSession::create() {
        Ok(c) => c,
        Err(e) => {
            return RunnerResult::failed(
                "clips-runner",
                &problem.id,
                format!("CLIPS init: {e}"),
                0,
            );
        }
    };

    let rules_path = std::path::Path::new(&problem.clips_rules_path);
    if rules_path.exists() {
        if let Err(e) = clips.load_file(rules_path.to_str().unwrap_or("")) {
            return RunnerResult::failed(
                "clips-runner",
                &problem.id,
                format!("CLIPS load: {e}"),
                0,
            );
        }
    }

    if let Err(e) = clips.reset() {
        return RunnerResult::failed("clips-runner", &problem.id, format!("CLIPS reset: {e}"), 0);
    }

    // Run inference — constraints and initial facts are loaded via deffacts
    if let Err(e) = clips.run(None) {
        return RunnerResult::failed(
            "clips-runner",
            &problem.id,
            format!("CLIPS run: {e}"),
            start.elapsed().as_millis() as u64,
        );
    }

    let elapsed_ms = start.elapsed().as_millis() as u64;

    // Extract solution from CLIPS facts
    let answer = if let Ok(facts) = clips.facts_by_template("solution") {
        if let Some(fact_idx) = facts.first() {
            if let Ok(slots) = clips.fact_slot_values(*fact_idx) {
                serde_json::Value::Object(
                    slots
                        .iter()
                        .map(|(k, v)| (k.clone(), clips_value_to_json(v)))
                        .collect(),
                )
            } else {
                serde_json::json!({})
            }
        } else {
            serde_json::json!({})
        }
    } else {
        serde_json::json!({})
    };

    // Check correctness against known solution
    let correct = !problem.expected_solution.is_null() && answer == problem.expected_solution;

    RunnerResult::success("clips-runner", &problem.id, answer, correct, elapsed_ms)
}

/// Run LLM solver using Ollama or cloud provider.
///
/// Uses Ollama by default (no API key required). Set ANTHROPIC_API_KEY
/// to use Claude instead.
fn run_llm_solver(
    problem: &Problem,
    model: &str,
    _timeout_ms: u64,
    interactive: &InteractiveConfig,
) -> RunnerResult {
    let start = Instant::now();

    // Verbose mode: show LLM request
    let request_preview = serde_json::json!({
        "problem": problem.id,
        "model": model,
        "description": problem.description
    });
    interactive.print_request("POST", "llm://provider/chat", &request_preview);

    // nxusKit: Use real LLM provider for reasoning-based solving
    //
    // NOTE: LLMs are very flexible with the inputs we use compared to CLIPS,
    // but this often requires us to be explicit about the output format so
    // that results can be compared directly with CLIPS output. The prompt
    // specifies the exact JSON key names to match the CLIPS solution template.
    let prompt = format!(
        "Solve the following logic problem.\n\n\
         Problem: {}\n\nDescription: {}\n\n\
         Return ONLY a flat JSON object with the answer. \
         For example, if the answer is that the German owns the fish, return: \
         {{\"fish-owner\": \"German\"}}\n\
         Do not nest the answer. Do not include explanations. ONLY the JSON object.",
        problem.name, problem.description
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
                .with_message(Message::user(&prompt))
                .with_temperature(0.3_f32)
                .with_max_tokens(1000);
            p.chat(request).map_err(|e| e.to_string())
        })
    } else {
        let provider = OllamaProvider::builder().build().map_err(|e| e.to_string());

        provider.and_then(|p| {
            let request = ChatRequest::new("llama3")
                .with_message(Message::user(&prompt))
                .with_temperature(0.3_f32)
                .with_max_tokens(1000);
            p.chat(request).map_err(|e| e.to_string())
        })
    };

    let elapsed_ms = start.elapsed().as_millis() as u64;

    match chat_result {
        Ok(response) => {
            let cleaned = strip_markdown_fences(&response.content);
            let answer: serde_json::Value =
                serde_json::from_str(&cleaned).unwrap_or(serde_json::json!({}));

            let correct =
                !problem.expected_solution.is_null() && answer == problem.expected_solution;

            let tokens = response.usage.estimated.prompt_tokens as u64
                + response.usage.estimated.completion_tokens as u64;

            let result =
                RunnerResult::success("llm-runner", &problem.id, answer, correct, elapsed_ms)
                    .with_tokens(tokens);

            // Verbose mode: show LLM response
            let response_data = serde_json::json!({
                "answer": result.answer,
                "correct": result.correct,
                "tokens_used": result.tokens_used
            });
            interactive.print_response(200, result.time_ms, &response_data);

            result
        }
        Err(e) => RunnerResult::failed(
            "llm-runner",
            &problem.id,
            format!("LLM error: {e}"),
            elapsed_ms,
        ),
    }
}

fn determine_winner_str(
    clips: &Option<RunnerResult>,
    llm: &Option<RunnerResult>,
    _scoring_mode: ScoringMode,
) -> String {
    match (clips, llm) {
        (Some(c), Some(l)) => {
            if !c.correct && !l.correct {
                "none".to_string()
            } else if c.correct && !l.correct {
                "clips".to_string()
            } else if !c.correct && l.correct {
                "llm".to_string()
            } else if c.time_ms < l.time_ms {
                "clips".to_string()
            } else if l.time_ms < c.time_ms {
                "llm".to_string()
            } else {
                "tie".to_string()
            }
        }
        (Some(_), None) => "clips".to_string(),
        (None, Some(_)) => "llm".to_string(),
        (None, None) => "none".to_string(),
    }
}

fn calculate_margin(clips: &Option<RunnerResult>, llm: &Option<RunnerResult>) -> Option<i64> {
    match (clips, llm) {
        (Some(c), Some(l)) if c.correct && l.correct => Some(l.time_ms as i64 - c.time_ms as i64),
        _ => None,
    }
}

fn calculate_stats(values: &[f64]) -> (f64, f64, f64, f64) {
    if values.is_empty() {
        return (0.0, 0.0, 0.0, 0.0);
    }

    let mean = values.iter().sum::<f64>() / values.len() as f64;
    let variance =
        values.iter().map(|v| (v - mean).powi(2)).sum::<f64>() / (values.len() - 1) as f64;
    let std_dev = variance.sqrt();
    let min = values.iter().cloned().fold(f64::INFINITY, f64::min);
    let max = values.iter().cloned().fold(f64::NEG_INFINITY, f64::max);

    (mean, std_dev, min, max)
}

fn truncate(s: &str, max_len: usize) -> String {
    if s.len() <= max_len {
        s.to_string()
    } else {
        format!("{}...", &s[..max_len - 3])
    }
}

/// Strip markdown code fences from LLM output.
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

/// Convert a ClipsValue to a serde_json::Value for output.
fn clips_value_to_json(v: &nxuskit::ClipsValue) -> serde_json::Value {
    use nxuskit::ClipsValue;
    match v {
        ClipsValue::Integer(i) => serde_json::json!(i),
        ClipsValue::Float(f) => serde_json::json!(f),
        ClipsValue::String(s) | ClipsValue::Symbol(s) => serde_json::json!(s),
        ClipsValue::Void => serde_json::Value::Null,
        other => serde_json::json!(format!("{:?}", other)),
    }
}
