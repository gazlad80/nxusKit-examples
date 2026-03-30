//! Example: Medical Triage with CLIPS - Thinking Mode Demo
//!
//! This example demonstrates the medical triage rule base and shows how
//! thinking mode (trace visibility) works with the CLIPS provider.
//!
//! - ThinkingMode::Enabled shows the trace (which rules fired)
//! - ThinkingMode::Disabled hides the trace
//! - ThinkingMode::Auto enables trace for CLIPS (rule firings = reasoning)
//!
//! Run with: cargo run --bin medical_triage
//!
//! DISCLAIMER: This is an educational example only. Not for actual medical use.

use nxuskit::{ChatRequest, Message, NxuskitProvider, ProviderConfig, ThinkingMode};

fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("=== Medical Triage Expert System ===\n");
    println!("DISCLAIMER: Educational example only - not for medical use\n");

    // Create CLIPS provider via NxuskitProvider
    let provider = NxuskitProvider::new(ProviderConfig {
        provider_type: "clips".to_string(),
        model: Some("../../../shared/rules".to_string()),
        ..Default::default()
    })?;

    // Example 1: Critical patient with thinking ENABLED (shows trace)
    println!("--- Case 1: Critical Patient (Thinking ENABLED) ---\n");
    println!("Trace shows which rules fired and why the decision was made.\n");

    let critical_patient = serde_json::json!({
        "facts": [
            {
                "template": "patient",
                "values": {
                    "id": "P001",
                    "name": "John Smith",
                    "age": 68,
                    "gender": {"symbol": "male"}
                }
            },
            {
                "template": "vital-signs",
                "values": {
                    "patient-id": "P001",
                    "temperature": 37.2,
                    "heart-rate": 45,
                    "systolic-bp": 85,
                    "diastolic-bp": 55,
                    "respiratory-rate": 22,
                    "oxygen-saturation": 88,
                    "consciousness": {"symbol": "verbal"}
                }
            },
            {
                "template": "symptom",
                "values": {
                    "patient-id": "P001",
                    "type": {"symbol": "chest-pain"},
                    "severity": {"symbol": "severe"},
                    "duration-hours": 2
                }
            },
            {
                "template": "medical-history",
                "values": {
                    "patient-id": "P001",
                    "condition": {"symbol": "cardiac-disease"},
                    "current": {"symbol": "yes"}
                }
            }
        ],
        "config": {
            "include_trace": true
        }
    });

    let request = ChatRequest::new("medical-triage.clp")
        .with_message(Message::user(critical_patient.to_string()))
        .with_thinking_mode(ThinkingMode::Enabled);

    let response = provider.chat(request)?;
    println!("Result with trace:\n{}\n", response.content);

    // Example 2: Same patient with thinking DISABLED (no trace)
    println!("--- Case 2: Same Patient (Thinking DISABLED) ---\n");
    println!("Only conclusions shown, no rule firing trace.\n");

    let request = ChatRequest::new("medical-triage.clp")
        .with_message(Message::user(critical_patient.to_string()))
        .with_thinking_mode(ThinkingMode::Disabled);

    let response = provider.chat(request)?;
    println!("Result without trace:\n{}\n", response.content);

    // Example 3: Moderate urgency patient
    println!("--- Case 3: Urgent Patient ---\n");

    let urgent_patient = serde_json::json!({
        "facts": [
            {
                "template": "patient",
                "values": {
                    "id": "P002",
                    "name": "Jane Doe",
                    "age": 45,
                    "gender": {"symbol": "female"}
                }
            },
            {
                "template": "vital-signs",
                "values": {
                    "patient-id": "P002",
                    "temperature": 38.8,
                    "heart-rate": 95,
                    "systolic-bp": 120,
                    "diastolic-bp": 80,
                    "respiratory-rate": 18,
                    "oxygen-saturation": 93,
                    "consciousness": {"symbol": "alert"}
                }
            },
            {
                "template": "symptom",
                "values": {
                    "patient-id": "P002",
                    "type": {"symbol": "abdominal-pain"},
                    "severity": {"symbol": "moderate"},
                    "duration-hours": 6
                }
            }
        ],
        "config": {
            "include_trace": true
        }
    });

    let request = ChatRequest::new("medical-triage.clp")
        .with_message(Message::user(urgent_patient.to_string()))
        .with_thinking_mode(ThinkingMode::Enabled);

    let response = provider.chat(request)?;
    println!("Urgent patient result:\n{}\n", response.content);

    // Example 4: Pediatric case with special alerts
    println!("--- Case 4: Pediatric High Fever ---\n");

    let pediatric_patient = serde_json::json!({
        "facts": [
            {
                "template": "patient",
                "values": {
                    "id": "P003",
                    "name": "Tommy Wilson",
                    "age": 3,
                    "gender": {"symbol": "male"}
                }
            },
            {
                "template": "vital-signs",
                "values": {
                    "patient-id": "P003",
                    "temperature": 39.5,
                    "heart-rate": 130,
                    "systolic-bp": 95,
                    "diastolic-bp": 60,
                    "respiratory-rate": 28,
                    "oxygen-saturation": 97,
                    "consciousness": {"symbol": "alert"}
                }
            },
            {
                "template": "symptom",
                "values": {
                    "patient-id": "P003",
                    "type": {"symbol": "fever"},
                    "severity": {"symbol": "moderate"},
                    "duration-hours": 4
                }
            }
        ]
    });

    let request = ChatRequest::new("medical-triage.clp")
        .with_message(Message::user(pediatric_patient.to_string()))
        .with_thinking_mode(ThinkingMode::Enabled);

    let response = provider.chat(request)?;
    println!("Pediatric case result:\n{}\n", response.content);

    // Example 5: Non-urgent stable patient
    println!("--- Case 5: Non-Urgent Stable Patient ---\n");

    let stable_patient = serde_json::json!({
        "facts": [
            {
                "template": "patient",
                "values": {
                    "id": "P004",
                    "name": "Alice Brown",
                    "age": 35,
                    "gender": {"symbol": "female"}
                }
            },
            {
                "template": "vital-signs",
                "values": {
                    "patient-id": "P004",
                    "temperature": 37.0,
                    "heart-rate": 72,
                    "systolic-bp": 118,
                    "diastolic-bp": 76,
                    "respiratory-rate": 14,
                    "oxygen-saturation": 99,
                    "consciousness": {"symbol": "alert"}
                }
            },
            {
                "template": "symptom",
                "values": {
                    "patient-id": "P004",
                    "type": {"symbol": "headache"},
                    "severity": {"symbol": "mild"},
                    "duration-hours": 2
                }
            }
        ]
    });

    let request = ChatRequest::new("medical-triage.clp")
        .with_message(Message::user(stable_patient.to_string()))
        .with_thinking_mode(ThinkingMode::Enabled);

    let response = provider.chat(request)?;
    println!("Stable patient result:\n{}\n", response.content);

    println!("=== Medical Triage Demo Complete ===");

    Ok(())
}
