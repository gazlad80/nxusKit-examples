//! BN Structure Learning Example (E6) -- nxusKit SDK
//!
//! Demonstrates causal structure discovery from CSV data using two algorithms:
//!   1. Hill-Climb with BDeu scoring -- greedy structure search
//!   2. K2 with BDeu scoring -- order-based structure search
//!
//! After discovering structure, learns parameters via MLE, evaluates model fit
//! with log-likelihood, and runs inference on the learned model.
//!
//! Usage:
//!   cargo run -- --scenario golf [--verbose] [--step]

use std::path::PathBuf;
use std::time::Instant;

use clap::Parser;
use nxuskit::bn::{BnEvidence, BnNetwork};

// ── CLI Arguments ───────────────────────────────────────────────────

#[derive(Parser, Debug)]
#[command(name = "bn-structure-learning-example")]
#[command(about = "Demonstrates BN structure learning from CSV data")]
struct Args {
    /// Scenario to run: golf, bmx, or sourdough
    #[arg(short = 'S', long, default_value = "golf")]
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
    /// Sample evidence for inference demo: (variable, state) pairs
    evidence: &'static [(&'static str, &'static str)],
    /// Query variable for inference
    query_var: &'static str,
}

fn scenario_config(name: &str) -> Option<ScenarioConfig> {
    match name {
        "golf" => Some(ScenarioConfig {
            title: "Golf Course Conditions",
            evidence: &[("weather", "rainy"), ("fertilizer", "heavy")],
            query_var: "green_speed",
        }),
        "bmx" => Some(ScenarioConfig {
            title: "BMX Rider Performance",
            evidence: &[("skill", "pro"), ("pump_timing", "perfect")],
            query_var: "lap_time",
        }),
        "sourdough" => Some(ScenarioConfig {
            title: "Sourdough Baking",
            evidence: &[("flour_type", "rye"), ("ambient_temp", "warm")],
            query_var: "flavor_profile",
        }),
        _ => None,
    }
}

// ── Helpers ─────────────────────────────────────────────────────────

/// Discover available scenario directories (those containing data.csv).
fn available_scenarios(scenarios_dir: &std::path::Path) -> Vec<String> {
    let mut names = Vec::new();
    if let Ok(entries) = std::fs::read_dir(scenarios_dir) {
        for entry in entries.flatten() {
            if entry.file_type().is_ok_and(|ft| ft.is_dir()) {
                let dir = entry.path();
                if dir.join("data.csv").exists()
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

/// Parse CSV header to get column names and row count.
fn csv_info(path: &std::path::Path) -> (Vec<String>, usize) {
    let content = std::fs::read_to_string(path).unwrap_or_default();
    let mut lines = content.lines();
    let columns: Vec<String> = lines
        .next()
        .unwrap_or("")
        .split(',')
        .map(|s| s.trim().to_string())
        .collect();
    let row_count = lines.filter(|l| !l.trim().is_empty()).count();
    (columns, row_count)
}

/// Print edges from structure learning result JSON.
fn print_edges(result: &serde_json::Value) {
    if let Some(edges) = result.get("edges").and_then(|e| e.as_array()) {
        if edges.is_empty() {
            println!("    (no edges discovered)");
        } else {
            for edge in edges {
                let from = edge.get("from").and_then(|v| v.as_str()).unwrap_or("?");
                let to = edge.get("to").and_then(|v| v.as_str()).unwrap_or("?");
                println!("    {from} -> {to}");
            }
        }
    }
}

/// Extract edges from a structure learning result for comparison.
fn extract_edges(result: &Option<serde_json::Value>) -> Vec<(String, String)> {
    let mut edges = Vec::new();
    if let Some(r) = result
        && let Some(arr) = r.get("edges").and_then(|e| e.as_array())
    {
        for edge in arr {
            let from = edge.get("from").and_then(|v| v.as_str()).unwrap_or("?");
            let to = edge.get("to").and_then(|v| v.as_str()).unwrap_or("?");
            edges.push((from.to_string(), to.to_string()));
        }
    }
    edges.sort();
    edges
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
        PathBuf::from("examples/integrations/bn-structure-learning/scenarios"),
    ];

    let scenarios_dir = candidate_bases
        .iter()
        .find(|p| p.exists())
        .cloned()
        .unwrap_or_else(|| {
            eprintln!("Error: Could not locate scenarios directory.");
            eprintln!(
                "Run from the bn-structure-learning/rust/ directory or pass --scenario with a valid name."
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
    let csv_path = scenario_dir.join("data.csv");

    if !csv_path.exists() {
        let available = available_scenarios(&scenarios_dir);
        eprintln!("Error: Scenario '{}' is missing data.csv.", args.scenario);
        if !available.is_empty() {
            eprintln!("Available scenarios: {}", available.join(", "));
        }
        std::process::exit(1);
    }

    // ── Print header ────────────────────────────────────────────

    println!("=== BN Structure Learning: {} ===", config.title);
    println!("Scenario: {}\n", args.scenario);

    // ================================================================
    // Step 1: Load CSV Data
    // ================================================================

    println!("--- Step 1: Load CSV Data ---");

    if interactive.step_pause(
        "Loading and inspecting CSV data...",
        &[
            "Reads the CSV file to discover column names and row count",
            "Column names become Bayesian Network variable names",
            "Row count affects structure learning quality",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let (columns, row_count) = csv_info(&csv_path);
    let csv_path_str = csv_path.to_str().expect("path is not valid UTF-8");

    println!("  File: {}", csv_path.display());
    println!("  Columns: {}", columns.len());
    for col in &columns {
        println!("    - {col}");
    }
    println!("  Rows: {row_count}");

    // Check for empty dataset
    if row_count == 0 {
        eprintln!("Error: CSV file has no data rows.");
        std::process::exit(1);
    }

    println!();

    // ================================================================
    // Step 2: Hill-Climb + BDeu Structure Learning
    // ================================================================

    println!("--- Step 2: Hill-Climb + BDeu Structure Learning ---");

    if interactive.step_pause(
        "Running Hill-Climb search with BDeu scoring...",
        &[
            "Hill-Climb is a greedy search that adds, removes, or reverses edges",
            "BDeu (Bayesian Dirichlet equivalent uniform) penalizes model complexity",
            "Discovers causal structure from observed data correlations",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let step2_start = Instant::now();

    // Create an empty network for structure learning
    let mut hc_net = BnNetwork::create().unwrap_or_else(|e| {
        eprintln!("Error: Failed to create BN network: {e}");
        std::process::exit(1);
    });

    let hc_result = match hc_net.search_structure(
        csv_path_str,
        "hill_climb",
        "bdeu",
        5,    // max_parents
        1000, // max_steps
        1.0,  // ess (equivalent sample size for BDeu)
        None, // ordering (unused for hill_climb)
    ) {
        Ok(result) => Some(result),
        Err(e) => {
            eprintln!("  Warning: Hill-Climb structure learning failed: {e}");
            None
        }
    };

    let step2_elapsed = step2_start.elapsed();

    if let Some(ref result) = hc_result {
        let edge_count = result
            .get("edges")
            .and_then(|e| e.as_array())
            .map_or(0, |a| a.len());
        let score = result.get("score").and_then(|s| s.as_f64()).unwrap_or(0.0);
        let iterations = result
            .get("iterations")
            .and_then(|i| i.as_u64())
            .unwrap_or(0);

        println!("  Algorithm: Hill-Climb");
        println!("  Scoring: BDeu");
        println!("  Edges discovered: {edge_count}");
        println!("  Score: {score:.2}");
        println!("  Iterations: {iterations}");
        println!("  Discovered edges:");
        print_edges(result);

        if args.verbose {
            eprintln!(
                "\n  [verbose] Raw HC result: {}",
                serde_json::to_string_pretty(result).unwrap_or_default()
            );
        }
    }

    println!("  Time: {}ms\n", step2_elapsed.as_millis());

    // ================================================================
    // Step 3: K2 Structure Learning
    // ================================================================

    println!("--- Step 3: K2 + BDeu Structure Learning ---");

    if interactive.step_pause(
        "Running K2 search with BDeu scoring...",
        &[
            "K2 is an order-based algorithm that requires a variable ordering",
            "BDeu (Bayesian Dirichlet equivalent uniform) is a Bayesian score",
            "Compares structure to Hill-Climb to assess algorithm sensitivity",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let step3_start = Instant::now();

    let k2_net = BnNetwork::create().unwrap_or_else(|e| {
        eprintln!("Error: Failed to create BN network for K2: {e}");
        std::process::exit(1);
    });

    // Use the CSV column order as K2 ordering
    let ordering_json = serde_json::to_string(&columns).expect("JSON serialization failed");

    let k2_result = match k2_net.search_structure(
        csv_path_str,
        "k2",
        "bdeu",
        3,    // max_parents
        0,    // max_steps (unused for K2)
        10.0, // ess for BDeu
        Some(&ordering_json),
    ) {
        Ok(result) => Some(result),
        Err(e) => {
            eprintln!("  Warning: K2 structure learning failed: {e}");
            None
        }
    };

    let step3_elapsed = step3_start.elapsed();

    if let Some(ref result) = k2_result {
        let edge_count = result
            .get("edges")
            .and_then(|e| e.as_array())
            .map_or(0, |a| a.len());
        let score = result.get("score").and_then(|s| s.as_f64()).unwrap_or(0.0);

        println!("  Algorithm: K2");
        println!("  Scoring: BDeu (ESS=10.0)");
        println!("  Edges discovered: {edge_count}");
        println!("  Score: {score:.2}");
        println!("  Discovered edges:");
        print_edges(result);

        if args.verbose {
            eprintln!(
                "\n  [verbose] Raw K2 result: {}",
                serde_json::to_string_pretty(result).unwrap_or_default()
            );
        }
    }

    println!("  Time: {}ms\n", step3_elapsed.as_millis());

    // ================================================================
    // Step 4: Parameter Learning (MLE) on Hill-Climb Structure
    // ================================================================

    println!("--- Step 4: MLE Parameter Learning ---");

    if interactive.step_pause(
        "Learning CPT parameters via Maximum Likelihood Estimation...",
        &[
            "Uses the Hill-Climb discovered structure as the model skeleton",
            "MLE estimates conditional probability tables from the CSV data",
            "Laplace smoothing (pseudocount=1.0) prevents zero-probability entries",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let step4_start = Instant::now();

    let mle_ok = hc_net.learn_mle(csv_path_str, 1.0).is_ok();

    let step4_elapsed = step4_start.elapsed();

    if mle_ok {
        let num_vars = hc_net.num_variables();
        println!("  Structure: Hill-Climb (from Step 2)");
        println!("  Pseudocount: 1.0 (Laplace smoothing)");
        println!("  Variables with learned CPTs: {num_vars}");
        println!("  Status: OK");
    } else {
        eprintln!("  Warning: MLE parameter learning failed.");
    }

    println!("  Time: {}ms\n", step4_elapsed.as_millis());

    // ================================================================
    // Step 5: Model Fit Evaluation
    // ================================================================

    println!("--- Step 5: Log-Likelihood Fit Evaluation ---");

    if interactive.step_pause(
        "Computing log-likelihood to evaluate model fit...",
        &[
            "Log-likelihood measures how well the model explains the observed data",
            "Higher (less negative) values indicate better fit",
            "Normalized per-sample LL allows comparison across datasets",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let step5_start = Instant::now();

    let ll = hc_net
        .log_likelihood(csv_path_str)
        .unwrap_or(f64::NEG_INFINITY);

    let step5_elapsed = step5_start.elapsed();

    let per_sample = if row_count > 0 {
        ll / row_count as f64
    } else {
        f64::NEG_INFINITY
    };

    println!("  Log-likelihood: {ll:.4}");
    println!("  Per-sample LL: {per_sample:.4}");
    println!("  Data rows: {row_count}");

    if ll.is_finite() {
        println!("  Status: OK");
    } else {
        println!("  Status: WARNING (non-finite log-likelihood)");
    }

    println!("  Time: {}ms\n", step5_elapsed.as_millis());

    // ================================================================
    // Step 6: Inference on Learned Model
    // ================================================================

    println!("--- Step 6: Inference on Learned Model ---");

    if interactive.step_pause(
        "Running Variable Elimination on the learned model...",
        &[
            "Sets sample evidence to test the learned model",
            "Variable Elimination computes exact posteriors",
            "Demonstrates that the learned model supports standard BN queries",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let step6_start = Instant::now();

    // Create evidence
    let mut ev = BnEvidence::create().unwrap_or_else(|e| {
        eprintln!("Error: Failed to create evidence object: {e}");
        std::process::exit(1);
    });

    println!("  Evidence:");
    for (var, state) in config.evidence {
        match ev.set_discrete(&hc_net, var, state) {
            Ok(()) => println!("    {var} = {state}"),
            Err(e) => eprintln!("    Warning: Failed to set evidence {var}={state}: {e}"),
        }
    }

    // Run VE inference
    match hc_net.infer(&ev, "ve") {
        Ok(bn_result) => {
            if let Ok(result_json) = bn_result.to_json()
                && let Ok(parsed) = serde_json::from_str::<serde_json::Value>(&result_json)
            {
                println!("\n  Query: P({} | evidence)", config.query_var);

                // Extract the query variable's distribution
                if let Some(marginals) = parsed.get("marginals").and_then(|m| m.as_object())
                    && let Some(dist) = marginals.get(config.query_var).and_then(|d| d.as_object())
                {
                    let mut entries: Vec<_> = dist.iter().collect();
                    entries.sort_by(|a, b| {
                        let va = a.1.as_f64().unwrap_or(0.0);
                        let vb = b.1.as_f64().unwrap_or(0.0);
                        vb.partial_cmp(&va).unwrap_or(std::cmp::Ordering::Equal)
                    });

                    for (state, prob) in &entries {
                        let p = prob.as_f64().unwrap_or(0.0);
                        let bar_len = (p * 40.0) as usize;
                        let bar: String = "#".repeat(bar_len);
                        println!("    {state:15} {p:.4}  {bar}");
                    }
                }

                if args.verbose {
                    eprintln!(
                        "\n  [verbose] Full inference result:\n  {}",
                        serde_json::to_string_pretty(&parsed).unwrap_or_default()
                    );
                }
            }
        }
        Err(e) => {
            eprintln!("  Warning: Inference failed: {e}");
        }
    }

    let step6_elapsed = step6_start.elapsed();
    println!("  Time: {}ms\n", step6_elapsed.as_millis());

    // ================================================================
    // Step 7: Algorithm Comparison
    // ================================================================

    println!("--- Step 7: Structure Comparison ---");

    if interactive.step_pause(
        "Comparing Hill-Climb and K2 discovered structures...",
        &[
            "Counts edges unique to each algorithm and shared edges",
            "Different algorithms may discover different causal relationships",
            "Shared edges are more likely to represent true causal links",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let hc_edges = extract_edges(&hc_result);
    let k2_edges = extract_edges(&k2_result);

    let shared: Vec<_> = hc_edges.iter().filter(|e| k2_edges.contains(e)).collect();
    let hc_only: Vec<_> = hc_edges.iter().filter(|e| !k2_edges.contains(e)).collect();
    let k2_only: Vec<_> = k2_edges.iter().filter(|e| !hc_edges.contains(e)).collect();

    println!("  Hill-Climb edges: {}", hc_edges.len());
    println!("  K2 edges: {}", k2_edges.len());
    println!("  Shared edges: {}", shared.len());

    if !shared.is_empty() {
        println!("  Shared (high-confidence causal links):");
        for (from, to) in &shared {
            println!("    {from} -> {to}");
        }
    }

    if !hc_only.is_empty() {
        println!("  Hill-Climb only:");
        for (from, to) in &hc_only {
            println!("    {from} -> {to}");
        }
    }

    if !k2_only.is_empty() {
        println!("  K2 only:");
        for (from, to) in &k2_only {
            println!("    {from} -> {to}");
        }
    }

    println!();

    // ================================================================
    // Summary
    // ================================================================

    let total_elapsed = total_start.elapsed();

    println!("=== Summary ===");
    println!("Scenario:         {} ({})", config.title, args.scenario);
    println!(
        "Data:             {} rows x {} columns",
        row_count,
        columns.len()
    );
    println!("HC edges:         {}", hc_edges.len());
    println!("K2 edges:         {}", k2_edges.len());
    println!("Shared edges:     {}", shared.len());
    println!("Log-likelihood:   {ll:.4}");
    println!("Per-sample LL:    {per_sample:.4}");
    println!("Total time:       {}ms", total_elapsed.as_millis());
}
