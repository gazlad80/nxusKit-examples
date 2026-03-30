//! Solver Example — nxusKit SDK
//!
//! Demonstrates the full solver session lifecycle using the nxusKit Rust SDK:
//!   1. Satisfaction (find any valid assignment)
//!   2. Single-objective optimization
//!   3. Multi-objective optimization (weighted mode)
//!   4. Soft constraints (weighted preferences)
//!   5. What-if analysis (push/pop scoping)
//!
//! Usage:
//!   cargo run -- --scenario theme-park [--verbose] [--step]

use std::path::PathBuf;
use std::time::Instant;

use clap::Parser;
use nxuskit::solver::SolverSession;
use nxuskit::solver_types::*;
use serde::Deserialize;

// ── CLI Arguments ───────────────────────────────────────────────────

#[derive(Parser, Debug)]
#[command(name = "solver-example")]
#[command(about = "Demonstrates the nxusKit solver lifecycle")]
struct Args {
    /// Scenario to solve: theme-park, space-colony, or fantasy-draft
    #[arg(short = 'S', long, default_value = "theme-park")]
    scenario: String,

    /// Enable verbose output showing raw JSON payloads
    #[arg(short, long)]
    verbose: bool,

    /// Enable step-through mode with explanations at each phase
    #[arg(short, long)]
    step: bool,
}

// ── Problem JSON Schema ─────────────────────────────────────────────
//
// These types mirror the problem.json structure. Constraints carry a
// human-readable `expression` for display; the solver uses `constraint_type`,
// `variables`, and `parameters` (injected if absent).

#[derive(Debug, Deserialize)]
struct Problem {
    name: String,
    description: String,
    variables: Vec<serde_json::Value>,
    constraints: Vec<serde_json::Value>,
    #[serde(default)]
    soft_constraints: Vec<serde_json::Value>,
    #[serde(default)]
    objectives: Vec<serde_json::Value>,
    #[serde(default)]
    what_if_scenarios: Vec<WhatIfScenario>,
}

#[derive(Debug, Deserialize)]
struct WhatIfScenario {
    name: String,
    description: String,
    additional_constraints: Vec<serde_json::Value>,
}

// ── Helpers ─────────────────────────────────────────────────────────

/// Print variable assignments from a SolveResult.
fn print_assignments(result: &SolveResult, verbose: bool) {
    let mut entries: Vec<_> = result.assignments.iter().collect();
    entries.sort_by_key(|(k, _)| k.to_string());
    for (name, val) in &entries {
        let display = match val {
            SolverValue::Integer(v) => format!("{v}"),
            SolverValue::Real(v) => format!("{v}"),
            SolverValue::Boolean(v) => format!("{v}"),
        };
        println!("  {name} = {display}");
    }
    if verbose && let Some(obj_val) = result.objective_value {
        println!("  [objective_value: {obj_val}]");
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

    // Try multiple base paths: relative to exe, relative to cwd, relative to manifest dir
    let candidate_bases = [
        exe_dir.join("../scenarios"),
        PathBuf::from("../scenarios"),
        PathBuf::from("scenarios"),
        PathBuf::from("examples/patterns/solver/scenarios"),
    ];

    let scenarios_dir = candidate_bases
        .iter()
        .find(|p| p.exists())
        .cloned()
        .unwrap_or_else(|| {
            eprintln!("Error: Could not locate scenarios directory.");
            eprintln!("Run from the solver/rust/ directory or pass --scenario with a valid name.");
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

    println!("=== Constraint Solver: {} ===", problem.name);
    println!("{}\n", problem.description);

    let var_names: Vec<String> = problem
        .variables
        .iter()
        .filter_map(|v| v.get("name").and_then(|n| n.as_str()).map(String::from))
        .collect();
    println!("Variables ({}): {}", var_names.len(), var_names.join(", "));
    println!();

    // ── Create solver session ───────────────────────────────────

    if interactive.step_pause(
        "Creating solver session...",
        &[
            "Allocates a Z3 solver context via the nxusKit Rust SDK",
            "Session persists state across incremental add/solve calls",
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

    // ── Step 1: Satisfaction ────────────────────────────────────

    println!("--- Step 1: Satisfaction ---");

    if interactive.step_pause(
        "Adding hard constraints and solving for satisfiability...",
        &[
            "Hard constraints must all be satisfied for a valid solution",
            "The solver finds ANY assignment that satisfies all constraints",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

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

    println!("Solving for satisfiability...");
    let step_start = Instant::now();
    let result = session.solve(None).unwrap_or_else(|e| {
        eprintln!("Error: Solver failed: {e}");
        std::process::exit(1);
    });
    let step_elapsed = step_start.elapsed();

    println!("Status: {:?}", result.status);
    print_assignments(&result, args.verbose);
    println!("Solve time: {}ms\n", step_elapsed.as_millis());

    // ── Step 2: Single-Objective Optimization ───────────────────

    println!("--- Step 2: Optimization ---");

    if problem.objectives.is_empty() {
        println!("(No objectives defined, skipping optimization)\n");
    } else {
        let first_obj_json = &problem.objectives[0];
        let obj_name = first_obj_json
            .get("name")
            .and_then(|n| n.as_str())
            .unwrap_or("unnamed");
        let obj_dir = first_obj_json
            .get("direction")
            .and_then(|d| d.as_str())
            .unwrap_or("maximize");
        let obj_expr = first_obj_json
            .get("expression")
            .and_then(|e| e.as_str())
            .unwrap_or("?");

        if interactive.step_pause(
            &format!("Setting objective: {obj_dir} {obj_expr}"),
            &[
                "Optimization finds the BEST assignment, not just any valid one",
                "The solver iteratively tightens bounds to find the optimum",
            ],
        ) == nxuskit_examples_interactive::StepAction::Quit
        {
            return;
        }

        println!("Objective: {obj_dir} {obj_expr}");

        let first_obj: ObjectiveDef = serde_json::from_value(first_obj_json.clone())
            .unwrap_or_else(|e| {
                eprintln!("Warning: Failed to parse objective '{obj_name}': {e}");
                std::process::exit(1);
            });

        if let Err(e) = session.set_objective(first_obj) {
            eprintln!("Warning: Failed to set objective '{obj_name}': {e}");
        } else {
            let step_start = Instant::now();
            let result = session.solve(None).unwrap_or_else(|e| {
                eprintln!("Warning: Optimization solve failed: {e}");
                std::process::exit(1);
            });
            let step_elapsed = step_start.elapsed();

            println!("Status: {:?}", result.status);
            print_assignments(&result, args.verbose);
            if let Some(obj_val) = result.objective_value {
                println!("Objective value: {obj_val}");
            }
            println!("Solve time: {}ms\n", step_elapsed.as_millis());
        }
    }

    // ── Step 3: Multi-Objective Optimization ────────────────────

    println!("--- Step 3: Multi-Objective ---");

    if problem.objectives.len() <= 1 {
        println!(
            "(Only {} objective defined, skipping multi-objective)\n",
            problem.objectives.len()
        );
    } else {
        let obj_summaries: Vec<String> = problem
            .objectives
            .iter()
            .map(|o| {
                let name = o.get("name").and_then(|n| n.as_str()).unwrap_or("?");
                let w = o.get("weight").and_then(|w| w.as_f64()).unwrap_or(1.0);
                format!("{name} (w={w})")
            })
            .collect();

        if interactive.step_pause(
            "Adding multiple objectives with weighted mode...",
            &[
                "Multi-objective combines several goals into a single weighted sum",
                "Higher-weight objectives have more influence on the solution",
            ],
        ) == nxuskit_examples_interactive::StepAction::Quit
        {
            return;
        }

        println!("Objectives: {}", obj_summaries.join(", "));
        println!("Mode: weighted");

        for obj_json in &problem.objectives {
            let obj: ObjectiveDef = serde_json::from_value(obj_json.clone()).unwrap_or_else(|e| {
                let name = obj_json.get("name").and_then(|n| n.as_str()).unwrap_or("?");
                eprintln!("Warning: Failed to parse objective '{name}': {e}");
                std::process::exit(1);
            });
            if let Err(e) = session.add_objective(obj) {
                let name = obj_json.get("name").and_then(|n| n.as_str()).unwrap_or("?");
                eprintln!("Warning: Failed to add objective '{name}': {e}");
            }
        }

        let config = SolverConfig {
            multi_objective_mode: Some(MultiObjectiveMode::Weighted),
            ..Default::default()
        };

        let step_start = Instant::now();
        let result = session.solve(Some(config)).unwrap_or_else(|e| {
            eprintln!("Warning: Multi-objective solve failed: {e}");
            std::process::exit(1);
        });
        let step_elapsed = step_start.elapsed();

        println!("Status: {:?}", result.status);
        print_assignments(&result, args.verbose);
        if let Some(obj_values) = &result.objective_values {
            for (name, val) in obj_values {
                println!("  [{name}]: {val}");
            }
        }
        println!("Solve time: {}ms\n", step_elapsed.as_millis());
    }

    // ── Step 4: Soft Constraints ────────────────────────────────

    println!("--- Step 4: Soft Constraints ---");

    if problem.soft_constraints.is_empty() {
        println!("(No soft constraints defined, skipping)\n");
    } else {
        let soft_summaries: Vec<String> = problem
            .soft_constraints
            .iter()
            .map(|c| {
                let name = c.get("name").and_then(|n| n.as_str()).unwrap_or("?");
                let w = c.get("weight").and_then(|w| w.as_f64()).unwrap_or(1.0);
                format!("{name} (weight={w})")
            })
            .collect();

        if interactive.step_pause(
            &format!(
                "Adding {} soft constraint(s)...",
                problem.soft_constraints.len()
            ),
            &[
                "Soft constraints are preferences, not requirements",
                "The solver satisfies them if possible but may violate low-weight ones",
            ],
        ) == nxuskit_examples_interactive::StepAction::Quit
        {
            return;
        }

        println!(
            "Adding {} soft constraint(s): {}",
            problem.soft_constraints.len(),
            soft_summaries.join(", ")
        );

        let soft_with_params = ensure_parameters(&problem.soft_constraints);
        let soft_constraints: Vec<ConstraintDef> =
            serde_json::from_value(serde_json::json!(soft_with_params)).unwrap_or_else(|e| {
                eprintln!("Warning: Failed to parse soft constraints: {e}");
                std::process::exit(1);
            });

        if let Err(e) = session.add_constraints(soft_constraints) {
            eprintln!("Warning: Failed to add soft constraints: {e}");
        }

        let step_start = Instant::now();
        let result = session.solve(None).unwrap_or_else(|e| {
            eprintln!("Warning: Soft constraint solve failed: {e}");
            std::process::exit(1);
        });
        let step_elapsed = step_start.elapsed();

        println!("Status: {:?}", result.status);
        print_assignments(&result, args.verbose);
        if let Some(violated) = &result.violated_soft_constraints {
            if violated.is_empty() {
                println!("All soft constraints satisfied.");
            } else {
                println!("Violated soft constraints: [{}]", violated.join(", "));
            }
        }
        println!("Solve time: {}ms\n", step_elapsed.as_millis());
    }

    // ── Step 5: What-If Analysis ────────────────────────────────

    println!("--- Step 5: What-If Analysis ---");

    if problem.what_if_scenarios.is_empty() {
        println!("(No what-if scenarios defined, skipping)\n");
    } else {
        for scenario in &problem.what_if_scenarios {
            if interactive.step_pause(
                &format!("What-if: \"{}\"", scenario.name),
                &[
                    &scenario.description.to_string(),
                    "Push saves the current model state",
                    "Pop restores it after the experiment",
                ],
            ) == nxuskit_examples_interactive::StepAction::Quit
            {
                return;
            }

            println!("Scenario: \"{}\"", scenario.name);
            println!("  Push scope...");

            if let Err(e) = session.push() {
                eprintln!("  Warning: Failed to push scope: {e}, skipping scenario.");
                continue;
            }

            let additional_with_params = ensure_parameters(&scenario.additional_constraints);
            let additional: Vec<ConstraintDef> = serde_json::from_value(serde_json::json!(
                additional_with_params
            ))
            .unwrap_or_else(|e| {
                eprintln!("  Warning: Failed to parse what-if constraints: {e}");
                Vec::new()
            });

            println!("  Adding {} temporary constraint(s)...", additional.len());

            if let Err(e) = session.add_constraints(additional) {
                eprintln!("  Warning: Failed to add what-if constraints: {e}");
            }

            println!("  Solving...");
            let step_start = Instant::now();
            match session.solve(None) {
                Ok(result) => {
                    let step_elapsed = step_start.elapsed();
                    println!("  Status: {:?}", result.status);
                    let mut entries: Vec<_> = result.assignments.iter().collect();
                    entries.sort_by_key(|(k, _)| k.to_string());
                    for (name, val) in &entries {
                        let display = match val {
                            SolverValue::Integer(v) => format!("{v}"),
                            SolverValue::Real(v) => format!("{v}"),
                            SolverValue::Boolean(v) => format!("{v}"),
                        };
                        println!("    {name} = {display}");
                    }
                    if let Some(obj_val) = result.objective_value {
                        println!("  Objective value: {obj_val}");
                    }
                    println!("  Solve time: {}ms", step_elapsed.as_millis());
                }
                Err(e) => {
                    eprintln!("  Warning: What-if solve failed: {e}");
                }
            }

            println!("  Pop scope (restoring base model)");
            if let Err(e) = session.pop() {
                eprintln!("  Warning: Failed to pop scope: {e}");
            }
            println!();
        }
    }

    // ── Summary ─────────────────────────────────────────────────

    let total_elapsed = total_start.elapsed();

    println!("=== Summary ===");
    println!("Scenario: {}", problem.name);
    println!("Total time: {}ms", total_elapsed.as_millis());

    let mut steps_completed = 1; // Satisfaction always runs
    if !problem.objectives.is_empty() {
        steps_completed += 1;
    }
    if problem.objectives.len() > 1 {
        steps_completed += 1;
    }
    if !problem.soft_constraints.is_empty() {
        steps_completed += 1;
    }
    if !problem.what_if_scenarios.is_empty() {
        steps_completed += 1;
    }
    println!("Steps completed: {steps_completed}");
}
