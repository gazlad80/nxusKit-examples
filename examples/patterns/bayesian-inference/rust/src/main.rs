//! Bayesian Inference Example — nxusKit SDK
//!
//! Demonstrates Bayesian Network inference using the nxusKit Rust SDK:
//!   1. Variable Elimination (exact)
//!   2. Junction Tree (exact)
//!   3. Loopy Belief Propagation (approximate)
//!   4. Gibbs Sampling (approximate, MCMC)
//!
//! Compares posterior marginals across all four algorithms and prints a summary.
//!
//! Usage:
//!   cargo run -- --scenario haunted-house [--verbose] [--step]

use std::collections::HashMap;
use std::path::PathBuf;

use clap::Parser;
use nxuskit::bn::{BnEvidence, BnNetwork, BnResult};

/// Per-variable marginal: variable name → list of (state, probability).
type Marginals = Vec<(String, Vec<(String, f64)>)>;

// ── CLI Arguments ───────────────────────────────────────────────────

#[derive(Parser, Debug)]
#[command(name = "bayesian-inference-example")]
#[command(about = "Demonstrates nxusKit Bayesian Network inference across four algorithms")]
struct Args {
    /// Scenario to analyse: haunted-house, coffee-shop, or plant-doctor
    #[arg(short = 'S', long, default_value = "haunted-house")]
    scenario: String,

    /// Enable verbose output showing raw JSON payloads
    #[arg(short, long)]
    verbose: bool,

    /// Enable step-through mode with explanations at each phase
    #[arg(short, long)]
    step: bool,
}

// ── Evidence JSON Schema ────────────────────────────────────────────

/// evidence.json is a flat map: variable_name -> observed_state
type EvidenceMap = HashMap<String, String>;

// ── Helpers ─────────────────────────────────────────────────────────

/// Discover available scenario directories (those containing model.bif).
fn available_scenarios(scenarios_dir: &std::path::Path) -> Vec<String> {
    let mut names = Vec::new();
    if let Ok(entries) = std::fs::read_dir(scenarios_dir) {
        for entry in entries.flatten() {
            if entry.file_type().is_ok_and(|ft| ft.is_dir()) {
                let bif = entry.path().join("model.bif");
                if bif.exists()
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

/// Human-readable title from a scenario slug.
fn scenario_title(slug: &str) -> &str {
    match slug {
        "haunted-house" => "Haunted House Investigation",
        "coffee-shop" => "Coffee Shop Diagnostics",
        "plant-doctor" => "Plant Doctor Diagnosis",
        _ => slug,
    }
}

/// Extract marginals for unobserved variables from a BnResult.
fn extract_marginals(
    result: &BnResult,
    all_variables: &[String],
    observed: &EvidenceMap,
) -> Vec<(String, Vec<(String, f64)>)> {
    let mut marginals = Vec::new();

    for var in all_variables {
        if observed.contains_key(var) {
            continue;
        }
        let dist = match result.query(var) {
            Ok(d) => d,
            Err(_) => continue,
        };
        let mut pairs: Vec<(String, f64)> = dist.into_iter().collect();
        pairs.sort_by(|a, b| a.0.cmp(&b.0));
        marginals.push((var.clone(), pairs));
    }

    marginals.sort_by(|a, b| a.0.cmp(&b.0));
    marginals
}

/// Print posterior marginals in a formatted table.
fn print_marginals(marginals: &[(String, Vec<(String, f64)>)]) {
    println!("Posterior marginals:");
    for (var, pairs) in marginals {
        let dist_str: Vec<String> = pairs.iter().map(|(s, p)| format!("{s}={p:.3}")).collect();
        println!("  {var:20} {}", dist_str.join("  "));
    }
}

/// Extract the elapsed_ms from the full result JSON.
fn extract_elapsed_ms(result: &BnResult) -> f64 {
    let Ok(json_str) = result.to_json() else {
        return 0.0;
    };
    let val: serde_json::Value = serde_json::from_str(&json_str).unwrap_or_default();
    val.get("elapsed_ms")
        .and_then(|v| v.as_f64())
        .unwrap_or(0.0)
}

// ── Algorithm runner ────────────────────────────────────────────────

/// Configuration for a single inference run.
struct AlgoRun<'a> {
    label: &'a str,
    algo_code: &'a str,
    description: &'a str,
    exact: bool,
    num_samples: u32,
    burn_in: u32,
    seed: u64,
}

/// Run inference with the given algorithm, print results, and return marginals for comparison.
fn run_algorithm(
    run: &AlgoRun<'_>,
    net: &BnNetwork,
    ev: &BnEvidence,
    all_variables: &[String],
    observed: &EvidenceMap,
    interactive: &mut nxuskit_examples_interactive::InteractiveConfig,
    verbose: bool,
) -> Option<Marginals> {
    let kind = if run.exact { "exact" } else { "approximate" };
    let suffix = if run.num_samples > 0 {
        format!(", {} samples", run.num_samples)
    } else {
        String::new()
    };
    println!("--- {} ({}{}) ---", run.label, kind, suffix);

    if interactive.step_pause(&format!("Running {}...", run.label), &[run.description])
        == nxuskit_examples_interactive::StepAction::Quit
    {
        return None;
    }

    let result = if run.num_samples > 0 || run.burn_in > 0 || run.seed > 0 {
        net.infer_with_options(ev, run.algo_code, run.num_samples, run.burn_in, run.seed)
    } else {
        net.infer(ev, run.algo_code)
    };

    match result {
        Ok(bn_result) => {
            let elapsed_ms = extract_elapsed_ms(&bn_result);
            let marginals = extract_marginals(&bn_result, all_variables, observed);

            if verbose && let Ok(json_str) = bn_result.to_json() {
                eprintln!("[verbose] {}: {json_str}", run.algo_code);
            }

            print_marginals(&marginals);
            println!("Inference time: {elapsed_ms:.1}ms\n");

            Some(marginals)
        }
        Err(e) => {
            eprintln!("Warning: {} failed: {e}", run.label);
            println!();
            Some(Vec::new())
        }
    }
}

// ── Comparison table ────────────────────────────────────────────────

/// Print a comparison table across all algorithms.
fn print_comparison(algo_labels: &[&str], results: &[Marginals]) {
    println!("=== Algorithm Comparison ===");

    // Build header
    let algo_header: Vec<String> = algo_labels.iter().map(|l| format!("{l:>8}")).collect();
    println!("{:25} | {}", "Variable", algo_header.join(" | "));
    println!("{}", "-".repeat(25 + 3 + algo_labels.len() * 11));

    // Collect all variable+state keys from the first non-empty result
    let Some(reference) = results.iter().find(|r| !r.is_empty()) else {
        println!("(no results to compare)");
        return;
    };

    for (var, pairs) in reference {
        for (state, _) in pairs {
            let key = format!("{var} ({state})");
            let values: Vec<String> = results
                .iter()
                .map(|r| {
                    r.iter()
                        .find(|(v, _)| v == var)
                        .and_then(|(_, ps)| ps.iter().find(|(s, _)| s == state))
                        .map(|(_, p)| format!("{p:>8.3}"))
                        .unwrap_or_else(|| "     N/A".to_string())
                })
                .collect();
            println!("{key:25} | {}", values.join(" | "));
        }
    }
}

// ── Main ────────────────────────────────────────────────────────────

fn main() {
    let args = Args::parse();
    let mut interactive =
        nxuskit_examples_interactive::InteractiveConfig::new(args.verbose, args.step);

    // ── Locate scenario directory ───────────────────────────────

    let exe_dir = std::env::current_exe()
        .ok()
        .and_then(|p| p.parent().map(|d| d.to_path_buf()))
        .unwrap_or_else(|| PathBuf::from("."));

    let candidate_bases = [
        exe_dir.join("../scenarios"),
        PathBuf::from("../scenarios"),
        PathBuf::from("scenarios"),
        PathBuf::from("examples/patterns/bayesian-inference/scenarios"),
    ];

    let scenarios_dir = candidate_bases
        .iter()
        .find(|p| p.exists())
        .cloned()
        .unwrap_or_else(|| {
            eprintln!("Error: Could not locate scenarios directory.");
            eprintln!("Run from the bayesian-inference/rust/ directory or pass --scenario with a valid name.");
            std::process::exit(1);
        });

    let scenario_dir = scenarios_dir.join(&args.scenario);
    let model_path = scenario_dir.join("model.bif");
    let evidence_path = scenario_dir.join("evidence.json");

    if !model_path.exists() {
        let available = available_scenarios(&scenarios_dir);
        eprintln!("Error: Unknown scenario '{}'.", args.scenario);
        if available.is_empty() {
            eprintln!("No scenarios found in {}", scenarios_dir.display());
        } else {
            eprintln!("Available scenarios: {}", available.join(", "));
        }
        std::process::exit(1);
    }

    // ── Load evidence ──────────────────────────────────────────

    let evidence_json = std::fs::read_to_string(&evidence_path).unwrap_or_else(|e| {
        eprintln!("Error: Failed to read {}: {e}", evidence_path.display());
        std::process::exit(1);
    });

    let observed: EvidenceMap = serde_json::from_str(&evidence_json).unwrap_or_else(|e| {
        eprintln!("Error: Failed to parse {}: {e}", evidence_path.display());
        std::process::exit(1);
    });

    // ── Load BIF model ─────────────────────────────────────────

    if interactive.step_pause(
        "Loading Bayesian Network from BIF model...",
        &[
            "BIF (Bayesian Interchange Format) defines variables, states, and CPTs",
            "The SDK parses the file and builds an internal network representation",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let model_path_str = model_path.to_str().expect("path is not valid UTF-8");
    let net = BnNetwork::load_file(model_path_str).unwrap_or_else(|e| {
        eprintln!(
            "Error: Failed to load BIF model from {}: {e}",
            model_path.display()
        );
        std::process::exit(1);
    });

    // ── Query network metadata ─────────────────────────────────

    let num_variables = net.num_variables();
    let all_variables = net.variables().unwrap_or_default();

    // ── Print header ───────────────────────────────────────────

    let title = scenario_title(&args.scenario);
    println!("=== Bayesian Inference: {} ===", title);
    println!(
        "Network: {} variables, {} with evidence\n",
        num_variables,
        observed.len()
    );

    println!("Evidence observed:");
    let mut obs_entries: Vec<_> = observed.iter().collect();
    obs_entries.sort_by_key(|(k, _)| k.to_string());
    for (var, state) in &obs_entries {
        println!("  {var} = {state}");
    }
    println!();

    // ── Create evidence object ─────────────────────────────────

    if interactive.step_pause(
        "Setting observed evidence...",
        &[
            "Evidence constrains the network: observed variables are fixed to known values",
            "Inference then computes posterior distributions for unobserved variables",
        ],
    ) == nxuskit_examples_interactive::StepAction::Quit
    {
        return;
    }

    let mut ev = BnEvidence::create().unwrap_or_else(|e| {
        eprintln!("Error: Failed to create evidence object: {e}");
        std::process::exit(1);
    });

    for (var, state) in &observed {
        if let Err(e) = ev.set_discrete(&net, var, state) {
            eprintln!("Warning: Failed to set evidence {var}={state}: {e}");
        }
    }

    // ── Run four inference algorithms ──────────────────────────

    let algorithms = [
        AlgoRun {
            label: "Step 1: Variable Elimination",
            algo_code: "ve",
            description: "Exact inference by summing out variables in an optimal elimination order",
            exact: true,
            num_samples: 0,
            burn_in: 0,
            seed: 0,
        },
        AlgoRun {
            label: "Step 2: Junction Tree",
            algo_code: "jt",
            description: "Exact inference via message passing on a clique tree (join tree)",
            exact: true,
            num_samples: 0,
            burn_in: 0,
            seed: 0,
        },
        AlgoRun {
            label: "Step 3: Loopy Belief Propagation",
            algo_code: "lbp",
            description: "Approximate inference via iterative message passing on the factor graph",
            exact: false,
            num_samples: 0,
            burn_in: 0,
            seed: 0,
        },
        AlgoRun {
            label: "Step 4: Gibbs Sampling",
            algo_code: "gibbs",
            description: "Approximate MCMC inference — samples from the joint distribution via conditional re-sampling",
            exact: false,
            num_samples: 10_000,
            burn_in: 1_000,
            seed: 42,
        },
    ];

    let algo_labels: Vec<&str> = vec!["VE", "JT", "LBP", "Gibbs"];
    let mut all_results: Vec<Marginals> = Vec::new();
    let mut quit = false;

    for run in &algorithms {
        match run_algorithm(
            run,
            &net,
            &ev,
            &all_variables,
            &observed,
            &mut interactive,
            args.verbose,
        ) {
            Some(marginals) => all_results.push(marginals),
            None => {
                quit = true;
                break;
            }
        }
    }

    // ── Comparison summary ─────────────────────────────────────

    if !quit && all_results.len() == algorithms.len() {
        print_comparison(&algo_labels, &all_results);
    }

    if !quit {
        println!("\nDone. All four inference algorithms completed successfully.");
    }
}
