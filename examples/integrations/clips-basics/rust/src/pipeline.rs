//! Example: CLIPS Pipeline Runner - Multi-Stage Rule Processing
//!
//! This example demonstrates how to run CLIPS rulebases as pipeline stages,
//! where output from one stage becomes input for the next.
//!
//! Two pipelines are demonstrated:
//!
//! 1. **Order Processing Pipeline** (3-stage linear):
//!    validation → pricing → fulfillment
//!
//! 2. **Incident Response Pipeline** (branching):
//!    detection → classification → security/operations/escalation (based on route-to)
//!
//! The pipeline convention uses a `pipeline-item` envelope that tracks:
//! - item-id: unique identifier
//! - stage: current pipeline stage
//! - status: pending/processing/completed/failed/routed/skipped
//! - route-to: next stage (for branching pipelines)
//!
//! Run with: cargo run --example clips_pipeline --features clips

// Clippy allowances for example code - prioritize readability over optimization
#![allow(clippy::collapsible_if)]
#![allow(clippy::map_clone)]
#![allow(clippy::useless_vec)]

use nxuskit::{ChatRequest, Message, NxuskitProvider, ProviderConfig, ThinkingMode};
use std::collections::HashMap;
use std::fs;
use std::path::Path;

/// Configuration for a pipeline stage
struct PipelineStage {
    name: &'static str,
    rules_file: &'static str,
    /// Maps route-to values to next stage names (for branching pipelines)
    routes: HashMap<&'static str, &'static str>,
}

/// Result of running a pipeline stage
#[derive(Debug)]
struct StageResult {
    #[allow(dead_code)] // Kept for Debug output and potential future use
    stage_name: String,
    status: String,
    route_to: Option<String>,
    output_facts: Vec<serde_json::Value>,
    #[allow(dead_code)] // Kept for Debug output and potential future use
    thinking: Option<String>,
}

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("╔════════════════════════════════════════════════════════════════════╗");
    println!("║            CLIPS Pipeline Runner - Multi-Stage Processing          ║");
    println!("╚════════════════════════════════════════════════════════════════════╝\n");

    // Create CLIPS provider via NxuskitProvider pointing to pipeline rules directory
    let provider = NxuskitProvider::new(ProviderConfig {
        provider_type: "clips".to_string(),
        model: Some("../../../shared/rules/pipeline".to_string()),
        ..Default::default()
    })?;

    // =========================================================================
    // Pipeline 1: Order Processing (Linear 3-Stage)
    // =========================================================================
    println!("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
    println!("  PIPELINE 1: Order Processing (validation → pricing → fulfillment)");
    println!("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n");

    // Load order scenarios
    let orders_path = Path::new("../../../shared/data/pipeline/orders.json");
    let orders_data: serde_json::Value = serde_json::from_str(&fs::read_to_string(orders_path)?)?;

    // Define the order pipeline stages
    let order_stages = vec![
        PipelineStage {
            name: "validation",
            rules_file: "order-validation.clp",
            routes: [("pricing", "pricing"), ("fulfillment", "fulfillment")]
                .into_iter()
                .collect(),
        },
        PipelineStage {
            name: "pricing",
            rules_file: "order-pricing.clp",
            routes: [("fulfillment", "fulfillment")].into_iter().collect(),
        },
        PipelineStage {
            name: "fulfillment",
            rules_file: "order-fulfillment.clp",
            routes: HashMap::new(), // Terminal stage
        },
    ];

    // Run a few order scenarios
    let order_scenarios = [
        "standard-order",
        "wholesale-express",
        "internal-prepaid",
        "invalid-order",
    ];

    for scenario_name in order_scenarios {
        println!("┌──────────────────────────────────────────────────────────────────┐");
        println!("│ Scenario: {:<54} │", scenario_name);
        println!("└──────────────────────────────────────────────────────────────────┘");

        if let Some(scenario) = orders_data["scenarios"].get(scenario_name) {
            println!(
                "  Description: {}",
                scenario["description"].as_str().unwrap_or("")
            );

            // Get initial facts for this scenario
            let mut current_facts: Vec<serde_json::Value> = scenario["facts"]
                .as_array()
                .map(|a| a.clone())
                .unwrap_or_default();

            // Run through pipeline stages
            let mut current_stage = "validation";
            let mut stages_run = 0;

            while stages_run < 10 {
                // Prevent infinite loops
                // Find the stage configuration
                let Some(stage_config) = order_stages.iter().find(|s| s.name == current_stage)
                else {
                    println!("  ✓ Pipeline complete (no more stages)\n");
                    break;
                };

                println!(
                    "\n  Stage: {} ({})",
                    stage_config.name, stage_config.rules_file
                );

                // Run the stage
                let result = run_pipeline_stage(
                    &provider,
                    stage_config.rules_file,
                    &current_facts,
                    ThinkingMode::Disabled,
                )?;

                println!("    Status: {}", result.status);
                if let Some(route) = &result.route_to {
                    println!("    Route-to: {}", route);
                }
                println!("    Output facts: {}", result.output_facts.len());

                // Check for specific outputs based on stage
                summarize_stage_output(&result, stage_config.name);

                // Determine next stage
                if result.status == "completed" || result.status == "routed" {
                    if let Some(route) = &result.route_to {
                        if let Some(next_stage) = stage_config.routes.get(route.as_str()) {
                            // Update facts: include outputs and update pipeline-item for next stage
                            current_facts =
                                prepare_next_stage_facts(&result.output_facts, next_stage);
                            current_stage = next_stage;
                            stages_run += 1;
                            continue;
                        }
                    }
                    // No route or unknown route - pipeline ends
                    println!("  ✓ Pipeline complete\n");
                    break;
                } else if result.status == "failed" {
                    println!("  ✗ Pipeline failed at {}\n", stage_config.name);
                    break;
                } else if result.status == "skipped" {
                    println!("  ○ Item skipped at {}\n", stage_config.name);
                    break;
                } else {
                    println!("  ? Unexpected status: {}\n", result.status);
                    break;
                }
            }
        }
    }

    // =========================================================================
    // Pipeline 2: Incident Response (Branching)
    // =========================================================================
    println!("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
    println!("  PIPELINE 2: Incident Response (detection → classification → branch)");
    println!("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n");

    // Load incident scenarios
    let incidents_path = Path::new("../../../shared/data/pipeline/incidents.json");
    let incidents_data: serde_json::Value =
        serde_json::from_str(&fs::read_to_string(incidents_path)?)?;

    // Define incident pipeline with branching
    let incident_stages: HashMap<&str, PipelineStage> = [
        (
            "detection",
            PipelineStage {
                name: "detection",
                rules_file: "incident-detection.clp",
                routes: [("classification", "classification")].into_iter().collect(),
            },
        ),
        (
            "classification",
            PipelineStage {
                name: "classification",
                rules_file: "incident-classification.clp",
                routes: [
                    ("security", "security"),
                    ("operations", "operations"),
                    ("escalation", "escalation"),
                ]
                .into_iter()
                .collect(),
            },
        ),
        (
            "security",
            PipelineStage {
                name: "security",
                rules_file: "incident-response-security.clp",
                routes: HashMap::new(),
            },
        ),
        (
            "operations",
            PipelineStage {
                name: "operations",
                rules_file: "incident-response-ops.clp",
                routes: HashMap::new(),
            },
        ),
        (
            "escalation",
            PipelineStage {
                name: "escalation",
                rules_file: "incident-response-escalation.clp",
                routes: HashMap::new(),
            },
        ),
    ]
    .into_iter()
    .collect();

    // Run incident scenarios
    let incident_scenarios = [
        "brute-force-attack",
        "malware-detected",
        "service-outage",
        "data-exfiltration",
    ];

    for scenario_name in incident_scenarios {
        println!("┌──────────────────────────────────────────────────────────────────┐");
        println!("│ Scenario: {:<54} │", scenario_name);
        println!("└──────────────────────────────────────────────────────────────────┘");

        if let Some(scenario) = incidents_data["scenarios"].get(scenario_name) {
            println!(
                "  Description: {}",
                scenario["description"].as_str().unwrap_or("")
            );

            // Get initial facts
            let mut current_facts: Vec<serde_json::Value> = scenario["facts"]
                .as_array()
                .map(|a| a.clone())
                .unwrap_or_default();

            // Track the path through the pipeline
            let mut pipeline_path: Vec<String> = Vec::new();
            let mut current_stage = "detection";
            let mut stages_run = 0;

            while stages_run < 10 {
                let Some(stage_config) = incident_stages.get(current_stage) else {
                    break;
                };

                pipeline_path.push(stage_config.name.to_string());
                println!(
                    "\n  Stage: {} ({})",
                    stage_config.name, stage_config.rules_file
                );

                let result = run_pipeline_stage(
                    &provider,
                    stage_config.rules_file,
                    &current_facts,
                    ThinkingMode::Disabled,
                )?;

                println!("    Status: {}", result.status);
                if let Some(route) = &result.route_to {
                    println!("    Route-to: {}", route);
                }
                println!("    Output facts: {}", result.output_facts.len());

                // Summarize stage-specific outputs
                summarize_incident_stage_output(&result, stage_config.name);

                // Determine next stage based on routing
                if result.status == "completed" || result.status == "routed" {
                    if let Some(route) = &result.route_to {
                        if let Some(next_stage) = stage_config.routes.get(route.as_str()) {
                            // Prepare facts for next stage - convert stage name to stage symbol
                            current_facts =
                                prepare_incident_stage_facts(&result.output_facts, next_stage);
                            current_stage = next_stage;
                            stages_run += 1;
                            continue;
                        }
                    }
                    // No route - pipeline complete
                    println!("\n  ✓ Pipeline complete");
                    println!("  Path: {}", pipeline_path.join(" → "));
                    break;
                } else if result.status == "failed" {
                    println!("\n  ✗ Pipeline failed at {}", stage_config.name);
                    println!("  Path: {}", pipeline_path.join(" → "));
                    break;
                } else if result.status == "skipped" {
                    println!("\n  ○ Item skipped at {}", stage_config.name);
                    println!("  Path: {}", pipeline_path.join(" → "));
                    break;
                } else {
                    println!("\n  ? Unexpected status: {}", result.status);
                    break;
                }
            }
            println!();
        }
    }

    println!("════════════════════════════════════════════════════════════════════════");
    println!("  Pipeline Examples Complete");
    println!("════════════════════════════════════════════════════════════════════════");

    Ok(())
}

/// Run a single pipeline stage and return the results
fn run_pipeline_stage(
    provider: &NxuskitProvider,
    rules_file: &str,
    input_facts: &[serde_json::Value],
    thinking_mode: ThinkingMode,
) -> Result<StageResult, Box<dyn std::error::Error>> {
    // Prepare input JSON
    // Note: derived_only_new=false ensures we get all facts including modified pipeline-item
    let input = serde_json::json!({
        "facts": input_facts,
        "config": {
            "include_trace": false,
            "derived_only_new": false
        }
    });

    let request = ChatRequest::new(rules_file)
        .with_message(Message::user(input.to_string()))
        .with_thinking_mode(thinking_mode);

    let response = provider.chat(request)?;
    let output: serde_json::Value = serde_json::from_str(&response.content)?;

    // Extract pipeline-item status and route-to
    let mut status = "unknown".to_string();
    let mut route_to = None;
    let mut stage_name = String::new();

    if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
        for fact in conclusions {
            let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
            if template == "pipeline-item" {
                if let Some(values) = fact.get("values") {
                    // Status can be either a symbol object or a string
                    if let Some(s) = values.get("status") {
                        if let Some(sym) = s.get("symbol").and_then(|s| s.as_str()) {
                            status = sym.to_string();
                        } else if let Some(str_val) = s.as_str() {
                            status = str_val.to_string();
                        }
                    }
                    // Route-to can be either a symbol object or a string
                    if let Some(r) = values.get("route-to") {
                        if let Some(sym) = r.get("symbol").and_then(|s| s.as_str()) {
                            if sym != "nil" {
                                route_to = Some(sym.to_string());
                            }
                        } else if let Some(str_val) = r.as_str() {
                            if str_val != "nil" {
                                route_to = Some(str_val.to_string());
                            }
                        }
                    }
                    if let Some(src) = values.get("source-stage") {
                        if let Some(sym) = src.get("symbol").and_then(|s| s.as_str()) {
                            stage_name = sym.to_string();
                        } else if let Some(str_val) = src.as_str() {
                            stage_name = str_val.to_string();
                        }
                    }
                }
            }
        }
    }

    // If we still don't have a status, check if there are domain-specific indicators
    // that tell us the stage completed (e.g., presence of validated-order, priced-order, etc.)
    if status == "unknown" {
        if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
            for fact in conclusions {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                match template {
                    "validated-order"
                    | "priced-order"
                    | "fulfillment-decision"
                    | "detected-incident"
                    | "classified-incident"
                    | "response-summary" => {
                        // If we have domain output facts, assume the stage completed
                        status = "completed".to_string();
                        break;
                    }
                    "validation-issue" => {
                        // Check if it's blocking
                        if let Some(values) = fact.get("values") {
                            if let Some(sev) = values
                                .get("severity")
                                .and_then(|v| v.get("symbol"))
                                .and_then(|s| s.as_str())
                            {
                                if sev == "blocking" {
                                    status = "failed".to_string();
                                }
                            }
                        }
                    }
                    _ => {}
                }
            }
        }
    }

    // Infer route-to from domain facts if not explicitly set
    if route_to.is_none() && status == "completed" {
        if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
            for fact in conclusions {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                match template {
                    "validated-order" => {
                        // Check if it's internal prepaid (routes to fulfillment)
                        if let Some(values) = fact.get("values") {
                            if let Some(ctype) = values
                                .get("customer-type")
                                .and_then(|v| v.get("symbol"))
                                .and_then(|s| s.as_str())
                            {
                                if ctype == "internal" {
                                    route_to = Some("fulfillment".to_string());
                                } else {
                                    route_to = Some("pricing".to_string());
                                }
                            } else {
                                route_to = Some("pricing".to_string());
                            }
                        }
                    }
                    "priced-order" => {
                        route_to = Some("fulfillment".to_string());
                    }
                    "detected-incident" => {
                        route_to = Some("classification".to_string());
                    }
                    "classified-incident" => {
                        if let Some(values) = fact.get("values") {
                            // Check response-team and escalation-required
                            let team = values
                                .get("response-team")
                                .and_then(|v| v.get("symbol"))
                                .and_then(|s| s.as_str())
                                .unwrap_or("");
                            let escalation = values
                                .get("escalation-required")
                                .and_then(|v| v.get("symbol"))
                                .and_then(|s| s.as_str())
                                .unwrap_or("no");
                            if escalation == "yes" || team == "all" {
                                route_to = Some("escalation".to_string());
                            } else if team == "security" {
                                route_to = Some("security".to_string());
                            } else if team == "operations" {
                                route_to = Some("operations".to_string());
                            }
                        }
                    }
                    _ => {}
                }
            }
        }
    }

    // Collect all output facts
    let output_facts = output
        .get("conclusions")
        .and_then(|c| c.as_array())
        .map(|a| a.clone())
        .unwrap_or_default();

    // Extract thinking if available
    let thinking = output
        .get("thinking")
        .and_then(|t| t.as_str())
        .map(|s| s.to_string());

    Ok(StageResult {
        stage_name,
        status,
        route_to,
        output_facts,
        thinking,
    })
}

/// Prepare facts for the next stage in order pipeline
fn prepare_next_stage_facts(
    output_facts: &[serde_json::Value],
    next_stage: &str,
) -> Vec<serde_json::Value> {
    let mut facts = Vec::new();
    let mut item_id = String::new();

    // First pass: find the item ID from any domain fact
    for fact in output_facts {
        if let Some(values) = fact.get("values") {
            // Try to extract item ID from various domain facts
            if let Some(id) = values.get("order-id").and_then(|v| v.as_str()) {
                item_id = id.to_string();
                break;
            }
            if let Some(id) = values.get("item-id").and_then(|v| v.as_str()) {
                item_id = id.to_string();
                break;
            }
        }
    }

    for fact in output_facts {
        let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");

        // Skip pipeline-item - we'll create a new one
        if template == "pipeline-item" {
            // Extract item-id from existing pipeline-item if not found yet
            if item_id.is_empty() {
                if let Some(values) = fact.get("values") {
                    if let Some(id) = values.get("item-id").and_then(|v| v.as_str()) {
                        item_id = id.to_string();
                    }
                }
            }
            continue;
        }

        // Skip stage-info and pipeline-error/metric (internal tracking)
        if template == "stage-info" || template == "pipeline-error" || template == "pipeline-metric"
        {
            continue;
        }

        // Skip intermediate/internal facts that are stage-specific
        // These are created as intermediate results but not meant for downstream stages
        if template == "shipping-rate"
            || template == "discount-applied"
            || template == "validation-issue"
        {
            continue;
        }

        // Only pass forward the main domain artifacts:
        // validated-order, priced-order, fulfillment-decision, etc.
        facts.push(fact.clone());
    }

    // Always create a new pipeline-item for next stage
    if !item_id.is_empty() {
        facts.insert(
            0,
            serde_json::json!({
                "template": "pipeline-item",
                "values": {
                    "item-id": item_id,
                    "item-type": {"symbol": "order"},
                    "stage": {"symbol": next_stage},
                    "status": {"symbol": "pending"}
                }
            }),
        );
    }

    facts
}

/// Prepare facts for the next stage in incident pipeline
fn prepare_incident_stage_facts(
    output_facts: &[serde_json::Value],
    next_stage: &str,
) -> Vec<serde_json::Value> {
    let mut facts = Vec::new();
    let mut incident_id = String::new();

    // First pass: find the incident ID from any fact
    for fact in output_facts {
        if let Some(values) = fact.get("values") {
            // Try various ID field names
            if let Some(id) = values.get("incident-id").and_then(|v| v.as_str()) {
                incident_id = id.to_string();
                break;
            }
            if let Some(id) = values.get("item-id").and_then(|v| v.as_str()) {
                incident_id = id.to_string();
                break;
            }
            if let Some(id) = values.get("event-id").and_then(|v| v.as_str()) {
                incident_id = id.to_string();
                break;
            }
        }
    }

    // Map next_stage name to stage symbol for pipeline-item
    let stage_symbol = match next_stage {
        "classification" => "classification",
        "security" => "response",
        "operations" => "response",
        "escalation" => "response",
        _ => next_stage,
    };

    // Second pass: collect facts (skip pipeline-item, we create a new one)
    for fact in output_facts {
        let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");

        // Skip pipeline-item - we'll create a new one
        if template == "pipeline-item" {
            continue;
        }

        // Skip internal tracking facts
        if template == "stage-info" || template == "pipeline-error" || template == "pipeline-metric"
        {
            continue;
        }

        // Skip intermediate facts that are stage-specific
        if template == "detection-indicator" || template == "classification-factor" {
            continue;
        }

        // Include main domain facts (detected-incident, classified-incident, etc.)
        facts.push(fact.clone());
    }

    // Always create a new pipeline-item for next stage
    if !incident_id.is_empty() {
        facts.insert(
            0,
            serde_json::json!({
                "template": "pipeline-item",
                "values": {
                    "item-id": incident_id,
                    "item-type": {"symbol": "incident"},
                    "stage": {"symbol": stage_symbol},
                    "status": {"symbol": "pending"}
                }
            }),
        );
    }

    facts
}

/// Summarize order processing stage output
fn summarize_stage_output(result: &StageResult, stage_name: &str) {
    match stage_name {
        "validation" => {
            let mut issues = Vec::new();
            let mut has_validated_order = false;

            for fact in &result.output_facts {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                if template == "validation-issue" {
                    if let Some(values) = fact.get("values") {
                        let msg = values.get("message").and_then(|v| v.as_str()).unwrap_or("");
                        let sev = values
                            .get("severity")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        issues.push(format!("[{}] {}", sev, msg));
                    }
                } else if template == "validated-order" {
                    has_validated_order = true;
                }
            }

            if has_validated_order {
                println!("    ✓ Order validated");
            }
            if !issues.is_empty() {
                println!("    Issues found:");
                for issue in issues.iter().take(3) {
                    println!("      - {}", issue);
                }
                if issues.len() > 3 {
                    println!("      ... and {} more", issues.len() - 3);
                }
            }
        }
        "pricing" => {
            for fact in &result.output_facts {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                if template == "priced-order" {
                    if let Some(values) = fact.get("values") {
                        let original = values
                            .get("original-amount")
                            .and_then(|v| v.as_f64())
                            .unwrap_or(0.0);
                        let final_amt = values
                            .get("final-amount")
                            .and_then(|v| v.as_f64())
                            .unwrap_or(0.0);
                        let discount = values
                            .get("total-discount")
                            .and_then(|v| v.as_f64())
                            .unwrap_or(0.0);
                        println!(
                            "    Pricing: ${:.2} → ${:.2} (discount: ${:.2})",
                            original, final_amt, discount
                        );
                    }
                }
            }
        }
        "fulfillment" => {
            for fact in &result.output_facts {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                if template == "fulfillment-plan" {
                    if let Some(values) = fact.get("values") {
                        let warehouse = values
                            .get("warehouse")
                            .and_then(|v| v.as_str())
                            .unwrap_or("?");
                        let carrier = values
                            .get("carrier")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        let routing = values
                            .get("routing")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        println!(
                            "    Fulfillment: {} via {} (routing: {})",
                            warehouse, carrier, routing
                        );
                    }
                }
            }
        }
        _ => {}
    }
}

/// Summarize incident response stage output
fn summarize_incident_stage_output(result: &StageResult, stage_name: &str) {
    match stage_name {
        "detection" => {
            for fact in &result.output_facts {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                if template == "detected-incident" {
                    if let Some(values) = fact.get("values") {
                        let inc_type = values
                            .get("incident-type")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        let category = values
                            .get("category")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        let severity = values
                            .get("initial-severity")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        println!(
                            "    Detected: {} / {} (severity: {})",
                            inc_type, category, severity
                        );
                    }
                }
            }
        }
        "classification" => {
            for fact in &result.output_facts {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                if template == "classified-incident" {
                    if let Some(values) = fact.get("values") {
                        let team = values
                            .get("response-team")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        let severity = values
                            .get("final-severity")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("?");
                        let sla = values
                            .get("sla-hours")
                            .and_then(|v| v.as_i64())
                            .unwrap_or(0);
                        let escalation = values
                            .get("escalation-required")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("no");
                        println!(
                            "    Classified: {} team, severity={}, SLA={}h, escalation={}",
                            team, severity, sla, escalation
                        );
                    }
                }
            }
        }
        "security" | "operations" | "escalation" => {
            let mut action_count = 0;
            let mut immediate_count = 0;

            for fact in &result.output_facts {
                let template = fact.get("template").and_then(|t| t.as_str()).unwrap_or("");
                if template == "response-action" {
                    action_count += 1;
                    if let Some(values) = fact.get("values") {
                        let priority = values
                            .get("priority")
                            .and_then(|v| v.get("symbol"))
                            .and_then(|s| s.as_str())
                            .unwrap_or("");
                        if priority == "immediate" {
                            immediate_count += 1;
                        }
                    }
                } else if template == "response-summary" {
                    if let Some(values) = fact.get("values") {
                        let total = values
                            .get("total-actions")
                            .and_then(|v| v.as_i64())
                            .unwrap_or(0);
                        let immed = values
                            .get("immediate-actions")
                            .and_then(|v| v.as_i64())
                            .unwrap_or(0);
                        let est_hours = values
                            .get("estimated-resolution-hours")
                            .and_then(|v| v.as_i64())
                            .unwrap_or(0);
                        println!(
                            "    Response: {} actions ({} immediate), est. {}h resolution",
                            total, immed, est_hours
                        );
                    }
                }
            }

            if action_count > 0 {
                println!(
                    "    Actions generated: {} ({} immediate priority)",
                    action_count, immediate_count
                );
            }
        }
        _ => {}
    }
}
