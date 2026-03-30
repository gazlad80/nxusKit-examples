//! Multi-Provider Pipeline Example (E3) — nxusKit SDK
//!
//! Demonstrates a 3-stage pipeline combining three nxusKit providers:
//!   Stage 1: BN Prediction — Bayesian Network predicts key scenario variable
//!   Stage 2: Solver Optimization — Constraint solver optimizes assignments
//!   Stage 3: CLIPS Safety Enforcement — Rule engine validates safety constraints
//!
//! The pipeline flows data forward: BN posteriors inform the solver's problem
//! context, and solver assignments are asserted as CLIPS facts for rule-based
//! safety checking.
//!
//! Usage:
//!   cargo run -- --scenario festival [--verbose] [--step]

use std::collections::HashMap;
use std::path::PathBuf;
use std::time::Instant;

use clap::Parser;
use nxuskit::ClipsValue;
use nxuskit::bn::{BnEvidence, BnNetwork};
use nxuskit::clips::ClipsSession;
use nxuskit::solver::SolverSession;
use nxuskit::solver_types::*;
use serde::Deserialize;

// ── CLI Arguments ───────────────────────────────────────────────────

#[derive(Parser, Debug)]
#[command(name = "bn-solver-clips-pipeline-example")]
#[command(about = "Demonstrates a BN -> Solver -> CLIPS multi-provider pipeline")]
struct Args {
    /// Scenario to run: festival, rescue, or bakery
    #[arg(short = 'S', long, default_value = "festival")]
    scenario: String,

    /// Enable verbose output showing raw JSON payloads
    #[arg(short, long)]
    verbose: bool,

    /// Enable step-through mode with explanations at each phase
    #[arg(short, long)]
    step: bool,
}

// ── Scenario Configuration ──────────────────────────────────────────

struct ScenarioConfig {
    title: &'static str,
    bn_target: &'static str,
    assignment_template: &'static str,
    alert_template: &'static str,
}

fn scenario_config(name: &str) -> Option<ScenarioConfig> {
    match name {
        "festival" => Some(ScenarioConfig {
            title: "Festival Safety Pipeline",
            bn_target: "crowd_size",
            assignment_template: "stage-assignment",
            alert_template: "safety-alert",
        }),
        "rescue" => Some(ScenarioConfig {
            title: "Rescue Operations Pipeline",
            bn_target: "survivor_probability",
            assignment_template: "rescue-assignment",
            alert_template: "protocol-alert",
        }),
        "bakery" => Some(ScenarioConfig {
            title: "Bakery Scheduling Pipeline",
            bn_target: "demand_level",
            assignment_template: "baking-assignment",
            alert_template: "health-alert",
        }),
        _ => None,
    }
}

// ── Problem JSON Schema ─────────────────────────────────────────────

#[derive(Debug, Deserialize)]
struct Problem {
    name: String,
    #[allow(dead_code)]
    description: String,
    variables: Vec<serde_json::Value>,
    constraints: Vec<serde_json::Value>,
    #[serde(default)]
    objectives: Vec<serde_json::Value>,
}

// ── Helpers ─────────────────────────────────────────────────────────

/// Discover available scenario directories (those containing model.bif + problem.json + rules.clp).
fn available_scenarios(scenarios_dir: &std::path::Path) -> Vec<String> {
    let mut names = Vec::new();
    if let Ok(entries) = std::fs::read_dir(scenarios_dir) {
        for entry in entries.flatten() {
            if entry.file_type().is_ok_and(|ft| ft.is_dir()) {
                let dir = entry.path();
                let has_all = dir.join("model.bif").exists()
                    && dir.join("problem.json").exists()
                    && dir.join("rules.clp").exists();
                if has_all && let Some(name) = entry.file_name().to_str() {
                    names.push(name.to_string());
                }
            }
        }
    }
    names.sort();
    names
}

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
        println!("    {name} = {display}");
    }
    if verbose && let Some(obj_val) = result.objective_value {
        println!("    [objective_value: {obj_val}]");
    }
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

/// Extract an integer value from a solver assignment map.
fn assignment_int(assignments: &HashMap<String, SolverValue>, key: &str) -> i64 {
    match assignments.get(key) {
        Some(SolverValue::Integer(v)) => *v,
        Some(SolverValue::Real(v)) => *v as i64,
        Some(SolverValue::Boolean(v)) => {
            if *v {
                1
            } else {
                0
            }
        }
        None => 0,
    }
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
        PathBuf::from("examples/integrations/bn-solver-clips-pipeline/scenarios"),
    ];

    let scenarios_dir = candidate_bases
        .iter()
        .find(|p| p.exists())
        .cloned()
        .unwrap_or_else(|| {
            eprintln!("Error: Could not locate scenarios directory.");
            eprintln!(
                "Run from the bn-solver-clips-pipeline/rust/ directory or pass --scenario with a valid name."
            );
            std::process::exit(1);
        });

    // Validate scenario
    let config = match scenario_config(&args.scenario) {
        Some(c) => c,
        None => {
            let available = available_scenarios(&scenarios_dir);
            eprintln!("Error: Unknown scenario '{}'.", args.scenario);
            if available.is_empty() {
                eprintln!("No scenarios found in {}", scenarios_dir.display());
            } else {
                eprintln!("Available scenarios: {}", available.join(", "));
            }
            std::process::exit(1);
        }
    };

    let scenario_dir = scenarios_dir.join(&args.scenario);
    let model_path = scenario_dir.join("model.bif");
    let evidence_path = scenario_dir.join("evidence.json");
    let problem_path = scenario_dir.join("problem.json");
    let rules_path = scenario_dir.join("rules.clp");

    if !model_path.exists() || !problem_path.exists() || !rules_path.exists() {
        let available = available_scenarios(&scenarios_dir);
        eprintln!(
            "Error: Scenario '{}' is missing required files.",
            args.scenario
        );
        if !available.is_empty() {
            eprintln!("Available scenarios: {}", available.join(", "));
        }
        std::process::exit(1);
    }

    // ── Print header ────────────────────────────────────────────

    println!("=== Multi-Provider Pipeline: {} ===", config.title);
    println!("Scenario: {}\n", args.scenario);
    println!("Pipeline: BN Prediction -> Solver Optimization -> CLIPS Safety\n");

    // ================================================================
    // STAGE 1: BN Prediction
    // ================================================================

    println!("--- Stage 1: BN Prediction ---");

    if interactive.step_pause(
        "Loading Bayesian Network and running inference...",
        &[
            "Loads the BIF model and observed evidence",
            "Runs Variable Elimination to compute posterior distributions",
            "Extracts the target prediction for downstream stages",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let stage1_start = Instant::now();

    // Load BIF model
    let model_path_str = model_path.to_str().expect("path is not valid UTF-8");
    let net = BnNetwork::load_file(model_path_str).unwrap_or_else(|e| {
        eprintln!(
            "Error: Failed to load BIF model from {}: {e}",
            model_path.display()
        );
        std::process::exit(1);
    });

    // Load evidence
    let evidence_json = std::fs::read_to_string(&evidence_path).unwrap_or_else(|e| {
        eprintln!("Error: Failed to read {}: {e}", evidence_path.display());
        std::process::exit(1);
    });

    let observed: HashMap<String, String> =
        serde_json::from_str(&evidence_json).unwrap_or_else(|e| {
            eprintln!("Error: Failed to parse {}: {e}", evidence_path.display());
            std::process::exit(1);
        });

    println!("  Model: {}", model_path.display());
    println!("  Evidence:");
    let mut obs_entries: Vec<_> = observed.iter().collect();
    obs_entries.sort_by_key(|(k, _)| k.to_string());
    for (var, state) in &obs_entries {
        println!("    {var} = {state}");
    }

    // Create evidence object
    let mut ev = BnEvidence::create().unwrap_or_else(|e| {
        eprintln!("Error: Failed to create evidence object: {e}");
        std::process::exit(1);
    });

    for (var, state) in &observed {
        if let Err(e) = ev.set_discrete(&net, var, state) {
            eprintln!("  Warning: Failed to set evidence {var}={state}: {e}");
        }
    }

    // Run Variable Elimination
    let bn_result = net.infer(&ev, "ve").unwrap_or_else(|e| {
        eprintln!("Error: BN inference failed: {e}");
        std::process::exit(1);
    });

    // Extract target posterior
    let target_dist = bn_result.query(config.bn_target).unwrap_or_default();

    println!("\n  Posterior P({} | evidence):", config.bn_target);
    let mut dist_entries: Vec<_> = target_dist.iter().collect();
    dist_entries.sort_by(|a, b| b.1.partial_cmp(a.1).unwrap_or(std::cmp::Ordering::Equal));
    let mut top_state = String::new();
    let mut top_prob = 0.0_f64;
    for (state, prob) in &dist_entries {
        let bar_len = ((**prob) * 40.0) as usize;
        let bar: String = "#".repeat(bar_len);
        println!("    {state:15} {prob:.3}  {bar}");
        if **prob > top_prob {
            top_prob = **prob;
            top_state = state.to_string();
        }
    }
    println!(
        "  Prediction: {} = {} (p={:.3})",
        config.bn_target, top_state, top_prob
    );

    // Show all marginals in verbose mode
    if args.verbose {
        let all_variables = net.variables().unwrap_or_default();
        eprintln!("\n  [verbose] All marginals:");
        for var in &all_variables {
            if observed.contains_key(var) {
                continue;
            }
            if let Ok(dist) = bn_result.query(var) {
                let json_str = serde_json::to_string(&dist).unwrap_or_default();
                eprintln!("    {var}: {json_str}");
            }
        }
    }

    let stage1_elapsed = stage1_start.elapsed();
    println!("  Stage 1 time: {}ms\n", stage1_elapsed.as_millis());

    // ================================================================
    // STAGE 2: Solver Optimization
    // ================================================================

    println!("--- Stage 2: Solver Optimization ---");

    if interactive.step_pause(
        "Loading problem and running constraint solver...",
        &[
            "Loads variables, constraints, and objectives from problem.json",
            "Creates a Z3 solver session via the nxusKit Rust SDK",
            "Solves with optimization to find the best assignment",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let stage2_start = Instant::now();

    // Load problem
    let problem_json_str = std::fs::read_to_string(&problem_path).unwrap_or_else(|e| {
        eprintln!("Error: Failed to read {}: {e}", problem_path.display());
        std::process::exit(1);
    });

    let problem: Problem = serde_json::from_str(&problem_json_str).unwrap_or_else(|e| {
        eprintln!("Error: Failed to parse {}: {e}", problem_path.display());
        std::process::exit(1);
    });

    println!("  Problem: {}", problem.name);
    println!(
        "  {} variables, {} constraints",
        problem.variables.len(),
        problem.constraints.len()
    );

    // Create solver session
    let mut session = SolverSession::create(None).unwrap_or_else(|e| {
        eprintln!("Error: Failed to create solver session: {e}");
        std::process::exit(1);
    });

    // Add variables
    let vars: Vec<VariableDef> = serde_json::from_value(serde_json::json!(problem.variables))
        .unwrap_or_else(|e| {
            eprintln!("Error: Failed to parse variables: {e}");
            std::process::exit(1);
        });

    session.add_variables(vars).unwrap_or_else(|e| {
        eprintln!("Error: Failed to add variables: {e}");
        std::process::exit(1);
    });

    // Add constraints
    let constraints_with_params = ensure_parameters(&problem.constraints);
    let constraints: Vec<ConstraintDef> =
        serde_json::from_value(serde_json::json!(constraints_with_params)).unwrap_or_else(|e| {
            eprintln!("Error: Failed to parse constraints: {e}");
            std::process::exit(1);
        });

    session.add_constraints(constraints).unwrap_or_else(|e| {
        eprintln!("Error: Failed to add constraints: {e}");
        std::process::exit(1);
    });

    // Set objective
    if !problem.objectives.is_empty() {
        let first_obj_json = &problem.objectives[0];
        let obj_dir = first_obj_json
            .get("direction")
            .and_then(|d| d.as_str())
            .unwrap_or("maximize");
        let obj_name = first_obj_json
            .get("name")
            .and_then(|n| n.as_str())
            .unwrap_or("unnamed");
        println!("  Objective: {obj_dir} {obj_name}");

        let first_obj: ObjectiveDef = serde_json::from_value(first_obj_json.clone())
            .unwrap_or_else(|e| {
                eprintln!("  Warning: Failed to parse objective: {e}");
                std::process::exit(1);
            });

        if let Err(e) = session.set_objective(first_obj) {
            eprintln!("  Warning: Failed to set objective, solving without optimization: {e}");
        }
    } else {
        println!("  (No objectives, solving for satisfiability)");
    }

    let solve_result = session.solve(None).unwrap_or_else(|e| {
        eprintln!("Error: Solver failed: {e}");
        std::process::exit(1);
    });

    let status = format!("{:?}", solve_result.status).to_lowercase();
    println!("\n  Solver status: {status}");
    println!("  Assignments:");
    print_assignments(&solve_result, args.verbose);

    if let Some(obj_val) = solve_result.objective_value {
        println!("  Objective value: {obj_val}");
    }

    let stage2_elapsed = stage2_start.elapsed();
    println!("  Stage 2 time: {}ms\n", stage2_elapsed.as_millis());

    // ================================================================
    // STAGE 3: CLIPS Safety Enforcement
    // ================================================================

    println!("--- Stage 3: CLIPS Safety Enforcement ---");

    if interactive.step_pause(
        "Loading safety rules and asserting solver assignments...",
        &[
            "Creates a CLIPS environment and loads scenario-specific rules",
            "Asserts facts from solver output using the fact builder API",
            "Runs rule inference to detect safety violations",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let stage3_start = Instant::now();

    let clips_session = ClipsSession::create().unwrap_or_else(|e| {
        eprintln!("Error: Failed to create CLIPS session: {e}");
        std::process::exit(1);
    });

    let rules_path_str = rules_path.to_str().expect("path is not valid UTF-8");
    clips_session.load_file(rules_path_str).unwrap_or_else(|e| {
        eprintln!(
            "Error: Failed to load rules from {}: {e}",
            rules_path.display()
        );
        std::process::exit(1);
    });

    clips_session.reset().unwrap_or_else(|e| {
        eprintln!("Error: Failed to reset CLIPS session: {e}");
        std::process::exit(1);
    });

    println!("  Rules loaded from: {}", rules_path.display());

    let facts_asserted = assert_scenario_facts(
        &clips_session,
        &args.scenario,
        config.assignment_template,
        &solve_result.assignments,
    );
    println!("  Facts asserted: {facts_asserted}");

    let rules_fired = clips_session.run(None).unwrap_or_else(|e| {
        eprintln!("Error: CLIPS inference failed: {e}");
        std::process::exit(1);
    });
    println!("  Rules fired: {rules_fired}");

    let alerts = collect_alerts(&clips_session, config.alert_template);

    if alerts.is_empty() {
        println!("\n  No safety alerts generated. All checks passed.");
    } else {
        // Sort by severity: critical > warning > info
        let mut sorted_alerts = alerts.clone();
        sorted_alerts.sort_by(|a, b| {
            let sev_order = |m: &HashMap<String, ClipsValue>| -> u8 {
                match m.get("severity") {
                    Some(ClipsValue::Symbol(s)) => match s.as_str() {
                        "critical" => 0,
                        "warning" => 1,
                        "info" => 2,
                        _ => 3,
                    },
                    _ => 3,
                }
            };
            sev_order(a).cmp(&sev_order(b))
        });

        println!("\n  Safety alerts ({}):", sorted_alerts.len());
        for alert in &sorted_alerts {
            let severity = match alert.get("severity") {
                Some(ClipsValue::Symbol(s)) => s.as_str(),
                _ => "?",
            };
            let alert_type = match alert.get("alert-type") {
                Some(ClipsValue::Symbol(s)) | Some(ClipsValue::String(s)) => s.as_str(),
                _ => "?",
            };
            let message = match alert.get("message") {
                Some(ClipsValue::String(s)) => s.as_str(),
                _ => "?",
            };
            let rule_name = match alert.get("rule-name") {
                Some(ClipsValue::Symbol(s)) | Some(ClipsValue::String(s)) => s.as_str(),
                _ => "?",
            };

            let sev_marker = match severity {
                "critical" => "[CRITICAL]",
                "warning" => "[WARNING] ",
                "info" => "[INFO]    ",
                _ => "[???]     ",
            };

            println!("    {sev_marker} {alert_type}: {message}");
            if args.verbose {
                eprintln!("      [verbose] rule: {rule_name}");
            }
        }
    }

    if args.verbose {
        if let Ok(indices) = clips_session.facts_by_template(config.assignment_template) {
            eprintln!("\n  [verbose] {} facts:", config.assignment_template);
            for idx in indices {
                if let Ok(slots) = clips_session.fact_slot_values(idx) {
                    let json_str = serde_json::to_string(&slots).unwrap_or_default();
                    eprintln!("    {json_str}");
                }
            }
        }
    }

    let stage3_elapsed = stage3_start.elapsed();
    println!("  Stage 3 time: {}ms\n", stage3_elapsed.as_millis());

    // ================================================================
    // Pipeline Summary
    // ================================================================

    let total_elapsed = total_start.elapsed();

    println!("=== Pipeline Summary ===");
    println!("Scenario:   {} ({})", config.title, args.scenario);
    println!(
        "BN target:  {} = {} (p={:.3})",
        config.bn_target, top_state, top_prob
    );
    println!("Solver:     {status}");
    println!("Alerts:     {} generated", alerts.len());
    println!();
    println!("  Stage 1 (BN):     {:>6}ms", stage1_elapsed.as_millis());
    println!("  Stage 2 (Solver): {:>6}ms", stage2_elapsed.as_millis());
    println!("  Stage 3 (CLIPS):  {:>6}ms", stage3_elapsed.as_millis());
    println!("  Total:            {:>6}ms", total_elapsed.as_millis());

    if args.verbose {
        eprintln!("\n[verbose] Pipeline completed successfully.");
    }
}

// ── Scenario-specific fact assertion ────────────────────────────────

fn assert_scenario_facts(
    env: &ClipsSession,
    scenario: &str,
    template_name: &str,
    assignments: &HashMap<String, SolverValue>,
) -> usize {
    match scenario {
        "festival" => assert_festival_facts(env, template_name, assignments),
        "rescue" => assert_rescue_facts(env, template_name, assignments),
        "bakery" => assert_bakery_facts(env, template_name, assignments),
        _ => 0,
    }
}

/// Festival: assert stage-assignment facts for each band.
fn assert_festival_facts(
    env: &ClipsSession,
    template_name: &str,
    assignments: &HashMap<String, SolverValue>,
) -> usize {
    let bands = [
        ("band_1_stage", "Band Alpha", true, "metal"),
        ("band_2_stage", "Band Beta", false, "wood"),
        ("band_3_stage", "Band Gamma", true, "concrete"),
        ("band_4_stage", "Band Delta", false, "metal"),
        ("band_5_stage", "Band Epsilon", false, "wood"),
    ];

    let mut count = 0;

    for (var_name, band_name, has_pyro, stage_material) in &bands {
        let stage_id = assignment_int(assignments, var_name);
        let crowd_key = format!("stage_{stage_id}_crowd");
        let predicted_crowd = assignment_int(assignments, &crowd_key);

        let mut slots = HashMap::new();
        slots.insert("stage-id".into(), ClipsValue::Integer(stage_id));
        slots.insert("band-name".into(), ClipsValue::String((*band_name).into()));
        slots.insert(
            "predicted-crowd".into(),
            ClipsValue::Integer(predicted_crowd),
        );
        slots.insert(
            "has-pyro".into(),
            ClipsValue::Symbol((if *has_pyro { "yes" } else { "no" }).into()),
        );
        slots.insert(
            "stage-material".into(),
            ClipsValue::Symbol((*stage_material).into()),
        );

        match env.fact_assert_structured(template_name, &slots) {
            Ok(_) => count += 1,
            Err(e) => eprintln!("  Warning: Failed to assert fact for {band_name}: {e}"),
        }
    }
    count
}

/// Rescue: assert rescue-assignment facts for each team.
fn assert_rescue_facts(
    env: &ClipsSession,
    template_name: &str,
    assignments: &HashMap<String, SolverValue>,
) -> usize {
    let teams = [
        ("team_1_zone", 1_i64, "ground", 15_i64, "urban"),
        ("team_2_zone", 2, "ground", 20, "rural"),
        ("team_3_zone", 3, "helicopter", 45, "mountainous"),
        ("team_4_zone", 4, "drone", 10, "rural"),
    ];

    let mut count = 0;

    for (var_name, team_id, team_type, wind_speed, zone_terrain) in &teams {
        let zone_id = assignment_int(assignments, var_name);

        let mut slots = HashMap::new();
        slots.insert("team-id".into(), ClipsValue::Integer(*team_id));
        slots.insert("zone-id".into(), ClipsValue::Integer(zone_id));
        slots.insert("team-type".into(), ClipsValue::Symbol((*team_type).into()));
        slots.insert("wind-speed".into(), ClipsValue::Integer(*wind_speed));
        slots.insert(
            "zone-terrain".into(),
            ClipsValue::Symbol((*zone_terrain).into()),
        );

        match env.fact_assert_structured(template_name, &slots) {
            Ok(_) => count += 1,
            Err(e) => eprintln!("  Warning: Failed to assert fact for team {team_id}: {e}"),
        }
    }
    count
}

/// Bakery: assert baking-assignment facts for each item.
fn assert_bakery_facts(
    env: &ClipsSession,
    template_name: &str,
    assignments: &HashMap<String, SolverValue>,
) -> usize {
    let items = [
        ("item_1", "Sourdough Loaf", true, false, "none"),
        ("item_2", "Almond Croissant", true, true, "nuts"),
        ("item_3", "Rye Bread", true, false, "gluten"),
        ("item_4", "Walnut Brownie", false, true, "nuts"),
        ("item_5", "GF Muffin", false, false, "none"),
        ("item_6", "Focaccia", true, false, "gluten"),
    ];

    let mut count = 0;

    for (prefix, item_name, contains_gluten, contains_nuts, oven_last_allergen) in &items {
        let oven_key = format!("{prefix}_oven");
        let slot_key = format!("{prefix}_slot");
        let oven_id = assignment_int(assignments, &oven_key);
        let time_slot = assignment_int(assignments, &slot_key);

        let mut slots = HashMap::new();
        slots.insert("item-name".into(), ClipsValue::String((*item_name).into()));
        slots.insert("oven-id".into(), ClipsValue::Integer(oven_id));
        slots.insert("time-slot".into(), ClipsValue::Integer(time_slot));
        slots.insert(
            "contains-nuts".into(),
            ClipsValue::Symbol((if *contains_nuts { "yes" } else { "no" }).into()),
        );
        slots.insert(
            "contains-gluten".into(),
            ClipsValue::Symbol((if *contains_gluten { "yes" } else { "no" }).into()),
        );
        slots.insert(
            "oven-last-allergen".into(),
            ClipsValue::Symbol((*oven_last_allergen).into()),
        );

        match env.fact_assert_structured(template_name, &slots) {
            Ok(_) => count += 1,
            Err(e) => eprintln!("  Warning: Failed to assert fact for {item_name}: {e}"),
        }
    }
    count
}

// ── CLIPS alert collection ──────────────────────────────────────────

/// Collect all alert facts from the given CLIPS template.
fn collect_alerts(env: &ClipsSession, alert_template: &str) -> Vec<HashMap<String, ClipsValue>> {
    let mut alerts = Vec::new();
    let Ok(indices) = env.facts_by_template(alert_template) else {
        return alerts;
    };
    for idx in indices {
        if let Ok(slots) = env.fact_slot_values(idx) {
            alerts.push(slots);
        }
    }
    alerts
}
