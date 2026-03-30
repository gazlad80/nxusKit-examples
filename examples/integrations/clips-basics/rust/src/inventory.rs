//! Example: Inventory Management with CLIPS
//!
//! This example demonstrates a real-world inventory management system that:
//! - Monitors stock levels across warehouses
//! - Generates reorder alerts based on inventory thresholds
//! - Suggests pricing adjustments based on velocity and stock
//! - Recommends warehouse transfers for stock balancing
//!
//! Facts are loaded from JSON files in the data/ directory.
//!
//! Run with: cargo run --bin inventory

use nxuskit::{ChatRequest, Message, NxuskitProvider, ProviderConfig, ThinkingMode};
use std::fs;
use std::path::Path;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=== Inventory Management Expert System ===\n");

    // Create CLIPS provider via NxuskitProvider
    let provider = NxuskitProvider::new(ProviderConfig {
        provider_type: "clips".to_string(),
        model: Some("../../../shared/rules".to_string()),
        ..Default::default()
    })?;

    // Load facts from JSON file
    let data_path = Path::new("../../../shared/data/inventory-scenario.json");

    let input_json = if data_path.exists() {
        println!("Loading facts from: {}\n", data_path.display());
        fs::read_to_string(data_path)?
    } else {
        println!("Data file not found, using inline facts\n");
        // Inline fallback for demonstration
        serde_json::json!({
            "facts": [
                // Products
                {
                    "template": "product",
                    "values": {
                        "sku": "WIDGET-001",
                        "name": "Premium Widget",
                        "category": {"symbol": "widgets"},
                        "unit-cost": 12.50,
                        "unit-price": 29.99,
                        "reorder-point": 50,
                        "reorder-quantity": 100,
                        "supplier": "Acme Widgets Inc."
                    }
                },
                {
                    "template": "product",
                    "values": {
                        "sku": "GADGET-002",
                        "name": "Super Gadget",
                        "category": {"symbol": "gadgets"},
                        "unit-cost": 25.00,
                        "unit-price": 59.99,
                        "reorder-point": 30,
                        "reorder-quantity": 60,
                        "supplier": "Gadget World"
                    }
                },
                {
                    "template": "product",
                    "values": {
                        "sku": "GIZMO-003",
                        "name": "Mega Gizmo",
                        "category": {"symbol": "gizmos"},
                        "unit-cost": 8.00,
                        "unit-price": 19.99,
                        "reorder-point": 100,
                        "reorder-quantity": 200,
                        "supplier": "Gizmo Factory"
                    }
                },
                // Inventory levels
                {
                    "template": "inventory",
                    "values": {
                        "sku": "WIDGET-001",
                        "warehouse": "MAIN",
                        "quantity-on-hand": 15,
                        "quantity-reserved": 5,
                        "last-updated": "2025-01-28"
                    }
                },
                {
                    "template": "inventory",
                    "values": {
                        "sku": "GADGET-002",
                        "warehouse": "MAIN",
                        "quantity-on-hand": 0,
                        "quantity-reserved": 0,
                        "last-updated": "2025-01-28"
                    }
                },
                {
                    "template": "inventory",
                    "values": {
                        "sku": "GIZMO-003",
                        "warehouse": "MAIN",
                        "quantity-on-hand": 450,
                        "quantity-reserved": 20,
                        "last-updated": "2025-01-28"
                    }
                },
                // Sales velocity
                {
                    "template": "sales-velocity",
                    "values": {
                        "sku": "WIDGET-001",
                        "daily-average": 8.5,
                        "trend": {"symbol": "stable"}
                    }
                },
                {
                    "template": "sales-velocity",
                    "values": {
                        "sku": "GIZMO-003",
                        "daily-average": 1.2,
                        "trend": {"symbol": "decreasing"}
                    }
                }
            ],
            "config": {
                "include_trace": true
            }
        })
        .to_string()
    };

    // Run inference
    let request = ChatRequest::new("inventory-management.clp")
        .with_message(Message::user(input_json))
        .with_thinking_mode(ThinkingMode::Enabled);

    let response = provider.chat(request)?;

    // Parse and display results
    let output: serde_json::Value = serde_json::from_str(&response.content)?;

    println!("=== Stock Status ===\n");
    if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
        for fact in conclusions {
            if fact.get("template").and_then(|t| t.as_str()) == Some("stock-status")
                && let Some(values) = fact.get("values")
            {
                println!(
                    "  {} - {}: {}",
                    values.get("sku").and_then(|v| v.as_str()).unwrap_or("?"),
                    values
                        .get("status")
                        .and_then(|v| v.get("symbol"))
                        .and_then(|s| s.as_str())
                        .unwrap_or("?"),
                    values.get("message").and_then(|v| v.as_str()).unwrap_or("")
                );
            }
        }
    }

    println!("\n=== Reorder Alerts ===\n");
    if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
        for fact in conclusions {
            if fact.get("template").and_then(|t| t.as_str()) == Some("reorder-alert")
                && let Some(values) = fact.get("values")
            {
                println!(
                    "  [{:?}] {} ({})",
                    values
                        .get("urgency")
                        .and_then(|v| v.get("symbol"))
                        .and_then(|s| s.as_str())
                        .unwrap_or("?"),
                    values
                        .get("product-name")
                        .and_then(|v| v.as_str())
                        .unwrap_or("?"),
                    values.get("sku").and_then(|v| v.as_str()).unwrap_or("?")
                );
                println!(
                    "    Current: {}, Available: {}, Reorder Point: {}",
                    values
                        .get("current-stock")
                        .and_then(|v| v.as_i64())
                        .unwrap_or(0),
                    values
                        .get("available-stock")
                        .and_then(|v| v.as_i64())
                        .unwrap_or(0),
                    values
                        .get("reorder-point")
                        .and_then(|v| v.as_i64())
                        .unwrap_or(0)
                );
                println!(
                    "    Recommended Order: {} | Supplier: {}",
                    values
                        .get("recommended-quantity")
                        .and_then(|v| v.as_i64())
                        .unwrap_or(0),
                    values
                        .get("supplier")
                        .and_then(|v| v.as_str())
                        .unwrap_or("?")
                );
                println!(
                    "    Reason: {}",
                    values.get("reason").and_then(|v| v.as_str()).unwrap_or("")
                );
                println!();
            }
        }
    }

    println!("=== Pricing Adjustments ===\n");
    if let Some(conclusions) = output.get("conclusions").and_then(|c| c.as_array()) {
        let pricing: Vec<_> = conclusions
            .iter()
            .filter(|f| f.get("template").and_then(|t| t.as_str()) == Some("pricing-adjustment"))
            .collect();

        if pricing.is_empty() {
            println!("  No pricing adjustments recommended\n");
        } else {
            for fact in pricing {
                if let Some(values) = fact.get("values") {
                    println!(
                        "  {} - ${:.2} -> ${:.2} ({:+.1}%)",
                        values.get("sku").and_then(|v| v.as_str()).unwrap_or("?"),
                        values
                            .get("current-price")
                            .and_then(|v| v.as_f64())
                            .unwrap_or(0.0),
                        values
                            .get("recommended-price")
                            .and_then(|v| v.as_f64())
                            .unwrap_or(0.0),
                        values
                            .get("adjustment-percent")
                            .and_then(|v| v.as_f64())
                            .unwrap_or(0.0)
                    );
                    println!(
                        "    Reason: {}",
                        values.get("reason").and_then(|v| v.as_str()).unwrap_or("")
                    );
                }
            }
        }
    }

    println!("\n=== Execution Stats ===\n");
    if let Some(stats) = output.get("stats") {
        println!(
            "  Rules fired: {}, Conclusions: {}, Time: {}ms",
            stats
                .get("total_rules_fired")
                .and_then(|v| v.as_u64())
                .unwrap_or(0),
            stats
                .get("conclusions_count")
                .and_then(|v| v.as_u64())
                .unwrap_or(0),
            stats
                .get("execution_time_ms")
                .and_then(|v| v.as_u64())
                .unwrap_or(0)
        );
    }

    println!("\n=== Inventory Management Complete ===");

    Ok(())
}
