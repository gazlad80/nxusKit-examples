//! Solver What-If Example — nxusKit SDK
//!
//! Demonstrates push/pop scoping, assumption-based solving, and what-if analysis:
//!   1. Load a problem with base constraints and objectives
//!   2. Solve the base problem optimally
//!   3. For each what-if scenario: push scope, add constraints, solve, compare, pop
//!   4. Print a summary comparing base vs. all what-if variants
//!
//! Key nxusKit features: push/pop scoping, explanation/unsat-core retrieval,
//! incremental constraint addition, and delta comparison.
//!
//! Usage:
//!   cargo run -- --scenario wedding [--verbose] [--step]

use std::collections::BTreeMap;
use std::path::PathBuf;
use std::time::Instant;

use clap::Parser;
use nxuskit::solver::SolverSession;
use nxuskit::solver_types::*;
use serde::Deserialize;

// ── CLI Arguments ───────────────────────────────────────────────────

#[derive(Parser, Debug)]
#[command(name = "solver-what-if-example")]
#[command(about = "Demonstrates push/pop what-if analysis with the nxusKit constraint solver")]
struct Args {
    /// Scenario to solve: wedding, mars, or recipe
    #[arg(short = 'S', long, default_value = "wedding")]
    scenario: String,

    /// Enable verbose output showing raw JSON payloads
    #[arg(short, long)]
    verbose: bool,

    /// Enable step-through mode with explanations at each phase
    #[arg(short, long)]
    step: bool,
}

// ── Problem JSON Schema ─────────────────────────────────────────────

#[derive(Debug, Deserialize)]
struct Problem {
    name: String,
    description: String,
    variables: Vec<serde_json::Value>,
    constraints: Vec<serde_json::Value>,
    #[serde(default)]
    objectives: Vec<serde_json::Value>,
    what_if_scenarios: Vec<WhatIfScenario>,
}

#[derive(Debug, Deserialize)]
struct WhatIfScenario {
    name: String,
    description: String,
    additional_constraints: Vec<serde_json::Value>,
}

// ── Helpers ─────────────────────────────────────────────────────────

/// Extract variable assignments from a SolveResult as a sorted map of f64.
fn extract_assignments(result: &SolveResult) -> BTreeMap<String, f64> {
    let mut map = BTreeMap::new();
    for (name, val) in &result.assignments {
        let v = match val {
            SolverValue::Integer(i) => *i as f64,
            SolverValue::Real(r) => *r,
            SolverValue::Boolean(b) => {
                if *b {
                    1.0
                } else {
                    0.0
                }
            }
        };
        map.insert(name.clone(), v);
    }
    map
}

/// Print variable assignments from a SolveResult.
fn print_assignments(result: &SolveResult, indent: &str) {
    let mut entries: Vec<_> = result.assignments.iter().collect();
    entries.sort_by_key(|(k, _)| k.to_string());
    for (name, val) in &entries {
        let display = match val {
            SolverValue::Integer(v) => format!("{v}"),
            SolverValue::Real(v) => format!("{v}"),
            SolverValue::Boolean(v) => format!("{v}"),
        };
        println!("{indent}{name} = {display}");
    }
}

/// Print a delta comparison between base and what-if assignments.
fn print_delta(base: &BTreeMap<String, f64>, what_if: &BTreeMap<String, f64>, indent: &str) {
    let mut any_delta = false;
    for (name, base_val) in base {
        if let Some(wi_val) = what_if.get(name) {
            let diff = wi_val - base_val;
            if diff.abs() > 0.001 {
                let sign = if diff > 0.0 { "+" } else { "" };
                println!("{indent}{name}: {base_val} -> {wi_val} ({sign}{diff:.0})");
                any_delta = true;
            }
        }
    }
    if !any_delta {
        println!("{indent}(no changes from base)");
    }
}

/// Discover available scenario directories.
fn available_scenarios(scenarios_dir: &std::path::Path) -> Vec<String> {
    let mut names = Vec::new();
    if let Ok(entries) = std::fs::read_dir(scenarios_dir) {
        for entry in entries.flatten() {
            if entry.file_type().is_ok_and(|ft| ft.is_dir())
                && let Some(name) = entry.file_name().to_str()
            {
                names.push(name.to_string());
            }
        }
    }
    names.sort();
    names
}

/// Ensure every constraint JSON object has a `parameters` field.
fn ensure_parameters(constraints: &[serde_json::Value]) -> Vec<serde_json::Value> {
    constraints
        .iter()
        .map(|c| {
            let mut obj = c.clone();
            if let Some(map) = obj.as_object_mut() {
                map.entry("parameters").or_insert(serde_json::json!({}));
            }
            obj
        })
        .collect()
}

// ── Scenario Result (for summary) ──────────────────────────────────

struct ScenarioResult {
    name: String,
    status: SolveStatus,
    objective_value: Option<f64>,
}

// ── Main ────────────────────────────────────────────────────────────

fn main() {
    let args = Args::parse();
    let mut interactive =
        nxuskit_examples_interactive::InteractiveConfig::new(args.verbose, args.step);
    let total_start = Instant::now();

    // ── Locate scenario directory ───────────────────────────────

    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|d| d.to_path_buf()))
        .unwrap_or_else(|| PathBuf::from("."));

    let candidate_bases = [
        exe_dir.join("../scenarios"),
        PathBuf::from("../scenarios"),
        PathBuf::from("scenarios"),
        PathBuf::from("examples/patterns/solver-what-if/scenarios"),
    ];

    let scenarios_dir = candidate_bases
        .iter()
        .find(|p| p.exists())
        .cloned()
        .unwrap_or_else(|| {
            eprintln!("Error: Could not locate scenarios directory.");
            eprintln!(
                "Run from the solver-what-if/rust/ directory or pass --scenario with a valid name."
            );
            std::process::exit(1);
        });

    let scenario_dir = scenarios_dir.join(&args.scenario);
    let problem_path = scenario_dir.join("problem.json");

    if !problem_path.exists() {
        let available = available_scenarios(&scenarios_dir);
        eprintln!("Error: Unknown scenario '{}'.", args.scenario);
        if available.is_empty() {
            eprintln!("No scenarios found in {}", scenarios_dir.display());
        } else {
            eprintln!("Available scenarios: {}", available.join(", "));
        }
        std::process::exit(1);
    }

    // ── Load problem ────────────────────────────────────────────

    let problem_json = std::fs::read_to_string(&problem_path).unwrap_or_else(|e| {
        eprintln!("Error: Failed to read {}: {e}", problem_path.display());
        std::process::exit(1);
    });

    let problem: Problem = serde_json::from_str(&problem_json).unwrap_or_else(|e| {
        eprintln!("Error: Failed to parse {}: {e}", problem_path.display());
        std::process::exit(1);
    });

    // ── Print header ────────────────────────────────────────────

    println!("=== Solver What-If: {} ===", problem.name);
    println!("{}\n", problem.description);

    let var_names: Vec<String> = problem
        .variables
        .iter()
        .filter_map(|v| v.get("name").and_then(|n| n.as_str()).map(String::from))
        .collect();
    println!("Variables ({}): {}", var_names.len(), var_names.join(", "));
    println!("Hard constraints: {}", problem.constraints.len());
    println!("What-if scenarios: {}", problem.what_if_scenarios.len());
    println!();

    // ── Create solver session ───────────────────────────────────

    if interactive.step_pause(
        "Creating solver session...",
        &[
            "Allocates a Z3 solver context via the nxusKit Rust SDK",
            "Session persists state across incremental add/solve calls",
            "Push/Pop enables reversible what-if exploration",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let mut session = SolverSession::create(None).unwrap_or_else(|e| {
        eprintln!("Error: Failed to create solver session: {e}");
        std::process::exit(1);
    });

    // ── Add variables ───────────────────────────────────────────

    let vars: Vec<VariableDef> = serde_json::from_value(serde_json::json!(problem.variables))
        .unwrap_or_else(|e| {
            eprintln!("Error: Failed to parse variables: {e}");
            std::process::exit(1);
        });

    if args.verbose {
        eprintln!("[verbose] Adding {} variables", vars.len());
    }

    session.add_variables(vars).unwrap_or_else(|e| {
        eprintln!("Error: Failed to add variables to solver session: {e}");
        std::process::exit(1);
    });

    // ── Add hard constraints ────────────────────────────────────

    let constraints_with_params = ensure_parameters(&problem.constraints);
    let constraints: Vec<ConstraintDef> =
        serde_json::from_value(serde_json::json!(constraints_with_params)).unwrap_or_else(|e| {
            eprintln!("Error: Failed to parse constraints: {e}");
            std::process::exit(1);
        });

    println!("Adding {} hard constraint(s)...", constraints.len());

    session.add_constraints(constraints).unwrap_or_else(|e| {
        eprintln!("Error: Failed to add constraints: {e}");
        std::process::exit(1);
    });

    // ── Set objective ───────────────────────────────────────────

    if !problem.objectives.is_empty() {
        let first_obj_json = &problem.objectives[0];
        let obj_dir = first_obj_json
            .get("direction")
            .and_then(|d| d.as_str())
            .unwrap_or("maximize");
        let obj_expr = first_obj_json
            .get("expression")
            .and_then(|e| e.as_str())
            .unwrap_or("?");

        println!("Setting objective: {obj_dir} {obj_expr}");

        let first_obj: ObjectiveDef = serde_json::from_value(first_obj_json.clone())
            .unwrap_or_else(|e| {
                eprintln!("Warning: Failed to parse objective: {e}");
                std::process::exit(1);
            });

        if let Err(e) = session.set_objective(first_obj) {
            eprintln!("Warning: Failed to set objective: {e}");
        }
    }

    println!();

    // ── Step 1: Solve base problem ──────────────────────────────

    println!("--- Base Problem ---");

    if interactive.step_pause(
        "Solving the base problem with all hard constraints and objective...",
        &[
            "This establishes the baseline optimal solution",
            "What-if scenarios will be compared against this result",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let explain_config = SolverConfig {
        produce_explanation: Some(true),
        ..Default::default()
    };

    let step_start = Instant::now();
    let base_result = session
        .solve(Some(explain_config.clone()))
        .unwrap_or_else(|e| {
            eprintln!("Error: Solver failed: {e}");
            std::process::exit(1);
        });
    let step_elapsed = step_start.elapsed();

    let base_assignments = extract_assignments(&base_result);
    let base_objective = base_result.objective_value;

    println!("Status: {:?}", base_result.status);
    print_assignments(&base_result, "  ");
    if let Some(obj_val) = base_result.objective_value {
        println!("Objective value: {obj_val}");
    }
    println!("Solve time: {}ms\n", step_elapsed.as_millis());

    let mut all_results: Vec<ScenarioResult> = vec![ScenarioResult {
        name: "Base".to_string(),
        status: base_result.status,
        objective_value: base_objective,
    }];

    // ── Step 2: What-If Scenarios ───────────────────────────────

    println!("--- What-If Analysis ---\n");

    for (i, scenario) in problem.what_if_scenarios.iter().enumerate() {
        println!("Scenario {}: \"{}\"", i + 1, scenario.name);
        println!("  {}\n", scenario.description);

        if interactive.step_pause(
            &format!("What-if: \"{}\"", scenario.name),
            &[
                &scenario.description,
                "Push saves the current model state",
                "Additional constraints are added temporarily",
                "Pop restores the base model after the experiment",
            ],
        ) == nxuskit_examples_interactive::StepAction::Quit
        {
            return;
        }

        // Push scope
        println!("  Push scope...");
        if let Err(e) = session.push() {
            eprintln!("  Warning: Failed to push scope: {e}, skipping scenario.");
            continue;
        }

        // Add what-if constraints
        let additional_with_params = ensure_parameters(&scenario.additional_constraints);

        if args.verbose {
            for c in &scenario.additional_constraints {
                let label = c
                    .get("label")
                    .and_then(|l| l.as_str())
                    .or_else(|| c.get("name").and_then(|n| n.as_str()))
                    .unwrap_or("?");
                let expr = c.get("expression").and_then(|e| e.as_str()).unwrap_or("?");
                eprintln!("  [verbose] {label}: {expr}");
            }
        }

        let additional: Vec<ConstraintDef> =
            serde_json::from_value(serde_json::json!(additional_with_params)).unwrap_or_else(|e| {
                eprintln!("  Warning: Failed to parse what-if constraints: {e}");
                Vec::new()
            });

        println!("  Adding {} temporary constraint(s)...", additional.len());

        if let Err(e) = session.add_constraints(additional) {
            eprintln!("  Warning: Failed to add what-if constraints: {e}");
        }

        // Solve with explanation enabled
        println!("  Solving...");
        let step_start = Instant::now();
        match session.solve(Some(explain_config.clone())) {
            Ok(result) => {
                let step_elapsed = step_start.elapsed();
                println!("  Status: {:?}", result.status);

                if result.status == SolveStatus::Unsat {
                    // Try to get explanation / unsat core
                    println!("  Attempting to retrieve explanation...");
                    match session.explanation() {
                        Ok(Some(expl)) => {
                            if let Some(labels) = &expl.unsat_core_labels
                                && !labels.is_empty()
                            {
                                println!("  Unsat core: [{}]", labels.join(", "));
                            }
                            if args.verbose {
                                eprintln!(
                                    "  [verbose] Explanation: {}",
                                    serde_json::to_string_pretty(&expl).unwrap_or_default()
                                );
                            }
                        }
                        Ok(None) | Err(_) => {
                            println!("  (no explanation available)");
                        }
                    }

                    all_results.push(ScenarioResult {
                        name: scenario.name.clone(),
                        status: result.status,
                        objective_value: None,
                    });
                } else {
                    print_assignments(&result, "    ");

                    if let Some(obj_val) = result.objective_value {
                        println!("  Objective value: {obj_val}");
                    }

                    // Show delta from base
                    let wi_assignments = extract_assignments(&result);
                    println!("  Delta from base:");
                    print_delta(&base_assignments, &wi_assignments, "    ");

                    // Show objective delta
                    if let (Some(base_obj), Some(wi_obj)) = (base_objective, result.objective_value)
                    {
                        let diff = wi_obj - base_obj;
                        let sign = if diff > 0.0 { "+" } else { "" };
                        println!("  Objective delta: {base_obj} -> {wi_obj} ({sign}{diff:.0})");
                    }

                    println!("  Solve time: {}ms", step_elapsed.as_millis());

                    all_results.push(ScenarioResult {
                        name: scenario.name.clone(),
                        status: result.status,
                        objective_value: result.objective_value,
                    });
                }
            }
            Err(e) => {
                eprintln!("  Warning: What-if solve failed: {e}");
                all_results.push(ScenarioResult {
                    name: scenario.name.clone(),
                    status: SolveStatus::Unknown,
                    objective_value: None,
                });
            }
        }

        // Pop scope
        println!("  Pop scope (restoring base model)\n");
        if let Err(e) = session.pop() {
            eprintln!("  Warning: Failed to pop scope: {e}");
        }
    }

    // ── Summary ─────────────────────────────────────────────────

    let total_elapsed = total_start.elapsed();

    println!("=== Summary ===");
    println!(
        "Scenario: {} ({} what-if variants)\n",
        problem.name,
        problem.what_if_scenarios.len()
    );

    let name_width = all_results
        .iter()
        .map(|r| r.name.len())
        .max()
        .unwrap_or(10)
        .max(10);

    println!(
        "  {:<width$}  {:>10}  {:>15}",
        "Variant",
        "Status",
        "Objective",
        width = name_width
    );
    println!(
        "  {:-<width$}  {:-^10}  {:-^15}",
        "",
        "",
        "",
        width = name_width
    );

    for r in &all_results {
        let obj_str = match r.objective_value {
            Some(v) => format!("{v:.0}"),
            None => "-".to_string(),
        };
        let status_str = format!("{:?}", r.status).to_lowercase();
        let icon = match r.status {
            SolveStatus::Sat | SolveStatus::Optimal => "[OK]",
            SolveStatus::Unsat => "[!!]",
            SolveStatus::Timeout => "[TO]",
            SolveStatus::Unknown => "[??]",
        };
        println!(
            "  {:<width$}  {icon} {:>5}  {:>15}",
            r.name,
            status_str,
            obj_str,
            width = name_width
        );
    }

    println!("\nTotal time: {}ms", total_elapsed.as_millis());
    println!("Done.");
}
