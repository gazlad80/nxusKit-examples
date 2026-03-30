//! ZEN Decision Table Example (E7) -- nxusKit SDK
//!
//! Demonstrates ZEN JSON Decision Model (JDM) evaluation with:
//!   1. Decision tables with "first" hit policy (maze-rat personality variants)
//!   2. Decision tables with "collect" hit policy (potion recipe selector)
//!   3. Expression nodes for computed outputs (food-truck planner)
//!
//! Usage:
//!   cargo run -- --scenario maze-rat [--verbose] [--step]
//!   cargo run -- --scenario potion --verbose
//!   cargo run -- --scenario food-truck --step

use std::path::PathBuf;
use std::time::Instant;

use clap::Parser;

// ── CLI Arguments ───────────────────────────────────────────────────

#[derive(Parser, Debug)]
#[command(name = "zen-decisions-example")]
#[command(about = "Demonstrates ZEN decision table evaluation with personality variants")]
struct Args {
    /// Scenario to run: maze-rat, potion, or food-truck
    #[arg(short = 'S', long, default_value = "maze-rat")]
    scenario: String,

    /// Enable verbose output showing raw JSON payloads
    #[arg(short, long)]
    verbose: bool,

    /// Enable step-through mode with explanations at each phase
    #[arg(short, long)]
    step: bool,
}

// ── Helpers ─────────────────────────────────────────────────────────

fn evaluate(model_json: &str, input_json: &str) -> Result<serde_json::Value, String> {
    nxuskit::zen_evaluate(model_json, input_json).map_err(|e| e.to_string())
}

/// Locate the scenarios directory by searching several candidate paths.
fn find_scenarios_dir() -> PathBuf {
    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|d| d.to_path_buf()))
        .unwrap_or_else(|| PathBuf::from("."));

    let candidates = [
        exe_dir.join("../scenarios"),
        PathBuf::from("../scenarios"),
        PathBuf::from("scenarios"),
        PathBuf::from("examples/integrations/zen-decisions/scenarios"),
    ];

    candidates
        .iter()
        .find(|p| p.exists())
        .cloned()
        .unwrap_or_else(|| {
            eprintln!("Error: Could not locate scenarios directory.");
            eprintln!("Run from the zen-decisions/rust/ directory.");
            std::process::exit(1);
        })
}

/// Discover available scenario directories (those containing input.json).
fn available_scenarios(scenarios_dir: &std::path::Path) -> Vec<String> {
    let mut names = Vec::new();
    if let Ok(entries) = std::fs::read_dir(scenarios_dir) {
        for entry in entries.flatten() {
            if entry.file_type().is_ok_and(|ft| ft.is_dir()) {
                let dir = entry.path();
                if dir.join("input.json").exists()
                    && let Some(name) = entry.file_name().to_str()
                {
                    names.push(name.to_string());
                }
            }
        }
    }
    names.sort();
    names
}

/// Pretty-print a serde_json::Value with indentation.
fn pretty_json(value: &serde_json::Value) -> String {
    serde_json::to_string_pretty(value).unwrap_or_else(|_| format!("{value:?}"))
}

/// Print result fields with indentation.
fn print_result_fields(result: &serde_json::Value, indent: &str) {
    if let Some(obj) = result.as_object() {
        for (key, value) in obj {
            match value {
                serde_json::Value::String(s) => println!("{indent}{key}: {s}"),
                serde_json::Value::Number(n) => println!("{indent}{key}: {n}"),
                serde_json::Value::Bool(b) => println!("{indent}{key}: {b}"),
                serde_json::Value::Null => println!("{indent}{key}: null"),
                _ => println!("{indent}{key}: {value}"),
            }
        }
    }
}

// ── Scenario Runners ────────────────────────────────────────────────

fn run_maze_rat(
    scenarios_dir: &std::path::Path,
    interactive: &mut nxuskit_examples_interactive::InteractiveConfig,
    verbose: bool,
) -> Result<(), String> {
    let scenario_dir = scenarios_dir.join("maze-rat");
    let input_json =
        std::fs::read_to_string(scenario_dir.join("input.json")).expect("cannot read input.json");
    let input: serde_json::Value = serde_json::from_str(&input_json).expect("invalid input.json");

    println!("=== ZEN Decision Tables: Maze Rat ===");
    println!("Personality variant comparison with first-hit policy\n");

    println!("Input:");
    print_result_fields(&input, "  ");
    println!();

    // ── Step 1: Evaluate all 3 personality JDMs ─────────────────

    let personalities = [
        ("cautious", "decision-model.json"),
        ("greedy", "greedy.json"),
        ("explorer", "explorer.json"),
    ];

    let mut results: Vec<(&str, serde_json::Value, std::time::Duration)> = Vec::new();

    for (name, filename) in &personalities {
        println!("--- Personality: {} ---", name);

        if interactive.step_pause(
            &format!("Evaluating {name} personality decision table..."),
            &[
                "Loads the JDM file for this personality variant",
                "Evaluates against the same input using first-hit policy",
                "First matching rule determines the action",
            ],
        ) == nxuskit_examples_interactive::StepAction::Quit
        {
            return Ok(());
        }

        let model_path = scenario_dir.join(filename);
        let model_json = std::fs::read_to_string(&model_path).unwrap_or_else(|e| {
            eprintln!("Error reading {}: {e}", model_path.display());
            std::process::exit(1);
        });

        let start = Instant::now();
        let result = evaluate(&model_json, &input_json);
        let elapsed = start.elapsed();

        match result {
            Ok(value) => {
                println!("  Result:");
                print_result_fields(&value, "    ");
                println!("  Time: {}us", elapsed.as_micros());

                if verbose {
                    eprintln!("\n  [verbose] Raw result:\n  {}", pretty_json(&value));
                }

                results.push((name, value, elapsed));
            }
            Err(e) => {
                return Err(format!("{name} evaluation failed: {e}"));
            }
        }
        println!();
    }

    // ── Step 2: Compare personality outcomes ────────────────────

    println!("--- Personality Comparison ---");

    if interactive.step_pause(
        "Comparing decisions across personality variants...",
        &[
            "Same input, different decision tables produce different actions",
            "Cautious avoids risk, greedy follows scent, explorer seeks new paths",
            "Confidence values reflect each personality's certainty",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return Ok(());
    }

    println!(
        "  {:<12} {:<15} {:<12} {:>8}",
        "Personality", "Action", "Confidence", "Time"
    );
    println!("  {}", "-".repeat(50));

    for (name, value, elapsed) in &results {
        let action = value.get("action").and_then(|v| v.as_str()).unwrap_or("?");
        let confidence = value
            .get("confidence")
            .and_then(|v| v.as_f64())
            .unwrap_or(0.0);
        println!(
            "  {:<12} {:<15} {:<12.2} {:>5}us",
            name,
            action,
            confidence,
            elapsed.as_micros()
        );
    }
    println!();

    Ok(())
}

fn run_potion(
    scenarios_dir: &std::path::Path,
    interactive: &mut nxuskit_examples_interactive::InteractiveConfig,
    verbose: bool,
) -> Result<(), String> {
    let scenario_dir = scenarios_dir.join("potion");
    let input_json =
        std::fs::read_to_string(scenario_dir.join("input.json")).expect("cannot read input.json");
    let input: serde_json::Value = serde_json::from_str(&input_json).expect("invalid input.json");
    let model_json = std::fs::read_to_string(scenario_dir.join("decision-model.json"))
        .expect("cannot read decision-model.json");

    println!("=== ZEN Decision Tables: Potion Recipes ===");
    println!("Collect hit policy -- returns all matching recipes\n");

    println!("Input:");
    print_result_fields(&input, "  ");
    println!();

    // ── Step 1: Evaluate with collect hit policy ────────────────

    println!("--- Evaluate Potion Recipes ---");

    if interactive.step_pause(
        "Evaluating potion decision table with collect hit policy...",
        &[
            "Collect hit policy returns ALL matching rules, not just the first",
            "Multiple recipes can match the same input",
            "Each result includes recipe name, steps, and warnings",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return Ok(());
    }

    let start = Instant::now();
    let result = evaluate(&model_json, &input_json);
    let elapsed = start.elapsed();

    match result {
        Ok(value) => {
            // Collect policy may return an array or an object with array fields
            if let Some(arr) = value.as_array() {
                println!("  Matching recipes: {}\n", arr.len());
                for (i, recipe) in arr.iter().enumerate() {
                    println!("  Recipe {}:", i + 1);
                    print_result_fields(recipe, "    ");
                    println!();
                }
            } else {
                // Single result (or object format)
                println!("  Result:");
                print_result_fields(&value, "    ");
                println!();
            }
            println!("  Time: {}us", elapsed.as_micros());

            if verbose {
                eprintln!("\n  [verbose] Raw result:\n  {}", pretty_json(&value));
            }
        }
        Err(e) => {
            return Err(format!("potion evaluation failed: {e}"));
        }
    }
    println!();

    Ok(())
}

fn run_food_truck(
    scenarios_dir: &std::path::Path,
    interactive: &mut nxuskit_examples_interactive::InteractiveConfig,
    verbose: bool,
) -> Result<(), String> {
    let scenario_dir = scenarios_dir.join("food-truck");
    let input_json =
        std::fs::read_to_string(scenario_dir.join("input.json")).expect("cannot read input.json");
    let input: serde_json::Value = serde_json::from_str(&input_json).expect("invalid input.json");
    let model_json = std::fs::read_to_string(scenario_dir.join("decision-model.json"))
        .expect("cannot read decision-model.json");

    println!("=== ZEN Decision Tables: Food Truck Planner ===");
    println!("Decision table + expression node pipeline\n");

    println!("Input:");
    print_result_fields(&input, "  ");
    println!();

    // ── Step 1: Evaluate decision + expression pipeline ─────────

    println!("--- Evaluate Food Truck Decision ---");

    if interactive.step_pause(
        "Evaluating food truck decision pipeline...",
        &[
            "Decision table selects location and base price multiplier",
            "Expression node computes menu adjustment and restock alert",
            "Pipeline: inputNode -> decisionTableNode -> expressionNode -> outputNode",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return Ok(());
    }

    let start = Instant::now();
    let result = evaluate(&model_json, &input_json);
    let elapsed = start.elapsed();

    match result {
        Ok(value) => {
            println!("  Decision output:");
            let location = value
                .get("location")
                .and_then(|v| v.as_str())
                .unwrap_or("?");
            let price_mult = value
                .get("price_multiplier")
                .and_then(|v| v.as_f64())
                .unwrap_or(0.0);
            let menu = value
                .get("menu_adjustment")
                .and_then(|v| v.as_str())
                .unwrap_or("?");
            let restock = value
                .get("restock_alert")
                .and_then(|v| v.as_bool())
                .unwrap_or(false);

            println!("    location:        {location}");
            println!("    price_multiplier: {price_mult:.2}");
            println!("    menu_adjustment: {menu}");
            println!("    restock_alert:   {restock}");
            println!();
            println!("  Time: {}us", elapsed.as_micros());

            if verbose {
                eprintln!("\n  [verbose] Raw result:\n  {}", pretty_json(&value));
            }
        }
        Err(e) => {
            return Err(format!("food-truck evaluation failed: {e}"));
        }
    }
    println!();

    Ok(())
}

// ── Main ────────────────────────────────────────────────────────────

fn main() {
    let args = Args::parse();
    let mut interactive =
        nxuskit_examples_interactive::InteractiveConfig::new(args.verbose, args.step);
    let total_start = Instant::now();

    // ── Locate scenario directory ───────────────────────────────

    let scenarios_dir = find_scenarios_dir();

    // Validate scenario
    let available = available_scenarios(&scenarios_dir);
    if !available.contains(&args.scenario) {
        eprintln!("Error: Unknown scenario '{}'.", args.scenario);
        if available.is_empty() {
            eprintln!("No scenarios found in {}", scenarios_dir.display());
        } else {
            eprintln!("Available scenarios: {}", available.join(", "));
        }
        std::process::exit(1);
    }

    // ── Run scenario ────────────────────────────────────────────

    let result = match args.scenario.as_str() {
        "maze-rat" => run_maze_rat(&scenarios_dir, &mut interactive, args.verbose),
        "potion" => run_potion(&scenarios_dir, &mut interactive, args.verbose),
        "food-truck" => run_food_truck(&scenarios_dir, &mut interactive, args.verbose),
        _ => {
            eprintln!(
                "Error: Unknown scenario '{}'. Available: {}",
                args.scenario,
                available.join(", ")
            );
            std::process::exit(1);
        }
    };

    if let Err(e) = result {
        eprintln!("Error: {e}");
        std::process::exit(1);
    }

    // ── Summary ─────────────────────────────────────────────────

    let total_elapsed = total_start.elapsed();

    println!("=== Summary ===");
    println!("Scenario:   {}", args.scenario);
    println!("Total time: {}ms", total_elapsed.as_millis());
    println!();
    println!("Done.");
}
